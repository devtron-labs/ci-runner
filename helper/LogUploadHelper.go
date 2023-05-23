package helper

import (
	"fmt"
	"github.com/devtron-labs/ci-runner/util"
	blob_storage "github.com/devtron-labs/common-lib/blob-storage"
	"log"
)

// UploadLogs
// Checks of blob storage is configured, if yes, uploads the locally created log file to configured storage
func UploadLogs(storageModuleConfigured bool, blogStorageLogKey string, cloudProvider blob_storage.BlobStorageType,
	blobStorageS3Config *blob_storage.BlobStorageS3Config, azureBlobConfig *blob_storage.AzureBlobConfig, gcpBlobConfig *blob_storage.GcpBlobConfig) {

	if !storageModuleConfigured {
		log.Println(util.DEVTRON, "not going to upload logs as storage module not configured...")
		return
	}

	//isEmpty, err := IsDirEmpty(util.TmpLogLocation)
	//if err != nil {
	//	log.Println(util.DEVTRON, "logs empty check error", err)
	//	return
	//} else if isEmpty {
	//	log.Println(util.DEVTRON, "log file is empty")
	//	return
	//}

	err := UploadFileToCloud(cloudProvider, util.TmpLogLocation, blogStorageLogKey, blobStorageS3Config, azureBlobConfig, gcpBlobConfig)
	if err != nil {
		fmt.Println("Failed to upload to blob storage with error", err)
		return
	}
}
