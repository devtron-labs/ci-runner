package helper

import (
	"encoding/json"
	test_data "github.com/devtron-labs/ci-runner/test-data"
	"github.com/devtron-labs/ci-runner/util"
	"os"
	"strings"
	"testing"
)

// before running test cases locally convert WORKINGDIR to "/tmp/devtroncd" from "/devtroncd"
func TestGitHelper(t *testing.T) {
	t.Run("Test1_ValidCiProjectDetailsAnonymous", func(t *testing.T) {

		// Prepare test data, ANONYMOUS and SOURCE_TYPE_BRANCH_FIXED data
		ciCdRequest := &CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventPayloadWithoutPrePostStep), ciCdRequest)
		ciProjectDetails := ciCdRequest.CiRequest.CiProjectDetails

		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		err := CloneAndCheckout(ciProjectDetails)

		// Assert the expected results
		if err != nil {
			t.Errorf("Error in Test1_ValidCiProjectDetailsAnonymous")
		}
	})
	t.Run("Test2_ValidCiProjectDetailsUsernamePassword", func(t *testing.T) {

		// Prepare test data, USERNAME_PASSWORD and SOURCE_TYPE_BRANCH_FIXED data
		ciCdRequest := &CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CdTriggerEventPayloadWithTaskYaml), ciCdRequest)
		ciProjectDetails := ciCdRequest.CdRequest.CiProjectDetails

		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		err := CloneAndCheckout(ciProjectDetails)

		// Assert the expected results
		if err != nil {
			t.Errorf("Error in Test2_ValidCiProjectDetailsUsernamePassword")
		}
	})
	t.Run("Test3_ValidCiProjectDetailsWebhookType", func(t *testing.T) {

		// Prepare test data, USERNAME_PASSWORD and WEBHOOK data
		ciCdRequest := &CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventSourceTypeWebhookPRBased), ciCdRequest)
		ciProjectDetails := ciCdRequest.CiRequest.CiProjectDetails

		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		err := CloneAndCheckout(ciProjectDetails)

		// Assert the expected results
		if err != nil {
			t.Errorf("Error in Test3_ValidCiProjectDetailsWebhookType")
		}
	})
	t.Run("Test4_ValidCiProjectDetailsSSHBasedGitTrigger", func(t *testing.T) {

		// Prepare test data, SSH and SOURCE_TYPE_BRANCH_FIXED data
		ciCdRequest := &CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventSSHBased), ciCdRequest)
		ciProjectDetails := ciCdRequest.CiRequest.CiProjectDetails

		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		err := CloneAndCheckout(ciProjectDetails)

		// Assert the expected results
		if err != nil {
			t.Errorf("Error in Test4_ValidCiProjectDetailsSSHBasedGitTrigger")
		}
	})
	t.Run("Test5_ValidCiProjectDetailsEmptyGitCommit", func(t *testing.T) {

		// Prepare test data, ANONYMOUS and SOURCE_TYPE_BRANCH_FIXED data
		ciCdRequest := &CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventWithEmptyGitHash), ciCdRequest)
		ciProjectDetails := ciCdRequest.CiRequest.CiProjectDetails

		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		err := CloneAndCheckout(ciProjectDetails)

		// Assert the expected results
		if err != nil {
			t.Errorf("Error in Test5_ValidCiProjectDetailsEmptyGitCommit")
		}
	})
	t.Run("Test6_ValidCiProjectDetailsEmptyGitCommitAndSourceValue", func(t *testing.T) {

		// Prepare test data, ANONYMOUS and SOURCE_TYPE_BRANCH_FIXED data
		ciCdRequest := &CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventWithEmptyGitHashAndSourceValue), ciCdRequest)
		ciProjectDetails := ciCdRequest.CiRequest.CiProjectDetails

		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		err := CloneAndCheckout(ciProjectDetails)

		// Assert the expected results
		if err != nil {
			t.Errorf("Error in Test6_ValidCiProjectDetailsEmptyGitCommitAndSourceValue")
		}
	})
	t.Run("Test7_ValidCiProjectDetailsPullSubmodules", func(t *testing.T) {

		// Prepare test data, ANONYMOUS and SOURCE_TYPE_BRANCH_FIXED data
		ciCdRequest := &CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventWithValidGitHash), ciCdRequest)
		ciProjectDetails := ciCdRequest.CiRequest.CiProjectDetails

		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		err := CloneAndCheckout(ciProjectDetails)

		// Assert the expected results
		if err != nil {
			t.Errorf("Error in Test7_ValidCiProjectDetailsPullSubmodules")
		}
	})
	t.Run("Test8_ValidCiProjectDetailsPullSubmodulesUsernamePassword", func(t *testing.T) {

		// Prepare test data, USERNAME_PASSWORD and SOURCE_TYPE_BRANCH_FIXED data
		ciCdRequest := &CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventUsernamePasswordAndPullSubmodules), ciCdRequest)
		ciProjectDetails := ciCdRequest.CiRequest.CiProjectDetails

		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		err := CloneAndCheckout(ciProjectDetails)

		// Assert the expected results
		if err != nil {
			t.Errorf("Error in Test8_ValidCiProjectDetailsPullSubmodulesUsernamePassword")
		}
	})
	t.Run("Test9_ValidCiProjectDetailsInvalidCommitHash", func(t *testing.T) {

		// Prepare test data, ANONYMOUS and SOURCE_TYPE_BRANCH_FIXED data
		ciCdRequest := &CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventWithInValidGitHash), ciCdRequest)
		ciProjectDetails := ciCdRequest.CiRequest.CiProjectDetails

		clonedRepo := ciProjectDetails[0].GitRepository[strings.LastIndex(ciProjectDetails[0].GitRepository, "/"):]
		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		err := CloneAndCheckout(ciProjectDetails)
		err = os.Chdir(util.WORKINGDIR + clonedRepo)
		// Assert the expected results
		if err == nil {
			t.Errorf("Error in Test9_ValidCiProjectDetailsInvalidCommitHash")
		}
	})
	t.Run("Test10_ValidCiProjectDetailsInvalidUsernamePassword", func(t *testing.T) {

		// Prepare test data, USERNAME_PASSWORD and SOURCE_TYPE_BRANCH_FIXED data
		ciCdRequest := &CiCdTriggerEvent{}
		json.Unmarshal([]byte(test_data.CiTriggerEventUsernamePasswordAndPullSubmodules), ciCdRequest)
		ciProjectDetails := ciCdRequest.CiRequest.CiProjectDetails
		ciProjectDetails[0].GitOptions.UserName = "hjgbuhibj"
		ciProjectDetails[0].GitOptions.Password = "ihvfis"
		clonedRepo := ciProjectDetails[0].GitRepository[strings.LastIndex(ciProjectDetails[0].GitRepository, "/"):]
		os.RemoveAll(util.WORKINGDIR)
		// Call the function
		err := CloneAndCheckout(ciProjectDetails)
		err = os.Chdir(util.WORKINGDIR + clonedRepo)
		// Assert the expected results
		if err == nil {
			t.Errorf("Error in Test10_ValidCiProjectDetailsInvalidUsernamePassword")
		}
	})
}
