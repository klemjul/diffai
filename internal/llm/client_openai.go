package llm

import (
	"context"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/ssestream"
)

type llmClientOpenAI struct {
	client openaiClient
	model  string
}

type openaiClient interface {
	New(ctx context.Context, body openai.ChatCompletionNewParams, opts ...option.RequestOption) (res *openai.ChatCompletion, err error)
	NewStreaming(ctx context.Context, body openai.ChatCompletionNewParams, opts ...option.RequestOption) (stream openaiChatStream)
}

type openaiChatStream interface {
	Next() bool
	Current() openai.ChatCompletionChunk
	Close() error
	Err() error
}

type defaultOpenaiChatStream struct {
	stream *ssestream.Stream[openai.ChatCompletionChunk]
}

func (s *defaultOpenaiChatStream) Next() bool {
	return s.stream.Next()
}

func (s *defaultOpenaiChatStream) Current() openai.ChatCompletionChunk {
	return s.stream.Current()
}

func (s *defaultOpenaiChatStream) Close() error {
	return s.stream.Close()
}

func (s *defaultOpenaiChatStream) Err() error {
	return s.stream.Err()
}

type defaultOpenAIClient struct {
	client openai.Client
}

func (c *defaultOpenAIClient) New(ctx context.Context, body openai.ChatCompletionNewParams, opts ...option.RequestOption) (*openai.ChatCompletion, error) {
	return c.client.Chat.Completions.New(ctx, body, opts...)
}

func (c *defaultOpenAIClient) NewStreaming(ctx context.Context, body openai.ChatCompletionNewParams, opts ...option.RequestOption) openaiChatStream {
	return &defaultOpenaiChatStream{stream: c.client.Chat.Completions.NewStreaming(ctx, body, opts...)}
}

func newOpenAIClient(openAIKey string, model string, opts ...option.RequestOption) LLMClient {
	return &llmClientOpenAI{
		client: &defaultOpenAIClient{client: openai.NewClient(
			append([]option.RequestOption{option.WithAPIKey(openAIKey)}, opts...)...,
		)},
		model: model,
	}
}

func (ai *llmClientOpenAI) toOpenAIMessages(messages []Message) []openai.ChatCompletionMessageParamUnion {
	var openAIMessages []openai.ChatCompletionMessageParamUnion
	for _, msg := range messages {
		switch msg.Role {
		case User:
			openAIMessages = append(openAIMessages, openai.UserMessage(msg.Content))
		case Assistant:
			openAIMessages = append(openAIMessages, openai.AssistantMessage(msg.Content))
		case System:
			openAIMessages = append(openAIMessages, openai.SystemMessage(msg.Content))
		default:
			openAIMessages = append(openAIMessages, openai.UserMessage(msg.Content))
		}
	}
	return openAIMessages
}

func (ai *llmClientOpenAI) Send(ctx context.Context, messages []Message) (res *LLMSendResponse, err error) {
	openAIRes, err := ai.client.New(
		ctx,
		openai.ChatCompletionNewParams{
			Model:    ai.model,
			Messages: ai.toOpenAIMessages(messages),
			N:        openai.Int(1),
		},
	)

	if err != nil {
		return
	}

	return &LLMSendResponse{
		Content: openAIRes.Choices[0].Message.Content,
		Usage: LLMTokenUsage{
			InputTokens:  openAIRes.Usage.PromptTokens,
			OutputTokens: openAIRes.Usage.CompletionTokens,
		},
	}, nil
}

func (ai *llmClientOpenAI) Stream(ctx context.Context, messages []Message) <-chan LLMStreamEvent {
	out := make(chan LLMStreamEvent)

	go func() {
		defer close(out)
		acc := openai.ChatCompletionAccumulator{}
		aiStream := ai.client.NewStreaming(
			ctx,
			openai.ChatCompletionNewParams{
				Model:    ai.model,
				Messages: ai.toOpenAIMessages(messages),
				StreamOptions: openai.ChatCompletionStreamOptionsParam{
					IncludeUsage: openai.Bool(true),
				},
				N: openai.Int(1),
			},
		)

		for aiStream.Next() {
			chunk := aiStream.Current()
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				acc.AddChunk(chunk)
				out <- LLMStreamEvent{
					Type:    LLMStreamEventTypeMessage,
					Content: chunk.Choices[0].Delta.Content,
					Usage: LLMTokenUsage{
						InputTokens:  chunk.Usage.PromptTokens,
						OutputTokens: chunk.Usage.CompletionTokens,
					},
				}
			}
		}

		err := aiStream.Err()
		if err != nil {
			out <- LLMStreamEvent{
				Type:    LLMStreamEventTypeError,
				Content: err.Error(),
			}
			return
		}

		out <- LLMStreamEvent{
			Type:    LLMStreamEventTypeComplete,
			Content: acc.Choices[0].Message.Content,
			Usage: LLMTokenUsage{
				InputTokens:  acc.Usage.PromptTokens,
				OutputTokens: acc.Usage.CompletionTokens,
			},
		}

	}()

	return out
}
