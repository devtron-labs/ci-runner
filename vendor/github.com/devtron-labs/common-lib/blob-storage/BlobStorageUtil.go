package blob_storage

func CreateBlobStorageRequest(cloudProvider BlobStorageType, sourceKey string, destinationKey string, blobStorageS3Config *BlobStorageS3Config, azureBlobConfig *AzureBlobConfig) *BlobStorageRequest {
	var awsS3BaseConfig *AwsS3BaseConfig
	if blobStorageS3Config != nil {
		awsS3BaseConfig = &AwsS3BaseConfig{
			AccessKey:   blobStorageS3Config.AccessKey,
			Passkey:     blobStorageS3Config.Passkey,
			EndpointUrl: blobStorageS3Config.EndpointUrl,
			BucketName:  blobStorageS3Config.CiArtifactBucketName,
			Region:      blobStorageS3Config.CiArtifactRegion,
		}
	}

	var azureBlobBaseConfig *AzureBlobBaseConfig
	if azureBlobConfig != nil {
		azureBlobBaseConfig = &AzureBlobBaseConfig{
			AccountKey:        azureBlobConfig.AccountKey,
			AccountName:       azureBlobConfig.AccountName,
			Enabled:           azureBlobConfig.Enabled,
			BlobContainerName: azureBlobConfig.BlobContainerArtifact,
		}
	}
	request := &BlobStorageRequest{
		StorageType:         cloudProvider,
		SourceKey:           sourceKey,
		DestinationKey:      destinationKey,
		AzureBlobBaseConfig: azureBlobBaseConfig,
		AwsS3BaseConfig:     awsS3BaseConfig,
	}
	return request
}
