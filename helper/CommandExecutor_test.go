package helper

import (
	"fmt"
	"os/exec"
	"testing"
)

func TestCommandExecutor(t *testing.T) {
	t.Run("execute cmd", func(t *testing.T) {
		//cmd := "/bin/sh -c 'ls -l /Users/kripanshbanga/Desktop/devtron-workspace/dashboard/package.json'"
		output, err := exec.Command("/bin/sh", "-c", "jq \".engines.node\" /Users/kripanshbanga/Desktop/devtron-workspace/dashboard/package.json").Output()
		fmt.Println(err)
		fmt.Println(string(output))
	})
}
