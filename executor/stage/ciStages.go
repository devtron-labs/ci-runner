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

package stage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/devtron-labs/ci-runner/executor"
	cicxt "github.com/devtron-labs/ci-runner/executor/context"
	util2 "github.com/devtron-labs/ci-runner/executor/util"
	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
	"github.com/devtron-labs/common-lib/utils"
	"github.com/devtron-labs/common-lib/utils/bean"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

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

type CiStage struct {
	gitManager           helper.GitManager
	dockerHelper         helper.DockerHelper
	stageExecutorManager executor.StageExecutor
}

func NewCiStage(gitManager helper.GitManager, dockerHelper helper.DockerHelper, stageExecutor executor.StageExecutor) *CiStage {
	return &CiStage{
		gitManager:           gitManager,
		dockerHelper:         dockerHelper,
		stageExecutorManager: stageExecutor,
	}
}

func (impl *CiStage) HandleCIEvent(ciCdRequest *helper.CiCdTriggerEvent, exitCode *int) {
	ciRequest := ciCdRequest.CommonWorkflowRequest
	ciContext := cicxt.BuildCiContext(context.Background(), ciRequest.EnableSecretMasking)
	artifactUploaded, err := impl.runCIStages(ciContext, ciCdRequest)
	log.Println(util.DEVTRON, artifactUploaded, err)
	var artifactUploadErr error
	if !artifactUploaded {
		cloudHelperBaseConfig := ciRequest.GetCloudHelperBaseConfig(util.BlobStorageObjectTypeArtifact)
		artifactUploadErr = helper.ZipAndUpload(cloudHelperBaseConfig, ciCdRequest.CommonWorkflowRequest.CiArtifactFileName)
		artifactUploaded = artifactUploadErr == nil
	}

	if err != nil {
		var stageError *helper.CiStageError
		log.Println(util.DEVTRON, err)
		if errors.As(err, &stageError) {
			*exitCode = util.CiStageFailErrorCode
			return
		}
		*exitCode = util.DefaultErrorCode
		return
	}

	if artifactUploadErr != nil {
		log.Println(util.DEVTRON, artifactUploadErr)
		if ciCdRequest.CommonWorkflowRequest.IsExtRun {
			log.Println(util.DEVTRON, "Ignoring artifactUploadErr")
			return
		}
		*exitCode = util.DefaultErrorCode
		return
	}

	// sync cache
	uploadCache := func() error {
		log.Println(util.DEVTRON, " cache-push")
		err = helper.SyncCache(ciRequest)
		if err != nil {
			log.Println(err)
			if ciCdRequest.CommonWorkflowRequest.IsExtRun {
				log.Println(util.DEVTRON, "Ignoring cache upload")
				// not returning error as we are ignoring the cache upload, todo: re confirm this
				return nil
			}
			*exitCode = util.DefaultErrorCode
			return err
		}
		log.Println(util.DEVTRON, " /cache-push")
		return nil
	}

	// not returning error by choice, do not want to report this error to caller
	// cache push can fail and we don't want to break the flow
	util.ExecuteWithStageInfoLog(util.PUSH_CACHE, uploadCache)
}

type CiFailReason string

const (
	PreCi  CiFailReason = "Pre-CI task failed: "
	PostCi CiFailReason = "Post-CI task failed: "
	Build  CiFailReason = "Docker build failed"
	Push   CiFailReason = "Docker push failed"
	Scan   CiFailReason = "Image scan failed"
)

