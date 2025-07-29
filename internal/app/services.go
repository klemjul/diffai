package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/klemjul/diffai/internal/format"
	"github.com/klemjul/diffai/internal/git"
	"github.com/klemjul/diffai/internal/llm"
	"github.com/klemjul/diffai/internal/ui"
)

type GitService interface {
	DiffStaged(diffOptions git.DiffOptions) (git.DiffResult, error)
	DiffRefs(refFrom string, refTo string, diffOptions git.DiffOptions) (git.DiffResult, error)
	DiffCommit(ref string, diffOptions git.DiffOptions) (git.DiffResult, error)
}

type TUIService interface {
	InitialModel(opts ui.InitialModelOptions) ui.ChatTUIModel
	Run(model ui.ChatTUIModel) (returnModel tea.Model, returnErr error)
}

type LLMService interface {
	NewClient(provider llm.LLMProvider, opts llm.LLMClientOptions) (llm.LLMClient, error)
}

type TextFormatService interface {
	FormatMarkdown(text string) (string, error)
}

type ServiceProvider interface {
	Git() GitService
	TUI() TUIService
	LLM() LLMService
	Format() TextFormatService
}

type DefaultGitService struct{}

type DefaultTUIService struct{}

type DefaultLLMService struct{}

type DefaultTextFormatService struct{}

type DefaultServiceProvider struct {
	git    GitService
	tui    TUIService
	llm    LLMService
	format TextFormatService
}

func (p *DefaultServiceProvider) Git() GitService           { return p.git }
func (p *DefaultServiceProvider) TUI() TUIService           { return p.tui }
func (p *DefaultServiceProvider) LLM() LLMService           { return p.llm }
func (p *DefaultServiceProvider) Format() TextFormatService { return p.format }

func (s *DefaultGitService) DiffStaged(diffOptions git.DiffOptions) (git.DiffResult, error) {
	return git.DiffStaged(diffOptions)
}
func (s *DefaultGitService) DiffRefs(refFrom string, refTo string, diffOptions git.DiffOptions) (git.DiffResult, error) {
	return git.DiffRefs(refFrom, refTo, diffOptions)
}
func (s *DefaultGitService) DiffCommit(ref string, diffOptions git.DiffOptions) (git.DiffResult, error) {
	return git.DiffCommit(ref, diffOptions)
}

func (s *DefaultTUIService) InitialModel(opts ui.InitialModelOptions) ui.ChatTUIModel {
	return ui.InitialModel(opts)
}
func (s *DefaultTUIService) Run(model ui.ChatTUIModel) (returnModel tea.Model, returnErr error) {
	return tea.NewProgram(model).Run()
}

func (s *DefaultLLMService) NewClient(provider llm.LLMProvider, opts llm.LLMClientOptions) (llm.LLMClient, error) {
	return llm.NewClient(provider, opts)
}

func (s *DefaultTextFormatService) FormatMarkdown(text string) (string, error) {
	return format.FormatMarkdown(text)
}

func NewDefaultServiceProvider() ServiceProvider {
	return &DefaultServiceProvider{git: &DefaultGitService{}, tui: &DefaultTUIService{}, llm: &DefaultLLMService{}, format: &DefaultTextFormatService{}}
}
