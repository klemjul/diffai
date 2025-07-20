package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/klemjul/diffai/internal/app"
	"github.com/klemjul/diffai/internal/config"
	"github.com/klemjul/diffai/internal/git"
	"github.com/klemjul/diffai/internal/llm"
	"github.com/klemjul/diffai/internal/ui"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockGitService struct {
	mock.Mock
}

func (m *MockGitService) DiffStaged(diffOptions git.DiffOptions) (git.DiffResult, error) {
	args := m.Called(diffOptions)
	return args.Get(0).(git.DiffResult), args.Error(1)
}

func (m *MockGitService) DiffRefs(refFrom string, refTo string, diffOptions git.DiffOptions) (git.DiffResult, error) {
	args := m.Called(refFrom, refTo, diffOptions)
	return args.Get(0).(git.DiffResult), args.Error(1)
}

func (m *MockGitService) DiffCommit(ref string, diffOptions git.DiffOptions) (git.DiffResult, error) {
	args := m.Called(ref, diffOptions)
	return args.Get(0).(git.DiffResult), args.Error(1)
}

type MockTUIService struct {
	mock.Mock
}

func (m *MockTUIService) InitialModel(opts ui.InitialModelOptions) ui.ChatTUIModel {
	args := m.Called(opts)
	return args.Get(0).(ui.ChatTUIModel)
}

func (m *MockTUIService) Run(model ui.ChatTUIModel) (returnModel tea.Model, returnErr error) {
	args := m.Called(model)
	return args.Get(0).(tea.Model), args.Error(1)
}

type MockLLMService struct {
	mock.Mock
}

func (m *MockLLMService) NewClient(provider llm.LLMProvider, opts llm.LLMClientOptions) (llm.LLMClient, error) {
	args := m.Called(provider, opts)
	return args.Get(0).(llm.LLMClient), args.Error(1)
}

type MockLLMClient struct {
	mock.Mock
}

func (c *MockLLMClient) Send(ctx context.Context, messages []llm.Message) (*llm.LLMSendResponse, error) {
	args := c.Called(ctx, messages)
	return args.Get(0).(*llm.LLMSendResponse), args.Error(1)
}

func (c *MockLLMClient) Stream(ctx context.Context, messages []llm.Message) <-chan llm.LLMStreamEvent {
	args := c.Called(ctx, messages)
	return args.Get(0).(<-chan llm.LLMStreamEvent)
}

type MockFormatClient struct {
	mock.Mock
}

func (l *MockFormatClient) FormatMarkdown(text string) (string, error) {
	args := l.Called(text)
	return args.Get(0).(string), args.Error(1)
}

type MockApp struct {
	git    *MockGitService
	tui    *MockTUIService
	llm    *MockLLMService
	format *MockFormatClient
}

func (a *MockApp) Git() app.GitService           { return a.git }
func (a *MockApp) TUI() app.TUIService           { return a.tui }
func (a *MockApp) LLM() app.LLMService           { return a.llm }
func (a *MockApp) Format() app.TextFormatService { return a.format }

func NewMockApp() app.App {
	return &MockApp{git: &MockGitService{}, tui: &MockTUIService{}, llm: &MockLLMService{}, format: &MockFormatClient{}}
}

func TestMain(m *testing.M) {
	clearEnvWithPrefix(config.ENV_PREFIX)
	m.Run()
}

func clearEnvWithPrefix(prefix string) {
	for _, env := range os.Environ() {
		kv := strings.SplitN(env, "=", 2)
		key := kv[0]
		if strings.HasPrefix(key, prefix) {
			_ = os.Unsetenv(key)
		}
	}
}

func executeRootCommand(app app.App, args ...string) (string, error) {
	viper.Reset()
	cmd := RootCommand(app)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Value.Set(f.DefValue)
	})

	_, err := cmd.ExecuteC()
	return buf.String(), err
}

