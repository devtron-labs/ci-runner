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
	//args := `{"type":"CI","commonWorkflowRequest":{"workflowNamePrefix":"25-ci-1-5b35-1","pipelineName":"ci-1-5b35","pipelineId":1,"dockerImageTag":"cf50e450-1-25","dockerRegistryId":"ashish-container","dockerRegistryType":"docker-hub","dockerRegistryURL":"docker.io","dockerConnection":"","dockerCert":"","dockerRepository":"devtronashish/devtron","checkoutPath":"","dockerUsername":"devtronashish","dockerPassword":"imfinnaWin@0210","awsRegion":"","accessKey":"","secretKey":"","ciCacheLocation":"","ciCacheRegion":"","ciCacheFileName":"ci-1-5b35-1.tar.gz","ciProjectDetails":[{"gitRepository":"https://github.com/deepak-devtron/sample-html","materialName":"1-sample-html","checkoutPath":"./","fetchSubmodules":false,"commitHash":"cf50e450cbd51288bbec97d526d6a04cbe5550df","gitTag":"","commitTime":"2023-08-17T08:45:28Z","type":"SOURCE_TYPE_BRANCH_FIXED","message":"Update app1.html\n\nfor testing purpose","author":"Deepak Panwar \u003c97603455+deepak-devtron@users.noreply.github.com\u003e","gitOptions":{"userName":"","password":"","sshPrivateKey":"","accessToken":"","authMode":"ANONYMOUS"},"sourceType":"SOURCE_TYPE_BRANCH_FIXED","sourceValue":"main","WebhookData":{"id":0,"eventActionType":"","data":null}}],"containerResources":{"minCpu":"","maxCpu":"","minStorage":"","maxStorage":"","minEphStorage":"","maxEphStorage":"","minMem":"","maxMem":""},"activeDeadlineSeconds":14400,"ciImage":"quay.io/devtron/test:18bf7e8d-333-4800","namespace":"devtron-ci","workflowId":25,"triggeredBy":2,"cacheLimit":5000000000,"beforeDockerBuildScripts":null,"afterDockerBuildScripts":null,"ciArtifactLocation":"s3://devtron-staging-log/arsenal-v1/ci-artifacts/25/25.zip","ciArtifactBucket":"devtron-staging-log","ciArtifactFileName":"arsenal-v1/ci-artifacts/25/25.zip","ciArtifactRegion":"","scanEnabled":false,"cloudProvider":"S3","blobStorageConfigured":false,"blobStorageS3Config":{"accessKey":"","passkey":"","endpointUrl":"","isInSecure":false,"ciLogBucketName":"devtron-staging-log","ciLogRegion":"","ciLogBucketVersioning":true,"ciCacheBucketName":"","ciCacheRegion":"","ciCacheBucketVersioning":true,"ciArtifactBucketName":"devtron-staging-log","ciArtifactRegion":"","ciArtifactBucketVersioning":true},"azureBlobConfig":null,"gcpBlobConfig":null,"blobStorageLogsKey":"/25-ci-1-5b35-1","inAppLoggingEnabled":false,"defaultAddressPoolBaseCidr":"","defaultAddressPoolSize":0,"preCiSteps":null,"postCiSteps":[{"name":"Vulnerability Scanning","index":1,"stepType":"REF_PLUGIN","executorType":"PLUGIN","refPluginId":13,"inputVars":null,"exposedPorts":null,"outputVars":null,"triggerSkipConditions":null,"successFailureConditions":null,"dockerImage":"","command":"","args":null,"customScriptMount":null,"sourceCodeMount":null,"extraVolumeMounts":null,"artifactPaths":null,"triggerIfParentStageFail":false}],"refPlugins":[{"id":13,"steps":[{"name":"Step 1","index":1,"stepType":"INLINE","executorType":"SHELL","script":"#!/bin/sh\necho \"IMAGE SCAN\"\n\nperform_curl_request() {\n    local attempt=1\n    while [ \"$attempt\" -le \"$MAX_RETRIES\" ]; do\n        response=$(curl -s -w \"\\n%{http_code}\" -X POST $IMAGE_SCANNER_ENDPOINT/scanner/image -H \"Content-Type: application/json\" -d \"{\\\"image\\\": \\\"$DEST\\\", \\\"imageDigest\\\": \\\"$DIGEST\\\", \\\"pipelineId\\\" : $PIPELINE_ID, \\\"userId\\\": $TRIGGERED_BY, \\\"dockerRegistryId\\\": \\\"$DOCKER_REGISTRY_ID\\\" }\")\n        http_status=$(echo \"$response\" | tail -n1)\n        if [ \"$http_status\" = \"200\" ]; then\n            echo \"Vulnerability Scanning request successful.\"\n            return 0\n        else\n            echo \"Attempt $attempt: Vulnerability Scanning request failed with HTTP status code $http_status\"\n            echo \"Response Body: $response\"\n            attempt=$((attempt + 1))\n            sleep \"$RETRY_DELAY\"\n        fi\n    done\n    echo -e \"\\033[1m======== Maximum retries reached. Vulnerability Scanning request failed ========\"\n    exit 1\n}\nperform_curl_request","inputVars":[{"name":"DEST","format":"STRING","variableType":"REF_GLOBAL","referenceVariableName":"DEST"},{"name":"DIGEST","format":"STRING","variableType":"REF_GLOBAL","referenceVariableName":"DIGEST"},{"name":"PIPELINE_ID","format":"STRING","variableType":"REF_GLOBAL","referenceVariableName":"PIPELINE_ID"},{"name":"TRIGGERED_BY","format":"STRING","variableType":"REF_GLOBAL","referenceVariableName":"TRIGGERED_BY"},{"name":"DOCKER_REGISTRY_ID","format":"STRING","variableType":"REF_GLOBAL","referenceVariableName":"DOCKER_REGISTRY_ID"},{"name":"IMAGE_SCANNER_ENDPOINT","format":"STRING","variableType":"REF_GLOBAL","referenceVariableName":"IMAGE_SCANNER_ENDPOINT"}],"exposedPorts":null,"outputVars":null,"triggerSkipConditions":null,"successFailureConditions":null,"dockerImage":"","command":"","args":null,"customScriptMount":null,"sourceCodeMount":null,"extraVolumeMounts":null,"artifactPaths":null,"triggerIfParentStageFail":false}]}],"appName":"test-app","triggerByAuthor":"admin","ciBuildConfig":{"id":1,"buildContextGitMaterialId":1,"useRootBuildContext":true,"ciBuildType":"self-dockerfile-build","dockerBuildConfig":{"dockerfileRelativePath":"Dockerfile","dockerfileContent":"","buildContext":".","useBuildx":true,"buildxProvenanceMode":""},"buildPackConfig":null,"pipelineType":"CI_BUILD"},"ciBuildDockerMtuValue":-1,"ignoreDockerCachePush":false,"ignoreDockerCachePull":false,"cacheInvalidate":false,"IsPvcMounted":false,"extraEnvironmentVariables":{},"enableBuildContext":true,"appId":1,"environmentId":0,"orchestratorHost":"http://devtroncd-orchestrator-service-prod.devtroncd/webhook/msg/nats","orchestratorToken":"","isExtRun":false,"imageRetryCount":0,"imageRetryInterval":5,"workflowRunnerId":0,"cdPipelineId":0,"stageYaml":"","artifactLocation":"","ciArtifactDTO":{"id":0,"pipelineId":0,"image":"","imageDigest":"","materialInfo":"","dataSource":"","workflowId":null},"cdImage":"","stageType":"","cdCacheLocation":"","cdCacheRegion":"","workflowPrefixForLog":"","deploymentTriggerTime":"0001-01-01T00:00:00Z","workflowExecutor":"AWF","prePostDeploySteps":null,"ciArtifactLastFetch":"0001-01-01T00:00:00Z","ciPipelineType":"","useExternalClusterBlob":false,"registryDestinationImageMap":null,"registryCredentialMap":null,"pluginArtifactStage":"","pushImageBeforePostCI":false,"maxRetries":3,"retryDelay":5,"Type":"CI","Pipeline":null,"Env":null,"AppLabels":{},"Scope":{"appId":1,"envId":0,"clusterId":0}}}`
	LoggingMode := "RunMode"
	if len(os.Args) > 1 {
		LoggingMode = os.Args[1]
	}

	if os.Getenv(util.InAppLogging) == "true" && LoggingMode == "PARENT_MODE" {
		log.Println(util.DEVTRON, " Starting in app logger... ")
		util.SpawnProcessWithLogging()
	}

	args := os.Getenv(util.CiCdEventEnvKey)
	log.Println("args = ", args)
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
