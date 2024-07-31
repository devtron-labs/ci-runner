/*
 * Copyright (c) 2024. Devtron Inc.
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
 */

package helper

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/caarlos0/env"
	cicxt "github.com/devtron-labs/ci-runner/executor/context"
	"github.com/devtron-labs/ci-runner/util"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	DEVTRON_ENV_VAR_PREFIX = "$devtron_env_"
	BUILD_ARG_FLAG         = "--build-arg"
	ROOT_PATH              = "."
	BUILDX_K8S_DRIVER_NAME = "devtron-buildx-builder"
	BUILDX_NODE_NAME       = "devtron-buildx-node-"
)

type DockerHelper interface {
	StartDockerDaemon(commonWorkflowRequest *CommonWorkflowRequest)
	DockerLogin(ciContext cicxt.CiContext, dockerCredentials *DockerCredentials) error
	BuildArtifact(ciRequest *CommonWorkflowRequest) (string, error)
	StopDocker(ciContext cicxt.CiContext) error
	PushArtifact(ciContext cicxt.CiContext, dest string) error
	ExtractDigestForBuildx(dest string) (string, error)
	CleanBuildxK8sDriver(ciContext cicxt.CiContext, nodes []map[string]string) error
	GetDestForNatsEvent(commonWorkflowRequest *CommonWorkflowRequest, dest string) (string, error)
}

type DockerHelperImpl struct {
	DockerCommandEnv []string
	cmdExecutor      CommandExecutor
}

func NewDockerHelperImpl(cmdExecutor CommandExecutor) *DockerHelperImpl {
	return &DockerHelperImpl{
		DockerCommandEnv: os.Environ(),
		cmdExecutor:      cmdExecutor,
	}
}

func (impl *DockerHelperImpl) GetDestForNatsEvent(commonWorkflowRequest *CommonWorkflowRequest, dest string) (string, error) {
	return dest, nil
}

func (impl *DockerHelperImpl) StartDockerDaemon(commonWorkflowRequest *CommonWorkflowRequest) {
	startDockerDaemon := func() error {
		connection := commonWorkflowRequest.DockerConnection
		dockerRegistryUrl := commonWorkflowRequest.IntermediateDockerRegistryUrl
		registryUrl, err := util.ParseUrl(dockerRegistryUrl)
		if err != nil {
			return err
		}
		host := registryUrl.Host
		dockerdstart := ""
		defaultAddressPoolFlag := ""
		dockerMtuValueFlag := ""
		if len(commonWorkflowRequest.DefaultAddressPoolBaseCidr) > 0 {
			if commonWorkflowRequest.DefaultAddressPoolSize <= 0 {
				commonWorkflowRequest.DefaultAddressPoolSize = 24
			}
			defaultAddressPoolFlag = fmt.Sprintf("--default-address-pool base=%s,size=%d", commonWorkflowRequest.DefaultAddressPoolBaseCidr, commonWorkflowRequest.DefaultAddressPoolSize)
		}
		if commonWorkflowRequest.CiBuildDockerMtuValue > 0 {
			dockerMtuValueFlag = fmt.Sprintf("--mtu=%d", commonWorkflowRequest.CiBuildDockerMtuValue)
		}
		if connection == util.INSECURE {
			dockerdstart = fmt.Sprintf("dockerd  %s --insecure-registry %s --host=unix:///var/run/docker.sock %s --host=tcp://0.0.0.0:2375 > /usr/local/bin/nohup.out 2>&1 &", defaultAddressPoolFlag, host, dockerMtuValueFlag)
			util.LogStage("Insecure Registry")
		} else {
			if connection == util.SECUREWITHCERT {
				os.MkdirAll(fmt.Sprintf("/etc/docker/certs.d/%s", host), os.ModePerm)
				f, err := os.Create(fmt.Sprintf("/etc/docker/certs.d/%s/ca.crt", host))

				if err != nil {
					return err
				}

				defer f.Close()

				_, err2 := f.WriteString(commonWorkflowRequest.DockerCert)

				if err2 != nil {
					return err
				}
				util.LogStage("Secure with Cert")
			}
			dockerdstart = fmt.Sprintf("dockerd %s --host=unix:///var/run/docker.sock %s --host=tcp://0.0.0.0:2375 > /usr/local/bin/nohup.out 2>&1 &", defaultAddressPoolFlag, dockerMtuValueFlag)
		}
		cmd := impl.GetCommandToExecute(dockerdstart)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Println("failed to start docker daemon")
			return err
		}
		log.Println("docker daemon started ", string(out))
		err = impl.waitForDockerDaemon(util.DOCKER_PS_START_WAIT_SECONDS)
		if err != nil {
			return err
		}
		return err
	}

	if err := util.ExecuteWithStageInfoLog(util.DOCKER_DAEMON, startDockerDaemon); err != nil {
		log.Fatal(err)
	}
	return
}

