package cmd

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/klemjul/diffai/internal/config"
	"github.com/klemjul/diffai/internal/format"
	"github.com/klemjul/diffai/internal/git"
	"github.com/klemjul/diffai/internal/llm"
	"github.com/klemjul/diffai/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "diffai <commit1> [commit2]",
	Short: "Review git reference(s) with AI",
	Args:  cobra.RangeArgs(0, 2),
	Example: `
diffai main dev   # Review diff of two branches
diffai abc123 def456   # Review diff of two commits
diffai cdce10   # Review diff of a commit
diffai   # Review diff of staged changes
	`,
	Run: run,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		provider := viper.GetString("PROVIDER")
		if !slices.Contains(llm.LLMProviders, llm.LLMProvider(provider)) {
			return fmt.Errorf("invalid provider '%s'. Valid providers are: %v", provider, llm.LLMProviders)
		}

		model := viper.GetString("MODEL")
		if model == "" {
			return fmt.Errorf("model must be specified '%s'", provider)
		}

		return nil
	},
}

func Execute() error {
	rootCmd.Flags().Int("token-limit", config.DEFAULT_TOKEN_LIMIT, "Maximum number of tokens to include in LLM requests")
	rootCmd.Flags().String("instructions", config.DEFAULT_INSTRUCTIONS, "Include review instructions in the LLM prompt")
	rootCmd.Flags().String("provider", "", "LLM provider to use (e.g., openai, ollama, etc.)")
	rootCmd.Flags().String("model", "", "LLM model to use depend on the provider")
	rootCmd.Flags().Bool("chat", false, "Run diffai in Chat Mode")

	viper.BindPFlag("TOKEN_LIMIT", rootCmd.Flags().Lookup("token-limit"))
	viper.BindPFlag("INSTRUCTIONS", rootCmd.Flags().Lookup("instructions"))
	viper.BindPFlag("PROVIDER", rootCmd.Flags().Lookup("provider"))
	viper.BindPFlag("MODEL", rootCmd.Flags().Lookup("model"))
	viper.BindPFlag("CHAT", rootCmd.Flags().Lookup("chat"))

	viper.SetEnvPrefix(config.ENV_PREFIX)
	viper.AutomaticEnv()

	return rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) {
	var diffRes git.DiffResult
	var err error
	workingDirectory, err := os.Getwd()
	if err != nil {
		cmd.PrintErrf("Error getting current working directory: %v\n", err)
		os.Exit(1)
		return
	}

	options := git.DiffOptions{
		CliPath:     "git",
		CliWd:       workingDirectory,
		Unified:     3,
		FindRenames: true,
		Filters:     []string{},
	}

	switch len(args) {
	case 2:
		to, from := args[0], args[1]
		diffRes, err = git.DiffRefs(to, from, options)
	case 1:
		ref := args[0]
		diffRes, err = git.DiffCommit(ref, options)
	default:
		diffRes, err = git.DiffStaged(options)
	}

	if err != nil {
		cmd.PrintErrf("Error generating diff: %v\n", err)
		os.Exit(1)
		return
	}

	diffContent := string(diffRes.Out)
	tokenLimit := viper.GetInt("TOKEN_LIMIT")
	model := viper.GetString("MODEL")
	provider := viper.GetString("PROVIDER")
	instructions := viper.GetString("INSTRUCTIONS")
	chat := viper.GetBool("CHAT")

	if llm.RoughEstimateCodeTokens(diffContent) > tokenLimit {
		cmd.PrintErrf("Diff exceeds estimated token limit of %d tokens. Please reduce the diff size or extend token limit.\n", tokenLimit)
		os.Exit(1)
		return
	}

	if strings.TrimSpace(diffContent) == "" {
		cmd.PrintErrf("No diff content found. Please ensure you have staged changes or valid git references.\n")
		os.Exit(1)
		return
	}
	client, err := llm.NewClient(llm.LLMProvider(provider), llm.LLMClientOptions{
		Model: model,
	})

	if err != nil {
		cmd.PrintErrf("Error creating LLM client: %v\n", err)
		os.Exit(1)
		return
	}

	if !chat {
		aiRes, err := client.Send(context.TODO(), []llm.Message{
			{
				Role:    llm.System,
				Content: instructions,
			},
			{
				Role:    llm.System,
				Content: diffContent,
			},
		})
		if err != nil {
			cmd.PrintErrf("Error generating review: %v\n", err)
			os.Exit(1)
			return
		}
		formatRes, err := format.FormatMarkdown(aiRes.Content)
		if err != nil {
			cmd.PrintErrf("Error generating review: %v\n", err)
			os.Exit(1)
			return
		}

		fmt.Println(formatRes)
		return
	}

	chatModel := ui.InitialModel(ui.InitialModelOptions{
		Title: fmt.Sprintf("💻 Reviewing %s with %s model from %s provider", diffRes.FullCommand, model, provider),
		Messages: []llm.Message{
			{
				Role:    llm.System,
				Content: instructions,
			},
			{
				Role:    llm.System,
				Content: diffContent,
			},
		},
		GetBotResponse: func(messages []llm.Message) tea.Cmd {
			return func() tea.Msg {
				aiRes, err := client.Send(context.TODO(), messages)

				if err != nil {
					return llm.Message{
						Role:    llm.Assistant,
						Content: "Failed to generate response: " + err.Error(),
					}
				}

				return llm.Message{
					Role:    llm.Assistant,
					Content: aiRes.Content,
				}
			}
		}})
	if _, err := tea.NewProgram(chatModel).Run(); err != nil {
		cmd.PrintErrf("Error running chat: %v\n", err)
		os.Exit(1)
		return
	}
}
