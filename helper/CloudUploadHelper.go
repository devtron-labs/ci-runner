package helper

import (
	"github.com/devtron-labs/ci-runner/util"
	"github.com/devtron-labs/common-lib/blob-storage"
	"log"
)

// UploadFileToCloud
// Uploads the source file to the destination key of configured blob storage /**
func UploadFileToCloud(cloudHelperBaseConfig *util.CloudHelperBaseConfig, sourceFilePath string, destinationKey string) error {

	blobStorageService := blob_storage.NewBlobStorageServiceImpl(nil)
	request := createBlobStorageRequest(cloudHelperBaseConfig, sourceFilePath, destinationKey)

	return blobStorageService.PutWithCommand(request)
}

func createBlobStorageRequest(cloudHelperBaseConfig *util.CloudHelperBaseConfig, sourceKey string, destinationKey string) *blob_storage.BlobStorageRequest {
	if cloudHelperBaseConfig.UseExternalClusterBlob {
		UpdateCloudHelperBaseConfigFromEnv(cloudHelperBaseConfig)
	}
	var awsS3BaseConfig *blob_storage.AwsS3BaseConfig
	if cloudHelperBaseConfig.BlobStorageS3Config != nil {
		awsS3BaseConfig = util.GetBlobStorageBaseS3Config(cloudHelperBaseConfig.BlobStorageS3Config, cloudHelperBaseConfig.BlobStorageObjectType)
	}
	var azureBlobBaseConfig *blob_storage.AzureBlobBaseConfig
	if cloudHelperBaseConfig.AzureBlobConfig != nil {
		azureBlobBaseConfig = util.GetBlobStorageBaseAzureConfig(cloudHelperBaseConfig.AzureBlobConfig, cloudHelperBaseConfig.BlobStorageObjectType)
	}
	var gcpBlobBaseConfig *blob_storage.GcpBlobBaseConfig
	if cloudHelperBaseConfig.GcpBlobConfig != nil {
		gcpBlobBaseConfig = util.GetBlobStorageBaseGcpConfig(cloudHelperBaseConfig.GcpBlobConfig, cloudHelperBaseConfig.BlobStorageObjectType)
	}
	request := &blob_storage.BlobStorageRequest{
		StorageType:         cloudHelperBaseConfig.CloudProvider,
		SourceKey:           sourceKey,
		DestinationKey:      destinationKey,
		AzureBlobBaseConfig: azureBlobBaseConfig,
		AwsS3BaseConfig:     awsS3BaseConfig,
		GcpBlobBaseConfig:   gcpBlobBaseConfig,
	}
	return request
}

func UpdateCloudHelperBaseConfigFromEnv(cloudHelperBaseConfig *util.CloudHelperBaseConfig) {
	log.Println(util.DEVTRON, "using external cluster blob")
	blobStorageConfig, err := util.GetBlobStorageConfig()
	if err != nil {
		log.Println(util.DEVTRON, "error in getting blob storage config, err : ", err)
	}
	log.Println("external cluster cloud provider: ", blobStorageConfig.CloudProvider)
	if blobStorageConfig == nil {
		return
	}
	setConfigForBlobType(cloudHelperBaseConfig, blobStorageConfig)
}

func setConfigForBlobType(cloudHelperBaseConfig *util.CloudHelperBaseConfig, blobStorageConfig *util.BlobStorageConfig) {
	cloudHelperBaseConfig.CloudProvider = blobStorageConfig.CloudProvider
	switch blobStorageConfig.CloudProvider {
	case blob_storage.BLOB_STORAGE_S3:
		cloudHelperBaseConfig.SetAwsBlobStorageS3Config(blobStorageConfig)
	case blob_storage.BLOB_STORAGE_GCP:
		cloudHelperBaseConfig.SetGcpBlobStorageConfig(blobStorageConfig)
	case blob_storage.BLOB_STORAGE_AZURE:
		cloudHelperBaseConfig.SetAzureBlobStorageConfig(blobStorageConfig)
	default:
		if cloudHelperBaseConfig.StorageModuleConfigured {
			log.Println(util.DEVTRON, "blob storage not supported, blobStorage: ", blobStorageConfig.CloudProvider)
		}
	}
}
