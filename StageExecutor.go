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

func RunPreDockerBuildTasks(ciRequest *CiRequest, scriptEnvs map[string]string, taskYaml *TaskYaml) error {
	//before task
	beforeTaskMap := make(map[string]*Task)
	for i, task := range ciRequest.BeforeDockerBuild {
		task.runStatus = true
		beforeTaskMap[task.Name] = task
		log.Println(devtron, "pre", task)
		//log running cmd
		logStage(task.Name)
		_, err := RunScripts(output_path, fmt.Sprintf("before-%d", i), task.Script, scriptEnvs, nil)
		if err != nil {
			return err
		}
	}

	beforeYamlTasks, err := GetBeforeDockerBuildTasks(ciRequest, taskYaml)
	if err != nil {
		log.Println(err)
		return err
	}

	// run before yaml tasks
	for i, task := range beforeYamlTasks {
		if _, ok := beforeTaskMap[task.Name]; ok {
			log.Println("duplicate task found in yaml, ran earlier so ignoring")
			continue
		}
		beforeTaskMap[task.Name] = task
		task.runStatus = true
		log.Println(devtron, "pre - yaml", task)
		//log running cmd
		logStage(task.Name)
		_, err = RunScripts(output_path, fmt.Sprintf("before-yaml-%d", i), task.Script, scriptEnvs, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func RunPostDockerBuildTasks(ciRequest *CiRequest, scriptEnvs map[string]string, taskYaml *TaskYaml) error {
	log.Println(devtron, " docker-build-post-processing")
	afterTaskMap := make(map[string]*Task)
	for i, task := range ciRequest.AfterDockerBuild {
		task.runStatus = true
		afterTaskMap[task.Name] = task
		log.Println(devtron, "post", task)
		logStage(task.Name)
		_, err := RunScripts(output_path, fmt.Sprintf("after-%d", i), task.Script, scriptEnvs, nil)
		if err != nil {
			return err
		}
	}

	afterYamlTasks, err := GetAfterDockerBuildTasks(ciRequest, taskYaml)
	if err != nil {
		log.Println(err)
		return err
	}

	for i, task := range afterYamlTasks {
		if _, ok := afterTaskMap[task.Name]; ok {
			log.Println("duplicate task found in yaml, already run so ignoring")
			continue
		}
		afterTaskMap[task.Name] = task
		task.runStatus = true
		log.Println(devtron, "post - yaml", task)
		//log running cmd
		logStage(task.Name)
		_, err = RunScripts(output_path, fmt.Sprintf("after-yaml-%d", i), task.Script, scriptEnvs, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func RunCdStageTasks(tasks []*Task, scriptEnvs map[string]string) error {
	log.Println(devtron, " cd-stage-processing")
	taskMap := make(map[string]*Task)
	for i, task := range tasks {
		if _, ok := taskMap[task.Name]; ok {
			log.Println("duplicate task found in yaml, already run so ignoring")
			continue
		}
		task.runStatus = true
		taskMap[task.Name] = task
		log.Println(devtron, "stage", task)
		logStage(task.Name)
		_, err := RunScripts(output_path, fmt.Sprintf("stage-%d", i), task.Script, scriptEnvs, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

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
