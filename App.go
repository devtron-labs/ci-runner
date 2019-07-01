package main

import (
	"encoding/json"
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

const retryCount = 10

func main() {
	err := os.Chdir("/")
	if err != nil {
		os.Exit(1)
	}

	// sample arg -> "{\"dockerImageTag\":\"abc-bcd\",\"dockerRegistryURL\":\"686244538589.dkr.ecr.us-east-2.amazonaws.com\",\"dockerFileLocation\":\"./notifier-test/Dockerfile\",\"dockerRepository\":\"notifier-test\",\"awsRegion\":\"us-east-2\",\"ciCacheLocation\":\"s3://ci-caching/\",\"ciCacheFileName\":\"cache.tar.gz\",\"ciProjectDetails\":[{\"gitRepository\":\"https://gitlab.com/devtron/notifier.git\",\"checkoutPath\":\"./notifier-test\",\"commitHash\":\"a6b809c4be87c217feba4af15cf5ebc3cafe21e0\",\"branch\":\"master\",\"gitOptions\":{\"userName\":\"Suraj24\",\"password\":\"Devtron@1234\",\"sshKey\":\"\",\"accessToken\":\"\",\"authMode\":\"\"}},{\"gitRepository\":\"https://gitlab.com/devtron/orchestrator.git\",\"checkoutPath\":\"./orchestrator-test\",\"branch\":\"ci_with_argo\",\"gitOptions\":{\"userName\":\"Suraj24\",\"password\":\"Devtron@1234\",\"sshKey\":\"\",\"accessToken\":\"\",\"authMode\":\"\"}}]}"
	args := os.Args[1]
	ciRequest := &CiRequest{}
	err = json.Unmarshal([]byte(args), ciRequest)
	if err != nil {
		os.Exit(1)
	}

	// Get ci cache
	GetCache(ciRequest)

	// git handling
	CloneAndCheckout(ciRequest)

	// Start docker daemon
	StartDockerDaemon()

	// build
	dest, err := BuildArtifact(ciRequest)
	if err != nil {
		return
	}

	// push to dest
	err = PushArtifact(ciRequest, dest)
	if err != nil {
		return
	}

	// sync cache
	err = SyncCache(ciRequest)
	if err != nil {
		return
	}

	// debug mode
	/*err = exec.Command("tail", "-f", "/dev/null").Run()
	CheckError(err, true)*/

}
