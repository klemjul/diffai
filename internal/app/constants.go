package app

import "fmt"

type EnvKey = string

const (
	EnvDiffTokenLimit EnvKey = "DIFF_TOKEN_LIMIT"
	EnvModel          EnvKey = "MODEL"
	EnvProvider       EnvKey = "PROVIDER"
	EnvPrompt         EnvKey = "PROMPT"
)

const (
	DefaultDiffTokenLimit = 100_000
	AppKey                = "DIFFAI"
)

func GetEnvWithPrefix(env EnvKey) string {
	return fmt.Sprintf("%s_%s", AppKey, env)
}
