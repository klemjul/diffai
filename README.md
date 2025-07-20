# DiffAI

A lightweight command-line tool that provides a simple way to ask questions about commits, staged changes, or diffs through seamless integration between Git and AI.

> ⚠️ **Early Development Notice**  
> This project is in early development. Features may change, break, or be incomplete. Use at your own risk.

## Features

- AI-Powered Diff Review: Get intelligent feedback on your Git diffs using multiple LLM providers
- Flexible Git Integration: Review staged changes, specific commits, or compare branches
- Interactive Chat Mode: Engage in conversations about your code changes
- Multiple LLM Providers: Support for various AI providers and models
- Customizable Prompts: Easily switch between custom review instructions
- Diff Filtering: Focus reviews on specific files or paths

## Installation

> **TODO**

## Usage

```man
Review git reference(s) with AI

Usage:
  diffai <commit1> [commit2] [flags]

Examples:

diffai main dev   # Review diff of two branches
diffai abc123 def456   # Review diff of two commits
diffai cdce10   # Review diff of a commit
diffai   # Review diff of staged changes


Flags:
  -p, --prompt string          Includes review instructions as system prompt. (env: DIFFAI_PROMPT)
                               - If <value> is a string, it will override the default and be used directly as the instructions.
                               - If <value> is a number, it will look for the environment variable DIFFAI_PROMPT_<number> instead.

      --provider string        LLM provider to use. (env: DIFFAI_PROVIDER)
      --model string           LLM model to use, depends on the provider. (env: DIFFAI_MODEL)
  -i, --interactive            Run diffai in Chat Mode.
      --diff-token-limit int   Maximum number of tokens for the diff content. (env: DIFFAI_DIFF_TOKEN_LIMIT) (default 100000)
  -f, --diff-filters strings   git diff -- <path> filters, used to limit the diff to the named paths or file exts
  -h, --help                   help for diffai
```

## Configuration

DiffAI can be configured through environment variables or command-line flags.

### Environment Variables

```bash
export DIFFAI_PROVIDER="openai"
export DIFFAI_MODEL="gpt-4.1"
export DIFFAI_PROMPT="Review this code for bugs, security issues, and best practices"
export DIFFAI_DIFF_TOKEN_LIMIT="200000"
```

### Predefined Prompts

You can use numbered prompts by setting environment variables.

```bash
export DIFFAI_PROMPT_1="Focus on security vulnerabilities"
export DIFFAI_PROMPT_2="Review for performance optimizations"
export DIFFAI_PROMPT_3="Check code style and maintainability"
```

Then use them with

```bash
diffai -p 1  # Uses DIFFAI_PROMPT_1
```

## Supported LLM Providers

`openai` and `ollama`

## License

DiffAI is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.

## Contributing

> **TODO**

## Next Steps

- [ ] Contributing doc
- [ ] Additional LLM provider support
- [ ] Custom output formats
