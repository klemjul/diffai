package format

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatMarkdown(t *testing.T) {
	res, _ := FormatMarkdown("**hello**")
	assert.Contains(t, res, "hello")
}
