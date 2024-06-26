/*
 * Copyright (c) 2024. Devtron Inc.
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
 */

package helper

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/devtron-labs/ci-runner/util"
	"gopkg.in/yaml.v2"
)

type Task struct {
	Id             int    `json:"id"`
	Index          int    `json:"index"`
	Name           string `json:"name" yaml:"name"`
	Script         string `json:"script" yaml:"script"`
	OutputLocation string `json:"outputLocation" yaml:"outputLocation"` // file/dir
	RunStatus      bool   `json:"-"`                                    // task run was attempted or not
}

type TaskYaml struct {
	Version          string             `yaml:"version"`
	PipelineConf     []PipelineConfig   `yaml:"pipelineConf"`
	CdPipelineConfig []CdPipelineConfig `yaml:"cdPipelineConf"`
}

type PipelineConfig struct {
	AppliesTo   []AppliesTo `yaml:"appliesTo"`
	BeforeTasks []*Task     `yaml:"beforeDockerBuildStages"`
	AfterTasks  []*Task     `yaml:"afterDockerBuildStages"`
}

type CdPipelineConfig struct {
	BeforeTasks []*Task `yaml:"beforeStages"`
	AfterTasks  []*Task `yaml:"afterStages"`
}

type AppliesTo struct {
	Type  string   `yaml:"type"`
	Value []string `yaml:"value"`
}

const BRANCH_FIXED = "BRANCH_FIXED"

func GetBeforeDockerBuildTasks(ciRequest *CommonWorkflowRequest, taskYaml *TaskYaml) ([]*Task, error) {
	if taskYaml == nil {
		log.Println(util.DEVTRON, "no tasks, devtron-ci yaml missing")
		return nil, nil
	}

	if taskYaml.Version != "0.0.1" {
		log.Println("invalid version for devtron-ci.yaml")
		return nil, errors.New("invalid version for devtron-ci.yaml")
	}

	pipelineConfig := taskYaml.PipelineConf
	log.Println(util.DEVTRON, "pipelineConf length: ", len(pipelineConfig))

	var tasks []*Task
	filteredOut := false
	for _, p := range pipelineConfig {
		if filteredOut {
			break
		}
		for _, a := range p.AppliesTo {
			triggerType := a.Type
			if triggerType == BRANCH_FIXED {
				branches := a.Value
				branchesMap := make(map[string]bool)
				for _, b := range branches {
					branchesMap[b] = true
				}
				if !isValidBranch(ciRequest, a) {
					log.Println(util.DEVTRON, "skipping current AppliesTo")
					continue
				}
				tasks = append(tasks, p.BeforeTasks...)
				filteredOut = true
			} else {
				log.Println(util.DEVTRON, "unknown triggerType ", triggerType)
			}
		}

	}
	return tasks, nil
}

func GetAfterDockerBuildTasks(ciRequest *CommonWorkflowRequest, taskYaml *TaskYaml) ([]*Task, error) {
	if taskYaml == nil {
		log.Println(util.DEVTRON, "no tasks, devtron-ci yaml missing")
		return nil, nil
	}

	if taskYaml.Version != "0.0.1" { // TODO: Get version from ciRequest based on ci_pipeline
		log.Println("invalid version for devtron-ci.yaml")
		return nil, errors.New("invalid version for devtron-ci.yaml")
	}

	pipelineConfig := taskYaml.PipelineConf
	log.Println(util.DEVTRON, "pipelineConf length: ", len(pipelineConfig))

	var tasks []*Task
	filteredOut := false
	for _, p := range pipelineConfig {
		if filteredOut {
			break
		}
		for _, a := range p.AppliesTo {
			triggerType := a.Type
			if triggerType == BRANCH_FIXED {
				isValidSourceType := true
				for _, p := range ciRequest.CiProjectDetails {
					// SOURCE_TYPE_WEBHOOK is not yet supported for pre-ci-stages. so handling here to get rid of fatal
					if p.SourceType != SOURCE_TYPE_BRANCH_FIXED && p.SourceType != SOURCE_TYPE_WEBHOOK {
						log.Println(util.DEVTRON, "skipping invalid source type")
						isValidSourceType = false
						break
					}
				}
				if isValidSourceType {
					if !isValidBranch(ciRequest, a) {
						log.Println(util.DEVTRON, "skipping current AppliesTo")
						continue
					}
					tasks = append(tasks, p.AfterTasks...)
					filteredOut = true
				}
			} else {
				log.Println(util.DEVTRON, "unknown triggerType ", triggerType)
			}
		}
	}
	return tasks, nil
}

func isValidBranch(ciRequest *CommonWorkflowRequest, a AppliesTo) bool {
	branches := a.Value
	branchesMap := make(map[string]bool)
	for _, b := range branches {
		branchesMap[b] = true
	}
	isValidBranch := true
	for _, prj := range ciRequest.CiProjectDetails {
		if _, ok := branchesMap[prj.SourceValue]; !ok {
			log.Println(util.DEVTRON, "invalid branch")
			isValidBranch = false
			break
		}
	}
	return isValidBranch
}

func isValidTag(ciRequest *CommonWorkflowRequest, a AppliesTo) bool {
	tagsRegex := a.Value
	for _, prj := range ciRequest.CiProjectDetails {
		for _, t := range tagsRegex {
			match, _ := regexp.MatchString(t, prj.GitTag)
			if match {
				return true
			}
		}
	}
	return false
}

func GetTaskYaml(yamlLocation string) (*TaskYaml, error) {
	filename := filepath.Join(yamlLocation, "devtron-ci.yaml")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		log.Println("file not found", filename)
		return nil, nil
	}

	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if yamlFile == nil || string(yamlFile) == "" {
		log.Println("file not found", filename)
		return nil, nil
	}

	taskYaml, err := ToTaskYaml(yamlFile)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	log.Println("yaml version: ", taskYaml.Version)
	return taskYaml, nil
}

func ToTaskYaml(yamlFile []byte) (*TaskYaml, error) {
	taskYaml := &TaskYaml{}
	err := yaml.Unmarshal(yamlFile, taskYaml)
	return taskYaml, err
}
