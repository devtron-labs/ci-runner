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
	"github.com/devtron-labs/ci-runner/util"
	blob_storage "github.com/devtron-labs/common-lib/blob-storage"
	"log"
	"os"
	"os/exec"
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
	cloudHelperBaseConfig := ciRequest.GetCloudHelperBaseConfig()
	request := createBlobStorageRequestForCache(cloudHelperBaseConfig, ciRequest.CiCacheFileName, ciRequest.CiCacheFileName)
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
	cloudHelperBaseConfig := ciRequest.GetCloudHelperBaseConfig()
	blobStorageService := blob_storage.NewBlobStorageServiceImpl(nil)
	request := createBlobStorageRequestForCache(cloudHelperBaseConfig, ciRequest.CiCacheFileName, ciRequest.CiCacheFileName)
	err = blobStorageService.PutWithCommand(request)
	if err != nil {
		log.Println(util.DEVTRON, " -----> push err", err)
	}
	return err
}

func createBlobStorageRequestForCache(cloudHelperBaseConfig *util.CloudHelperBaseConfig, sourceKey string, destinationKey string) *blob_storage.BlobStorageRequest {
	if cloudHelperBaseConfig.UseExternalClusterBlob {
		UpdateCloudHelperBaseConfigForExtCluster(cloudHelperBaseConfig)
	}
	var awsS3BaseConfig *blob_storage.AwsS3BaseConfig
	if cloudHelperBaseConfig.BlobStorageS3Config != nil {
		awsS3BaseConfig = cloudHelperBaseConfig.BlobStorageS3Config.GetBlobStorageBaseS3Config(BlobStorageObjectTypeCache)
	}

	var azureBlobBaseConfig *blob_storage.AzureBlobBaseConfig
	if cloudHelperBaseConfig.AzureBlobConfig != nil {
		azureBlobBaseConfig = cloudHelperBaseConfig.AzureBlobConfig.GetBlobStorageBaseAzureConfig(BlobStorageObjectTypeCache)
	}

	var gcpBlobBaseConfig *blob_storage.GcpBlobBaseConfig
	if cloudHelperBaseConfig.GcpBlobConfig != nil {
		gcpBlobBaseConfig = cloudHelperBaseConfig.GcpBlobConfig.GetBlobStorageBaseGcpConfig(BlobStorageObjectTypeCache)
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
