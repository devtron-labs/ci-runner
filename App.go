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
	"errors"
	"fmt"
	_ "github.com/aws/aws-sdk-go/aws"
	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
	blob_storage "github.com/devtron-labs/common-lib/blob-storage"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func spawnProcess() error {

	os.Setenv("IN_APP_LOGGING", "false")

	// Create an in-memory pipe
	pr, pw := io.Pipe()

	// Create the cirunner command
	cirunnerCmd := exec.Command("./cirunner")
	cirunnerCmd.Stdout = pw
	cirunnerCmd.Stderr = pw

	// Create the tee command
	teeCmd := exec.Command("tee", "main.log")
	teeCmd.Stdin = pr
	teeCmd.Stdout = os.Stdout

	var err error
	var ciRunnerPid = 0
	var exitCode int
	// Start cirunner
	if err = cirunnerCmd.Start(); err != nil {
		return err
	}

	ciRunnerPid = cirunnerCmd.Process.Pid
	fmt.Println("ci runner PID: ", ciRunnerPid)

	// Start tee
	if err = teeCmd.Start(); err != nil {
		return err
	}

	// Create a channel to receive the SIGTERM signal
	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, syscall.SIGTERM)
	go func() {
		log.Println(util.DEVTRON, "SIGTERM listener started in parent process!")
		receivedSignal := <-sigTerm
		log.Println(util.DEVTRON, "signal received in parent process: ", receivedSignal)
		cirunnerCmd.Process.Signal(syscall.SIGTERM)
		os.Exit(util.DefaultErrorCode)
	}()

	p, err := cirunnerCmd.Process.Wait()
	// Wait for cirunner to finish
	//if p, err := cirunnerCmd.Process.Wait(); err != nil {
	//	exitCode = p.ExitCode()
	//	fmt.Println("ci runner exit code: ", ciRunnerPid)
	//	//return err
	//}

	exitCode = p.ExitCode()
	fmt.Println("ci runner exit code: ", ciRunnerPid)

	// Close write end of the pipe
	err = pw.Close()

	os.Exit(exitCode)
	return nil
}

var handleOnce sync.Once

func handleCleanup(ciCdRequest helper.CiCdTriggerEvent, exitCode *int, source string) {
	handleOnce.Do(func() {
		log.Println(util.DEVTRON, " CI-Runner cleanup executed with exit Code", *exitCode, source)
		uploadLogs(ciCdRequest, exitCode)
		//if source == util.Source_Signal {
		//
		//}
		log.Println(util.DEVTRON, " Exiting with exit code ", *exitCode)
		os.Exit(*exitCode)
	})
}

