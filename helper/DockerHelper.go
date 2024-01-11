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
	BUILDX_K8S_DRIVER_NAME = "devtron-buildx-builder"
	BUILDX_NODE_NAME       = "devtron-buildx-node-"
)

func StartDockerDaemon(dockerConnection, dockerRegistryUrl, dockerCert, defaultAddressPoolBaseCidr string, defaultAddressPoolSize int, ciRunnerDockerMtuValue int) {
	connection := dockerConnection
	u, err := url.Parse(dockerRegistryUrl)
	if err != nil {
		log.Fatal(err)
	}
	dockerdStart := util.NewCommand()
	dockerdStart.AppendCommand("dockerd")
	if len(defaultAddressPoolBaseCidr) > 0 {
		if defaultAddressPoolSize <= 0 {
			defaultAddressPoolSize = 24
		}
		defaultAddressPoolFlag := fmt.Sprintf("base=%s,size=%d", defaultAddressPoolBaseCidr, defaultAddressPoolSize)
		dockerdStart.AppendCommand("--default-address-pool", defaultAddressPoolFlag)
	}

	if connection == util.INSECURE {
		dockerdStart.AppendCommand("--insecure-registry", u.Host)
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
	}
	dockerdStart.AppendCommand("--host=unix:///var/run/docker.sock")
	if ciRunnerDockerMtuValue > 0 {
		dockerMtuValueFlag := fmt.Sprintf("--mtu=%d", ciRunnerDockerMtuValue)
		dockerdStart.AppendCommand(dockerMtuValueFlag)
	}
	dockerdStart.AppendCommand("--host=tcp://0.0.0.0:2375", ">", "/usr/local/bin/nohup.out", "2>&1", "&")
	out, _ := exec.Command("/bin/sh", dockerdStart.GetCommandToBeExecuted("-c")...).Output()
	log.Println(string(out))
	waitForDockerDaemon(util.RETRYCOUNT)
}