func (impl *CiStage) runCIStages(ciContext cicxt.CiContext, ciCdRequest *helper.CiCdTriggerEvent) (artifactUploaded bool, err error) {

	metrics := &helper.CIMetrics{}
	start := time.Now()
	metrics.TotalStartTime = start
	artifactUploaded = false
	// change the current working directory to '/'
	err = os.Chdir(util.HOMEDIR)
	if err != nil {
		return artifactUploaded, err
	}

	// using stat to get check if WORKINGDIR exist or not
	if _, err := os.Stat(util.WORKINGDIR); os.IsNotExist(err) {
		// Creating the WORKINGDIR if in case in doesn't exit
		_ = os.Mkdir(util.WORKINGDIR, os.ModeDir)
	}

	// Get ci cache TODO
	pullCacheStage := func() error {
		log.Println(util.DEVTRON, " cache-pull")
		start = time.Now()
		metrics.CacheDownStartTime = start

		defer func() {
			log.Println(util.DEVTRON, " /cache-pull")
			metrics.CacheDownDuration = time.Since(start).Seconds()
		}()

		err = helper.GetCache(ciCdRequest.CommonWorkflowRequest)
		if err != nil {
			return err
		}
		return nil
	}

	if err = util.ExecuteWithStageInfoLog(util.CACHE_PULL, pullCacheStage); err != nil {
		return artifactUploaded, err
	}

	// change the current working directory to WORKINGDIR
	err = os.Chdir(util.WORKINGDIR)
	if err != nil {
		return artifactUploaded, err
	}
	// git handling
	log.Println(util.DEVTRON, " git")
	ciBuildConfi := ciCdRequest.CommonWorkflowRequest.CiBuildConfig
	buildSkipEnabled := ciBuildConfi != nil && ciBuildConfi.CiBuildType == helper.BUILD_SKIP_BUILD_TYPE
	skipCheckout := ciBuildConfi != nil && ciBuildConfi.PipelineType == helper.CI_JOB
	if !skipCheckout {
		err = impl.gitManager.CloneAndCheckout(ciCdRequest.CommonWorkflowRequest.CiProjectDetails)
	}
	if err != nil {
		log.Println(util.DEVTRON, "clone err", err)
		return artifactUploaded, err
	}
	log.Println(util.DEVTRON, " /git")

	// Start docker daemon TODO
	log.Println(util.DEVTRON, " docker-build")
	impl.dockerHelper.StartDockerDaemon(ciCdRequest.CommonWorkflowRequest)
	extraEnvVars, err := impl.AddExtraEnvVariableFromRuntimeParamsToCiCdEvent(ciCdRequest.CommonWorkflowRequest)
	if err != nil {
		return artifactUploaded, err
	}
	ciCdRequest.CommonWorkflowRequest.ExtraEnvironmentVariables = extraEnvVars
	scriptEnvs, err := util2.GetGlobalEnvVariables(ciCdRequest)
	if err != nil {
		return artifactUploaded, err
	}
	// Get devtron-ci yaml
	yamlLocation := ciCdRequest.CommonWorkflowRequest.CheckoutPath
	log.Println(util.DEVTRON, "devtron-ci yaml location ", yamlLocation)
	taskYaml, err := helper.GetTaskYaml(yamlLocation)
	if err != nil {
		return artifactUploaded, err
	}
	ciCdRequest.CommonWorkflowRequest.TaskYaml = taskYaml
	if ciBuildConfi != nil && ciBuildConfi.CiBuildType == helper.MANAGED_DOCKERFILE_BUILD_TYPE {
		err = makeDockerfile(ciBuildConfi.DockerBuildConfig, ciCdRequest.CommonWorkflowRequest.CheckoutPath)
		if err != nil {
			return artifactUploaded, err
		}
	}

	refStageMap := make(map[int][]*helper.StepObject)
	for _, ref := range ciCdRequest.CommonWorkflowRequest.RefPlugins {
		refStageMap[ref.Id] = ref.Steps
	}

	var preCiStageOutVariable map[int]map[string]*helper.VariableObject
	start = time.Now()
	metrics.PreCiStartTime = start
	var resultsFromPlugin json.RawMessage
	if len(ciCdRequest.CommonWorkflowRequest.PreCiSteps) > 0 {
		resultsFromPlugin, preCiStageOutVariable, err = impl.runPreCiSteps(ciCdRequest, metrics, buildSkipEnabled, refStageMap, scriptEnvs, artifactUploaded)
		if err != nil {
			return artifactUploaded, err
		}
	}
	var dest string
	var digest string
	if !buildSkipEnabled {
		dest, digest, err = impl.getImageDestAndDigest(ciCdRequest, metrics, scriptEnvs, refStageMap, preCiStageOutVariable, artifactUploaded)
		if err != nil {
			return artifactUploaded, err
		}
	}
	var postCiDuration float64
	start = time.Now()
	metrics.PostCiStartTime = start
	var pluginArtifacts *helper.PluginArtifacts
	if len(ciCdRequest.CommonWorkflowRequest.PostCiSteps) > 0 {
		pluginArtifacts, err = impl.runPostCiSteps(ciCdRequest, scriptEnvs, refStageMap, preCiStageOutVariable, metrics, artifactUploaded, dest, digest)
		postCiDuration = time.Since(start).Seconds()
		if err != nil {
			return artifactUploaded, err
		}
	}
	metrics.PostCiDuration = postCiDuration
	log.Println(util.DEVTRON, " /docker-push")

	log.Println(util.DEVTRON, " artifact-upload")
	cloudHelperBaseConfig := ciCdRequest.CommonWorkflowRequest.GetCloudHelperBaseConfig(util.BlobStorageObjectTypeArtifact)
	err = helper.ZipAndUpload(cloudHelperBaseConfig, ciCdRequest.CommonWorkflowRequest.CiArtifactFileName)
	if err != nil {
		return artifactUploaded, nil
	} else {
		artifactUploaded = true
	}
	log.Println(util.DEVTRON, " /artifact-upload")

	dest, err = impl.dockerHelper.GetDestForNatsEvent(ciCdRequest.CommonWorkflowRequest, dest)
	if err != nil {
		return artifactUploaded, err
	}
	// scan only if ci scan enabled
	if helper.IsEventTypeEligibleToScanImage(ciCdRequest.Type) &&
		ciCdRequest.CommonWorkflowRequest.ScanEnabled {
		err = runImageScanning(dest, digest, ciCdRequest, metrics, artifactUploaded)
		if err != nil {
			return artifactUploaded, err
		}
	}

	log.Println(util.DEVTRON, " event")
	metrics.TotalDuration = time.Since(metrics.TotalStartTime).Seconds()
	// When externalCiArtifact is provided (run time Env at time of build) then this image will be used further in the pipeline
	// imageDigest and ciProjectDetails are optional fields
	if scriptEnvs["externalCiArtifact"] != "" {
		log.Println(util.DEVTRON, "external ci artifact found! exiting now with success event")
		dest = scriptEnvs["externalCiArtifact"]
		digest = scriptEnvs["imageDigest"]
		if len(digest) == 0 {
			var useAppDockerConfigForPrivateRegistries bool
			var err error
			useAppDockerConfig, ok := ciCdRequest.CommonWorkflowRequest.ExtraEnvironmentVariables["useAppDockerConfig"]
			if ok && len(useAppDockerConfig) > 0 {
				useAppDockerConfigForPrivateRegistries, err = strconv.ParseBool(useAppDockerConfig)
				if err != nil {
					fmt.Println(fmt.Sprintf("Error in parsing useAppDockerConfig runtime param to bool from string useAppDockerConfigForPrivateRegistries:- %s, err:", useAppDockerConfig), err)
				}
			}
			var dockerAuthConfig *bean.DockerAuthConfig
			if useAppDockerConfigForPrivateRegistries {
				dockerAuthConfig = impl.dockerHelper.GetDockerAuthConfigForPrivateRegistries(ciCdRequest.CommonWorkflowRequest)
			}
			startTime := time.Now()
			//user has not provided imageDigest in that case fetch from docker.
			imgDigest, err := impl.dockerHelper.ExtractDigestFromImage(dest, ciCdRequest.CommonWorkflowRequest.UseDockerApiToGetDigest, dockerAuthConfig)
			if err != nil {
				fmt.Println(fmt.Sprintf("Error in extracting digest from image %s, err:", dest), err)
				return artifactUploaded, err
			}
			log.Println(fmt.Sprintf("time since extract digest from image process:- %s", time.Since(startTime).String()))
			digest = imgDigest
		}
		var tempDetails []*helper.CiProjectDetailsMin
		err := json.Unmarshal([]byte(scriptEnvs["ciProjectDetails"]), &tempDetails)
		if err != nil {
			fmt.Println("Error unmarshalling ciProjectDetails JSON:", err)
			fmt.Println("ignoring the error and continuing without saving ciProjectDetails")
		}

		if len(tempDetails) > 0 && len(ciCdRequest.CommonWorkflowRequest.CiProjectDetails) > 0 {
			detail := tempDetails[0]
			ciCdRequest.CommonWorkflowRequest.CiProjectDetails[0].CommitHash = detail.CommitHash
			ciCdRequest.CommonWorkflowRequest.CiProjectDetails[0].Message = detail.Message
			ciCdRequest.CommonWorkflowRequest.CiProjectDetails[0].Author = detail.Author
			ciCdRequest.CommonWorkflowRequest.CiProjectDetails[0].CommitTime = detail.CommitTime
		}
	}

	err = helper.SendEvents(ciCdRequest.CommonWorkflowRequest, digest, dest, *metrics, artifactUploaded, "", resultsFromPlugin, pluginArtifacts)
	if err != nil {
		log.Println(err)
		return artifactUploaded, err
	}
	log.Println(util.DEVTRON, " /event")

	err = impl.dockerHelper.StopDocker(ciContext)
	if err != nil {
		log.Println("err", err)
		return artifactUploaded, err
	}
	return artifactUploaded, nil
}