func main() {
	//args := `{"type":"CI","ciRequest":{"DockerBuildTargetPlatform":"linux/arm64", "workflowNamePrefix":"16-ci-25-w5x1-70","pipelineName":"ci-25-w5x1","pipelineId":70,"dockerImageTag":"da3ba326-70-17","dockerRegistryId":"devtron-quay","dockerRegistryType":"other","dockerRegistryURL":"https://quay.io/devtron","dockerConnection":"secure","dockerCert":"","dockerBuildArgs":"{}","dockerRepository":"test","dockerfileLocation":"Dockerfile","dockerUsername":"devtron+devtest","dockerPassword":"5WEDXDJMP6RV1CG1KKFJQL3MQOLC64JKM6K684WPEBKVWKOZ4LSMBHEHJU1HBGXK","awsRegion":"","accessKey":"","secretKey":"","ciCacheLocation":"","ciCacheRegion":"","ciCacheFileName":"ci-25-w5x1-70.tar.gz","ciProjectDetails":[{"gitRepository":"https://github.com/devtron-labs/getting-started-nodejs","materialName":"1-getting-started-nodejs","checkoutPath":"./","fetchSubmodules":false,"commitHash":"da3ba3254712965b5944a6271e71bff91fe51f20","gitTag":"","commitTime":"2022-04-12T20:26:08+05:30","type":"SOURCE_TYPE_BRANCH_FIXED","message":"Update README.md","author":"Prakarsh \u003c71125043+prakarsh-dt@users.noreply.github.com\u003e","gitOptions":{"userName":"","password":"","sshPrivateKey":"","accessToken":"","authMode":"ANONYMOUS"},"sourceType":"SOURCE_TYPE_BRANCH_FIXED","sourceValue":"master","WebhookData":{"Id":0,"EventActionType":"","Data":null}}],"containerResources":{"minCpu":"","maxCpu":"","minStorage":"","maxStorage":"","minEphStorage":"","maxEphStorage":"","minMem":"","maxMem":""},"activeDeadlineSeconds":3600,"ciImage":"quay.io/devtron/ci-runner:1290cf23-182-8015","namespace":"devtron-ci","workflowId":16,"triggeredBy":8,"cacheLimit":5000000000,"beforeDockerBuildScripts":null,"afterDockerBuildScripts":null,"ciArtifactLocation":"","invalidateCache":true,"scanEnabled":false,"cloudProvider":"AZURE","azureBlobConfig":{"enabled":true,"accountName":"devtrondemoblob","blobContainerCiLog":"","blobContainerCiCache":"cache","accountKey":"y1/K13YMp/v7uuvZNkKJ4dS3CyGc37bPIN9Hv8MVhog6OkG0joV05proQReMQIJQ8qXp0JVpj+mz+AStHNKR3Q=="},"minioEndpoint":"","defaultAddressPoolBaseCidr":"","defaultAddressPoolSize":0,"preCiSteps":[{"name":"Task 1","index":1,"stepType":"INLINE","executorType":"SHELL","refPluginId":0,"script":"echo $","inputVars":null,"exposedPorts":{"0":0},"outputVars":null,"triggerSkipConditions":null,"successFailureConditions":null,"dockerImage":"","command":"","args":null,"customScriptMountDestinationPath":{"sourcePath":"","destinationPath":""},"sourceCodeMountDestinationPath":{"sourcePath":"","destinationPath":""},"extraVolumeMounts":null,"artifactPaths":null}],"postCiSteps":null,"refPlugins":null},"cdRequest":null}`
	//args := `{"type":"DryRun","dryRunRequest":{"buildPackParams":{"builderId":"gcr.io/buildpacks/builder:v1"},"DockerBuildTargetPlatform":"", "workflowNamePrefix":"16-ci-25-w5x1-70","pipelineName":"ci-25-w5x1","pipelineId":70,"dockerImageTag":"da3ba326-70-17","dockerRegistryId":"devtron-quay","dockerRegistryType":"other","dockerRegistryURL":"https://quay.io/devtron","dockerConnection":"secure","dockerCert":"","dockerBuildArgs":"{}","dockerRepository":"test","dockerfileLocation":"Dockerfile","dockerUsername":"devtron+devtest","dockerPassword":"5WEDXDJMP6RV1CG1KKFJQL3MQOLC64JKM6K684WPEBKVWKOZ4LSMBHEHJU1HBGXK","awsRegion":"","accessKey":"","secretKey":"","ciCacheLocation":"","ciCacheRegion":"","ciCacheFileName":"ci-25-w5x1-70.tar.gz","ciProjectDetails":[{"gitRepository":"https://github.com/devtron-labs/sample-go-app","materialName":"1-getting-started-nodejs","checkoutPath":"./","fetchSubmodules":false,"commitHash":"8654623ec2bd9efd663935cb8332c8c765541837","gitTag":"","commitTime":"2022-04-12T20:26:08+05:30","type":"SOURCE_TYPE_BRANCH_FIXED","message":"Update README.md","author":"Prakarsh \u003c71125043+prakarsh-dt@users.noreply.github.com\u003e","gitOptions":{"userName":"","password":"","sshPrivateKey":"","accessToken":"","authMode":"ANONYMOUS"},"sourceType":"SOURCE_TYPE_BRANCH_FIXED","sourceValue":"master","WebhookData":{"Id":0,"EventActionType":"","Data":null}}],"containerResources":{"minCpu":"","maxCpu":"","minStorage":"","maxStorage":"","minEphStorage":"","maxEphStorage":"","minMem":"","maxMem":""},"activeDeadlineSeconds":3600,"ciImage":"quay.io/devtron/ci-runner:1290cf23-182-8015","namespace":"devtron-ci","workflowId":16,"triggeredBy":8,"cacheLimit":5000000000,"beforeDockerBuildScripts":null,"afterDockerBuildScripts":null,"ciArtifactLocation":"","invalidateCache":true,"scanEnabled":false,"cloudProvider":"AZURE","azureBlobConfig":{"enabled":true,"accountName":"devtrondemoblob","blobContainerCiLog":"","blobContainerCiCache":"cache","accountKey":"y1/K13YMp/v7uuvZNkKJ4dS3CyGc37bPIN9Hv8MVhog6OkG0joV05proQReMQIJQ8qXp0JVpj+mz+AStHNKR3Q=="},"minioEndpoint":"","defaultAddressPoolBaseCidr":"","defaultAddressPoolSize":0,"preCiSteps":[{"name":"Task 1","index":1,"stepType":"INLINE","executorType":"SHELL","refPluginId":0,"script":"echo $","inputVars":null,"exposedPorts":{"0":0},"outputVars":null,"triggerSkipConditions":null,"successFailureConditions":null,"dockerImage":"","command":"","args":null,"customScriptMountDestinationPath":{"sourcePath":"","destinationPath":""},"sourceCodeMountDestinationPath":{"sourcePath":"","destinationPath":""},"extraVolumeMounts":null,"artifactPaths":null}],"postCiSteps":null,"refPlugins":null},"cdRequest":null}`
	//' {"workflowNamePrefix":"55-suraj-23-ci-suraj-test-pipeline-8","pipelineName":"suraj-23-ci-suraj-test-pipeline","pipelineId":8,"dockerImageTag":"a6b809c4be87c217feba4af15cf5ebc3cafe21e0","dockerRegistryURL":"686244538589.dkr.ecr.us-east-2.amazonaws.com","dockerRepository":"test/suraj-23","dockerfileLocation":"./notifier/Dockerfile","awsRegion":"us-east-2","ciCacheLocation":"ci-caching","ciCacheFileName":"suraj-23-ci-suraj-test-pipeline.tar.gz","ciProjectDetails":[{"gitRepository":"https://gitlab.com/devtron/notifier.git","materialName":"1-notifier","checkoutPath":"./notifier","commitHash":"d4df38bcd065004014d255c2203d592a91585955","commitTime":"0001-01-01T00:00:00Z","branch":"ci_with_argo","type":"SOURCE_TYPE_BRANCH_FIXED","message":"test-commit","gitOptions":{"userName":"Suraj24","password":"Devtron@1234","sshKey":"","accessToken":"","authMode":"USERNAME_PASSWORD"}},{"gitRepository":"https://gitlab.com/devtron/orchestrator.git","materialName":"2-orchestrator","checkoutPath":"./orch","commitHash":"","commitTime":"0001-01-01T00:00:00Z","branch":"ci_with_argo","type":"SOURCE_TYPE_BRANCH_FIXED","message":"","gitOptions":{"userName":"Suraj24","password":"Devtron@1234","sshKey":"","accessToken":"","authMode":""}}],"ciImage":"686244538589.dkr.ecr.us-east-2.amazonaws.com/cirunner:latest","namespace":"default"}'

	if os.Getenv("IN_APP_LOGGING") == "true" {
		spawnProcess()
	}

	args := os.Getenv(util.CiCdEventEnvKey)
	processEvent(args)

}

