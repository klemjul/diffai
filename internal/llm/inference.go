package llm

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func Generate(prompt string, maxTokens int, onToken func(string)) (string, error) {
	openAIKey, present := os.LookupEnv("OPENAI_API_KEY")
	if !present {
		return "", fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}
	client := openai.NewClient(
		option.WithAPIKey(openAIKey),
	)

	stream := client.Chat.Completions.NewStreaming(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
		Seed:  openai.Int(0),
		Model: openai.ChatModelGPT4_1,
	})

	acc := openai.ChatCompletionAccumulator{}

	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)

		if len(chunk.Choices) > 0 {
			if onToken != nil {
				onToken(chunk.Choices[0].Delta.Content)
			}
		}
	}

	return acc.Choices[0].Message.Content, stream.Err()
}
