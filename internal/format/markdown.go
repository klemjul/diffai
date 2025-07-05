package format

import "github.com/charmbracelet/glamour"

func FormatMarkdown(text string) (string, error) {
	return glamour.Render(text, "dark")
}
