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

	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
)

const (
	STEP_TYPE_PRE  = "PRE"
	STEP_TYPE_POST = "POST"
)

func RunCiSteps(steps []*StepObject, refPlugins []*RefPluginObject, globalEnvironmentVariables map[string]string, preeCiStageVariable map[int]map[string]*VariableObject) (outVars map[int]map[string]*VariableObject, err error) {
	/*if stageType == STEP_TYPE_POST {
		postCiStageVariable = make(map[int]map[string]*VariableObject) // [stepId]name[]value
	}*/
	stageVariable := make(map[int]map[string]*VariableObject)
	refStageMap := make(map[int][]*StepObject)
	for _, ref := range refPlugins {
		refStageMap[ref.Id] = ref.Steps
	}
	for i, preciStage := range steps {
		vars, err := deduceVariables(preciStage.InputVars, globalEnvironmentVariables, preeCiStageVariable, stageVariable)
		if err != nil {
			return nil, err
		}
		preciStage.InputVars = vars
		scriptEnvs := make(map[string]string)
		for _, v := range preciStage.InputVars {
			scriptEnvs[v.Name] = v.Value
		}
		if len(preciStage.TriggerSkipConditions) > 0 {
			shouldTrigger, err := shouldTriggerStage(preciStage.TriggerSkipConditions, preciStage.InputVars)
			if err != nil {
				log.Println(err)
				return nil, err
			}
			if !shouldTrigger {
				log.Println("skipping stage as per pree Condition")
				continue
			}
		}

		var outVars []string
		for _, outVar := range preciStage.OutputVars {
			outVars = append(outVars, outVar.Name)
		}
		//cleaning the directory
		err = os.RemoveAll(output_path)
		if err != nil {
			log.Println(devtron, err)
			return nil, err
		}
		err = os.MkdirAll(output_path, os.ModePerm|os.ModeDir)
		if err != nil {
			log.Println(devtron, err)
			return nil, err
		}

		var stageOutputVarsFinal map[string]string
		//---------------------------------------------------------------------------------------------------
		if preciStage.StepType == STEP_TYPE_INLINE {
			if preciStage.ExecutorType == SHELL {
				stageOutputVars, err := RunScripts(output_path, fmt.Sprintf("stage-%d", i), preciStage.Script, scriptEnvs, outVars)
				if err != nil {
					return nil, err
				}
				stageOutputVarsFinal = stageOutputVars
			} else {
				executionConf := &executionConf{
					Script:            preciStage.Script,
					EnvInputVars:      scriptEnvs,
					ExposedPorts:      preciStage.ExposedPorts,
					OutputVars:        outVars,
					DockerImage:       preciStage.DockerImage,
					command:           preciStage.Command,
					args:              preciStage.Args,
					CustomScriptMount: preciStage.CustomScriptMount,
					SourceCodeMount:   preciStage.SourceCodeMount,
					ExtraVolumeMounts: preciStage.ExtraVolumeMounts,

					scriptFileName: fmt.Sprintf("stage-%d", i),
					workDirectory:  output_path,
				}
				if executionConf.SourceCodeMount != nil {
					executionConf.SourceCodeMount.SrcPath = workingDir
				}
				stageOutputVars, err := RunScriptsInDocker(executionConf)
				if err != nil {
					return nil, err
				}
				stageOutputVarsFinal = stageOutputVars
			}
		} else if preciStage.StepType == STEP_TYPE_REF_PLUGIN {
			steps := refStageMap[preciStage.RefPluginId]
			//FIXME: sdcsdc
			preCiStageVariablePlugin := make(map[int]map[string]*VariableObject)
			opt, err := RunCiSteps(steps, refPlugins, globalEnvironmentVariables, preCiStageVariablePlugin)
			if err != nil {
				fmt.Println(err)
				return nil, err
			}
			fmt.Println(opt)
			//stageOutputVarsFinal=opt
			//manupulate pree and post variables
			// artifact path
			//
		} else {
			return nil, fmt.Errorf("step Type :%s not supported", preciStage.StepType)
		}
		//---------------------------------------------------------------------------------------------------
		finalOutVars, err := populateOutVars(stageOutputVarsFinal, preciStage.OutputVars)
		if err != nil {
			return nil, err
		}
		preciStage.OutputVars = finalOutVars
		if len(preciStage.SuccessFailureConditions) > 0 {
			success, err := stageIsSuccess(preciStage.SuccessFailureConditions, finalOutVars)
			if err != nil {
				return nil, err
			}
			if !success {
				return nil, fmt.Errorf("stage not success")
			}
		}
		finalOutVarMap := make(map[string]*VariableObject)
		for _, out := range preciStage.OutputVars {
			finalOutVarMap[out.Name] = out
		}
		stageVariable[preciStage.Index] = finalOutVarMap
	}
	return preeCiStageVariable, nil
}

func populateOutVars(outData map[string]string, desired []*VariableObject) ([]*VariableObject, error) {
	var finalOutVars []*VariableObject
	for _, d := range desired {
		value := outData[d.Name]
		if len(value) == 0 {
			log.Printf("%s not present\n", d.Name)
			continue
		}
		typedVal, err := typeConverter(value, d.Format)
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

func deduceVariables(desiredVars []*VariableObject, globalVars map[string]string, preeCiStageVariable map[int]map[string]*VariableObject, postCiStageVariables map[int]map[string]*VariableObject) ([]*VariableObject, error) {
	var inputVars []*VariableObject
	for _, desired := range desiredVars {
		switch desired.VariableType {
		case VALUE:
			inputVars = append(inputVars, desired)
		case REF_PRE_CI:
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
		case REF_POST_CI:
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
		case REF_GLOBAL:
			desired.Value = globalVars[desired.ReferenceVariableName]
			err := desired.TypeCheck()
			if err != nil {
				return nil, err
			}
			inputVars = append(inputVars, desired)
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
