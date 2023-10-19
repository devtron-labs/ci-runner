package util

import blob_storage "github.com/devtron-labs/common-lib/blob-storage"

const (
	BlobStorageAzure                      = "AZURE"
	BlobStorageS3                         = "S3"
	BlobStorageGcp                        = "GCP"
	CloudProvider                  string = "BLOB_STORAGE_PROVIDER"
	BlobStorageS3AccessKey         string = "BLOB_STORAGE_S3_ACCESS_KEY"
	BlobStorageS3SecretKey         string = "BLOB_STORAGE_S3_SECRET_KEY"
	BlobStorageS3Endpoint          string = "BLOB_STORAGE_S3_ENDPOINT"
	BlobStorageS3EndpointInsecure  string = "BLOB_STORAGE_S3_ENDPOINT_INSECURE"
	BlobStorageS3BucketVersioned   string = "BLOB_STORAGE_S3_BUCKET_VERSIONED"
	BlobStorageGcpCredentialJson   string = "BLOB_STORAGE_GCP_CREDENTIALS_JSON"
	AzureAccountName               string = "AZURE_ACCOUNT_NAME"
	AzureGatewayUrl                string = "AZURE_GATEWAY_URL"
	AzureGatewayConnectionInsecure string = "AZURE_GATEWAY_CONNECTION_INSECURE"
	AzureBlobContainerCiLog        string = "AZURE_BLOB_CONTAINER_CI_LOG"
	AzureBlobContainerCiCache      string = "AZURE_BLOB_CONTAINER_CI_CACHE"
	AzureAccountKey                string = "AZURE_ACCOUNT_KEY"
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
