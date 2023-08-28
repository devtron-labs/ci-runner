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
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/devtron-labs/ci-runner/util"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

type GitOptions struct {
	UserName      string   `json:"userName"`
	Password      string   `json:"password"`
	SshPrivateKey string   `json:"sshPrivateKey"`
	AccessToken   string   `json:"accessToken"`
	AuthMode      AuthMode `json:"authMode"`
}

type WebhookData struct {
	Id              int               `json:"id"`
	EventActionType string            `json:"eventActionType"`
	Data            map[string]string `json:"data"`
}

type GitContext struct {
	context.Context // Embedding original Go context
	auth            *http.BasicAuth
}

type AuthMode string

const (
	AUTH_MODE_USERNAME_PASSWORD AuthMode = "USERNAME_PASSWORD"
	AUTH_MODE_SSH               AuthMode = "SSH"
	AUTH_MODE_ACCESS_TOKEN      AuthMode = "ACCESS_TOKEN"
	AUTH_MODE_ANONYMOUS         AuthMode = "ANONYMOUS"
)

type SourceType string

const (
	SOURCE_TYPE_BRANCH_FIXED SourceType = "SOURCE_TYPE_BRANCH_FIXED"
	SOURCE_TYPE_WEBHOOK      SourceType = "WEBHOOK"
)

const (
	WEBHOOK_SELECTOR_TARGET_CHECKOUT_NAME string = "target checkout"
	WEBHOOK_SELECTOR_SOURCE_CHECKOUT_NAME string = "source checkout"

	WEBHOOK_EVENT_MERGED_ACTION_TYPE     string = "merged"
	WEBHOOK_EVENT_NON_MERGED_ACTION_TYPE string = "non-merged"
)

func CloneAndCheckout(ciProjectDetails []CiProjectDetails) error {
	gitCli := NewGitUtil()
	for index, prj := range ciProjectDetails {

		// git clone
		log.Println("-----> git cloning " + prj.GitRepository)

		if prj.CheckoutPath != "./" {
			if _, err := os.Stat(prj.CheckoutPath); os.IsNotExist(err) {
				_ = os.Mkdir(prj.CheckoutPath, os.ModeDir)
			}
		}
		var cErr error
		var auth *http.BasicAuth
		authMode := prj.GitOptions.AuthMode
		switch authMode {
		case AUTH_MODE_USERNAME_PASSWORD:
			auth = &http.BasicAuth{Password: prj.GitOptions.Password, Username: prj.GitOptions.UserName}
		case AUTH_MODE_ACCESS_TOKEN:
			auth = &http.BasicAuth{Password: prj.GitOptions.AccessToken, Username: prj.GitOptions.UserName}
		default:
			auth = &http.BasicAuth{}
		}

		gitContext := GitContext{
			auth: auth,
		}
		// create ssh private key on disk
		if authMode == AUTH_MODE_SSH {
			cErr = util.CreateSshPrivateKeyOnDisk(index, prj.GitOptions.SshPrivateKey)
			if cErr != nil {
				log.Fatal("could not create ssh private key on disk ", " err ", cErr)
			}
		}

		_, msgMsg, cErr := gitCli.Clone(gitContext, filepath.Join(util.WORKINGDIR, prj.CheckoutPath), prj.GitRepository)
		if cErr != nil {
			log.Fatal("could not clone repo ", " err ", cErr, "msgMsg", msgMsg)
		}

		// checkout code
		if prj.SourceType == SOURCE_TYPE_BRANCH_FIXED {
			// checkout incoming commit hash or branch name
			checkoutSource := ""
			if len(prj.CommitHash) > 0 {
				checkoutSource = prj.CommitHash
			} else {
				if len(prj.SourceValue) == 0 {
					prj.SourceValue = "main"
				}
				checkoutSource = prj.SourceValue
			}
			log.Println("checkout commit in branch fix : ", checkoutSource)
			msgMsg, cErr = Checkout(gitContext, gitCli, prj.CheckoutPath, checkoutSource, authMode, prj.FetchSubmodules, prj.GitRepository)
			if cErr != nil {
				log.Fatal("could not checkout hash ", " err ", cErr, "msgMsg", msgMsg)
			}

		} else if prj.SourceType == SOURCE_TYPE_WEBHOOK {

			webhookData := prj.WebhookData
			webhookDataData := webhookData.Data

			targetCheckout := webhookDataData[WEBHOOK_SELECTOR_TARGET_CHECKOUT_NAME]
			if len(targetCheckout) == 0 {
				log.Fatal("could not get target checkout from request data")
			}

			log.Println("checkout commit in webhook : ", targetCheckout)

			// checkout target hash
			msgMsg, cErr = Checkout(gitContext, gitCli, prj.CheckoutPath, targetCheckout, authMode, prj.FetchSubmodules, prj.GitRepository)
			if cErr != nil {
				log.Fatal("could not checkout  ", "targetCheckout ", targetCheckout, " err ", cErr, " msgMsg", msgMsg)
				return cErr
			}

			// merge source if action type is merged
			if webhookData.EventActionType == WEBHOOK_EVENT_MERGED_ACTION_TYPE {
				sourceCheckout := webhookDataData[WEBHOOK_SELECTOR_SOURCE_CHECKOUT_NAME]

				// throw error if source checkout is empty
				if len(sourceCheckout) == 0 {
					log.Fatal("sourceCheckout is empty")
				}

				log.Println("merge commit in webhook : ", sourceCheckout)

				// merge source
				_, msgMsg, cErr = gitCli.Merge(filepath.Join(util.WORKINGDIR, prj.CheckoutPath), sourceCheckout)
				if cErr != nil {
					log.Fatal("could not merge ", "sourceCheckout ", sourceCheckout, " err ", cErr, " msgMsg", msgMsg)
					return cErr
				}

			}

		}

	}
	return nil
}

