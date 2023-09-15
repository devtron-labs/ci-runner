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
	fmt.Println("Docker login successful with username ", username, " on docker registry URL ", dockerCredentials.DockerRegistryURL)
	awsLoginCmd := exec.Command("/bin/sh", "-c", dockerLogin)
	err := util.RunCommand(awsLoginCmd)
	if err != nil {
		log.Println(err)
		return err
	}
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
		dockerBuild := "docker build "
		if ciRequest.CacheInvalidate && ciRequest.IsPvcMounted {
			dockerBuild = dockerBuild + "--no-cache "
		}
		dockerBuildConfig := ciBuildConfig.DockerBuildConfig

		isTargetPlatformSet := dockerBuildConfig.TargetPlatform != ""
		useBuildx := dockerBuildConfig.CheckForBuildX()
		dockerBuildxBuild := "docker buildx build "
		if useBuildx {
			if ciRequest.CacheInvalidate && ciRequest.IsPvcMounted {
				dockerBuild = dockerBuildxBuild + "--no-cache "
			} else {
				dockerBuild = dockerBuildxBuild + " "
			}
			if isTargetPlatformSet {
				dockerBuild += "--platform " + dockerBuildConfig.TargetPlatform + " "
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
				err = createBuildxBuilderWithK8sDriver(eligibleK8sDriverNodes)
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

			dockerBuild = getBuildxBuildCommand(cacheEnabled, dockerBuild, oldCacheBuildxPath, localCachePath, dest, dockerBuildConfig)
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

func getBuildxBuildCommand(cacheEnabled bool, dockerBuild, oldCacheBuildxPath, localCachePath, dest string, dockerBuildConfig *DockerBuildConfig) string {
	dockerBuild = fmt.Sprintf("%s -f %s -t %s --push %s --network host --allow network.host --allow security.insecure", dockerBuild, dockerBuildConfig.DockerfilePath, dest, dockerBuildConfig.BuildContext)
	if cacheEnabled {
		dockerBuild = fmt.Sprintf("%s --cache-to=type=local,dest=%s,mode=max --cache-from=type=local,src=%s", dockerBuild, localCachePath, oldCacheBuildxPath)
	}

	provenanceFlag := dockerBuildConfig.GetProvenanceFlag()
	dockerBuild = fmt.Sprintf("%s %s", dockerBuild, provenanceFlag)
	manifestLocation := util.LOCAL_BUILDX_LOCATION + "/manifest.json"
	dockerBuild = fmt.Sprintf("%s --metadata-file %s", dockerBuild, manifestLocation)

	return dockerBuild
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

func createBuildxBuilderWithK8sDriver(builderNodes []map[string]string) error {

	if len(builderNodes) == 0 {
		return errors.New("atleast one node is expected for builder with kubernetes driver")
	}
	defaultNodeOpts := builderNodes[0]

	buildxCreate := getBuildxK8sDriverCmd(defaultNodeOpts)
	buildxCreate = fmt.Sprintf("%s %s", buildxCreate, "--use")

	err, errBuf := runCmd(buildxCreate)
	if err != nil {
		fmt.Println(util.DEVTRON, "buildxCreate : ", buildxCreate, " err : ", err, " error : ", errBuf.String(), "\n ")
		return err
	}

	//appending other nodes to the builder,except default node ,since we already added it
	for i := 1; i < len(builderNodes); i++ {
		nodeOpts := builderNodes[i]
		appendNode := getBuildxK8sDriverCmd(nodeOpts)
		appendNode = fmt.Sprintf("%s %s", appendNode, "--append")

		err, errBuf = runCmd(appendNode)
		if err != nil {
			fmt.Println(util.DEVTRON, " appendNode : ", appendNode, " err : ", err, " error : ", errBuf.String(), "\n ")
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
		removeCmd := fmt.Sprintf("docker buildx rm %s", BUILDX_K8S_DRIVER_NAME)
		err, errBuf = runCmd(removeCmd)
		if err != nil {
			log.Println(util.DEVTRON, "error in removing docker buildx err : ", errBuf.String())
		}
	}()
	for _, node := range nodeNames {
		cmds := fmt.Sprintf("docker buildx create --name=%s --node=%s --leave", BUILDX_K8S_DRIVER_NAME, node)
		err, errBuf = runCmd(cmds)
		if err != nil {
			log.Println(util.DEVTRON, "error in leaving node : ", errBuf.String())
			return err, errBuf
		}
	}
	return err, errBuf
}

func runCmd(cmd string) (error, *bytes.Buffer) {
	fmt.Println(util.DEVTRON, " cmd : ", cmd)
	builderCreateCmd := exec.Command("/bin/sh", "-c", cmd)
	errBuf := &bytes.Buffer{}
	builderCreateCmd.Stderr = errBuf
	err := builderCreateCmd.Run()
	return err, errBuf
}

func getBuildxK8sDriverCmd(driverOpts map[string]string) string {
	buildxCreate := "docker buildx create --buildkitd-flags '--allow-insecure-entitlement network.host --allow-insecure-entitlement security.insecure' --name=%s --driver=kubernetes --node=%s --bootstrap "
	nodeName := driverOpts["node"]
	if nodeName == "" {
		nodeName = BUILDX_NODE_NAME + util.Generate(5)
	}
	buildxCreate = fmt.Sprintf(buildxCreate, BUILDX_K8S_DRIVER_NAME, nodeName)
	platforms := driverOpts["platform"]
	if platforms != "" {
		buildxCreate += " --platform=%s "
		buildxCreate = fmt.Sprintf(buildxCreate, platforms)
	}
	if len(driverOpts["driverOptions"]) > 0 {
		buildxCreate += " '--driver-opt=%s' "
		buildxCreate = fmt.Sprintf(buildxCreate, driverOpts["driverOptions"])
	}
	return buildxCreate
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

func ValidBuildxK8sDriverOptions(ciRequest *CommonWorkflowRequest) (bool, []map[string]string) {
	valid := ciRequest != nil && ciRequest.CiBuildConfig != nil && ciRequest.CiBuildConfig.DockerBuildConfig != nil
	if valid {
		return ciRequest.CiBuildConfig.DockerBuildConfig.CheckForBuildXK8sDriver()
	}
	return false, nil
}