const DOCKER_REGISTRY_TYPE_ECR = "ecr"
const DOCKER_REGISTRY_TYPE_DOCKERHUB = "docker-hub"
const DOCKER_REGISTRY_TYPE_OTHER = "other"
const REGISTRY_TYPE_ARTIFACT_REGISTRY = "artifact-registry"
const REGISTRY_TYPE_GCR = "gcr"
const JSON_KEY_USERNAME = "_json_key"

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

	} else if (dockerCredentials.DockerRegistryType == REGISTRY_TYPE_GCR || dockerCredentials.DockerRegistryType == REGISTRY_TYPE_ARTIFACT_REGISTRY) && username == JSON_KEY_USERNAME {
		// for gcr and artifact registry password is already saved as string in DB
		if strings.HasPrefix(pwd, "'") {
			pwd = pwd[1:]
		}
		if strings.HasSuffix(pwd, "'") {
			pwd = pwd[:len(pwd)-1]
		}
	}
	awsLoginCmd := exec.Command("/bin/sh", "-c", "docker", "login", "-u", username, "-p", pwd, dockerCredentials.DockerRegistryURL)
	err := util.RunCommand(awsLoginCmd)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println("Docker login successful with username ", username, " on docker registry URL ", dockerCredentials.DockerRegistryURL)
	return nil
}
func BuildArtifact(ciRequest *CommonWorkflowRequest) (string, error) {
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
		dockerBuild := util.NewCommand("docker", "build")
		if ciRequest.CacheInvalidate && ciRequest.IsPvcMounted {
			dockerBuild.AppendCommand("--no-cache")
		}
		dockerBuildConfig := ciBuildConfig.DockerBuildConfig

		isTargetPlatformSet := dockerBuildConfig.TargetPlatform != ""
		useBuildx := dockerBuildConfig.CheckForBuildX()
		dockerBuildxBuild := util.NewCommand("docker", "buildx", "build")
		if useBuildx {
			dockerBuild = dockerBuildxBuild
			if ciRequest.CacheInvalidate && ciRequest.IsPvcMounted {
				dockerBuild.AppendCommand("--no-cache")
			}
			if isTargetPlatformSet {
				dockerBuild.AppendCommand("--platform", dockerBuildConfig.TargetPlatform)
			}
		}
		dockerBuildArgsMap := dockerBuildConfig.Args
		for k, v := range dockerBuildArgsMap {
			dockerBuild.AppendCommand(BUILD_ARG_FLAG)
			if strings.HasPrefix(v, DEVTRON_ENV_VAR_PREFIX) {
				valueFromEnv := os.Getenv(strings.TrimPrefix(v, DEVTRON_ENV_VAR_PREFIX))
				dockerBuildArg := fmt.Sprintf("%s=\"%s\"", strings.TrimSpace(k), strings.TrimSpace(valueFromEnv))
				dockerBuild.AppendCommand(dockerBuildArg)
			} else {
				dockerBuildArg := fmt.Sprintf("%s=%s", strings.TrimSpace(k), strings.TrimSpace(v))
				dockerBuild.AppendCommand(dockerBuildArg)
			}
		}
		dockerBuildOptionsMap := dockerBuildConfig.DockerBuildOptions
		for k, v := range dockerBuildOptionsMap {
			dockerBuildFlag := fmt.Sprintf("--%s", strings.TrimSpace(k))
			if strings.HasPrefix(v, DEVTRON_ENV_VAR_PREFIX) {
				valueFromEnv := os.Getenv(strings.TrimPrefix(v, DEVTRON_ENV_VAR_PREFIX))
				dockerBuildFlag += fmt.Sprintf("=%s", strings.TrimSpace(valueFromEnv))
			} else {
				dockerBuildFlag += fmt.Sprintf("=%s", strings.TrimSpace(v))
			}
			dockerBuild.AppendCommand(dockerBuildFlag)
		}

		if !ciRequest.EnableBuildContext || dockerBuildConfig.BuildContext == "" {
			dockerBuildConfig.BuildContext = ROOT_PATH
		}
		dockerBuildConfig.BuildContext = path.Join(ROOT_PATH, dockerBuildConfig.BuildContext)
		if useBuildx {
			err := checkAndCreateDirectory(util.LOCAL_BUILDX_LOCATION)
			if err != nil {
				log.Println(util.DEVTRON, " error in creating LOCAL_BUILDX_LOCATION ", util.LOCAL_BUILDX_LOCATION)
				return "", err
			}
			useBuildxK8sDriver, eligibleK8sDriverNodes := dockerBuildConfig.CheckForBuildXK8sDriver()
			if useBuildxK8sDriver {
				err = createBuildxBuilderWithK8sDriver(eligibleK8sDriverNodes, ciRequest.PipelineId, ciRequest.WorkflowId)
				if err != nil {
					log.Println(util.DEVTRON, " error in creating buildxDriver , err : ", err.Error())
					return "", err
				}
			} else {
				err = createBuildxBuilderForMultiArchBuild()
				if err != nil {
					return "", err
				}
			}

			cacheEnabled := (ciRequest.IsPvcMounted || ciRequest.BlobStorageConfigured)
			oldCacheBuildxPath, localCachePath := "", ""

			if cacheEnabled {
				log.Println(" -----> Setting up cache directory for Buildx")
				oldCacheBuildxPath = util.LOCAL_BUILDX_LOCATION + "/old"
				localCachePath = util.LOCAL_BUILDX_CACHE_LOCATION
				err = setupCacheForBuildx(localCachePath, oldCacheBuildxPath)
				if err != nil {
					return "", err
				}
				oldCacheBuildxPath = oldCacheBuildxPath + "/cache"
			}
			createBuildxBuildCommand(dockerBuild, cacheEnabled, oldCacheBuildxPath, localCachePath, dest, dockerBuildConfig)
		} else {
			dockerBuild.AppendCommand("-f", dockerBuildConfig.DockerfilePath, "--network host", "-t", ciRequest.DockerRepository, dockerBuildConfig.BuildContext)
		}
		if envVars.ShowDockerBuildCmdInLogs {
			log.Println("Starting docker build : ", dockerBuild.PrintCommand())
		} else {
			log.Println("Docker build started..")
		}
		err = executeCmd(dockerBuild)
		if err != nil {
			return "", err
		}

		if useBuildK8sDriver, eligibleK8sDriverNodes := dockerBuildConfig.CheckForBuildXK8sDriver(); useBuildK8sDriver {
			err = CleanBuildxK8sDriver(eligibleK8sDriverNodes)
			if err != nil {
				log.Println(util.DEVTRON, " error in cleaning buildx K8s driver ", " err: ", err)
			}
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
		buildPackCmd := util.NewCommand("pack", "build", dest, "--path", projectPath, "--builder", buildPackParams.BuilderId)
		BuildPackArgsMap := buildPackParams.Args
		for k, v := range BuildPackArgsMap {
			buildPackCmd.AppendCommand("--env", fmt.Sprintf("%s=%s", strings.TrimSpace(k), strings.TrimSpace(v)))
		}

		if len(buildPackParams.BuildPacks) > 0 {
			for _, buildPack := range buildPackParams.BuildPacks {
				buildPackCmd.AppendCommand("--buildpack", strings.TrimSpace(buildPack))
			}
		}
		log.Println(" -----> " + buildPackCmd.PrintCommand())
		err = executeCmd(buildPackCmd)
		if err != nil {
			return "", err
		}
		builderRmCmd := exec.Command("/bin/sh", "-c", "docker", "image", "rm", buildPackParams.BuilderId)
		err := builderRmCmd.Run()
		if err != nil {
			return "", err
		}
	}

	return dest, nil
}

