package helper

import (
	"github.com/devtron-labs/ci-runner/util"
	"os/exec"
)

type CommandExecutor interface {
	RunCommand(cmd *exec.Cmd) error
}

type CommandExecutorImpl struct {
}

func NewCommandExecutorImpl() *CommandExecutorImpl {
	return &CommandExecutorImpl{}
}

func (c *CommandExecutorImpl) RunCommand(cmd *exec.Cmd) error {
	return util.RunCommand(cmd)
}
