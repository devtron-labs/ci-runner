package main

import (
	"fmt"
	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
	"os"
	"strconv"
	"strings"
)

func getGlobalEnvVariables(cicdRequest *helper.CiCdTriggerEvent) (map[string]string, error) {
	envs := make(map[string]string)
	envs["WORKING_DIRECTORY"] = util.WORKINGDIR
	if cicdRequest.Type == util.CIEVENT {
		image, err := helper.BuildDockerImagePath(cicdRequest.CommonWorkflowRequest)
		if err != nil {
			return nil, err
		}
		envs["DOCKER_IMAGE_TAG"] = cicdRequest.CommonWorkflowRequest.DockerImageTag
		envs["DOCKER_REPOSITORY"] = cicdRequest.CommonWorkflowRequest.DockerRepository
		envs["DOCKER_REGISTRY_URL"] = cicdRequest.CommonWorkflowRequest.DockerRegistryURL
		envs["APP_NAME"] = cicdRequest.CommonWorkflowRequest.AppName
		envs["TRIGGER_BY_AUTHOR"] = cicdRequest.CommonWorkflowRequest.TriggerByAuthor
		envs["DOCKER_IMAGE"] = image

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

		// setting extraEnvironmentVariables
		for k, v := range cicdRequest.CommonWorkflowRequest.ExtraEnvironmentVariables {
			envs[k] = v
		}
	} else {
		envs["DOCKER_IMAGE"] = cicdRequest.CommonWorkflowRequest.CiArtifactDTO.Image
		envs["DEPLOYMENT_RELEASE_ID"] = strconv.Itoa(cicdRequest.CommonWorkflowRequest.DeploymentReleaseCounter)
		envs["DEPLOYMENT_UNIQUE_ID"] = strconv.Itoa(cicdRequest.CommonWorkflowRequest.WorkflowRunnerId)
		envs["CD_TRIGGERED_BY"] = cicdRequest.CommonWorkflowRequest.DeploymentTriggeredBy
		envs["CD_TRIGGER_TIME"] = cicdRequest.CommonWorkflowRequest.DeploymentTriggerTime.String()

		// to support legacy yaml based script trigger
		envs["DEVTRON_CD_TRIGGERED_BY"] = cicdRequest.CommonWorkflowRequest.DeploymentTriggeredBy
		envs["DEVTRON_CD_TRIGGER_TIME"] = cicdRequest.CommonWorkflowRequest.DeploymentTriggerTime.String()

		for k, v := range cicdRequest.CommonWorkflowRequest.ExtraEnvironmentVariables {
			envs[k] = v
		}
	}
	return envs, nil
}

func getSystemEnvVariables() map[string]string {
	envs := make(map[string]string)
	//get all environment variables
	envVars := os.Environ()
	for _, envVar := range envVars {
		subs := strings.SplitN(envVar, "=", 2)
		envs[subs[0]] = subs[1]
	}
	return envs
}
