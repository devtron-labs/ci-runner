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
	"fmt"
	"github.com/devtron-labs/common-lib/utils/secretScanner"
	"io"
	"os"
	"os/exec"
)

var maskSecrets = false

func DeleteFile(path string) error {
	var err = os.Remove(path)
	if err != nil {
		return err
	}
	return nil
}

func RunCommand(cmd *exec.Cmd) error {

	// Run the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Command execution failed: %v\n", err)
	}
	outBuf := bytes.NewBuffer(output)
	if maskSecrets {
		buf := new(bytes.Buffer)
		// Call the function to mask secrets and print the masked output
		maskedStream, err := secretScanner.MaskSecretsStream(outBuf)
		if err != nil {
			fmt.Printf("Error masking secrets: %v\n", err)
			return err
		}
		_, err = io.Copy(buf, maskedStream)
		if err != nil {
			fmt.Printf("Error reading from masked stream: %v\n", err)
			return err
		}
		fmt.Println(buf.String())
	} else {
		fmt.Println(outBuf.String())
	}
	return nil
}
