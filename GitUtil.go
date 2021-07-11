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

package main

import (
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"log"
	"os"
	"path/filepath"
)

func CloneAndCheckout(ciProjectDetails []CiProjectDetails) error {
	gitCli := NewGitUtil()
	for _, prj := range ciProjectDetails {

		// git clone
		log.Println("-----> git cloning " + prj.GitRepository)

		if prj.CheckoutPath != "./" {
			if _, err := os.Stat(prj.CheckoutPath); os.IsNotExist(err) {
				_ = os.Mkdir(prj.CheckoutPath, os.ModeDir)
			}
		}
		var cErr error
		var auth *http.BasicAuth
		switch prj.GitOptions.AuthMode {
		case AUTH_MODE_USERNAME_PASSWORD:
			auth = &http.BasicAuth{Password: prj.GitOptions.Password, Username: prj.GitOptions.UserName}
		case AUTH_MODE_ACCESS_TOKEN:
			auth = &http.BasicAuth{Password: prj.GitOptions.AccessToken, Username: prj.GitOptions.UserName}
		default:
			auth = &http.BasicAuth{}
		}

		_, msgMsg, cErr := gitCli.Clone(filepath.Join(workingDir, prj.CheckoutPath), prj.GitRepository, auth.Username, auth.Password)
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
			_, msgMsg, cErr = gitCli.Checkout(filepath.Join(workingDir, prj.CheckoutPath), checkoutSource)
			if cErr != nil {
				log.Fatal("could not checkout hash ", " err ", cErr, "msgMsg", msgMsg)
			}

		} else if prj.SourceType == SOURCE_TYPE_TAG_REGEX {
			// checkout incoming tag
			checkoutSource := prj.GitTag
			_, msgMsg, cErr = gitCli.Checkout(filepath.Join(workingDir, prj.CheckoutPath), checkoutSource)
			if cErr != nil {
				log.Fatal("could not checkout hash ", " err ", cErr, "msgMsg", msgMsg)
			}
		} else if prj.SourceType == SOURCE_TYPE_WEBHOOK {

			targetCommitHash := prj.WebhookData.Data[WEBHOOK_SELECTOR_TARGET_COMMIT_HASH_NAME]
			if len(targetCommitHash) == 0{
				log.Fatal("could not get target commit hash from request data")
			}

			// checkout target hash
			_, msgMsg, cErr = gitCli.Checkout(filepath.Join(workingDir, prj.CheckoutPath), targetCommitHash)
			if cErr != nil {
				log.Fatal("could not checkout hash ", "hash ", targetCommitHash, " err ", cErr, "msgMsg", msgMsg)
				return cErr
			}

			// merge source hash if found
			sourceCommitHash := prj.WebhookData.Data[WEBHOOK_SELECTOR_SOURCE_COMMIT_HASH_NAME]
			if len(sourceCommitHash) != 0 {
				_, msgMsg, cErr = gitCli.Merge(filepath.Join(workingDir, prj.CheckoutPath), sourceCommitHash)
				if cErr != nil {
					log.Fatal("could not merge hash ", "hash ", sourceCommitHash, "err ", cErr, "msgMsg", msgMsg)
					return cErr
				}
			}

		}

	}
	return nil
}
