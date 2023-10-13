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

package helper

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	blob_storage "github.com/devtron-labs/common-lib/blob-storage"

	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/caarlos0/env"
	"github.com/devtron-labs/ci-runner/pubsub"
	"github.com/devtron-labs/ci-runner/util"
	pubsub1 "github.com/devtron-labs/common-lib/pubsub-lib"
	"github.com/go-resty/resty/v2"
)

type TestExecutorImageProperties struct {
	ImageName string `json:"imageName,omitempty"`
	Arg       string `json:"arg,omitempty"`
}

type CiBuildType string

const (
	SELF_DOCKERFILE_BUILD_TYPE    CiBuildType = "self-dockerfile-build"
	MANAGED_DOCKERFILE_BUILD_TYPE CiBuildType = "managed-dockerfile-build"
	BUILD_SKIP_BUILD_TYPE         CiBuildType = "skip-build"
	BUILDPACK_BUILD_TYPE          CiBuildType = "buildpack-build"
)

const CI_JOB string = "CI_JOB"

type CiBuildConfigBean struct {
	CiBuildType       CiBuildType        `json:"ciBuildType"`
	DockerBuildConfig *DockerBuildConfig `json:"dockerBuildConfig,omitempty"`
	BuildPackConfig   *BuildPackConfig   `json:"buildPackConfig"`
	PipelineType      string             `json:"pipelineType"`
}

type DockerBuildConfig struct {
	DockerfilePath         string              `json:"dockerfileRelativePath,omitempty" validate:"required"`
	DockerfileContent      string              `json:"DockerfileContent"`
	Args                   map[string]string   `json:"args,omitempty"`
	DockerBuildOptions     map[string]string   `json:"dockerBuildOptions"`
	TargetPlatform         string              `json:"targetPlatform,omitempty"`
	BuildContext           string              `json:"buildContext,omitempty"`
	UseBuildx              bool                `json:"useBuildx"`
	BuildxProvenanceMode   string              `json:"buildxProvenanceMode"`
	BuildxK8sDriverOptions []map[string]string `json:"buildxK8SDriverOptions"`
}

type BuildPackConfig struct {
	BuilderId       string            `json:"builderId"`
	Language        string            `json:"language"`
	LanguageVersion string            `json:"languageVersion"`
	BuildPacks      []string          `json:"buildPacks"`
	Args            map[string]string `json:"args"`
	ProjectPath     string            `json:"projectPath"`
}

type BuildpackVersionConfig struct {
	BuilderPrefix string `json:"builderPrefix"`
	Language      string `json:"language"`
	FileName      string `json:"fileName"`
	FileOverride  bool   `json:"fileOverride"`
	EntryRegex    string `json:"entryRegex"`
}

