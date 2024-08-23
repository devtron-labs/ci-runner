/*
 * Copyright (c) 2020-2024. Devtron Inc.
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

package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

const (
	SSH_PRIVATE_KEY_DIR       = ".ssh"
	SSH_PRIVATE_KEY_FILE_NAME = "id_rsa"
	GIT_CREDENTIAL_FILE_NAME  = ".git-credentials"
	CLONING_MODE_SHALLOW      = "SHALLOW"
	CLONING_MODE_FULL         = "FULL"
)

const (
	CACHE_PULL                           = "Pulling Cache"
	GIT_CLONE_CHECKOUT                   = "Git Clone & Checkout"
	DOCKER_DAEMON                        = "Starting Docker Daemon"
	DOCKER_LOGIN_STAGE                   = "Docker Login"
	DOCKER_PUSH                          = "Docker Push"
	DOCKER_BUILD                         = "Docker Build"
	DOCKER_STOP                          = "Docker Stop"
	BUILD_ARTIFACT                       = "Build Artifact"
	UPLOAD_ARTIFACT                      = "Uploading Artifact"
	PUSH_CACHE                           = "Pushing Cache"
	DOCKER_PUSH_AND_EXTRACT_IMAGE_DIGEST = "Docker Push And Extract Image Digest"
	IMAGE_SCAN                           = "Image Scanning"
	SETUP_BUILDX_BUILDER                 = "Setting Up Buildx Builder"
	CLEANUP_BUILDX_BUILDER               = "Cleaning Up Buildx Builder"
	BUILD_PACK_BUILD                     = "Build Packs Build"
	EXPORT_BUILD_CACHE                   = "Exporting Build Cache"
)

func CreateSshPrivateKeyOnDisk(fileId int, sshPrivateKeyContent string) error {

	userHomeDirectory, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	sshPrivateKeyFilePath := path.Join(userHomeDirectory, SSH_PRIVATE_KEY_DIR, SSH_PRIVATE_KEY_FILE_NAME)

	// if file exists then delete file
	if _, err := os.Stat(sshPrivateKeyFilePath); os.IsExist(err) {
		os.Remove(sshPrivateKeyFilePath)
	}

	// create file with content
	err = ioutil.WriteFile(sshPrivateKeyFilePath, []byte(sshPrivateKeyContent), 0600)
	if err != nil {
		return err
	}

	return nil
}

func CreateGitCredentialFileAndWriteData(data string) error {

	userHomeDirectory, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	fileName := path.Join(userHomeDirectory, GIT_CREDENTIAL_FILE_NAME)

	// if file exists then delete file
	if _, err := os.Stat(fileName); os.IsExist(err) {
		os.Remove(fileName)
	}

	// create file with content
	err = ioutil.WriteFile(fileName, []byte(data), 0600)
	if err != nil {
		return err
	}

	return nil
}

func CleanupAfterFetchingHttpsSubmodules() error {

	userHomeDirectory, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// remove ~/.git-credentials
	gitCredentialsFile := path.Join(userHomeDirectory, GIT_CREDENTIAL_FILE_NAME)
	if _, err := os.Stat(gitCredentialsFile); os.IsExist(err) {
		os.Remove(gitCredentialsFile)
	}

	return nil
}

var chars = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

// Generates random string
func Generate(size int) string {
	rand.Seed(time.Now().UnixNano())
	var b strings.Builder
	for i := 0; i < size; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	str := b.String()
	return str
}

// CheckFileExists returns boolean value of file existence else error (ignoring file does not exist error)
func CheckFileExists(filename string) (bool, error) {
	if _, err := os.Stat(filename); err == nil {
		// exists
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		// not exists
		return false, nil
	} else {
		// Some other error
		return false, err
	}
}
func ParseUrl(rawURL string) (parsedURL *url.URL, err error) {
	parsedURL, err = url.Parse(rawURL)
	if err != nil || parsedURL.Host == "" {
		parsedURL, err = url.Parse("//" + rawURL)
	}
	return parsedURL, err
}

// GetProjectName this function has been designed for returning project name of git-lab and git-hub providers only
// do not remove this function
func GetProjectName(url string) string {
	//if url = https://github.com/devtron-labs/git-sensor.git then it will return git-sensor
	projName := strings.Split(url, ".")[1]
	projectName := projName[strings.LastIndex(projName, "/")+1:]
	return projectName
}

func newStageInfo(name string) *StageLogData {
	return &StageLogData{
		Stage: name,
	}
}

type Status string

const (
	success Status = "Success"
	failure Status = "Failure"
)

type StageLogData struct {
	//eg : 'STAGE_INFO|{"stage":"Resource availability","startTime":"2021-01-01T00:00:00Z"}'
	Stage     string     `json:"stage,omitempty"`
	StartTime *time.Time `json:"startTime,omitempty"`
	EndTime   *time.Time `json:"endTime,omitempty"`
	Status    Status     `json:"status,omitempty"`
}

func (stageLogData *StageLogData) withStatus(status Status) *StageLogData {
	stageLogData.Status = status
	return stageLogData
}

func (stageLogData *StageLogData) withCurrentStartTime() *StageLogData {
	currentTime := time.Now()
	stageLogData.StartTime = &currentTime
	return stageLogData
}

func (stageLogData *StageLogData) withCurrentEndTime() *StageLogData {
	currentTime := time.Now()
	stageLogData.EndTime = &currentTime
	return stageLogData
}

func (stageLogData *StageLogData) log() {
	infoLog := fmt.Sprintf("STAGE_INFO|%s\n", stageLogData.string())
	log.SetFlags(0)
	log.Println(infoLog)
	log.SetFlags(log.Ldate | log.Ltime)
}

func (stageLogData *StageLogData) string() string {
	bytes, _ := json.Marshal(stageLogData)
	return string(bytes)
}

// ExecuteWithStageInfoLog logs the stage info.
// it will log info for pre stage execution and post the stage execution
// return the error returned by the stageExecutor func
func ExecuteWithStageInfoLog(stageName string, stageExecutor func() error) (err error) {
	startDockerStageInfo := newStageInfo(stageName).withCurrentStartTime()
	startDockerStageInfo.log()
	defer func() {
		status := success
		if err != nil {
			status = failure
		}
		startDockerStageInfo.withStatus(status).withCurrentEndTime().log()
	}()

	return stageExecutor()
}

func GenerateBuildkitdContent(host string) string {
	return fmt.Sprintf(`debug = true
[registry."%s"]
  ca=["/etc/docker/certs.d/%s/ca.crt"]`, host, host)
}

func CreateAndWriteFile(filePath string, content string) error {
	f, err := os.Create(filePath)
	if err != nil {
		log.Printf("Error creating file %s: %v", filePath, err)
		return err
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		log.Printf("Error writing content to file %s: %v", filePath, err)
	}
	return err
}
