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

package app

import (
	"context"
	"encoding/json"
	"fmt"
	cicxt "github.com/devtron-labs/ci-runner/executor/context"
	"github.com/devtron-labs/ci-runner/executor/stage"
	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
	blob_storage "github.com/devtron-labs/common-lib/blob-storage"
	"github.com/devtron-labs/common-lib/utils/workFlow"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
)

type CiCdProcessor struct {
	ciStage      *stage.CiStage
	cdStage      *stage.CdStage
	dockerHelper helper.DockerHelper
}

func NewCiCdProcessor(ciStage *stage.CiStage, cdStage *stage.CdStage, dockerHelper helper.DockerHelper) *CiCdProcessor {
	return &CiCdProcessor{
		ciStage:      ciStage,
		cdStage:      cdStage,
		dockerHelper: dockerHelper,
	}
}

var handleOnce sync.Once

func (impl *CiCdProcessor) HandleCleanup(ciCdRequest helper.CiCdTriggerEvent, exitCode *int, source string) {
	handleOnce.Do(func() {
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go impl.CleanUpBuildxK8sDriver(ciCdRequest, wg)
		log.Println(util.DEVTRON, " CI-Runner cleanup executed with exit Code", *exitCode, source)
		impl.UploadLogs(ciCdRequest, exitCode)
		wg.Wait()
		log.Println(util.DEVTRON, " Exiting with exit code ", *exitCode)
		os.Exit(*exitCode)
	})
}

func (impl *CiCdProcessor) ProcessEvent(args string) {
	impl.ProcessCiCdEvent(impl.getCiCdRequestFromArg(args))
	return
}

func (impl *CiCdProcessor) getCiCdRequestFromArg(args string) (*helper.CiCdTriggerEvent, error) {
	ciCdRequest := &helper.CiCdTriggerEvent{}
	err := json.Unmarshal([]byte(args), ciCdRequest)
	if ciCdRequest != nil && ciCdRequest.CommonWorkflowRequest != nil {
		ciCdRequest.CommonWorkflowRequest.IntermediateDockerRegistryUrl = ciCdRequest.CommonWorkflowRequest.DockerRegistryURL
	}
	return ciCdRequest, err
}

func (impl *CiCdProcessor) ProcessCiCdEvent(ciCdRequest *helper.CiCdTriggerEvent, ciCdRequestErr error) {
	exitCode := 0
	if ciCdRequestErr != nil {
		log.Println(ciCdRequestErr)
		exitCode = workFlow.DefaultErrorCode
		return
	}
	// Create a channel to receive the SIGTERM signal
	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, syscall.SIGTERM)

	go func() {
		var abortErrorCode = workFlow.AbortErrorCode
		log.Println(util.DEVTRON, "SIGTERM listener started!")
		receivedSignal := <-sigTerm
		log.Println(util.DEVTRON, "signal received: ", receivedSignal)
		impl.HandleCleanup(*ciCdRequest, &abortErrorCode, util.Source_Signal)
	}()

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" || logLevel == "DEBUG" {
		log.Println(util.DEVTRON, " ci-cd request details -----> ", ciCdRequest)
	}

	defer impl.HandleCleanup(*ciCdRequest, &exitCode, util.Source_Defer)
	if helper.IsCIOrJobTypeEvent(ciCdRequest.Type) {
		impl.ciStage.HandleCIEvent(ciCdRequest, &exitCode)
	} else {
		impl.cdStage.HandleCDEvent(ciCdRequest, &exitCode)
	}
	return
}

func (impl *CiCdProcessor) CleanUpBuildxK8sDriver(ciCdRequest helper.CiCdTriggerEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	ciContext := cicxt.BuildCiContext(context.Background(), ciCdRequest.CommonWorkflowRequest.EnableSecretMasking)
	if valid, eligibleBuildxK8sDriverNodes := helper.ValidBuildxK8sDriverOptions(ciCdRequest.CommonWorkflowRequest); valid {
		log.Println(util.DEVTRON, "starting buildx k8s driver clean up ,before terminating ci-runner")
		err := impl.dockerHelper.CleanBuildxK8sDriver(ciContext, eligibleBuildxK8sDriverNodes)
		if err != nil {
			log.Println(util.DEVTRON, "error in cleaning up buildx K8s driver, err : ", err)
		}
	}
}

func (impl *CiCdProcessor) UploadLogs(event helper.CiCdTriggerEvent, exitCode *int) {
	var storageModuleConfigured bool
	var blobStorageLogKey string
	var cloudProvider blob_storage.BlobStorageType
	var blobStorageS3Config *blob_storage.BlobStorageS3Config
	var azureBlobConfig *blob_storage.AzureBlobConfig
	var gcpBlobConfig *blob_storage.GcpBlobConfig
	var inAppLoggingEnabled bool
	if event.CommonWorkflowRequest.BlobStorageConfigured &&
		helper.IsEventTypeEligibleToUploadLogs(event.Type) {
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