func createBuildxBuildCommand(dockerBuild *util.CommandType, cacheEnabled bool, oldCacheBuildxPath, localCachePath, dest string, dockerBuildConfig *DockerBuildConfig) {
	dockerBuild.AppendCommand("-f", dockerBuildConfig.DockerfilePath, "-t", dest, "--push", dockerBuildConfig.BuildContext, "--network", "host", "--allow network.host", "--allow security.insecure")
	if cacheEnabled {
		cacheDest := fmt.Sprintf("--cache-to=type=local,dest=%s,mode=max", localCachePath)
		cacheSrc := fmt.Sprintf("--cache-from=type=local,src=%s", oldCacheBuildxPath)
		dockerBuild.AppendCommand(cacheDest, cacheSrc)
	}

	provenanceFlag := dockerBuildConfig.GetProvenanceFlag()
	dockerBuild.AppendCommand(provenanceFlag)
	manifestLocation := util.LOCAL_BUILDX_LOCATION + "/manifest.json"
	dockerBuild.AppendCommand("--metadata-file", manifestLocation)
	return
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
				jqCmd := util.NewCommand("jq", "'.engines.node'", finalPath)
				outputBytes, err := exec.Command("/bin/sh", jqCmd.GetCommandToBeExecuted("-c")...).Output()
				if err != nil {
					log.Println("error occurred while fetching node version", "err", err)
					return
				}
				if strings.TrimSpace(string(outputBytes)) == "null" {
					languageVersionFlag := fmt.Sprintf("'.engines.node = \"%s\"'", languageVersion)
					tmpJsonFile := "./tmp.json"
					tmpJsonFileFlag := fmt.Sprintf(">%s", tmpJsonFile)
					versionUpdateCmd := util.NewCommand("jq", languageVersionFlag, finalPath, tmpJsonFileFlag)
					err := executeCmd(versionUpdateCmd)
					if err != nil {
						log.Println("error occurred while inserting node version", "err", err)
						return
					}
					fileReplaceCmd := util.NewCommand("mv", tmpJsonFile, finalPath)
					err = executeCmd(fileReplaceCmd)
					if err != nil {
						log.Println("error occurred while executing cmd ", fileReplaceCmd.PrintCommand(), "err", err)
						return
					}
				}
			}
		} else {
			log.Println("file already exists, so ignoring version override!!", finalPath)
		}
	}

}

// executeCmd uses CLI to run git command and it is prone to script injection |
// Don'ts:
// 1- Never concatenate the whole cmd args into a single string and pass it as exec.Command(name, fmt.Sprintf("--flag1 %s --flag2 %s  --flag3 %s", value1, value2, value3)) |
// DOs:
// 1- Break the command to name and []args as exec.Command(name, []arg...)
// 2- Use strings.TrimSpace() to build an user defined flags; e.g: fmt.Sprintf("--%s", strings.TrimSpace(userDefinedFlag))
// 3- In case a single arg contains multiple user defined inputs, then use fmt.Sprintf(); exec.Command(name, "--flag=", fmt.Sprintf("key1=%s,key2=%s,key3=%s", userDefinedArg-1, userDefinedArg-2, userDefinedArg-2))
func executeCmd(dockerBuild *util.CommandType) error {
	dockerBuildCMD := exec.Command("/bin/sh", dockerBuild.GetCommandToBeExecuted("-c")...)
	err := util.RunCommand(dockerBuildCMD)
	if err != nil {
		log.Println(err)
	}
	return err
}