func (impl *CiStage) runPreCiSteps(ciCdRequest *helper.CiCdTriggerEvent, metrics *helper.CIMetrics,
	buildSkipEnabled bool, refStageMap map[int][]*helper.StepObject,
	scriptEnvs map[string]string, artifactUploaded bool) (json.RawMessage, map[int]map[string]*helper.VariableObject, error) {
	start := time.Now()
	metrics.PreCiStartTime = start
	if !buildSkipEnabled {
		log.Println("running PRE-CI steps")
	}
	// run pre artifact processing
	_, preCiStageOutVariable, step, err := impl.stageExecutorManager.RunCiCdSteps(helper.STEP_TYPE_PRE, ciCdRequest.CommonWorkflowRequest, ciCdRequest.CommonWorkflowRequest.PreCiSteps, refStageMap, scriptEnvs, nil)
	preCiDuration := time.Since(start).Seconds()
	if err != nil {
		log.Println("error in running pre Ci Steps", "err", err)
		err = sendFailureNotification(string(PreCi)+step.Name, ciCdRequest.CommonWorkflowRequest, "", "", *metrics, artifactUploaded, err)
		return nil, nil, err
	}
	// considering pull images from Container repo Plugin in Pre ci steps only.
	// making it non-blocking if results are not available (in case of err)
	resultsFromPlugin, fileErr := extractOutResultsIfExists()
	if fileErr != nil {
		log.Println("error in getting results", "err", fileErr.Error())
	}
	metrics.PreCiDuration = preCiDuration
	return resultsFromPlugin, preCiStageOutVariable, nil
}

