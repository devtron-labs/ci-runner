package helper

import (
	"os/exec"

	cicxt "github.com/devtron-labs/ci-runner/executor/context"
	"github.com/devtron-labs/ci-runner/util"
)

type CommandExecutor interface {
	RunCommand(ctx cicxt.CiContext, cmd *exec.Cmd) error
}

type CommandExecutorImpl struct {
}

func NewCommandExecutorImpl() *CommandExecutorImpl {
	return &CommandExecutorImpl{}
}

func (c *CommandExecutorImpl) RunCommand(ctx cicxt.CiContext, cmd *exec.Cmd) error {
	return util.RunCommand(cmd)
}