type CommonWorkflowRequest struct {
	WorkflowNamePrefix         string                            `json:"workflowNamePrefix"`
	PipelineName               string                            `json:"pipelineName"`
	PipelineId                 int                               `json:"pipelineId"`
	DockerImageTag             string                            `json:"dockerImageTag"`
	DockerRegistryId           string                            `json:"dockerRegistryId"`
	DockerRegistryType         string                            `json:"dockerRegistryType"`
	DockerRegistryURL          string                            `json:"dockerRegistryURL"`
	DockerConnection           string                            `json:"dockerConnection"`
	DockerCert                 string                            `json:"dockerCert"`
	DockerRepository           string                            `json:"dockerRepository"`
	CheckoutPath               string                            `json:"checkoutPath"`
	DockerUsername             string                            `json:"dockerUsername"`
	DockerPassword             string                            `json:"dockerPassword"`
	AwsRegion                  string                            `json:"awsRegion"`
	AccessKey                  string                            `json:"accessKey"`
	SecretKey                  string                            `json:"secretKey"`
	CiCacheLocation            string                            `json:"ciCacheLocation"`
	CiCacheRegion              string                            `json:"ciCacheRegion"`
	CiCacheFileName            string                            `json:"ciCacheFileName"`
	CiProjectDetails           []CiProjectDetails                `json:"ciProjectDetails"`
	ActiveDeadlineSeconds      int64                             `json:"activeDeadlineSeconds"`
	CiImage                    string                            `json:"ciImage"`
	Namespace                  string                            `json:"namespace"`
	WorkflowId                 int                               `json:"workflowId"`
	TriggeredBy                int                               `json:"triggeredBy"`
	CacheLimit                 int64                             `json:"cacheLimit"`
	BeforeDockerBuildScripts   []*Task                           `json:"beforeDockerBuildScripts"`
	AfterDockerBuildScripts    []*Task                           `json:"afterDockerBuildScripts"`
	CiArtifactLocation         string                            `json:"ciArtifactLocation"`
	CiArtifactBucket           string                            `json:"ciArtifactBucket"`
	CiArtifactFileName         string                            `json:"ciArtifactFileName"`
	CiArtifactRegion           string                            `json:"ciArtifactRegion"`
	ScanEnabled                bool                              `json:"scanEnabled"`
	CloudProvider              blob_storage.BlobStorageType      `json:"cloudProvider"`
	BlobStorageConfigured      bool                              `json:"blobStorageConfigured"`
	BlobStorageS3Config        *blob_storage.BlobStorageS3Config `json:"blobStorageS3Config"`
	AzureBlobConfig            *blob_storage.AzureBlobConfig     `json:"azureBlobConfig"`
	GcpBlobConfig              *blob_storage.GcpBlobConfig       `json:"gcpBlobConfig"`
	BlobStorageLogsKey         string                            `json:"blobStorageLogsKey"`
	InAppLoggingEnabled        bool                              `json:"inAppLoggingEnabled"`
	DefaultAddressPoolBaseCidr string                            `json:"defaultAddressPoolBaseCidr"`
	DefaultAddressPoolSize     int                               `json:"defaultAddressPoolSize"`
	PreCiSteps                 []*StepObject                     `json:"preCiSteps"`
	PostCiSteps                []*StepObject                     `json:"postCiSteps"`
	RefPlugins                 []*RefPluginObject                `json:"refPlugins"`
	AppName                    string                            `json:"appName"`
	TriggerByAuthor            string                            `json:"triggerByAuthor"`
	CiBuildConfig              *CiBuildConfigBean                `json:"ciBuildConfig"`
	CiBuildDockerMtuValue      int                               `json:"ciBuildDockerMtuValue"`
	IgnoreDockerCachePush      bool                              `json:"ignoreDockerCachePush"`
	IgnoreDockerCachePull      bool                              `json:"ignoreDockerCachePull"`
	CacheInvalidate            bool                              `json:"cacheInvalidate"`
	IsPvcMounted               bool                              `json:"IsPvcMounted"`
	ExtraEnvironmentVariables  map[string]string                 `json:"extraEnvironmentVariables"`
	EnableBuildContext         bool                              `json:"enableBuildContext"`
	AppId                      int                               `json:"appId"`
	EnvironmentId              int                               `json:"environmentId"`
	OrchestratorHost           string                            `json:"orchestratorHost"`
	OrchestratorToken          string                            `json:"orchestratorToken"`
	IsExtRun                   bool                              `json:"isExtRun"`
	ImageRetryCount            int                               `json:"imageRetryCount"`
	ImageRetryInterval         int                               `json:"imageRetryInterval"`
	// Data from CD Workflow service
	WorkflowRunnerId         int           `json:"workflowRunnerId"`
	CdPipelineId             int           `json:"cdPipelineId"`
	StageYaml                string        `json:"stageYaml"`
	ArtifactLocation         string        `json:"artifactLocation"`
	CiArtifactDTO            CiArtifactDTO `json:"ciArtifactDTO"`
	CdImage                  string        `json:"cdImage"`
	StageType                string        `json:"stageType"`
	CdCacheLocation          string        `json:"cdCacheLocation"`
	CdCacheRegion            string        `json:"cdCacheRegion"`
	WorkflowPrefixForLog     string        `json:"workflowPrefixForLog"`
	DeploymentTriggeredBy    string        `json:"deploymentTriggeredBy,omitempty"`
	DeploymentTriggerTime    time.Time     `json:"deploymentTriggerTime,omitempty"`
	DeploymentReleaseCounter int           `json:"deploymentReleaseCounter,omitempty"`
	PrePostDeploySteps       []*StepObject `json:"prePostDeploySteps"`
	TaskYaml                 *TaskYaml     `json:"-"`
	IsDryRun                 bool          `json:"isDryRun"`
	CiArtifactLastFetch      time.Time     `json:"ciArtifactLastFetch"`
}

