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
	"encoding/json"
	"log"
	"os"
	"strings"

	_ "github.com/aws/aws-sdk-go/aws"
	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
)

func main() {
	//' {"workflowNamePrefix":"55-suraj-23-ci-suraj-test-pipeline-8","pipelineName":"suraj-23-ci-suraj-test-pipeline","pipelineId":8,"dockerImageTag":"a6b809c4be87c217feba4af15cf5ebc3cafe21e0","dockerRegistryURL":"686244538589.dkr.ecr.us-east-2.amazonaws.com","dockerRepository":"test/suraj-23","dockerfileLocation":"./notifier/Dockerfile","awsRegion":"us-east-2","ciCacheLocation":"ci-caching","ciCacheFileName":"suraj-23-ci-suraj-test-pipeline.tar.gz","ciProjectDetails":[{"gitRepository":"https://gitlab.com/devtron/notifier.git","materialName":"1-notifier","checkoutPath":"./notifier","commitHash":"d4df38bcd065004014d255c2203d592a91585955","commitTime":"0001-01-01T00:00:00Z","branch":"ci_with_argo","type":"SOURCE_TYPE_BRANCH_FIXED","message":"test-commit","gitOptions":{"userName":"Suraj24","password":"Devtron@1234","sshKey":"","accessToken":"","authMode":"USERNAME_PASSWORD"}},{"gitRepository":"https://gitlab.com/devtron/orchestrator.git","materialName":"2-orchestrator","checkoutPath":"./orch","commitHash":"","commitTime":"0001-01-01T00:00:00Z","branch":"ci_with_argo","type":"SOURCE_TYPE_BRANCH_FIXED","message":"","gitOptions":{"userName":"Suraj24","password":"Devtron@1234","sshKey":"","accessToken":"","authMode":""}}],"ciImage":"686244538589.dkr.ecr.us-east-2.amazonaws.com/cirunner:latest","namespace":"default"}'
	args := os.Args[1]
	ciCdRequest := &helper.CiCdTriggerEvent{}
	err := json.Unmarshal([]byte(args), ciCdRequest)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	log.Println(util.DEVTRON, " ci-cd request details -----> ", args)

	if ciCdRequest.Type == util.CIEVENT {
		ciRequest := ciCdRequest.CiRequest
		artifactUploaded, err := runCIStages(ciCdRequest)
		log.Println(util.DEVTRON, artifactUploaded, err)
		var artifactUploadErr error
		if !artifactUploaded {
			artifactUploadErr = collectAndUploadArtifact(ciRequest)
		}

		if err != nil || artifactUploadErr != nil {
			log.Println(util.DEVTRON, err, artifactUploadErr)
			os.Exit(1)
		}

		// sync cache
		log.Println(util.DEVTRON, " cache-push")
		err = helper.SyncCache(ciRequest)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		log.Println(util.DEVTRON, " /cache-push")
	} else {
		err = runCDStages(ciCdRequest)
		artifactUploadErr := collectAndUploadCDArtifacts(ciCdRequest.CdRequest)
		if err != nil || artifactUploadErr != nil {
			log.Println(err)
			os.Exit(1)
		}
	}
}

func collectAndUploadCDArtifacts(cdRequest *helper.CdRequest) error {
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
	return helper.UploadArtifact(artifactFiles, cdRequest.ArtifactLocation, cdRequest.CloudProvider, cdRequest.MinioEndpoint, cdRequest.AzureBlobConfig)
}

func collectAndUploadArtifact(ciRequest *helper.CiRequest) error {
	artifactFiles := make(map[string]string)
	var allTasks []*helper.Task
	if ciRequest.TaskYaml != nil {
		for _, pc := range ciRequest.TaskYaml.PipelineConf {
			for _, t := range append(pc.BeforeTasks, pc.AfterTasks...) {
				allTasks = append(allTasks, t)
			}
		}
	}

	allTasks = append(allTasks, ciRequest.BeforeDockerBuild...)
	allTasks = append(allTasks, ciRequest.AfterDockerBuild...)

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
	return helper.UploadArtifact(artifactFiles, ciRequest.CiArtifactLocation, ciRequest.CloudProvider, ciRequest.MinioEndpoint, ciRequest.AzureBlobConfig)
}