func processEvent(args string) {

	exitCode := 0
	ciCdRequest := &helper.CiCdTriggerEvent{}
	err := json.Unmarshal([]byte(args), ciCdRequest)
	if err != nil {
		log.Println(err)
		exitCode = util.DefaultErrorCode
		return
	}

	// Create a channel to receive the SIGTERM signal
	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGSEGV, syscall.SIGKILL)

	go func() {
		var defaultErrorCode = util.DefaultErrorCode
		log.Println(util.DEVTRON, "SIGTERM listener started!")
		receivedSignal := <-sigTerm
		log.Println(util.DEVTRON, "signal received: ", receivedSignal)
		handleCleanup(*ciCdRequest, &defaultErrorCode, util.Source_Signal)
	}()

	defer handleCleanup(*ciCdRequest, &exitCode, util.Source_Defer)
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
			artifactUploaded, artifactUploadErr = helper.ZipAndUpload(ciRequest.BlobStorageConfigured, ciCdRequest.CiRequest.BlobStorageS3Config, ciCdRequest.CiRequest.CiArtifactFileName, ciCdRequest.CiRequest.CloudProvider, ciCdRequest.CiRequest.AzureBlobConfig, ciCdRequest.CiRequest.GcpBlobConfig)
		}

		if err != nil {
			var stageError *helper.CiStageError
			log.Println(util.DEVTRON, err)
			if errors.As(err, &stageError) {
				exitCode = util.CiStageFailErrorCode
				return
			}
			exitCode = util.DefaultErrorCode
			return
		}

		if artifactUploadErr != nil {
			log.Println(util.DEVTRON, artifactUploadErr)
			exitCode = util.DefaultErrorCode
			return
		}

		// sync cache
		log.Println(util.DEVTRON, " cache-push")
		err = helper.SyncCache(ciRequest)
		if err != nil {
			log.Println(err)
			exitCode = util.DefaultErrorCode
			return
		}
		log.Println(util.DEVTRON, " /cache-push")
	} else {
		err = runCDStages(ciCdRequest)
		artifactUploadErr := collectAndUploadCDArtifacts(ciCdRequest.CdRequest)
		if err != nil || artifactUploadErr != nil {
			log.Println(err)
			exitCode = util.DefaultErrorCode
			return
		}
	}
	return
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

