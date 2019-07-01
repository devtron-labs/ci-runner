package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
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
	getCache(ciRequest)

	// git handling
	cloneAndCheckout(ciRequest)

	// Start docker daemon
	startDockerDaemon()

	// build
	dest,err := buildArtifact(ciRequest)
	if err != nil {
		return
	}

	// push to dest
	err = pushArtifact(ciRequest, dest)
	if err != nil {
		return
	}

	// sync cache
	err = syncCache(ciRequest)
	if err != nil {
		return
	}

	// debug mode
	/*err = exec.Command("tail", "-f", "/dev/null").Run()
	CheckError(err, true)*/

}

func syncCache(ciRequest *CiRequest) error {
	deleteFile(ciRequest.CiCacheFileName)

	// Generate new cache
	log.Println("------> generating new cache")
	tarCmd := exec.Command("tar", "-cf", ciRequest.CiCacheFileName, "/var/lib/docker")
	tarCmd.Dir = "/"
	tarCmd.Run()

	//aws s3 cp cache.tar.gz s3://ci-caching/
	log.Println("------> pushing new cache")
	cachePush := exec.Command("aws", "s3", "cp", ciRequest.CiCacheFileName, ciRequest.CiCacheLocation+ciRequest.CiCacheFileName)
	return runCommand(cachePush)
}

func pushArtifact(ciRequest *CiRequest, dest string) error {
	awsLogin := "$(aws ecr get-login --no-include-email --region " + ciRequest.AwsRegion + ")"
	log.Println("------> " + awsLogin)
	awsLoginCmd := exec.Command("/bin/sh", "-c", awsLogin)
	err := runCommand(awsLoginCmd)
	if err != nil {
		log.Println(err)
		return err
	}

	dockerPush := "docker push " + dest
	log.Println("------> " + dockerPush)
	dockerPushCMD := exec.Command("/bin/sh", "-c", dockerPush)
	err = runCommand(dockerPushCMD)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func buildArtifact(ciRequest *CiRequest) (string, error) {
	if ciRequest.DockerImageTag == "" {
		ciRequest.DockerImageTag = "latest"
	}
	// Docker build, tag image and push
	dockerFileLocationDir := ciRequest.DockerFileLocation[:strings.LastIndex(ciRequest.DockerFileLocation, "/")+1]
	dockerBuild := "docker build -f " + ciRequest.DockerFileLocation + " -t " + ciRequest.DockerRepository + " " + dockerFileLocationDir
	log.Println("------> " + dockerBuild)
	dockerBuildCMD := exec.Command("/bin/sh", "-c", dockerBuild)
	err := runCommand(dockerBuildCMD)
	if err != nil {
		log.Println(err)
		return "", err
	}

	dest := ciRequest.DockerRegistryURL + "/" + ciRequest.DockerRepository + ":" + ciRequest.DockerImageTag
	dockerTag := "docker tag " + ciRequest.DockerRepository + ":latest" + " " + dest
	log.Println("------> " + dockerTag)
	dockerTagCMD := exec.Command("/bin/sh", "-c", dockerTag)
	err = runCommand(dockerTagCMD)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return dest, nil
}

func getCache(ciRequest *CiRequest) {
	ciCacheLocation := ciRequest.CiCacheLocation + ciRequest.CiCacheFileName
	cmd := exec.Command("aws", "s3", "cp", ciCacheLocation, ".")
	err := runCommand(cmd)

	// Extract cache
	if err == nil {
		extractCmd := exec.Command("tar", "-xvf", ciRequest.CiCacheFileName)
		extractCmd.Dir = "/"
		extractCmd.Run()
	}
}

func cloneAndCheckout(ciRequest *CiRequest) error {
	for _, prj := range ciRequest.CiProjectDetails {
		// git clone
		log.Println("------> git cloning " + prj.GitRepository)
		if _, err := os.Stat(prj.CheckoutPath); os.IsNotExist(err) {
			mErr := os.Mkdir(prj.CheckoutPath, os.ModeDir)
			if mErr != nil {
				log.Println(err)
				os.Exit(2)
			}
		}

		var r *git.Repository
		var cErr error
		if prj.Branch == "" || prj.Branch == "master" {
			log.Println("------> " + prj.GitRepository + " cloning master")
			r, cErr = git.PlainClone(prj.CheckoutPath, false, &git.CloneOptions{
				Auth: &http.BasicAuth{
					Username: prj.GitOptions.UserName,
					Password: prj.GitOptions.Password,
				},
				URL:      prj.GitRepository,
				Progress: os.Stdout,
			})
		} else {
			log.Println("------> " + prj.GitRepository + " checking branch " + prj.Branch)
			r, cErr = git.PlainClone(prj.CheckoutPath, false, &git.CloneOptions{
				Auth: &http.BasicAuth{
					Username: prj.GitOptions.UserName,
					Password: prj.GitOptions.Password,
				},
				URL:           prj.GitRepository,
				Progress:      os.Stdout,
				ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", prj.Branch)),
				SingleBranch:  true,
			})
		}
		if cErr != nil {
			log.Println(cErr)
			return cErr
		}

		w, wErr := r.Worktree()
		if wErr != nil {
			log.Println(wErr)
			return wErr
		}

		if prj.CommitHash != "" {
			log.Println("------> " + prj.GitRepository + " git checking out " + prj.CommitHash)
			cErr := CheckoutHash(w, prj.CommitHash)
			if cErr != nil {
				log.Println(cErr)
				return cErr
			}
		}
	}
	return nil
}

func startDockerDaemon() {
	dockerdStart := "dockerd --host=unix:///var/run/docker.sock --host=tcp://0.0.0.0:2375 > /usr/local/bin/nohup.out 2>&1 &"
	out, _ := exec.Command("/bin/sh", "-c", dockerdStart).Output()
	log.Println(string(out))
	waitForDockerDaemon(retryCount)
}

func waitForDockerDaemon(retryCount int) {
	err := dockerdUpCheck()
	retry := 0
	for err != nil {
		if retry == retryCount {
			break
		}
		time.Sleep(1 * time.Second)
		err = dockerdUpCheck()
		retry++
	}
}

func deleteFile(path string) error {
	// delete file
	var err = os.Remove(path)
	if err != nil {
		return err
	}
	return nil
}

func dockerdUpCheck() error {
	dockerCheck := "docker ps"
	dockerCheckCmd := exec.Command("/bin/sh", "-c", dockerCheck)
	err := runCommand(dockerCheckCmd)
	return err
}

func runCommand(cmd *exec.Cmd) error {
	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(os.Stdout, &stdBuffer)
	cmd.Stdout = mw
	cmd.Stderr = mw
	if err := cmd.Run(); err != nil {
		return err
	}
	log.Println(stdBuffer.String())
	return nil
}

func CheckoutHash(workTree *git.Worktree, hash string) error {
	log.Println("checking out hash ", hash)
	err := workTree.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(hash),
	})
	return err
}
