package helper

import "github.com/devtron-labs/common-lib/blob-storage"

// UploadFileToCloud
// Uploads the source file to the destination key of configured blob storage /**
func UploadFileToCloud(cloudProvider blob_storage.BlobStorageType, sourceFilePath string, destinationKey string,
	blobStorageS3Config *blob_storage.BlobStorageS3Config, azureBlobConfig *blob_storage.AzureBlobConfig,
	gcpBlobConfig *blob_storage.GcpBlobConfig) error {

	blobStorageService := blob_storage.NewBlobStorageServiceImpl(nil)
	request := createBlobStorageRequest(cloudProvider, sourceFilePath, destinationKey, blobStorageS3Config, azureBlobConfig, gcpBlobConfig)

	return blobStorageService.PutWithCommand(request)
}

func createBlobStorageRequest(cloudProvider blob_storage.BlobStorageType, sourceKey string, destinationKey string,
	blobStorageS3Config *blob_storage.BlobStorageS3Config, azureBlobConfig *blob_storage.AzureBlobConfig,
	gcpBlobConfig *blob_storage.GcpBlobConfig) *blob_storage.BlobStorageRequest {
	var awsS3BaseConfig *blob_storage.AwsS3BaseConfig
	if blobStorageS3Config != nil {
		awsS3BaseConfig = &blob_storage.AwsS3BaseConfig{
			AccessKey:         blobStorageS3Config.AccessKey,
			Passkey:           blobStorageS3Config.Passkey,
			EndpointUrl:       blobStorageS3Config.EndpointUrl,
			IsInSecure:        blobStorageS3Config.IsInSecure,
			BucketName:        blobStorageS3Config.CiArtifactBucketName,
			Region:            blobStorageS3Config.CiArtifactRegion,
			VersioningEnabled: blobStorageS3Config.CiArtifactBucketVersioning,
		}
	}

	var azureBlobBaseConfig *blob_storage.AzureBlobBaseConfig
	if azureBlobConfig != nil {
		azureBlobBaseConfig = &blob_storage.AzureBlobBaseConfig{
			AccountKey:        azureBlobConfig.AccountKey,
			AccountName:       azureBlobConfig.AccountName,
			Enabled:           azureBlobConfig.Enabled,
			BlobContainerName: azureBlobConfig.BlobContainerArtifact,
		}
	}
	var gcpBlobBaseConfig *blob_storage.GcpBlobBaseConfig
	if gcpBlobConfig != nil {
		gcpBlobBaseConfig = &blob_storage.GcpBlobBaseConfig{
			CredentialFileJsonData: gcpBlobConfig.CredentialFileJsonData,
			BucketName:             gcpBlobConfig.ArtifactBucketName,
		}
	}
	request := &blob_storage.BlobStorageRequest{
		StorageType:         cloudProvider,
		SourceKey:           sourceKey,
		DestinationKey:      destinationKey,
		AzureBlobBaseConfig: azureBlobBaseConfig,
		AwsS3BaseConfig:     awsS3BaseConfig,
		GcpBlobBaseConfig:   gcpBlobBaseConfig,
	}
	return request
}
