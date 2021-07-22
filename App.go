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
	"github.com/nats-io/stan.go"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CiCdTriggerEvent struct {
	Type      string     `json:"type"`
	CiRequest *CiRequest `json:"ciRequest"`
	CdRequest *CdRequest `json:"cdRequest"`
}

type CdRequest struct {
	WorkflowId                int                `json:"workflowId"`
	WorkflowRunnerId          int                `json:"workflowRunnerId"`
	CdPipelineId              int                `json:"cdPipelineId"`
	TriggeredBy               int32              `json:"triggeredBy"`
	StageYaml                 string             `json:"stageYaml"`
	ArtifactLocation          string             `json:"artifactLocation"`
	TaskYaml                  *TaskYaml          `json:"-"`
	CiProjectDetails          []CiProjectDetails `json:"ciProjectDetails"`
	CiArtifactDTO             CiArtifactDTO      `json:"ciArtifactDTO"`
	DockerUsername            string             `json:"dockerUsername"`
	DockerPassword            string             `json:"dockerPassword"`
	AwsRegion                 string             `json:"awsRegion"`
	AccessKey                 string             `json:"accessKey"`
	SecretKey                 string             `json:"secretKey"`
	DockerRegistryURL         string             `json:"dockerRegistryUrl"`
	DockerRegistryType        string             `json:"dockerRegistryType"`
	OrchestratorHost          string             `json:"orchestratorHost"`
	OrchestratorToken         string             `json:"orchestratorToken"`
	IsExtRun                  bool               `json:"isExtRun"`
	ExtraEnvironmentVariables map[string]string  `json:"extraEnvironmentVariables"`
	CloudProvider             string             `json:"cloudProvider"`
	AzureBlobConfig           *AzureBlobConfig   `json:"azureBlobConfig"`
	MinioEndpoint             string             `json:"minioEndpoint"`
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

type CiRequest struct {
	CiProjectDetails            []CiProjectDetails           `json:"ciProjectDetails"`
	DockerImageTag              string                       `json:"dockerImageTag"`
	DockerRegistryType          string                       `json:"dockerRegistryType"`
	DockerRegistryURL           string                       `json:"dockerRegistryURL"`
	DockerRepository            string                       `json:"dockerRepository"`
	DockerBuildArgs             string                       `json:"dockerBuildArgs"`
	DockerFileLocation          string                       `json:"dockerfileLocation"`
	DockerUsername              string                       `json:"dockerUsername"`
	DockerPassword              string                       `json:"dockerPassword"`
	AwsRegion                   string                       `json:"awsRegion"`
	AccessKey                   string                       `json:"accessKey"`
	SecretKey                   string                       `json:"secretKey"`
	CiCacheLocation             string                       `json:"ciCacheLocation"`
	CiArtifactLocation          string                       `json:"ciArtifactLocation"` // s3 bucket+ path
	CiCacheRegion               string                       `json:"ciCacheRegion"`
	CiCacheFileName             string                       `json:"ciCacheFileName"`
	PipelineId                  int                          `json:"pipelineId"`
	PipelineName                string                       `json:"pipelineName"`
	WorkflowId                  int                          `json:"workflowId"`
	TriggeredBy                 int                          `json:"triggeredBy"`
	CacheLimit                  int64                        `json:"cacheLimit"`
	BeforeDockerBuild           []*Task                      `json:"beforeDockerBuildScripts"`
	AfterDockerBuild            []*Task                      `json:"afterDockerBuildScripts"`
	CiYamlLocation              string                       `json:"CiYamlLocations"`
	TaskYaml                    *TaskYaml                    `json:"-"`
	TestExecutorImageProperties *TestExecutorImageProperties `json:"testExecutorImageProperties"`
	InvalidateCache             bool                         `json:"invalidateCache"`
	ScanEnabled                 bool                         `json:"scanEnabled"`
	CloudProvider               string                       `json:"cloudProvider"`
	AzureBlobConfig             *AzureBlobConfig             `json:"azureBlobConfig"`
	MinioEndpoint               string                       `json:"minioEndpoint"`
}

const BLOB_STORAGE_AZURE = "AZURE"
const BLOB_STORAGE_S3 = "S3"
const BLOB_STORAGE_GCP = "GCP"
const BLOB_STORAGE_MINIO = "MINIO"

type AzureBlobConfig struct {
	Enabled              bool   `json:"enabled"`
	AccountName          string `json:"accountName"`
	BlobContainerCiLog   string `json:"blobContainerCiLog"`
	BlobContainerCiCache string `json:"blobContainerCiCache"`
	AccountKey           string `json:"accountKey"`
}

type Task struct {
	Id             int    `json:"id"`
	Index          int    `json:"index"`
	Name           string `json:"name" yaml:"name"`
	Script         string `json:"script" yaml:"script"`
	OutputLocation string `json:"outputLocation" yaml:"outputLocation"` // file/dir
	runStatus      bool   `json:"-"`                                    // task run was attempted or not
}

type TestExecutorImageProperties struct {
	ImageName string `json:"imageName,omitempty"`
	Arg       string `json:"arg,omitempty"`
}

type CiCompleteEvent struct {
	CiProjectDetails []CiProjectDetails `json:"ciProjectDetails"`
	DockerImage      string             `json:"dockerImage"`
	Digest           string             `json:"digest"`
	PipelineId       int                `json:"pipelineId"`
	DataSource       string             `json:"dataSource"`
	PipelineName     string             `json:"pipelineName"`
	WorkflowId       int                `json:"workflowId"`
	TriggeredBy      int                `json:"triggeredBy"`
	MaterialType     string             `json:"materialType"`
}

type CdStageCompleteEvent struct {
	CiProjectDetails []CiProjectDetails `json:"ciProjectDetails"`
	WorkflowId       int                `json:"workflowId"`
	WorkflowRunnerId int                `json:"workflowRunnerId"`
	CdPipelineId     int                `json:"cdPipelineId"`
	TriggeredBy      int32              `json:"triggeredBy"`
	StageYaml        string             `json:"stageYaml"`
	ArtifactLocation string             `json:"artifactLocation"`
	TaskYaml         *TaskYaml          `json:"-"`
	PipelineName     string             `json:"pipelineName"`
	CiArtifactDTO    CiArtifactDTO      `json:"ciArtifactDTO"`
}

type CiProjectDetails struct {
	GitRepository string     `json:"gitRepository"`
	MaterialName  string     `json:"materialName"`
	CheckoutPath  string     `json:"checkoutPath"`
	CommitHash    string     `json:"commitHash"`
	GitTag        string     `json:"gitTag"`
	CommitTime    time.Time  `json:"commitTime"`
	SourceType    SourceType `json:"sourceType"`
	SourceValue   string     `json:"sourceValue"`
	Type          string     `json:"type"`
	Message       string     `json:"message"`
	Author        string     `json:"author"`
	GitOptions    GitOptions `json:"gitOptions"`
}

type GitOptions struct {
	UserName    string   `json:"userName"`
	Password    string   `json:"password"`
	SSHKey      string   `json:"sshKey"`
	AccessToken string   `json:"accessToken"`
	AuthMode    AuthMode `json:"authMode"`
}
type AuthMode string

const (
	AUTH_MODE_USERNAME_PASSWORD AuthMode = "USERNAME_PASSWORD"
	AUTH_MODE_SSH               AuthMode = "SSH"
	AUTH_MODE_ACCESS_TOKEN      AuthMode = "ACCESS_TOKEN"
	AUTH_MODE_ANONYMOUS         AuthMode = "ANONYMOUS"
)

type SourceType string

const (
	SOURCE_TYPE_BRANCH_FIXED SourceType = "SOURCE_TYPE_BRANCH_FIXED"
	SOURCE_TYPE_BRANCH_REGEX SourceType = "SOURCE_TYPE_BRANCH_REGEX"
	SOURCE_TYPE_TAG_ANY      SourceType = "SOURCE_TYPE_TAG_ANY"
	SOURCE_TYPE_TAG_REGEX    SourceType = "SOURCE_TYPE_TAG_REGEX"
)

const CI_COMPLETE_TOPIC = "CI-RUNNER.CI-COMPLETE"
const CD_COMPLETE_TOPIC = "CI-RUNNER.CD-STAGE-COMPLETE"

type PubSubClient struct {
	Conn stan.Conn
}

type PubSubConfig struct {
	NatsServerHost       string `env:"NATS_SERVER_HOST" envDefault:"nats://devtron-nats.devtroncd:4222"`
	ClusterId            string `env:"CLUSTER_ID" envDefault:"devtron-stan"`
	ClientId             string `env:"CLIENT_ID" envDefault:"CI-RUNNER"`
	ImageScannerEndpoint string `env:"IMAGE_SCANNER_ENDPOINT" envDefault:"http://image-scanner-new-demo-devtroncd-service.devtroncd:80"`
}

const retryCount = 10
const workingDir = "/devtroncd"
const devtron = "DEVTRON"

const ciEvent = "CI"
const cdStage = "CD"

const ImageScannerEndpoint string = "http://image-scanner-new-demo-devtroncd-service.devtroncd:80"

var (
	output_path = filepath.Join("./process")
	bash_script = filepath.Join("_script.sh")
)

func logStage(name string) {
	stageTemplate := `
------------------------------------------------------------------------------------------------------------------------
STAGE:  %s
------------------------------------------------------------------------------------------------------------------------`
	log.Println(fmt.Sprintf(stageTemplate, name))
}

func main() {
	//' {"workflowNamePrefix":"55-suraj-23-ci-suraj-test-pipeline-8","pipelineName":"suraj-23-ci-suraj-test-pipeline","pipelineId":8,"dockerImageTag":"a6b809c4be87c217feba4af15cf5ebc3cafe21e0","dockerRegistryURL":"686244538589.dkr.ecr.us-east-2.amazonaws.com","dockerRepository":"test/suraj-23","dockerfileLocation":"./notifier/Dockerfile","awsRegion":"us-east-2","ciCacheLocation":"ci-caching","ciCacheFileName":"suraj-23-ci-suraj-test-pipeline.tar.gz","ciProjectDetails":[{"gitRepository":"https://gitlab.com/devtron/notifier.git","materialName":"1-notifier","checkoutPath":"./notifier","commitHash":"d4df38bcd065004014d255c2203d592a91585955","commitTime":"0001-01-01T00:00:00Z","branch":"ci_with_argo","type":"SOURCE_TYPE_BRANCH_FIXED","message":"test-commit","gitOptions":{"userName":"Suraj24","password":"Devtron@1234","sshKey":"","accessToken":"","authMode":"USERNAME_PASSWORD"}},{"gitRepository":"https://gitlab.com/devtron/orchestrator.git","materialName":"2-orchestrator","checkoutPath":"./orch","commitHash":"","commitTime":"0001-01-01T00:00:00Z","branch":"ci_with_argo","type":"SOURCE_TYPE_BRANCH_FIXED","message":"","gitOptions":{"userName":"Suraj24","password":"Devtron@1234","sshKey":"","accessToken":"","authMode":""}}],"ciImage":"686244538589.dkr.ecr.us-east-2.amazonaws.com/cirunner:latest","namespace":"default"}'
	args := os.Args[1]
	ciCdRequest := &CiCdTriggerEvent{}
	err := json.Unmarshal([]byte(args), ciCdRequest)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	log.Println(devtron, " ci-cd request details -----> ", args)

	if ciCdRequest.Type == ciEvent {
		ciRequest := ciCdRequest.CiRequest
		artifactUploaded, err := run(ciCdRequest)
		log.Println(devtron, artifactUploaded, err)
		var artifactUploadErr error
		if !artifactUploaded {
			artifactUploadErr = collectAndUploadArtifact(ciRequest)
		}

		if err != nil || artifactUploadErr != nil {
			log.Println(devtron, err, artifactUploadErr)
			os.Exit(1)
		}

		// sync cache
		log.Println(devtron, " cache-push")
		err = SyncCache(ciRequest)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		log.Println(devtron, " /cache-push")
	} else {
		err = runCDStages(ciCdRequest)
		artifactUploadErr := collectAndUploadCDArtifacts(ciCdRequest.CdRequest)
		if err != nil || artifactUploadErr != nil {
			log.Println(err)
			os.Exit(1)
		}
	}
}

func collectAndUploadCDArtifacts(cdRequest *CdRequest) error {
	artifactFiles := make(map[string]string)
	var allTasks []*Task
	if cdRequest.TaskYaml != nil {
		for _, pc := range cdRequest.TaskYaml.CdPipelineConfig {
			for _, t := range append(pc.BeforeTasks, pc.AfterTasks...) {
				allTasks = append(allTasks, t)
			}
		}
	}
	for _, task := range allTasks {
		if task.runStatus {
			if _, err := os.Stat(task.OutputLocation); os.IsNotExist(err) { // Ignore if no file/folder
				log.Println(devtron, "artifact not found ", err)
				continue
			}
			artifactFiles[task.Name] = task.OutputLocation
		}
	}
	log.Println(devtron, " artifacts", artifactFiles)
	return UploadArtifact(artifactFiles, cdRequest.ArtifactLocation, cdRequest.CloudProvider, cdRequest.MinioEndpoint, cdRequest.AzureBlobConfig)
}

func collectAndUploadArtifact(ciRequest *CiRequest) error {
	artifactFiles := make(map[string]string)
	var allTasks []*Task
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
		if task.runStatus {
			if _, err := os.Stat(task.OutputLocation); os.IsNotExist(err) { // Ignore if no file/folder
				log.Println(devtron, "artifact not found ", err)
				continue
			}
			artifactFiles[task.Name] = task.OutputLocation
		}
	}
	log.Println(devtron, " artifacts", artifactFiles)
	return UploadArtifact(artifactFiles, ciRequest.CiArtifactLocation, ciRequest.CloudProvider, ciRequest.MinioEndpoint, ciRequest.AzureBlobConfig)
}

