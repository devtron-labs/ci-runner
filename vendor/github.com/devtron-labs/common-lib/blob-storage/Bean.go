package blob_storage

type BlobStorageRequest struct {
	StorageType         BlobStorageType
	SourceKey           string
	DestinationKey      string
	AwsS3BaseConfig     *AwsS3BaseConfig
	AzureBlobBaseConfig *AzureBlobBaseConfig
	GcpBlobBaseConfig   *GcpBlobBaseConfig
}

type BlobStorageS3Config struct {
	AccessKey                  string `json:"accessKey"`
	Passkey                    string `json:"passkey"`
	EndpointUrl                string `json:"endpointUrl"`
	IsInSecure                 bool   `json:"isInSecure"`
	CiLogBucketName            string `json:"ciLogBucketName"`
	CiLogRegion                string `json:"ciLogRegion"`
	CiLogBucketVersioning      bool   `json:"ciLogBucketVersioning"`
	CiCacheBucketName          string `json:"ciCacheBucketName"`
	CiCacheRegion              string `json:"ciCacheRegion"`
	CiCacheBucketVersioning    bool   `json:"ciCacheBucketVersioning"`
	CiArtifactBucketName       string `json:"ciArtifactBucketName"`
	CiArtifactRegion           string `json:"ciArtifactRegion"`
	CiArtifactBucketVersioning bool   `json:"ciArtifactBucketVersioning"`
}

func (b *BlobStorageS3Config) GetBlobStorageBaseS3Config(blobStorageObjectType string) *AwsS3BaseConfig {
	switch blobStorageObjectType {
	case BlobStorageObjectTypeCache:
		return &AwsS3BaseConfig{
			AccessKey:         b.AccessKey,
			Passkey:           b.Passkey,
			EndpointUrl:       b.EndpointUrl,
			IsInSecure:        b.IsInSecure,
			BucketName:        b.CiCacheBucketName,
			Region:            b.CiCacheRegion,
			VersioningEnabled: b.CiCacheBucketVersioning,
		}
	case BlobStorageObjectTypeLog:
		return &AwsS3BaseConfig{
			AccessKey:         b.AccessKey,
			Passkey:           b.Passkey,
			EndpointUrl:       b.EndpointUrl,
			IsInSecure:        b.IsInSecure,
			BucketName:        b.CiLogBucketName,
			Region:            b.CiLogRegion,
			VersioningEnabled: b.CiLogBucketVersioning,
		}
	case BlobStorageObjectTypeArtifact:
		return &AwsS3BaseConfig{
			AccessKey:         b.AccessKey,
			Passkey:           b.Passkey,
			EndpointUrl:       b.EndpointUrl,
			IsInSecure:        b.IsInSecure,
			BucketName:        b.CiArtifactBucketName,
			Region:            b.CiArtifactRegion,
			VersioningEnabled: b.CiArtifactBucketVersioning,
		}
	default:
		return nil
	}
}

type AwsS3BaseConfig struct {
	AccessKey         string `json:"accessKey"`
	Passkey           string `json:"passkey"`
	EndpointUrl       string `json:"endpointUrl"`
	IsInSecure        bool   `json:"isInSecure"`
	BucketName        string `json:"bucketName"`
	Region            string `json:"region"`
	VersioningEnabled bool   `json:"versioningEnabled"`
}

type AzureBlobConfig struct {
	Enabled               bool   `json:"enabled"`
	AccountName           string `json:"accountName"`
	BlobContainerCiLog    string `json:"blobContainerCiLog"`
	BlobContainerCiCache  string `json:"blobContainerCiCache"`
	BlobContainerArtifact string `json:"blobStorageArtifact"`
	AccountKey            string `json:"accountKey"`
}

func (b *AzureBlobConfig) GetBlobStorageBaseAzureConfig(blobStorageObjectType string) *AzureBlobBaseConfig {
	switch blobStorageObjectType {
	case BlobStorageObjectTypeCache:
		return &AzureBlobBaseConfig{
			Enabled:           b.Enabled,
			AccountName:       b.AccountName,
			AccountKey:        b.AccountKey,
			BlobContainerName: b.BlobContainerCiCache,
		}
	case BlobStorageObjectTypeLog:
		return &AzureBlobBaseConfig{
			Enabled:           b.Enabled,
			AccountName:       b.AccountName,
			AccountKey:        b.AccountKey,
			BlobContainerName: b.BlobContainerCiLog,
		}
	case BlobStorageObjectTypeArtifact:
		return &AzureBlobBaseConfig{
			Enabled:           b.Enabled,
			AccountName:       b.AccountName,
			AccountKey:        b.AccountKey,
			BlobContainerName: b.BlobContainerArtifact,
		}
	default:
		return nil
	}
}

type AzureBlobBaseConfig struct {
	Enabled           bool   `json:"enabled"`
	AccountName       string `json:"accountName"`
	AccountKey        string `json:"accountKey"`
	BlobContainerName string `json:"blobContainerName"`
}

type GcpBlobConfig struct {
	CredentialFileJsonData string `json:"credentialFileData"`
	CacheBucketName        string `json:"ciCacheBucketName"`
	LogBucketName          string `json:"logBucketName"`
	ArtifactBucketName     string `json:"artifactBucketName"`
}

func (b *GcpBlobConfig) GetBlobStorageBaseS3Config(blobStorageObjectType string) *GcpBlobBaseConfig {
	switch blobStorageObjectType {
	case BlobStorageObjectTypeCache:
		return &GcpBlobBaseConfig{
			BucketName:             b.CacheBucketName,
			CredentialFileJsonData: b.CredentialFileJsonData,
		}
	case BlobStorageObjectTypeLog:
		return &GcpBlobBaseConfig{
			BucketName:             b.LogBucketName,
			CredentialFileJsonData: b.CredentialFileJsonData,
		}
	case BlobStorageObjectTypeArtifact:
		return &GcpBlobBaseConfig{
			BucketName:             b.ArtifactBucketName,
			CredentialFileJsonData: b.CredentialFileJsonData,
		}
	default:
		return nil
	}
}

type GcpBlobBaseConfig struct {
	BucketName             string `json:"bucketName"`
	CredentialFileJsonData string `json:"credentialFileData"`
}

type BlobStorageType string

const (
	BLOB_STORAGE_AZURE            BlobStorageType = "AZURE"
	BLOB_STORAGE_S3                               = "S3"
	BLOB_STORAGE_GCP                              = "GCP"
	BLOB_STORAGE_MINIO                            = "MINIO"
	BlobStorageObjectTypeCache                    = "cache"
	BlobStorageObjectTypeArtifact                 = "artifact"
	BlobStorageObjectTypeLog                      = "log"
)
