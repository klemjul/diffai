package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/klemjul/diffai/internal/llm"
	"github.com/stretchr/testify/assert"
)

func mockGetBotResponse(messages []llm.Message) tea.Cmd {
	return func() tea.Msg {
		return llm.Message{
			Role:    llm.Assistant,
			Content: "Mock bot response",
		}
	}
}

func TestInitialModel(t *testing.T) {
	tests := []struct {
		name string
		opts InitialModelOptions
	}{
		{
			name: "basic initialization",
			opts: InitialModelOptions{
				Title:          "Test Chat",
				GetBotResponse: mockGetBotResponse,
				Messages:       []llm.Message{},
			},
		},
		{
			name: "initialization with initial messages",
			opts: InitialModelOptions{
				Title:          "Chat with initial messages",
				GetBotResponse: mockGetBotResponse,
				Messages: []llm.Message{
					{Role: llm.User, Content: "Hello"},
					{Role: llm.Assistant, Content: "Hi there!"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := InitialModel(tt.opts)

			assert.Equal(t, tt.opts.Title, m.title)
			assert.Equal(t, true, m.waiting)
			assert.Equal(t, CHAT_INPUT_PLACEHOLDER, m.textInput.Placeholder)
			assert.Equal(t, m.viewport.Height, 0)
			assert.Equal(t, m.viewport.Width, 0)
			assert.Len(t, m.messages, len(tt.opts.Messages))
			assert.NotNil(t, m.getBotResponse)
		})
	}
}

func TestModelUpdate_WindowSizeMsg(t *testing.T) {
	tests := []struct {
		name      string
		screenW   int
		screenH   int
		expectedW int
		expectedH int
		uiTitle   string
	}{
		{
			name:      "title in ui width",
			screenW:   80,
			screenH:   24,
			expectedW: 80,
			expectedH: 20,
			uiTitle:   "with one line title",
		},
		{
			name:      "title exceed ui width",
			screenW:   10,
			screenH:   24,
			expectedW: 10,
			expectedH: 19,
			uiTitle:   "with one line title",
		},
		{
			name:      "title exceed a lot ui width",
			screenW:   5,
			screenH:   24,
			expectedW: 5,
			expectedH: 16,
			uiTitle:   "with two lines title",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			initModel := InitialModel(InitialModelOptions{
				GetBotResponse: mockGetBotResponse,
				Title:          tc.uiTitle,
			})

			windowMsg := tea.WindowSizeMsg{Width: tc.screenW, Height: tc.screenH}
			updatedModel, cmd := initModel.Update(windowMsg)
			updated := updatedModel.(model)

			assert.Equal(t, tc.expectedH, updated.viewport.Height)
			assert.Equal(t, tc.expectedW, updated.viewport.Width)
			assert.Nil(t, cmd, "WindowSizeMsg should not return a command")
		})
	}

}

func TestModelUpdate_MouseMsg(t *testing.T) {
	initContent := strings.Repeat("line\n", 20) // add 20 lines to test scroll
	initYOffset := 10
	testCases := []struct {
		name            string
		msg             tea.MouseMsg
		expectedYOffset int
	}{
		{
			name: "wheel up",
			msg: tea.MouseMsg{
				Action: tea.MouseActionPress,
				Button: tea.MouseButtonWheelUp,
			},
			expectedYOffset: 9,
		},
		{
			name: "wheel down",
			msg: tea.MouseMsg{
				Action: tea.MouseActionPress,
				Button: tea.MouseButtonWheelDown,
			},
			expectedYOffset: 11,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initModel := InitialModel(InitialModelOptions{
				GetBotResponse: mockGetBotResponse,
			})
			initModel.viewport.SetContent(initContent)
			initModel.viewport.SetYOffset(initYOffset)

			updatedModel, cmd := initModel.Update(tc.msg)
			updated := updatedModel.(model)

			assert.Equal(t, updated.viewport.YOffset, tc.expectedYOffset)

			assert.IsType(t, model{}, updatedModel)
			assert.Nil(t, cmd, "Mouse message should not return a command")
		})
	}
}

func TestModelUpdate_LLMMessage(t *testing.T) {
	initModel := InitialModel(InitialModelOptions{
		GetBotResponse: mockGetBotResponse,
		Title:          "test llm message update",
	})

	llmMsg := llm.Message{
		Role:    llm.Assistant,
		Content: "Test response",
	}

	assert.True(t, initModel.waiting, "waiting should be false after receiving LLM message")

	updatedModel, cmd := initModel.Update(llmMsg)
	updated := updatedModel.(model)

	assert.False(t, updated.waiting, "waiting should be false after receiving LLM message")
	assert.Len(t, updated.messages, 1)

	lastMessage := updated.messages[len(updated.messages)-1]
	assert.Equal(t, llm.Assistant, lastMessage.Role)
	assert.Equal(t, "Test response", lastMessage.Content)
	assert.Nil(t, cmd, "LLM message should not return a command")
}

