package secretScanner

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// MaskSecretsOnString takes an input string and masks any secrets found based on the provided rules
func MaskSecretsOnString(input string, rules []Rule) string {
	maskedInput := input

	for _, rule := range rules {
		maskedInput = rule.Regex.ReplaceAllString(maskedInput, "******")

	}
	return maskedInput
}

func scan() {
	// Create a new command to execute `cat` to read the file
	cmd := exec.Command("cat", "synthetic_log_data.txt")

	// Create a buffer to capture the command's stdout
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = os.Stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		fmt.Printf("Command execution failed: %v\n", err)
		os.Exit(1)
	}

	// Call the function to mask secrets and print the masked output
	buf := new(bytes.Buffer)
	maskedStream, err := MaskSecretsStream(&outBuf)
	if err != nil {
		fmt.Printf("Error masking secrets: %v\n", err)
		os.Exit(1)
	}
	_, er := buf.ReadFrom(maskedStream)
	if er != nil {
		fmt.Printf("Error reading from masked stream: %v\n", er)
		os.Exit(1)
	}
	fmt.Println(buf.String())
}

func MaskSecretsStream(input *bytes.Buffer) (io.Reader, error) {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		scanner := bufio.NewScanner(input)
		const maxCapacity int = 256 * 1024 // 256KB
		buf := make([]byte, maxCapacity)
		scanner.Buffer(buf, maxCapacity)

		for scanner.Scan() {
			line := scanner.Text()
			if len(line) == 0 {
				_, err := pw.Write([]byte("\n"))
				if err != nil {
					// handle error appropriately
					return
				}
			} else {
				maskedString := MaskSecretsOnString(line, BuiltinRules)
				_, err := pw.Write([]byte(maskedString + "\n"))
				if err != nil {
					// handle error appropriately
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			if errors.Is(err, bufio.ErrTooLong) {
				for {
					n, err := input.Read(buf)
					if err != nil {
						if err == io.EOF {
							break
						}
						// handle error appropriately
						return
					}
					line := string(buf[:n])
					maskedString := MaskSecretsOnString(line, BuiltinRules)
					_, err = pw.Write([]byte(maskedString + "\n"))
					if err != nil {
						// handle error appropriately
						return
					}
				}
			} else {
				// handle other errors appropriately
				return
			}
		}
	}()

	return pr, nil
}
