package git

import (
	"os/exec"
)

type Command interface {
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

func newExecCommander(name string, arg ...string) Command {
	execCmd := exec.Command(name, arg...)
	return execCommand{Cmd: execCmd}
}