const DOCKER_REGISTRY_TYPE_ECR = "ecr"
const DOCKER_REGISTRY_TYPE_DOCKERHUB = "docker-hub"
const DOCKER_REGISTRY_TYPE_OTHER = "other"
const REGISTRY_TYPE_ARTIFACT_REGISTRY = "artifact-registry"
const REGISTRY_TYPE_GCR = "gcr"
const JSON_KEY_USERNAME = "_json_key"
const CacheModeMax = "max"
const CacheModeMin = "min"

type DockerCredentials struct {
	DockerUsername, DockerPassword, AwsRegion, AccessKey, SecretKey, DockerRegistryURL, DockerRegistryType string
}

type EnvironmentVariables struct {
	ShowDockerBuildCmdInLogs bool `env:"SHOW_DOCKER_BUILD_ARGS" envDefault:"true"`
}

func (impl *DockerHelperImpl) GetCommandToExecute(cmd string) *exec.Cmd {
	execCmd := exec.Command("/bin/sh", "-c", cmd)
	execCmd.Env = append(execCmd.Env, impl.DockerCommandEnv...)
	return execCmd
}

func (impl *DockerHelperImpl) DockerLogin(ciContext cicxt.CiContext, dockerCredentials *DockerCredentials) error {
	performDockerLogin := func() error {
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
		host := dockerCredentials.DockerRegistryURL
		dockerLogin := fmt.Sprintf("docker login -u '%s' -p '%s' '%s' ", username, pwd, host)

		awsLoginCmd := impl.GetCommandToExecute(dockerLogin)
		err := impl.cmdExecutor.RunCommand(ciContext, awsLoginCmd)
		if err != nil {
			log.Println(err)
			return err
		}
		log.Println("Docker login successful with username ", username, " on docker registry URL ", dockerCredentials.DockerRegistryURL)
		return nil
	}

	return util.ExecuteWithStageInfoLog(util.DOCKER_LOGIN_STAGE, performDockerLogin)
}