func getScriptEnvVariables(cicdRequest *helper.CiCdTriggerEvent) map[string]string {
	envs := make(map[string]string)
	//TODO ADD MORE env variable
	if cicdRequest.Type == util.CIEVENT {
		envs["DOCKER_IMAGE_TAG"] = cicdRequest.CiRequest.DockerImageTag
		envs["DOCKER_REPOSITORY"] = cicdRequest.CiRequest.DockerRepository
		envs["DOCKER_REGISTRY_URL"] = cicdRequest.CiRequest.DockerRegistryURL
	} else {
		envs["DOCKER_IMAGE"] = cicdRequest.CdRequest.CiArtifactDTO.Image
		for k, v := range cicdRequest.CdRequest.ExtraEnvironmentVariables {
			envs[k] = v
		}
	}
	return envs
}

func runCIStages(ciCdRequest *helper.CiCdTriggerEvent) (artifactUploaded bool, err error) {
	artifactUploaded = false
	err = os.Chdir("/")
	if err != nil {
		return artifactUploaded, err
	}

	if _, err := os.Stat(util.WORKINGDIR); os.IsNotExist(err) {
		_ = os.Mkdir(util.WORKINGDIR, os.ModeDir)
	}

	// Get ci cache
	log.Println(util.DEVTRON, " cache-pull")
	err = helper.GetCache(ciCdRequest.CiRequest)
	if err != nil {
		return artifactUploaded, err
	}
	log.Println(util.DEVTRON, " /cache-pull")

	err = os.Chdir(util.WORKINGDIR)
	if err != nil {
		return artifactUploaded, err
	}
	// git handling
	log.Println(util.DEVTRON, " git")
	err = helper.CloneAndCheckout(ciCdRequest.CiRequest.CiProjectDetails)
	if err != nil {
		log.Println(util.DEVTRON, "clone err: ", err)
		return artifactUploaded, err
	}
	log.Println(util.DEVTRON, " /git")

	// Start docker daemon
	log.Println(util.DEVTRON, " docker-build")
	helper.StartDockerDaemon(ciCdRequest.CiRequest.DockerConnection, ciCdRequest.CiRequest.DockerRegistryURL, ciCdRequest.CiRequest.DockerCert, ciCdRequest.CiRequest.DefaultAddressPoolBaseCidr, ciCdRequest.CiRequest.DefaultAddressPoolSize)
	scriptEnvs := getScriptEnvVariables(ciCdRequest)

	// Get devtron-ci yaml
	yamlLocation := ciCdRequest.CiRequest.DockerFileLocation[:strings.LastIndex(ciCdRequest.CiRequest.DockerFileLocation, "/")+1]
	log.Println(util.DEVTRON, "devtron-ci yaml location ", yamlLocation)
	taskYaml, err := helper.GetTaskYaml(yamlLocation)
	if err != nil {
		return artifactUploaded, err
	}
	ciCdRequest.CiRequest.TaskYaml = taskYaml

	// run pre artifact processing
	err = RunPreDockerBuildTasks(ciCdRequest.CiRequest, scriptEnvs, taskYaml)
	if err != nil {
		log.Println(err)
		return artifactUploaded, err
	}

	util.LogStage("docker build")
	// build
	dest, err := helper.BuildArtifact(ciCdRequest.CiRequest)
	if err != nil {
		return artifactUploaded, err
	}
	log.Println(util.DEVTRON, " /docker-build")

	// run post artifact processing
	err = RunPostDockerBuildTasks(ciCdRequest.CiRequest, scriptEnvs, taskYaml)
	if err != nil {
		return artifactUploaded, err
	}

	util.LogStage("docker push")
	// push to dest
	log.Println(util.DEVTRON, " docker-push")
	digest, err := helper.PushArtifact(ciCdRequest.CiRequest, dest)
	if err != nil {
		return artifactUploaded, err
	}
	log.Println(util.DEVTRON, " /docker-push")

	log.Println(util.DEVTRON, " artifact-upload")
	err = collectAndUploadArtifact(ciCdRequest.CiRequest)
	if err != nil {
		return artifactUploaded, err
	} else {
		artifactUploaded = true
	}
	log.Println(util.DEVTRON, " /artifact-upload")

	// scan only if ci scan enabled
	if ciCdRequest.CiRequest.ScanEnabled {
		util.LogStage("IMAGE SCAN")
		log.Println(util.DEVTRON, " /image-scanner")
		scanEvent := &helper.ScanEvent{Image: dest, ImageDigest: digest, PipelineId: ciCdRequest.CiRequest.PipelineId, UserId: ciCdRequest.CiRequest.TriggeredBy}
		scanEvent.DockerRegistryId = ciCdRequest.CiRequest.DockerRegistryId
		err = helper.SendEventToClairUtility(scanEvent)
		if err != nil {
			log.Println(err)
			return artifactUploaded, err
		}
		log.Println(util.DEVTRON, " /image-scanner")
	}

	log.Println(util.DEVTRON, " event")
	err = helper.SendEvents(ciCdRequest.CiRequest, digest, dest)
	if err != nil {
		log.Println(err)
		return artifactUploaded, err
	}
	log.Println(util.DEVTRON, " /event")

	err = helper.StopDocker()
	if err != nil {
		log.Println("err", err)
		return artifactUploaded, err
	}
	return artifactUploaded, nil
}

