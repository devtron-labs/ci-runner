package utils

import (
	"bytes"
	"fmt"
	"github.com/devtron-labs/common-lib/utils/secretScanner"
	"io"
	"os"
	"os/exec"
)

var maskSecrets = true

func RunCommand(cmd *exec.Cmd) error {

	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		fmt.Printf("Command execution failed: %v\n", err)
		return err
	}

	if maskSecrets {
		buf := new(bytes.Buffer)
		// Call the function to mask secrets and print the masked output
		maskedStream, err := secretScanner.MaskSecretsStream(&outBuf)
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
