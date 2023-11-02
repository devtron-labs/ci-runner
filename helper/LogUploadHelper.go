package helper

import (
	"fmt"
	"github.com/devtron-labs/ci-runner/util"
	"log"
	"path"
)

// UploadLogs
// Checks of blob storage is configured, if yes, uploads the locally created log file to configured storage
func UploadLogs(cloudHelperBaseConfig *util.CloudHelperBaseConfig) {

	if !cloudHelperBaseConfig.StorageModuleConfigured {
		log.Println(util.DEVTRON, "not going to upload logs as storage module not configured...")
		return
	}

	err := UploadFileToCloud(cloudHelperBaseConfig, util.TmpLogLocation, path.Join(cloudHelperBaseConfig.BlobStorageLogKey, util.TmpLogLocation))
	if err != nil {
		fmt.Println("Failed to upload to blob storage with error", err)
		return
	}
}
