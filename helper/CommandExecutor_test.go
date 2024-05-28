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
	"os/exec"
	"testing"
)

func TestCommandExecutor(t *testing.T) {
	t.Run("execute cmd", func(t *testing.T) {
		//cmd := "/bin/sh -c 'ls -l /Users/kripanshbanga/Desktop/devtron-workspace/dashboard/package.json'"
		output, err := exec.Command("/bin/sh", "-c", "jq \".engines.node\" /Users/kripanshbanga/Desktop/devtron-workspace/dashboard/package.json").Output()
		fmt.Println(err)
		fmt.Println(string(output))
	})
}
