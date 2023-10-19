package helper

import (
	"github.com/devtron-labs/ci-runner/util"
	"github.com/devtron-labs/common-lib/blob-storage"
	"log"
)

// UploadFileToCloud
// Uploads the source file to the destination key of configured blob storage /**
func UploadFileToCloud(cloudHelperBaseConfig *util.CloudHelperBaseConfig, sourceFilePath string, destinationKey string) error {
	if cloudHelperBaseConfig.UseExternalClusterBlob {
		log.Println(util.DEVTRON, "uploading blob in external cluster blob")
		blobStorageConfig, err := GetBlobStorageConfig()
		if err != nil {
			log.Println(util.DEVTRON, "error in getting blob storage config, err : ", err)
		}
		switch blobStorageConfig.CloudProvider {
		case util.BlobStorageS3:
			//here we are only uploading logs
			blobStorageS3Config := &blob_storage.BlobStorageS3Config{
				AccessKey:                  blobStorageConfig.S3AccessKey,
				Passkey:                    blobStorageConfig.S3SecretKey,
				EndpointUrl:                blobStorageConfig.S3Endpoint,
				IsInSecure:                 blobStorageConfig.S3EndpointInsecure,
				CiCacheBucketName:          blobStorageConfig.S3CacheBucketName,
				CiCacheRegion:              blobStorageConfig.S3CacheRegion,
				CiCacheBucketVersioning:    blobStorageConfig.S3BucketVersioned,
				CiArtifactBucketName:       blobStorageConfig.S3ArtifactBucketName,
				CiArtifactRegion:           blobStorageConfig.S3ArtifactRegion,
				CiArtifactBucketVersioning: blobStorageConfig.S3BucketVersioned,
				CiLogBucketName:            blobStorageConfig.S3LogBucketName,
				CiLogRegion:                blobStorageConfig.S3LogRegion,
				CiLogBucketVersioning:      blobStorageConfig.S3BucketVersioned,
			}
			cloudHelperBaseConfig.BlobStorageS3Config = blobStorageS3Config
		case util.BlobStorageGcp:
			gcpBlobConfig := &blob_storage.GcpBlobConfig{
				CredentialFileJsonData: blobStorageConfig.GcpBlobStorageCredentialJson,
				CacheBucketName:        blobStorageConfig.GcpCacheBucketName,
				ArtifactBucketName:     blobStorageConfig.GcpArtifactBucketName,
				LogBucketName:          blobStorageConfig.GcpLogBucketName,
			}
			cloudHelperBaseConfig.GcpBlobConfig = gcpBlobConfig
		case util.BlobStorageAzure:
			azureBlobConfig := &blob_storage.AzureBlobConfig{
				Enabled:               true,
				AccountName:           blobStorageConfig.AzureAccountName,
				BlobContainerCiCache:  blobStorageConfig.AzureBlobContainerCiCache,
				AccountKey:            blobStorageConfig.AzureAccountKey,
				BlobContainerCiLog:    blobStorageConfig.AzureBlobContainerCiLog,
				BlobContainerArtifact: blobStorageConfig.AzureBlobContainerCiLog,
			}
			cloudHelperBaseConfig.AzureBlobConfig = azureBlobConfig
			//blobStorageS3Config = &blob_storage.BlobStorageS3Config{
			//	EndpointUrl:     blobStorageConfig.AzureGatewayUrl,
			//	IsInSecure:      blobStorageConfig.AzureGatewayConnectionInsecure,
			//	CiLogBucketName: blobStorageConfig.AzureBlobContainerCiLog,
			//	CiLogRegion:     "",
			//	AccessKey:       blobStorageConfig.AzureAccountName,
			//}
		default:
			if cloudHelperBaseConfig.StorageModuleConfigured {
				log.Println(util.DEVTRON, "blob storage not supported, blobStorage: ", blobStorageConfig.CloudProvider)
			}
		}
	}
	blobStorageService := blob_storage.NewBlobStorageServiceImpl(nil)
	request := createBlobStorageRequest(cloudHelperBaseConfig.CloudProvider, sourceFilePath, destinationKey, cloudHelperBaseConfig.BlobStorageS3Config, cloudHelperBaseConfig.AzureBlobConfig, cloudHelperBaseConfig.GcpBlobConfig)

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