func getScriptEnvVariables(cicdRequest *CiCdTriggerEvent) map[string]string {
	envs := make(map[string]string)
	//TODO ADD MORE env variable
	if cicdRequest.Type == ciEvent {
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

func run(ciCdRequest *CiCdTriggerEvent) (artifactUploaded bool, err error) {
	artifactUploaded = false
	err = os.Chdir("/")
	if err != nil {
		return artifactUploaded, err
	}

	if _, err := os.Stat(workingDir); os.IsNotExist(err) {
		_ = os.Mkdir(workingDir, os.ModeDir)
	}

	// Get ci cache
	log.Println(devtron, " cache-pull")
	err = GetCache(ciCdRequest.CiRequest)
	if err != nil {
		return artifactUploaded, err
	}
	log.Println(devtron, " /cache-pull")

	err = os.Chdir(workingDir)
	if err != nil {
		return artifactUploaded, err
	}
	// git handling
	log.Println(devtron, " git")
	err = CloneAndCheckout(ciCdRequest.CiRequest.CiProjectDetails)
	if err != nil {
		log.Println(devtron, "clone err: ", err)
		return artifactUploaded, err
	}
	log.Println(devtron, " /git")

	// Start docker daemon
	log.Println(devtron, " docker-build")
	StartDockerDaemon()
	scriptEnvs := getScriptEnvVariables(ciCdRequest)

	// Get devtron-ci yaml
	yamlLocation := ciCdRequest.CiRequest.DockerFileLocation[:strings.LastIndex(ciCdRequest.CiRequest.DockerFileLocation, "/")+1]
	log.Println(devtron, "devtron-ci yaml location ", yamlLocation)
	taskYaml, err := GetTaskYaml(yamlLocation)
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

	logStage("docker build")
	// build
	dest, err := BuildArtifact(ciCdRequest.CiRequest)
	if err != nil {
		return artifactUploaded, err
	}
	log.Println(devtron, " /docker-build")

	// run post artifact processing
	err = RunPostDockerBuildTasks(ciCdRequest.CiRequest, scriptEnvs, taskYaml)
	if err != nil {
		return artifactUploaded, err
	}

	logStage("docker push")
	// push to dest
	log.Println(devtron, " docker-push")
	digest, err := PushArtifact(ciCdRequest.CiRequest, dest)
	if err != nil {
		return artifactUploaded, err
	}
	log.Println(devtron, " /docker-push")

	log.Println(devtron, " artifact-upload")
	err = collectAndUploadArtifact(ciCdRequest.CiRequest)
	if err != nil {
		return artifactUploaded, err
	} else {
		artifactUploaded = true
	}
	log.Println(devtron, " /artifact-upload")

	// scan only if ci scan enabled
	if ciCdRequest.CiRequest.ScanEnabled {
		logStage("IMAGE SCAN")
		log.Println(devtron, " /image-scanner")
		scanEvent := &ScanEvent{Image: dest, ImageDigest: digest, PipelineId: ciCdRequest.CiRequest.PipelineId, UserId: ciCdRequest.CiRequest.TriggeredBy}
		scanEvent.AccessKey = ciCdRequest.CiRequest.AccessKey
		scanEvent.SecretKey = ciCdRequest.CiRequest.SecretKey
		err = SendEventToClairUtility(scanEvent)
		if err != nil {
			log.Println(err)
			return artifactUploaded, err
		}
		log.Println(devtron, " /image-scanner")
	}

	log.Println(devtron, " event")
	err = SendEvents(ciCdRequest.CiRequest, digest, dest)
	if err != nil {
		log.Println(err)
		return artifactUploaded, err
	}
	log.Println(devtron, " /event")

	err = StopDocker()
	if err != nil {
		log.Println("err", err)
		return artifactUploaded, err
	}
	return artifactUploaded, nil
}

func runCDStages(cicdRequest *CiCdTriggerEvent) error {
	err := os.Chdir("/")
	if err != nil {
		return err
	}

	if _, err := os.Stat(workingDir); os.IsNotExist(err) {
		_ = os.Mkdir(workingDir, os.ModeDir)
	}
	err = os.Chdir(workingDir)
	if err != nil {
		return err
	}
	// git handling
	log.Println(devtron, " git")
	err = CloneAndCheckout(cicdRequest.CdRequest.CiProjectDetails)
	if err != nil {
		log.Println(devtron, "clone err: ", err)
		return err
	}
	log.Println(devtron, " /git")

	// Start docker daemon
	log.Println(devtron, " docker-start")
	StartDockerDaemon()
	err = DockerLogin(&DockerCredentials{
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
	taskYaml, err := ToTaskYaml([]byte(cicdRequest.CdRequest.StageYaml))
	if err != nil {
		log.Println(err)
		return err
	}
	cicdRequest.CdRequest.TaskYaml = taskYaml

	// run post artifact processing
	log.Println(devtron, " stage yaml", taskYaml)
	var tasks []*Task
	for _, t := range taskYaml.CdPipelineConfig {
		tasks = append(tasks, t.BeforeTasks...)
		tasks = append(tasks, t.AfterTasks...)
	}

	scriptEnvs := getScriptEnvVariables(cicdRequest)
	err = RunCdStageTasks(tasks, scriptEnvs)
	if err != nil {
		return err
	}

	log.Println(devtron, " event")
	err = SendCDEvent(cicdRequest.CdRequest)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println(devtron, " /event")

	err = StopDocker()
	if err != nil {
		log.Println("err", err)
		return err
	}
	return nil
}
