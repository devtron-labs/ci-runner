package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func RunScripts(workDirectory string, scriptFileName string, script string, envInputVars map[string]string, outputVars []string) (map[string]string, error) {
	log.Println("running script commands")
	envOutFileName := filepath.Join(workDirectory, fmt.Sprintf("%s_out.env", scriptFileName))

	//------------
	finalScript, err := prepareFinaleScript(script, outputVars, envOutFileName)
	if err != nil {
		log.Println(devtron, err)
		return nil, err
	}
	//--------------
	err = os.MkdirAll(workDirectory, os.ModePerm|os.ModeDir)
	if err != nil {
		log.Println(devtron, err)
		return nil, err
	}
	scriptPath := filepath.Join(workDirectory, scriptFileName)
	file, err := os.Create(scriptPath)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer file.Close()
	_, err = file.WriteString(finalScript)
	//log.Println(devtron, "final script ", finalScript) removed it shows some part on ui
	log.Println(devtron, scriptPath)
	if err != nil {
		log.Println(devtron, err)
		return nil, err
	}
	var inputEnvironmentVariable []string
	for k, v := range envInputVars {
		inputEnvironmentVariable = append(inputEnvironmentVariable, fmt.Sprintf("%s=%s", k, v))
	}

	runScriptCMD := exec.Command("/bin/sh", scriptPath)
	runScriptCMD.Env = inputEnvironmentVariable
	err = RunCommand(runScriptCMD)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	envMap, err := readEnvironmentFile(envOutFileName)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return envMap, nil
}

//prepare final shell script to be executed
func prepareFinaleScript(script string, outputVars []string, envOutFileName string) (string, error) {
	scriptTemplate := `
#!/bin/sh
set -e
set -o pipefail
{{.script}}
> {{.envOutFileName}}
{{$envOutFileName := .envOutFileName}}
{{range .outputVars}} 
  printf "\n{{.}}=%s" "${{.}}" >> {{$envOutFileName}}
{{end}}
`
	templateData := make(map[string]interface{})
	templateData["script"] = script
	templateData["outputVars"] = outputVars
	templateData["envOutFileName"] = envOutFileName
	finalScript, err := Tprintf(scriptTemplate, templateData)
	if err != nil {
		return "", err
	}
	return finalScript, nil
}

//read content of .env file
func readEnvironmentFile(filename string) (envMap map[string]string, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	envMap = parseEnvironmentFileLines(lines)
	return envMap, nil
}
func parseEnvironmentFileLines(lines []string) map[string]string {
	envMap := make(map[string]string)
	for _, fullLine := range lines {
		if !isIgnoredLine(fullLine) {
			parts := strings.SplitN(fullLine, "=", 2)
			if len(parts) != 2 {
				continue
			}
			envMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return envMap
}
func isIgnoredLine(line string) bool {
	trimmedLine := strings.TrimSpace(line)
	return len(trimmedLine) == 0 || strings.HasPrefix(trimmedLine, "#")
}