func sendFailureNotification(failureMessage string, ciRequest *helper.CiRequest,
	digest string, image string, ciMetrics helper.CIMetrics,
	artifactUploaded bool, err error) (bool, error) {
	e := helper.SendEvents(ciRequest, digest, image, ciMetrics, artifactUploaded, failureMessage)
	if e != nil {
		log.Println(e)
		return artifactUploaded, e
	}
	return artifactUploaded, &helper.CiStageError{Err: err}
}

type CiFailReason string

const (
	PreCi  CiFailReason = "Pre-CI task failed: "
	PostCi CiFailReason = "Post-CI task failed: "
	Build  CiFailReason = "Docker build failed"
	Push   CiFailReason = "Docker push failed"
	Scan   CiFailReason = "Image scan failed"
)

func runCIStages(ciCdRequest *helper.CiCdTriggerEvent) (artifactUploaded bool, err error) {

	var metrics helper.CIMetrics
	start := time.Now()
	metrics.TotalStartTime = start
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
	start = time.Now()
	metrics.CacheDownStartTime = start
	err = helper.GetCache(ciCdRequest.CiRequest)
	metrics.CacheDownDuration = time.Since(start).Seconds()
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
		err = makeDockerfile(ciBuildConfigBean.DockerBuildConfig, ciCdRequest.CiRequest.CheckoutPath)
		if err != nil {
			return artifactUploaded, err
		}
	}

	refStageMap := make(map[int][]*helper.StepObject)
	for _, ref := range ciCdRequest.CiRequest.RefPlugins {
		refStageMap[ref.Id] = ref.Steps
	}

	var preeCiStageOutVariable map[int]map[string]*helper.VariableObject
	var step *helper.StepObject
	var preCiDuration float64
	start = time.Now()
	metrics.PreCiStartTime = start
	buildSkipEnabled := ciBuildConfigBean != nil && ciBuildConfigBean.CiBuildType == helper.BUILD_SKIP_BUILD_TYPE
	if len(ciCdRequest.CiRequest.PreCiSteps) > 0 {
		if !buildSkipEnabled {
			util.LogStage("running PRE-CI steps")
		}
		// run pre artifact processing
		preeCiStageOutVariable, step, err = RunCiSteps(STEP_TYPE_PRE, ciCdRequest.CiRequest.PreCiSteps, refStageMap, scriptEnvs, nil)
		preCiDuration = time.Since(start).Seconds()
		if err != nil {
			log.Println(err)
			return sendFailureNotification(string(PreCi)+step.Name, ciCdRequest.CiRequest, "", "", metrics, artifactUploaded, err)

		}
	}
	metrics.PreCiDuration = preCiDuration
	var dest string
	if !buildSkipEnabled {
		util.LogStage("Build")
		// build
		start = time.Now()
		metrics.BuildStartTime = start
		dest, err = helper.BuildArtifact(ciCdRequest.CiRequest) //TODO make it skipable
		metrics.BuildDuration = time.Since(start).Seconds()
		if err != nil {
			// code-block starts : run post-ci which are enabled to run on ci fail
			postCiStepsToTriggerOnCiFail := getPostCiStepToRunOnCiFail(ciCdRequest.CiRequest.PostCiSteps)
			if len(postCiStepsToTriggerOnCiFail) > 0 {
				util.LogStage("Running POST-CI steps which are enabled to RUN even on CI FAIL")
				// build success will always be false
				scriptEnvs[util.ENV_VARIABLE_BUILD_SUCCESS] = "false"
				// run post artifact processing
				RunCiSteps(STEP_TYPE_POST, postCiStepsToTriggerOnCiFail, refStageMap, scriptEnvs, preeCiStageOutVariable)
			}
			// code-block ends
			return sendFailureNotification(string(Build), ciCdRequest.CiRequest, "", "", metrics, artifactUploaded, err)
		}
		log.Println(util.DEVTRON, " /Build")
	}
	var postCiDuration float64
	start = time.Now()
	metrics.PostCiStartTime = start
	if len(ciCdRequest.CiRequest.PostCiSteps) > 0 {
		util.LogStage("running POST-CI steps")
		// sending build success as true always as post-ci triggers only if ci gets success
		scriptEnvs[util.ENV_VARIABLE_BUILD_SUCCESS] = "true"
		// run post artifact processing
		_, step, err = RunCiSteps(STEP_TYPE_POST, ciCdRequest.CiRequest.PostCiSteps, refStageMap, scriptEnvs, preeCiStageOutVariable)
		postCiDuration = time.Since(start).Seconds()
		if err != nil {
			return sendFailureNotification(string(PostCi)+step.Name, ciCdRequest.CiRequest, "", "", metrics, artifactUploaded, err)
		}
	}
	metrics.PostCiDuration = postCiDuration
	var digest string

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
				return sendFailureNotification(string(Push), ciCdRequest.CiRequest, digest, dest, metrics, artifactUploaded, err)
			}
			digest, err = helper.ExtractDigestUsingPull(dest)
		}
	}

	if err != nil {
		return artifactUploaded, err
	}
	log.Println(util.DEVTRON, " /docker-push")

	log.Println(util.DEVTRON, " artifact-upload")

	artifactUploaded, err = helper.ZipAndUpload(ciCdRequest.CiRequest.BlobStorageConfigured, ciCdRequest.CiRequest.BlobStorageS3Config, ciCdRequest.CiRequest.CiArtifactFileName, ciCdRequest.CiRequest.CloudProvider, ciCdRequest.CiRequest.AzureBlobConfig, ciCdRequest.CiRequest.GcpBlobConfig)

	if err != nil {
		return artifactUploaded, err
	}
	//else {
	//	artifactUploaded = true
	//}
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
			return sendFailureNotification(string(Scan), ciCdRequest.CiRequest, digest, dest, metrics, artifactUploaded, err)

		}
		log.Println(util.DEVTRON, " /image-scanner")
	}

	log.Println(util.DEVTRON, " event")
	metrics.TotalDuration = time.Since(metrics.TotalStartTime).Seconds()

	err = helper.SendEvents(ciCdRequest.CiRequest, digest, dest, metrics, artifactUploaded, "")
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

