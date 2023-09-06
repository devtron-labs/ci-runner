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

package util

import "path/filepath"

const DEVTRON = "DEVTRON"

const INSECURE = "insecure"
const SECUREWITHCERT = "secure-with-cert"
const RETRYCOUNT = 10
const WORKINGDIR = "/devtroncd"
const LOCAL_BUILDX_LOCATION = "/var/lib/devtron/buildx"
const LOCAL_BUILDX_CACHE_LOCATION = LOCAL_BUILDX_LOCATION + "/cache"

const CIEVENT = "CI"
const CDSTAGE = "CD"
const WEBHOOK = "WEBHOOK"
const DRY_RUN = "DryRun"
const ENV_VARIABLE_BUILD_SUCCESS = "BUILD_SUCCESS"

var TmpArtifactLocation = "./job-artifact"
var TmpLogLocation = "/main.log"

const CiCdEventEnvKey = "CI_CD_EVENT"

const Source_Signal = "Source_Signal"
const Source_Defer = "Source_Defer"

const DefaultErrorCode = 1
const AbortErrorCode = 143
const CiStageFailErrorCode = 2
const InAppLogging = "IN_APP_LOGGING"
const CiRunnerCommand = "./cirunner"
const TeeCommand = "tee"
const LogFileName = "main.log"

const NewLineChar = "\n"

var (
	Output_path = filepath.Join(WORKINGDIR, "./process")

	Bash_script = filepath.Join("_script.sh")
)
