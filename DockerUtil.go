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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
)

func StartDockerDaemon(dockerConnection, dockerRegistryUrl, dockerCert string) {
	connection := dockerConnection
	u, err := url.Parse(dockerRegistryUrl)
	fmt.Println(u)
	if err != nil {
		log.Fatal(err)
	}
	if connection == "insecure" {
		dockerdstart := "dockerd --insecure-registry " + u.Host + " --host=unix:///var/run/docker.sock --host=tcp://0.0.0.0:2375 > /usr/local/bin/nohup.out 2>&1 &"
		out, _ := exec.Command("/bin/sh", "-c", dockerdstart).Output()
		log.Println(string(out))
		waitForDockerDaemon(retryCount)
	} else {
		if connection == "secure-with-cert" {
			os.MkdirAll("/etc/docker/certs.d/"+u.Host, os.ModePerm)
			f, err := os.Create("/etc/docker/certs.d/" + u.Host + "/ca.crt")

			if err != nil {
				log.Fatal(err)
			}

			defer f.Close()

			_, err2 := f.WriteString(dockerCert)

			if err2 != nil {
				log.Fatal(err2)
			}

		}
		dockerdStart := "dockerd --host=unix:///var/run/docker.sock --host=tcp://0.0.0.0:2375 > /usr/local/bin/nohup.out 2>&1 &"
		out, _ := exec.Command("/bin/sh", "-c", dockerdStart).Output()
		log.Println(string(out))
		waitForDockerDaemon(retryCount)
	}
}

const DOCKER_REGISTRY_TYPE_ECR = "ecr"
const DOCKER_REGISTRY_TYPE_DOCKERHUB = "docker-hub"
const DOCKER_REGISTRY_TYPE_OTHER = "other"

type DockerCredentials struct {
	DockerUsername, DockerPassword, AwsRegion, AccessKey, SecretKey, DockerRegistryURL, DockerRegistryType string
}

func DockerLogin(dockerCredentials *DockerCredentials) error {
	username := dockerCredentials.DockerUsername
	pwd := dockerCredentials.DockerPassword
	if dockerCredentials.DockerRegistryType == DOCKER_REGISTRY_TYPE_ECR {
		svc := ecr.New(session.New(&aws.Config{
			Region:      &dockerCredentials.AwsRegion,
			Credentials: credentials.NewStaticCredentials(dockerCredentials.AccessKey, dockerCredentials.SecretKey, ""),
		}))
		input := &ecr.GetAuthorizationTokenInput{}
		authData, err := svc.GetAuthorizationToken(input)
		if err != nil {
			log.Println(err)
			return err
		}
		// decode token
		token := authData.AuthorizationData[0].AuthorizationToken
		decodedToken, err := base64.StdEncoding.DecodeString(*token)
		if err != nil {
			log.Println(err)
			return err
		}
		credsSlice := strings.Split(string(decodedToken), ":")
		username = credsSlice[0]
		pwd = credsSlice[1]
	}
	dockerLogin := "docker login -u " + username + " -p " + pwd + " " + dockerCredentials.DockerRegistryURL
	log.Println(devtron, " -----> "+dockerLogin)
	awsLoginCmd := exec.Command("/bin/sh", "-c", dockerLogin)
	err := RunCommand(awsLoginCmd)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
func BuildArtifact(ciRequest *CiRequest) (string, error) {
	err := DockerLogin(&DockerCredentials{
		DockerUsername:     ciRequest.DockerUsername,
		DockerPassword:     ciRequest.DockerPassword,
		AwsRegion:          ciRequest.AwsRegion,
		AccessKey:          ciRequest.AccessKey,
		SecretKey:          ciRequest.SecretKey,
		DockerRegistryURL:  ciRequest.DockerRegistryURL,
		DockerRegistryType: ciRequest.DockerRegistryType,
	})
	if err != nil {
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
	if DOCKER_REGISTRY_TYPE_DOCKERHUB == ciRequest.DockerRegistryType {
		dest = ciRequest.DockerRepository + ":" + ciRequest.DockerImageTag
	} else {
		u, err := url.Parse(ciRequest.DockerRegistryURL)
		if err != nil {
			log.Println("not a valid docker repository url")
			return "", err
		}
		u.Path = path.Join(u.Path, "/", ciRequest.DockerRepository)
		dockerRegistryURL := u.Host + u.Path
		dest = dockerRegistryURL + ":" + ciRequest.DockerImageTag
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
	//awsLogin := "$(aws ecr get-login --no-include-email --region " + ciRequest.AwsRegion + ")"
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
	out, err := exec.Command("docker", "ps", "-a", "-q").Output()
	if err != nil {
		return err
	}
	if len(out) > 0 {
		stopCmdS := "docker stop -t 5 $(docker ps -a -q)"
		log.Println(devtron, " -----> stopping docker container")
		stopCmd := exec.Command("/bin/sh", "-c", stopCmdS)
		err := RunCommand(stopCmd)
		log.Println(devtron, " -----> stopped docker container")
		if err != nil {
			log.Fatal(err)
			return err
		}
		removeContainerCmds := "docker rm -v -f $(docker ps -a -q)"
		log.Println(devtron, " -----> removing docker container")
		removeContainerCmd := exec.Command("/bin/sh", "-c", removeContainerCmds)
		err = RunCommand(removeContainerCmd)
		log.Println(devtron, " -----> removed docker container")
		if err != nil {
			log.Fatal(err)
			return err
		}
	}
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