func TestRun_WithNoArgs_ShouldCallDiffStaged(t *testing.T) {
	app := NewMockApp()
	wd, _ := os.Getwd()
	app.Git().(*MockGitService).
		On("DiffStaged", git.DiffOptions{
			CliPath:     "git",
			CliWd:       wd,
			Unified:     3,
			FindRenames: true,
			Filters:     []string{},
		}).
		Return(git.DiffResult{
			Out:         []byte("diffout"),
			FullCommand: "fullcommand",
		}, nil)
	mockLLMClient := MockLLMClient{}
	app.LLM().(*MockLLMService).
		On("NewClient", llm.LLMProvider("ollama"), llm.LLMClientOptions{
			Model: "model",
		}).
		Return(&mockLLMClient, nil)
	app.Format().(*MockFormatClient).
		On("FormatMarkdown", "aires").Return("formated res", nil)
	mockLLMClient.
		On("Send", mock.Anything, []llm.Message{
			{
				Role:    llm.System,
				Content: "prompt",
				Hidden:  true,
			},
			{
				Role:    llm.User,
				Content: "diffout",
				Hidden:  true,
			},
		}).
		Return(&llm.LLMSendResponse{
			Content: "aires",
		}, nil)

	output, _ := executeRootCommand(app, "--provider", "ollama", "--model=model", "-p=prompt")

	assert.Equal(t, "formated res", output)
	app.Git().(*MockGitService).AssertExpectations(t)
	app.LLM().(*MockLLMService).AssertExpectations(t)
	app.Format().(*MockFormatClient).AssertExpectations(t)
}

func TestRun_WithOneArgs_ShouldCallDiffCommit(t *testing.T) {
	app := NewMockApp()
	wd, _ := os.Getwd()
	app.Git().(*MockGitService).
		On("DiffCommit", "shacommit", git.DiffOptions{
			CliPath:     "git",
			CliWd:       wd,
			Unified:     3,
			FindRenames: true,
			Filters:     []string{},
		}).
		Return(git.DiffResult{
			Out:         []byte("diffout"),
			FullCommand: "fullcommand",
		}, nil)
	mockLLMClient := MockLLMClient{}
	app.LLM().(*MockLLMService).
		On("NewClient", llm.LLMProvider("openai"), llm.LLMClientOptions{
			Model: "modelo",
		}).
		Return(&mockLLMClient, nil)
	app.Format().(*MockFormatClient).
		On("FormatMarkdown", "modelores").Return("formated modelores", nil)
	mockLLMClient.
		On("Send", mock.Anything, []llm.Message{
			{
				Role:    llm.System,
				Content: "prompt2",
				Hidden:  true,
			},
			{
				Role:    llm.User,
				Content: "diffout",
				Hidden:  true,
			},
		}).
		Return(&llm.LLMSendResponse{
			Content: "modelores",
		}, nil)

	output, _ := executeRootCommand(app, "shacommit", "--provider", "openai", "--model=modelo", "-p=prompt2")

	assert.Equal(t, "formated modelores", output)
	app.Git().(*MockGitService).AssertExpectations(t)
	app.LLM().(*MockLLMService).AssertExpectations(t)
	app.Format().(*MockFormatClient).AssertExpectations(t)
}

func TestRun_WithTwoArgs_ShouldCallDiffRefs(t *testing.T) {
	app := NewMockApp()
	wd, _ := os.Getwd()
	app.Git().(*MockGitService).
		On("DiffRefs", "diffFrom", "diffTo", git.DiffOptions{
			CliPath:     "git",
			CliWd:       wd,
			Unified:     3,
			FindRenames: true,
			Filters:     []string{},
		}).
		Return(git.DiffResult{
			Out:         []byte("diffout"),
			FullCommand: "fullcommand",
		}, nil)
	mockLLMClient := MockLLMClient{}
	app.LLM().(*MockLLMService).
		On("NewClient", llm.LLMProvider("ollama"), llm.LLMClientOptions{
			Model: "ollamam",
		}).
		Return(&mockLLMClient, nil)
	app.Format().(*MockFormatClient).
		On("FormatMarkdown", "ollamamres").Return("formated ollamamres", nil)
	mockLLMClient.
		On("Send", mock.Anything, []llm.Message{
			{
				Role:    llm.System,
				Content: "prompt3",
				Hidden:  true,
			},
			{
				Role:    llm.User,
				Content: "diffout",
				Hidden:  true,
			},
		}).
		Return(&llm.LLMSendResponse{
			Content: "ollamamres",
		}, nil)

	output, _ := executeRootCommand(app, "diffFrom", "diffTo", "--provider", "ollama", "--model=ollamam", "-p=prompt3")

	assert.Equal(t, "formated ollamamres", output)
	app.Git().(*MockGitService).AssertExpectations(t)
	app.LLM().(*MockLLMService).AssertExpectations(t)
	app.Format().(*MockFormatClient).AssertExpectations(t)
}