func runCDStages(cicdRequest *helper.CiCdTriggerEvent) error {
	err := os.Chdir("/")
	if err != nil {
		return err
	}

	if _, err := os.Stat(util.WORKINGDIR); os.IsNotExist(err) {
		_ = os.Mkdir(util.WORKINGDIR, os.ModeDir)
	}
	err = os.Chdir(util.WORKINGDIR)
	if err != nil {
		return err
	}
	// git handling
	log.Println(util.DEVTRON, " git")
	err = helper.CloneAndCheckout(cicdRequest.CdRequest.CiProjectDetails)
	if err != nil {
		log.Println(util.DEVTRON, "clone err: ", err)
		return err
	}
	log.Println(util.DEVTRON, " /git")

	// Start docker daemon
	log.Println(util.DEVTRON, " docker-start")
	helper.StartDockerDaemon(cicdRequest.CdRequest.DockerConnection, cicdRequest.CdRequest.DockerRegistryURL, cicdRequest.CdRequest.DockerCert, cicdRequest.CdRequest.DefaultAddressPoolBaseCidr, cicdRequest.CdRequest.DefaultAddressPoolSize)

	err = helper.DockerLogin(&helper.DockerCredentials{
		DockerUsername:     cicdRequest.CdRequest.DockerUsername,
		DockerPassword:     cicdRequest.CdRequest.DockerPassword,
		AwsRegion:          cicdRequest.CdRequest.AwsRegion,
		AccessKey:          cicdRequest.CdRequest.AccessKey,
		SecretKey:          cicdRequest.CdRequest.SecretKey,
		DockerRegistryURL:  cicdRequest.CdRequest.DockerRegistryURL,
		DockerRegistryType: cicdRequest.CdRequest.DockerRegistryType,
	})
	if err != nil {
		return err
	}
	// Get devtron-cd yaml
	taskYaml, err := helper.ToTaskYaml([]byte(cicdRequest.CdRequest.StageYaml))
	if err != nil {
		log.Println(err)
		return err
	}
	cicdRequest.CdRequest.TaskYaml = taskYaml

	// run post artifact processing
	log.Println(util.DEVTRON, " stage yaml", taskYaml)
	var tasks []*helper.Task
	for _, t := range taskYaml.CdPipelineConfig {
		tasks = append(tasks, t.BeforeTasks...)
		tasks = append(tasks, t.AfterTasks...)
	}

	scriptEnvs := getScriptEnvVariables(cicdRequest)
	err = RunCdStageTasks(tasks, scriptEnvs)
	if err != nil {
		return err
	}

	log.Println(util.DEVTRON, " event")
	err = helper.SendCDEvent(cicdRequest.CdRequest)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println(util.DEVTRON, " /event")

	err = helper.StopDocker()
	if err != nil {
		log.Println("err", err)
		return err
	}
	return nil
}