func (impl *DockerHelperImpl) BuildArtifact(ciRequest *CommonWorkflowRequest) (string, error) {
	ciContext := cicxt.BuildCiContext(context.Background(), ciRequest.EnableSecretMasking)
	err := impl.DockerLogin(ciContext, &DockerCredentials{
		DockerUsername:     ciRequest.DockerUsername,
		DockerPassword:     ciRequest.DockerPassword,
		AwsRegion:          ciRequest.AwsRegion,
		AccessKey:          ciRequest.AccessKey,
		SecretKey:          ciRequest.SecretKey,
		DockerRegistryURL:  ciRequest.IntermediateDockerRegistryUrl,
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

		useBuildx := dockerBuildConfig.CheckForBuildX()
		dockerBuildxBuild := "docker buildx build "
		if useBuildx {
			if ciRequest.CacheInvalidate && ciRequest.IsPvcMounted {
				dockerBuild = dockerBuildxBuild + "--no-cache "
			} else {
				dockerBuild = dockerBuildxBuild + " "
			}

		}
		dockerBuildFlags := getDockerBuildFlagsMap(dockerBuildConfig)
		for key, value := range dockerBuildFlags {
			dockerBuild = dockerBuild + " " + key + value
		}
		if !ciRequest.EnableBuildContext || dockerBuildConfig.BuildContext == "" {
			dockerBuildConfig.BuildContext = ROOT_PATH
		}
		dockerBuildConfig.BuildContext = path.Join(ROOT_PATH, dockerBuildConfig.BuildContext)

		dockerfilePath := getDockerfilePath(ciBuildConfig, ciRequest.CheckoutPath)
		var buildxExportCacheFunc func() = nil
		if useBuildx {
			setupBuildxBuilder := func() error {
				err := impl.checkAndCreateDirectory(ciContext, util.LOCAL_BUILDX_LOCATION)
				if err != nil {
					log.Println(util.DEVTRON, " error in creating LOCAL_BUILDX_LOCATION ", util.LOCAL_BUILDX_LOCATION)
					return err
				}
				useBuildxK8sDriver, eligibleK8sDriverNodes := dockerBuildConfig.CheckForBuildXK8sDriver()
				if useBuildxK8sDriver {
					err = impl.createBuildxBuilderWithK8sDriver(ciContext, eligibleK8sDriverNodes, ciRequest.PipelineId, ciRequest.WorkflowId)
					if err != nil {
						log.Println(util.DEVTRON, " error in creating buildxDriver , err : ", err.Error())
						return err
					}
				} else {
					err = impl.createBuildxBuilderForMultiArchBuild(ciContext)
					if err != nil {
						return err
					}
				}
				return nil
			}

			if err = util.ExecuteWithStageInfoLog(util.SETUP_BUILDX_BUILDER, setupBuildxBuilder); err != nil {
				return "", err
			}

			cacheEnabled := (ciRequest.IsPvcMounted || ciRequest.BlobStorageConfigured)
			oldCacheBuildxPath, localCachePath := "", ""

			if cacheEnabled {
				log.Println(" -----> Setting up cache directory for Buildx")
				oldCacheBuildxPath = util.LOCAL_BUILDX_LOCATION + "/old"
				localCachePath = util.LOCAL_BUILDX_CACHE_LOCATION
				err = impl.setupCacheForBuildx(ciContext, localCachePath, oldCacheBuildxPath)
				if err != nil {
					return "", err
				}
				oldCacheBuildxPath = oldCacheBuildxPath + "/cache"
			}

			// need to export the cache after the build if k8s driver mode is enabled.
			// when we use k8s driver, if we give export cache flag in the build command itself then all the k8s driver nodes will push the cache to same location.
			// then we will endup with having any one of the node cache in the end and we cannot use this cache for all the platforms in subsequent builds.

			// so we will export the cache after build for all the platforms independently at different locations.
			// refer buildxExportCacheFunc

			multiNodeK8sDriver := useBuildxK8sDriver && len(eligibleK8sDriverNodes) > 1
			exportBuildxCacheAfterBuild := ciRequest.AsyncBuildxCacheExport && multiNodeK8sDriver
			dockerBuild, buildxExportCacheFunc = impl.getBuildxBuildCommand(ciContext, exportBuildxCacheAfterBuild, cacheEnabled, ciRequest.BuildxCacheModeMin, dockerBuild, oldCacheBuildxPath, localCachePath, dest, dockerBuildConfig, dockerfilePath)
		} else {
			dockerBuild = fmt.Sprintf("%s -f %s --network host -t %s %s", dockerBuild, dockerfilePath, ciRequest.DockerRepository, dockerBuildConfig.BuildContext)
		}

		buildImageStage := func() error {
			if envVars.ShowDockerBuildCmdInLogs {
				log.Println("Starting docker build : ", dockerBuild)
			} else {
				log.Println("Docker build started..")
			}
			err = impl.executeCmd(ciContext, dockerBuild)
			if err != nil {
				return err
			}
			return nil
		}

		if err = util.ExecuteWithStageInfoLog(util.DOCKER_BUILD, buildImageStage); err != nil {
			return "", nil
		}

		// todo: gireesh
		if buildxExportCacheFunc != nil {
			buildxExportCacheFunc()
		}

		if useBuildK8sDriver, eligibleK8sDriverNodes := dockerBuildConfig.CheckForBuildXK8sDriver(); useBuildK8sDriver {

			buildxCleanupSatge := func() error {
				err = impl.CleanBuildxK8sDriver(ciContext, eligibleK8sDriverNodes)
				if err != nil {
					log.Println(util.DEVTRON, " error in cleaning buildx K8s driver ", " err: ", err)
				}
				return nil
			}

			// do not need to handle the below error
			util.ExecuteWithStageInfoLog(util.CLEANUP_BUILDX_BUILDER, buildxCleanupSatge)
		}

		if !useBuildx {
			err = impl.tagDockerBuild(ciContext, ciRequest.DockerRepository, dest)
			if err != nil {
				return "", err
			}
		} else {
			return dest, nil
		}
	} else if ciBuildConfig.CiBuildType == BUILDPACK_BUILD_TYPE {

		buildPacksImageBuildStage := func() error {
			buildPackParams := ciRequest.CiBuildConfig.BuildPackConfig
			projectPath := buildPackParams.ProjectPath
			if projectPath == "" || !strings.HasPrefix(projectPath, "./") {
				projectPath = "./" + projectPath
			}
			impl.handleLanguageVersion(ciContext, projectPath, buildPackParams)
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
			err = impl.executeCmd(ciContext, buildPackCmd)
			if err != nil {
				return err
			}
			builderRmCmdString := "docker image rm " + buildPackParams.BuilderId
			builderRmCmd := impl.GetCommandToExecute(builderRmCmdString)
			err := builderRmCmd.Run()
			if err != nil {
				return err
			}
			return nil
		}

		if err = util.ExecuteWithStageInfoLog(util.BUILD_PACK_BUILD, buildPacksImageBuildStage); err != nil {
			return "", nil
		}

	}

	return dest, nil
}

func getDockerBuildFlagsMap(dockerBuildConfig *DockerBuildConfig) map[string]string {
	dockerBuildFlags := make(map[string]string)
	dockerBuildArgsMap := dockerBuildConfig.Args
	for k, v := range dockerBuildArgsMap {
		flagKey := fmt.Sprintf("%s %s", BUILD_ARG_FLAG, k)
		dockerBuildFlags[flagKey] = parseDockerFlagParam(v)
	}
	dockerBuildOptionsMap := dockerBuildConfig.DockerBuildOptions
	for k, v := range dockerBuildOptionsMap {
		flagKey := "--" + k
		dockerBuildFlags[flagKey] = parseDockerFlagParam(v)
	}
	return dockerBuildFlags
}

func parseDockerFlagParam(param string) string {
	value := param
	if strings.HasPrefix(param, DEVTRON_ENV_VAR_PREFIX) {
		value = os.Getenv(strings.TrimPrefix(param, DEVTRON_ENV_VAR_PREFIX))
	}

	return wrapSingleOrDoubleQuotedValue(value)
}

func wrapSingleOrDoubleQuotedValue(value string) string {
	if strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`) {
		unquotedString := strings.Trim(value, `'`)
		return fmt.Sprintf(`='%s'`, unquotedString)
	}

	unquotedString := strings.Trim(value, `"`)

	return fmt.Sprintf(`="%s"`, unquotedString)
}

