package git

import (
	"fmt"
	"os/exec"
)

type DiffOptions struct {
	CliPath     string
	CliWd       string
	Unified     int
	FindRenames bool
	Filters     []string
}

func DiffStaged(diffOptions DiffOptions) ([]byte, error) {
	args := buildGenericArgs([]string{"diff", "--cached"}, diffOptions)
	return runCli(diffOptions.CliPath, diffOptions.CliWd, args...)
}

func DiffRefs(refFrom string, refTo string, diffOptions DiffOptions) ([]byte, error) {
	args := buildGenericArgs([]string{"diff", refFrom, refTo}, diffOptions)
	return runCli(diffOptions.CliPath, diffOptions.CliWd, args...)
}

func DiffCommit(ref string, diffOptions DiffOptions) ([]byte, error) {
	args := buildGenericArgs([]string{"show", ref}, diffOptions)
	return runCli(diffOptions.CliPath, diffOptions.CliWd, args...)
}

func runCli(cliPath string, dirName string, args ...string) ([]byte, error) {
	fmt.Println(cliPath, args)
	cmd := exec.Command(cliPath, args...)
	cmd.Dir = dirName

	return cmd.CombinedOutput()
}

func buildGenericArgs(base []string, options DiffOptions) []string {
	args := append([]string{}, base...)

	args = append(args, fmt.Sprintf("--unified=%v", options.Unified))

	if options.FindRenames {
		args = append(args, "--find-renames")
	}

	if options.Filters != nil && len(options.Filters) > 0 {
		args = append(args, "--")
		for _, filter := range options.Filters {
			args = append(args, filter)
		}
	}

	return args
}
