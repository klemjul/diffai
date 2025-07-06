package cmd

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
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
			return fmt.Errorf("model must be specified '%s'", model)
		}
		prompt := viper.GetString("PROMPT")
		if prompt == "" {
			return fmt.Errorf("prompt must be specified '%s'", prompt)
		}

		return nil
	},
}

func Execute() error {
	rootCmd.Flags().SortFlags = false

	rootCmd.Flags().StringP("prompt", "p", "",
		fmt.Sprintf(
			`Includes review instructions as system prompt. (env: %s)
- If <value> is a string, it will override the default and be used directly as the instructions.
- If <value> is a number, it will look for the environment variable %s_<number> instead.
`, config.GetEnvWithPrefix(config.ENV_PROMPT), config.GetEnvWithPrefix(config.ENV_PROMPT)))
	rootCmd.Flags().String("provider", "",
		fmt.Sprintf("LLM provider to use. (env: %s)", config.GetEnvWithPrefix(config.ENV_PROVIDER)))
	rootCmd.Flags().String("model", "",
		fmt.Sprintf("LLM model to use, depend on the provider. (env: %s)", config.GetEnvWithPrefix(config.ENV_MODEL)))
	rootCmd.Flags().BoolP("interactive", "i", false, "Run diffai in Chat Mode.")
	rootCmd.Flags().Int("diff-token-limit", config.DEFAULT_DIFF_TOKEN_LIMIT,
		fmt.Sprintf("Maximum number of tokens for the diff content. (env: %s)", config.GetEnvWithPrefix(config.ENV_DIFF_TOKEN_LIMIT)))
	rootCmd.Flags().StringSliceP("diff-filters", "f", []string{}, "git diff -- <path> filters, used to limit the diff to the named paths or file exts")

	viper.BindPFlag(config.ENV_DIFF_TOKEN_LIMIT, rootCmd.Flags().Lookup("diff-token-limit"))
	viper.BindPFlag(config.ENV_PROMPT, rootCmd.Flags().Lookup("prompt"))
	viper.BindPFlag(config.ENV_PROVIDER, rootCmd.Flags().Lookup("provider"))
	viper.BindPFlag(config.ENV_MODEL, rootCmd.Flags().Lookup("model"))

	viper.SetEnvPrefix(config.ENV_PREFIX)
	viper.AutomaticEnv()

	return rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) {
	diffTokenLimit := viper.GetInt(config.ENV_DIFF_TOKEN_LIMIT)
	model := viper.GetString(config.ENV_MODEL)
	provider := viper.GetString(config.ENV_PROVIDER)
	prompt := viper.GetString(config.ENV_PROMPT)
	promptNo, err := strconv.Atoi(prompt)
	if err == nil {
		promptEnv := fmt.Sprintf("%s_%v", config.ENV_PROMPT, promptNo)
		prompt = viper.GetString(promptEnv)
		if prompt == "" {
			cmd.PrintErrf("invalid instructions no, env variable not found %s\n", promptEnv)
			os.Exit(1)
			return
		}
	}

	interactive, err := cmd.Flags().GetBool("interactive")
	if err != nil {
		interactive = false
	}

	diffFilters, err := cmd.Flags().GetStringSlice("diff-filters")
	if err != nil {
		diffFilters = []string{}
	}

	var diffRes git.DiffResult
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
		Filters:     diffFilters,
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

	if llm.RoughEstimateCodeTokens(diffContent) > diffTokenLimit {
		cmd.PrintErrf("Diff exceeds estimated token limit of %d tokens. Please reduce the diff size or extend token limit.\n", diffTokenLimit)
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

	initialMessages := []llm.Message{
		{
			Role:    llm.System,
			Content: prompt,
			Hidden:  true,
		},
		{
			Role:    llm.User,
			Content: diffContent,
			Hidden:  true,
		},
	}

	if !interactive {
		aiRes, err := client.Send(context.TODO(), initialMessages)
		if err != nil {
			cmd.PrintErrf("Failed to generate response: %v\n", err)
			os.Exit(1)
			return
		}
		formattedRes, err := format.FormatMarkdown(aiRes.Content)
		if err != nil {
			cmd.PrintErrf("Failed to format response: %v\n", err)
			os.Exit(1)
			return
		}
		fmt.Println(formattedRes)

	} else {
		chatModel := ui.InitialModel(ui.InitialModelOptions{
			Title:    fmt.Sprintf("💻 Reviewing %s with %s model from %s provider", diffRes.FullCommand, model, provider),
			Messages: initialMessages,
			GetBotResponse: func(messages []llm.Message) tea.Cmd {
				return func() tea.Msg {
					aiRes, err := client.Send(context.TODO(), messages)

					if err != nil {
						return llm.Message{
							Role:    llm.Assistant,
							Content: fmt.Sprintf("Failed to generate response: %v", err.Error()),
						}
					}

					return llm.Message{
						Role:    llm.Assistant,
						Content: aiRes.Content,
					}
				}
			}})
		if _, err := tea.NewProgram(chatModel).Run(); err != nil {
			cmd.PrintErrf("Error running interactive mode: %v\n", err)
			os.Exit(1)
			return
		}
	}
}
