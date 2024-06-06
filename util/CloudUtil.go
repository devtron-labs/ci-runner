/*
 * Copyright (c) 2024. Devtron Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"github.com/caarlos0/env"
	blob_storage "github.com/devtron-labs/common-lib/blob-storage"
)

const (
	BlobStorageObjectTypeCache    = "cache"
	BlobStorageObjectTypeArtifact = "artifact"
	BlobStorageObjectTypeLog      = "log"
)

// BlobStorageConfig is the blob storage config for external cluster added via cm/secret code
// will be expecting these env variables acc to cloud provider if UseExternalClusterBlob is true.
type BlobStorageConfig struct {
	//AWS credentials
	CloudProvider      blob_storage.BlobStorageType `env:"BLOB_STORAGE_PROVIDER"`
	S3AccessKey        string                       `env:"BLOB_STORAGE_S3_ACCESS_KEY"`
	S3SecretKey        string                       `env:"BLOB_STORAGE_S3_SECRET_KEY"`
	S3Endpoint         string                       `env:"BLOB_STORAGE_S3_ENDPOINT"`
	S3EndpointInsecure bool                         `env:"BLOB_STORAGE_S3_ENDPOINT_INSECURE" envDefault:"false"`
	S3BucketVersioned  bool                         `env:"BLOB_STORAGE_S3_BUCKET_VERSIONED" envDefault:"true"`
	//artifact and logs bucket name in s3 will get their values from DEFAULT_BUILD_LOGS_BUCKET
	CdDefaultBuildLogsBucket string `env:"DEFAULT_BUILD_LOGS_BUCKET" `
	//logs and artifact region in s3 will get their values from DEFAULT_CD_LOGS_BUCKET_REGION
	CdDefaultCdLogsBucketRegion string `env:"DEFAULT_CD_LOGS_BUCKET_REGION" `
	//cache bucket name in s3 will get its value from DEFAULT_CACHE_BUCKET
	DefaultCacheBucket string `env:"DEFAULT_CACHE_BUCKET"`
	//cache region in s3 will get its value from DEFAULT_CACHE_BUCKET_REGION
	DefaultCacheBucketRegion string `env:"DEFAULT_CACHE_BUCKET_REGION"`

	//GCP credentials
	GcpBlobStorageCredentialJson string `env:"BLOB_STORAGE_GCP_CREDENTIALS_JSON"`
	//ArtifactBucketName and LogBucketName for gcp will get their values from DEFAULT_BUILD_LOGS_BUCKET
	//CacheBucketName for gcp will get its value from DEFAULT_CACHE_BUCKET

	//Azure credentials
	AzureAccountName               string `env:"AZURE_ACCOUNT_NAME"`
	AzureGatewayUrl                string `env:"AZURE_GATEWAY_URL"`
	AzureGatewayConnectionInsecure bool   `env:"AZURE_GATEWAY_CONNECTION_INSECURE" envDefault:"true"`
	AzureAccountKey                string `env:"AZURE_ACCOUNT_KEY"`
	//log and artifact container name in azure will get their values from AZURE_BLOB_CONTAINER_CI_LOG
	AzureBlobContainerCiLog string `env:"AZURE_BLOB_CONTAINER_CI_LOG"`
	//cache container name in azure will get their values from AZURE_BLOB_CONTAINER_CI_CACHE
	AzureBlobContainerCiCache string `env:"AZURE_BLOB_CONTAINER_CI_CACHE"`
}

func GetBlobStorageConfig() (*BlobStorageConfig, error) {
	cfg := &BlobStorageConfig{}
	err := env.Parse(cfg)
	return cfg, err
}

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

func (c *CloudHelperBaseConfig) SetAwsBlobStorageS3Config(blobStorageConfig *BlobStorageConfig) {
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

func (c *CloudHelperBaseConfig) SetAzureBlobStorageConfig(blobStorageConfig *BlobStorageConfig) {
	c.AzureBlobConfig = &blob_storage.AzureBlobConfig{
		Enabled:               blobStorageConfig.CloudProvider == blob_storage.BLOB_STORAGE_AZURE,
		AccountName:           blobStorageConfig.AzureAccountName,
		BlobContainerCiLog:    blobStorageConfig.AzureBlobContainerCiLog,
		BlobContainerCiCache:  blobStorageConfig.AzureBlobContainerCiCache,
		BlobContainerArtifact: blobStorageConfig.AzureBlobContainerCiLog,
		AccountKey:            blobStorageConfig.AzureAccountKey,
	}
}

func (c *CloudHelperBaseConfig) SetGcpBlobStorageConfig(blobStorageConfig *BlobStorageConfig) {
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
