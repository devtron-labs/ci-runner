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

package helper

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
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/devtron-labs/ci-runner/util"
)

func StartDockerDaemon(dockerConnection, dockerRegistryUrl, dockerCert, defaultAddressPoolBaseCidr string, defaultAddressPoolSize int) {
	connection := dockerConnection
	u, err := url.Parse(dockerRegistryUrl)
	if err != nil {
		log.Fatal(err)
	}
	dockerdstart := ""
	defaultAddressPoolFlag := ""
	if len(defaultAddressPoolBaseCidr) > 0 {
		if defaultAddressPoolSize <= 0 {
			defaultAddressPoolSize = 24
		}
		defaultAddressPoolFlag = fmt.Sprintf("--default-address-pool base=%s,size=%d", defaultAddressPoolBaseCidr, defaultAddressPoolSize)
	}
	if connection == util.INSECURE {
		dockerdstart = fmt.Sprintf("dockerd  %s --insecure-registry %s --host=unix:///var/run/docker.sock --host=tcp://0.0.0.0:2375 > /usr/local/bin/nohup.out 2>&1 &", defaultAddressPoolFlag, u.Host)
		util.LogStage("Insecure Registry")
	} else {
		if connection == util.SECUREWITHCERT {
			os.MkdirAll(fmt.Sprintf("/etc/docker/certs.d/%s", u.Host), os.ModePerm)
			f, err := os.Create(fmt.Sprintf("/etc/docker/certs.d/%s/ca.crt", u.Host))

			if err != nil {
				log.Fatal(err)
			}

			defer f.Close()

			_, err2 := f.WriteString(dockerCert)

			if err2 != nil {
				log.Fatal(err2)
			}
			util.LogStage("Secure with Cert")
		}
		dockerdstart = fmt.Sprintf("dockerd %s --host=unix:///var/run/docker.sock --host=tcp://0.0.0.0:2375 > /usr/local/bin/nohup.out 2>&1 &", defaultAddressPoolFlag)
	}
	out, _ := exec.Command("/bin/sh", "-c", dockerdstart).Output()
	log.Println(string(out))
	waitForDockerDaemon(util.RETRYCOUNT)
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
		accessKey, secretKey := dockerCredentials.AccessKey, dockerCredentials.SecretKey
		//fmt.Printf("accessKey %s, secretKey %s\n", accessKey, secretKey)

		var creds *credentials.Credentials

		if len(dockerCredentials.AccessKey) == 0 || len(dockerCredentials.SecretKey) == 0 {
			//fmt.Println("empty accessKey or secretKey")
			sess, err := session.NewSession(&aws.Config{
				Region: &dockerCredentials.AwsRegion,
			})
			if err != nil {
				log.Println(err)
				return err
			}
			creds = ec2rolecreds.NewCredentials(sess)
		} else {
			creds = credentials.NewStaticCredentials(accessKey, secretKey, "")
		}
		sess, err := session.NewSession(&aws.Config{
			Region:      &dockerCredentials.AwsRegion,
			Credentials: creds,
		})
		if err != nil {
			log.Println(err)
			return err
		}
		svc := ecr.New(sess)
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
	log.Println(util.DEVTRON, " -----> "+dockerLogin)
	awsLoginCmd := exec.Command("/bin/sh", "-c", dockerLogin)
	err := util.RunCommand(awsLoginCmd)
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
	log.Println(util.DEVTRON, " docker file location: ", dockerFileLocationDir)

	dockerBuild := "docker build "
	useBuildx := ciRequest.DockerBuildTargetPlatform != ""
	if useBuildx {
		dockerBuild = "docker buildx build --platform " + ciRequest.DockerBuildTargetPlatform + " "
	}
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

	if useBuildx {
		err := installAllSupportedPlatforms(err)
		if err != nil {
			return "", err
		}

		err2 := createBuildxBuilder()
		if err2 != nil {
			return "", err2
		}
	}
	dest, err := BuildDockerImagePath(ciRequest)
	if err != nil {
		return "", err
	}
	if useBuildx {
		log.Println(" -----> Setting up cache directory for Buildx")
		oldCacheBuildxPath := util.LOCAL_BUILDX_LOCATION + "/old"
		localCachePath := util.LOCAL_BUILDX_CACHE_LOCATION
		err := setupCacheForBuildx(localCachePath, oldCacheBuildxPath)
		if err != nil {
			return "", err
		}
		oldCacheBuildxPath = oldCacheBuildxPath + "/cache"
		manifestLocation := util.LOCAL_BUILDX_LOCATION + "/manifest.json"
		dockerBuild = fmt.Sprintf("%s -f %s --network host -t %s --push . --cache-to=type=local,dest=%s,mode=max --cache-from=type=local,src=%s --allow network.host --allow security.insecure --metadata-file %s", dockerBuild, ciRequest.DockerFileLocation, dest, localCachePath, oldCacheBuildxPath, manifestLocation)
	} else {
		dockerBuild = fmt.Sprintf("%s -f %s --network host -t %s .", dockerBuild, ciRequest.DockerFileLocation, ciRequest.DockerRepository)
	}
	log.Println(" -----> " + dockerBuild)

	dockerBuildCMD := exec.Command("/bin/sh", "-c", dockerBuild)
	err = util.RunCommand(dockerBuildCMD)
	if err != nil {
		log.Println(err)
		return "", err
	}

	if !useBuildx {
		dockerTag := "docker tag " + ciRequest.DockerRepository + ":latest" + " " + dest
		log.Println(" -----> " + dockerTag)
		dockerTagCMD := exec.Command("/bin/sh", "-c", dockerTag)
		err = util.RunCommand(dockerTagCMD)
		if err != nil {
			log.Println(err)
			return "", err
		}

		log.Println(" -----> copying content")

		// copy content
		dockerCopyCommand := "docker run --rm --entrypoint \"/bin/sh\" -v /devtroncd/.m2:/root/.m3 " + dest + " -c \"rsync -tr /root/.m2/repository /root/.m3\""
		dockerCopyExec := exec.Command("/bin/sh", "-c", dockerCopyCommand)
		err = util.RunCommand(dockerCopyExec)
		if err != nil {
			log.Println(err)
			return "", err
		}

		log.Println(" -----> copied content")

	}
	return dest, nil
}