func TestRun_WithDynamicPrompt_ShouldUseFromEnv(t *testing.T) {
	app := NewMockApp()
	t.Setenv("DIFFAI_PROMPT_5", "dynamic prompt 5")
	app.Git().(*MockGitService).
		On("DiffStaged", mock.AnythingOfType("git.DiffOptions")).
		Return(git.DiffResult{
			Out:         []byte("diffout"),
			FullCommand: "fullcommand",
		}, nil)
	mockLLMClient := MockLLMClient{}
	app.LLM().(*MockLLMService).
		On("NewClient", llm.LLMProvider("ollama"), llm.LLMClientOptions{
			Model: "model",
		}).
		Return(&mockLLMClient, nil)
	app.Format().(*MockFormatClient).
		On("FormatMarkdown", "aires").Return("formated res", nil)
	mockLLMClient.
		On("Send", mock.Anything, []llm.Message{
			{
				Role:    llm.System,
				Content: "dynamic prompt 5",
				Hidden:  true,
			},
			{
				Role:    llm.User,
				Content: "diffout",
				Hidden:  true,
			},
		}).
		Return(&llm.LLMSendResponse{
			Content: "aires",
		}, nil)

	_, err := executeRootCommand(app, "--provider", "ollama", "--model=model", "--prompt", "5")
	if err != nil {
		t.Error(err)
	}
	app.Git().(*MockGitService).AssertExpectations(t)
	app.LLM().(*MockLLMService).AssertExpectations(t)
	app.Format().(*MockFormatClient).AssertExpectations(t)
}

func TestRun_WithFilters_ShouldSetDiffFilters(t *testing.T) {
	app := NewMockApp()
	t.Setenv("DIFFAI_PROMPT", "default prompt")
	wd, _ := os.Getwd()
	app.Git().(*MockGitService).
		On("DiffStaged", git.DiffOptions{
			CliPath:     "git",
			CliWd:       wd,
			Unified:     3,
			FindRenames: true,
			Filters:     []string{"*.ts", "*.go"},
		}).
		Return(git.DiffResult{
			Out:         []byte("diffout"),
			FullCommand: "fullcommand",
		}, nil)
	app.LLM().(*MockLLMService).
		On("NewClient", llm.LLMProvider("ollama"), llm.LLMClientOptions{
			Model: "model",
		}).
		Return(&MockLLMClient{}, fmt.Errorf("NewClient error"))

	_, err := executeRootCommand(app, "--provider", "ollama", "--model=model", "--diff-filters", "*.ts", "-f", "*.go")
	assert.Error(t, err)
	app.Git().(*MockGitService).AssertExpectations(t)
	app.LLM().(*MockLLMService).AssertExpectations(t)
}

func TestRun_WithInteractive_ShouldOpenChatMode(t *testing.T) {
	app := NewMockApp()
	app.Git().(*MockGitService).
		On("DiffStaged", mock.AnythingOfType("git.DiffOptions")).
		Return(git.DiffResult{
			Out:         []byte("diffout"),
			FullCommand: "fullcommand",
		}, nil)
	app.LLM().(*MockLLMService).
		On("NewClient", llm.LLMProvider("ollama"), llm.LLMClientOptions{
			Model: "model",
		}).
		Return(&MockLLMClient{}, nil)
	app.TUI().(*MockTUIService).
		On("InitialModel", mock.MatchedBy(func(model ui.InitialModelOptions) bool {
			return model.Title == "fullcommand" && model.Messages[0].Content == "prompt" && model.Messages[1].Content == "diffout"
		})).
		Return(ui.ChatTUIModel{})
	app.TUI().(*MockTUIService).
		On("Run", mock.AnythingOfType("ui.ChatTUIModel")).
		Return(ui.ChatTUIModel{}, nil)

	executeRootCommand(app, "--provider", "ollama", "--model=model", "-p=prompt", "-i")

	app.Git().(*MockGitService).AssertExpectations(t)
	app.LLM().(*MockLLMService).AssertExpectations(t)
	app.TUI().(*MockTUIService).AssertExpectations(t)
}

func TestRun_WithDiffTokenLimit_ShouldReturnError(t *testing.T) {
	app := NewMockApp()
	app.Git().(*MockGitService).
		On("DiffRefs", "diffFrom", "diffTo", mock.AnythingOfType("git.DiffOptions")).
		Return(git.DiffResult{
			Out:         []byte("diffMoreThanTheLimit"),
			FullCommand: "fullcommand",
		}, nil)

	_, err := executeRootCommand(app, "diffFrom", "diffTo", "--provider", "ollama", "--model=ollamam", "-p=prompt3", "--diff-token-limit", "5")
	assert.ErrorContains(t, err, "diff exceeds estimated token limit")
	app.Git().(*MockGitService).AssertExpectations(t)
}

