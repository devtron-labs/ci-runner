package main

import (
	"encoding/json"
	"errors"
	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

func HandleCIEvent(ciCdRequest *helper.CiCdTriggerEvent, exitCode *int) {
	ciRequest := ciCdRequest.CommonWorkflowRequest
	artifactUploaded, err := runCIStages(ciCdRequest)
	log.Println(util.DEVTRON, artifactUploaded, err)
	var artifactUploadErr error
	if !artifactUploaded {
		artifactUploaded, artifactUploadErr = helper.ZipAndUpload(ciRequest.BlobStorageConfigured, ciCdRequest.CommonWorkflowRequest.BlobStorageS3Config, ciCdRequest.CommonWorkflowRequest.CiArtifactFileName, ciCdRequest.CommonWorkflowRequest.CloudProvider, ciCdRequest.CommonWorkflowRequest.AzureBlobConfig, ciCdRequest.CommonWorkflowRequest.GcpBlobConfig)
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
	log.Println(util.DEVTRON, " cache-push")
	err = helper.SyncCache(ciRequest)
	if err != nil {
		log.Println(err)
		if ciCdRequest.CommonWorkflowRequest.IsExtRun {
			log.Println(util.DEVTRON, "Ignoring cache upload")
			return
		}
		*exitCode = util.DefaultErrorCode
		return
	}
	log.Println(util.DEVTRON, " /cache-push")
}

type CiFailReason string

const (
	PreCi  CiFailReason = "Pre-CI task failed: "
	PostCi CiFailReason = "Post-CI task failed: "
	Build  CiFailReason = "Docker build failed"
	Push   CiFailReason = "Docker push failed"
	Scan   CiFailReason = "Image scan failed"
)

func runCIStages(ciCdRequest *helper.CiCdTriggerEvent) (artifactUploaded bool, err error) {

	var metrics helper.CIMetrics
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

	// Get ci cache
	log.Println(util.DEVTRON, " cache-pull")
	start = time.Now()
	metrics.CacheDownStartTime = start
	err = helper.GetCache(ciCdRequest.CommonWorkflowRequest)
	metrics.CacheDownDuration = time.Since(start).Seconds()
	if err != nil {
		return artifactUploaded, err
	}
	log.Println(util.DEVTRON, " /cache-pull")

	// change the current working directory to WORKINGDIR
	err = os.Chdir(util.WORKINGDIR)
	if err != nil {
		return artifactUploaded, err
	}
	// git handling
	log.Println(util.DEVTRON, " git")
	ciBuildConfigBean := ciCdRequest.CommonWorkflowRequest.CiBuildConfig
	buildSkipEnabled := ciBuildConfigBean != nil && ciBuildConfigBean.CiBuildType == helper.BUILD_SKIP_BUILD_TYPE
	skipCheckout := ciBuildConfigBean != nil && ciBuildConfigBean.PipelineType == helper.CI_JOB
	if !skipCheckout {
		err = helper.CloneAndCheckout(ciCdRequest.CommonWorkflowRequest.CiProjectDetails)
	}
	if err != nil {
		log.Println(util.DEVTRON, "clone err", err)
		return artifactUploaded, err
	}
	log.Println(util.DEVTRON, " /git")

	// Start docker daemon
	log.Println(util.DEVTRON, " docker-build")

	helper.StartDockerDaemon(ciCdRequest.CommonWorkflowRequest.DockerConnection, ciCdRequest.CommonWorkflowRequest.DockerRegistryURL, ciCdRequest.CommonWorkflowRequest.DockerCert, ciCdRequest.CommonWorkflowRequest.DefaultAddressPoolBaseCidr, ciCdRequest.CommonWorkflowRequest.DefaultAddressPoolSize, ciCdRequest.CommonWorkflowRequest.CiBuildDockerMtuValue)

	scriptEnvs, err := getGlobalEnvVariables(ciCdRequest)
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
	if ciBuildConfigBean != nil && ciBuildConfigBean.CiBuildType == helper.MANAGED_DOCKERFILE_BUILD_TYPE {
		err = makeDockerfile(ciBuildConfigBean.DockerBuildConfig, ciCdRequest.CommonWorkflowRequest.CheckoutPath)
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
	var resultsFromPlugin *helper.ImageDetailsFromCR
	if len(ciCdRequest.CommonWorkflowRequest.PreCiSteps) > 0 {
		resultsFromPlugin, preCiStageOutVariable, err = runPreCiSteps(ciCdRequest, &metrics, buildSkipEnabled, refStageMap, scriptEnvs, artifactUploaded)
		if err != nil {
			return artifactUploaded, err
		}
	}
	var dest string
	var digest string
	if !buildSkipEnabled {
		dest, digest, err = getImageDestAndDigest(ciCdRequest, metrics, scriptEnvs, refStageMap, preCiStageOutVariable, artifactUploaded)
		if err != nil {
			return artifactUploaded, err
		}
	}
	var postCiDuration float64
	start = time.Now()
	metrics.PostCiStartTime = start
	if len(ciCdRequest.CommonWorkflowRequest.PostCiSteps) > 0 {
		err = runPostCiSteps(ciCdRequest, scriptEnvs, refStageMap, preCiStageOutVariable, metrics, artifactUploaded, dest, digest)
		postCiDuration = time.Since(start).Seconds()
		if err != nil {
			return artifactUploaded, err
		}
	}
	metrics.PostCiDuration = postCiDuration
	log.Println(util.DEVTRON, " /docker-push")

	log.Println(util.DEVTRON, " artifact-upload")

	artifactUploaded, err = helper.ZipAndUpload(ciCdRequest.CommonWorkflowRequest.BlobStorageConfigured, ciCdRequest.CommonWorkflowRequest.BlobStorageS3Config, ciCdRequest.CommonWorkflowRequest.CiArtifactFileName, ciCdRequest.CommonWorkflowRequest.CloudProvider, ciCdRequest.CommonWorkflowRequest.AzureBlobConfig, ciCdRequest.CommonWorkflowRequest.GcpBlobConfig)

	if err != nil {
		return artifactUploaded, nil
	}
	//else {
	//	artifactUploaded = true
	//}
	log.Println(util.DEVTRON, " /artifact-upload")

	// scan only if ci scan enabled
	if ciCdRequest.CommonWorkflowRequest.ScanEnabled {
		err = runImageScanning(dest, digest, ciCdRequest, metrics, artifactUploaded)
		if err != nil {
			return artifactUploaded, err
		}
	}

	log.Println(util.DEVTRON, " event")
	metrics.TotalDuration = time.Since(metrics.TotalStartTime).Seconds()

	err = helper.SendEvents(ciCdRequest.CommonWorkflowRequest, digest, dest, metrics, artifactUploaded, "", resultsFromPlugin)
	if err != nil {
		log.Println(err)
		return artifactUploaded, err
	}
	log.Println(util.DEVTRON, " /event")

	err = helper.StopDocker()
	if err != nil {
		log.Println("err", err)
		return artifactUploaded, err
	}
	return artifactUploaded, nil
}

func runPreCiSteps(ciCdRequest *helper.CiCdTriggerEvent, metrics *helper.CIMetrics,
	buildSkipEnabled bool, refStageMap map[int][]*helper.StepObject,
	scriptEnvs map[string]string, artifactUploaded bool) (*helper.ImageDetailsFromCR, map[int]map[string]*helper.VariableObject, error) {
	start := time.Now()
	metrics.PreCiStartTime = start
	var resultsFromPlugin *helper.ImageDetailsFromCR
	if !buildSkipEnabled {
		util.LogStage("running PRE-CI steps")
	}
	// run pre artifact processing
	preCiStageOutVariable, step, err := RunCiCdSteps(STEP_TYPE_PRE, ciCdRequest.CommonWorkflowRequest.PreCiSteps, refStageMap, scriptEnvs, nil)
	preCiDuration := time.Since(start).Seconds()
	if err != nil {
		log.Println("error in running pre Ci Steps", "err", err)
		err = sendFailureNotification(string(PreCi)+step.Name, ciCdRequest.CommonWorkflowRequest, "", "", *metrics, artifactUploaded, err)
		return nil, nil, err
	}
	// considering pull images from Container repo Plugin in Pre ci steps only.
	// making it non-blocking if results are not available (in case of err)
	resultsFromPlugin, err1 := extractOutResultsIfExists()
	if err1 != nil {
		log.Println("error in getting results", "err", err1.Error())
	}
	metrics.PreCiDuration = preCiDuration
	return resultsFromPlugin, preCiStageOutVariable, nil
}

func runBuildArtifact(ciCdRequest *helper.CiCdTriggerEvent, metrics *helper.CIMetrics,
	refStageMap map[int][]*helper.StepObject, scriptEnvs map[string]string, artifactUploaded bool,
	preCiStageOutVariable map[int]map[string]*helper.VariableObject) (string, error) {
	util.LogStage("Build")
	// build
	start := time.Now()
	metrics.BuildStartTime = start
	dest, err := helper.BuildArtifact(ciCdRequest.CommonWorkflowRequest) //TODO make it skipable
	metrics.BuildDuration = time.Since(start).Seconds()
	if err != nil {
		log.Println("Error in building artifact", "err", err)
		// code-block starts : run post-ci which are enabled to run on ci fail
		postCiStepsToTriggerOnCiFail := getPostCiStepToRunOnCiFail(ciCdRequest.CommonWorkflowRequest.PostCiSteps)
		if len(postCiStepsToTriggerOnCiFail) > 0 {
			util.LogStage("Running POST-CI steps which are enabled to RUN even on CI FAIL")
			// build success will always be false
			scriptEnvs[util.ENV_VARIABLE_BUILD_SUCCESS] = "false"
			// run post artifact processing
			RunCiCdSteps(STEP_TYPE_POST, postCiStepsToTriggerOnCiFail, refStageMap, scriptEnvs, preCiStageOutVariable)
		}
		// code-block ends
		err = sendFailureNotification(string(Build), ciCdRequest.CommonWorkflowRequest, "", "", *metrics, artifactUploaded, err)
	}
	log.Println(util.DEVTRON, " Build artifact completed", "dest", dest, "err", err)
	return dest, err
}

func extractDigest(ciCdRequest *helper.CiCdTriggerEvent, dest string, metrics *helper.CIMetrics, artifactUploaded bool) (string, error) {
	ciBuildConfigBean := ciCdRequest.CommonWorkflowRequest.CiBuildConfig
	isBuildX := ciBuildConfigBean != nil && ciBuildConfigBean.DockerBuildConfig != nil && ciBuildConfigBean.DockerBuildConfig.CheckForBuildX()
	var digest string
	var err error
	if isBuildX {
		digest, err = helper.ExtractDigestForBuildx(dest)
	} else {
		util.LogStage("docker push")
		// push to dest
		log.Println(util.DEVTRON, "Docker push Artifact", "dest", dest)
		pushArtifact(ciCdRequest, dest, digest, metrics, artifactUploaded)
		digest, err = helper.ExtractDigestUsingPull(dest)
	}
	return digest, err
}

func runPostCiSteps(ciCdRequest *helper.CiCdTriggerEvent, scriptEnvs map[string]string, refStageMap map[int][]*helper.StepObject, preCiStageOutVariable map[int]map[string]*helper.VariableObject, metrics helper.CIMetrics, artifactUploaded bool, dest string, digest string) error {
	util.LogStage("running POST-CI steps")
	// sending build success as true always as post-ci triggers only if ci gets success
	scriptEnvs[util.ENV_VARIABLE_BUILD_SUCCESS] = "true"
	scriptEnvs["DEST"] = dest
	scriptEnvs["DIGEST"] = digest
	// run post artifact processing
	_, step, err := RunCiCdSteps(STEP_TYPE_POST, ciCdRequest.CommonWorkflowRequest.PostCiSteps, refStageMap, scriptEnvs, preCiStageOutVariable)
	if err != nil {
		log.Println("error in running Post Ci Steps", "err", err)
		return sendFailureNotification(string(PostCi)+step.Name, ciCdRequest.CommonWorkflowRequest, "", "", metrics, artifactUploaded, err)
	}
	return nil
}

func runImageScanning(dest string, digest string, ciCdRequest *helper.CiCdTriggerEvent, metrics helper.CIMetrics, artifactUploaded bool) error {
	util.LogStage("IMAGE SCAN")
	log.Println(util.DEVTRON, " Image Scanning Started for digest", digest)
	scanEvent := &helper.ScanEvent{Image: dest, ImageDigest: digest, PipelineId: ciCdRequest.CommonWorkflowRequest.PipelineId, UserId: ciCdRequest.CommonWorkflowRequest.TriggeredBy}
	scanEvent.DockerRegistryId = ciCdRequest.CommonWorkflowRequest.DockerRegistryId
	err := helper.SendEventToClairUtility(scanEvent)
	if err != nil {
		log.Println("error in running Image Scan", "err", err)
		err = sendFailureNotification(string(Scan), ciCdRequest.CommonWorkflowRequest, digest, dest, metrics, artifactUploaded, err)
		return err
	}
	log.Println(util.DEVTRON, "Image scanning completed with scanEvent", scanEvent)
	return nil
}

func getImageDestAndDigest(ciCdRequest *helper.CiCdTriggerEvent, metrics helper.CIMetrics, scriptEnvs map[string]string, refStageMap map[int][]*helper.StepObject, preCiStageOutVariable map[int]map[string]*helper.VariableObject, artifactUploaded bool) (string, string, error) {
	dest, err := runBuildArtifact(ciCdRequest, &metrics, refStageMap, scriptEnvs, artifactUploaded, preCiStageOutVariable)
	if err != nil {
		return "", "", err
	}
	digest, err := extractDigest(ciCdRequest, dest, &metrics, artifactUploaded)
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

// extractOutResultsIfExists will unmarshall the results from file(json) (if file exist) into ImageDetailsFromCR
func extractOutResultsIfExists() (*helper.ImageDetailsFromCR, error) {
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
	imageDetailsFromCr := helper.ImageDetailsFromCR{}
	err = json.Unmarshal(file, &imageDetailsFromCr)
	if err != nil {
		log.Println("error in unmarshalling imageDetailsFromCr results", "err", err.Error())
		return nil, err
	}
	return &imageDetailsFromCr, nil

}

func makeDockerfile(config *helper.DockerBuildConfig, checkoutPath string) error {
	dockerfileContent := config.DockerfileContent
	dockerfilePath := filepath.Join(util.WORKINGDIR, checkoutPath, "./Dockerfile")
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
	e := helper.SendEvents(ciRequest, digest, image, ciMetrics, artifactUploaded, failureMessage, nil)
	if e != nil {
		log.Println(e)
		return e
	}
	return &helper.CiStageError{Err: err}
}

func pushArtifact(ciCdRequest *helper.CiCdTriggerEvent, dest string, digest string, metrics *helper.CIMetrics, artifactUploaded bool) error {
	imageRetryCountValue := ciCdRequest.CommonWorkflowRequest.ImageRetryCount
	imageRetryIntervalValue := ciCdRequest.CommonWorkflowRequest.ImageRetryInterval
	var err error
	for i := 0; i < imageRetryCountValue+1; i++ {
		if i != 0 {
			time.Sleep(time.Duration(imageRetryIntervalValue) * time.Second)
		}
		err = helper.PushArtifact(dest)
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
