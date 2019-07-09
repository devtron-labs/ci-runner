package main

import (
	"encoding/json"
	"fmt"
	"github.com/caarlos0/env"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"log"
	"os"
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
}

type CiCompleteEvent struct {
	CiProjectDetails []CiProjectDetails `json:"ciProjectDetails"`
	DockerImage      string             `json:"dockerImage"`
	Digest           string             `json:"digest"`
	PipelineId       int                `json:"pipelineId"`
	DataSource       string             `json:"dataSource"`
}

type CiProjectDetails struct {
	GitRepository string     `json:"gitRepository"`
	CheckoutPath  string     `json:"checkoutPath"`
	CommitHash    string     `json:"commitHash"`
	GitOptions    GitOptions `json:"gitOptions"`
	Branch        string     `json:"branch"`
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
	NatsServerHost string `env:"NATS_SERVER_HOST" envDefault:"nats://example-nats:4222"`
	ClusterId      string `env:"CLUSTER_ID" envDefault:"example-stan"`
	ClientId       string `env:"CLIENT_ID" envDefault:"ci-runner"`
}

const retryCount = 10

func main() {
	err := os.Chdir("/")
	if err != nil {
		os.Exit(1)
	}

	// sample arg -> "{\"dockerImageTag\":\"abc-bcd\",\"dockerRegistryURL\":\"686244538589.dkr.ecr.us-east-2.amazonaws.com\",\"dockerFileLocation\":\"./notifier-test/Dockerfile\",\"dockerRepository\":\"notifier-test\",\"awsRegion\":\"us-east-2\",\"ciCacheLocation\":\"s3://ci-caching/\",\"ciCacheFileName\":\"cache.tar.gz\",\"ciProjectDetails\":[{\"gitRepository\":\"https://gitlab.com/devtron/notifier.git\",\"checkoutPath\":\"./notifier-test\",\"commitHash\":\"a6b809c4be87c217feba4af15cf5ebc3cafe21e0\",\"branch\":\"master\",\"gitOptions\":{\"userName\":\"Suraj24\",\"password\":\"Devtron@1234\",\"sshKey\":\"\",\"accessToken\":\"\",\"authMode\":\"\"}},{\"gitRepository\":\"https://gitlab.com/devtron/orchestrator.git\",\"checkoutPath\":\"./orchestrator-test\",\"branch\":\"ci_with_argo\",\"gitOptions\":{\"userName\":\"Suraj24\",\"password\":\"Devtron@1234\",\"sshKey\":\"\",\"accessToken\":\"\",\"authMode\":\"\"}}]}"
	args := os.Args[1]
	fmt.Println("ci request -----> " + args)
	ciRequest := &CiRequest{}
	err = json.Unmarshal([]byte(args), ciRequest)
	if err != nil {
		os.Exit(1)
	}

	// Get ci cache
	log.Println("cf:start")
	GetCache(ciRequest)
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

	// sync cache
	log.Println("cs:start")
	err = SyncCache(ciRequest)
	if err != nil {
		os.Exit(1)
	}
	log.Println("cs:done")

	// debug mode
	//exec.Command("tail", "-f", "/dev/null").Run()
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
		DataSource:       "CI-RUNNER",
	}
	err = SendCiCompleteEvent(client, event)
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
		return &PubSubClient{}, err
	}
	sc, err := stan.Connect(cfg.ClusterId, cfg.ClientId, stan.NatsConn(nc))
	if err != nil {
		log.Println("err", err)
		return &PubSubClient{}, err
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
	log.Println("ci complete evt -----> ", reqBody)
	err = client.Conn.Publish(CI_COMPLETE_TOPIC, reqBody) // does not return until an ack has been received from NATS Streaming
	if err != nil {
		log.Println("publish err", "err", err)
		return err
	}
	return nil
}
