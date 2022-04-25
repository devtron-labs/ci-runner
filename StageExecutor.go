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
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
)

func RunPreDockerBuildTasks(ciRequest *helper.CiRequest, scriptEnvs map[string]string, taskYaml *helper.TaskYaml) error {
	//before task
	beforeTaskMap := make(map[string]*helper.Task)
	for i, task := range ciRequest.BeforeDockerBuild {
		task.RunStatus = true
		beforeTaskMap[task.Name] = task
		log.Println(util.DEVTRON, "pre", task)
		//log running cmd
		util.LogStage(task.Name)
		err := RunScripts(util.Output_path, fmt.Sprintf("before-%d", i), task.Script, scriptEnvs)
		if err != nil {
			return err
		}
	}

	beforeYamlTasks, err := helper.GetBeforeDockerBuildTasks(ciRequest, taskYaml)
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
		task.RunStatus = true
		log.Println(util.DEVTRON, "pre - yaml", task)
		//log running cmd
		util.LogStage(task.Name)
		err = RunScripts(util.Output_path, fmt.Sprintf("before-yaml-%d", i), task.Script, scriptEnvs)
		if err != nil {
			return err
		}
	}
	return nil
}

func RunPostDockerBuildTasks(ciRequest *helper.CiRequest, scriptEnvs map[string]string, taskYaml *helper.TaskYaml) error {
	log.Println(util.DEVTRON, " docker-build-post-processing")
	afterTaskMap := make(map[string]*helper.Task)
	for i, task := range ciRequest.AfterDockerBuild {
		task.RunStatus = true
		afterTaskMap[task.Name] = task
		log.Println(util.DEVTRON, "post", task)
		util.LogStage(task.Name)
		err := RunScripts(util.Output_path, fmt.Sprintf("after-%d", i), task.Script, scriptEnvs)
		if err != nil {
			return err
		}
	}

	afterYamlTasks, err := helper.GetAfterDockerBuildTasks(ciRequest, taskYaml)
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
		task.RunStatus = true
		log.Println(util.DEVTRON, "post - yaml", task)
		//log running cmd
		util.LogStage(task.Name)
		err = RunScripts(util.Output_path, fmt.Sprintf("after-yaml-%d", i), task.Script, scriptEnvs)
		if err != nil {
			return err
		}
	}
	return nil
}

func RunCdStageTasks(tasks []*helper.Task, scriptEnvs map[string]string) error {
	log.Println(util.DEVTRON, " cd-stage-processing")
	taskMap := make(map[string]*helper.Task)
	for i, task := range tasks {
		if _, ok := taskMap[task.Name]; ok {
			log.Println("duplicate task found in yaml, already run so ignoring")
			continue
		}
		task.RunStatus = true
		taskMap[task.Name] = task
		log.Println(util.DEVTRON, "stage", task)
		util.LogStage(task.Name)
		err := RunScripts(util.Output_path, fmt.Sprintf("stage-%d", i), task.Script, scriptEnvs)
		if err != nil {
			return err
		}
	}
	return nil
}

func RunScripts(outputPath string, bashScript string, script string, envVars map[string]string) error {
	log.Println("running script commands")
	scriptTemplate := `#!/bin/sh
{{ range $key, $value := .envVr }}
export {{ $key }}={{ $value }} ;
{{ end }}
{{.script}}
`

	templateData := make(map[string]interface{})
	templateData["envVr"] = envVars
	templateData["script"] = script
	finalScript, err := Tprintf(scriptTemplate, templateData)
	if err != nil {
		log.Println(util.DEVTRON, err)
		return err
	}
	err = os.MkdirAll(outputPath, os.ModePerm|os.ModeDir)
	if err != nil {
		log.Println(util.DEVTRON, err)
		return err
	}
	scriptPath := filepath.Join(outputPath, bashScript)
	file, err := os.Create(scriptPath)
	if err != nil {
		log.Println(err)
		return err
	}
	defer file.Close()
	_, err = file.WriteString(finalScript)
	//log.Println(devtron, "final script ", finalScript) removed it shows some part on ui
	log.Println(util.DEVTRON, scriptPath)
	if err != nil {
		log.Println(util.DEVTRON, err)
		return err
	}

	runScriptCMD := exec.Command("/bin/sh", scriptPath)
	err = util.RunCommand(runScriptCMD)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
