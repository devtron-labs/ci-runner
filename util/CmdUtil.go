/*
 *  Copyright 2020 Devtron Labs
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package util

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
)

func DeleteFile(path string) error {
	var err = os.Remove(path)
	if err != nil {
		return err
	}
	return nil
}

func RunCommand(cmd *exec.Cmd) error {
	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(os.Stdout, &stdBuffer)
	cmd.Stdout = mw
	cmd.Stderr = mw
	if err := cmd.Run(); err != nil {
		return err
	}
	//log.Println(stdBuffer.String())
	return nil
}

type CommandType []string

func NewCommand(newArgs ...string) *CommandType {
	cmd := make(CommandType, 0)
	cmd.AppendCommand(newArgs...)
	return &cmd
}

func (c *CommandType) AppendCommand(newArgs ...string) {
	for _, newArg := range newArgs {
		trimmedArg := strings.TrimSpace(newArg)
		if trimmedArg != "" {
			*c = append(*c, trimmedArg)
		}
	}
}

func (c *CommandType) PrintCommand() string {
	if c == nil {
		return ""
	}
	return strings.Join(*c, " ")
}

func (c *CommandType) GetCommandToBeExecuted(initialArgs ...string) []string {
	runCmd := initialArgs
	if c == nil {
		return runCmd
	}
	runCmd = append(runCmd, *c...)
	return runCmd
}