func TestRun_WithEmptyDiff_ShouldReturnError(t *testing.T) {
	app := NewMockApp()
	app.Git().(*MockGitService).
		On("DiffStaged", mock.AnythingOfType("git.DiffOptions")).
		Return(git.DiffResult{
			Out:         []byte(""),
			FullCommand: "fullcommand",
		}, nil)

	_, err := executeRootCommand(app, "--provider", "ollama", "--model=model", "-p=prompt")
	assert.ErrorContains(t, err, "no diff content found")
	app.Git().(*MockGitService).AssertExpectations(t)
}

func TestRun_WithInvalidProvider_ShouldReturnError(t *testing.T) {
	app := NewMockApp()
	_, err := executeRootCommand(app, "--provider", "invalidprovider", "--model=model", "-p=prompt")
	assert.ErrorContains(t, err, "invalid provider")
}

func TestRun_WithNoModel_ShouldReturnError(t *testing.T) {
	app := NewMockApp()
	_, err := executeRootCommand(app, "--provider", "ollama", "-p=prompt")
	assert.ErrorContains(t, err, "model must be specified")
}

func TestRun_WithNoPrompt_ShouldReturnError(t *testing.T) {
	app := NewMockApp()
	_, err := executeRootCommand(app, "--provider", "ollama", "--model=model")
	assert.ErrorContains(t, err, "prompt must be specified")
}

func TestRun_WithInvalidDynamicPrompt_ShouldReturnError(t *testing.T) {
	app := NewMockApp()
	_, err := executeRootCommand(app, "--provider", "ollama", "--model=model", "--prompt", "1")
	assert.ErrorContains(t, err, "invalid instructions no, env variable not found PROMPT_1")
}

func TestRun_WithDiffError_ShouldReturnError(t *testing.T) {
	app := NewMockApp()
	app.Git().(*MockGitService).
		On("DiffStaged", mock.AnythingOfType("git.DiffOptions")).
		Return(git.DiffResult{}, fmt.Errorf("DiffStaged error"))

	_, err := executeRootCommand(app, "--provider", "ollama", "--model=model", "--prompt", "prompt")
	assert.ErrorContains(t, err, "error generating diff")
	app.Git().(*MockGitService).AssertExpectations(t)
}

func TestRun_WithLLMNewClientError_ShouldReturnError(t *testing.T) {
	app := NewMockApp()
	app.Git().(*MockGitService).
		On("DiffRefs", "diffFrom", "diffTo", mock.AnythingOfType("git.DiffOptions")).
		Return(git.DiffResult{
			Out:         []byte("diffout"),
			FullCommand: "fullcommand",
		}, nil)
	app.LLM().(*MockLLMService).
		On("NewClient", llm.LLMProvider("ollama"), mock.AnythingOfType("llm.LLMClientOptions")).
		Return(&MockLLMClient{}, fmt.Errorf("NewClient error"))

	_, err := executeRootCommand(app, "diffFrom", "diffTo", "--provider", "ollama", "--model=ollamam", "-p=prompt3")

	assert.ErrorContains(t, err, "failed to create LLM client")
	app.Git().(*MockGitService).AssertExpectations(t)
	app.LLM().(*MockLLMService).AssertExpectations(t)
}

func TestRun_WithLLMSendError_ShouldReturnError(t *testing.T) {
	app := NewMockApp()
	app.Git().(*MockGitService).
		On("DiffRefs", "diffFrom", "diffTo", mock.AnythingOfType("git.DiffOptions")).
		Return(git.DiffResult{
			Out:         []byte("diffout"),
			FullCommand: "fullcommand",
		}, nil)
	mockLLMClient := MockLLMClient{}
	app.LLM().(*MockLLMService).
		On("NewClient", llm.LLMProvider("ollama"), mock.AnythingOfType("llm.LLMClientOptions")).
		Return(&mockLLMClient, nil)
	mockLLMClient.On("Send", mock.Anything, mock.AnythingOfType("[]llm.Message")).
		Return(&llm.LLMSendResponse{}, fmt.Errorf("Send error"))

	_, err := executeRootCommand(app, "diffFrom", "diffTo", "--provider", "ollama", "--model=ollamam", "-p=prompt3")

	assert.ErrorContains(t, err, "failed to generate response")
	app.Git().(*MockGitService).AssertExpectations(t)
	app.LLM().(*MockLLMService).AssertExpectations(t)
}

