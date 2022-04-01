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
