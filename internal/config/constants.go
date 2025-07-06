package config

import "fmt"

const (
	DEFAULT_DIFF_TOKEN_LIMIT = 100_000
	ENV_PREFIX               = "DIFFAI"
	ENV_DIFF_TOKEN_LIMIT     = "DIFF_TOKEN_LIMIT"
	ENV_MODEL                = "MODEL"
	ENV_PROVIDER             = "PROVIDER"
	ENV_PROMPT               = "PROMPT"
)

func GetEnvWithPrefix(env string) string {
	return fmt.Sprintf("%s_%s", ENV_PREFIX, env)
}
