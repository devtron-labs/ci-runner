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
	"context"
	"errors"
	"github.com/devtron-labs/ci-runner/util"
	"log"
	"os"
	"path/filepath"
)

type GitOptions struct {
	UserName              string   `json:"userName"`
	Password              string   `json:"password"`
	SshPrivateKey         string   `json:"sshPrivateKey"`
	AccessToken           string   `json:"accessToken"`
	AuthMode              AuthMode `json:"authMode"`
	TlsKey                string   `json:"tlsKey"`
	TlsCert               string   `json:"tlsCert"`
	CaCert                string   `json:"caCert"`
	EnableTLSVerification bool     `json:"enableTLSVerification"`
}

type WebhookData struct {
	Id              int               `json:"id"`
	EventActionType string            `json:"eventActionType"`
	Data            map[string]string `json:"data"`
}

type GitContext struct {
	context.Context        // Embedding original Go context
	Auth                   *BasicAuth
	CACert                 string
	TLSKey                 string
	TLSCertificate         string
	TLSVerificationEnabled bool
}

func (gitCtx GitContext) WithTLSData(caData string, tlsKey string, tlsCertificate string, tlsVerificationEnabled bool) GitContext {
	gitCtx.CACert = caData
	gitCtx.TLSKey = tlsKey
	gitCtx.TLSCertificate = tlsCertificate
	gitCtx.TLSVerificationEnabled = tlsVerificationEnabled
	return gitCtx
}

type BasicAuth struct {
	Username, Password string
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
	WEBHOOK_SELECTOR_TARGET_CHECKOUT_NAME        string = "target checkout"
	WEBHOOK_SELECTOR_SOURCE_CHECKOUT_NAME        string = "source checkout"
	WEBHOOK_SELECTOR_TARGET_CHECKOUT_BRANCH_NAME string = "target branch name"

	WEBHOOK_EVENT_MERGED_ACTION_TYPE     string = "merged"
	WEBHOOK_EVENT_NON_MERGED_ACTION_TYPE string = "non-merged"
)

type GitManager struct {
	gitCliManager GitCliManager
}

func NewGitManagerImpl(gitCliManager GitCliManager) *GitManager {
	return &GitManager{
		gitCliManager: gitCliManager,
	}
}

func (impl *GitManager) CloneAndCheckout(ciProjectDetails []CiProjectDetails) error {
	cloneAndCheckoutGitMaterials := func() error {
		for index, prj := range ciProjectDetails {
			// git clone
			log.Println("-----> git " + prj.CloningMode + " cloning " + prj.GitRepository)

			if prj.CheckoutPath != "./" {
				if _, err := os.Stat(prj.CheckoutPath); os.IsNotExist(err) {
					_ = os.Mkdir(prj.CheckoutPath, os.ModeDir)
				}
			}
			var cErr error
			var auth *BasicAuth
			authMode := prj.GitOptions.AuthMode
			switch authMode {
			case AUTH_MODE_USERNAME_PASSWORD:
				auth = &BasicAuth{Password: prj.GitOptions.Password, Username: prj.GitOptions.UserName}
			case AUTH_MODE_ACCESS_TOKEN:
				auth = &BasicAuth{Password: prj.GitOptions.AccessToken, Username: prj.GitOptions.UserName}
			default:
				auth = &BasicAuth{}
			}

			gitContext := GitContext{
				Auth: auth,
			}
			// create ssh private key on disk
			if authMode == AUTH_MODE_SSH {
				cErr = util.CreateSshPrivateKeyOnDisk(index, prj.GitOptions.SshPrivateKey)
				cErr = util.CreateSshPrivateKeyOnDisk(index, prj.GitOptions.SshPrivateKey)
				if cErr != nil {
					log.Println("could not create ssh private key on disk ", " err ", cErr)
					return cErr
				}
			}

			_, msgMsg, cErr := impl.gitCliManager.Clone(gitContext, prj)
			if cErr != nil {
				log.Fatal("could not clone repo ", " err ", cErr, "msgMsg", msgMsg)
				return cErr
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
				msgMsg, cErr = impl.gitCliManager.GitCheckout(gitContext, prj.CheckoutPath, checkoutSource, authMode, prj.FetchSubmodules, prj.GitRepository, prj)
				if cErr != nil {
					log.Println("could not checkout hash ", " err ", cErr, "msgMsg", msgMsg)
					return cErr
				}

			} else if prj.SourceType == SOURCE_TYPE_WEBHOOK {

				webhookData := prj.WebhookData
				webhookDataData := webhookData.Data

				targetCheckout := webhookDataData[WEBHOOK_SELECTOR_TARGET_CHECKOUT_NAME]
				if len(targetCheckout) == 0 {
					log.Println("could not get target checkout from request data")
					return errors.New("could not get target checkout from request data for webhook")
				}

				log.Println("checkout commit in webhook : ", targetCheckout)

				// checkout target hash
				msgMsg, cErr = impl.gitCliManager.GitCheckout(gitContext, prj.CheckoutPath, targetCheckout, authMode, prj.FetchSubmodules, prj.GitRepository, prj)
				if cErr != nil {
					log.Println("could not checkout  ", "targetCheckout ", targetCheckout, " err ", cErr, " msgMsg", msgMsg)
					return cErr
				}

				// merge source if action type is merged
				if webhookData.EventActionType == WEBHOOK_EVENT_MERGED_ACTION_TYPE {
					sourceCheckout := webhookDataData[WEBHOOK_SELECTOR_SOURCE_CHECKOUT_NAME]
					// throw error if source checkout is empty
					if len(sourceCheckout) == 0 {
						log.Println("sourceCheckout is empty")
						return errors.New("sourceCheckout is empty")
					}

					log.Println("merge commit in webhook : ", sourceCheckout)

					// merge source
					_, msgMsg, cErr = impl.gitCliManager.Merge(filepath.Join(util.WORKINGDIR, prj.CheckoutPath), sourceCheckout)
					if cErr != nil {
						log.Println("could not merge ", "sourceCheckout ", sourceCheckout, " err ", cErr, " msgMsg", msgMsg)
						return cErr
					}

				}

			}

		}
		return nil
	}

	err := util.ExecuteWithStageInfoLog(util.GIT_CLONE_CHECKOUT, cloneAndCheckoutGitMaterials)
	if err != nil {
		log.Fatal("error in cloning and checking out the git materials ", "err : ", err)
	}
	return err
}
