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
	"fmt"
	"github.com/devtron-labs/common-lib/utils/secretScanner"
	"io"
	"os"
	"os/exec"
)

var maskSecrets = true

func DeleteFile(path string) error {
	var err = os.Remove(path)
	if err != nil {
		return err
	}
	return nil
}

func RunCommand(cmd *exec.Cmd) error {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error creating stdout pipe: %v\n", err)
		return err
	}
	cmd.Stderr = cmd.Stdout // Combine stderr and stdout

	// Start the command
	if err := cmd.Start(); err != nil {
		fmt.Printf("Command execution failed: %v\n", err)
		return err
	}

	if maskSecrets {
		// Wrap the pipe reader to mask secrets
		maskedStream, err := secretScanner.MaskSecretsOnStream(stdoutPipe)
		if err != nil {
			fmt.Printf("error masking secrets: %v", err)
			return err
		}

		// Copy the masked stream to stdout
		if _, err := io.Copy(os.Stdout, maskedStream); err != nil {
			fmt.Printf("error reading masked stream: %v", err)
			return err
		}

	} else {
		if _, err := io.Copy(os.Stdout, stdoutPipe); err != nil {
			fmt.Printf("error reading stream: %v", err)
			return err
		}

	}

	// Wait for the command to complete
	if err := cmd.Wait(); err != nil {
		fmt.Printf("Command execution failed: %v\n", err)
	}

	return nil
}
