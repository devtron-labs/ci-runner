package helper

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

func TestCommandExecutor(t *testing.T) {
	t.SkipNow()
	t.Run("execute cmd", func(t *testing.T) {
		//cmd := "/bin/sh -c 'ls -l ./package.json'"
		output, err := exec.Command("/bin/sh", "-c", "jq \".engines.node\" ./package.json").Output()
		fmt.Println(err)
		fmt.Println(string(output))
	})

	t.Run("read json file", func(t *testing.T) {
		readFile, _ := os.ReadFile("./buildpack.json")
		var mapData []*map[string]interface{}
		err := json.Unmarshal(readFile, &mapData)
		map1 := *mapData[0]
		value := map1["builderPrefix"]
		m2 := value.(map[string]interface{})
		value1 := m2["key"]
		fmt.Println(err)
		fmt.Println(value1)
	})
}