func (impl *CiStage) runBuildArtifact(ciCdRequest *helper.CiCdTriggerEvent, metrics *helper.CIMetrics,
	refStageMap map[int][]*helper.StepObject, scriptEnvs map[string]string, artifactUploaded bool,
	preCiStageOutVariable map[int]map[string]*helper.VariableObject) (string, error) {
	// build
	start := time.Now()
	metrics.BuildStartTime = start
	dest, err := impl.dockerHelper.BuildArtifact(ciCdRequest.CommonWorkflowRequest) // TODO make it skipable
	metrics.BuildDuration = time.Since(start).Seconds()
	if err != nil {
		log.Println("Error in building artifact", "err", err)
		// code-block starts : run post-ci which are enabled to run on ci fail
		postCiStepsToTriggerOnCiFail := getPostCiStepToRunOnCiFail(ciCdRequest.CommonWorkflowRequest.PostCiSteps)
		if len(postCiStepsToTriggerOnCiFail) > 0 {
			log.Println("Running POST-CI steps which are enabled to RUN even on CI FAIL")
			// build success will always be false
			scriptEnvs[util.ENV_VARIABLE_BUILD_SUCCESS] = "false"
			// run post artifact processing
			impl.stageExecutorManager.RunCiCdSteps(helper.STEP_TYPE_POST, ciCdRequest.CommonWorkflowRequest, postCiStepsToTriggerOnCiFail, refStageMap, scriptEnvs, preCiStageOutVariable)
		}
		// code-block ends
		err = sendFailureNotification(string(Build), ciCdRequest.CommonWorkflowRequest, "", "", *metrics, artifactUploaded, err)
	}
	log.Println(util.DEVTRON, " Build artifact completed", "dest", dest, "err", err)
	return dest, err
}

