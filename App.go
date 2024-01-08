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
	blob_storage "github.com/devtron-labs/common-lib/blob-storage"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
)

var handleOnce sync.Once

func handleCleanup(ciCdRequest helper.CiCdTriggerEvent, exitCode *int, source string) {
	handleOnce.Do(func() {
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go cleanUpBuildxK8sDriver(ciCdRequest, wg)
		log.Println(util.DEVTRON, " CI-Runner cleanup executed with exit Code", *exitCode, source)
		uploadLogs(ciCdRequest, exitCode)
		wg.Wait()
		log.Println(util.DEVTRON, " Exiting with exit code ", *exitCode)
		os.Exit(*exitCode)
	})
}

func main() {
	//args := `{"type":"CI","ciRequest":{"DockerBuildTargetPlatform":"linux/arm64", "workflowNamePrefix":"16-ci-25-w5x1-70","pipelineName":"ci-25-w5x1","pipelineId":70,"dockerImageTag":"da3ba326-70-17","dockerRegistryId":"devtron-quay","dockerRegistryType":"other","dockerRegistryURL":"https://quay.io/devtron","dockerConnection":"secure","dockerCert":"","dockerBuildArgs":"{}","dockerRepository":"test","dockerfileLocation":"Dockerfile","dockerUsername":"devtron+devtest","dockerPassword":"5WEDXDJMP6RV1CG1KKFJQL3MQOLC64JKM6K684WPEBKVWKOZ4LSMBHEHJU1HBGXK","awsRegion":"","accessKey":"","secretKey":"","ciCacheLocation":"","ciCacheRegion":"","ciCacheFileName":"ci-25-w5x1-70.tar.gz","ciProjectDetails":[{"gitRepository":"https://github.com/devtron-labs/getting-started-nodejs","materialName":"1-getting-started-nodejs","checkoutPath":"./","fetchSubmodules":false,"commitHash":"da3ba3254712965b5944a6271e71bff91fe51f20","gitTag":"","commitTime":"2022-04-12T20:26:08+05:30","type":"SOURCE_TYPE_BRANCH_FIXED","message":"Update README.md","author":"Prakarsh \u003c71125043+prakarsh-dt@users.noreply.github.com\u003e","gitOptions":{"userName":"","password":"","sshPrivateKey":"","accessToken":"","authMode":"ANONYMOUS"},"sourceType":"SOURCE_TYPE_BRANCH_FIXED","sourceValue":"master","WebhookData":{"Id":0,"EventActionType":"","Data":null}}],"containerResources":{"minCpu":"","maxCpu":"","minStorage":"","maxStorage":"","minEphStorage":"","maxEphStorage":"","minMem":"","maxMem":""},"activeDeadlineSeconds":3600,"ciImage":"quay.io/devtron/ci-runner:1290cf23-182-8015","namespace":"devtron-ci","workflowId":16,"triggeredBy":8,"cacheLimit":5000000000,"beforeDockerBuildScripts":null,"afterDockerBuildScripts":null,"ciArtifactLocation":"","invalidateCache":true,"scanEnabled":false,"cloudProvider":"AZURE","azureBlobConfig":{"enabled":true,"accountName":"devtrondemoblob","blobContainerCiLog":"","blobContainerCiCache":"cache","accountKey":"y1/K13YMp/v7uuvZNkKJ4dS3CyGc37bPIN9Hv8MVhog6OkG0joV05proQReMQIJQ8qXp0JVpj+mz+AStHNKR3Q=="},"minioEndpoint":"","defaultAddressPoolBaseCidr":"","defaultAddressPoolSize":0,"preCiSteps":[{"name":"Task 1","index":1,"stepType":"INLINE","executorType":"SHELL","refPluginId":0,"script":"echo $","inputVars":null,"exposedPorts":{"0":0},"outputVars":null,"triggerSkipConditions":null,"successFailureConditions":null,"dockerImage":"","command":"","args":null,"customScriptMountDestinationPath":{"sourcePath":"","destinationPath":""},"sourceCodeMountDestinationPath":{"sourcePath":"","destinationPath":""},"extraVolumeMounts":null,"artifactPaths":null}],"postCiSteps":null,"refPlugins":null},"cdRequest":null}`
	//args := `{"type":"DryRun","dryRunRequest":{"buildPackParams":{"builderId":"gcr.io/buildpacks/builder:v1"},"DockerBuildTargetPlatform":"", "workflowNamePrefix":"16-ci-25-w5x1-70","pipelineName":"ci-25-w5x1","pipelineId":70,"dockerImageTag":"da3ba326-70-17","dockerRegistryId":"devtron-quay","dockerRegistryType":"other","dockerRegistryURL":"https://quay.io/devtron","dockerConnection":"secure","dockerCert":"","dockerBuildArgs":"{}","dockerRepository":"test","dockerfileLocation":"Dockerfile","dockerUsername":"devtron+devtest","dockerPassword":"5WEDXDJMP6RV1CG1KKFJQL3MQOLC64JKM6K684WPEBKVWKOZ4LSMBHEHJU1HBGXK","awsRegion":"","accessKey":"","secretKey":"","ciCacheLocation":"","ciCacheRegion":"","ciCacheFileName":"ci-25-w5x1-70.tar.gz","ciProjectDetails":[{"gitRepository":"https://github.com/devtron-labs/sample-go-app","materialName":"1-getting-started-nodejs","checkoutPath":"./","fetchSubmodules":false,"commitHash":"8654623ec2bd9efd663935cb8332c8c765541837","gitTag":"","commitTime":"2022-04-12T20:26:08+05:30","type":"SOURCE_TYPE_BRANCH_FIXED","message":"Update README.md","author":"Prakarsh \u003c71125043+prakarsh-dt@users.noreply.github.com\u003e","gitOptions":{"userName":"","password":"","sshPrivateKey":"","accessToken":"","authMode":"ANONYMOUS"},"sourceType":"SOURCE_TYPE_BRANCH_FIXED","sourceValue":"master","WebhookData":{"Id":0,"EventActionType":"","Data":null}}],"containerResources":{"minCpu":"","maxCpu":"","minStorage":"","maxStorage":"","minEphStorage":"","maxEphStorage":"","minMem":"","maxMem":""},"activeDeadlineSeconds":3600,"ciImage":"quay.io/devtron/ci-runner:1290cf23-182-8015","namespace":"devtron-ci","workflowId":16,"triggeredBy":8,"cacheLimit":5000000000,"beforeDockerBuildScripts":null,"afterDockerBuildScripts":null,"ciArtifactLocation":"","invalidateCache":true,"scanEnabled":false,"cloudProvider":"AZURE","azureBlobConfig":{"enabled":true,"accountName":"devtrondemoblob","blobContainerCiLog":"","blobContainerCiCache":"cache","accountKey":"y1/K13YMp/v7uuvZNkKJ4dS3CyGc37bPIN9Hv8MVhog6OkG0joV05proQReMQIJQ8qXp0JVpj+mz+AStHNKR3Q=="},"minioEndpoint":"","defaultAddressPoolBaseCidr":"","defaultAddressPoolSize":0,"preCiSteps":[{"name":"Task 1","index":1,"stepType":"INLINE","executorType":"SHELL","refPluginId":0,"script":"echo $","inputVars":null,"exposedPorts":{"0":0},"outputVars":null,"triggerSkipConditions":null,"successFailureConditions":null,"dockerImage":"","command":"","args":null,"customScriptMountDestinationPath":{"sourcePath":"","destinationPath":""},"sourceCodeMountDestinationPath":{"sourcePath":"","destinationPath":""},"extraVolumeMounts":null,"artifactPaths":null}],"postCiSteps":null,"refPlugins":null},"cdRequest":null}`
	//' {"workflowNamePrefix":"55-suraj-23-ci-suraj-test-pipeline-8","pipelineName":"suraj-23-ci-suraj-test-pipeline","pipelineId":8,"dockerImageTag":"a6b809c4be87c217feba4af15cf5ebc3cafe21e0","dockerRegistryURL":"686244538589.dkr.ecr.us-east-2.amazonaws.com","dockerRepository":"test/suraj-23","dockerfileLocation":"./notifier/Dockerfile","awsRegion":"us-east-2","ciCacheLocation":"ci-caching","ciCacheFileName":"suraj-23-ci-suraj-test-pipeline.tar.gz","ciProjectDetails":[{"gitRepository":"https://gitlab.com/devtron/notifier.git","materialName":"1-notifier","checkoutPath":"./notifier","commitHash":"d4df38bcd065004014d255c2203d592a91585955","commitTime":"0001-01-01T00:00:00Z","branch":"ci_with_argo","type":"SOURCE_TYPE_BRANCH_FIXED","message":"test-commit","gitOptions":{"userName":"Suraj24","password":"Devtron@1234","sshKey":"","accessToken":"","authMode":"USERNAME_PASSWORD"}},{"gitRepository":"https://gitlab.com/devtron/orchestrator.git","materialName":"2-orchestrator","checkoutPath":"./orch","commitHash":"","commitTime":"0001-01-01T00:00:00Z","branch":"ci_with_argo","type":"SOURCE_TYPE_BRANCH_FIXED","message":"","gitOptions":{"userName":"Suraj24","password":"Devtron@1234","sshKey":"","accessToken":"","authMode":""}}],"ciImage":"686244538589.dkr.ecr.us-east-2.amazonaws.com/cirunner:latest","namespace":"default"}'
	LoggingMode := "RunMode"
	if len(os.Args) > 1 {
		LoggingMode = os.Args[1]
	}

	if os.Getenv(util.InAppLogging) == "true" && LoggingMode == "PARENT_MODE" {
		log.Println(util.DEVTRON, " Starting in app logger... ")
		util.SpawnProcessWithLogging()
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
	signal.Notify(sigTerm, syscall.SIGTERM)

	go func() {
		var abortErrorCode = util.AbortErrorCode
		log.Println(util.DEVTRON, "SIGTERM listener started!")
		receivedSignal := <-sigTerm
		log.Println(util.DEVTRON, "signal received: ", receivedSignal)
		handleCleanup(*ciCdRequest, &abortErrorCode, util.Source_Signal)
	}()

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" || logLevel == "DEBUG" {
		log.Println(util.DEVTRON, " ci-cd request details -----> ", args)
	}

	defer handleCleanup(*ciCdRequest, &exitCode, util.Source_Defer)
	if ciCdRequest.Type == util.CIEVENT {
		HandleCIEvent(ciCdRequest, &exitCode)
	} else {
		HandleCDEvent(ciCdRequest, &exitCode)
	}
	return
}

