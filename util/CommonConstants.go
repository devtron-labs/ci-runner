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
const WORKINGDIR = "/tmp"

const CIEVENT = "CI"
const CDSTAGE = "CD"

var TmpArtifactLocation = "./job-artifact"

var (
	Output_path = filepath.Join(WORKINGDIR, "./process")

	Bash_script = filepath.Join("_script.sh")
)
