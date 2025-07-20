package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_OpenAI_MissingAPIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")

	client, err := NewClient(LLMProviderOpenAI, LLMClientOptions{Model: "gpt-4"})
	assert.Nil(t, client)
	require.Error(t, err)
	assert.Equal(t, "OPENAI_API_KEY environment variable is not set", err.Error())
}

func TestNewClient_OpenAI_WithAPIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "dummy-key")

	client, err := NewClient(LLMProviderOpenAI, LLMClientOptions{Model: "gpt-4-o"})
	require.NoError(t, err)
	assert.IsType(t, &llmClientOpenAi{}, client)
	assert.Equal(t, client.(*llmClientOpenAi).model, "gpt-4-o")
}

func TestNewClient_Ollama_MissingEndpoint(t *testing.T) {
	t.Setenv("OLLAMA_ENDPOINT", "")

	client, err := NewClient(LLMProviderOllama, LLMClientOptions{Model: "llama2"})
	assert.Nil(t, client)
	require.Error(t, err)
	assert.Equal(t, "OLLAMA_ENDPOINT environment variable is not set", err.Error())
}

func TestNewClient_Ollama_InvalidURL(t *testing.T) {
	t.Setenv("OLLAMA_ENDPOINT", "http://bad::url")

	client, err := NewClient(LLMProviderOllama, LLMClientOptions{Model: "llama2"})
	assert.Nil(t, client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OLLAMA_ENDPOINT URL is invalid")
}

func TestNewClient_Ollama_ValidURL(t *testing.T) {
	t.Setenv("OLLAMA_ENDPOINT", "http://localhost:11434")

	client, err := NewClient(LLMProviderOllama, LLMClientOptions{Model: "llama2"})
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.IsType(t, &llmClientOllama{}, client)
	assert.Equal(t, client.(*llmClientOllama).model, "llama2")
}

func TestNewClient_InvalidProvider(t *testing.T) {
	client, err := NewClient(LLMProvider("unknown"), LLMClientOptions{Model: "x"})
	assert.Nil(t, client)
	require.Error(t, err)
	assert.Equal(t, "unknown: invalid provider", err.Error())
}