type CiRequest struct {
	CiProjectDetails            []CiProjectDetails                `json:"ciProjectDetails"`
	DockerImageTag              string                            `json:"dockerImageTag"`
	DockerRegistryId            string                            `json:"dockerRegistryId"`
	DockerRegistryType          string                            `json:"dockerRegistryType"`
	DockerRegistryURL           string                            `json:"dockerRegistryURL"`
	DockerConnection            string                            `json:"dockerConnection"`
	DockerCert                  string                            `json:"dockerCert"`
	DockerRepository            string                            `json:"dockerRepository"`
	CheckoutPath                string                            `json:"checkoutPath"`
	DockerUsername              string                            `json:"dockerUsername"`
	DockerPassword              string                            `json:"dockerPassword"`
	AwsRegion                   string                            `json:"awsRegion"`
	AccessKey                   string                            `json:"accessKey"`
	SecretKey                   string                            `json:"secretKey"`
	CiCacheLocation             string                            `json:"ciCacheLocation"`
	CiArtifactLocation          string                            `json:"ciArtifactLocation"` // s3 bucket+ path
	CiArtifactBucket            string                            `json:"ciArtifactBucket"`
	CiArtifactFileName          string                            `json:"ciArtifactFileName"`
	CiArtifactRegion            string                            `json:"ciArtifactRegion"`
	BlobStorageS3Config         *blob_storage.BlobStorageS3Config `json:"blobStorageS3Config"`
	CiCacheRegion               string                            `json:"ciCacheRegion"`
	CiCacheFileName             string                            `json:"ciCacheFileName"`
	PipelineId                  int                               `json:"pipelineId"`
	PipelineName                string                            `json:"pipelineName"`
	WorkflowId                  int                               `json:"workflowId"`
	TriggeredBy                 int                               `json:"triggeredBy"`
	CacheLimit                  int64                             `json:"cacheLimit"`
	BeforeDockerBuild           []*Task                           `json:"beforeDockerBuildScripts"`
	AfterDockerBuild            []*Task                           `json:"afterDockerBuildScripts"`
	CiYamlLocation              string                            `json:"CiYamlLocations"`
	TaskYaml                    *TaskYaml                         `json:"-"`
	TestExecutorImageProperties *TestExecutorImageProperties      `json:"testExecutorImageProperties"`
	BlobStorageConfigured       bool                              `json:"blobStorageConfigured"`
	IgnoreDockerCachePull       bool                              `json:"ignoreDockerCachePull"`
	IgnoreDockerCachePush       bool                              `json:"ignoreDockerCachePush"`
	ScanEnabled                 bool                              `json:"scanEnabled"`
	CloudProvider               blob_storage.BlobStorageType      `json:"cloudProvider"`
	AzureBlobConfig             *blob_storage.AzureBlobConfig     `json:"azureBlobConfig"`
	GcpBlobConfig               *blob_storage.GcpBlobConfig       `json:"gcpBlobConfig"`
	BlobStorageLogsKey          string                            `json:"blobStorageLogsKey"`
	InAppLoggingEnabled         bool                              `json:"inAppLoggingEnabled"`
	MinioEndpoint               string                            `json:"minioEndpoint"`
	DefaultAddressPoolBaseCidr  string                            `json:"defaultAddressPoolBaseCidr"`
	DefaultAddressPoolSize      int                               `json:"defaultAddressPoolSize"`
	PreCiSteps                  []*StepObject                     `json:"preCiSteps"`
	PostCiSteps                 []*StepObject                     `json:"postCiSteps"`
	RefPlugins                  []*RefPluginObject                `json:"refPlugins"`
	AppName                     string                            `json:"appName"`
	TriggerByAuthor             string                            `json:"triggerByAuthor"`
	CiBuildConfig               *CiBuildConfigBean                `json:"ciBuildConfig"`
	CiBuildDockerMtuValue       int                               `json:"ciBuildDockerMtuValue"`
	CacheInvalidate             bool                              `json:"cacheInvalidate"`
	IsPvcMounted                bool                              `json:"IsPvcMounted"`
	ExtraEnvironmentVariables   map[string]string                 `json:"extraEnvironmentVariables"`
	EnableBuildContext          bool                              `json:"enableBuildContext"`
	IsExtRun                    bool                              `json:"isExtRun"`
	OrchestratorHost            string                            `json:"orchestratorHost"`
	OrchestratorToken           string                            `json:"orchestratorToken"`
	ImageRetryCount             int                               `json:"imageRetryCount"`
	ImageRetryInterval          int                               `json:"imageRetryInterval"`
}