func (impl *CiStage) extractDigest(ciCdRequest *helper.CiCdTriggerEvent, dest string, metrics *helper.CIMetrics, artifactUploaded bool) (string, error) {

	var digest string
	var err error

	extractDigestStage := func() error {
		ciBuildConfi := ciCdRequest.CommonWorkflowRequest.CiBuildConfig
		isBuildX := ciBuildConfi != nil && ciBuildConfi.DockerBuildConfig != nil && ciBuildConfi.DockerBuildConfig.CheckForBuildX()
		if isBuildX {
			digest, err = impl.dockerHelper.ExtractDigestForBuildx(dest, ciCdRequest.CommonWorkflowRequest)
		} else {
			// push to dest
			log.Println(util.DEVTRON, "Docker push Artifact", "dest", dest)
			err = impl.pushArtifact(ciCdRequest, dest, digest, metrics, artifactUploaded)
			if err != nil {
				return err
			}
			digest, err = impl.dockerHelper.ExtractDigestForBuildx(dest, ciCdRequest.CommonWorkflowRequest)
		}
		return err
	}

	err = util.ExecuteWithStageInfoLog(util.DOCKER_PUSH_AND_EXTRACT_IMAGE_DIGEST, extractDigestStage)
	return digest, err
}

func (impl *CiStage) runPostCiSteps(ciCdRequest *helper.CiCdTriggerEvent, scriptEnvs map[string]string, refStageMap map[int][]*helper.StepObject, preCiStageOutVariable map[int]map[string]*helper.VariableObject, metrics *helper.CIMetrics, artifactUploaded bool, dest string, digest string) (*helper.PluginArtifacts, error) {
	log.Println("running POST-CI steps")
	// sending build success as true always as post-ci triggers only if ci gets success
	scriptEnvs[util.ENV_VARIABLE_BUILD_SUCCESS] = "true"
	scriptEnvs["DEST"] = dest
	scriptEnvs["DIGEST"] = digest
	// run post artifact processing
	pluginArtifactsFromFile, _, step, err := impl.stageExecutorManager.RunCiCdSteps(helper.STEP_TYPE_POST, ciCdRequest.CommonWorkflowRequest, ciCdRequest.CommonWorkflowRequest.PostCiSteps, refStageMap, scriptEnvs, preCiStageOutVariable)
	if err != nil {
		log.Println("error in running Post Ci Steps", "err", err)
		return nil, sendFailureNotification(string(PostCi)+step.Name, ciCdRequest.CommonWorkflowRequest, "", "", *metrics, artifactUploaded, err)
	}
	//sent by orchestrator if copy container image v2 is configured
	return pluginArtifactsFromFile, nil
}

