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

package stage

import (
	"context"
	"errors"
	"fmt"
	"github.com/devtron-labs/ci-runner/helper/adaptor"
	"github.com/devtron-labs/common-lib/utils/workFlow"
	"log"
	"os"

	"github.com/devtron-labs/ci-runner/executor"
	cictx "github.com/devtron-labs/ci-runner/executor/context"
	util2 "github.com/devtron-labs/ci-runner/executor/util"
	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
)

type CdStage struct {
	gitManager           helper.GitManager
	dockerHelper         helper.DockerHelper
	stageExecutorManager executor.StageExecutor
}

func NewCdStage(gitManager helper.GitManager, dockerHelper helper.DockerHelper, stageExecutor executor.StageExecutor) *CdStage {
	return &CdStage{
		gitManager:           gitManager,
		dockerHelper:         dockerHelper,
		stageExecutorManager: stageExecutor,
	}
}

func deferCDEvent(cdRequest *helper.CommonWorkflowRequest, artifactUploaded bool, err error) (exitCode int) {
	log.Println(util.DEVTRON, "defer CD stage data.", "err: ", err, "artifactUploaded: ", artifactUploaded)
	if err != nil {
		exitCode = workFlow.DefaultErrorCode
		var stageError *helper.CdStageError
		if errors.As(err, &stageError) {
			// update artifact uploaded status
			if !stageError.IsArtifactUploaded() {
				stageError = stageError.WithArtifactUploaded(artifactUploaded)
			}
		} else {
			stageError = helper.NewCdStageError(fmt.Errorf(workFlow.CdStageFailed.String(), cdRequest.GetCdStageType(), err)).
				WithArtifactUploaded(artifactUploaded)
		}
		// send ci failure event, for ci failure notification
		sendCDFailureEvent(cdRequest, stageError)
		// populate stage error
		util.PopulateStageError(stageError.ErrorMessage())
	}
	return exitCode
}

func (impl *CdStage) HandleCDEvent(ciCdRequest *helper.CiCdTriggerEvent, exitCode *int) {
	var artifactUploaded bool
	var err error
	var allPluginArtifacts *helper.PluginArtifacts
	defer func() {
		*exitCode = deferCDEvent(ciCdRequest.CommonWorkflowRequest, artifactUploaded, err)
	}()
	allPluginArtifacts, err = impl.runCDStages(ciCdRequest)
	if err != nil {
		log.Println("cd stage error: ", err)
		// not returning error as we want to upload artifacts
	}
	var artifactUploadErr error
	artifactUploaded, artifactUploadErr = collectAndUploadCDArtifacts(ciCdRequest.CommonWorkflowRequest)
	if artifactUploadErr != nil {
		log.Println("error in artifact upload: ", artifactUploadErr)
		// if artifact upload fails, treat it as exit status code 1 and set err to artifact upload error
		if err == nil {
			err = artifactUploadErr
		}
	}
	// IsVirtualExecution run flag indicates that cd stage is running in virtual mode.
	// specifically for isolated environment type, for IsVirtualExecution we don't send success event.
	// but failure event is sent in case of error.
	if err == nil && !ciCdRequest.CommonWorkflowRequest.IsVirtualExecution {
		log.Println(util.DEVTRON, " event")
		event := adaptor.NewCdCompleteEvent(ciCdRequest.CommonWorkflowRequest).
			WithPluginArtifacts(allPluginArtifacts).
			WithIsArtifactUploaded(artifactUploaded)
		err = helper.SendCDEvent(ciCdRequest.CommonWorkflowRequest, event)
		if err != nil {
			log.Println(err)
		}
		log.Println(util.DEVTRON, " /event")
	}
	return
}

func collectAndUploadCDArtifacts(cdRequest *helper.CommonWorkflowRequest) (artifactUploaded bool, err error) {
	cloudHelperBaseConfig := cdRequest.GetCloudHelperBaseConfig(util.BlobStorageObjectTypeArtifact)
	if cdRequest.PrePostDeploySteps != nil && len(cdRequest.PrePostDeploySteps) > 0 {
		return helper.ZipAndUpload(cloudHelperBaseConfig, cdRequest.CiArtifactFileName)
	}

	// to support stage YAML outputs
	artifactFiles := make(map[string]string)
	var allTasks []*helper.Task
	if cdRequest.TaskYaml != nil {
		for _, pc := range cdRequest.TaskYaml.CdPipelineConfig {
			for _, t := range append(pc.BeforeTasks, pc.AfterTasks...) {
				allTasks = append(allTasks, t)
			}
		}
	}
	for _, task := range allTasks {
		if task.RunStatus {
			if _, err := os.Stat(task.OutputLocation); os.IsNotExist(err) { // Ignore if no file/folder
				log.Println(util.DEVTRON, "artifact not found ", err)
				continue
			}
			artifactFiles[task.Name] = task.OutputLocation
		}
	}
	log.Println(util.DEVTRON, " artifacts", artifactFiles)
	return helper.UploadArtifact(cloudHelperBaseConfig, artifactFiles, cdRequest.CiArtifactFileName)
}

