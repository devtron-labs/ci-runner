package utils

import (
	"fmt"
	"github.com/devtron-labs/common-lib/utils/secretScanner"
	"io"
	"os"
	"os/exec"
)

var maskSecrets = true

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