func runImageScanning(dest string, digest string, ciCdRequest *helper.CiCdTriggerEvent, metrics *helper.CIMetrics, artifactUploaded bool) error {
	imageScanningStage := func() error {
		log.Println("Image Scanning Started for digest", digest)
		scanEvent := &helper.ScanEvent{
			Image:               dest,
			ImageDigest:         digest,
			PipelineId:          ciCdRequest.CommonWorkflowRequest.PipelineId,
			UserId:              ciCdRequest.CommonWorkflowRequest.TriggeredBy,
			DockerRegistryId:    ciCdRequest.CommonWorkflowRequest.DockerRegistryId,
			DockerConnection:    ciCdRequest.CommonWorkflowRequest.DockerConnection,
			DockerCert:          ciCdRequest.CommonWorkflowRequest.DockerCert,
			ImageScanMaxRetries: ciCdRequest.CommonWorkflowRequest.ImageScanMaxRetries,
			ImageScanRetryDelay: ciCdRequest.CommonWorkflowRequest.ImageScanRetryDelay,
		}
		err := helper.SendEventToClairUtility(scanEvent)
		if err != nil {
			log.Println("error in running Image Scan", "err", err)
			err = sendFailureNotification(string(Scan), ciCdRequest.CommonWorkflowRequest, digest, dest, *metrics, artifactUploaded, err)
			return err
		}
		log.Println("Image scanning completed with scanEvent", scanEvent)
		return nil
	}

	return util.ExecuteWithStageInfoLog(util.IMAGE_SCAN, imageScanningStage)
}

func (impl *CiStage) getImageDestAndDigest(ciCdRequest *helper.CiCdTriggerEvent, metrics *helper.CIMetrics, scriptEnvs map[string]string, refStageMap map[int][]*helper.StepObject, preCiStageOutVariable map[int]map[string]*helper.VariableObject, artifactUploaded bool) (string, string, error) {
	dest, err := impl.runBuildArtifact(ciCdRequest, metrics, refStageMap, scriptEnvs, artifactUploaded, preCiStageOutVariable)
	if err != nil {
		return "", "", err
	}
	digest, err := impl.extractDigest(ciCdRequest, dest, metrics, artifactUploaded)
	if err != nil {
		log.Println("Error in extracting digest", "err", err)
		return "", "", err
	}
	return dest, digest, nil
}

func getPostCiStepToRunOnCiFail(postCiSteps []*helper.StepObject) []*helper.StepObject {
	var postCiStepsToTriggerOnCiFail []*helper.StepObject
	if len(postCiSteps) > 0 {
		for _, postCiStep := range postCiSteps {
			if postCiStep.TriggerIfParentStageFail {
				postCiStepsToTriggerOnCiFail = append(postCiStepsToTriggerOnCiFail, postCiStep)
			}
		}
	}
	return postCiStepsToTriggerOnCiFail
}

// extractOutResultsIfExists will return the json.RawMessage results from file
// if file doesn't exist, returns nil
func extractOutResultsIfExists() (json.RawMessage, error) {
	exists, err := util.CheckFileExists(util.ResultsDirInCIRunnerPath)
	if err != nil || !exists {
		log.Println("err", err)
		return nil, err
	}
	file, err := ioutil.ReadFile(util.ResultsDirInCIRunnerPath)
	if err != nil {
		log.Println("error in reading file", "err", err.Error())
		return nil, err
	}
	return file, nil
}

