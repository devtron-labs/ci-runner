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
	"path/filepath"

	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
	"github.com/otiai10/copy"
)

type StepType string

const (
	STEP_TYPE_PRE        StepType = "PRE"
	STEP_TYPE_POST       StepType = "POST"
	STEP_TYPE_REF_PLUGIN StepType = "REF_PLUGIN"
)

func RunCiSteps(stepType StepType, steps []*helper.StepObject, refStageMap map[int][]*helper.StepObject, globalEnvironmentVariables map[string]string, preeCiStageVariable map[int]map[string]*helper.VariableObject) (outVars map[int]map[string]*helper.VariableObject, err error) {
	/*if stageType == STEP_TYPE_POST {
		postCiStageVariable = make(map[int]map[string]*VariableObject) // [stepId]name[]value
	}*/
	stageVariable := make(map[int]map[string]*helper.VariableObject)
	for i, ciStep := range steps {
		var vars []*helper.VariableObject
		if stepType == STEP_TYPE_REF_PLUGIN {
			vars, err = deduceVariables(ciStep.InputVars, globalEnvironmentVariables, nil, nil, stageVariable)
		} else {
			log.Printf("running step : %s\n", ciStep.Name)
			if stepType == STEP_TYPE_PRE {
				vars, err = deduceVariables(ciStep.InputVars, globalEnvironmentVariables, stageVariable, nil, nil)
			} else if stepType == STEP_TYPE_POST {
				vars, err = deduceVariables(ciStep.InputVars, globalEnvironmentVariables, preeCiStageVariable, stageVariable, nil)
			}
		}
		if err != nil {
			return nil, err
		}
		ciStep.InputVars = vars
		scriptEnvs := make(map[string]string)
		for _, v := range ciStep.InputVars {
			scriptEnvs[v.Name] = v.Value
		}
		if len(ciStep.TriggerSkipConditions) > 0 {
			shouldTrigger, err := helper.ShouldTriggerStage(ciStep.TriggerSkipConditions, ciStep.InputVars)
			if err != nil {
				log.Println(err)
				return nil, err
			}
			if !shouldTrigger {
				log.Printf("skipping %s as per pass Condition\n", ciStep.Name)
				continue
			}
		}

		var outVars []string
		for _, outVar := range ciStep.OutputVars {
			outVars = append(outVars, outVar.Name)
		}
		//cleaning the directory
		err = os.RemoveAll(util.Output_path)
		if err != nil {
			log.Println(util.DEVTRON, err)
			return nil, err
		}
		err = os.MkdirAll(util.Output_path, os.ModePerm|os.ModeDir)
		if err != nil {
			log.Println(util.DEVTRON, err)
			return nil, err
		}

		var stepOutputVarsFinal map[string]string
		//---------------------------------------------------------------------------------------------------
		if ciStep.StepType == helper.STEP_TYPE_INLINE {
			if ciStep.ExecutorType == helper.SHELL {
				stageOutputVars, err := RunScripts(util.Output_path, fmt.Sprintf("stage-%d", i), ciStep.Script, scriptEnvs, outVars)
				if err != nil {
					return nil, err
				}
				stepOutputVarsFinal = stageOutputVars
				if len(ciStep.ArtifactPaths) > 0 {
					for _, path := range ciStep.ArtifactPaths {
						err = copy.Copy(path, filepath.Join(util.TmpArtifactLocation, ciStep.Name, path))
					}
				}
			} else if ciStep.ExecutorType == helper.CONTAINER_IMAGE {
				executionConf := &executionConf{
					Script:            ciStep.Script,
					EnvInputVars:      scriptEnvs,
					ExposedPorts:      ciStep.ExposedPorts,
					OutputVars:        outVars,
					DockerImage:       ciStep.DockerImage,
					command:           ciStep.Command,
					args:              ciStep.Args,
					CustomScriptMount: ciStep.CustomScriptMount,
					SourceCodeMount:   ciStep.SourceCodeMount,
					ExtraVolumeMounts: ciStep.ExtraVolumeMounts,

					scriptFileName: fmt.Sprintf("stage-%d", i),
					workDirectory:  util.Output_path,
				}
				if executionConf.SourceCodeMount != nil {
					executionConf.SourceCodeMount.SrcPath = util.WORKINGDIR
				}
				stageOutputVars, err := RunScriptsInDocker(executionConf)
				if err != nil {
					return nil, err
				}
				stepOutputVarsFinal = stageOutputVars
			}
		} else if ciStep.StepType == helper.STEP_TYPE_REF_PLUGIN {
			steps := refStageMap[ciStep.RefPluginId]
			stepIndexVarNameValueMap := make(map[int]map[string]string)
			for _, inVar := range ciStep.InputVars {
				if varMap, ok := stepIndexVarNameValueMap[inVar.VariableStepIndexInPlugin]; ok {
					varMap[inVar.Name] = inVar.Value
					stepIndexVarNameValueMap[inVar.VariableStepIndexInPlugin] = varMap
				} else {
					varMap := map[string]string{inVar.Name: inVar.Value}
					stepIndexVarNameValueMap[inVar.VariableStepIndexInPlugin] = varMap
				}
			}
			for _, step := range steps {
				if varMap, ok := stepIndexVarNameValueMap[step.Index]; ok {
					for _, inVar := range step.InputVars {
						if value, ok := varMap[inVar.Name]; ok {
							inVar.Value = value
						}
					}
				}
			}
			opt, err := RunCiSteps(STEP_TYPE_REF_PLUGIN, steps, refStageMap, globalEnvironmentVariables, nil)
			if err != nil {
				fmt.Println(err)
				return nil, err
			}
			for _, outputVar := range ciStep.OutputVars {
				if varObj, ok := opt[outputVar.VariableStepIndexInPlugin]; ok {
					if v, ok1 := varObj[outputVar.Name]; ok1 {
						stepOutputVarsFinal[v.Name] = v.Value
					}
				}
			}
			fmt.Println(opt)
			//stepOutputVarsFinal=opt
			//manipulate pre and post variables
			// artifact path
			//
		} else {
			return nil, fmt.Errorf("step Type :%s not supported", ciStep.StepType)
		}
		//---------------------------------------------------------------------------------------------------
		finalOutVars, err := populateOutVars(stepOutputVarsFinal, ciStep.OutputVars)
		if err != nil {
			return nil, err
		}
		ciStep.OutputVars = finalOutVars
		if len(ciStep.SuccessFailureConditions) > 0 {
			success, err := helper.StageIsSuccess(ciStep.SuccessFailureConditions, finalOutVars)
			if err != nil {
				return nil, err
			}
			if !success {
				return nil, fmt.Errorf("stage not success")
			}
		}
		finalOutVarMap := make(map[string]*helper.VariableObject)
		for _, out := range ciStep.OutputVars {
			finalOutVarMap[out.Name] = out
		}
		stageVariable[ciStep.Index] = finalOutVarMap
	}
	return stageVariable, nil
}

