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

func RunCiCdSteps(stepType StepType, steps []*helper.StepObject, refStageMap map[int][]*helper.StepObject, globalEnvironmentVariables map[string]string, preeCiStageVariable map[int]map[string]*helper.VariableObject) (outVars map[int]map[string]*helper.VariableObject, failedStep *helper.StepObject, err error) {
	/*if stageType == STEP_TYPE_POST {
		postCiStageVariable = make(map[int]map[string]*VariableObject) // [stepId]name[]value
	}*/
	log.Println("steps = ", steps)
	log.Println("globalEnvironmentVariables = ", globalEnvironmentVariables)
	log.Println("preeCiStageVariable = ", preeCiStageVariable)
	stageVariable := make(map[int]map[string]*helper.VariableObject)
	for i, step := range steps {
		var vars []*helper.VariableObject
		if stepType == STEP_TYPE_REF_PLUGIN {
			vars, err = deduceVariables(step.InputVars, globalEnvironmentVariables, nil, nil, stageVariable)
		} else {
			log.Printf("running step : %s\n", step.Name)
			if stepType == STEP_TYPE_PRE {
				vars, err = deduceVariables(step.InputVars, globalEnvironmentVariables, stageVariable, nil, nil)
			} else if stepType == STEP_TYPE_POST {
				vars, err = deduceVariables(step.InputVars, globalEnvironmentVariables, preeCiStageVariable, stageVariable, nil)
			}
		}
		if err != nil {
			return nil, step, err
		}
		step.InputVars = vars

		//variables with empty value
		var emptyVariableList []string
		scriptEnvs := make(map[string]string)
		for _, v := range step.InputVars {
			scriptEnvs[v.Name] = v.Value
			if len(v.Value) == 0 {
				emptyVariableList = append(emptyVariableList, v.Name)
			}
		}
		if stepType == STEP_TYPE_PRE || stepType == STEP_TYPE_POST {
			log.Println(fmt.Sprintf("variables with empty value : %v", emptyVariableList))
		}
		if len(step.TriggerSkipConditions) > 0 {
			shouldTrigger, err := helper.ShouldTriggerStage(step.TriggerSkipConditions, step.InputVars)
			if err != nil {
				log.Println(err)
				return nil, step, err
			}
			if !shouldTrigger {
				log.Printf("skipping %s as per pass Condition\n", step.Name)
				continue
			}
		}

		var outVars []string
		for _, outVar := range step.OutputVars {
			outVars = append(outVars, outVar.Name)
		}
		//cleaning the directory
		err = os.RemoveAll(util.Output_path)
		if err != nil {
			log.Println(util.DEVTRON, err)
			return nil, step, err
		}
		err = os.MkdirAll(util.Output_path, os.ModePerm|os.ModeDir)
		if err != nil {
			log.Println(util.DEVTRON, err)
			return nil, step, err
		}

		stepOutputVarsFinal := make(map[string]string)
		//---------------------------------------------------------------------------------------------------
		if step.StepType == helper.STEP_TYPE_INLINE {
			if step.ExecutorType == helper.SHELL {
				log.Println("scriptEnvs = ", scriptEnvs)
				stageOutputVars, err := RunScripts(util.Output_path, fmt.Sprintf("stage-%d", i), step.Script, scriptEnvs, outVars)
				if err != nil {
					return nil, step, err
				}
				stepOutputVarsFinal = stageOutputVars
				if len(step.ArtifactPaths) > 0 {
					for _, path := range step.ArtifactPaths {
						err = copy.Copy(path, filepath.Join(util.TmpArtifactLocation, step.Name, path))
						if err != nil {
							if _, ok := err.(*os.PathError); ok {
								log.Println(util.DEVTRON, "dir not exists", path)
								continue
							} else {
								return nil, step, err
							}
						}
					}
				}
			} else if step.ExecutorType == helper.CONTAINER_IMAGE {
				var outputDirMount []*helper.MountPath
				stepArtifact := filepath.Join(util.Output_path, "opt")

				for _, artifact := range step.ArtifactPaths {
					hostPath := filepath.Join(stepArtifact, artifact)
					err = os.MkdirAll(hostPath, os.ModePerm|os.ModeDir)
					if err != nil {
						log.Println(util.DEVTRON, err)
						return nil, step, err
					}
					path := &helper.MountPath{DstPath: artifact, SrcPath: filepath.Join(stepArtifact, artifact)}
					outputDirMount = append(outputDirMount, path)
				}
				executionConf := &executionConf{
					Script:            step.Script,
					EnvInputVars:      scriptEnvs,
					ExposedPorts:      step.ExposedPorts,
					OutputVars:        outVars,
					DockerImage:       step.DockerImage,
					command:           step.Command,
					args:              step.Args,
					CustomScriptMount: step.CustomScriptMount,
					SourceCodeMount:   step.SourceCodeMount,
					ExtraVolumeMounts: step.ExtraVolumeMounts,
					scriptFileName:    fmt.Sprintf("stage-%d", i),
					workDirectory:     util.Output_path,
					OutputDirMount:    outputDirMount,
				}
				if executionConf.SourceCodeMount != nil {
					executionConf.SourceCodeMount.SrcPath = util.WORKINGDIR
				}
				stageOutputVars, err := RunScriptsInDocker(executionConf)
				if err != nil {
					return nil, step, err
				}
				stepOutputVarsFinal = stageOutputVars
				if _, err := os.Stat(stepArtifact); os.IsNotExist(err) {
					// Ignore if no file/folder
					log.Println(util.DEVTRON, "artifact not found ", err)
				} else {
					err = copy.Copy(stepArtifact, filepath.Join(util.TmpArtifactLocation, step.Name))
					if err != nil {
						return nil, step, err
					}
				}
			}
		} else if step.StepType == helper.STEP_TYPE_REF_PLUGIN {
			steps := refStageMap[step.RefPluginId]
			stepIndexVarNameValueMap := make(map[int]map[string]string)
			for _, inVar := range step.InputVars {
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
			opt, _, err := RunCiCdSteps(STEP_TYPE_REF_PLUGIN, steps, refStageMap, globalEnvironmentVariables, nil)
			if err != nil {
				fmt.Println(err)
				return nil, step, err
			}
			for _, outputVar := range step.OutputVars {
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
			return nil, step, fmt.Errorf("step Type :%s not supported", step.StepType)
		}
		//---------------------------------------------------------------------------------------------------
		finalOutVars, err := populateOutVars(stepOutputVarsFinal, step.OutputVars)
		if err != nil {
			return nil, step, err
		}
		step.OutputVars = finalOutVars
		if len(step.SuccessFailureConditions) > 0 {
			success, err := helper.StageIsSuccess(step.SuccessFailureConditions, finalOutVars)
			if err != nil {
				return nil, step, err
			}
			if !success {
				return nil, step, fmt.Errorf("stage not successful because of condition failure")
			}
		}
		finalOutVarMap := make(map[string]*helper.VariableObject)
		for _, out := range step.OutputVars {
			finalOutVarMap[out.Name] = out
		}
		stageVariable[step.Index] = finalOutVarMap
	}
	return stageVariable, nil, nil
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
		log.Println("desired = ", desired)
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
		err := RunScriptsV1(util.Output_path, fmt.Sprintf("stage-%d", i), task.Script, scriptEnvs)
		if err != nil {
			return err
		}
	}
	return nil
}
