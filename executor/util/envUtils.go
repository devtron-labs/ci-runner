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

package util

import (
	"encoding/json"
	"fmt"
	"github.com/caarlos0/env"
	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/pubsub"
	"github.com/devtron-labs/ci-runner/util"
	"os"
	"strconv"
	"strings"
)

func GetGlobalEnvVariables(cicdRequest *helper.CiCdTriggerEvent) (map[string]string, error) {
	envs := make(map[string]string)
	envs["WORKING_DIRECTORY"] = util.WORKINGDIR
	cfg := &pubsub.PubSubConfig{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}
	if helper.IsCIOrJobTypeEvent(cicdRequest.Type) {
		image, err := helper.BuildDockerImagePath(cicdRequest.CommonWorkflowRequest)
		if err != nil {
			return nil, err
		}

		envs["DOCKER_IMAGE_TAG"] = cicdRequest.CommonWorkflowRequest.DockerImageTag
		envs["DOCKER_REPOSITORY"] = cicdRequest.CommonWorkflowRequest.DockerRepository
		envs["DOCKER_REGISTRY_URL"] = cicdRequest.CommonWorkflowRequest.DockerRegistryURL
		envs["TRIGGER_BY_AUTHOR"] = cicdRequest.CommonWorkflowRequest.TriggerByAuthor
		envs["DOCKER_IMAGE"] = image
		envs["DOCKER_IMAGE_TAG"] = cicdRequest.CommonWorkflowRequest.DockerImageTag

		if cicdRequest.Type == util.JOBEVENT {
			envs["JOB_NAME"] = cicdRequest.CommonWorkflowRequest.AppName
		} else {
			envs["APP_NAME"] = cicdRequest.CommonWorkflowRequest.AppName
		}
		//adding GIT_MATERIAL_REQUEST in env for semgrep plugin
		CiMaterialRequestArr := ""
		if cicdRequest.CommonWorkflowRequest.CiProjectDetails != nil {
			for _, ciProjectDetail := range cicdRequest.CommonWorkflowRequest.CiProjectDetails {
				GitRepoSplit := strings.Split(ciProjectDetail.GitRepository, "/")
				GitRepoName := ""
				if len(GitRepoSplit) > 0 {
					GitRepoName = strings.Split(GitRepoSplit[len(GitRepoSplit)-1], ".")[0]
				}
				CiMaterialRequestArr = CiMaterialRequestArr +
					fmt.Sprintf("%s,%s,%s,%s|", GitRepoName, ciProjectDetail.CheckoutPath, ciProjectDetail.SourceValue, ciProjectDetail.CommitHash)
			}
		}
		envs["GIT_MATERIAL_REQUEST"] = CiMaterialRequestArr // GIT_MATERIAL_REQUEST will be of form "<repoName>/<checkoutPath>/<BranchName>/<CommitHash>"
		fmt.Println(envs["GIT_MATERIAL_REQUEST"])

		// adding envs for polling-plugin
		envs["DOCKER_REGISTRY_TYPE"] = cicdRequest.CommonWorkflowRequest.DockerRegistryType
		envs["DOCKER_USERNAME"] = cicdRequest.CommonWorkflowRequest.DockerUsername
		envs["DOCKER_PASSWORD"] = cicdRequest.CommonWorkflowRequest.DockerPassword
		envs["ACCESS_KEY"] = cicdRequest.CommonWorkflowRequest.AccessKey
		envs["SECRET_KEY"] = cicdRequest.CommonWorkflowRequest.SecretKey
		envs["AWS_REGION"] = cicdRequest.CommonWorkflowRequest.AwsRegion
		envs["LAST_FETCHED_TIME"] = cicdRequest.CommonWorkflowRequest.CiArtifactLastFetch.String()

		//adding some envs for Image scanning plugin
		envs["PIPELINE_ID"] = strconv.Itoa(cicdRequest.CommonWorkflowRequest.PipelineId)
		envs["TRIGGERED_BY"] = strconv.Itoa(cicdRequest.CommonWorkflowRequest.TriggeredBy)
		envs["DOCKER_REGISTRY_ID"] = cicdRequest.CommonWorkflowRequest.DockerRegistryId
		envs["IMAGE_SCANNER_ENDPOINT"] = cfg.ImageScannerEndpoint
		envs["IMAGE_SCAN_MAX_RETRIES"] = strconv.Itoa(cicdRequest.CommonWorkflowRequest.ImageScanMaxRetries)
		envs["IMAGE_SCAN_RETRY_DELAY"] = strconv.Itoa(cicdRequest.CommonWorkflowRequest.ImageScanRetryDelay)

		// setting extraEnvironmentVariables
		for k, v := range cicdRequest.CommonWorkflowRequest.ExtraEnvironmentVariables {
			envs[k] = v
		}
		// for skopeo plugin, list of destination images againt registry name eg: <registry_name>: [<i1>,<i2>]
		RegistryDestinationImage, _ := json.Marshal(cicdRequest.CommonWorkflowRequest.RegistryDestinationImageMap)
		RegistryCredentials, _ := json.Marshal(cicdRequest.CommonWorkflowRequest.RegistryCredentialMap)
		envs["REGISTRY_DESTINATION_IMAGE_MAP"] = string(RegistryDestinationImage)
		envs["REGISTRY_CREDENTIALS"] = string(RegistryCredentials)
	} else {
		envs["DOCKER_IMAGE"] = cicdRequest.CommonWorkflowRequest.CiArtifactDTO.Image
		envs["DOCKER_IMAGE_TAG"] = cicdRequest.CommonWorkflowRequest.DockerImageTag
		envs["DEPLOYMENT_RELEASE_ID"] = strconv.Itoa(cicdRequest.CommonWorkflowRequest.DeploymentReleaseCounter)
		envs["DEPLOYMENT_UNIQUE_ID"] = strconv.Itoa(cicdRequest.CommonWorkflowRequest.WorkflowRunnerId)
		envs["CD_TRIGGERED_BY"] = cicdRequest.CommonWorkflowRequest.DeploymentTriggeredBy
		envs["CD_TRIGGER_TIME"] = cicdRequest.CommonWorkflowRequest.DeploymentTriggerTime.String()

		// to support legacy yaml based script trigger
		envs["DEVTRON_CD_TRIGGERED_BY"] = cicdRequest.CommonWorkflowRequest.DeploymentTriggeredBy
		envs["DEVTRON_CD_TRIGGER_TIME"] = cicdRequest.CommonWorkflowRequest.DeploymentTriggerTime.String()

		//adding some envs for Image scanning plugin
		envs["TRIGGERED_BY"] = strconv.Itoa(cicdRequest.CommonWorkflowRequest.TriggeredBy)
		envs["DOCKER_REGISTRY_ID"] = cicdRequest.CommonWorkflowRequest.DockerRegistryId
		envs["IMAGE_SCANNER_ENDPOINT"] = cfg.ImageScannerEndpoint
		envs["IMAGE_SCAN_MAX_RETRIES"] = strconv.Itoa(cicdRequest.CommonWorkflowRequest.ImageScanMaxRetries)
		envs["IMAGE_SCAN_RETRY_DELAY"] = strconv.Itoa(cicdRequest.CommonWorkflowRequest.ImageScanRetryDelay)

		for k, v := range cicdRequest.CommonWorkflowRequest.ExtraEnvironmentVariables {
			envs[k] = v
		}
		// for skopeo plugin, list of destination images againt registry name eg: <registry_name>: [<i1>,<i2>]
		RegistryDestinationImage, _ := json.Marshal(cicdRequest.CommonWorkflowRequest.RegistryDestinationImageMap)
		RegistryCredentials, _ := json.Marshal(cicdRequest.CommonWorkflowRequest.RegistryCredentialMap)
		envs["REGISTRY_DESTINATION_IMAGE_MAP"] = string(RegistryDestinationImage)
		envs["REGISTRY_CREDENTIALS"] = string(RegistryCredentials)
	}
	return envs, nil
}

func GetSystemEnvVariables() map[string]string {
	envs := make(map[string]string)
	//get all environment variables
	envVars := os.Environ()
	for _, envVar := range envVars {
		subs := strings.SplitN(envVar, "=", 2)
		envs[subs[0]] = subs[1]
	}
	return envs
}