func (impl *CdStage) runCDStages(ciCdRequest *helper.CiCdTriggerEvent) (*helper.PluginArtifacts, error) {
	err := os.Chdir("/")
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(util.WORKINGDIR); os.IsNotExist(err) {
		_ = os.Mkdir(util.WORKINGDIR, os.ModeDir)
	}
	err = os.Chdir(util.WORKINGDIR)
	if err != nil {
		return nil, err
	}
	// git handling
	// we are skipping clone and checkout in case of ci job type poll cr images plugin does not require it.(ci-job)
	skipCheckout := ciCdRequest.CommonWorkflowRequest.CiPipelineType == helper.CI_JOB
	if !skipCheckout {
		log.Println(util.DEVTRON, " git")
		err = impl.gitManager.CloneAndCheckout(ciCdRequest.CommonWorkflowRequest.CiProjectDetails)
		if err != nil {
			log.Println(util.DEVTRON, "clone err: ", err)
			return nil, err
		}
	}
	log.Println(util.DEVTRON, " /git")
	// Start docker daemon
	log.Println(util.DEVTRON, " docker-start")
	impl.dockerHelper.StartDockerDaemon(ciCdRequest.CommonWorkflowRequest)
	ciContext := cictx.BuildCiContext(context.Background(), ciCdRequest.CommonWorkflowRequest.EnableSecretMasking)
	err = impl.dockerHelper.DockerLogin(ciContext, &helper.DockerCredentials{
		DockerUsername:     ciCdRequest.CommonWorkflowRequest.DockerUsername,
		DockerPassword:     ciCdRequest.CommonWorkflowRequest.DockerPassword,
		AwsRegion:          ciCdRequest.CommonWorkflowRequest.AwsRegion,
		AccessKey:          ciCdRequest.CommonWorkflowRequest.AccessKey,
		SecretKey:          ciCdRequest.CommonWorkflowRequest.SecretKey,
		DockerRegistryURL:  ciCdRequest.CommonWorkflowRequest.IntermediateDockerRegistryUrl,
		DockerRegistryType: ciCdRequest.CommonWorkflowRequest.DockerRegistryType,
	})
	if err != nil {
		return nil, err
	}

	scriptEnvs, err := util2.GetGlobalEnvVariables(ciCdRequest)

	allPluginArtifacts := helper.NewPluginArtifact()
	if len(ciCdRequest.CommonWorkflowRequest.PrePostDeploySteps) > 0 {
		refStageMap := make(map[int][]*helper.StepObject)
		for _, ref := range ciCdRequest.CommonWorkflowRequest.RefPlugins {
			refStageMap[ref.Id] = ref.Steps
		}
		scriptEnvs["DEST"] = ciCdRequest.CommonWorkflowRequest.CiArtifactDTO.Image
		scriptEnvs["DIGEST"] = ciCdRequest.CommonWorkflowRequest.CiArtifactDTO.ImageDigest
		var stage = helper.StepType(ciCdRequest.CommonWorkflowRequest.StageType)
		pluginArtifacts, _, step, err := impl.stageExecutorManager.RunCiCdSteps(stage, ciCdRequest.CommonWorkflowRequest, ciCdRequest.CommonWorkflowRequest.PrePostDeploySteps, refStageMap, scriptEnvs, nil)
		if err != nil {
			return allPluginArtifacts, helper.NewCdStageError(err).
				WithFailureMessage(fmt.Sprintf(workFlow.CdStageTaskFailed.String(), ciCdRequest.CommonWorkflowRequest.GetCdStageType(), step.Name)).
				WithArtifactUploaded(false)
		}
		allPluginArtifacts.MergePluginArtifact(pluginArtifacts)
	} else {

		// Get devtron-cd yaml
		taskYaml, err := helper.ToTaskYaml([]byte(ciCdRequest.CommonWorkflowRequest.StageYaml))
		if err != nil {
			log.Println(err)
			return allPluginArtifacts, err
		}
		ciCdRequest.CommonWorkflowRequest.TaskYaml = taskYaml

		// run post artifact processing
		log.Println(util.DEVTRON, " stage yaml", taskYaml)
		var tasks []*helper.Task
		for _, t := range taskYaml.CdPipelineConfig {
			tasks = append(tasks, t.BeforeTasks...)
			tasks = append(tasks, t.AfterTasks...)
		}

		err = impl.stageExecutorManager.RunCdStageTasks(ciContext, tasks, scriptEnvs, ciCdRequest.CommonWorkflowRequest.GetCdStageType())
		if err != nil {
			return allPluginArtifacts, err
		}
	}

	err = impl.dockerHelper.StopDocker(ciContext)
	if err != nil {
		log.Println("error while stopping docker", err)
		return allPluginArtifacts, err
	}
	return allPluginArtifacts, nil
}
