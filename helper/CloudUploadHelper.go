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

func UpdateCloudHelperBaseConfigForExtCluster(cloudHelperBaseConfig *util.CloudHelperBaseConfig) {
	log.Println(util.DEVTRON, "using external cluster blob")
	blobStorageConfig, err := GetBlobStorageConfig()
	if err != nil {
		log.Println(util.DEVTRON, "error in getting blob storage config, err : ", err)
	}
	log.Println(util.DEVTRON, "external cluster cloud provider: ", blobStorageConfig.CloudProvider)
	if blobStorageConfig != nil {
		cloudHelperBaseConfig.CloudProvider = blobStorageConfig.CloudProvider
		switch blobStorageConfig.CloudProvider {
		case util.BlobStorageS3:
			cloudHelperBaseConfig.BlobStorageS3Config = &blob_storage.BlobStorageS3Config{
				AccessKey:                  blobStorageConfig.S3AccessKey,
				Passkey:                    blobStorageConfig.S3SecretKey,
				EndpointUrl:                blobStorageConfig.S3Endpoint,
				IsInSecure:                 blobStorageConfig.S3EndpointInsecure,
				CiLogBucketName:            blobStorageConfig.CdDefaultBuildLogsBucket,
				CiLogRegion:                blobStorageConfig.CdDefaultCdLogsBucketRegion,
				CiLogBucketVersioning:      blobStorageConfig.S3BucketVersioned,
				CiCacheBucketName:          blobStorageConfig.DefaultCacheBucket,
				CiCacheRegion:              blobStorageConfig.DefaultCacheBucketRegion,
				CiCacheBucketVersioning:    blobStorageConfig.S3BucketVersioned,
				CiArtifactBucketName:       blobStorageConfig.CdDefaultBuildLogsBucket,
				CiArtifactRegion:           blobStorageConfig.CdDefaultCdLogsBucketRegion,
				CiArtifactBucketVersioning: blobStorageConfig.S3BucketVersioned,
			}
		case util.BlobStorageGcp:
			cloudHelperBaseConfig.GcpBlobConfig = &blob_storage.GcpBlobConfig{
				CredentialFileJsonData: blobStorageConfig.GcpBlobStorageCredentialJson,
				CacheBucketName:        blobStorageConfig.DefaultCacheBucket,
				LogBucketName:          blobStorageConfig.CdDefaultBuildLogsBucket,
				ArtifactBucketName:     blobStorageConfig.CdDefaultBuildLogsBucket,
			}
		case util.BlobStorageAzure:
			cloudHelperBaseConfig.AzureBlobConfig = &blob_storage.AzureBlobConfig{
				Enabled:               blobStorageConfig.CloudProvider == blob_storage.BLOB_STORAGE_AZURE,
				AccountName:           blobStorageConfig.AzureAccountName,
				BlobContainerCiLog:    blobStorageConfig.AzureBlobContainerCiLog,
				BlobContainerCiCache:  blobStorageConfig.AzureBlobContainerCiCache,
				BlobContainerArtifact: blobStorageConfig.AzureBlobContainerCiLog,
				AccountKey:            blobStorageConfig.AzureAccountKey,
			}
		default:
			if cloudHelperBaseConfig.StorageModuleConfigured {
				log.Println(util.DEVTRON, "blob storage not supported, blobStorage: ", blobStorageConfig.CloudProvider)
			}
		}

	}
}