func populateOutVars(outData map[string]string, desired []*helper.VariableObject) ([]*helper.VariableObject, error) {
	var finalOutVars []*helper.VariableObject
	for _, d := range desired {
		value := outData[d.Name]
		if len(value) == 0 {
			log.Printf("%s not present\n", d.Name)
			continue
		}
		typedVal, err := helper.TypeConverter(value, d.Format)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		d.Value = value
		d.TypedValue = typedVal
		finalOutVars = append(finalOutVars, d)
	}
	return finalOutVars, nil
}

func deduceVariables(desiredVars []*helper.VariableObject, globalVars map[string]string, preeCiStageVariable map[int]map[string]*helper.VariableObject, postCiStageVariables map[int]map[string]*helper.VariableObject, refPluginStageVariables map[int]map[string]*helper.VariableObject) ([]*helper.VariableObject, error) {
	var inputVars []*helper.VariableObject
	for _, desired := range desiredVars {
		switch desired.VariableType {
		case helper.VALUE:
			inputVars = append(inputVars, desired)
		case helper.REF_PRE_CI:
			if v, found := preeCiStageVariable[desired.ReferenceVariableStepIndex]; found {
				if d, foundD := v[desired.ReferenceVariableName]; foundD {
					desired.Value = d.Value
					err := desired.TypeCheck()
					if err != nil {
						return nil, err
					}
					inputVars = append(inputVars, desired)
				} else {
					return nil, fmt.Errorf("RUNTIME_ERROR_%s_not_found ", desired.Name)
				}
			} else {
				return nil, fmt.Errorf("RUNTIME_ERROR_%s_not_found ", desired.Name)
			}
		case helper.REF_POST_CI:
			if v, found := postCiStageVariables[desired.ReferenceVariableStepIndex]; found {
				if d, foundD := v[desired.ReferenceVariableName]; foundD {
					desired.Value = d.Value
					err := desired.TypeCheck()
					if err != nil {
						return nil, err
					}
					inputVars = append(inputVars, desired)
				} else {
					return nil, fmt.Errorf("RUNTIME_ERROR_%s_not_found ", desired.Name)
				}
			} else {
				return nil, fmt.Errorf("RUNTIME_ERROR_%s_not_found ", desired.Name)
			}
		case helper.REF_GLOBAL:
			desired.Value = globalVars[desired.ReferenceVariableName]
			err := desired.TypeCheck()
			if err != nil {
				return nil, err
			}
			inputVars = append(inputVars, desired)
		case helper.REF_PLUGIN:
			if v, found := refPluginStageVariables[desired.ReferenceVariableStepIndex]; found {
				if d, foundD := v[desired.ReferenceVariableName]; foundD {
					desired.Value = d.Value
					err := desired.TypeCheck()
					if err != nil {
						return nil, err
					}
					inputVars = append(inputVars, desired)
				} else {
					return nil, fmt.Errorf("RUNTIME_ERROR_%s_not_found ", desired.Name)
				}
			} else {
				return nil, fmt.Errorf("RUNTIME_ERROR_%s_not_found ", desired.Name)
			}
		}
	}
	return inputVars, nil

}

