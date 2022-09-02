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
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/devtron-labs/ci-runner/util"
	"github.com/otiai10/copy"
)

const BLOB_STORAGE_AZURE = "AZURE"
const BLOB_STORAGE_S3 = "S3"
const BLOB_STORAGE_GCP = "GCP"
const BLOB_STORAGE_MINIO = "MINIO"

type AzureBlobConfig struct {
	Enabled              bool   `json:"enabled"`
	AccountName          string `json:"accountName"`
	BlobContainerCiLog   string `json:"blobContainerCiLog"`
	BlobContainerCiCache string `json:"blobContainerCiCache"`
	AccountKey           string `json:"accountKey"`
}

func UploadArtifact(storageModuleConfigured bool, artifactFiles map[string]string, s3Location string, cloudProvider string, minioEndpoint string, azureBlobConfig *AzureBlobConfig) error {
	if len(artifactFiles) == 0 {
		log.Println(util.DEVTRON, "no artifact to upload")
		return nil
	}
	//collect in a dir
	log.Println(util.DEVTRON, "artifact upload ", artifactFiles, s3Location)
	err := os.Mkdir(util.TmpArtifactLocation, os.ModePerm)
	if err != nil {
		return err
	}
	for key, val := range artifactFiles {
		loc := filepath.Join(util.TmpArtifactLocation, key)
		err := os.Mkdir(loc, os.ModePerm)
		if err != nil {
			return err
		}
		err = copy.Copy(val, filepath.Join(loc, val))
		if err != nil {
			return err
		}
	}
	err = ZipAndUpload(storageModuleConfigured, s3Location, cloudProvider, minioEndpoint, azureBlobConfig)
	return err
}

func ZipAndUpload(storageModuleConfigured bool, artifactLocation string, cloudProvider string, minioEndpoint string, azureBlobConfig *AzureBlobConfig) error {
	if !storageModuleConfigured {
		log.Println(util.DEVTRON, "not going to upload artifact as storage module not configured...")
		return nil
	}
	isEmpty, err := IsDirEmpty(util.TmpArtifactLocation)
	if err != nil {
		log.Println(util.DEVTRON, "artifact empty check error ")
		return err
	} else if isEmpty {
		log.Println(util.DEVTRON, "no artifact to upload")
		return nil
	}
	log.Println(util.DEVTRON, "artifact to upload")
	zipFile := "job-artifact.zip"
	zipCmd := exec.Command("zip", "-r", zipFile, util.TmpArtifactLocation)
	err = util.RunCommand(zipCmd)
	if err != nil {
		return err
	}
	log.Println(util.DEVTRON, " artifact upload to ", zipFile, artifactLocation)
	switch cloudProvider {
	case BLOB_STORAGE_S3:
		artifactPush := exec.Command("aws", "s3", "cp", zipFile, artifactLocation)
		err = util.RunCommand(artifactPush)
		return err
	case BLOB_STORAGE_MINIO:
		artifactPush := exec.Command("aws", "--endpoint-url", minioEndpoint, "s3", "cp", zipFile, artifactLocation)
		err = util.RunCommand(artifactPush)
		return err
	case BLOB_STORAGE_AZURE:
		b := AzureBlob{}
		err = b.UploadBlob(context.Background(), artifactLocation, azureBlobConfig, zipFile, azureBlobConfig.BlobContainerCiLog)
		return err
	default:
		return fmt.Errorf("cloudprovider %s not supported", cloudProvider)
	}
}

func IsDirEmpty(name string) (bool, error) {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return true, nil
	}
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	// read in ONLY one file
	_, err = f.Readdir(1)

	// and if the file is EOF... well, the dir is empty.
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
