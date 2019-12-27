package main

import (
	"errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

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
	BeforeTasks []*Task     `yaml:"beforeStages"`
	AfterTasks  []*Task     `yaml:"afterStages"`
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
	filteredOut := false
	for _, p := range pipelineConfig {
		if filteredOut {
			break
		}
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
				tasks = append(tasks, p.BeforeTasks...)
				filteredOut = true
			case TAG_PATTERN:
				isValidSourceType := true
				for _, p := range ciRequest.CiProjectDetails {
					if p.SourceType != SOURCE_TYPE_TAG_REGEX {
						log.Println(devtron, "skipping invalid source type")
						isValidSourceType = false
						break
					}
				}
				if isValidSourceType {
					if !isValidTag(ciRequest, a) {
						log.Println(devtron, "skipping current AppliesTo")
						continue
					}
					tasks = append(tasks, p.BeforeTasks...)
					filteredOut = true
				}
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

	if taskYaml.Version != "0.0.1" { // TODO: Get version from ciRequest based on ci_pipeline
		log.Println("invalid version for devtron-ci.yaml")
		return nil, errors.New("invalid version for devtron-ci.yaml")
	}

	pipelineConfig := taskYaml.PipelineConf
	log.Println(devtron, "pipelineConf length: ", len(pipelineConfig))

	var tasks []*Task
	filteredOut := false
	for _, p := range pipelineConfig {
		if filteredOut {
			break
		}
		for _, a := range p.AppliesTo {
			triggerType := a.Type
			switch triggerType {
			case BRANCH_FIXED:
				isValidSourceType := true
				for _, p := range ciRequest.CiProjectDetails {
					if p.SourceType != SOURCE_TYPE_BRANCH_FIXED {
						log.Println(devtron, "skipping invalid source type")
						isValidSourceType = false
						break
					}
				}
				if isValidSourceType {
					if !isValidBranch(ciRequest, a) {
						log.Println(devtron, "skipping current AppliesTo")
						continue
					}
					tasks = append(tasks, p.AfterTasks...)
					filteredOut = true
				}
			case TAG_PATTERN:
				isValidSourceType := true
				for _, p := range ciRequest.CiProjectDetails {
					if p.SourceType != SOURCE_TYPE_TAG_REGEX {
						log.Println(devtron, "skipping invalid source type")
						isValidSourceType = false
						break
					}
					if len(ciRequest.CiProjectDetails) > 1 {
						isValidSourceType = false
					}
				}
				if isValidSourceType {
					if !isValidTag(ciRequest, a) {
						log.Println(devtron, "skipping current AppliesTo")
						continue
					}
					tasks = append(tasks, p.AfterTasks...)
					filteredOut = true
				}
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
		if _, ok := branchesMap[prj.SourceValue]; !ok {
			log.Println(devtron, "invalid branch")
			isValidBranch = false
			break
		}
	}
	return isValidBranch
}

func isValidTag(ciRequest *CiRequest, a AppliesTo) bool {
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