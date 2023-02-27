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
	"fmt"
	_ "github.com/aws/aws-sdk-go/aws"
	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	//args := `{"type":"CI","ciRequest":{"workflowNamePrefix":"12829-ci-624-4nx5-278","pipelineName":"ci-624-4nx5","pipelineId":278,"dockerImageTag":"7c97426a-278-12829","dockerRegistryId":"devtron-quay","dockerRegistryType":"other","dockerRegistryURL":"https://quay.io/devtron","dockerConnection":"secure","dockerCert":"","dockerRepository":"test","checkoutPath":"","dockerUsername":"devtron+devtron_ci","dockerPassword":"15CR23O1CFDOZT3VJVIWYJ610S757ZSGVWS87PCD1XDSHRYR5QKE179QX7O3W11O","awsRegion":"","accessKey":"","secretKey":"","ciCacheLocation":"devtron-ci-caching-prod","ciCacheRegion":"us-east-2","ciCacheFileName":"ci-624-4nx5-278.tar.gz","ciProjectDetails":[{"gitRepository":"git@github.com:ajaydevtron/devtron-test.git","materialName":"11-devtron-test","checkoutPath":"./","fetchSubmodules":false,"commitHash":"7c97426aa94528319578c898a732efbcecaf0758","gitTag":"","commitTime":"2023-01-24T23:19:06+05:30","type":"SOURCE_TYPE_BRANCH_FIXED","message":"mg test","author":"AJAY KUMAR\u003c99399155+ajaydevtron@users.noreply.github.com\u003e","gitOptions":{"userName":"","password":"","sshPrivateKey":"-----BEGIN OPENSSH PRIVATE KEY----- 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-----END OPENSSH PRIVATE KEY-----","accessToken":"","authMode":"SSH"},"sourceType":"SOURCE_TYPE_BRANCH_FIXED","sourceValue":"main","WebhookData":{"Id":0,"EventActionType":"","Data":null}}],"containerResources":{"minCpu":"","maxCpu":"","minStorage":"","maxStorage":"","minEphStorage":"","maxEphStorage":"","minMem":"","maxMem":""},"activeDeadlineSeconds":7200,"ciImage":"quay.io/devtron/ci-runner:b779491b-95-12827","namespace":"devtron-ci","workflowId":12829,"triggeredBy":15,"cacheLimit":5000000000,"beforeDockerBuildScripts":null,"afterDockerBuildScripts":null,"ciArtifactLocation":"s3://devtron-ci-logs-prod/arsenal-v1/ci-artifacts/12829/12829.zip","ciArtifactBucket":"devtron-ci-logs-prod","ciArtifactFileName":"arsenal-v1/ci-artifacts/12829/12829.zip","ciArtifactRegion":"","scanEnabled":true,"cloudProvider":"S3","blobStorageConfigured":true,"blobStorageS3Config":{"accessKey":"","passkey":"","endpointUrl":"","isInSecure":false,"ciLogBucketName":"devtron-ci-logs-prod","ciLogRegion":"us-east-2","ciLogBucketVersioning":true,"ciCacheBucketName":"devtron-ci-caching-prod","ciCacheRegion":"us-east-2","ciCacheBucketVersioning":true,"ciArtifactBucketName":"devtron-ci-logs-prod","ciArtifactRegion":"us-east-2","ciArtifactBucketVersioning":true},"azureBlobConfig":null,"gcpBlobConfig":null,"defaultAddressPoolBaseCidr":"","defaultAddressPoolSize":0,"preCiSteps":null,"postCiSteps":[{"name":"Task1","index":1,"stepType":"INLINE","executorType":"SHELL","script":"#!/bin/sh sleep 300#set -v  ## uncomment this to debug the script","inputVars":null,"exposedPorts":null,"outputVars":null,"triggerSkipConditions":null,"successFailureConditions":null,"dockerImage":"","command":"","args":null,"customScriptMount":null,"sourceCodeMount":null,"extraVolumeMounts":null,"artifactPaths":null}],"refPlugins":null,"appName":"common-app","triggerByAuthor":"admin","ciBuildConfig":{"id":11,"ciBuildType":"self-dockerfile-build","dockerBuildConfig":{"dockerfileRelativePath":"Dockerfile","dockerfileContent":""},"buildPackConfig":null},"ciBuildDockerMtuValue":-1,"ignoreDockerCachePush":false,"ignoreDockerCachePull":false,"cacheInvalidate":false,"IsPvcMounted":false,"extraEnvironmentVariables":{}},"cdRequest":null}`
	//args := `{"type":"DryRun","dryRunRequest":{"buildPackParams":{"builderId":"gcr.io/buildpacks/builder:v1"},"DockerBuildTargetPlatform":"", "workflowNamePrefix":"16-ci-25-w5x1-70","pipelineName":"ci-25-w5x1","pipelineId":70,"dockerImageTag":"da3ba326-70-17","dockerRegistryId":"devtron-quay","dockerRegistryType":"other","dockerRegistryURL":"https://quay.io/devtron","dockerConnection":"secure","dockerCert":"","dockerBuildArgs":"{}","dockerRepository":"test","dockerfileLocation":"Dockerfile","dockerUsername":"devtron+devtest","dockerPassword":"5WEDXDJMP6RV1CG1KKFJQL3MQOLC64JKM6K684WPEBKVWKOZ4LSMBHEHJU1HBGXK","awsRegion":"","accessKey":"","secretKey":"","ciCacheLocation":"","ciCacheRegion":"","ciCacheFileName":"ci-25-w5x1-70.tar.gz","ciProjectDetails":[{"gitRepository":"https://github.com/devtron-labs/sample-go-app","materialName":"1-getting-started-nodejs","checkoutPath":"./","fetchSubmodules":false,"commitHash":"8654623ec2bd9efd663935cb8332c8c765541837","gitTag":"","commitTime":"2022-04-12T20:26:08+05:30","type":"SOURCE_TYPE_BRANCH_FIXED","message":"Update README.md","author":"Prakarsh \u003c71125043+prakarsh-dt@users.noreply.github.com\u003e","gitOptions":{"userName":"","password":"","sshPrivateKey":"","accessToken":"","authMode":"ANONYMOUS"},"sourceType":"SOURCE_TYPE_BRANCH_FIXED","sourceValue":"master","WebhookData":{"Id":0,"EventActionType":"","Data":null}}],"containerResources":{"minCpu":"","maxCpu":"","minStorage":"","maxStorage":"","minEphStorage":"","maxEphStorage":"","minMem":"","maxMem":""},"activeDeadlineSeconds":3600,"ciImage":"quay.io/devtron/ci-runner:1290cf23-182-8015","namespace":"devtron-ci","workflowId":16,"triggeredBy":8,"cacheLimit":5000000000,"beforeDockerBuildScripts":null,"afterDockerBuildScripts":null,"ciArtifactLocation":"","invalidateCache":true,"scanEnabled":false,"cloudProvider":"AZURE","azureBlobConfig":{"enabled":true,"accountName":"devtrondemoblob","blobContainerCiLog":"","blobContainerCiCache":"cache","accountKey":"y1/K13YMp/v7uuvZNkKJ4dS3CyGc37bPIN9Hv8MVhog6OkG0joV05proQReMQIJQ8qXp0JVpj+mz+AStHNKR3Q=="},"minioEndpoint":"","defaultAddressPoolBaseCidr":"","defaultAddressPoolSize":0,"preCiSteps":[{"name":"Task 1","index":1,"stepType":"INLINE","executorType":"SHELL","refPluginId":0,"script":"echo $","inputVars":null,"exposedPorts":{"0":0},"outputVars":null,"triggerSkipConditions":null,"successFailureConditions":null,"dockerImage":"","command":"","args":null,"customScriptMountDestinationPath":{"sourcePath":"","destinationPath":""},"sourceCodeMountDestinationPath":{"sourcePath":"","destinationPath":""},"extraVolumeMounts":null,"artifactPaths":null}],"postCiSteps":null,"refPlugins":null},"cdRequest":null}`
	//' {"workflowNamePrefix":"55-suraj-23-ci-suraj-test-pipeline-8","pipelineName":"suraj-23-ci-suraj-test-pipeline","pipelineId":8,"dockerImageTag":"a6b809c4be87c217feba4af15cf5ebc3cafe21e0","dockerRegistryURL":"686244538589.dkr.ecr.us-east-2.amazonaws.com","dockerRepository":"test/suraj-23","dockerfileLocation":"./notifier/Dockerfile","awsRegion":"us-east-2","ciCacheLocation":"ci-caching","ciCacheFileName":"suraj-23-ci-suraj-test-pipeline.tar.gz","ciProjectDetails":[{"gitRepository":"https://gitlab.com/devtron/notifier.git","materialName":"1-notifier","checkoutPath":"./notifier","commitHash":"d4df38bcd065004014d255c2203d592a91585955","commitTime":"0001-01-01T00:00:00Z","branch":"ci_with_argo","type":"SOURCE_TYPE_BRANCH_FIXED","message":"test-commit","gitOptions":{"userName":"Suraj24","password":"Devtron@1234","sshKey":"","accessToken":"","authMode":"USERNAME_PASSWORD"}},{"gitRepository":"https://gitlab.com/devtron/orchestrator.git","materialName":"2-orchestrator","checkoutPath":"./orch","commitHash":"","commitTime":"0001-01-01T00:00:00Z","branch":"ci_with_argo","type":"SOURCE_TYPE_BRANCH_FIXED","message":"","gitOptions":{"userName":"Suraj24","password":"Devtron@1234","sshKey":"","accessToken":"","authMode":""}}],"ciImage":"686244538589.dkr.ecr.us-east-2.amazonaws.com/cirunner:latest","namespace":"default"}'
	args := os.Args[1]
	ciCdRequest := &helper.CiCdTriggerEvent{}
	err := json.Unmarshal([]byte(args), ciCdRequest)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" || logLevel == "DEBUG" {
		log.Println(util.DEVTRON, " ci-cd request details -----> ", args)
	}

	if ciCdRequest.Type == util.CIEVENT {

		ciRequest := ciCdRequest.CiRequest
		artifactUploaded, err := runCIStages(ciCdRequest)
		log.Println(util.DEVTRON, artifactUploaded, err)
		var artifactUploadErr error
		if !artifactUploaded {
			artifactUploadErr = helper.ZipAndUpload(ciRequest.BlobStorageConfigured, ciCdRequest.CiRequest.BlobStorageS3Config, ciCdRequest.CiRequest.CiArtifactFileName, ciCdRequest.CiRequest.CloudProvider, ciCdRequest.CiRequest.AzureBlobConfig, ciCdRequest.CiRequest.GcpBlobConfig)
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
	return helper.UploadArtifact(cdRequest.BlobStorageConfigured, artifactFiles, cdRequest.BlobStorageS3Config, cdRequest.ArtifactFileName, cdRequest.CloudProvider, cdRequest.AzureBlobConfig, cdRequest.GcpBlobConfig)
}

func getGlobalEnvVariables(cicdRequest *helper.CiCdTriggerEvent) (map[string]string, error) {
	envs := make(map[string]string)
	envs["WORKING_DIRECTORY"] = util.WORKINGDIR
	if cicdRequest.Type == util.CIEVENT {
		image, err := helper.BuildDockerImagePath(cicdRequest.CiRequest)
		if err != nil {
			return nil, err
		}
		envs["DOCKER_IMAGE_TAG"] = cicdRequest.CiRequest.DockerImageTag
		envs["DOCKER_REPOSITORY"] = cicdRequest.CiRequest.DockerRepository
		envs["DOCKER_REGISTRY_URL"] = cicdRequest.CiRequest.DockerRegistryURL
		envs["APP_NAME"] = cicdRequest.CiRequest.AppName
		envs["TRIGGER_BY_AUTHOR"] = cicdRequest.CiRequest.TriggerByAuthor
		envs["DOCKER_IMAGE"] = image

		//adding GIT_MATERIAL_REQUEST in env for semgrep plugin
		CiMaterialRequestArr := ""
		if cicdRequest.CiRequest.CiProjectDetails != nil {
			for _, ciProjectDetail := range cicdRequest.CiRequest.CiProjectDetails {
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
		for k, v := range cicdRequest.CiRequest.ExtraEnvironmentVariables {
			envs[k] = v
		}
	} else {
		envs["DOCKER_IMAGE"] = cicdRequest.CdRequest.CiArtifactDTO.Image
		envs["DEPLOYMENT_RELEASE_ID"] = strconv.Itoa(cicdRequest.CdRequest.DeploymentReleaseCounter)
		envs["DEPLOYMENT_UNIQUE_ID"] = strconv.Itoa(cicdRequest.CdRequest.WorkflowRunnerId)
		envs["DEVTRON_CD_TRIGGERED_BY"] = cicdRequest.CdRequest.DeploymentTriggeredBy
		envs["DEVTRON_CD_TRIGGER_TIME"] = cicdRequest.CdRequest.DeploymentTriggerTime.String()
		for k, v := range cicdRequest.CdRequest.ExtraEnvironmentVariables {
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
		a := strings.Split(envVar, "=")
		envs[a[0]] = a[1]
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
	helper.StartDockerDaemon(ciCdRequest.CiRequest.DockerConnection, ciCdRequest.CiRequest.DockerRegistryURL, ciCdRequest.CiRequest.DockerCert, ciCdRequest.CiRequest.DefaultAddressPoolBaseCidr, ciCdRequest.CiRequest.DefaultAddressPoolSize, ciCdRequest.CiRequest.CiBuildDockerMtuValue)

	scriptEnvs, err := getGlobalEnvVariables(ciCdRequest)
	if err != nil {
		return artifactUploaded, err
	}
	// Get devtron-ci yaml
	yamlLocation := ciCdRequest.CiRequest.CheckoutPath
	log.Println(util.DEVTRON, "devtron-ci yaml location ", yamlLocation)
	taskYaml, err := helper.GetTaskYaml(yamlLocation)
	if err != nil {
		return artifactUploaded, err
	}
	ciCdRequest.CiRequest.TaskYaml = taskYaml

	ciBuildConfigBean := ciCdRequest.CiRequest.CiBuildConfig
	if ciBuildConfigBean != nil && ciBuildConfigBean.CiBuildType == helper.MANAGED_DOCKERFILE_BUILD_TYPE {
		err = makeDockerfile(ciBuildConfigBean.DockerBuildConfig)
		if err != nil {
			return artifactUploaded, err
		}
	}

	refStageMap := make(map[int][]*helper.StepObject)
	for _, ref := range ciCdRequest.CiRequest.RefPlugins {
		refStageMap[ref.Id] = ref.Steps
	}

	var preeCiStageOutVariable map[int]map[string]*helper.VariableObject
	if len(ciCdRequest.CiRequest.PreCiSteps) > 0 {
		util.LogStage("running PRE-CI steps")
		// run pre artifact processing
		preeCiStageOutVariable, err = RunCiSteps(STEP_TYPE_PRE, ciCdRequest.CiRequest.PreCiSteps, refStageMap, scriptEnvs, nil)
		if err != nil {
			log.Println(err)
			return artifactUploaded, err
		}
	}

	util.LogStage("Build")
	// build
	dest, err := helper.BuildArtifact(ciCdRequest.CiRequest) //TODO make it skipable
	if err != nil {
		return artifactUploaded, err
	}
	log.Println(util.DEVTRON, " /Build")

	if len(ciCdRequest.CiRequest.PostCiSteps) > 0 {
		util.LogStage("running POST-CI steps")
		// sending build success as true always as post-ci triggers only if ci gets success
		scriptEnvs[util.ENV_VARIABLE_BUILD_SUCCESS] = "true"
		// run post artifact processing
		_, err = RunCiSteps(STEP_TYPE_POST, ciCdRequest.CiRequest.PostCiSteps, refStageMap, scriptEnvs, preeCiStageOutVariable)
		if err != nil {
			return artifactUploaded, err
		}
	}

	var digest string
	buildSkipEnabled := ciBuildConfigBean != nil && ciBuildConfigBean.CiBuildType == helper.BUILD_SKIP_BUILD_TYPE
	if !buildSkipEnabled {
		isBuildX := ciBuildConfigBean != nil && ciBuildConfigBean.DockerBuildConfig != nil && ciBuildConfigBean.DockerBuildConfig.TargetPlatform != ""
		if isBuildX {
			digest, err = helper.ExtractDigestForBuildx(dest)
		} else {
			util.LogStage("docker push")
			// push to dest
			log.Println(util.DEVTRON, " docker-push")
			err = helper.PushArtifact(dest)
			if err != nil {
				return artifactUploaded, err
			}
			digest, err = helper.ExtractDigestUsingPull(dest)
		}
	}

	if err != nil {
		return artifactUploaded, err
	}
	log.Println(util.DEVTRON, " /docker-push")

	log.Println(util.DEVTRON, " artifact-upload")
	err = helper.ZipAndUpload(ciCdRequest.CiRequest.BlobStorageConfigured, ciCdRequest.CiRequest.BlobStorageS3Config, ciCdRequest.CiRequest.CiArtifactFileName, ciCdRequest.CiRequest.CloudProvider, ciCdRequest.CiRequest.AzureBlobConfig, ciCdRequest.CiRequest.GcpBlobConfig)
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

func makeDockerfile(config *helper.DockerBuildConfig) error {
	dockerfileContent := config.DockerfileContent
	dockerfilePath := filepath.Join(util.WORKINGDIR, "./Dockerfile")
	f, err := os.Create(dockerfilePath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(dockerfileContent)
	return err
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
	helper.StartDockerDaemon(cicdRequest.CdRequest.DockerConnection, cicdRequest.CdRequest.DockerRegistryURL, cicdRequest.CdRequest.DockerCert, cicdRequest.CdRequest.DefaultAddressPoolBaseCidr, cicdRequest.CdRequest.DefaultAddressPoolSize, -1)

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

	scriptEnvs, err := getGlobalEnvVariables(cicdRequest)
	if err != nil {
		return err
	}
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