func makeDockerfile(config *helper.DockerBuildConfig, checkoutPath string) error {
	dockerfilePath := helper.GetSelfManagedDockerfilePath(checkoutPath)
	dockerfileContent := config.DockerfileContent
	f, err := os.Create(dockerfilePath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(dockerfileContent)
	return err
}

func sendFailureNotification(failureMessage string, ciRequest *helper.CommonWorkflowRequest,
	digest string, image string, ciMetrics helper.CIMetrics,
	artifactUploaded bool, err error) error {
	e := helper.SendEvents(ciRequest, digest, image, ciMetrics, artifactUploaded, failureMessage, nil, nil)
	if e != nil {
		log.Println(e)
		return e
	}
	return &helper.CiStageError{Err: err}
}

func (impl *CiStage) pushArtifact(ciCdRequest *helper.CiCdTriggerEvent, dest string, digest string, metrics *helper.CIMetrics, artifactUploaded bool) error {
	imageRetryCountValue := ciCdRequest.CommonWorkflowRequest.ImageRetryCount
	imageRetryIntervalValue := ciCdRequest.CommonWorkflowRequest.ImageRetryInterval
	var err error
	for i := 0; i < imageRetryCountValue+1; i++ {
		if i != 0 {
			time.Sleep(time.Duration(imageRetryIntervalValue) * time.Second)
		}
		ciContext := cicxt.BuildCiContext(context.Background(), ciCdRequest.CommonWorkflowRequest.EnableSecretMasking)
		err = impl.dockerHelper.PushArtifact(ciContext, dest)
		if err == nil {
			break
		}
		if err != nil {
			log.Println("Error in pushing artifact", "err", err)
		}
	}
	if err != nil {
		err = sendFailureNotification(string(Push), ciCdRequest.CommonWorkflowRequest, digest, dest, *metrics, artifactUploaded, err)
		return err
	}
	return err
}

func (impl *CiStage) AddExtraEnvVariableFromRuntimeParamsToCiCdEvent(ciRequest *helper.CommonWorkflowRequest) (map[string]string, error) {
	if len(ciRequest.ExtraEnvironmentVariables["externalCiArtifact"]) > 0 {
		var err error
		image := ciRequest.ExtraEnvironmentVariables["externalCiArtifact"]
		if !strings.Contains(image, ":") {
			//check for tag name
			if utils.IsValidDockerTagName(image) {
				ciRequest.DockerImageTag = image
				image, err = helper.BuildDockerImagePath(ciRequest)
				if err != nil {
					log.Println("Error in building docker image", "err", err)
					return nil, err
				}
				ciRequest.ExtraEnvironmentVariables["externalCiArtifact"] = image
			} else {
				return nil, errors.New("external-ci-artifact image is neither a url nor a tag name")
			}

		}
		if ciRequest.ShouldPullDigest {
			var useAppDockerConfigForPrivateRegistries bool
			var err error
			useAppDockerConfig, ok := ciRequest.ExtraEnvironmentVariables["useAppDockerConfig"]
			if ok && len(useAppDockerConfig) > 0 {
				useAppDockerConfigForPrivateRegistries, err = strconv.ParseBool(useAppDockerConfig)
				if err != nil {
					fmt.Println(fmt.Sprintf("Error in parsing useAppDockerConfig runtime param to bool from string useAppDockerConfigForPrivateRegistries:- %s, err:", useAppDockerConfig), err)
					return ciRequest.ExtraEnvironmentVariables, err
				}
			}
			var dockerAuthConfig *bean.DockerAuthConfig
			if useAppDockerConfigForPrivateRegistries {
				dockerAuthConfig = impl.dockerHelper.GetDockerAuthConfigForPrivateRegistries(ciRequest)
			}
			log.Println("image scanning plugin configured and digest not provided hence pulling image digest")
			startTime := time.Now()
			//user has not provided imageDigest in that case fetch from docker.
			imgDigest, err := impl.dockerHelper.ExtractDigestFromImage(image, ciRequest.UseDockerApiToGetDigest, dockerAuthConfig)
			if err != nil {
				fmt.Println(fmt.Sprintf("Error in extracting digest from image %s, err:", image), err)
				return ciRequest.ExtraEnvironmentVariables, err
			}
			log.Println(fmt.Sprintf("time since extract digest from image process:- %s", time.Since(startTime).String()))
			log.Println(fmt.Sprintf("image:- %s , image digest:- %s", image, imgDigest))
			ciRequest.ExtraEnvironmentVariables["imageDigest"] = imgDigest
		}
	}
	return ciRequest.ExtraEnvironmentVariables, nil
}