func cleanUpBuildxK8sDriver(ciCdRequest helper.CiCdTriggerEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	if valid, eligibleBuildxK8sDriverNodes := helper.ValidBuildxK8sDriverOptions(ciCdRequest.CommonWorkflowRequest); valid {
		log.Println(util.DEVTRON, "starting buildx k8s driver clean up ,before terminating ci-runner")
		err := helper.CleanBuildxK8sDriver(eligibleBuildxK8sDriverNodes)
		if err != nil {
			log.Println(util.DEVTRON, "error in cleaning up buildx K8s driver, err : ", err)
		}
	}
}

func uploadLogs(event helper.CiCdTriggerEvent, exitCode *int) {
	var storageModuleConfigured bool
	var blobStorageLogKey string
	var cloudProvider blob_storage.BlobStorageType
	var blobStorageS3Config *blob_storage.BlobStorageS3Config
	var azureBlobConfig *blob_storage.AzureBlobConfig
	var gcpBlobConfig *blob_storage.GcpBlobConfig
	var inAppLoggingEnabled bool

	if event.Type == util.CIEVENT && event.CommonWorkflowRequest.BlobStorageConfigured {
		storageModuleConfigured = true
		blobStorageLogKey = event.CommonWorkflowRequest.BlobStorageLogsKey
		cloudProvider = event.CommonWorkflowRequest.CloudProvider
		blobStorageS3Config = event.CommonWorkflowRequest.BlobStorageS3Config
		azureBlobConfig = event.CommonWorkflowRequest.AzureBlobConfig
		gcpBlobConfig = event.CommonWorkflowRequest.GcpBlobConfig
		inAppLoggingEnabled = event.CommonWorkflowRequest.InAppLoggingEnabled

	} else if event.Type == util.CDSTAGE && event.CommonWorkflowRequest.BlobStorageConfigured {
		storageModuleConfigured = true
		blobStorageLogKey = event.CommonWorkflowRequest.BlobStorageLogsKey
		cloudProvider = event.CommonWorkflowRequest.CloudProvider
		blobStorageS3Config = event.CommonWorkflowRequest.BlobStorageS3Config
		azureBlobConfig = event.CommonWorkflowRequest.AzureBlobConfig
		gcpBlobConfig = event.CommonWorkflowRequest.GcpBlobConfig
		inAppLoggingEnabled = event.CommonWorkflowRequest.InAppLoggingEnabled
	}

	cloudHelperConfig := &util.CloudHelperBaseConfig{
		StorageModuleConfigured: storageModuleConfigured,
		BlobStorageLogKey:       blobStorageLogKey,
		CloudProvider:           cloudProvider,
		UseExternalClusterBlob:  event.CommonWorkflowRequest.UseExternalClusterBlob,
		BlobStorageS3Config:     blobStorageS3Config,
		AzureBlobConfig:         azureBlobConfig,
		GcpBlobConfig:           gcpBlobConfig,
		BlobStorageObjectType:   util.BlobStorageObjectTypeLog,
	}
	if r := recover(); r != nil {
		fmt.Println(r, string(debug.Stack()))
		*exitCode = 1
	}
	log.Println(util.DEVTRON, " blob storage configured ", storageModuleConfigured)
	log.Println(util.DEVTRON, " in app logging enabled ", inAppLoggingEnabled)
	if inAppLoggingEnabled {
		helper.UploadLogs(cloudHelperConfig)
	} else {
		log.Println(util.DEVTRON, "not uploading logs from app")
	}
}
