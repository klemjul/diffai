package llm

import (
	"context"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/ssestream"
)

type LLMClientOpenAI LLMClient

type llmClientOpenAi struct {
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

type defaultOpenAiClient struct {
	client openai.Client
}

func (c *defaultOpenAiClient) New(ctx context.Context, body openai.ChatCompletionNewParams, opts ...option.RequestOption) (*openai.ChatCompletion, error) {
	return c.client.Chat.Completions.New(ctx, body, opts...)
}

func (c *defaultOpenAiClient) NewStreaming(ctx context.Context, body openai.ChatCompletionNewParams, opts ...option.RequestOption) openaiChatStream {
	return &defaultOpenaiChatStream{stream: c.client.Chat.Completions.NewStreaming(ctx, body, opts...)}
}

func newOpenAIClient(openAIKey string, model string, opts ...option.RequestOption) LLMClientOpenAI {
	return &llmClientOpenAi{
		client: &defaultOpenAiClient{client: openai.NewClient(
			append([]option.RequestOption{option.WithAPIKey(openAIKey)}, opts...)...,
		)},
		model: model,
	}
}

func (ai *llmClientOpenAi) toOpenAiMessages(messages []Message) []openai.ChatCompletionMessageParamUnion {
	var openAiMessages []openai.ChatCompletionMessageParamUnion
	for _, msg := range messages {
		switch msg.Role {
		case User:
			openAiMessages = append(openAiMessages, openai.UserMessage(msg.Content))
		case Assistant:
			openAiMessages = append(openAiMessages, openai.AssistantMessage(msg.Content))
		case System:
			openAiMessages = append(openAiMessages, openai.SystemMessage(msg.Content))
		default:
			openAiMessages = append(openAiMessages, openai.UserMessage(msg.Content))
		}
	}
	return openAiMessages
}

func (ai *llmClientOpenAi) Send(ctx context.Context, messages []Message) (*LLMSendResponse, error) {
	res, err := ai.client.New(
		ctx,
		openai.ChatCompletionNewParams{
			Model:    ai.model,
			Messages: ai.toOpenAiMessages(messages),
			N:        openai.Int(1),
		},
	)

	if err != nil {
		return nil, err
	}

	return &LLMSendResponse{
		Content: res.Choices[0].Message.Content,
		Usage: LLMTokenUsage{
			InputTokens:  res.Usage.PromptTokens,
			OutputTokens: res.Usage.CompletionTokens,
		},
	}, nil
}

func (ai *llmClientOpenAi) Stream(ctx context.Context, messages []Message) <-chan LLMStreamEvent {
	out := make(chan LLMStreamEvent)

	go func() {
		defer close(out)
		acc := openai.ChatCompletionAccumulator{}
		aiStream := ai.client.NewStreaming(
			ctx,
			openai.ChatCompletionNewParams{
				Model:    ai.model,
				Messages: ai.toOpenAiMessages(messages),
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
			Content: acc.ChatCompletion.Choices[0].Message.Content,
			Usage: LLMTokenUsage{
				InputTokens:  acc.ChatCompletion.Usage.PromptTokens,
				OutputTokens: acc.ChatCompletion.Usage.CompletionTokens,
			},
		}

	}()

	return out
}