func getPostCiStepToRunOnCiFail(postCiSteps []*helper.StepObject) []*helper.StepObject {
	var postCiStepsToTriggerOnCiFail []*helper.StepObject
	if len(postCiSteps) > 0 {
		for _, postCiStep := range postCiSteps {
			if postCiStep.TriggerIfParentStageFail {
				postCiStepsToTriggerOnCiFail = append(postCiStepsToTriggerOnCiFail, postCiStep)
			}
		}
	}
	return postCiStepsToTriggerOnCiFail
}

func makeDockerfile(config *helper.DockerBuildConfig, checkoutPath string) error {
	dockerfileContent := config.DockerfileContent
	dockerfilePath := filepath.Join(util.WORKINGDIR, checkoutPath, "./Dockerfile")
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

func uploadLogs(event helper.CiCdTriggerEvent, exitCode *int) {
	var storageModuleConfigured bool
	var blobStorageLogKey string
	var cloudProvider blob_storage.BlobStorageType
	var blobStorageS3Config *blob_storage.BlobStorageS3Config
	var azureBlobConfig *blob_storage.AzureBlobConfig
	var gcpBlobConfig *blob_storage.GcpBlobConfig
	var inAppLoggingEnabled bool

	if event.Type == util.CIEVENT && event.CiRequest.BlobStorageConfigured {
		storageModuleConfigured = true
		blobStorageLogKey = event.CiRequest.BlobStorageLogsKey
		cloudProvider = event.CiRequest.CloudProvider
		blobStorageS3Config = event.CiRequest.BlobStorageS3Config
		azureBlobConfig = event.CiRequest.AzureBlobConfig
		gcpBlobConfig = event.CiRequest.GcpBlobConfig
		inAppLoggingEnabled = event.CiRequest.InAppLoggingEnabled

	} else if event.Type == util.CDSTAGE && event.CdRequest.BlobStorageConfigured {
		storageModuleConfigured = true
		blobStorageLogKey = event.CdRequest.BlobStorageLogsKey
		cloudProvider = event.CdRequest.CloudProvider
		blobStorageS3Config = event.CdRequest.BlobStorageS3Config
		azureBlobConfig = event.CdRequest.AzureBlobConfig
		gcpBlobConfig = event.CdRequest.GcpBlobConfig
		inAppLoggingEnabled = event.CdRequest.InAppLoggingEnabled
	}

	if r := recover(); r != nil {
		fmt.Println(r, string(debug.Stack()))
		*exitCode = 1
	}
	log.Println(util.DEVTRON, " blob storage configured ", storageModuleConfigured)
	log.Println(util.DEVTRON, " in app logging enabled ", inAppLoggingEnabled)
	if inAppLoggingEnabled {
		helper.UploadLogs(storageModuleConfigured, blobStorageLogKey, cloudProvider, blobStorageS3Config, azureBlobConfig, gcpBlobConfig)
	} else {
		log.Println(util.DEVTRON, "not uploading logs from app")
	}
}
