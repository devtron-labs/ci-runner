package adapter

import "github.com/devtron-labs/ci-runner/helper"

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
		TriggerBy:        ciCdRequest.TriggeredBy,
		DockerRegistryId: ciCdRequest.DockerRegistryId,
		Image:            ciCdRequest.CiArtifactDTO.Image,
		Digest:           ciCdRequest.CiArtifactDTO.ImageDigest,
	}
	return event
}