type CdRequest struct {
	WorkflowId                 int                               `json:"workflowId"`
	WorkflowRunnerId           int                               `json:"workflowRunnerId"`
	CdPipelineId               int                               `json:"cdPipelineId"`
	TriggeredBy                int32                             `json:"triggeredBy"`
	StageYaml                  string                            `json:"stageYaml"`
	ArtifactLocation           string                            `json:"artifactLocation"`
	ArtifactBucket             string                            `json:"ciArtifactBucket"`
	ArtifactFileName           string                            `json:"ciArtifactFileName"`
	ArtifactRegion             string                            `json:"ciArtifactRegion"`
	BlobStorageS3Config        *blob_storage.BlobStorageS3Config `json:"blobStorageS3Config"`
	TaskYaml                   *TaskYaml                         `json:"-"`
	CiProjectDetails           []CiProjectDetails                `json:"ciProjectDetails"`
	CiArtifactDTO              CiArtifactDTO                     `json:"ciArtifactDTO"`
	DockerUsername             string                            `json:"dockerUsername"`
	DockerPassword             string                            `json:"dockerPassword"`
	AwsRegion                  string                            `json:"awsRegion"`
	AccessKey                  string                            `json:"accessKey"`
	SecretKey                  string                            `json:"secretKey"`
	DockerRegistryURL          string                            `json:"dockerRegistryUrl"`
	DockerRegistryType         string                            `json:"dockerRegistryType"`
	DockerConnection           string                            `json:"dockerConnection"`
	DockerCert                 string                            `json:"dockerCert"`
	OrchestratorHost           string                            `json:"orchestratorHost"`
	OrchestratorToken          string                            `json:"orchestratorToken"`
	IsExtRun                   bool                              `json:"isExtRun"`
	ExtraEnvironmentVariables  map[string]string                 `json:"extraEnvironmentVariables"`
	CloudProvider              blob_storage.BlobStorageType      `json:"cloudProvider"`
	BlobStorageConfigured      bool                              `json:"blobStorageConfigured"`
	AzureBlobConfig            *blob_storage.AzureBlobConfig     `json:"azureBlobConfig"`
	GcpBlobConfig              *blob_storage.GcpBlobConfig       `json:"gcpBlobConfig"`
	BlobStorageLogsKey         string                            `json:"blobStorageLogsKey"`
	InAppLoggingEnabled        bool                              `json:"inAppLoggingEnabled"`
	MinioEndpoint              string                            `json:"minioEndpoint"`
	DefaultAddressPoolBaseCidr string                            `json:"defaultAddressPoolBaseCidr"`
	DefaultAddressPoolSize     int                               `json:"defaultAddressPoolSize"`
	DeploymentTriggeredBy      string                            `json:"deploymentTriggeredBy"`
	DeploymentTriggerTime      time.Time                         `json:"deploymentTriggerTime"`
	CiRunnerDockerMtuValue     int                               `json:"ciRunnerDockerMtuValue"`
	DeploymentReleaseCounter   int                               `json:"deploymentReleaseCounter,omitempty"`
	IsDryRun                   bool                              `json:"isDryRun"`
	PrePostDeploySteps         []*StepObject                     `json:"prePostDeploySteps"`
	RefPlugins                 []*RefPluginObject                `json:"refPlugins"`
	StageType                  string                            `json:"stageType"`
}

type CiCdTriggerEvent struct {
	Type                  string                 `json:"type"`
	CommonWorkflowRequest *CommonWorkflowRequest `json:"commonWorkflowRequest"`
}

type ExtEnvRequest struct {
	OrchestratorHost  string `json:"orchestratorHost"`
	OrchestratorToken string `json:"orchestratorToken"`
	IsExtRun          bool   `json:"isExtRun"`
}

