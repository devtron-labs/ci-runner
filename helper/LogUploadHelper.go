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
	"fmt"
	"github.com/devtron-labs/ci-runner/util"
	"log"
	"path"
)

// UploadLogs
// Checks of blob storage is configured, if yes, uploads the locally created log file to configured storage
func UploadLogs(cloudHelperBaseConfig *util.CloudHelperBaseConfig) {

	if !cloudHelperBaseConfig.StorageModuleConfigured {
		log.Println(util.DEVTRON, "not going to upload logs as storage module not configured...")
		return
	}

	err := UploadFileToCloud(cloudHelperBaseConfig, util.TmpLogLocation, path.Join(cloudHelperBaseConfig.BlobStorageLogKey, util.TmpLogLocation))
	if err != nil {
		fmt.Println("Failed to upload to blob storage with error", err)
		return
	}
}
