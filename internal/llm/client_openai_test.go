package llm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type openaiMockClient struct {
	mock.Mock
}

func (m *openaiMockClient) New(ctx context.Context, body openai.ChatCompletionNewParams, opts ...option.RequestOption) (res *openai.ChatCompletion, err error) {
	args := m.Called(ctx, body, opts)
	resVal := args.Get(0)
	if resVal == nil {
		return nil, args.Error(1)
	}
	return resVal.(*openai.ChatCompletion), args.Error(1)
}

func (m *openaiMockClient) NewStreaming(ctx context.Context, body openai.ChatCompletionNewParams, opts ...option.RequestOption) (stream openaiChatStream) {
	args := m.Called(ctx, body, opts)

	return args.Get(0).(openaiChatStream)
}

type openaiMockStream struct {
	mock.Mock
}

func (m *openaiMockStream) Next() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *openaiMockStream) Current() openai.ChatCompletionChunk {
	args := m.Called()
	return args.Get(0).(openai.ChatCompletionChunk)
}

func (m *openaiMockStream) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *openaiMockStream) Err() error {
	args := m.Called()
	return args.Error(0)
}

func newOpenaiMockClient(model string) *llmClientOpenAi {
	mockClient := new(openaiMockClient)

	return &llmClientOpenAi{
		client: mockClient,
		model:  model,
	}
}

func TestSendOpenai_Success(t *testing.T) {
	messages := []Message{
		{Role: System, Content: "This is a system message"},
		{Role: User, Content: "Hello"},
		{Role: Assistant, Content: "Hi, how can I help?"},
	}
	mockClient := newOpenaiMockClient("openai-model")
	mockClient.client.(*openaiMockClient).
		On("New", t.Context(), openai.ChatCompletionNewParams{
			Model: "openai-model",
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage("This is a system message"),
				openai.UserMessage("Hello"),
				openai.AssistantMessage("Hi, how can I help?"),
			},
			N: openai.Int(1),
		}, mock.Anything).
		Return(&openai.ChatCompletion{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "Open ai response content",
					},
				},
			},
			Usage: openai.CompletionUsage{
				PromptTokens:     50,
				CompletionTokens: 100,
			},
		}, nil)

	res, err := mockClient.Send(t.Context(), messages)

	assert.Nil(t, err)
	assert.Equal(t, &LLMSendResponse{
		Content: "Open ai response content",
		Usage: LLMTokenUsage{
			InputTokens:  50,
			OutputTokens: 100,
		},
	}, res)
}

func TestSendOpenai_Error(t *testing.T) {
	messages := []Message{
		{Role: User, Content: "Hello"},
	}
	mockClient := newOpenaiMockClient("openai-model")
	mockClient.client.(*openaiMockClient).
		On("New", t.Context(), openai.ChatCompletionNewParams{
			Model: "openai-model",
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("Hello"),
			},
			N: openai.Int(1),
		}, mock.Anything).
		Return(nil, errors.New("failed to send message"))

	res, err := mockClient.Send(t.Context(), messages)

	assert.Error(t, err, "failed to send message")
	assert.Nil(t, res)
}

func TestSendStream_Success(t *testing.T) {
	messages := []Message{
		{Role: User, Content: "Hello"},
	}
	mockClient := newOpenaiMockClient("openai-model")
	mockStream := new(openaiMockStream)
	mockClient.client.(*openaiMockClient).
		On("NewStreaming", t.Context(), openai.ChatCompletionNewParams{
			Model: "openai-model",
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("Hello"),
			},
			StreamOptions: openai.ChatCompletionStreamOptionsParam{
				IncludeUsage: openai.Bool(true),
			},
			N: openai.Int(1),
		}, mock.Anything).
		Return(mockStream)
	mockStream.On("Next").Return(true).Once()
	mockStream.On("Current").Return(openai.ChatCompletionChunk{
		Choices: []openai.ChatCompletionChunkChoice{
			{Delta: openai.ChatCompletionChunkChoiceDelta{
				Content: "hello ",
			}},
		},
		Usage: openai.CompletionUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
		},
	}).Once()

	mockStream.On("Next").Return(true).Once()
	mockStream.On("Current").Return(openai.ChatCompletionChunk{
		Choices: []openai.ChatCompletionChunkChoice{
			{Delta: openai.ChatCompletionChunkChoiceDelta{
				Content: "back",
			}},
		},
		Usage: openai.CompletionUsage{
			PromptTokens:     5,
			CompletionTokens: 5,
		},
	}).Once()
	mockStream.On("Err").Return(nil)

	mockStream.On("Next").Return(true).Once()
	mockStream.On("Current").Return(openai.ChatCompletionChunk{
		Choices: []openai.ChatCompletionChunkChoice{
			{Delta: openai.ChatCompletionChunkChoiceDelta{
				Content: "",
			}},
		},
		Usage: openai.CompletionUsage{
			PromptTokens:     5,
			CompletionTokens: 5,
		},
	}).Once()
	mockStream.On("Err").Return(nil)

	mockStream.On("Next").Return(false).Once()

	stream := mockClient.Stream(t.Context(), messages)

	var events []LLMStreamEvent

	for event := range stream {
		events = append(events, event)
	}

	require.Len(t, events, 3)
	assert.Equal(t, "hello ", events[0].Content)
	assert.Equal(t, LLMStreamEventTypeMessage, events[0].Type)

	assert.Equal(t, "back", events[1].Content)
	assert.Equal(t, LLMStreamEventTypeMessage, events[1].Type)

	assert.Equal(t, LLMStreamEvent{
		Content: "hello back",
		Type:    LLMStreamEventTypeComplete,
		Usage: LLMTokenUsage{
			InputTokens:  15,
			OutputTokens: 25,
		},
	}, events[2])
	assert.Equal(t, LLMStreamEventTypeComplete, events[2].Type)

}