type CiArtifactDTO struct {
	Id           int    `json:"id"`
	PipelineId   int    `json:"pipelineId"` //id of the ci pipeline from which this webhook was triggered
	Image        string `json:"image"`
	ImageDigest  string `json:"imageDigest"`
	MaterialInfo string `json:"materialInfo"` //git material metadata json array string
	DataSource   string `json:"dataSource"`
	WorkflowId   *int   `json:"workflowId"`
}

type ImageDetailsFromCR struct {
	ImageDetails []types.ImageDetail `json:"imageDetails"`
	Region       string              `json:"region"`
}

type CiCompleteEvent struct {
	CiProjectDetails   []CiProjectDetails  `json:"ciProjectDetails"`
	DockerImage        string              `json:"dockerImage"`
	Digest             string              `json:"digest"`
	PipelineId         int                 `json:"pipelineId"`
	DataSource         string              `json:"dataSource"`
	PipelineName       string              `json:"pipelineName"`
	WorkflowId         int                 `json:"workflowId"`
	TriggeredBy        int                 `json:"triggeredBy"`
	MaterialType       string              `json:"materialType"`
	Metrics            CIMetrics           `json:"metrics"`
	AppName            string              `json:"appName"`
	IsArtifactUploaded bool                `json:"isArtifactUploaded"`
	FailureReason      string              `json:"failureReason"`
	ImageDetailsFromCR *ImageDetailsFromCR `json:"imageDetailsFromCR"`
}

type CdStageCompleteEvent struct {
	CiProjectDetails []CiProjectDetails `json:"ciProjectDetails"`
	WorkflowId       int                `json:"workflowId"`
	WorkflowRunnerId int                `json:"workflowRunnerId"`
	CdPipelineId     int                `json:"cdPipelineId"`
	TriggeredBy      int                `json:"triggeredBy"`
	StageYaml        string             `json:"stageYaml"`
	ArtifactLocation string             `json:"artifactLocation"`
	TaskYaml         *TaskYaml          `json:"-"`
	PipelineName     string             `json:"pipelineName"`
	CiArtifactDTO    CiArtifactDTO      `json:"ciArtifactDTO"`
}

type CiProjectDetails struct {
	GitRepository   string      `json:"gitRepository"`
	FetchSubmodules bool        `json:"fetchSubmodules"`
	MaterialName    string      `json:"materialName"`
	CheckoutPath    string      `json:"checkoutPath"`
	CommitHash      string      `json:"commitHash"`
	GitTag          string      `json:"gitTag"`
	CommitTime      time.Time   `json:"commitTime"`
	SourceType      SourceType  `json:"sourceType"`
	SourceValue     string      `json:"sourceValue"`
	Type            string      `json:"type"`
	Message         string      `json:"message"`
	Author          string      `json:"author"`
	GitOptions      GitOptions  `json:"gitOptions"`
	WebhookData     WebhookData `json:"webhookData"`
}

type PublishRequest struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

type CIMetrics struct {
	CacheDownDuration  float64   `json:"cacheDownDuration"`
	PreCiDuration      float64   `json:"preCiDuration"`
	BuildDuration      float64   `json:"buildDuration"`
	PostCiDuration     float64   `json:"postCiDuration"`
	CacheUpDuration    float64   `json:"cacheUpDuration"`
	TotalDuration      float64   `json:"totalDuration"`
	CacheDownStartTime time.Time `json:"cacheDownStartTime"`
	PreCiStartTime     time.Time `json:"preCiStart"`
	BuildStartTime     time.Time `json:"buildStartTime"`
	PostCiStartTime    time.Time `json:"postCiStartTime"`
	CacheUpStartTime   time.Time `json:"cacheUpStartTime"`
	TotalStartTime     time.Time `json:"totalStartTime"`
}

func SendCDEvent(cdRequest *CommonWorkflowRequest) error {

	event := CdStageCompleteEvent{
		CiProjectDetails: cdRequest.CiProjectDetails,
		CdPipelineId:     cdRequest.CdPipelineId,
		WorkflowId:       cdRequest.WorkflowId,
		WorkflowRunnerId: cdRequest.WorkflowRunnerId,
		CiArtifactDTO:    cdRequest.CiArtifactDTO,
		TriggeredBy:      cdRequest.TriggeredBy,
	}
	err := SendCdCompleteEvent(cdRequest, event)
	if err != nil {
		log.Println(util.DEVTRON, "err", err)
		return err
	}
	return nil
}

