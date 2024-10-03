package adaptor

import (
	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
)

func NewCiCompleteEvent(ciRequest *helper.CommonWorkflowRequest) *helper.CiCompleteEvent {
	event := &helper.CiCompleteEvent{
		CiProjectDetails:              ciRequest.CiProjectDetails,
		PipelineId:                    ciRequest.PipelineId,
		PipelineName:                  ciRequest.PipelineName,
		DataSource:                    util.ArtifactSourceType,
		WorkflowId:                    ciRequest.WorkflowId,
		TriggeredBy:                   ciRequest.TriggeredBy,
		AppName:                       ciRequest.AppName,
		PluginRegistryArtifactDetails: ciRequest.RegistryDestinationImageMap,
		PluginArtifactStage:           ciRequest.PluginArtifactStage,
		IsScanEnabled:                 ciRequest.ScanEnabled,
		MaterialType:                  util.ArtifactMaterialType,
	}
	return event
}

func NewCdCompleteEvent(cdRequest *helper.CommonWorkflowRequest) *helper.CdStageCompleteEvent {
	event := &helper.CdStageCompleteEvent{
		CiProjectDetails:              cdRequest.CiProjectDetails,
		CdPipelineId:                  cdRequest.CdPipelineId,
		WorkflowId:                    cdRequest.WorkflowId,
		WorkflowRunnerId:              cdRequest.WorkflowRunnerId,
		CiArtifactDTO:                 cdRequest.CiArtifactDTO,
		TriggeredBy:                   cdRequest.TriggeredBy,
		PluginRegistryArtifactDetails: cdRequest.RegistryDestinationImageMap,
		PluginArtifactStage:           cdRequest.PluginArtifactStage,
	}
	return event
}
