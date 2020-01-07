package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
		err := RunScripts(output_path, fmt.Sprintf("before-%d", i), task.Script, scriptEnvs)
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
		err = RunScripts(output_path, fmt.Sprintf("before-yaml-%d", i), task.Script, scriptEnvs)
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
		err := RunScripts(output_path, fmt.Sprintf("after-%d", i), task.Script, scriptEnvs)
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
		err = RunScripts(output_path, fmt.Sprintf("after-yaml-%d", i), task.Script, scriptEnvs)
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
		err := RunScripts(output_path, fmt.Sprintf("stage-%d", i), task.Script, scriptEnvs)
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
		log.Println(devtron, err)
		return err
	}
	err = os.MkdirAll(outputPath, os.ModePerm|os.ModeDir)
	if err != nil {
		log.Println(devtron, err)
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
	log.Println(devtron, scriptPath)
	if err != nil {
		log.Println(devtron, err)
		return err
	}

	runScriptCMD := exec.Command("/bin/sh", scriptPath)
	err = RunCommand(runScriptCMD)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