func SendEvents(ciRequest *CommonWorkflowRequest, digest string, image string, metrics CIMetrics, artifactUploaded bool, failureReason string, imageDetailsFromCR *ImageDetailsFromCR) error {

	event := CiCompleteEvent{
		CiProjectDetails:   ciRequest.CiProjectDetails,
		DockerImage:        image,
		Digest:             digest,
		PipelineId:         ciRequest.PipelineId,
		PipelineName:       ciRequest.PipelineName,
		DataSource:         "CI-RUNNER",
		WorkflowId:         ciRequest.WorkflowId,
		TriggeredBy:        ciRequest.TriggeredBy,
		MaterialType:       "git",
		Metrics:            metrics,
		AppName:            ciRequest.AppName,
		IsArtifactUploaded: artifactUploaded,
		FailureReason:      failureReason,
		ImageDetailsFromCR: imageDetailsFromCR,
	}

	err := SendCiCompleteEvent(ciRequest, event)
	if err != nil {
		log.Println(util.DEVTRON, "err", err)
		return err
	}
	log.Println(util.DEVTRON, " housekeeping done. exiting now")
	return nil
}

func SendCiCompleteEvent(ciRequest *CommonWorkflowRequest, event CiCompleteEvent) error {
	jsonBody, err := json.Marshal(event)
	if err != nil {
		log.Println(util.DEVTRON, "err", err)
		return err
	}
	extEnvRequest := ExtEnvRequest{
		OrchestratorHost:  ciRequest.OrchestratorHost,
		OrchestratorToken: ciRequest.OrchestratorToken,
		IsExtRun:          ciRequest.IsExtRun,
	}
	err = PublishEvent(jsonBody, pubsub1.CI_COMPLETE_TOPIC, &extEnvRequest)
	log.Println(util.DEVTRON, "ci complete event notification done")
	return err
}

func SendCdCompleteEvent(cdRequest *CommonWorkflowRequest, event CdStageCompleteEvent) error {
	jsonBody, err := json.Marshal(event)
	if err != nil {
		log.Println(util.DEVTRON, "err", err)
		return err
	}
	extEnvRequest := ExtEnvRequest{
		OrchestratorHost:  cdRequest.OrchestratorHost,
		OrchestratorToken: cdRequest.OrchestratorToken,
		IsExtRun:          cdRequest.IsExtRun,
	}
	err = PublishCDEvent(jsonBody, pubsub1.CD_STAGE_COMPLETE_TOPIC, &extEnvRequest)
	log.Println(util.DEVTRON, "cd stage complete event notification done")
	return err
}

func PublishCDEvent(jsonBody []byte, topic string, cdRequest *ExtEnvRequest) error {
	if cdRequest.IsExtRun {
		return PublishEventsOnRest(jsonBody, topic, cdRequest)
	}
	return pubsub.PublishEventsOnNats(jsonBody, topic)
}

func PublishEvent(jsonBody []byte, topic string, ciRequest *ExtEnvRequest) error {
	if ciRequest.IsExtRun {
		return PublishEventsOnRest(jsonBody, topic, ciRequest)
	}
	return pubsub.PublishEventsOnNats(jsonBody, topic)
}

func PublishEventsOnRest(jsonBody []byte, topic string, cdRequest *ExtEnvRequest) error {
	publishRequest := &PublishRequest{
		Topic:   topic,
		Payload: jsonBody,
	}
	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	resp, err := client.SetRetryCount(4).R().
		SetHeader("Content-Type", "application/json").
		SetBody(publishRequest).
		SetAuthToken(cdRequest.OrchestratorToken).
		//SetResult().    // or SetResult(AuthSuccess{}).
		Post(cdRequest.OrchestratorHost)
	if err != nil {
		log.Println(util.DEVTRON, "err in publishing over rest", err)
		return err
	}
	log.Println(util.DEVTRON, "res ", string(resp.Body()))
	return nil
}

func SendEventToClairUtility(event *ScanEvent) error {
	jsonBody, err := json.Marshal(event)
	if err != nil {
		log.Println(util.DEVTRON, "err", err)
		return err
	}

	cfg := &pubsub.PubSubConfig{}
	err = env.Parse(cfg)
	if err != nil {
		return err
	}

	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(jsonBody).
		Post(fmt.Sprintf("%s/%s", cfg.ImageScannerEndpoint, "scanner/image"))
	if err != nil {
		log.Println(util.DEVTRON, "err in image scanner app over rest", err)
		return err
	}
	log.Println(util.DEVTRON, resp.StatusCode())
	log.Println(util.DEVTRON, resp)
	return nil
}

