package main

import (
	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
	"log"
	"os"
)

func HandleCDEvent(err error, ciCdRequest *helper.CiCdTriggerEvent, exitCode *int) {
	err = runCDStages(ciCdRequest)
	artifactUploadErr := collectAndUploadCDArtifacts(ciCdRequest.CdRequest)
	if err != nil || artifactUploadErr != nil {
		log.Println(err)
		*exitCode = util.DefaultErrorCode
	}
}

func collectAndUploadCDArtifacts(cdRequest *helper.CdRequest) error {

	if len(cdRequest.PrePostDeploySteps) > 0 {
		_, err := helper.ZipAndUpload(cdRequest.BlobStorageConfigured, cdRequest.BlobStorageS3Config, cdRequest.ArtifactFileName, cdRequest.CloudProvider, cdRequest.AzureBlobConfig, cdRequest.GcpBlobConfig)
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
	return helper.UploadArtifact(cdRequest.BlobStorageConfigured, artifactFiles, cdRequest.BlobStorageS3Config, cdRequest.ArtifactFileName, cdRequest.CloudProvider, cdRequest.AzureBlobConfig, cdRequest.GcpBlobConfig)
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
	log.Println(util.DEVTRON, " git")
	err = helper.CloneAndCheckout(cicdRequest.CdRequest.CiProjectDetails)
	if err != nil {
		log.Println(util.DEVTRON, "clone err: ", err)
		return err
	}
	log.Println(util.DEVTRON, " /git")

	// Start docker daemon
	log.Println(util.DEVTRON, " docker-start")
	helper.StartDockerDaemon(cicdRequest.CdRequest.DockerConnection, cicdRequest.CdRequest.DockerRegistryURL, cicdRequest.CdRequest.DockerCert, cicdRequest.CdRequest.DefaultAddressPoolBaseCidr, cicdRequest.CdRequest.DefaultAddressPoolSize, -1)

	err = helper.DockerLogin(&helper.DockerCredentials{
		DockerUsername:     cicdRequest.CdRequest.DockerUsername,
		DockerPassword:     cicdRequest.CdRequest.DockerPassword,
		AwsRegion:          cicdRequest.CdRequest.AwsRegion,
		AccessKey:          cicdRequest.CdRequest.AccessKey,
		SecretKey:          cicdRequest.CdRequest.SecretKey,
		DockerRegistryURL:  cicdRequest.CdRequest.DockerRegistryURL,
		DockerRegistryType: cicdRequest.CdRequest.DockerRegistryType,
	})
	if err != nil {
		return err
	}

	scriptEnvs, err := getGlobalEnvVariables(cicdRequest)

	if len(cicdRequest.CdRequest.PrePostDeploySteps) > 0 {
		refStageMap := make(map[int][]*helper.StepObject)
		for _, ref := range cicdRequest.CdRequest.RefPlugins {
			refStageMap[ref.Id] = ref.Steps
		}
		_, _, err = RunCiCdSteps(STEP_TYPE_PRE, cicdRequest.CdRequest.PrePostDeploySteps, refStageMap, scriptEnvs, nil)

	} else {

		// Get devtron-cd yaml
		taskYaml, err := helper.ToTaskYaml([]byte(cicdRequest.CdRequest.StageYaml))
		if err != nil {
			log.Println(err)
			return err
		}
		cicdRequest.CdRequest.TaskYaml = taskYaml

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
	if !cicdRequest.CdRequest.IsDryRun {
		log.Println(util.DEVTRON, " event")
		err = helper.SendCDEvent(cicdRequest.CdRequest)
		if err != nil {
			log.Println(err)
			return err
		}
		log.Println(util.DEVTRON, " /event")
	}
	err = helper.StopDocker()
	if err != nil {
		log.Println("err", err)
		return err
	}
	return nil
}