func tagDockerBuild(dockerRepository string, dest string) error {
	dockerTagCmd := util.NewCommand("docker", "tag", fmt.Sprintf("%s:latest", dockerRepository), dest)
	log.Println(" -----> " + dockerTagCmd.PrintCommand())
	dockerTagCMD := exec.Command("/bin/sh", dockerTagCmd.GetCommandToBeExecuted("-c")...)
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
	copyContentCmd := exec.Command("/bin/sh", "-c", "cp", "-R", localCachePath, oldCacheBuildxPath)
	err = util.RunCommand(copyContentCmd)
	if err != nil {
		log.Println(err)
		return err
	}

	cleanContentCmd := exec.Command("/bin/sh", "-c", "rm", "-rf", localCachePath, "/*")
	err = util.RunCommand(cleanContentCmd)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func createBuildxBuilder() error {
	multiPlatformCmd := util.NewCommand("docker", "buildx", "create", "--use", "--buildkitd-flags", "'--allow-insecure-entitlement network.host --allow-insecure-entitlement security.insecure'")
	log.Println(" -----> " + multiPlatformCmd.PrintCommand())
	dockerBuildCMD := exec.Command("/bin/sh", multiPlatformCmd.GetCommandToBeExecuted("-c")...)
	err := util.RunCommand(dockerBuildCMD)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func installAllSupportedPlatforms() error {
	multiPlatformCmd := util.NewCommand("docker", "run", "--privileged", "--rm", "quay.io/devtron/binfmt:stable", "--install", "all")
	log.Println(" -----> " + multiPlatformCmd.PrintCommand())
	dockerBuildCMD := exec.Command("/bin/sh", multiPlatformCmd.GetCommandToBeExecuted("-c")...)
	err := util.RunCommand(dockerBuildCMD)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func checkAndCreateDirectory(localCachePath string) error {
	pathCreateCommand := exec.Command("/bin/sh", "-c", "mkdir", "-p", localCachePath)
	err := util.RunCommand(pathCreateCommand)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func BuildDockerImagePath(ciRequest *CommonWorkflowRequest) (string, error) {
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
	dockerPush := fmt.Sprintf("docker push %s", dest)
	log.Println("-----> " + dockerPush)
	dockerPushCMD := exec.Command("/bin/sh", "-c", "docker", "push", dest)
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
	dockerPullCmd := exec.Command("/bin/sh", "-c", "docker", "pull", dest)
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

func createBuildxBuilderForMultiArchBuild() error {
	err := installAllSupportedPlatforms()
	if err != nil {
		return err
	}
	err = createBuildxBuilder()
	if err != nil {
		return err
	}
	return nil
}

func createBuildxBuilderWithK8sDriver(builderNodes []map[string]string, ciPipelineId, ciWorkflowId int) error {

	if len(builderNodes) == 0 {
		return errors.New("atleast one node is expected for builder with kubernetes driver")
	}
	defaultNodeOpts := builderNodes[0]

	buildxCreate := getBuildxK8sDriverCmd(defaultNodeOpts, ciPipelineId, ciWorkflowId)
	buildxCreate.AppendCommand("--use")
	err, errBuf := runCmd(buildxCreate)
	if err != nil {
		fmt.Println(util.DEVTRON, "buildxCreate : ", buildxCreate.PrintCommand(), " err : ", err, " error : ", errBuf.String(), "\n ")
		return err
	}

	//appending other nodes to the builder,except default node ,since we already added it
	for i := 1; i < len(builderNodes); i++ {
		nodeOpts := builderNodes[i]
		appendNode := getBuildxK8sDriverCmd(nodeOpts, ciPipelineId, ciWorkflowId)
		appendNode.AppendCommand("--append")

		err, errBuf = runCmd(appendNode)
		if err != nil {
			fmt.Println(util.DEVTRON, " appendNode : ", appendNode.PrintCommand(), " err : ", err, " error : ", errBuf.String(), "\n ")
			return err
		}
	}

	return nil
}

func CleanBuildxK8sDriver(nodes []map[string]string) error {
	nodeNames := make([]string, 0)
	for _, nOptsMp := range nodes {
		if nodeName, ok := nOptsMp["node"]; ok && nodeName != "" {
			nodeNames = append(nodeNames, nodeName)
		}
	}
	err, errBuf := leaveNodesFromBuildxK8sDriver(nodeNames)
	if err != nil {
		log.Println(util.DEVTRON, " error in deleting nodes created by ci-runner , err : ", errBuf.String())
		return err
	}
	log.Println(util.DEVTRON, "successfully cleaned up buildx k8s driver")
	return nil
}

func leaveNodesFromBuildxK8sDriver(nodeNames []string) (error, *bytes.Buffer) {
	var err error
	var errBuf *bytes.Buffer
	defer func() {
		removeCmd := util.NewCommand("docker", "buildx", "rm", BUILDX_K8S_DRIVER_NAME)
		err, errBuf = runCmd(removeCmd)
		if err != nil {
			log.Println(util.DEVTRON, "error in removing docker buildx err : ", errBuf.String())
		}
	}()
	for _, node := range nodeNames {
		k8sDriverNameFlag := fmt.Sprintf("--name=%s", BUILDX_K8S_DRIVER_NAME)
		k8sDriverNodeFlag := fmt.Sprintf("--node=%s", node, BUILDX_K8S_DRIVER_NAME)
		k8sDriverLeaveNodeCmd := util.NewCommand("docker", "buildx", "create", k8sDriverNameFlag, k8sDriverNodeFlag, "--leave")
		err, errBuf = runCmd(k8sDriverLeaveNodeCmd)
		if err != nil {
			log.Println(util.DEVTRON, "error in leaving node : ", errBuf.String())
			return err, errBuf
		}
	}
	return err, errBuf
}

func runCmd(cmd *util.CommandType) (error, *bytes.Buffer) {
	fmt.Println(util.DEVTRON, " cmd : ", cmd.PrintCommand())
	builderCreateCmd := exec.Command("/bin/sh", cmd.GetCommandToBeExecuted("-c")...)
	errBuf := &bytes.Buffer{}
	builderCreateCmd.Stderr = errBuf
	err := builderCreateCmd.Run()
	return err, errBuf
}

func getBuildxK8sDriverCmd(driverOpts map[string]string, ciPipelineId, ciWorkflowId int) *util.CommandType {
	nodeName := driverOpts["node"]
	if nodeName == "" {
		nodeName = BUILDX_NODE_NAME + fmt.Sprintf("%v-%v", ciPipelineId, ciWorkflowId) + util.Generate(3) //need this to generate unique name for builder node in same builder.
	}
	k8sDriverNameFlag := fmt.Sprintf("--name=%s", BUILDX_K8S_DRIVER_NAME)
	k8sDriverNodeFlag := fmt.Sprintf("--node=%s", nodeName)
	buildxCreateCmd := util.NewCommand("docker", "buildx", "create", "--buildkitd-flags", "'--allow-insecure-entitlement network.host --allow-insecure-entitlement security.insecure'", k8sDriverNameFlag, "--driver=kubernetes", k8sDriverNodeFlag, "--bootstrap")

	platforms := driverOpts["platform"]
	if platforms != "" {
		buildxPlatformFlag := fmt.Sprintf("--platform=%s", platforms)
		buildxCreateCmd.AppendCommand(buildxPlatformFlag)
	}
	if len(driverOpts["driverOptions"]) > 0 {
		buildxDriverOptions := fmt.Sprintf("'--driver-opt=%s'", driverOpts["driverOptions"])
		buildxCreateCmd.AppendCommand(buildxDriverOptions)
	}
	return buildxCreateCmd
}

func StopDocker() error {
	out, err := exec.Command("docker", "ps", "-a", "-q").Output()
	if err != nil {
		return err
	}
	if len(out) > 0 {
		log.Println(util.DEVTRON, " -----> stopping docker container")
		stopCmd := exec.Command("/bin/sh", "-c", "docker", "stop", "-t", "5", "$(docker ps -a -q)")
		err := util.RunCommand(stopCmd)
		log.Println(util.DEVTRON, " -----> stopped docker container")
		if err != nil {
			log.Fatal(err)
			return err
		}
		removeContainerCmds := util.NewCommand("docker", "rm", "-v", "-f", "$(docker ps -a -q)")
		log.Println(util.DEVTRON, " -----> removing docker container")
		removeContainerCmd := exec.Command("/bin/sh", removeContainerCmds.GetCommandToBeExecuted("-c")...)
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
	dockerCheck := util.NewCommand("docker", "ps")
	dockerCheckCmd := exec.Command("/bin/sh", dockerCheck.GetCommandToBeExecuted("-c")...)
	err := dockerCheckCmd.Run()
	return err
}

func ValidBuildxK8sDriverOptions(ciRequest *CommonWorkflowRequest) (bool, []map[string]string) {
	valid := ciRequest != nil && ciRequest.CiBuildConfig != nil && ciRequest.CiBuildConfig.DockerBuildConfig != nil
	if valid {
		return ciRequest.CiBuildConfig.DockerBuildConfig.CheckForBuildXK8sDriver()
	}
	return false, nil
}
