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

type CiRequest struct {
	CiProjectDetails   []CiProjectDetails `json:"ciProjectDetails"`
	DockerImageTag     string             `json:"dockerImageTag"`
	DockerRegistryType string             `json:"dockerRegistryType"`
	DockerRegistryURL  string             `json:"dockerRegistryURL"`
	DockerRepository   string             `json:"dockerRepository"`
	DockerBuildArgs    string             `json:"dockerBuildArgs"`
	DockerFileLocation string             `json:"dockerfileLocation"`
	DockerUsername     string             `json:"dockerUsername"`
	DockerPassword     string             `json:"dockerPassword"`
	AwsRegion          string             `json:"awsRegion"`
	AccessKey          string             `json:"accessKey"`
	SecretKey          string             `json:"secretKey"`
	CiCacheLocation    string             `json:"ciCacheLocation"`
	CiArtifactLocation string             `json:"ciArtifactLocation"` // s3 bucket+ path
	CiCacheRegion      string             `json:"ciCacheRegion"`
	CiCacheFileName    string             `json:"ciCacheFileName"`
	PipelineId         int                `json:"pipelineId"`
	PipelineName       string             `json:"pipelineName"`
	WorkflowId         int                `json:"workflowId"`
	TriggeredBy        int                `json:"triggeredBy"`
	CacheLimit         int64              `json:"cacheLimit"`
	BeforeDockerBuild  []*Task            `json:"beforeDockerBuildScripts"`
	AfterDockerBuild   []*Task            `json:"afterDockerBuildScripts"`
	CiYamlLocation     string             `json:"CiYamlLocations"`

	TestExecutorImageProperties *TestExecutorImageProperties `json:"testExecutorImageProperties"`
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

type CiProjectDetails struct {
	GitRepository string     `json:"gitRepository"`
	MaterialName  string     `json:"materialName"`
	CheckoutPath  string     `json:"checkoutPath"`
	CommitHash    string     `json:"commitHash"`
	CommitTime    time.Time  `json:"commitTime"`
	Branch        string     `json:"branch"`
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

const CI_COMPLETE_TOPIC = "CI-RUNNER.CI-COMPLETE"

type PubSubClient struct {
	Conn stan.Conn
}

type PubSubConfig struct {
	NatsServerHost string `env:"NATS_SERVER_HOST" envDefault:"nats://devtron-nats.devtroncd:4222"`
	ClusterId      string `env:"CLUSTER_ID" envDefault:"devtron-stan"`
	ClientId       string `env:"CLIENT_ID" envDefault:"CI-RUNNER"`
}

const retryCount = 10
const workingDir = "/devtroncd"
const devtron = "DEVTRON"

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
	// ' {"workflowNamePrefix":"55-suraj-23-ci-suraj-test-pipeline-8","pipelineName":"suraj-23-ci-suraj-test-pipeline","pipelineId":8,"dockerImageTag":"a6b809c4be87c217feba4af15cf5ebc3cafe21e0","dockerRegistryURL":"686244538589.dkr.ecr.us-east-2.amazonaws.com","dockerRepository":"test/suraj-23","dockerfileLocation":"./notifier/Dockerfile","awsRegion":"us-east-2","ciCacheLocation":"ci-caching","ciCacheFileName":"suraj-23-ci-suraj-test-pipeline.tar.gz","ciProjectDetails":[{"gitRepository":"https://gitlab.com/devtron/notifier.git","materialName":"1-notifier","checkoutPath":"./notifier","commitHash":"d4df38bcd065004014d255c2203d592a91585955","commitTime":"0001-01-01T00:00:00Z","branch":"ci_with_argo","type":"SOURCE_TYPE_BRANCH_FIXED","message":"test-commit","gitOptions":{"userName":"Suraj24","password":"Devtron@1234","sshKey":"","accessToken":"","authMode":"USERNAME_PASSWORD"}},{"gitRepository":"https://gitlab.com/devtron/orchestrator.git","materialName":"2-orchestrator","checkoutPath":"./orch","commitHash":"","commitTime":"0001-01-01T00:00:00Z","branch":"ci_with_argo","type":"SOURCE_TYPE_BRANCH_FIXED","message":"","gitOptions":{"userName":"Suraj24","password":"Devtron@1234","sshKey":"","accessToken":"","authMode":""}}],"ciImage":"686244538589.dkr.ecr.us-east-2.amazonaws.com/cirunner:latest","namespace":"default"}'
	args := os.Args[1]
	ciRequest := &CiRequest{}
	err := json.Unmarshal([]byte(args), ciRequest)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	log.Println(devtron, " ci request details -----> ", args)

	err = run(ciRequest)
	artifactUploadErr := collectAndUploadArtifact(ciRequest)
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
}

func collectAndUploadArtifact(ciRequest *CiRequest) error {
	artifactFiles := make(map[string]string)
	for _, task := range append(ciRequest.BeforeDockerBuild, ciRequest.AfterDockerBuild...) {
		if task.runStatus {
			if _, err := os.Stat(task.OutputLocation); os.IsNotExist(err) { // Ignore if no file/folder
				log.Println(devtron, "artifact not found ", err)
				continue
			}
			artifactFiles[task.Name] = task.OutputLocation
		}
	}
	log.Println(devtron, " artifacts", artifactFiles)
	return UploadArtifact(artifactFiles, ciRequest.CiArtifactLocation)
}

func getScriptEnvVariables(ciRequest *CiRequest) map[string]string {
	envs := make(map[string]string)
	//TODO ADD MORE env variable
	envs["DOCKER_IMAGE_TAG"] = ciRequest.DockerImageTag
	envs["DOCKER_REPOSITORY"] = ciRequest.DockerRepository
	envs["DOCKER_REGISTRY_URL"] = ciRequest.DockerRegistryURL
	return envs
}

func run(ciRequest *CiRequest) error {
	err := os.Chdir("/")
	if err != nil {
		return err
	}

	if _, err := os.Stat(workingDir); os.IsNotExist(err) {
		_ = os.Mkdir(workingDir, os.ModeDir)
	}

	// Get ci cache
	log.Println(devtron, " cache-pull")
	err = GetCache(ciRequest)
	if err != nil {
		return err
	}
	log.Println(devtron, " /cache-pull")

	err = os.Chdir(workingDir)
	if err != nil {
		return err
	}
	// git handling
	log.Println(devtron, " git")
	err = CloneAndCheckout(ciRequest)
	if err != nil {
		log.Println(devtron, "clone err: ", err)
		return err
	}
	log.Println(devtron, " /git")

	// Start docker daemon
	log.Println(devtron, " docker-build")
	StartDockerDaemon()
	scriptEnvs := getScriptEnvVariables(ciRequest)

	// Get devtron-ci yaml
	yamlLocation := ciRequest.DockerFileLocation[:strings.LastIndex(ciRequest.DockerFileLocation, "/")+1]
	log.Println(devtron, "devtron-ci yaml location ", yamlLocation)
	taskYaml, err := GetTaskYaml(yamlLocation)
	if err != nil {
		return err
	}

	// run pre artifact processing
	err = RunPreDockerBuildTasks(ciRequest, scriptEnvs, taskYaml)
	if err != nil {
		log.Println(err)
		return err
	}

	logStage("docker build")
	// build
	dest, err := BuildArtifact(ciRequest)
	if err != nil {
		return err
	}
	log.Println(devtron, " /docker-build")

	// run post artifact processing
	err = RunPostDockerBuildTasks(ciRequest, scriptEnvs, taskYaml)
	if err != nil {
		return err
	}

	logStage("docker push")
	// push to dest
	log.Println(devtron, " docker-push")
	digest, err := PushArtifact(ciRequest, dest)
	if err != nil {
		return err
	}
	log.Println(devtron, " /docker-push")

	log.Println(devtron, " event")
	err = SendEvents(ciRequest, digest, dest)
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

func RunPreDockerBuildTasks(ciRequest *CiRequest, scriptEnvs map[string]string, taskYaml *TaskYaml) error {
	beforeYamlTasks, err := GetBeforeDockerBuildTasks(ciRequest, taskYaml)
	if err != nil {
		log.Println(err)
		return err
	}

	//before task
	beforeTaskMap := make(map[string]*Task)
	for i, task := range ciRequest.BeforeDockerBuild {
		task.runStatus = true
		beforeTaskMap[task.Name] = task
		log.Println(devtron, "pre", task)
		//log running cmd
		logStage(task.Name)
		err = RunScripts(output_path, fmt.Sprintf("before-%d", i), task.Script, scriptEnvs)
		if err != nil {
			return err
		}
	}

	// run before yaml tasks
	for i, task := range beforeYamlTasks {
		if _, ok := beforeTaskMap[task.Name]; ok {
			log.Println("duplicate task found in yaml, ran earlier so ignoring")
			continue
		}
		task.runStatus = true
		log.Println(devtron, "pre - yaml", task)
		//log running cmd
		logStage(task.Name)
		err = RunScripts(output_path, fmt.Sprintf("before-yaml-%d", i), task.Script, scriptEnvs)
		if err != nil {
			return err
		}
	}
	return nil
}

func RunPostDockerBuildTasks(ciRequest *CiRequest, scriptEnvs map[string]string, taskYaml *TaskYaml) error {
	afterYamlTasks, err := GetAfterDockerBuildTasks(ciRequest, taskYaml)
	if err != nil {
		log.Println(err)
		return err
	}

	log.Println(devtron, " docker-build-post-processing")
	afterTaskMap := make(map[string]*Task)
	for i, task := range ciRequest.AfterDockerBuild {
		task.runStatus = true
		afterTaskMap[task.Name] = task
		log.Println(devtron, "post", task)
		logStage(task.Name)
		err = RunScripts(output_path, fmt.Sprintf("after-%d", i), task.Script, scriptEnvs)
		if err != nil {
			return err
		}
	}
	for i, task := range afterYamlTasks {
		if _, ok := afterTaskMap[task.Name]; ok {
			log.Println("duplicate task found in yaml, already run so ignoring")
			continue
		}
		task.runStatus = true
		log.Println(devtron, "post - yaml", task)
		//log running cmd
		logStage(task.Name)
		err = RunScripts(output_path, fmt.Sprintf("after-yaml-%d", i), task.Script, scriptEnvs)
		if err != nil {
			return err
		}
	}
	return nil
}
