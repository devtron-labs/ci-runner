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

package helper

import (
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/devtron-labs/ci-runner/util"
	"github.com/otiai10/copy"
)

//const BLOB_STORAGE_AZURE = "AZURE"
//const BLOB_STORAGE_S3 = "S3"
//const BLOB_STORAGE_GCP = "GCP"

func UploadArtifact(cloudHelperBaseConfig *util.CloudHelperBaseConfig, artifactFiles map[string]string, artifactFileLocation string) error {
	if len(artifactFiles) == 0 {
		log.Println(util.DEVTRON, "no artifact to upload")
		return nil
	}
	//collect in a dir
	log.Println(util.DEVTRON, "artifact upload ", artifactFiles, artifactFileLocation)
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
	err = ZipAndUpload(cloudHelperBaseConfig, artifactFileLocation)
	return err
}

func ZipAndUpload(cloudHelperBaseConfig *util.CloudHelperBaseConfig, artifactFileName string) error {
	uploadArtifact := func() error {
		if !cloudHelperBaseConfig.StorageModuleConfigured {
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
		log.Println(util.DEVTRON, " artifact upload to ", zipFile, artifactFileName)
		err = UploadFileToCloud(cloudHelperBaseConfig, zipFile, artifactFileName)
		return err
	}
	return util.ExecuteWithStageInfoLog(util.UPLOAD_ARTIFACT, uploadArtifact)
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
