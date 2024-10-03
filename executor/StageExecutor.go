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

package executor

import (
	"context"
	"fmt"
	cictx "github.com/devtron-labs/ci-runner/executor/context"
	util2 "github.com/devtron-labs/ci-runner/executor/util"
	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
	"github.com/devtron-labs/common-lib/utils/workFlow"
	copylib "github.com/otiai10/copy"
	"log"
	"os"
	"path/filepath"
)

type StageExecutorImpl struct {
	cmdExecutor    helper.CommandExecutor
	scriptExecutor ScriptExecutor
}

type StageExecutor interface {
	RunCiCdSteps(stepType helper.StepType, ciCdRequest *helper.CommonWorkflowRequest, steps []*helper.StepObject, refStageMap map[int][]*helper.StepObject, globalEnvironmentVariables map[string]string, preCiStageVariable map[int]map[string]*helper.VariableObject) (pluginArtifacts *helper.PluginArtifacts, outVars map[int]map[string]*helper.VariableObject, failedStep *helper.StepObject, err error)
	RunCdStageTasks(ciContext cictx.CiContext, tasks []*helper.Task, scriptEnvs map[string]string, stageType helper.PipelineType) error
}

func NewStageExecutorImpl(cmdExecutor helper.CommandExecutor, scriptExecutor ScriptExecutor) *StageExecutorImpl {
	return &StageExecutorImpl{
		cmdExecutor:    cmdExecutor,
		scriptExecutor: scriptExecutor,
	}
}

func (impl *StageExecutorImpl) RunCiCdSteps(stepType helper.StepType, ciCdRequest *helper.CommonWorkflowRequest, steps []*helper.StepObject, refStageMap map[int][]*helper.StepObject, globalEnvironmentVariables map[string]string, preCiStageVariable map[int]map[string]*helper.VariableObject) (*helper.PluginArtifacts, map[int]map[string]*helper.VariableObject, *helper.StepObject, error) {
	/*if stageType == STEP_TYPE_POST {
		postCiStageVariable = make(map[int]map[string]*VariableObject) // [stepId]name[]value
	}*/

	stageVariable := make(map[int]map[string]*helper.VariableObject)
	pluginArtifactsFromFile := helper.NewPluginArtifact()
	for i, step := range steps {

		stageInfoLoggingRequired := stepType != helper.STEP_TYPE_REF_PLUGIN
		failedStep := step
		var (
			err                error
			refPluginArtifacts *helper.PluginArtifacts
		)

		executeStep := func() error {
			refPluginArtifacts, failedStep, err = impl.RunCiCdStep(stepType, *ciCdRequest, i, step, refStageMap, globalEnvironmentVariables, preCiStageVariable, stageVariable)
			if err != nil {
				return err
			}
			pluginArtifactsFromFile.MergePluginArtifact(refPluginArtifacts)
			return nil
		}

		if stageInfoLoggingRequired {
			log.Println(util.DEVTRON, "stage logging required")
			err = util.ExecuteWithStageInfoLog(step.Name, executeStep)
		} else {
			log.Println(util.DEVTRON, "stage logging not required")
			err = executeStep()
		}
		// if errored, we can return the failed step and the error
		if err != nil {
			return nil, stageVariable, failedStep, err
		}
		pluginArtifacts, err := helper.ExtractPluginArtifactsAndRemoveFile()
		if err != nil {
			log.Println("error in extracting plugin artifacts from file", "err", err)
			return nil, nil, nil, err
		}
		pluginArtifactsFromFile.MergePluginArtifact(pluginArtifacts)
	}

	return pluginArtifactsFromFile, stageVariable, nil, nil
}

