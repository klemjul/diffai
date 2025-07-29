package git

import (
	"os/exec"
)

type command interface {
	CombinedOutput() ([]byte, error)
	SetDir(string)
	GetArgs() []string
}

type execCommand struct {
	*exec.Cmd
}

func (exc execCommand) SetDir(dir string) {
	exc.Dir = dir
}

func (exc execCommand) GetArgs() []string {
	return exc.Args
}

func newExecCommander(name string, args ...string) command {
	execCmd := exec.Command(name, args...)
	return execCommand{Cmd: execCmd}
}