func TestModelUpdate_KeyMsg(t *testing.T) {
	testCases := []struct {
		name               string
		key                tea.KeyMsg
		initInputValue     string
		initWaiting        bool
		expectedCmd        tea.Cmd
		expectedWaiting    bool
		expectedInputValue string
		expectedMessages   []llm.Message
	}{
		{
			name: "ctrl+c quits",
			key: tea.KeyMsg{
				Type: tea.KeyCtrlC,
			},
			expectedCmd: tea.Quit,
		},
		{
			name: "esc quits",
			key: tea.KeyMsg{
				Type: tea.KeyEsc,
			},
			expectedCmd: tea.Quit,
		},
		{
			name: "enter with input and not waiting",
			key: tea.KeyMsg{
				Type: tea.KeyEnter,
			},
			initInputValue:     "Hello",
			initWaiting:        false,
			expectedCmd:        mockGetBotResponse([]llm.Message{}),
			expectedWaiting:    true,
			expectedInputValue: "",
			expectedMessages: []llm.Message{
				{Role: llm.User, Content: "Hello"},
			},
		},
		{
			name: "enter with empty input",
			key: tea.KeyMsg{
				Type: tea.KeyEnter,
			},
			initInputValue:     "",
			initWaiting:        true,
			expectedWaiting:    true,
			expectedInputValue: "",
		},
		{
			name: "enter while waiting",
			key: tea.KeyMsg{
				Type: tea.KeyEnter,
			},
			initInputValue:     "Hello",
			initWaiting:        true,
			expectedWaiting:    true,
			expectedInputValue: "Hello",
		},
		{
			name: "unsuported key",
			key: tea.KeyMsg{
				Type: tea.KeyHome,
			},
			expectedCmd: nil,
		},
		{
			name: "key rune",
			key: tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune{'a'},
			},
			initInputValue:     "",
			expectedInputValue: "a",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initModel := InitialModel(InitialModelOptions{
				GetBotResponse: mockGetBotResponse,
				Title:          tc.name,
			})
			initModel.textInput.SetValue(tc.initInputValue)
			initModel.waiting = tc.initWaiting

			updatedModel, cmd := initModel.Update(tc.key)

			if tc.expectedCmd != nil {
				assert.NotNil(t, cmd)
				assert.Equal(t, cmd(), tc.expectedCmd())
			} else {
				assert.Nil(t, cmd)
			}

			model := updatedModel.(model)

			assert.Equal(t, tc.expectedInputValue, model.textInput.Value())

			assert.Equal(t, tc.expectedWaiting, model.waiting)

			if tc.expectedMessages != nil {
				assert.Equal(t, tc.expectedMessages, model.messages)
			}
		})
	}
}

func TestModelUpdate_TextInputFocusBlur(t *testing.T) {
	initModel := InitialModel(InitialModelOptions{
		GetBotResponse: mockGetBotResponse,
		Title:          t.Name(),
	})

	initModel.waiting = false
	updatedModel, _ := initModel.Update(tea.KeyMsg{})
	updated := updatedModel.(model)

	assert.True(t, updated.textInput.Focused(), "Text input should be focused when not waiting")

	initModel.waiting = true
	updatedModel, _ = initModel.Update(tea.KeyMsg{})
	updated = updatedModel.(model)

	assert.False(t, updated.textInput.Focused(), "Text input should be blurred when waiting")
}

func TestUpdateViewport(t *testing.T) {
	initModel := InitialModel(InitialModelOptions{
		GetBotResponse: mockGetBotResponse,
		Title:          t.Name(),
		Messages: []llm.Message{
			{Role: llm.User, Content: "Hello"},
			{Role: llm.Assistant, Content: "Hi there!"},
			{Role: llm.User, Content: "How are you?"},
		},
	})

	initModel.viewport.Width = 80
	initModel.viewport.Height = 20

	initModel.updateViewport()

	content := initModel.viewport.View()
	assert.NotEmpty(t, content, "Viewport content should not be empty")
	assert.Contains(t, content, "> Hello", "User messages should be prefixed with '>'")
	assert.Contains(t, content, "> How are you?", "User messages should be prefixed with '>'")
}

func TestUpdateViewport_HiddenMessages(t *testing.T) {
	initModel := InitialModel(InitialModelOptions{
		GetBotResponse: mockGetBotResponse,
		Title:          t.Name(),
		Messages: []llm.Message{
			{Role: llm.User, Content: "Visible message"},
			{Role: llm.Assistant, Content: "Hidden message", Hidden: true},
			{Role: llm.User, Content: "Another visible message"},
		},
	})

	initModel.viewport.Width = 80
	initModel.viewport.Height = 20

	initModel.updateViewport()

	content := initModel.viewport.View()
	assert.NotContains(t, content, "Hidden message", "Hidden messages should not appear in viewport")
	assert.Contains(t, content, "Visible message", "Visible messages should appear in viewport")
}

func TestModelView_Waiting(t *testing.T) {
	model := InitialModel(InitialModelOptions{
		Title:          t.Name(),
		GetBotResponse: mockGetBotResponse,
	})

	model.viewport.Width = 40
	model.viewport.Height = 10
	model.waiting = true

	view := model.View()

	assert.Contains(t, view, CHAT_WAITING_RESPONSE)
	assert.Contains(t, view, t.Name())
}

func TestModelView_InputShownWhenNotWaiting(t *testing.T) {
	model := InitialModel(InitialModelOptions{
		Title:          t.Name(),
		GetBotResponse: mockGetBotResponse,
	})

	model.viewport.Width = 40
	model.viewport.Height = 10
	model.waiting = false
	model.textInput.SetValue("Hello")

	view := model.View()

	assert.NotContains(t, view, CHAT_WAITING_RESPONSE)
	assert.Contains(t, view, "Hello")
	assert.Contains(t, view, t.Name())
}
