package secretScanner

import (
	"bufio"
	"errors"
	"io"
	"log"
)

// MaskSecretsOnString takes an input string and masks any secrets found based on the provided rules
func MaskSecretsOnString(input string, rules []Rule) string {
	maskedInput := input

	for _, rule := range rules {
		maskedInput = rule.Regex.ReplaceAllString(maskedInput, "******")

	}
	return maskedInput
}

func MaskSecretsOnStream(input io.Reader) (io.Reader, error) {
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
					log.Printf("Error writing masked string to pipe: %v", err)
					pw.CloseWithError(err)
					return
				}
			} else {
				maskedString := MaskSecretsOnString(line, BuiltinRules)
				_, err := pw.Write([]byte(maskedString + "\n"))
				if err != nil {
					log.Printf("Error reading input buffer: %v", err)
					pw.CloseWithError(err)
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
						log.Printf("Error reading input buffer: %v", err)
						pw.CloseWithError(err)
						return
					}
					line := string(buf[:n])
					maskedString := MaskSecretsOnString(line, BuiltinRules)
					_, err = pw.Write([]byte(maskedString + "\n"))
					if err != nil {
						log.Printf("Error writing masked string to pipe: %v", err)
						pw.CloseWithError(err)
						return
					}
				}
			} else {
				log.Printf("Scanner error: %v", err)
				pw.CloseWithError(err)
				return
			}
		}
	}()

	return pr, nil
}
