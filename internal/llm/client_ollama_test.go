package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type ollamaMockClient struct {
	mock.Mock
}

func (m *ollamaMockClient) Chat(ctx context.Context, req *api.ChatRequest, callback api.ChatResponseFunc) error {
	args := m.Called(ctx, req, callback)

	return args.Error(0)
}

type ollamaMockClientOptions struct {
	Context context.Context
	ChatErr error
	ChatOut []api.ChatResponse
}

func newOllamaMockClient(model string, opts ollamaMockClientOptions) *llmClientOllama {
	mockClient := new(ollamaMockClient)
	mockClient.On("Chat", opts.Context, mock.AnythingOfType("*api.ChatRequest"), mock.AnythingOfType("api.ChatResponseFunc")).
		Return(opts.ChatErr).
		Run(
			func(args mock.Arguments) {
				callback := args.Get(2).(api.ChatResponseFunc)
				if opts.ChatOut != nil {
					for _, res := range opts.ChatOut {
						callback(res)
					}
				}

			},
		)
	return &llmClientOllama{
		client: mockClient,
		model:  model,
	}
}

func TestSendOllama_Success(t *testing.T) {
	client := newOllamaMockClient("test-model", ollamaMockClientOptions{
		Context: t.Context(),
		ChatOut: []api.ChatResponse{
			{
				Message: api.Message{
					Content: "hello from mock",
				},
				Done: true,
			},
		},
	})

	messages := []Message{{Content: "hello"}}

	res, err := client.Send(t.Context(), messages)

	assert.Equal(t, "hello from mock", res.Content)
	assert.Nil(t, err)

	client.client.(*ollamaMockClient).AssertExpectations(t)
}

func TestSendOllama_Error(t *testing.T) {
	client := newOllamaMockClient("test-model", ollamaMockClientOptions{
		Context: t.Context(),
		ChatErr: errors.New("failed to stream response"),
	})

	messages := []Message{{Content: "hello"}}

	res, err := client.Send(t.Context(), messages)
	assert.Nil(t, res)
	assert.Error(t, err)

	client.client.(*ollamaMockClient).AssertExpectations(t)
}

func TestStreamOllama_Success(t *testing.T) {
	client := newOllamaMockClient("test-model", ollamaMockClientOptions{
		Context: t.Context(),
		ChatOut: []api.ChatResponse{
			{
				Message: api.Message{
					Content: "hello",
				},
				Done: false,
			},
			{
				Message: api.Message{
					Content: " from mock",
				},
				Done: true,
			},
		},
	})

	stream := client.Stream(t.Context(), []Message{{Content: "hello"}})
	var events []LLMStreamEvent

	for event := range stream {
		events = append(events, event)
	}

	require.Len(t, events, 3)
	assert.Equal(t, "hello", events[0].Content)
	assert.Equal(t, LLMStreamEventTypeMessage, events[0].Type)
	assert.Equal(t, " from mock", events[1].Content)
	assert.Equal(t, LLMStreamEventTypeMessage, events[1].Type)
	assert.Equal(t, LLMStreamEventTypeComplete, events[2].Type)

	client.client.(*ollamaMockClient).AssertExpectations(t)
}

func TestStreamOllama_Error(t *testing.T) {
	client := newOllamaMockClient("test-model", ollamaMockClientOptions{
		Context: t.Context(),
		ChatErr: errors.New("failed to stream response"),
	})

	stream := client.Stream(t.Context(), []Message{{Content: "hello"}})
	var events []LLMStreamEvent

	for event := range stream {
		events = append(events, event)
	}

	require.Len(t, events, 1)
	assert.Equal(t, "failed to stream response", events[0].Content)
	assert.Equal(t, LLMStreamEventTypeError, events[0].Type)

	client.client.(*ollamaMockClient).AssertExpectations(t)
}

func TestToOllamaMessages(t *testing.T) {
	ai := &llmClientOllama{}

	input := []Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi, how can I help?"},
	}

	expected := []api.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi, how can I help?"},
	}

	result := ai.toOllamaMessages(input)

	require.Equal(t, expected, result)
}
