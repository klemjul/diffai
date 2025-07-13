package git

import (
	"fmt"
	"strings"
)

var (
	execCommander = newExecCommander
)

type DiffOptions struct {
	CliPath     string
	CliWd       string
	Unified     int
	FindRenames bool
	Filters     []string
}

type DiffResult struct {
	Out         []byte
	FullCommand string
}

func DiffStaged(diffOptions DiffOptions) (DiffResult, error) {
	args := buildGenericArgs([]string{"diff", "--cached"}, diffOptions)
	return runCli(diffOptions.CliPath, diffOptions.CliWd, args...)
}

func DiffRefs(refFrom string, refTo string, diffOptions DiffOptions) (DiffResult, error) {
	args := buildGenericArgs([]string{"diff", refFrom, refTo}, diffOptions)
	return runCli(diffOptions.CliPath, diffOptions.CliWd, args...)
}

func DiffCommit(ref string, diffOptions DiffOptions) (DiffResult, error) {
	args := buildGenericArgs([]string{"show", ref}, diffOptions)
	return runCli(diffOptions.CliPath, diffOptions.CliWd, args...)
}

func runCli(cliPath string, dirName string, args ...string) (DiffResult, error) {
	cmd := execCommander(cliPath, args...)
	cmd.SetDir(dirName)
	out, err := cmd.CombinedOutput()

	return DiffResult{
		Out:         out,
		FullCommand: strings.Join(cmd.GetArgs(), " "),
	}, err
}

func buildGenericArgs(base []string, options DiffOptions) []string {
	args := append([]string{}, base...)

	args = append(args, fmt.Sprintf("--unified=%v", options.Unified))

	if options.FindRenames {
		args = append(args, "--find-renames")
	}

	if len(options.Filters) > 0 {
		args = append(args, "--")
		args = append(args, options.Filters...)
	}

	return args
}
