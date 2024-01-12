package ciCdStageExecutor

import (
	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
	"log"
	"os"
)

type CdStage struct {
}

func NewCdStage() *CdStage {
	return &CdStage{}
}

func (impl CdStage) HandleCDEvent(ciCdRequest *helper.CiCdTriggerEvent, exitCode *int) {
	err := runCDStages(ciCdRequest)
	artifactUploadErr := collectAndUploadCDArtifacts(ciCdRequest.CommonWorkflowRequest)
	if err != nil || artifactUploadErr != nil {
		log.Println(err)
		*exitCode = util.DefaultErrorCode
	}

}

func collectAndUploadCDArtifacts(cdRequest *helper.CommonWorkflowRequest) error {
	cloudHelperBaseConfig := cdRequest.GetCloudHelperBaseConfig(util.BlobStorageObjectTypeArtifact)
	if cdRequest.PrePostDeploySteps != nil && len(cdRequest.PrePostDeploySteps) > 0 {
		_, err := helper.ZipAndUpload(cloudHelperBaseConfig, cdRequest.CiArtifactFileName)
		return err
	}

	// to support stage YAML outputs
	artifactFiles := make(map[string]string)
	var allTasks []*helper.Task
	if cdRequest.TaskYaml != nil {
		for _, pc := range cdRequest.TaskYaml.CdPipelineConfig {
			for _, t := range append(pc.BeforeTasks, pc.AfterTasks...) {
				allTasks = append(allTasks, t)
			}
		}
	}
	for _, task := range allTasks {
		if task.RunStatus {
			if _, err := os.Stat(task.OutputLocation); os.IsNotExist(err) { // Ignore if no file/folder
				log.Println(util.DEVTRON, "artifact not found ", err)
				continue
			}
			artifactFiles[task.Name] = task.OutputLocation
		}
	}
	log.Println(util.DEVTRON, " artifacts", artifactFiles)
	return helper.UploadArtifact(cloudHelperBaseConfig, artifactFiles, cdRequest.CiArtifactFileName)
}

func runCDStages(cicdRequest *helper.CiCdTriggerEvent) error {
	err := os.Chdir("/")
	if err != nil {
		return err
	}

	if _, err := os.Stat(util.WORKINGDIR); os.IsNotExist(err) {
		_ = os.Mkdir(util.WORKINGDIR, os.ModeDir)
	}
	err = os.Chdir(util.WORKINGDIR)
	if err != nil {
		return err
	}
	// git handling
	// we are skipping clone and checkout in case of ci job type poll cr images plugin does not require it.(ci-job)
	skipCheckout := cicdRequest.CommonWorkflowRequest.CiPipelineType == helper.CI_JOB
	if !skipCheckout {
		log.Println(util.DEVTRON, " git")
		err = helper.CloneAndCheckout(cicdRequest.CommonWorkflowRequest.CiProjectDetails)
		if err != nil {
			log.Println(util.DEVTRON, "clone err: ", err)
			return err
		}
	}
	log.Println(util.DEVTRON, " /git")
	// Start docker daemon
	log.Println(util.DEVTRON, " docker-start")
	helper.StartDockerDaemon(cicdRequest.CommonWorkflowRequest.DockerConnection, cicdRequest.CommonWorkflowRequest.DockerRegistryURL, cicdRequest.CommonWorkflowRequest.DockerCert, cicdRequest.CommonWorkflowRequest.DefaultAddressPoolBaseCidr, cicdRequest.CommonWorkflowRequest.DefaultAddressPoolSize, -1)

	err = helper.DockerLogin(&helper.DockerCredentials{
		DockerUsername:     cicdRequest.CommonWorkflowRequest.DockerUsername,
		DockerPassword:     cicdRequest.CommonWorkflowRequest.DockerPassword,
		AwsRegion:          cicdRequest.CommonWorkflowRequest.AwsRegion,
		AccessKey:          cicdRequest.CommonWorkflowRequest.AccessKey,
		SecretKey:          cicdRequest.CommonWorkflowRequest.SecretKey,
		DockerRegistryURL:  cicdRequest.CommonWorkflowRequest.DockerRegistryURL,
		DockerRegistryType: cicdRequest.CommonWorkflowRequest.DockerRegistryType,
	})
	if err != nil {
		return err
	}

	scriptEnvs, err := getGlobalEnvVariables(cicdRequest)

	if len(cicdRequest.CommonWorkflowRequest.PrePostDeploySteps) > 0 {
		refStageMap := make(map[int][]*helper.StepObject)
		for _, ref := range cicdRequest.CommonWorkflowRequest.RefPlugins {
			refStageMap[ref.Id] = ref.Steps
		}
		scriptEnvs["DEST"] = cicdRequest.CommonWorkflowRequest.CiArtifactDTO.Image
		scriptEnvs["DIGEST"] = cicdRequest.CommonWorkflowRequest.CiArtifactDTO.ImageDigest
		var stage = StepType(cicdRequest.CommonWorkflowRequest.StageType)
		_, _, err = RunCiCdSteps(stage, cicdRequest.CommonWorkflowRequest.PrePostDeploySteps, refStageMap, scriptEnvs, nil)
		if err != nil {
			return err
		}
	} else {

		// Get devtron-cd yaml
		taskYaml, err := helper.ToTaskYaml([]byte(cicdRequest.CommonWorkflowRequest.StageYaml))
		if err != nil {
			log.Println(err)
			return err
		}
		cicdRequest.CommonWorkflowRequest.TaskYaml = taskYaml

		// run post artifact processing
		log.Println(util.DEVTRON, " stage yaml", taskYaml)
		var tasks []*helper.Task
		for _, t := range taskYaml.CdPipelineConfig {
			tasks = append(tasks, t.BeforeTasks...)
			tasks = append(tasks, t.AfterTasks...)
		}

		if err != nil {
			return err
		}
		err = RunCdStageTasks(tasks, scriptEnvs)
		if err != nil {
			return err
		}
	}

	// dry run flag indicates that ci runner image is being run from external helm chart
	if !cicdRequest.CommonWorkflowRequest.IsDryRun {
		log.Println(util.DEVTRON, " event")
		err = helper.SendCDEvent(cicdRequest.CommonWorkflowRequest)
		if err != nil {
			log.Println(err)
			return err
		}
		log.Println(util.DEVTRON, " /event")
	}
	err = helper.StopDocker()
	if err != nil {
		log.Println("error while stopping docker", err)
		return err
	}
	return nil
}
