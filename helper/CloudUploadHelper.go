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
		UpdateCloudHelperBaseConfigForExtCluster(cloudHelperBaseConfig)
	}
	var awsS3BaseConfig *blob_storage.AwsS3BaseConfig
	if cloudHelperBaseConfig.BlobStorageS3Config != nil {
		awsS3BaseConfig = cloudHelperBaseConfig.BlobStorageS3Config.GetBlobStorageBaseS3Config(BlobStorageObjectTypeArtifact)
	}
	var azureBlobBaseConfig *blob_storage.AzureBlobBaseConfig
	if cloudHelperBaseConfig.AzureBlobConfig != nil {
		azureBlobBaseConfig = cloudHelperBaseConfig.AzureBlobConfig.GetBlobStorageBaseAzureConfig(BlobStorageObjectTypeArtifact)
	}
	var gcpBlobBaseConfig *blob_storage.GcpBlobBaseConfig
	if cloudHelperBaseConfig.GcpBlobConfig != nil {
		gcpBlobBaseConfig = cloudHelperBaseConfig.GcpBlobConfig.GetBlobStorageBaseGcpConfig(BlobStorageObjectTypeArtifact)
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

func UpdateCloudHelperBaseConfigForExtCluster(cloudHelperBaseConfig *util.CloudHelperBaseConfig) {
	log.Println(util.DEVTRON, "using external cluster blob")
	blobStorageConfig, err := GetBlobStorageConfig()
	if err != nil {
		log.Println(util.DEVTRON, "error in getting blob storage config, err : ", err)
	}
	switch blobStorageConfig.CloudProvider {
	case util.BlobStorageS3:
		//here we are only uploading logs
		cloudHelperBaseConfig.BlobStorageS3Config.AccessKey = blobStorageConfig.S3AccessKey
		cloudHelperBaseConfig.BlobStorageS3Config.Passkey = blobStorageConfig.S3SecretKey
		cloudHelperBaseConfig.BlobStorageS3Config.EndpointUrl = blobStorageConfig.S3Endpoint
		cloudHelperBaseConfig.BlobStorageS3Config.IsInSecure = blobStorageConfig.S3EndpointInsecure

	case util.BlobStorageGcp:
		cloudHelperBaseConfig.GcpBlobConfig.CredentialFileJsonData = blobStorageConfig.GcpBlobStorageCredentialJson

	case util.BlobStorageAzure:
		cloudHelperBaseConfig.AzureBlobConfig.Enabled = blobStorageConfig.CloudProvider == blob_storage.BLOB_STORAGE_AZURE
		cloudHelperBaseConfig.AzureBlobConfig.AccountName = blobStorageConfig.AzureAccountName
		cloudHelperBaseConfig.AzureBlobConfig.AccountKey = blobStorageConfig.AzureAccountKey
	default:
		if cloudHelperBaseConfig.StorageModuleConfigured {
			log.Println(util.DEVTRON, "blob storage not supported, blobStorage: ", blobStorageConfig.CloudProvider)
		}
	}
}
