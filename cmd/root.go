package cmd

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/klemjul/diffai/internal/app"
	"github.com/klemjul/diffai/internal/config"
	"github.com/klemjul/diffai/internal/git"
	"github.com/klemjul/diffai/internal/llm"
	"github.com/klemjul/diffai/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RootCommand(app app.App) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "diffai <commit1> [commit2]",
		Short: "Ask questions about git changes using AI in the command line.",
		Args:  cobra.RangeArgs(0, 2),
		Example: `
diffai main dev   # Review diff of two branches
diffai abc123 def456   # Review diff of two commits
diffai cdce10   # Review diff of a commit
diffai   # Review diff of staged changes
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, app)
		},
		PreRunE: validate,
	}

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
		fmt.Sprintf("LLM model to use, depends on the provider. (env: %s)", config.GetEnvWithPrefix(config.ENV_MODEL)))
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

	return rootCmd
}

func validate(cmd *cobra.Command, args []string) error {
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
}

func run(cmd *cobra.Command, args []string, app app.App) error {

	diffTokenLimit := viper.GetInt(config.ENV_DIFF_TOKEN_LIMIT)
	model := viper.GetString(config.ENV_MODEL)
	provider := viper.GetString(config.ENV_PROVIDER)
	prompt := viper.GetString(config.ENV_PROMPT)
	promptNo, err := strconv.Atoi(prompt)
	if err == nil {
		promptEnv := fmt.Sprintf("%s_%v", config.ENV_PROMPT, promptNo)
		prompt = viper.GetString(promptEnv)
		if prompt == "" {
			return fmt.Errorf("invalid instructions no, env variable not found %s", promptEnv)
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
	if len(diffFilters) >= 1 && diffFilters[0] == "[]" {
		diffFilters = diffFilters[1:]
	}

	var diffRes git.DiffResult
	workingDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting current working directory: %v", err)
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
		diffRes, err = app.Git().DiffRefs(to, from, options)
	case 1:
		ref := args[0]
		diffRes, err = app.Git().DiffCommit(ref, options)
	default:
		diffRes, err = app.Git().DiffStaged(options)
	}

	if err != nil {
		return fmt.Errorf("error generating diff: %v", err)
	}

	diffContent := string(diffRes.Out)

	if llm.RoughEstimateCodeTokens(diffContent) > diffTokenLimit {
		return fmt.Errorf("diff exceeds estimated token limit of %d tokens. Please reduce the diff size or extend token limit", diffTokenLimit)
	}

	if strings.TrimSpace(diffContent) == "" {
		return fmt.Errorf("no diff content found. Please ensure you have staged changes or valid git references")
	}
	client, err := app.LLM().NewClient(llm.LLMProvider(provider), llm.LLMClientOptions{
		Model: model,
	})

	if err != nil {
		return fmt.Errorf("failed to create LLM client: %v", err)
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
		aiRes, err := client.Send(cmd.Context(), initialMessages)
		if err != nil {
			return fmt.Errorf("failed to generate response: %v", err)
		}
		formattedRes, err := app.Format().FormatMarkdown(aiRes.Content)
		if err != nil {
			return fmt.Errorf("failed to format response: %v", err)
		}
		cmd.OutOrStdout().Write([]byte(formattedRes))

	} else {
		TUIModel := app.TUI().InitialModel(ui.InitialModelOptions{
			Title:          diffRes.FullCommand,
			Messages:       initialMessages,
			GetBotResponse: makeLLMBotResponder(client, cmd.Context()),
		})
		if _, err := app.TUI().Run(TUIModel); err != nil {
			return fmt.Errorf("error running interactive mode: %v", err)
		}
	}
	return nil
}

func makeLLMBotResponder(client llm.LLMClient, ctx context.Context) func([]llm.Message) tea.Cmd {
	return func(messages []llm.Message) tea.Cmd {
		return func() tea.Msg {
			aiRes, err := client.Send(ctx, messages)

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
	}
}
