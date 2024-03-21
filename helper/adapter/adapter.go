package adapter

import (
	"github.com/devtron-labs/ci-runner/executor"
	"github.com/devtron-labs/ci-runner/helper"
)

func GetExternalEnvRequest(ciCdRequest helper.CommonWorkflowRequest) helper.ExtEnvRequest {
	extEnvRequest := helper.ExtEnvRequest{
		OrchestratorHost:  ciCdRequest.OrchestratorHost,
		OrchestratorToken: ciCdRequest.OrchestratorToken,
		IsExtRun:          ciCdRequest.IsExtRun,
	}
	return extEnvRequest
}

func GetImageScanningEvent(ciCdRequest helper.CommonWorkflowRequest) helper.ImageScanningEvent {
	event := helper.ImageScanningEvent{
		CiPipelineId:     ciCdRequest.PipelineId,
		CdPipelineId:     ciCdRequest.CdPipelineId,
		TriggerBy:        ciCdRequest.TriggeredBy,
		DockerRegistryId: ciCdRequest.DockerRegistryId,
		Image:            ciCdRequest.CiArtifactDTO.Image,
		Digest:           ciCdRequest.CiArtifactDTO.ImageDigest,
	}
	var stage helper.NotifyPipelineType
	if ciCdRequest.StageType == string(executor.STEP_TYPE_PRE) {
		stage = helper.PRE_CD
	} else if ciCdRequest.StageType == string(executor.STEP_TYPE_POST) {
		stage = helper.POST_CD
	}
	event.PipelineType = stage
	return event
}
