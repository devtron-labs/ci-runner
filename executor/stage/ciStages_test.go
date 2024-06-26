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

package stage

import (
	"encoding/json"
	"github.com/devtron-labs/ci-runner/helper"
	test_data "github.com/devtron-labs/ci-runner/test-data"
	"github.com/devtron-labs/ci-runner/util"
	"os"
	"testing"
)

func TestHandleCIEvent(t *testing.T) {

	t.Run("CiTriggerEventWithoutPrePostStep", func(t *testing.T) {

		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventPayloadWithoutPrePostStep), ciCdRequest)

		exitCode := 0
		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		ciStage := NewCiStage(*helper.NewGitManagerImpl(helper.NewGitCliManager()))
		ciStage.HandleCIEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != util.CiStageFailErrorCode {
			t.Errorf("Expected exitCode to be %d, but got %d", 0, exitCode)
		}
	})
	t.Run("CiTriggerEventWithPreStep", func(t *testing.T) {

		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventPayloadWithPreStep), ciCdRequest)

		exitCode := 0
		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		ciStage := NewCiStage(*helper.NewGitManagerImpl(helper.NewGitCliManager()))
		ciStage.HandleCIEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != util.CiStageFailErrorCode {
			t.Errorf("Expected exitCode to be %d, but got %d", 0, exitCode)
		}
	})
	t.Run("CiTriggerEventWithPrePostStep", func(t *testing.T) {

		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventPayloadWithPrePostStep), ciCdRequest)

		exitCode := 0
		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		ciStage := NewCiStage(*helper.NewGitManagerImpl(helper.NewGitCliManager()))
		ciStage.HandleCIEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != util.CiStageFailErrorCode {
			t.Errorf("Expected exitCode to be %d, but got %d", 0, exitCode)
		}
	})
	t.Run("CiTriggerEventWithValidGitHash", func(t *testing.T) {

		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventWithValidGitHash), ciCdRequest)

		exitCode := 0
		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		ciStage := NewCiStage(*helper.NewGitManagerImpl(helper.NewGitCliManager()))
		ciStage.HandleCIEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != util.CiStageFailErrorCode {
			t.Errorf("Expected exitCode to be %d, but got %d", 0, exitCode)
		}
	})
	t.Run("CiTriggerEventWithInValidGitHash", func(t *testing.T) {

		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventWithInValidGitHash), ciCdRequest)

		exitCode := 0
		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		ciStage := NewCiStage(*helper.NewGitManagerImpl(helper.NewGitCliManager()))
		ciStage.HandleCIEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != util.CiStageFailErrorCode {
			t.Errorf("Expected exitCode to be %d, but got %d", 0, exitCode)
		}
	})
	t.Run("CiTriggerEventWithEmptyGitHash", func(t *testing.T) {

		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventWithEmptyGitHash), ciCdRequest)

		exitCode := 0
		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		ciStage := NewCiStage(*helper.NewGitManagerImpl(helper.NewGitCliManager()))
		ciStage.HandleCIEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != util.CiStageFailErrorCode {
			t.Errorf("Expected exitCode to be %d, but got %d", 0, exitCode)
		}
	})
	t.Run("CiTriggerEventWithEmptyGitHashAndSourceValue", func(t *testing.T) {

		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventWithEmptyGitHashAndSourceValue), ciCdRequest)

		exitCode := 0
		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		ciStage := NewCiStage(*helper.NewGitManagerImpl(helper.NewGitCliManager()))
		ciStage.HandleCIEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != util.CiStageFailErrorCode {
			t.Errorf("Expected exitCode to be %d, but got %d", 0, exitCode)
		}
	})
	t.Run("CiTriggerEventWithValidGitTag", func(t *testing.T) {

		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventWithValidGitTag), ciCdRequest)

		exitCode := 0
		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		ciStage := NewCiStage(*helper.NewGitManagerImpl(helper.NewGitCliManager()))
		ciStage.HandleCIEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != util.CiStageFailErrorCode {
			t.Errorf("Expected exitCode to be %d, but got %d", 0, exitCode)
		}
	})
	t.Run("CiTriggerEventWithInValidGitTag", func(t *testing.T) {

		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventWithInValidGitTag), ciCdRequest)

		exitCode := 0
		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		ciStage := NewCiStage(*helper.NewGitManagerImpl(helper.NewGitCliManager()))
		ciStage.HandleCIEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != util.CiStageFailErrorCode {
			t.Errorf("Expected exitCode to be %d, but got %d", 0, exitCode)
		}
	})
	t.Run("CiTriggerEventSourceTypeWebhookPRBased", func(t *testing.T) {

		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventSourceTypeWebhookPRBased), ciCdRequest)

		exitCode := 0
		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		ciStage := NewCiStage(*helper.NewGitManagerImpl(helper.NewGitCliManager()))
		ciStage.HandleCIEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != util.CiStageFailErrorCode {
			t.Errorf("Expected exitCode to be %d, but got %d", 0, exitCode)
		}
	})

}
