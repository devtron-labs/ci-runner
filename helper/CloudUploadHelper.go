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

	CheckForExtClusterBlobAndUpdateCloudHelperBaseConfig(cloudHelperBaseConfig)
	var awsS3BaseConfig *blob_storage.AwsS3BaseConfig
	if cloudHelperBaseConfig.BlobStorageS3Config != nil {
		awsS3BaseConfig = &blob_storage.AwsS3BaseConfig{
			AccessKey:         cloudHelperBaseConfig.BlobStorageS3Config.AccessKey,
			Passkey:           cloudHelperBaseConfig.BlobStorageS3Config.Passkey,
			EndpointUrl:       cloudHelperBaseConfig.BlobStorageS3Config.EndpointUrl,
			IsInSecure:        cloudHelperBaseConfig.BlobStorageS3Config.IsInSecure,
			BucketName:        cloudHelperBaseConfig.BlobStorageS3Config.CiArtifactBucketName,
			Region:            cloudHelperBaseConfig.BlobStorageS3Config.CiArtifactRegion,
			VersioningEnabled: cloudHelperBaseConfig.BlobStorageS3Config.CiArtifactBucketVersioning,
		}
	}
	var azureBlobBaseConfig *blob_storage.AzureBlobBaseConfig
	if cloudHelperBaseConfig.AzureBlobConfig != nil {
		azureBlobBaseConfig = &blob_storage.AzureBlobBaseConfig{
			AccountKey:        cloudHelperBaseConfig.AzureBlobConfig.AccountKey,
			AccountName:       cloudHelperBaseConfig.AzureBlobConfig.AccountName,
			Enabled:           cloudHelperBaseConfig.AzureBlobConfig.Enabled,
			BlobContainerName: cloudHelperBaseConfig.AzureBlobConfig.BlobContainerArtifact,
		}
	}
	var gcpBlobBaseConfig *blob_storage.GcpBlobBaseConfig
	if cloudHelperBaseConfig.GcpBlobConfig != nil {
		gcpBlobBaseConfig = &blob_storage.GcpBlobBaseConfig{
			CredentialFileJsonData: cloudHelperBaseConfig.GcpBlobConfig.CredentialFileJsonData,
			BucketName:             cloudHelperBaseConfig.GcpBlobConfig.ArtifactBucketName,
		}
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

func CheckForExtClusterBlobAndUpdateCloudHelperBaseConfig(cloudHelperBaseConfig *util.CloudHelperBaseConfig) {
	if cloudHelperBaseConfig.UseExternalClusterBlob {
		log.Println(util.DEVTRON, "using external cluster blob")
		blobStorageConfig, err := GetBlobStorageConfig()
		if err != nil {
			log.Println(util.DEVTRON, "error in getting blob storage config, err : ", err)
		}
		switch blobStorageConfig.CloudProvider {
		case util.BlobStorageS3:
			//here we are only uploading logs
			blobStorageS3Config := &blob_storage.BlobStorageS3Config{
				AccessKey:   blobStorageConfig.S3AccessKey,
				Passkey:     blobStorageConfig.S3SecretKey,
				EndpointUrl: blobStorageConfig.S3Endpoint,
				IsInSecure:  blobStorageConfig.S3EndpointInsecure,
			}
			cloudHelperBaseConfig.BlobStorageS3Config = blobStorageS3Config
		case util.BlobStorageGcp:
			gcpBlobConfig := &blob_storage.GcpBlobConfig{
				CredentialFileJsonData: blobStorageConfig.GcpBlobStorageCredentialJson,
			}
			cloudHelperBaseConfig.GcpBlobConfig = gcpBlobConfig
		case util.BlobStorageAzure:
			azureBlobConfig := &blob_storage.AzureBlobConfig{
				Enabled:     blobStorageConfig.CloudProvider == blob_storage.BLOB_STORAGE_AZURE,
				AccountName: blobStorageConfig.AzureAccountName,
				AccountKey:  blobStorageConfig.AzureAccountKey,
			}
			cloudHelperBaseConfig.AzureBlobConfig = azureBlobConfig
		default:
			if cloudHelperBaseConfig.StorageModuleConfigured {
				log.Println(util.DEVTRON, "blob storage not supported, blobStorage: ", blobStorageConfig.CloudProvider)
			}
		}
	}
}
