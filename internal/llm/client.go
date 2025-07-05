package llm

import (
	"context"
	"fmt"
	"net/url"
	"os"
)

type LLMTokenUsage struct {
	InputTokens  int64
	OutputTokens int64
}

type LLMSendResponse struct {
	Content string
	Usage   LLMTokenUsage
}

type LLMStreamEventType string

const (
	LLMStreamEventTypeMessage  LLMStreamEventType = "content"
	LLMStreamEventTypeComplete LLMStreamEventType = "complete"
	LLMStreamEventTypeError    LLMStreamEventType = "error"
)

type LLMStreamEvent struct {
	Content string
	Usage   LLMTokenUsage
	Type    LLMStreamEventType
}

type LLMClient interface {
	Send(ctx context.Context, messages []Message) (*LLMSendResponse, error)
	Stream(ctx context.Context, messages []Message) <-chan LLMStreamEvent
}

type LLMProvider string

const (
	LLMProviderOpenAI LLMProvider = "openai"
	LLMProviderOllama LLMProvider = "ollama"
)

var LLMProviders = []LLMProvider{LLMProviderOpenAI, LLMProviderOllama}

type LLMClientOptions struct {
	Model string
}

func NewClient(provider LLMProvider, opts LLMClientOptions) (LLMClient, error) {
	switch provider {
	case LLMProviderOpenAI:
		apiKey, exists := os.LookupEnv("OPENAI_API_KEY")
		if !exists || apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set")
		}
		return newOpenAIClient(apiKey, opts.Model), nil
	case LLMProviderOllama:
		ollameEndpoint, exists := os.LookupEnv("OLLAMA_ENDPOINT")
		if !exists || ollameEndpoint == "" {
			return nil, fmt.Errorf("OLLAMA_ENDPOINT environment variable is not set")
		}
		localEndpoint, err := url.Parse(ollameEndpoint)
		if err != nil {
			return nil, fmt.Errorf("OLLAMA_ENDPOINT URL is invalid: %v", err)
		}
		return newOllamaClient(*localEndpoint, opts.Model), nil
	default:
		return nil, fmt.Errorf("%s: invalid provider", provider)
	}
}