func TestRun_WithLLMFormatError_ShouldReturnError(t *testing.T) {
	app := NewMockApp()
	app.Git().(*MockGitService).
		On("DiffRefs", "diffFrom", "diffTo", mock.AnythingOfType("git.DiffOptions")).
		Return(git.DiffResult{
			Out:         []byte("diffout"),
			FullCommand: "fullcommand",
		}, nil)
	mockLLMClient := MockLLMClient{}
	app.LLM().(*MockLLMService).
		On("NewClient", llm.LLMProvider("ollama"), mock.AnythingOfType("llm.LLMClientOptions")).
		Return(&mockLLMClient, nil)
	mockLLMClient.On("Send", mock.Anything, mock.AnythingOfType("[]llm.Message")).
		Return(&llm.LLMSendResponse{
			Content: "content",
		}, nil)
	app.Format().(*MockFormatClient).
		On("FormatMarkdown", "content").
		Return("", fmt.Errorf("FormatMarkdownError"))

	_, err := executeRootCommand(app, "diffFrom", "diffTo", "--provider", "ollama", "--model=ollamam", "-p=prompt3")

	assert.ErrorContains(t, err, "failed to format response")
	app.Git().(*MockGitService).AssertExpectations(t)
	app.LLM().(*MockLLMService).AssertExpectations(t)
	app.Format().(*MockFormatClient).AssertExpectations(t)
}

func TestMakeLLMBotResponder_Success(t *testing.T) {
	app := NewMockApp()
	mockLLMClient := MockLLMClient{}
	app.LLM().(*MockLLMService).
		On("NewClient", llm.LLMProvider("ollama"), llm.LLMClientOptions{
			Model: "model",
		}).
		Return(&mockLLMClient, nil)
	messages := []llm.Message{
		{Role: llm.User, Content: "What is Go?"},
	}
	expectedResponse := "Go is a statically typed programming language."

	mockLLMClient.
		On("Send", t.Context(), messages).
		Return(&llm.LLMSendResponse{Content: expectedResponse}, nil)

	responder := makeLLMBotResponder(&mockLLMClient, t.Context())
	cmd := responder(messages)

	msg := cmd()

	llmMsg, _ := msg.(llm.Message)
	assert.Equal(t, llm.Assistant, llmMsg.Role)
	assert.Equal(t, expectedResponse, llmMsg.Content)

	mockLLMClient.AssertExpectations(t)
}

func TestMakeLLMBotResponder_Error(t *testing.T) {
	app := NewMockApp()
	mockLLMClient := MockLLMClient{}
	app.LLM().(*MockLLMService).
		On("NewClient", llm.LLMProvider("ollama"), llm.LLMClientOptions{
			Model: "model",
		}).
		Return(&mockLLMClient, nil)
	messages := []llm.Message{
		{Role: llm.User, Content: "What is Go?"},
	}

	mockLLMClient.
		On("Send", t.Context(), messages).
		Return(&llm.LLMSendResponse{}, errors.New("Send error"))

	responder := makeLLMBotResponder(&mockLLMClient, t.Context())
	cmd := responder(messages)

	msg := cmd()

	llmMsg, _ := msg.(llm.Message)
	assert.Equal(t, llm.Assistant, llmMsg.Role)
	assert.Contains(t, llmMsg.Content, "Failed to generate response")

	mockLLMClient.AssertExpectations(t)
}

// func TestMakeLLMBotResponder_Error(t *testing.T) {
// 	mockClient := new(MockLLMClient)
// 	ctx := context.Background()

// 	messages := []llm.Message{
// 		{Role: llm.User, Content: "What is Go?"},
// 	}
// 	mockErr := errors.New("network failure")

// 	mockClient.On("Send", ctx, messages).
// 		Return(llm.AIResponse{}, mockErr)

// 	responder := makeLLMBotResponder(mockClient, ctx)
// 	cmd := responder(messages)

// 	msg := cmd()

// 	// Assert message
// 	llmMsg, ok := msg.(llm.Message)
// 	require.True(t, ok)
// 	assert.Equal(t, llm.Assistant, llmMsg.Role)
// 	assert.Contains(t, llmMsg.Content, "Failed to generate response: network failure")

// 	mockClient.AssertExpectations(t)
// }
