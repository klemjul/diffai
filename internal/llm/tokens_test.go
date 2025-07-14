package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoughEstimateCodeTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: 1,
		},
		{
			name:     "Short string (2 runes)",
			input:    "Go",
			expected: 1,
		},
		{
			name:     "Exactly 3 runes",
			input:    "Hey",
			expected: 1,
		},
		{
			name:     "6 runes",
			input:    "Hello!",
			expected: 2,
		},
		{
			name:     "9 runes",
			input:    "Hello GPT",
			expected: 3,
		},
		{
			name:     "Longer sentence",
			input:    "This is a longer sentence with multiple words.",
			expected: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoughEstimateCodeTokens(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
