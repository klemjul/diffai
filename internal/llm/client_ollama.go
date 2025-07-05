package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ollama/ollama/api"
)

type llmClientOllama struct {
	client api.Client
	model  string
}
type LlmClientOllama LLMClient

func newOllamaClient(localEndpoint url.URL, model string) LlmClientOllama {
	return &llmClientOllama{
		client: *api.NewClient(&localEndpoint, http.DefaultClient),
		model:  model,
	}
}

func (ai *llmClientOllama) Send(ctx context.Context, messages []Message) (*LLMSendResponse, error) {
	stream := ai.chat(ctx, messages)
	var fullResult string
	for event := range stream {
		switch event.Type {
		case LLMStreamEventTypeMessage:
			fullResult += event.Content
		case LLMStreamEventTypeComplete:
			return &LLMSendResponse{
				Content: fullResult,
			}, nil
		case LLMStreamEventTypeError:
			return nil, fmt.Errorf("ollama error: %s", event.Content)
		default:
			fullResult += event.Content
		}
	}
	return &LLMSendResponse{
		Content: fullResult,
	}, nil
}
func (ai *llmClientOllama) Stream(ctx context.Context, messages []Message) <-chan LLMStreamEvent {
	return ai.chat(ctx, messages)
}

func (ai *llmClientOllama) chat(ctx context.Context, messages []Message) <-chan LLMStreamEvent {
	out := make(chan LLMStreamEvent)
	stream := true
	go func() {
		defer close(out)

		err := ai.client.Chat(ctx, &api.ChatRequest{
			Model:    ai.model,
			Messages: ai.toOllamaMessages(messages),
			Stream:   &stream,
		}, func(resp api.ChatResponse) error {
			out <- LLMStreamEvent{Content: resp.Message.Content, Type: LLMStreamEventTypeMessage}
			if resp.Done {
				out <- LLMStreamEvent{Type: LLMStreamEventTypeComplete}
			}
			return nil
		})

		if err != nil {
			out <- LLMStreamEvent{
				Type:    LLMStreamEventTypeError,
				Content: err.Error(),
			}
		}
	}()
	return out
}

func (ai *llmClientOllama) toOllamaMessages(messages []Message) []api.Message {
	var ollamaMessages []api.Message
	for _, msg := range messages {
		ollamaMessages = append(ollamaMessages, api.Message{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}
	return ollamaMessages
}
