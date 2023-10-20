package util

import blob_storage "github.com/devtron-labs/common-lib/blob-storage"

const (
	BlobStorageAzure = "AZURE"
	BlobStorageS3    = "S3"
	BlobStorageGcp   = "GCP"
)

type CloudHelperBaseConfig struct {
	StorageModuleConfigured bool
	BlobStorageLogKey       string
	CloudProvider           blob_storage.BlobStorageType
	UseExternalClusterBlob  bool
	BlobStorageS3Config     *blob_storage.BlobStorageS3Config
	AzureBlobConfig         *blob_storage.AzureBlobConfig
	GcpBlobConfig           *blob_storage.GcpBlobConfig
}
