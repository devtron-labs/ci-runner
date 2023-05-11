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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/caarlos0/env"
	"github.com/devtron-labs/ci-runner/util"
)

const (
	DEVTRON_ENV_VAR_PREFIX = "$devtron_env_"
	BUILD_ARG_FLAG         = "--build-arg"
	ROOT_PATH              = "."
)

func StartDockerDaemon(dockerConnection, dockerRegistryUrl, dockerCert, defaultAddressPoolBaseCidr string, defaultAddressPoolSize int, ciRunnerDockerMtuValue int) {
	connection := dockerConnection
	u, err := url.Parse(dockerRegistryUrl)
	if err != nil {
		log.Fatal(err)
	}
	dockerdstart := ""
	defaultAddressPoolFlag := ""
	dockerMtuValueFlag := ""
	if len(defaultAddressPoolBaseCidr) > 0 {
		if defaultAddressPoolSize <= 0 {
			defaultAddressPoolSize = 24
		}
		defaultAddressPoolFlag = fmt.Sprintf("--default-address-pool base=%s,size=%d", defaultAddressPoolBaseCidr, defaultAddressPoolSize)
	}
	if ciRunnerDockerMtuValue > 0 {
		dockerMtuValueFlag = fmt.Sprintf("--mtu=%d", ciRunnerDockerMtuValue)
	}
	if connection == util.INSECURE {
		dockerdstart = fmt.Sprintf("dockerd  %s --insecure-registry %s --host=unix:///var/run/docker.sock %s --host=tcp://0.0.0.0:2375 > /usr/local/bin/nohup.out 2>&1 &", defaultAddressPoolFlag, u.Host, dockerMtuValueFlag)
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
		dockerdstart = fmt.Sprintf("dockerd %s --host=unix:///var/run/docker.sock %s --host=tcp://0.0.0.0:2375 > /usr/local/bin/nohup.out 2>&1 &", defaultAddressPoolFlag, dockerMtuValueFlag)
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

type EnvironmentVariables struct {
	ShowDockerBuildCmdInLogs bool `env:"SHOW_DOCKER_BUILD_ARGS" envDefault:"true"`
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
	envVars := &EnvironmentVariables{}
	err = env.Parse(envVars)
	if err != nil {
		log.Println("Error while parsing environment variables", err)
	}
	if ciRequest.DockerImageTag == "" {
		ciRequest.DockerImageTag = "latest"
	}
	ciBuildConfig := ciRequest.CiBuildConfig
	// Docker build, tag image and push
	dockerFileLocationDir := ciRequest.CheckoutPath
	log.Println(util.DEVTRON, " docker file location: ", dockerFileLocationDir)

	dest, err := BuildDockerImagePath(ciRequest)
	if err != nil {
		return "", err
	}
	if ciBuildConfig.CiBuildType == SELF_DOCKERFILE_BUILD_TYPE || ciBuildConfig.CiBuildType == MANAGED_DOCKERFILE_BUILD_TYPE {
		dockerBuild := "docker build "
		if ciRequest.CacheInvalidate && ciRequest.IsPvcMounted {
			dockerBuild = dockerBuild + "--no-cache"
		}
		dockerBuildConfig := ciBuildConfig.DockerBuildConfig
		useBuildx := dockerBuildConfig.TargetPlatform != ""
		dockerBuildxBuild := "docker buildx build "
		if useBuildx {
			if ciRequest.CacheInvalidate && ciRequest.IsPvcMounted {
				dockerBuild = dockerBuildxBuild + "--no-cache --platform " + dockerBuildConfig.TargetPlatform + " "
			} else {
				dockerBuild = dockerBuildxBuild + "--platform " + dockerBuildConfig.TargetPlatform + " "
			}
		}
		dockerBuildFlags := make(map[string]string)
		dockerBuildArgsMap := dockerBuildConfig.Args
		for k, v := range dockerBuildArgsMap {
			flagKey := fmt.Sprintf("%s %s", BUILD_ARG_FLAG, k)
			if strings.HasPrefix(v, DEVTRON_ENV_VAR_PREFIX) {
				valueFromEnv := os.Getenv(strings.TrimPrefix(v, DEVTRON_ENV_VAR_PREFIX))
				dockerBuildFlags[flagKey] = fmt.Sprintf("=\"%s\"", valueFromEnv)
			} else {
				dockerBuildFlags[flagKey] = fmt.Sprintf("=%s", v)
			}
		}
		dockerBuildOptionsMap := dockerBuildConfig.DockerBuildOptions
		for k, v := range dockerBuildOptionsMap {
			flagKey := "--" + k
			if strings.HasPrefix(v, DEVTRON_ENV_VAR_PREFIX) {
				valueFromEnv := os.Getenv(strings.TrimPrefix(v, DEVTRON_ENV_VAR_PREFIX))
				dockerBuildFlags[flagKey] = fmt.Sprintf("=%s", valueFromEnv)
			} else {
				dockerBuildFlags[flagKey] = fmt.Sprintf("=%s", v)
			}
		}
		for key, value := range dockerBuildFlags {
			dockerBuild = dockerBuild + " " + key + value
		}
		if dockerBuildConfig.BuildContext == "" {
			dockerBuildConfig.BuildContext = ROOT_PATH
		}
		dockerBuildConfig.BuildContext = path.Join(ROOT_PATH, dockerBuildConfig.BuildContext)
		if useBuildx {
			err = installAllSupportedPlatforms()
			if err != nil {
				return "", err
			}

			err = createBuildxBuilder()
			if err != nil {
				return "", err
			}

			log.Println(" -----> Setting up cache directory for Buildx")
			oldCacheBuildxPath := util.LOCAL_BUILDX_LOCATION + "/old"
			localCachePath := util.LOCAL_BUILDX_CACHE_LOCATION
			err = setupCacheForBuildx(localCachePath, oldCacheBuildxPath)
			if err != nil {
				return "", err
			}
			oldCacheBuildxPath = oldCacheBuildxPath + "/cache"
			manifestLocation := util.LOCAL_BUILDX_LOCATION + "/manifest.json"
			dockerBuild = fmt.Sprintf("%s -f %s --network host -t %s --push %s --cache-to=type=local,dest=%s,mode=max --cache-from=type=local,src=%s --allow network.host --allow security.insecure --metadata-file %s", dockerBuild, dockerBuildConfig.DockerfilePath, dest, dockerBuildConfig.BuildContext, localCachePath, oldCacheBuildxPath, manifestLocation)
		} else {
			dockerBuild = fmt.Sprintf("%s -f %s --network host -t %s %s", dockerBuild, dockerBuildConfig.DockerfilePath, ciRequest.DockerRepository, dockerBuildConfig.BuildContext)
		}
		if envVars.ShowDockerBuildCmdInLogs {
			log.Println("Starting docker build : ", dockerBuild)
		} else {
			log.Println("Docker build started..")
		}
		err = executeCmd(dockerBuild)
		if err != nil {
			return "", err
		}

		if !useBuildx {
			err = tagDockerBuild(ciRequest.DockerRepository, dest)
			if err != nil {
				return "", err
			}
		}
	} else if ciBuildConfig.CiBuildType == BUILDPACK_BUILD_TYPE {
		buildPackParams := ciRequest.CiBuildConfig.BuildPackConfig
		projectPath := buildPackParams.ProjectPath
		if projectPath == "" || !strings.HasPrefix(projectPath, "./") {
			projectPath = "./" + projectPath
		}
		handleLanguageVersion(projectPath, buildPackParams)
		buildPackCmd := fmt.Sprintf("pack build %s --path %s --builder %s", dest, projectPath, buildPackParams.BuilderId)
		BuildPackArgsMap := buildPackParams.Args
		for k, v := range BuildPackArgsMap {
			buildPackCmd = buildPackCmd + " --env " + k + "=" + v
		}

		if len(buildPackParams.BuildPacks) > 0 {
			for _, buildPack := range buildPackParams.BuildPacks {
				buildPackCmd = buildPackCmd + " --buildpack " + buildPack
			}
		}
		log.Println(" -----> " + buildPackCmd)
		err = executeCmd(buildPackCmd)
		if err != nil {
			return "", err
		}
		builderRmCmdString := "docker image rm " + buildPackParams.BuilderId
		builderRmCmd := exec.Command("/bin/sh", "-c", builderRmCmdString)
		err := builderRmCmd.Run()
		if err != nil {
			return "", err
		}
	}

	return dest, nil
}

func handleLanguageVersion(projectPath string, buildpackConfig *BuildPackConfig) {
	fileData, err := os.ReadFile("/buildpack.json")
	if err != nil {
		log.Println("error occurred while reading buildpack json", err)
		return
	}
	var buildpackDataArray []*BuildpackVersionConfig
	err = json.Unmarshal(fileData, &buildpackDataArray)
	if err != nil {
		log.Println("error occurred while reading buildpack json", string(fileData))
		return
	}
	language := buildpackConfig.Language
	//languageVersion := buildpackConfig.LanguageVersion
	buildpackEnvArgs := buildpackConfig.Args
	languageVersion, present := buildpackEnvArgs["DEVTRON_LANG_VERSION"]
	if !present {
		return
	}
	var matchedBuildpackConfig *BuildpackVersionConfig
	for _, versionConfig := range buildpackDataArray {
		builderPrefix := versionConfig.BuilderPrefix
		configLanguage := versionConfig.Language
		builderId := buildpackConfig.BuilderId
		if strings.HasPrefix(builderId, builderPrefix) && strings.ToLower(language) == configLanguage {
			matchedBuildpackConfig = versionConfig
			break
		}
	}
	if matchedBuildpackConfig != nil {
		fileName := matchedBuildpackConfig.FileName
		finalPath := filepath.Join(projectPath, "./"+fileName)
		_, err := os.Stat(finalPath)
		fileNotExists := errors.Is(err, os.ErrNotExist)
		if fileNotExists {
			file, err := os.Create(finalPath)
			if err != nil {
				fmt.Println("error occurred while creating file at path " + finalPath)
				return
			}
			entryRegex := matchedBuildpackConfig.EntryRegex
			languageEntry := fmt.Sprintf(entryRegex, languageVersion)
			_, err = file.WriteString(languageEntry)
			log.Println(util.DEVTRON, fmt.Sprintf(" file %s created for language %s with version %s", finalPath, language, languageVersion))
		} else if matchedBuildpackConfig.FileOverride {
			log.Println("final Path is ", finalPath)
			ext := filepath.Ext(finalPath)
			if ext == ".json" {
				jqCmd := fmt.Sprintf("jq '.engines.node' %s", finalPath)
				outputBytes, err := exec.Command("/bin/sh", "-c", jqCmd).Output()
				if err != nil {
					log.Println("error occurred while fetching node version", "err", err)
					return
				}
				if strings.TrimSpace(string(outputBytes)) == "null" {
					tmpJsonFile := "./tmp.json"
					versionUpdateCmd := fmt.Sprintf("jq '.engines.node = \"%s\"' %s >%s", languageVersion, finalPath, tmpJsonFile)
					err := executeCmd(versionUpdateCmd)
					if err != nil {
						log.Println("error occurred while inserting node version", "err", err)
						return
					}
					fileReplaceCmd := fmt.Sprintf("mv %s %s", tmpJsonFile, finalPath)
					err = executeCmd(fileReplaceCmd)
					if err != nil {
						log.Println("error occurred while executing cmd ", fileReplaceCmd, "err", err)
						return
					}
				}
			}
		} else {
			log.Println("file already exists, so ignoring version override!!", finalPath)
		}
	}

}

func executeCmd(dockerBuild string) error {
	dockerBuildCMD := exec.Command("/bin/sh", "-c", dockerBuild)
	err := util.RunCommand(dockerBuildCMD)
	if err != nil {
		log.Println(err)
	}
	return err
}

func tagDockerBuild(dockerRepository string, dest string) error {
	dockerTag := "docker tag " + dockerRepository + ":latest" + " " + dest
	log.Println(" -----> " + dockerTag)
	dockerTagCMD := exec.Command("/bin/sh", "-c", dockerTag)
	err := util.RunCommand(dockerTagCMD)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
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

func installAllSupportedPlatforms() error {
	multiPlatformCmd := "docker run --privileged --rm quay.io/devtron/binfmt:stable --install all"
	log.Println(" -----> " + multiPlatformCmd)
	dockerBuildCMD := exec.Command("/bin/sh", "-c", multiPlatformCmd)
	err := util.RunCommand(dockerBuildCMD)
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