func getDockerfilePath(CiBuildConfig *CiBuildConfigBean, checkoutPath string) string {
	var dockerFilePath string
	if CiBuildConfig.CiBuildType == MANAGED_DOCKERFILE_BUILD_TYPE {
		dockerFilePath = GetSelfManagedDockerfilePath(checkoutPath)
	} else {
		dockerFilePath = CiBuildConfig.DockerBuildConfig.DockerfilePath
	}
	return dockerFilePath
}

// getBuildxExportCacheFunc  will concurrently execute the given export cache commands
func (impl *DockerHelperImpl) getBuildxExportCacheFunc(ciContext cicxt.CiContext, exportCacheCmds map[string]string) func() {
	exportCacheFunc := func() {
		// run export cache cmd for buildx
		if len(exportCacheCmds) > 0 {
			log.Println("exporting build caches...")
			wg := sync.WaitGroup{}
			wg.Add(len(exportCacheCmds))
			for platform, exportCacheCmd := range exportCacheCmds {
				go func(platform, exportCacheCmd string) {
					defer wg.Done()
					log.Println("exporting build cache, platform : ", platform)
					log.Println(exportCacheCmd)
					err := impl.executeCmd(ciContext, exportCacheCmd)
					if err != nil {
						log.Println("error in exporting ", "err : ", err)
						return
					}
				}(platform, exportCacheCmd)
			}
			wg.Wait()
		}
	}
	return exportCacheFunc
}