func Checkout(gitContext GitContext, gitCli *GitUtil, checkoutPath string, targetCheckout string, authMode AuthMode, fetchSubmodules bool, gitRepository string) (errMsg string, error error) {

	rootDir := filepath.Join(util.WORKINGDIR, checkoutPath)

	// checkout target hash
	_, eMsg, cErr := gitCli.Checkout(rootDir, targetCheckout)
	if cErr != nil {
		return eMsg, cErr
	}

	log.Println(util.DEVTRON, " fetchSubmodules ", fetchSubmodules, " authMode ", authMode)

	if fetchSubmodules {
		httpsAuth := (authMode == AUTH_MODE_USERNAME_PASSWORD) || (authMode == AUTH_MODE_ACCESS_TOKEN)
		if httpsAuth {
			// first remove protocol
			modifiedUrl := strings.ReplaceAll(gitRepository, "https://", "")
			// for bitbucket - if git repo url is started with username, then we need to remove username
			if strings.Contains(modifiedUrl, "bitbucket.org") && !strings.HasPrefix(modifiedUrl, "bitbucket.org") {
				modifiedUrl = modifiedUrl[strings.Index(modifiedUrl, "bitbucket.org"):]
			}
			// build url
			modifiedUrl = "https://" + gitContext.auth.Username + ":" + gitContext.auth.Password + "@" + modifiedUrl

			_, errMsg, cErr = gitCli.UpdateCredentialHelper(rootDir)
			if cErr != nil {
				return errMsg, cErr
			}

			cErr = util.CreateGitCredentialFileAndWriteData(modifiedUrl)
			if cErr != nil {
				return "Error in creating git credential file", cErr
			}

		}

		_, errMsg, cErr = gitCli.RecursiveFetchSubmodules(rootDir)
		if cErr != nil {
			return errMsg, cErr
		}

		// cleanup

		if httpsAuth {
			_, errMsg, cErr = gitCli.UnsetCredentialHelper(rootDir)
			if cErr != nil {
				return errMsg, cErr
			}

			// delete file (~/.git-credentials) (which was created above)
			cErr = util.CleanupAfterFetchingHttpsSubmodules()
			if cErr != nil {
				return "", cErr
			}
		}
	}

	return "", nil

}