func TestSendStream_Error(t *testing.T) {
	messages := []Message{
		{Role: User, Content: "Hello"},
	}
	mockClient := newOpenaiMockClient("openai-model")
	mockStream := new(openaiMockStream)
	mockClient.client.(*openaiMockClient).
		On("NewStreaming", t.Context(), openai.ChatCompletionNewParams{
			Model: "openai-model",
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("Hello"),
			},
			StreamOptions: openai.ChatCompletionStreamOptionsParam{
				IncludeUsage: openai.Bool(true),
			},
			N: openai.Int(1),
		}, mock.Anything).
		Return(mockStream)
	mockStream.On("Next").Return(true).Once()
	mockStream.On("Current").Return(openai.ChatCompletionChunk{
		Choices: []openai.ChatCompletionChunkChoice{
			{Delta: openai.ChatCompletionChunkChoiceDelta{
				Content: "hello",
			}},
		},
		Usage: openai.CompletionUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
		},
	}).Once()
	mockStream.On("Next").Return(false).Once()

	mockStream.On("Err").Return(errors.New("failed to stream content"))

	stream := mockClient.Stream(t.Context(), messages)

	var events []LLMStreamEvent

	for event := range stream {
		events = append(events, event)
	}

	require.Len(t, events, 2)
	assert.Equal(t, "hello", events[0].Content)
	assert.Equal(t, LLMStreamEventTypeMessage, events[0].Type)

	assert.Equal(t, LLMStreamEvent{
		Type:    LLMStreamEventTypeMessage,
		Content: "hello",
		Usage: LLMTokenUsage{
			InputTokens:  10,
			OutputTokens: 20,
		},
	}, events[0])

	assert.Equal(t, LLMStreamEvent{
		Type:    LLMStreamEventTypeError,
		Content: "failed to stream content",
	}, events[1])

}

func TestSendStream_HttpSuccess(t *testing.T) {
	messages := []Message{
		{Role: User, Content: "Hello"},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		// Simulated streamed JSON chunks (like OpenAI sends)
		chunks := []string{
			`data: {"id":"chatcmpl-1","choices":[{"delta":{"content":"hello"},"finish_reason":null}],"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}}` + "\n\n",
			`data: {"id":"chatcmpl-2","choices":[{"delta":{"content":" world"},"finish_reason":null}],"usage":{"prompt_tokens":5,"completion_tokens":7,"total_tokens":12}}` + "\n\n",
			`data: {"id":"chatcmpl-3","choices":[{"delta":{"content":"!"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":8,"total_tokens":13}}` + "\n\n",
		}

		for _, chunk := range chunks {
			_, err := w.Write([]byte(chunk))
			if err != nil {
				return
			}
			flusher.Flush()
			time.Sleep(50 * time.Millisecond)
		}
	}))
	defer ts.Close()

	client := newOpenAIClient("", "model", option.WithHTTPClient(ts.Client()), option.WithBaseURL(ts.URL))

	stream := client.Stream(t.Context(), messages)

	var events []LLMStreamEvent

	for event := range stream {
		events = append(events, event)
	}

	require.Len(t, events, 4)
	assert.Equal(t, "hello", events[0].Content)
	assert.Equal(t, LLMStreamEventTypeMessage, events[0].Type)

	assert.Equal(t, LLMStreamEvent{
		Type:    LLMStreamEventTypeMessage,
		Content: "hello",
		Usage: LLMTokenUsage{
			InputTokens:  5,
			OutputTokens: 3,
		},
	}, events[0])

}
