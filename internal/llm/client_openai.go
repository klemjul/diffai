package llm

import (
	"context"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type llmClientOpenAi struct {
	client openai.Client
	model  string
}
type LLMClientOpenAI LLMClient

func newOpenAIClient(openAIKey string, model string) LLMClientOpenAI {
	return &llmClientOpenAi{
		client: openai.NewClient(
			option.WithAPIKey(openAIKey),
		),
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
	res, err := ai.client.Chat.Completions.New(
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
		aiStream := ai.client.Chat.Completions.NewStreaming(
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
			acc.AddChunk(chunk)
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				out <- LLMStreamEvent{
					Type:    LLMStreamEventTypeMessage,
					Content: chunk.Choices[0].Delta.Content,
					Usage: LLMTokenUsage{
						InputTokens:  chunk.Usage.CompletionTokens,
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
