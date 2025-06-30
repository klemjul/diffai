package llm

func RoughEstimateCodeTokens(text string) int {
	avgCharsPerToken := 3.0
	tokens := int(float64(len([]rune(text))) / avgCharsPerToken)
	if tokens < 1 {
		tokens = 1
	}
	return tokens
}