func (impl *StageExecutorImpl) RunCiCdStep(stepType helper.StepType, ciCdRequest helper.CommonWorkflowRequest, index int, step *helper.StepObject,
	refStageMap map[int][]*helper.StepObject, globalEnvironmentVariables map[string]string,
	preCiStageVariable map[int]map[string]*helper.VariableObject,
	stageVariable map[int]map[string]*helper.VariableObject) (artifacts *helper.PluginArtifacts, failedStep *helper.StepObject, err error) {
	var vars []*helper.VariableObject
	if stepType == helper.STEP_TYPE_REF_PLUGIN {
		vars, err = deduceVariables(step.InputVars, globalEnvironmentVariables, nil, nil, stageVariable)
	} else {
		log.Printf("running step : %s\n", step.Name)
		if stepType == helper.STEP_TYPE_PRE {
			vars, err = deduceVariables(step.InputVars, globalEnvironmentVariables, stageVariable, nil, nil)
		} else if stepType == helper.STEP_TYPE_POST {
			vars, err = deduceVariables(step.InputVars, globalEnvironmentVariables, preCiStageVariable, stageVariable, nil)
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
	for key, value := range globalEnvironmentVariables {
		scriptEnvs[key] = value
	}
	if stepType == helper.STEP_TYPE_PRE || stepType == helper.STEP_TYPE_POST {
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
			return nil, nil, nil
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

	ciContext := cictx.BuildCiContext(context.Background(), ciCdRequest.EnableSecretMasking)

	stepOutputVarsFinal := make(map[string]string)
	var pluginArtifacts *helper.PluginArtifacts
	//---------------------------------------------------------------------------------------------------
	if step.StepType == helper.STEP_TYPE_INLINE {
		//add system env variable
		for k, v := range util2.GetSystemEnvVariables() {
			//add only when not overridden by user
			if _, ok := scriptEnvs[k]; !ok {
				scriptEnvs[k] = v
			}
		}
		if step.ExecutorType == helper.SHELL {
			stageOutputVars, err := impl.scriptExecutor.RunScripts(ciContext, util.Output_path, fmt.Sprintf("stage-%d", index), step.Script, scriptEnvs, outVars)
			if err != nil {
				return nil, step, err
			}
			stepOutputVarsFinal = stageOutputVars
			if len(step.ArtifactPaths) > 0 {
				for _, path := range step.ArtifactPaths {
					err = copylib.Copy(path, filepath.Join(util.TmpArtifactLocation, step.Name, path))
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
				scriptFileName:    fmt.Sprintf("stage-%d", index),
				workDirectory:     util.Output_path,
				OutputDirMount:    outputDirMount,
			}
			if executionConf.SourceCodeMount != nil {
				executionConf.SourceCodeMount.SrcPath = util.WORKINGDIR
			}
			stageOutputVars, err := RunScriptsInDocker(ciContext, impl, executionConf)
			if err != nil {
				return nil, step, err
			}
			stepOutputVarsFinal = stageOutputVars
			if _, err := os.Stat(stepArtifact); os.IsNotExist(err) {
				// Ignore if no file/folder
				log.Println(util.DEVTRON, "artifact not found ", err)
			} else {
				err = copylib.Copy(stepArtifact, filepath.Join(util.TmpArtifactLocation, step.Name))
				if err != nil {
					return nil, step, err
				}
			}
		}
	} else if step.StepType == string(helper.STEP_TYPE_REF_PLUGIN) {
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
		refPluginArtifacts, opt, _, err := impl.RunCiCdSteps(helper.STEP_TYPE_REF_PLUGIN, &ciCdRequest, steps, refStageMap, globalEnvironmentVariables, nil)
		if err != nil {
			fmt.Println(err)
			return nil, step, err
		}
		pluginArtifacts = refPluginArtifacts
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
	return pluginArtifacts, nil, nil
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

func (impl *StageExecutorImpl) RunCdStageTasks(ciContext cictx.CiContext, tasks []*helper.Task, scriptEnvs map[string]string, stageType helper.PipelineType) error {
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
		err := impl.scriptExecutor.RunScriptsV1(ciContext, util.Output_path, fmt.Sprintf("stage-%d", i), task.Script, scriptEnvs)
		if err != nil {
			return helper.NewCdStageError(err).
				WithFailureMessage(fmt.Sprintf(workFlow.CdStageTaskFailed.String(), stageType, task.Name)).
				WithArtifactUploaded(false)
		}
	}
	return nil
}