type ScanEvent struct {
	Image            string `json:"image"`
	ImageDigest      string `json:"imageDigest"`
	AppId            int    `json:"appId"`
	EnvId            int    `json:"envId"`
	PipelineId       int    `json:"pipelineId"`
	CiArtifactId     int    `json:"ciArtifactId"`
	UserId           int    `json:"userId"`
	AccessKey        string `json:"accessKey"`
	SecretKey        string `json:"secretKey"`
	Token            string `json:"token"`
	AwsRegion        string `json:"awsRegion"`
	DockerRegistryId string `json:"dockerRegistryId"`
}

func (dockerBuildConfig *DockerBuildConfig) GetProvenanceFlag() string {
	// if provenance mode is provided, set provenance, else set to false
	if dockerBuildConfig.BuildxProvenanceMode != "" {
		return fmt.Sprintf("--provenance=mode=%s ", dockerBuildConfig.BuildxProvenanceMode)
	}

	// --provinance is set to true by default by docker. this will add some build related data in generated build manifest.it also adds some
	// unknown:unknown key:value pair which may not be compatible by some container registries.

	// with buildx k8s driver , --provinenance=true is causing issue when push manifest to quay registry, so setting it to false
	// above issue is being tracked in https://github.com/moby/buildkit/issues/3222
	return "--provenance=false"
}
func (dockerBuildConfig *DockerBuildConfig) CheckForBuildX() bool {
	return dockerBuildConfig.TargetPlatform != "" || dockerBuildConfig.UseBuildx
}

func (dockerBuildConfig *DockerBuildConfig) CheckForBuildXK8sDriver() (bool, []map[string]string) {
	buildxEnabled := dockerBuildConfig.CheckForBuildX()
	eligibleK8sNodes := dockerBuildConfig.GetEligibleK8sDriverNodes()
	useBuildxK8sDriver := buildxEnabled && len(eligibleK8sNodes) > 0
	return useBuildxK8sDriver, eligibleK8sNodes
}

func (dockerBuildConfig *DockerBuildConfig) GetEligibleK8sDriverNodes() []map[string]string {
	if dockerBuildConfig.TargetPlatform == "" {
		return findDefaultBuildxNodes(dockerBuildConfig.BuildxK8sDriverOptions)
	}
	return filterBuilderNodes(dockerBuildConfig.BuildxK8sDriverOptions, dockerBuildConfig.TargetPlatform)
}

func filterBuilderNodes(builderNodes []map[string]string, targetPlatformStr string) []map[string]string {
	filteredBuilderNodes := make([]map[string]string, 0)
	requiredTargetPlatformSet := make(map[string]bool)   //user requested platforms for build
	canBeBuildTargetPlatformSet := make(map[string]bool) //platforms that can be built with provided k8s Driver Nodes
	for _, platform := range strings.Split(targetPlatformStr, ",") {
		requiredTargetPlatformSet[platform] = true
	}
	for _, builderNode := range builderNodes {
		platformStr := builderNode["platform"]
		for _, platform := range strings.Split(platformStr, ",") {
			if requiredTargetPlatformSet[platform] {
				canBeBuildTargetPlatformSet[platform] = true
				filteredBuilderNodes = append(filteredBuilderNodes, builderNode) //filtering out required k8s Driver nodes only
			}
		}
	}
	if len(requiredTargetPlatformSet) != len(canBeBuildTargetPlatformSet) {
		fmt.Println(util.DEVTRON, " Docker k8s driver nodes required to build for these platforms ", targetPlatformStr, " are not present, so not using docker k8s driver for this build ")
		return nil
	}
	return filteredBuilderNodes
}

func findDefaultBuildxNodes(builderNodes []map[string]string) []map[string]string {

	defaultNodes := make([]map[string]string, 0)
	for _, builderNode := range builderNodes {
		if isDefault, _ := builderNode[util.DEFAULT_KEY]; isDefault == "true" {
			defaultNodes = append(defaultNodes, builderNode)
			break
		}
	}
	return builderNodes
}
