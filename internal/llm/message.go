package llm

type MessageRole string

const (
	Assistant MessageRole = "assistant"
	User      MessageRole = "user"
	System    MessageRole = "system"
)

type Message struct {
	Role    MessageRole
	Content string
}
