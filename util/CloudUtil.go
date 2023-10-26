package util

import (
	"github.com/devtron-labs/ci-runner/helper"
	blob_storage "github.com/devtron-labs/common-lib/blob-storage"
)

const (
	BlobStorageObjectTypeCache    = "cache"
	BlobStorageObjectTypeArtifact = "artifact"
	BlobStorageObjectTypeLog      = "log"
)

type CloudHelperBaseConfig struct {
	StorageModuleConfigured bool
	BlobStorageLogKey       string
	CloudProvider           blob_storage.BlobStorageType
	UseExternalClusterBlob  bool
	BlobStorageS3Config     *blob_storage.BlobStorageS3Config
	AzureBlobConfig         *blob_storage.AzureBlobConfig
	GcpBlobConfig           *blob_storage.GcpBlobConfig
	BlobStorageObjectType   string
}

func (c *CloudHelperBaseConfig) SetAwsBlobStorageS3Config(blobStorageConfig *helper.BlobStorageConfig) {
	c.BlobStorageS3Config = &blob_storage.BlobStorageS3Config{
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
}

func (c *CloudHelperBaseConfig) SetAzureBlobStorageConfig(blobStorageConfig *helper.BlobStorageConfig) {
	c.AzureBlobConfig = &blob_storage.AzureBlobConfig{
		Enabled:               blobStorageConfig.CloudProvider == blob_storage.BLOB_STORAGE_AZURE,
		AccountName:           blobStorageConfig.AzureAccountName,
		BlobContainerCiLog:    blobStorageConfig.AzureBlobContainerCiLog,
		BlobContainerCiCache:  blobStorageConfig.AzureBlobContainerCiCache,
		BlobContainerArtifact: blobStorageConfig.AzureBlobContainerCiLog,
		AccountKey:            blobStorageConfig.AzureAccountKey,
	}
}

func (c *CloudHelperBaseConfig) SetGcpBlobStorageConfig(blobStorageConfig *helper.BlobStorageConfig) {
	c.GcpBlobConfig = &blob_storage.GcpBlobConfig{
		CredentialFileJsonData: blobStorageConfig.GcpBlobStorageCredentialJson,
		CacheBucketName:        blobStorageConfig.DefaultCacheBucket,
		LogBucketName:          blobStorageConfig.CdDefaultBuildLogsBucket,
		ArtifactBucketName:     blobStorageConfig.CdDefaultBuildLogsBucket,
	}
}

func GetBlobStorageBaseS3Config(b *blob_storage.BlobStorageS3Config, blobStorageObjectType string) *blob_storage.AwsS3BaseConfig {
	awsS3BaseConfig := &blob_storage.AwsS3BaseConfig{
		AccessKey:   b.AccessKey,
		Passkey:     b.Passkey,
		EndpointUrl: b.EndpointUrl,
		IsInSecure:  b.IsInSecure,
	}
	switch blobStorageObjectType {
	case BlobStorageObjectTypeCache:
		awsS3BaseConfig.BucketName = b.CiCacheBucketName
		awsS3BaseConfig.Region = b.CiCacheRegion
		awsS3BaseConfig.VersioningEnabled = b.CiCacheBucketVersioning
		return awsS3BaseConfig
	case BlobStorageObjectTypeLog:
		awsS3BaseConfig.BucketName = b.CiLogBucketName
		awsS3BaseConfig.Region = b.CiLogRegion
		awsS3BaseConfig.VersioningEnabled = b.CiLogBucketVersioning
		return awsS3BaseConfig
	case BlobStorageObjectTypeArtifact:
		awsS3BaseConfig.BucketName = b.CiArtifactBucketName
		awsS3BaseConfig.Region = b.CiArtifactRegion
		awsS3BaseConfig.VersioningEnabled = b.CiArtifactBucketVersioning
		return awsS3BaseConfig
	default:
		return nil
	}
}

func GetBlobStorageBaseAzureConfig(b *blob_storage.AzureBlobConfig, blobStorageObjectType string) *blob_storage.AzureBlobBaseConfig {
	azureBlobBaseConfig := &blob_storage.AzureBlobBaseConfig{
		Enabled:     b.Enabled,
		AccountName: b.AccountName,
		AccountKey:  b.AccountKey,
	}
	switch blobStorageObjectType {
	case BlobStorageObjectTypeCache:
		azureBlobBaseConfig.BlobContainerName = b.BlobContainerCiCache
		return azureBlobBaseConfig
	case BlobStorageObjectTypeLog:
		azureBlobBaseConfig.BlobContainerName = b.BlobContainerCiLog
		return azureBlobBaseConfig
	case BlobStorageObjectTypeArtifact:
		azureBlobBaseConfig.BlobContainerName = b.BlobContainerArtifact
		return azureBlobBaseConfig
	default:
		return nil
	}
}

func GetBlobStorageBaseGcpConfig(b *blob_storage.GcpBlobConfig, blobStorageObjectType string) *blob_storage.GcpBlobBaseConfig {
	gcpBlobBaseConfig := &blob_storage.GcpBlobBaseConfig{
		CredentialFileJsonData: b.CredentialFileJsonData,
	}
	switch blobStorageObjectType {
	case BlobStorageObjectTypeCache:
		gcpBlobBaseConfig.BucketName = b.CacheBucketName
		return gcpBlobBaseConfig
	case BlobStorageObjectTypeLog:
		gcpBlobBaseConfig.BucketName = b.LogBucketName
		return gcpBlobBaseConfig
	case BlobStorageObjectTypeArtifact:
		gcpBlobBaseConfig.BucketName = b.ArtifactBucketName
		return gcpBlobBaseConfig
	default:
		return nil
	}
}