// getExportCacheCmds will return build commands exclusively for exporting cache for all the given target platforms.
func getExportCacheCmds(targetPlatforms, dockerBuild, localCachePath string, useCacheMin bool) map[string]string {

	cacheMode := CacheModeMax
	if useCacheMin {
		cacheMode = CacheModeMin
	}

	cacheCmd := "%s --platform=%s --cache-to=type=local,dest=%s,mode=" + cacheMode
	platforms := strings.Split(targetPlatforms, ",")

	exportCacheCmds := make(map[string]string)
	for _, platform := range platforms {
		cachePath := strings.Join(strings.Split(platform, "/"), "-")
		exportCacheCmds[platform] = fmt.Sprintf(cacheCmd, dockerBuild, platform, localCachePath+"/"+cachePath)
	}
	return exportCacheCmds
}

func getSourceCaches(targetPlatforms, oldCachePathLocation string) string {
	cacheCmd := " --cache-from=type=local,src=%s "
	platforms := strings.Split(targetPlatforms, ",")
	allCachePaths := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		cachePath := strings.Join(strings.Split(platform, "/"), "-")
		allCachePaths = append(allCachePaths, fmt.Sprintf(cacheCmd, oldCachePathLocation+"/"+cachePath))
	}
	return strings.Join(allCachePaths, " ")
}

func (impl *DockerHelperImpl) getBuildxBuildCommandV2(ciContext cicxt.CiContext, cacheEnabled bool, useCacheMin bool, dockerBuild, oldCacheBuildxPath, localCachePath, dest string, dockerBuildConfig *DockerBuildConfig, dockerfilePath string) (string, func()) {
	dockerBuild = fmt.Sprintf("%s %s -f %s --network host --allow network.host --allow security.insecure", dockerBuild, dockerBuildConfig.BuildContext, dockerfilePath)
	exportCacheCmds := make(map[string]string)

	provenanceFlag := dockerBuildConfig.GetProvenanceFlag()
	dockerBuild = fmt.Sprintf("%s %s", dockerBuild, provenanceFlag)

	// separate out export cache and source cache cmds here
	isTargetPlatformSet := dockerBuildConfig.TargetPlatform != ""
	if isTargetPlatformSet {
		if cacheEnabled {
			exportCacheCmds = getExportCacheCmds(dockerBuildConfig.TargetPlatform, dockerBuild, localCachePath, useCacheMin)
		}

		dockerBuild = fmt.Sprintf("%s --platform %s", dockerBuild, dockerBuildConfig.TargetPlatform)
	}

	if cacheEnabled {
		dockerBuild = fmt.Sprintf("%s %s", dockerBuild, getSourceCaches(dockerBuildConfig.TargetPlatform, oldCacheBuildxPath))
	}

	manifestLocation := util.LOCAL_BUILDX_LOCATION + "/manifest.json"
	dockerBuild = fmt.Sprintf("%s -t %s --push --metadata-file %s", dockerBuild, dest, manifestLocation)

	return dockerBuild, impl.getBuildxExportCacheFunc(ciContext, exportCacheCmds)
}

func (impl *DockerHelperImpl) getBuildxBuildCommandV1(cacheEnabled bool, useCacheMin bool, dockerBuild, oldCacheBuildxPath, localCachePath, dest string, dockerBuildConfig *DockerBuildConfig, dockerfilePath string) (string, func()) {

	cacheMode := CacheModeMax
	if useCacheMin {
		cacheMode = CacheModeMin
	}
	dockerBuild = fmt.Sprintf("%s -f %s -t %s --push %s --network host --allow network.host --allow security.insecure", dockerBuild, dockerfilePath, dest, dockerBuildConfig.BuildContext)
	if cacheEnabled {
		dockerBuild = fmt.Sprintf("%s --cache-to=type=local,dest=%s,mode=%s --cache-from=type=local,src=%s", dockerBuild, localCachePath, cacheMode, oldCacheBuildxPath)
	}

	provenanceFlag := dockerBuildConfig.GetProvenanceFlag()
	dockerBuild = fmt.Sprintf("%s %s", dockerBuild, provenanceFlag)
	manifestLocation := util.LOCAL_BUILDX_LOCATION + "/manifest.json"
	dockerBuild = fmt.Sprintf("%s --metadata-file %s", dockerBuild, manifestLocation)

	return dockerBuild, nil
}

