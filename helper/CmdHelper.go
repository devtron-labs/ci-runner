package helper

import "os/exec"

type CmdHelper interface {
	GetCommandToExecute(cmd string) *exec.Cmd
}

type CmdHelperImpl struct {
}

func NewCmdHelperImpl() *CmdHelperImpl {
	return &CmdHelperImpl{}
}

func (impl *CmdHelperImpl) GetCommandToExecute(cmd string) *exec.Cmd {
	execCmd := exec.Command("/bin/sh", "-c", cmd)
	return execCmd
}
