package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/klemjul/diffai/internal/format"
	"github.com/klemjul/diffai/internal/llm"
)

type ChatTUIModel struct {
	textInput textinput.Model
	viewport  viewport.Model
	messages  []llm.Message
	title     string
	waiting   bool

	getBotResponse func(messages []llm.Message) tea.Cmd
}

const (
	CHAT_INPUT_PLACEHOLDER = "Type a message..."
	CHAT_INIT_LOADING      = "Loading..."
	CHAT_WAITING_RESPONSE  = "> ⏳ Waiting for response..."
	CHAT_TYPING_INDICATOR  = "Bot: typing..."
)

var (
	userStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	botStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	titleStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true)
	inputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true)
)

type InitialModelOptions struct {
	Title          string
	GetBotResponse func(messages []llm.Message) tea.Cmd
	Messages       []llm.Message
}

func InitialModel(opts InitialModelOptions) ChatTUIModel {
	ti := textinput.New()
	ti.Placeholder = CHAT_INPUT_PLACEHOLDER
	ti.Focus()

	return ChatTUIModel{
		textInput:      ti,
		viewport:       viewport.New(0, 0),
		title:          opts.Title,
		getBotResponse: opts.GetBotResponse,
		messages:       opts.Messages,
		waiting:        true,
	}
}

func (m ChatTUIModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.getBotResponse(m.messages),
		tea.EnableMouseCellMotion,
	)
}

func (m ChatTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		titleLines := (len(m.title) / msg.Width) + 1
		m.viewport = viewport.New(msg.Width, msg.Height-(3+titleLines))
		m.updateViewport()

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				m.viewport.ScrollUp(1)
			case tea.MouseButtonWheelDown:
				m.viewport.ScrollDown(1)
			}
		}

	case llm.Message:
		m.waiting = false
		m.messages = append(m.messages, msg)
		m.updateViewport()

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			cmd = tea.Quit
		case tea.KeyEnter:
			if m.textInput.Value() != "" && !m.waiting {
				userMsg := llm.Message{
					Role:    llm.User,
					Content: m.textInput.Value(),
				}
				m.messages = append(m.messages, userMsg)
				m.waiting = true
				m.textInput.SetValue("")
				m.updateViewport()

				cmd = m.getBotResponse(m.messages)
			}
		}
	}

	m.textInput, _ = m.textInput.Update(msg)

	if m.waiting {
		m.textInput.Blur()
	} else {
		m.textInput.Focus()
	}

	return m, cmd
}

func (m *ChatTUIModel) updateViewport() {
	displayedMessages := make([]string, len(m.messages))
	for i, msg := range m.messages {
		if msg.Hidden {
			continue
		}
		switch msg.Role {
		case llm.Assistant:
			out, _ := format.FormatMarkdown(msg.Content)
			displayedMessages[i] = botStyle.Render(strings.TrimSpace(out))
		case llm.User:
			displayedMessages[i] = userStyle.Render(fmt.Sprintf("> %s", msg.Content))
		}
	}

	content := strings.Join(displayedMessages, "\n\n")
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m ChatTUIModel) View() string {
	input := m.textInput.View()

	if m.waiting {
		input = CHAT_WAITING_RESPONSE
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Width(m.viewport.Width).Render(m.title),
		m.viewport.View(),
		inputStyle.Width(m.viewport.Width).Render(input),
	)
}