func setupCacheForBuildx(localCachePath string, oldCacheBuildxPath string) error {
	err := checkAndCreateDirectory(localCachePath)
	if err != nil {
		return err
	}
	err = checkAndCreateDirectory(oldCacheBuildxPath)
	if err != nil {
		return err
	}
	copyContent := "cp -R " + localCachePath + " " + oldCacheBuildxPath
	copyContentCmd := exec.Command("/bin/sh", "-c", copyContent)
	err = util.RunCommand(copyContentCmd)
	if err != nil {
		log.Println(err)
		return err
	}

	cleanContent := "rm -rf " + localCachePath + "/*"
	cleanContentCmd := exec.Command("/bin/sh", "-c", cleanContent)
	err = util.RunCommand(cleanContentCmd)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func createBuildxBuilder() error {
	multiPlatformCmd := "docker buildx create --use --buildkitd-flags '--allow-insecure-entitlement network.host --allow-insecure-entitlement security.insecure'"
	log.Println(" -----> " + multiPlatformCmd)
	dockerBuildCMD := exec.Command("/bin/sh", "-c", multiPlatformCmd)
	err := util.RunCommand(dockerBuildCMD)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func installAllSupportedPlatforms(err error) error {
	multiPlatformCmd := "docker run --privileged --rm tonistiigi/binfmt --install all"
	log.Println(" -----> " + multiPlatformCmd)
	dockerBuildCMD := exec.Command("/bin/sh", "-c", multiPlatformCmd)
	err = util.RunCommand(dockerBuildCMD)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func checkAndCreateDirectory(localCachePath string) error {
	makeDirCmd := "mkdir -p " + localCachePath
	pathCreateCommand := exec.Command("/bin/sh", "-c", makeDirCmd)
	err := util.RunCommand(pathCreateCommand)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func BuildDockerImagePath(ciRequest *CiRequest) (string, error) {
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
	return dest, nil
}

func PushArtifact(dest string) error {
	//awsLogin := "$(aws ecr get-login --no-include-email --region " + ciRequest.AwsRegion + ")"
	dockerPush := "docker push " + dest
	log.Println("-----> " + dockerPush)
	dockerPushCMD := exec.Command("/bin/sh", "-c", dockerPush)
	err := util.RunCommand(dockerPushCMD)
	if err != nil {
		log.Println(err)
		return err
	}

	//digest := extractDigestUsingPull(dest)
	//log.Println("Digest -----> ", digest)
	//return digest, nil
	return nil
}

func ExtractDigestForBuildx(dest string) (string, error) {

	var digest string
	var err error
	manifestLocation := util.LOCAL_BUILDX_LOCATION + "/manifest.json"
	digest, err = readImageDigestFromManifest(manifestLocation)
	if err != nil {
		log.Println("error occurred while extracting digest from manifest reason ", err)
		err = nil // would extract digest using docker pull cmd
	}
	if digest == "" {
		digest, err = ExtractDigestUsingPull(dest)
	}
	log.Println("Digest -----> ", digest)

	return digest, err
}

func ExtractDigestUsingPull(dest string) (string, error) {
	dockerPull := "docker pull " + dest
	dockerPullCmd := exec.Command("/bin/sh", "-c", dockerPull)
	digest, err := runGetDockerImageDigest(dockerPullCmd)
	if err != nil {
		log.Println(err)
	}
	return digest, err
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

func readImageDigestFromManifest(manifestFilePath string) (string, error) {
	manifestFile, err := ioutil.ReadFile(manifestFilePath)
	if err != nil {
		return "", err
	}
	var data map[string]interface{}
	err = json.Unmarshal(manifestFile, &data)
	if err != nil {
		return "", err
	}
	imageDigest, found := data["containerimage.digest"]
	if !found {
		return "", nil
	}
	return imageDigest.(string), nil
}

func StopDocker() error {
	out, err := exec.Command("docker", "ps", "-a", "-q").Output()
	if err != nil {
		return err
	}
	if len(out) > 0 {
		stopCmdS := "docker stop -t 5 $(docker ps -a -q)"
		log.Println(util.DEVTRON, " -----> stopping docker container")
		stopCmd := exec.Command("/bin/sh", "-c", stopCmdS)
		err := util.RunCommand(stopCmd)
		log.Println(util.DEVTRON, " -----> stopped docker container")
		if err != nil {
			log.Fatal(err)
			return err
		}
		removeContainerCmds := "docker rm -v -f $(docker ps -a -q)"
		log.Println(util.DEVTRON, " -----> removing docker container")
		removeContainerCmd := exec.Command("/bin/sh", "-c", removeContainerCmds)
		err = util.RunCommand(removeContainerCmd)
		log.Println(util.DEVTRON, " -----> removed docker container")
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
	log.Println(util.DEVTRON, " -----> checking docker status")
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