func (impl *DockerHelperImpl) getBuildxBuildCommand(ciContext cicxt.CiContext, exportBuildxCacheAfterBuild bool, cacheEnabled bool, useCacheMin bool, dockerBuild, oldCacheBuildxPath, localCachePath, dest string, dockerBuildConfig *DockerBuildConfig, dockerfilePath string) (string, func()) {
	if exportBuildxCacheAfterBuild {
		return impl.getBuildxBuildCommandV2(ciContext, cacheEnabled, useCacheMin, dockerBuild, oldCacheBuildxPath, localCachePath, dest, dockerBuildConfig, dockerfilePath)
	}
	return impl.getBuildxBuildCommandV1(cacheEnabled, useCacheMin, dockerBuild, oldCacheBuildxPath, localCachePath, dest, dockerBuildConfig, dockerfilePath)
}

func (impl *DockerHelperImpl) handleLanguageVersion(ciContext cicxt.CiContext, projectPath string, buildpackConfig *BuildPackConfig) {
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
	// languageVersion := buildpackConfig.LanguageVersion
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
					err := impl.executeCmd(ciContext, versionUpdateCmd)
					if err != nil {
						log.Println("error occurred while inserting node version", "err", err)
						return
					}
					fileReplaceCmd := fmt.Sprintf("mv %s %s", tmpJsonFile, finalPath)
					err = impl.executeCmd(ciContext, fileReplaceCmd)
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

func (impl *DockerHelperImpl) executeCmd(ciContext cicxt.CiContext, dockerBuild string) error {
	dockerBuildCMD := impl.GetCommandToExecute(dockerBuild)
	err := impl.cmdExecutor.RunCommand(ciContext, dockerBuildCMD)
	if err != nil {
		log.Println(err)
	}
	return err
}

func (impl *DockerHelperImpl) tagDockerBuild(ciContext cicxt.CiContext, dockerRepository string, dest string) error {
	dockerTag := "docker tag " + dockerRepository + ":latest" + " " + dest
	log.Println(" -----> " + dockerTag)
	dockerTagCMD := impl.GetCommandToExecute(dockerTag)
	err := impl.cmdExecutor.RunCommand(ciContext, dockerTagCMD)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (impl *DockerHelperImpl) setupCacheForBuildx(ciContext cicxt.CiContext, localCachePath string, oldCacheBuildxPath string) error {
	err := impl.checkAndCreateDirectory(ciContext, localCachePath)
	if err != nil {
		return err
	}
	err = impl.checkAndCreateDirectory(ciContext, oldCacheBuildxPath)
	if err != nil {
		return err
	}
	copyContent := "cp -R " + localCachePath + " " + oldCacheBuildxPath
	copyContentCmd := exec.Command("/bin/sh", "-c", copyContent)
	err = impl.cmdExecutor.RunCommand(ciContext, copyContentCmd)

	if err != nil {
		log.Println(err)
		return err
	}

	cleanContent := "rm -rf " + localCachePath + "/*"
	cleanContentCmd := exec.Command("/bin/sh", "-c", cleanContent)
	err = impl.cmdExecutor.RunCommand(ciContext, cleanContentCmd)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (impl *DockerHelperImpl) createBuildxBuilder(ciContext cicxt.CiContext) error {
	multiPlatformCmd := "docker buildx create --use --buildkitd-flags '--allow-insecure-entitlement network.host --allow-insecure-entitlement security.insecure'"
	log.Println(" -----> " + multiPlatformCmd)
	dockerBuildCMD := impl.GetCommandToExecute(multiPlatformCmd)
	err := impl.cmdExecutor.RunCommand(ciContext, dockerBuildCMD)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (impl *DockerHelperImpl) installAllSupportedPlatforms(ciContext cicxt.CiContext) error {
	multiPlatformCmd := "docker run --privileged --rm quay.io/devtron/binfmt:stable --install all"
	log.Println(" -----> " + multiPlatformCmd)
	dockerBuildCMD := impl.GetCommandToExecute(multiPlatformCmd)
	err := impl.cmdExecutor.RunCommand(ciContext, dockerBuildCMD)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (impl *DockerHelperImpl) checkAndCreateDirectory(ciContext cicxt.CiContext, localCachePath string) error {
	makeDirCmd := "mkdir -p " + localCachePath
	pathCreateCommand := exec.Command("/bin/sh", "-c", makeDirCmd)
	err := impl.cmdExecutor.RunCommand(ciContext, pathCreateCommand)
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
		registryUrl := ciRequest.IntermediateDockerRegistryUrl
		u, err := util.ParseUrl(registryUrl)
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

func (impl *DockerHelperImpl) PushArtifact(ciContext cicxt.CiContext, dest string) error {
	//awsLogin := "$(aws ecr get-login --no-include-email --region " + ciRequest.AwsRegion + ")"
	dockerPush := "docker push " + dest
	log.Println("-----> " + dockerPush)
	dockerPushCMD := impl.GetCommandToExecute(dockerPush)
	err := impl.cmdExecutor.RunCommand(ciContext, dockerPushCMD)
	if err != nil {
		log.Println(err)
		return err
	}

	// digest := extractDigestUsingPull(dest)
	// log.Println("Digest -----> ", digest)
	// return digest, nil
	return nil
}

func (impl *DockerHelperImpl) ExtractDigestForBuildx(dest string) (string, error) {

	var digest string
	var err error
	manifestLocation := util.LOCAL_BUILDX_LOCATION + "/manifest.json"
	digest, err = readImageDigestFromManifest(manifestLocation)
	if err != nil {
		log.Println("error occurred while extracting digest from manifest reason ", err)
		err = nil // would extract digest using docker pull cmd
	}
	if digest == "" {
		digest, err = impl.ExtractDigestUsingPull(dest)
	}
	log.Println("Digest -----> ", digest)

	return digest, err
}

func (impl *DockerHelperImpl) ExtractDigestUsingPull(dest string) (string, error) {
	dockerPull := "docker pull " + dest
	dockerPullCmd := impl.GetCommandToExecute(dockerPull)
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

func (impl *DockerHelperImpl) createBuildxBuilderForMultiArchBuild(ciContext cicxt.CiContext) error {
	err := impl.installAllSupportedPlatforms(ciContext)
	if err != nil {
		return err
	}
	err = impl.createBuildxBuilder(ciContext)
	if err != nil {
		return err
	}
	return nil
}

func (impl *DockerHelperImpl) createBuildxBuilderWithK8sDriver(ciContext cicxt.CiContext, builderNodes []map[string]string, ciPipelineId, ciWorkflowId int) error {

	if len(builderNodes) == 0 {
		return errors.New("atleast one node is expected for builder with kubernetes driver")
	}
	defaultNodeOpts := builderNodes[0]

	buildxCreate := getBuildxK8sDriverCmd(defaultNodeOpts, ciPipelineId, ciWorkflowId)
	buildxCreate = fmt.Sprintf("%s %s", buildxCreate, "--use")
	fmt.Println(util.DEVTRON, " cmd : ", buildxCreate)
	builderCreateCmd := impl.GetCommandToExecute(buildxCreate)
	err := impl.cmdExecutor.RunCommand(ciContext, builderCreateCmd)
	if err != nil {
		fmt.Println(util.DEVTRON, "buildxCreate : ", buildxCreate, " err : ", err, " error : ")
		return err
	}

	// appending other nodes to the builder,except default node ,since we already added it
	for i := 1; i < len(builderNodes); i++ {
		nodeOpts := builderNodes[i]
		appendNode := getBuildxK8sDriverCmd(nodeOpts, ciPipelineId, ciWorkflowId)
		appendNode = fmt.Sprintf("%s %s", appendNode, "--append")
		fmt.Println(util.DEVTRON, " cmd : ", appendNode)
		appendNodeCmd := impl.GetCommandToExecute(appendNode)
		err = impl.cmdExecutor.RunCommand(ciContext, appendNodeCmd)
		if err != nil {
			fmt.Println(util.DEVTRON, " appendNode : ", appendNode, " err : ", err, " error : ")
			return err
		}
	}

	return nil
}

func (impl *DockerHelperImpl) CleanBuildxK8sDriver(ciContext cicxt.CiContext, nodes []map[string]string) error {
	nodeNames := make([]string, 0)
	for _, nOptsMp := range nodes {
		if nodeName, ok := nOptsMp["node"]; ok && nodeName != "" {
			nodeNames = append(nodeNames, nodeName)
		}
	}
	err := impl.leaveNodesFromBuildxK8sDriver(ciContext, nodeNames)
	if err != nil {
		log.Println(util.DEVTRON, " error in deleting nodes created by ci-runner , err : ", err)
		return err
	}
	log.Println(util.DEVTRON, "successfully cleaned up buildx k8s driver")
	return nil
}

func (impl *DockerHelperImpl) leaveNodesFromBuildxK8sDriver(ciContext cicxt.CiContext, nodeNames []string) error {
	var err error
	defer func() {
		removeCmd := fmt.Sprintf("docker buildx rm %s", BUILDX_K8S_DRIVER_NAME)
		fmt.Println(util.DEVTRON, " cmd : ", removeCmd)
		execRemoveCmd := impl.GetCommandToExecute(removeCmd)
		_ = impl.cmdExecutor.RunCommand(ciContext, execRemoveCmd)

	}()

	for _, node := range nodeNames {
		createCmd := fmt.Sprintf("docker buildx create --name=%s --node=%s --leave", BUILDX_K8S_DRIVER_NAME, node)
		fmt.Println(util.DEVTRON, " cmd : ", createCmd)
		execCreateCmd := impl.GetCommandToExecute(createCmd)
		err = impl.cmdExecutor.RunCommand(ciContext, execCreateCmd)
		if err != nil {
			log.Println(util.DEVTRON, "error in leaving node : ", err)
			return err
		}
	}
	return err
}

// this function is deprecated, use cmdExecutor.RunCommand instead
func (impl *DockerHelperImpl) runCmd(cmd string) (error, *bytes.Buffer) {
	fmt.Println(util.DEVTRON, " cmd : ", cmd)
	builderCreateCmd := impl.GetCommandToExecute(cmd)
	errBuf := &bytes.Buffer{}
	builderCreateCmd.Stderr = errBuf
	err := builderCreateCmd.Run()
	return err, errBuf
}

func getBuildxK8sDriverCmd(driverOpts map[string]string, ciPipelineId, ciWorkflowId int) string {
	buildxCreate := "docker buildx create --buildkitd-flags '--allow-insecure-entitlement network.host --allow-insecure-entitlement security.insecure' --name=%s --driver=kubernetes --node=%s --bootstrap "
	nodeName := driverOpts["node"]
	if nodeName == "" {
		nodeName = BUILDX_NODE_NAME + fmt.Sprintf("%v-%v-", ciPipelineId, ciWorkflowId) + util.Generate(3) // need this to generate unique name for builder node in same builder.
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

func (impl *DockerHelperImpl) StopDocker(ciContext cicxt.CiContext) error {
	cmd := exec.Command("docker", "ps", "-a", "-q")
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	if len(out) > 0 {
		stopCmdS := "docker stop -t 5 $(docker ps -a -q)"
		log.Println(util.DEVTRON, " -----> stopping docker container")
		stopCmd := impl.GetCommandToExecute(stopCmdS)
		err := impl.cmdExecutor.RunCommand(ciContext, stopCmd)
		log.Println(util.DEVTRON, " -----> stopped docker container")
		if err != nil {
			log.Fatal(err)
			return err
		}
		removeContainerCmds := "docker rm -v -f $(docker ps -a -q)"
		log.Println(util.DEVTRON, " -----> removing docker container")
		removeContainerCmd := impl.GetCommandToExecute(removeContainerCmds)
		err = impl.cmdExecutor.RunCommand(ciContext, removeContainerCmd)
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
	impl.DockerdUpCheck() // FIXME: this call should be removed
	// ensureDockerDaemonHasStopped(20)
	return nil
}

func (impl *DockerHelperImpl) ensureDockerDaemonHasStopped(retryCount int) error {
	var err error
	retry := 0
	for err == nil {
		time.Sleep(1 * time.Second)
		err = impl.DockerdUpCheck()
		retry++
		if retry == retryCount {
			break
		}
	}
	return err
}

func (impl *DockerHelperImpl) waitForDockerDaemon(retryCount int) error {
	err := impl.DockerdUpCheck()
	retry := 0
	for err != nil {
		if retry == retryCount {
			break
		}
		time.Sleep(1 * time.Second)
		err = impl.DockerdUpCheck()
		retry++
	}
	return err
}

func (impl *DockerHelperImpl) DockerdUpCheck() error {
	dockerCheck := "docker ps"
	dockerCheckCmd := impl.GetCommandToExecute(dockerCheck)
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

func GetSelfManagedDockerfilePath(checkoutPath string) string {
	return filepath.Join(util.WORKINGDIR, checkoutPath, "./Dockerfile")
}
