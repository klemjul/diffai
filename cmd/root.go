package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/klemjul/diffai/internal/config"
	"github.com/klemjul/diffai/internal/git"
	"github.com/klemjul/diffai/internal/llm"
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
}

func Execute() error {
	rootCmd.Flags().Int("token-limit", config.DEFAULT_TOKEN_LIMIT, "Maximum number of tokens to include in LLM requests.")
	rootCmd.Flags().Bool("stream", config.DEFAULT_STREAM, "Stream output to stdout as it is generated (true), or wait until complete (false)")
	rootCmd.Flags().String("instructions", config.DEFAULT_INSTRUCTIONS, "Include review instructions in the LLM prompt")

	viper.BindPFlag("TOKEN_LIMIT", rootCmd.Flags().Lookup("token-limit"))
	viper.BindPFlag("INSTRUCTIONS", rootCmd.Flags().Lookup("instructions"))
	viper.BindPFlag("STREAM", rootCmd.Flags().Lookup("stream"))

	viper.SetEnvPrefix(config.ENV_PREFIX)
	viper.AutomaticEnv()

	return rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) {
	var output []byte
	var err error

	workingDirectory, err := os.Getwd()
	if err != nil {
		cmd.PrintErrf("Error getting current working directory: %v\n", err)
		os.Exit(1)
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
		output, err = git.DiffRefs(to, from, options)
	case 1:
		ref := args[0]
		output, err = git.DiffCommit(ref, options)
	default:
		output, err = git.DiffStaged(options)
	}

	if err != nil {
		cmd.PrintErrf("Error generating diff: %v\n", err)
		os.Exit(1)
	}

	diffContent := string(output)
	tokenLimit := viper.GetInt("TOKEN_LIMIT")
	stream := viper.GetBool("STREAM")
	instructions := viper.GetString("INSTRUCTIONS")
	prompt := diffContent

	if llm.RoughEstimateCodeTokens(diffContent) > tokenLimit {
		cmd.PrintErrf("Diff exceeds estimated token limit of %d tokens. Please reduce the diff size or extend token limit.\n", tokenLimit)
		os.Exit(1)
	}

	if strings.TrimSpace(diffContent) == "" {
		cmd.PrintErrf("No diff content found. Please ensure you have staged changes or valid git references.\n")
		os.Exit(1)
	}

	if instructions != "" {
		prompt = fmt.Sprintf("%s\n\n%s", instructions, diffContent)
	}

	if stream {
		_, err = llm.Generate(prompt, tokenLimit, func(token string) {
			os.Stdout.Write([]byte(token))
		})
		if err != nil {
			cmd.PrintErrf("Error streaming review: %v\n", err)
			os.Exit(1)
		}
	} else {
		llmOutput, err := llm.Generate(prompt, tokenLimit, nil)
		fmt.Println(llmOutput)
		if err != nil {
			cmd.PrintErrf("Error generating review: %v\n", err)
			os.Exit(1)
		}
	}
}
