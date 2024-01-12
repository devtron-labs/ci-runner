package CiCdStageExecutor

import (
	"encoding/json"
	"github.com/devtron-labs/ci-runner/helper"
	test_data "github.com/devtron-labs/ci-runner/test-data"
	"github.com/devtron-labs/ci-runner/util"
	"os"
	"testing"
)

func TestHandleCDEvent(t *testing.T) {
	t.Run("StageYamlNoWithNoError", func(t *testing.T) {

		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CdTriggerEventPayloadWithTaskYaml), ciCdRequest)

		exitCode := 0

		// Call the function
		HandleCDEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != 0 {
			t.Errorf("Expected exitCode to be %d, but got %d", 0, exitCode)
		}
	})

	t.Run("StageYamlWithError", func(t *testing.T) {
		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CdTriggerEventPayloadWithTaskYamlBad), ciCdRequest)

		exitCode := 0

		os.RemoveAll(util.WORKINGDIR)
		// Call the function with an error
		HandleCDEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != util.DefaultErrorCode {
			t.Errorf("Expected exitCode to be %d, but got %d", util.DefaultErrorCode, exitCode)
		}
	})

	t.Run("StageYamlWithNoArtifact", func(t *testing.T) {
		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CdTriggerEventPayloadWithTaskYamlWrongOutputPath), ciCdRequest)

		exitCode := 0

		os.RemoveAll(util.WORKINGDIR)
		// Call the function with an error
		HandleCDEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != util.DefaultErrorCode {
			t.Errorf("Expected exitCode to be %d, but got %d", util.DefaultErrorCode, exitCode)
		}
	})

	t.Run("StepsStageWithNoError", func(t *testing.T) {

		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CdTriggerEventPayloadWithSteps1), ciCdRequest)

		exitCode := 0

		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		HandleCDEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != 0 {
			t.Errorf("Expected exitCode to be %d, but got %d", 0, exitCode)
		}
	})

	t.Run("StepsStageVarOutputCheckFail", func(t *testing.T) {

		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CdTriggerEventPayloadWithStepsVarCheckBad), ciCdRequest)

		exitCode := 0

		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		HandleCDEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != util.DefaultErrorCode {
			t.Errorf("Expected exitCode to be %d, but got %d", 0, exitCode)
		}
	})

	t.Run("StepsStageOutputWithError", func(t *testing.T) {

		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CdTriggerEventPayloadWithSteps2), ciCdRequest)

		exitCode := 0

		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		HandleCDEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != util.DefaultErrorCode {
			t.Errorf("Expected exitCode to be %d, but got %d", util.DefaultErrorCode, exitCode)
		}
	})

	t.Run("StepsStageWithError", func(t *testing.T) {

		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CdTriggerEventPayloadWithStepsBad), ciCdRequest)

		exitCode := 0

		os.RemoveAll(util.WORKINGDIR)
		os.RemoveAll("/output")
		// Call the function
		HandleCDEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != util.DefaultErrorCode {
			t.Errorf("Expected exitCode to be %d, but got %d", util.DefaultErrorCode, exitCode)
		}
	})

	t.Run("StepsStageWithSuccessTriggerCriteria", func(t *testing.T) {

		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CdTriggerEventPayloadWithSteps3), ciCdRequest)

		exitCode := 0

		os.RemoveAll(util.WORKINGDIR)
		os.RemoveAll("/output")
		// Call the function
		HandleCDEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != util.DefaultErrorCode {
			t.Errorf("Expected exitCode to be %d, but got %d", util.DefaultErrorCode, exitCode)
		}
	})

	t.Run("StepsStagePlugin", func(t *testing.T) {

		// Prepare test data
		ciCdRequest := &helper.CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CdTriggerEventPayloadWithStepsWithPlugin), ciCdRequest)

		exitCode := 0

		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		HandleCDEvent(ciCdRequest, &exitCode)

		// Assert the expected results
		if exitCode != 0 {
			t.Errorf("Expected exitCode to be %d, but got %d", 0, exitCode)
		}
	})
}
