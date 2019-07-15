package main

import (
	"encoding/json"
	"fmt"
	_ "github.com/aws/aws-sdk-go/aws"
	"github.com/caarlos0/env"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"syscall"
	"time"
)

type CiRequest struct {
	CiProjectDetails   []CiProjectDetails `json:"ciProjectDetails"`
	DockerImageTag     string             `json:"dockerImageTag"`
	DockerRegistryURL  string             `json:"dockerRegistryURL"`
	DockerRepository   string             `json:"dockerRepository"`
	DockerFileLocation string             `json:"dockerfileLocation"`
	AwsRegion          string             `json:"awsRegion"`
	CiCacheLocation    string             `json:"ciCacheLocation"`
	CiCacheFileName    string             `json:"ciCacheFileName"`
	PipelineId         int                `json:"pipelineId"`
	PipelineName       string             `json:"pipelineName"`
}

type CiCompleteEvent struct {
	CiProjectDetails []CiProjectDetails `json:"ciProjectDetails"`
	DockerImage      string             `json:"dockerImage"`
	Digest           string             `json:"digest"`
	PipelineId       int                `json:"pipelineId"`
	DataSource       string             `json:"dataSource"`
	PipelineName     string             `json:"pipelineName"`
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
	ClientId       string `env:"CLIENT_ID" envDefault:"ci-runner"`
}

const retryCount = 10

func main() {
	err := os.Chdir("/")
	if err != nil {
		os.Exit(1)
	}

	// ' {"workflowNamePrefix":"55-suraj-23-ci-suraj-test-pipeline-8","pipelineName":"suraj-23-ci-suraj-test-pipeline","pipelineId":8,"dockerImageTag":"a6b809c4be87c217feba4af15cf5ebc3cafe21e0","dockerRegistryURL":"686244538589.dkr.ecr.us-east-2.amazonaws.com","dockerRepository":"test/suraj-23","dockerfileLocation":"./notifier/Dockerfile","awsRegion":"us-east-2","ciCacheLocation":"ci-caching","ciCacheFileName":"suraj-23-ci-suraj-test-pipeline.tar.gz","ciProjectDetails":[{"gitRepository":"https://gitlab.com/devtron/notifier.git","materialName":"1-notifier","checkoutPath":"./notifier","commitHash":"d4df38bcd065004014d255c2203d592a91585955","commitTime":"0001-01-01T00:00:00Z","branch":"ci_with_argo","type":"SOURCE_TYPE_BRANCH_FIXED","message":"test-commit","gitOptions":{"userName":"Suraj24","password":"Devtron@1234","sshKey":"","accessToken":"","authMode":"USERNAME_PASSWORD"}},{"gitRepository":"https://gitlab.com/devtron/orchestrator.git","materialName":"2-orchestrator","checkoutPath":"./orch","commitHash":"","commitTime":"0001-01-01T00:00:00Z","branch":"ci_with_argo","type":"SOURCE_TYPE_BRANCH_FIXED","message":"","gitOptions":{"userName":"Suraj24","password":"Devtron@1234","sshKey":"","accessToken":"","authMode":""}}],"ciImage":"686244538589.dkr.ecr.us-east-2.amazonaws.com/cirunner:latest","namespace":"default"}'
	args := os.Args[1]
	fmt.Println("ci request -----> " + args)
	ciRequest := &CiRequest{}
	err = json.Unmarshal([]byte(args), ciRequest)
	if err != nil {
		os.Exit(1)
	}

	// Get ci cache
	log.Println("cf:start")
	err = GetCache(ciRequest)
	if err != nil {
		os.Exit(1)
	}
	log.Println("cf:done")

	// git handling
	log.Println("gf:start")
	CloneAndCheckout(ciRequest)
	log.Println("gf:done")

	// Start docker daemon
	log.Println("db:start")
	StartDockerDaemon()

	// build
	dest, err := BuildArtifact(ciRequest)
	if err != nil {
		os.Exit(1)
	}
	log.Println("db:done")

	// push to dest
	log.Println("dp:start")
	digest, err := PushArtifact(ciRequest, dest)
	if err != nil {
		os.Exit(1)
	}
	log.Println("dp:done")

	log.Println("ns:start")
	err = SendEvents(ciRequest, digest, dest)
	if err != nil {
		os.Exit(1)
	}
	log.Println("ns:done")

	err = StopDocker()
	if err != nil {
		log.Println("err", err)
		os.Exit(1)
	}

	// sync cache
	log.Println("cs:start")
	err = SyncCache(ciRequest)
	if err != nil {
		os.Exit(1)
	}
	log.Println("cs:done")
}

func SendEvents(ciRequest *CiRequest, digest string, image string) error {
	client, err := NewPubSubClient()
	if err != nil {
		log.Println("err", err)
		os.Exit(1)
	}
	event := CiCompleteEvent{
		CiProjectDetails: ciRequest.CiProjectDetails,
		DockerImage:      image,
		Digest:           digest,
		PipelineId:       ciRequest.PipelineId,
		PipelineName:     ciRequest.PipelineName,
		DataSource:       "CI-RUNNER",
	}
	err = SendCiCompleteEvent(client, event)
	nc := client.Conn.NatsConn()

	err = client.Conn.Close()
	if err != nil {
		log.Println("error in closing stan", "err", err)
	}

	err = nc.Drain()
	if err != nil {
		log.Println("error in draining nats", "err", err)
	}
	nc.Close()
	log.Println("housekeeping done. exiting now")
	return err
}

func NewPubSubClient() (*PubSubClient, error) {
	cfg := &PubSubConfig{}
	err := env.Parse(cfg)
	if err != nil {
		return &PubSubClient{}, err
	}
	nc, err := nats.Connect(cfg.NatsServerHost)
	if err != nil {
		log.Println("err", err)
		os.Exit(1)
	}
	s := rand.NewSource(time.Now().UnixNano())
	uuid := rand.New(s)
	uniqueClienId := "ci-runner-" + strconv.Itoa(uuid.Int())

	sc, err := stan.Connect(cfg.ClusterId, uniqueClienId, stan.NatsConn(nc))
	if err != nil {
		log.Println("err", err)
		os.Exit(1)
	}
	natsClient := &PubSubClient{
		Conn: sc,
	}
	return natsClient, nil
}

func SendCiCompleteEvent(client *PubSubClient, event CiCompleteEvent) error {
	jsonBody, err := json.Marshal(event)
	if err != nil {
		log.Println("err", err)
		return err
	}
	var reqBody = []byte(jsonBody)
	log.Println("ci complete evt -----> ", string(reqBody))
	err = client.Conn.Publish(CI_COMPLETE_TOPIC, reqBody) // does not return until an ack has been received from NATS Streaming
	if err != nil {
		log.Println("publish err", "err", err)
		return err
	}
	return nil
}

func StopDocker() error {
	file := "/var/run/docker.pid"
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
		return err
	}

	pid, err := strconv.Atoi(string(content))
	if err != nil {
		return err
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		log.Println(err)
		return err
	}
	// Kill the process
	err = proc.Signal(syscall.SIGTERM)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println("-----> checking docker status")
	DockerdUpCheck()
	return nil
}