func RunPreDockerBuildTasks(ciRequest *helper.CiRequest, scriptEnvs map[string]string, taskYaml *helper.TaskYaml) error {
	//before task
	beforeTaskMap := make(map[string]*helper.Task)
	for i, task := range ciRequest.BeforeDockerBuild {
		task.RunStatus = true
		beforeTaskMap[task.Name] = task
		log.Println(util.DEVTRON, "pre", task)
		//log running cmd
		util.LogStage(task.Name)
		_, err := RunScripts(util.Output_path, fmt.Sprintf("before-%d", i), task.Script, scriptEnvs, nil)
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
		_, err = RunScripts(util.Output_path, fmt.Sprintf("before-yaml-%d", i), task.Script, scriptEnvs, nil)
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
		_, err := RunScripts(util.Output_path, fmt.Sprintf("after-%d", i), task.Script, scriptEnvs, nil)
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
		_, err = RunScripts(util.Output_path, fmt.Sprintf("after-yaml-%d", i), task.Script, scriptEnvs, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func RunCdStageTasks(tasks []*helper.Task, scriptEnvs map[string]string) error {
	log.Println(util.DEVTRON, " cd-stage-processing")
	//cleaning the directory
	err := os.RemoveAll(util.Output_path)
	if err != nil {
		log.Println(util.DEVTRON, err)
		return err
	}
	err = os.MkdirAll(util.Output_path, os.ModePerm|os.ModeDir)
	if err != nil {
		log.Println(util.DEVTRON, err)
		return err
	}
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
		_, err := RunScripts(util.Output_path, fmt.Sprintf("stage-%d", i), task.Script, scriptEnvs, nil)
		if err != nil {
			return err
		}
	}
	return nil
}
