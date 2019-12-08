package main

import (
	"encoding/json"
	"fmt"
	_ "github.com/aws/aws-sdk-go/aws"
	"github.com/nats-io/stan.go"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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

	TestExecutorImageProperties *TestExecutorImageProperties `json:"testExecutorImageProperties"`
}

type Task struct {
	Id             int    `json:"id"`
	Index          int    `json:"index"`
	Name           string `json:"name"`
	Script         string `json:"script"`
	OutputLocation string `json:"outputLocation"` // file/dir
	runStatus      bool   `json:"-"`
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
------------------------------------------------------------------------------------------------------------
%s
------------------------------------------------------------------------------------------------------------`
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
	//before task
	for i, task := range ciRequest.BeforeDockerBuild {
		log.Println(devtron, "pre", task)
		//log running cmd
		logStage(task.Name)
		err = RunScripts(output_path, fmt.Sprintf("before-%d", i), task.Script, scriptEnvs)
		if err != nil {
			return err
		}
		task.runStatus = true
	}
	logStage("docker build")
	// build
	dest, err := BuildArtifact(ciRequest)
	if err != nil {
		return err
	}
	log.Println(devtron, " /docker-build")

	// run post artifact processing
	log.Println(devtron, " docker-build-post-processing")
	//after task
	for i, task := range ciRequest.AfterDockerBuild {
		log.Println(devtron, "post", task)
		logStage(task.Name)
		err = RunScripts(output_path, fmt.Sprintf("after-%d", i), task.Script, scriptEnvs)
		if err != nil {
			return err
		}
		task.runStatus = true
	}

	/*	// TODO: Remove
		ciRequest.AfterDockerBuildScript = "docker run --network=\"host\" -itd testsuraj-1:latest; sleep 10; curl -X GET http://localhost:8080/health;"
		err = RunScripts(output_path, bash_script, ciRequest.AfterDockerBuildScript)
		if err != nil {
			return err
		}
		log.Println(devtron, " /docker-build-post-processing")*/

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

	err = os.Chdir("/")
	if err != nil {
		log.Println(err)
		return err
	}

	tail := exec.Command("/bin/sh", "-c", "tail -f /dev/null")
	err = RunCommand(tail)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
