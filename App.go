package main

import (
	"encoding/json"
	_ "github.com/aws/aws-sdk-go/aws"
	"github.com/nats-io/stan.go"
	"log"
	"os"
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
	CiCacheRegion      string             `json:"ciCacheRegion"`
	CiCacheFileName    string             `json:"ciCacheFileName"`
	PipelineId         int                `json:"pipelineId"`
	PipelineName       string             `json:"pipelineName"`
	WorkflowId         int                `json:"workflowId"`
	TriggeredBy        int                `json:"triggeredBy"`
	CacheLimit         int64              `json:"cacheLimit"`
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
const workingDir = "./devtroncd"
const devtron = "DEVTRON"

func main() {
	err := os.Chdir("/")
	if err != nil {
		os.Exit(1)
	}

	if _, err := os.Stat(workingDir); os.IsNotExist(err) {
		_ = os.Mkdir(workingDir, os.ModeDir)
	}

	// ' {"workflowNamePrefix":"55-suraj-23-ci-suraj-test-pipeline-8","pipelineName":"suraj-23-ci-suraj-test-pipeline","pipelineId":8,"dockerImageTag":"a6b809c4be87c217feba4af15cf5ebc3cafe21e0","dockerRegistryURL":"686244538589.dkr.ecr.us-east-2.amazonaws.com","dockerRepository":"test/suraj-23","dockerfileLocation":"./notifier/Dockerfile","awsRegion":"us-east-2","ciCacheLocation":"ci-caching","ciCacheFileName":"suraj-23-ci-suraj-test-pipeline.tar.gz","ciProjectDetails":[{"gitRepository":"https://gitlab.com/devtron/notifier.git","materialName":"1-notifier","checkoutPath":"./notifier","commitHash":"d4df38bcd065004014d255c2203d592a91585955","commitTime":"0001-01-01T00:00:00Z","branch":"ci_with_argo","type":"SOURCE_TYPE_BRANCH_FIXED","message":"test-commit","gitOptions":{"userName":"Suraj24","password":"Devtron@1234","sshKey":"","accessToken":"","authMode":"USERNAME_PASSWORD"}},{"gitRepository":"https://gitlab.com/devtron/orchestrator.git","materialName":"2-orchestrator","checkoutPath":"./orch","commitHash":"","commitTime":"0001-01-01T00:00:00Z","branch":"ci_with_argo","type":"SOURCE_TYPE_BRANCH_FIXED","message":"","gitOptions":{"userName":"Suraj24","password":"Devtron@1234","sshKey":"","accessToken":"","authMode":""}}],"ciImage":"686244538589.dkr.ecr.us-east-2.amazonaws.com/cirunner:latest","namespace":"default"}'
	args := os.Args[1]
	ciRequest := &CiRequest{}
	err = json.Unmarshal([]byte(args), ciRequest)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	log.Println(devtron, " ci request details -----> ", args)

	// Get ci cache
	log.Println(devtron, " cache-pull")
	err = GetCache(ciRequest)
	if err != nil {
		os.Exit(1)
	}
	log.Println(devtron, " /cache-pull")

	err = os.Chdir(workingDir)
	if err != nil {
		os.Exit(1)
	}
	// git handling
	log.Println(devtron, " git")
	CloneAndCheckout(ciRequest)
	log.Println(devtron, " /git")

	// Start docker daemon
	log.Println(devtron, " docker-build")
	StartDockerDaemon()

	// build
	dest, err := BuildArtifact(ciRequest)
	if err != nil {
		os.Exit(1)
	}
	log.Println(devtron, " /docker-build")

	// push to dest
	log.Println(devtron, " docker-push")
	digest, err := PushArtifact(ciRequest, dest)
	if err != nil {
		os.Exit(1)
	}
	log.Println(devtron, " /docker-push")

	log.Println(devtron, " event")
	err = SendEvents(ciRequest, digest, dest)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	log.Println(devtron, " /event")

	err = StopDocker()
	if err != nil {
		log.Println("err", err)
		os.Exit(1)
	}

	err = os.Chdir("/")
	if err != nil {
		log.Println(err)
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
