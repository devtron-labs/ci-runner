package main

import (
	"log"
	"os/exec"
	"strings"
	"time"
)

func StartDockerDaemon() {
	dockerdStart := "dockerd --host=unix:///var/run/docker.sock --host=tcp://0.0.0.0:2375 > /usr/local/bin/nohup.out 2>&1 &"
	out, _ := exec.Command("/bin/sh", "-c", dockerdStart).Output()
	log.Println(string(out))
	waitForDockerDaemon(retryCount)
}

func BuildArtifact(ciRequest *CiRequest) (string, error) {
	if ciRequest.DockerImageTag == "" {
		ciRequest.DockerImageTag = "latest"
	}
	// Docker build, tag image and push
	dockerFileLocationDir := ciRequest.DockerFileLocation[:strings.LastIndex(ciRequest.DockerFileLocation, "/")+1]
	dockerBuild := "docker build -f " + ciRequest.DockerFileLocation + " -t " + ciRequest.DockerRepository + " " + dockerFileLocationDir
	log.Println("------> " + dockerBuild)
	dockerBuildCMD := exec.Command("/bin/sh", "-c", dockerBuild)
	err := RunCommand(dockerBuildCMD)
	if err != nil {
		log.Println(err)
		return "", err
	}

	ciRequest.DockerRegistryURL = strings.TrimPrefix(ciRequest.DockerRegistryURL, "https://")
	dest := ciRequest.DockerRegistryURL + "/" + ciRequest.DockerRepository + ":" + ciRequest.DockerImageTag
	dockerTag := "docker tag " + ciRequest.DockerRepository + ":latest" + " " + dest
	log.Println("------> " + dockerTag)
	dockerTagCMD := exec.Command("/bin/sh", "-c", dockerTag)
	err = RunCommand(dockerTagCMD)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return dest, nil
}

func PushArtifact(ciRequest *CiRequest, dest string) error {
	awsLogin := "$(aws ecr get-login --no-include-email --region " + ciRequest.AwsRegion + ")"
	log.Println("------> " + awsLogin)
	awsLoginCmd := exec.Command("/bin/sh", "-c", awsLogin)
	err := RunCommand(awsLoginCmd)
	if err != nil {
		log.Println(err)
		return err
	}

	dockerPush := "docker push " + dest
	log.Println("------> " + dockerPush)
	dockerPushCMD := exec.Command("/bin/sh", "-c", dockerPush)
	err = RunCommand(dockerPushCMD)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
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

func dockerdUpCheck() error {
	dockerCheck := "docker ps"
	dockerCheckCmd := exec.Command("/bin/sh", "-c", dockerCheck)
	err := RunCommand(dockerCheckCmd)
	return err
}

