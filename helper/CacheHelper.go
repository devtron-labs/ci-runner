/*
 *  Copyright 2020 Devtron Labs
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
 *
 */

package helper

import (
	"log"
	"os"
	"os/exec"

	"github.com/devtron-labs/ci-runner/util"
	blob_storage "github.com/devtron-labs/common-lib/blob-storage"
)

func GetCache(ciRequest *CommonWorkflowRequest) error {
	if !ciRequest.BlobStorageConfigured {
		log.Println("ignoring cache as storage module not configured ... ") //TODO not needed
		return nil
	}
	if ciRequest.IgnoreDockerCachePull || ciRequest.CacheInvalidate {
		if !ciRequest.IsPvcMounted {
			log.Println("ignoring cache ... ")
		}
		return nil
	}
	log.Println("setting build cache ...............")

	//----------download file
	blobStorageService := blob_storage.NewBlobStorageServiceImpl(nil)
	request := createBlobStorageRequestForCache(ciRequest.CloudProvider, ciRequest.CiCacheFileName, ciRequest.CiCacheFileName, ciRequest.BlobStorageS3Config, ciRequest.AzureBlobConfig, ciRequest.GcpBlobConfig)
	downloadSuccess, bytesSize, err := blobStorageService.Get(request)
	if bytesSize >= ciRequest.CacheLimit {
		log.Println(util.DEVTRON, " cache upper limit exceeded, ignoring old cache")
		downloadSuccess = false
	}

	// Extract cache
	if err == nil && downloadSuccess {
		extractCmd := exec.Command("tar", "-xvzf", ciRequest.CiCacheFileName)
		extractCmd.Dir = "/"
		err = extractCmd.Run()
		if err != nil {
			log.Fatal(" Could not extract cache blob ", err)
		}
	} else if err != nil {
		log.Println(util.DEVTRON, "build cache error", err.Error())
	}
	return nil
}

func SyncCache(ciRequest *CommonWorkflowRequest) error {
	if !ciRequest.BlobStorageConfigured {
		log.Println("ignoring cache as storage module not configured... ")
		return nil
	}
	if ciRequest.IgnoreDockerCachePush {
		if ciRequest.IsPvcMounted {
			return nil
		}
		log.Println("ignoring cache as cache push is disabled... ")
		return nil
	}
	err := os.Chdir("/")
	if err != nil {
		log.Println(err)
		return err
	}
	util.DeleteFile(ciRequest.CiCacheFileName)
	// Generate new cache
	log.Println("Generating new cache")
	var cachePath string
	ciBuildConfig := ciRequest.CiBuildConfig
	if (ciBuildConfig.CiBuildType == SELF_DOCKERFILE_BUILD_TYPE || ciBuildConfig.CiBuildType == MANAGED_DOCKERFILE_BUILD_TYPE) &&
		ciBuildConfig.DockerBuildConfig.CheckForBuildX() {
		cachePath = util.LOCAL_BUILDX_CACHE_LOCATION
	} else {
		cachePath = "/var/lib/docker"
	}

	tarCmd := exec.Command("tar", "-cvzf", ciRequest.CiCacheFileName, cachePath)
	tarCmd.Dir = "/"
	err = tarCmd.Run()
	if err != nil {
		log.Fatal("Could not compress cache", err)
	}

	//aws s3 cp cache.tar.gz s3://ci-caching/
	//----------upload file

	log.Println(util.DEVTRON, " -----> pushing new cache")
	blobStorageService := blob_storage.NewBlobStorageServiceImpl(nil)
	request := createBlobStorageRequestForCache(ciRequest.CloudProvider, ciRequest.CiCacheFileName, ciRequest.CiCacheFileName, ciRequest.BlobStorageS3Config, ciRequest.AzureBlobConfig, ciRequest.GcpBlobConfig)
	err = blobStorageService.PutWithCommand(request)
	if err != nil {
		log.Println(util.DEVTRON, " -----> push err", err)
	}
	return err
}

func createBlobStorageRequestForCache(cloudProvider blob_storage.BlobStorageType, sourceKey string, destinationKey string, blobStorageS3Config *blob_storage.BlobStorageS3Config, azureBlobConfig *blob_storage.AzureBlobConfig, gcpBlobConfig *blob_storage.GcpBlobConfig) *blob_storage.BlobStorageRequest {
	var awsS3BaseConfig *blob_storage.AwsS3BaseConfig
	if blobStorageS3Config != nil {
		awsS3BaseConfig = &blob_storage.AwsS3BaseConfig{
			AccessKey:         blobStorageS3Config.AccessKey,
			Passkey:           blobStorageS3Config.Passkey,
			EndpointUrl:       blobStorageS3Config.EndpointUrl,
			IsInSecure:        blobStorageS3Config.IsInSecure,
			BucketName:        blobStorageS3Config.CiCacheBucketName,
			Region:            blobStorageS3Config.CiCacheRegion,
			VersioningEnabled: blobStorageS3Config.CiCacheBucketVersioning,
		}
	}

	var azureBlobBaseConfig *blob_storage.AzureBlobBaseConfig
	if azureBlobConfig != nil {
		azureBlobBaseConfig = &blob_storage.AzureBlobBaseConfig{
			AccountKey:        azureBlobConfig.AccountKey,
			AccountName:       azureBlobConfig.AccountName,
			Enabled:           azureBlobConfig.Enabled,
			BlobContainerName: azureBlobConfig.BlobContainerCiCache,
		}
	}

	var gcpBlobBaseConfig *blob_storage.GcpBlobBaseConfig
	if gcpBlobConfig != nil {
		gcpBlobBaseConfig = &blob_storage.GcpBlobBaseConfig{
			CredentialFileJsonData: gcpBlobConfig.CredentialFileJsonData,
			BucketName:             gcpBlobConfig.CacheBucketName,
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
