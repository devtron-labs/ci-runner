package main

import (
	"errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type TaskYaml struct {
	Version      string           `yaml:"version"`
	PipelineConf []PipelineConfig `yaml:"pipelineConf"`
}

type PipelineConfig struct {
	AppliesTo         []AppliesTo `yaml:"appliesTo"`
	BeforeDockerBuild []*Task       `yaml:"beforeDockerBuildStages"`
	AfterDockerBuild  []*Task       `yaml:"afterDockerBuildStages"`
}

type AppliesTo struct {
	Type  string   `yaml:"type"`
	Value []string `yaml:"value"`
}

const BRANCH_FIXED = "BRANCH_FIXED"
const TAG_PATTERN = "TAG_PATTERN"

func GetBeforeDockerBuildTasks(ciRequest *CiRequest, taskYaml *TaskYaml) ([]*Task, error) {
	if taskYaml == nil {
		log.Println(devtron, "no tasks, devtron-ci yaml missing")
		return nil, nil
	}

	if taskYaml.Version != "0.0.1" {
		log.Println("invalid version for devtron-ci.yaml")
		return nil, errors.New("invalid version for devtron-ci.yaml")
	}

	pipelineConfig := taskYaml.PipelineConf
	log.Println(devtron, "pipelineConf length: ", len(pipelineConfig))

	var tasks []*Task
	for _, p := range pipelineConfig {
		for _, a := range p.AppliesTo {
			triggerType := a.Type
			switch triggerType {
			case BRANCH_FIXED:
				branches := a.Value
				branchesMap := make(map[string]bool)
				for _, b := range branches {
					branchesMap[b] = true
				}
				if !isValidBranch(ciRequest, a) {
					log.Println(devtron, "skipping current AppliesTo")
					continue
				}
				tasks = append(tasks, p.BeforeDockerBuild...)
			case TAG_PATTERN:
				// TODO:
			}
		}
	}
	return tasks, nil
}

func GetAfterDockerBuildTasks(ciRequest *CiRequest, taskYaml *TaskYaml) ([]*Task, error) {
	if taskYaml == nil {
		log.Println(devtron, "no tasks, devtron-ci yaml missing")
		return nil, nil
	}

	if taskYaml.Version != "0.0.1" {
		log.Println("invalid version for devtron-ci.yaml")
		return nil, errors.New("invalid version for devtron-ci.yaml")
	}

	pipelineConfig := taskYaml.PipelineConf
	log.Println(devtron, "pipelineConf length: ", len(pipelineConfig))

	var tasks []*Task
	for _, p := range pipelineConfig {
		for _, a := range p.AppliesTo {
			triggerType := a.Type
			switch triggerType {
			case BRANCH_FIXED:
				branches := a.Value
				branchesMap := make(map[string]bool)
				for _, b := range branches {
					branchesMap[b] = true
				}
				if !isValidBranch(ciRequest, a) {
					log.Println(devtron, "skipping current AppliesTo")
					continue
				}
				tasks = append(tasks, p.AfterDockerBuild...)
			case TAG_PATTERN:
				// TODO:
			}
		}
	}
	return tasks, nil
}

func isValidBranch(ciRequest *CiRequest, a AppliesTo) bool {
	branches := a.Value
	branchesMap := make(map[string]bool)
	for _, b := range branches {
		branchesMap[b] = true
	}
	isValidBranch := true
	for _, prj := range ciRequest.CiProjectDetails {
		if _, ok := branchesMap[prj.Branch]; !ok {
			log.Println(devtron, "invalid branch")
			isValidBranch = false
			break
		}
	}
	return isValidBranch
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

	taskYaml := &TaskYaml{}
	err = yaml.Unmarshal(yamlFile, taskYaml)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	log.Println("yaml version: ", taskYaml.Version)
	return taskYaml, nil
}
