package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func StartDockerDaemon() {
	dockerdStart := "dockerd --host=unix:///var/run/docker.sock --host=tcp://0.0.0.0:2375 > /usr/local/bin/nohup.out 2>&1 &"
	out, _ := exec.Command("/bin/sh", "-c", dockerdStart).Output()
	log.Println(string(out))
	waitForDockerDaemon(retryCount)
}

func BuildArtifact(ciRequest *CiRequest) (string, error) {
	username := ciRequest.DockerUsername
	pwd := ciRequest.DockerPassword

	if ciRequest.DockerRegistryType == "ecr" {
		svc := ecr.New(session.New(&aws.Config{
			Region:      &ciRequest.AwsRegion,
			Credentials: credentials.NewStaticCredentials(ciRequest.AccessKey, ciRequest.SecretKey, ""),
		}))

		input := &ecr.GetAuthorizationTokenInput{}
		authData, err := svc.GetAuthorizationToken(input)
		if err != nil {
			log.Println(err)
			return "", err
		}

		// decode token
		token := authData.AuthorizationData[0].AuthorizationToken
		decodedToken, err := base64.StdEncoding.DecodeString(*token)
		if err != nil {
			log.Println(err)
			return "", err
		}

		credsSlice := strings.Split(string(decodedToken), ":")
		username = credsSlice[0]
		pwd = credsSlice[1]
	}

	dockerLogin := "docker login -u " + username + " -p " + pwd + " " + ciRequest.DockerRegistryURL
	log.Println(devtron, " -----> "+dockerLogin)
	awsLoginCmd := exec.Command("/bin/sh", "-c", dockerLogin)
	err := RunCommand(awsLoginCmd)
	if err != nil {
		log.Println(err)
		return "", err
	}

	if ciRequest.DockerImageTag == "" {
		ciRequest.DockerImageTag = "latest"
	}
	// Docker build, tag image and push
	dockerFileLocationDir := ciRequest.DockerFileLocation[:strings.LastIndex(ciRequest.DockerFileLocation, "/")+1]
	log.Println(devtron, " docker file location: ", dockerFileLocationDir)

	dockerBuild := "docker build "
	if ciRequest.DockerBuildArgs != "" {
		dockerBuildArgsMap := make(map[string]string)
		err := json.Unmarshal([]byte(ciRequest.DockerBuildArgs), &dockerBuildArgsMap)
		if err != nil {
			log.Println("err", err)
			return "", err
		}
		for k, v := range dockerBuildArgsMap {
			dockerBuild = dockerBuild + " --build-arg " + k + "=" + v
		}
	}
	dockerBuild = fmt.Sprintf("%s -f %s --network host -t %s .", dockerBuild, ciRequest.DockerFileLocation, ciRequest.DockerRepository)
	log.Println(" -----> " + dockerBuild)

	dockerBuildCMD := exec.Command("/bin/sh", "-c", dockerBuild)
	err = RunCommand(dockerBuildCMD)
	if err != nil {
		log.Println(err)
		return "", err
	}

	dest := ""
	if "ecr" == ciRequest.DockerRegistryType {
		dockerRegistryURL := strings.TrimPrefix(ciRequest.DockerRegistryURL, "https://")
		dest = dockerRegistryURL + "/" + ciRequest.DockerRepository + ":" + ciRequest.DockerImageTag
	} else {
		dest = ciRequest.DockerRepository + ":" + ciRequest.DockerImageTag
	}

	dockerTag := "docker tag " + ciRequest.DockerRepository + ":latest" + " " + dest
	log.Println(" -----> " + dockerTag)
	dockerTagCMD := exec.Command("/bin/sh", "-c", dockerTag)
	err = RunCommand(dockerTagCMD)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return dest, nil
}

func PushArtifact(ciRequest *CiRequest, dest string) (string, error) {
	//awsLogin := "$(aws ecr get-login --no-include-email --region " + ciRequest.AwsRegion + ")"ss
	dockerPush := "docker push " + dest
	log.Println("-----> " + dockerPush)
	dockerPushCMD := exec.Command("/bin/sh", "-c", dockerPush)
	err := RunCommand(dockerPushCMD)
	if err != nil {
		log.Println(err)
		return "", err
	}
	dockerPull := "docker pull " + dest
	dockerPullCmd := exec.Command("/bin/sh", "-c", dockerPull)
	digest, err := runGetDockerImageDigest(dockerPullCmd)
	if err != nil {
		log.Println(err)
		return "", err
	}
	log.Println("Digest -----> ", digest)
	return digest, nil
}

func runGetDockerImageDigest(cmd *exec.Cmd) (string, error) {
	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(os.Stdout, &stdBuffer)
	cmd.Stdout = mw
	cmd.Stderr = mw
	if err := cmd.Run(); err != nil {
		return "", err
	}
	output := stdBuffer.String()
	outArr := strings.Split(output, "\n")
	var digest string
	for _, s := range outArr {
		if strings.HasPrefix(s, "Digest: ") {
			digest = s[strings.Index(s, "sha256:"):]
		}

	}
	return digest, nil
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
	log.Println(devtron, " -----> checking docker status")
	DockerdUpCheck()
	return nil
}

func waitForDockerDaemon(retryCount int) {
	err := DockerdUpCheck()
	retry := 0
	for err != nil {
		if retry == retryCount {
			break
		}
		time.Sleep(1 * time.Second)
		err = DockerdUpCheck()
		retry++
	}
}

func DockerdUpCheck() error {
	dockerCheck := "docker ps"
	dockerCheckCmd := exec.Command("/bin/sh", "-c", dockerCheck)
	err := dockerCheckCmd.Run()
	return err
}
