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
	PUSH_CASH                            = "Pushing Cache"
	DOCKER_PUSH_AND_EXTRACT_IMAGE_DIGEST = "Docker Push And Extract Image Digest"
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

func LogStage(name string) {
	//stageTemplate := `
	//------------------------------------------------------------------------------------------------------------------------
	//STAGE:  %s
	//------------------------------------------------------------------------------------------------------------------------`
	//log.Println(fmt.Sprintf(stageTemplate, name))
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

func NewStageInfo(name, status string, startTime, endTime *time.Time) *StageLogData {
	return &StageLogData{
		Status:    status,
		Stage:     name,
		StartTime: startTime,
		EndTime:   endTime,
	}
}

func NewStageInfoWithStartLog(name, status string, startTime, endTime *time.Time) *StageLogData {
	stageInfo := &StageLogData{
		Status:    status,
		Stage:     name,
		StartTime: startTime,
		EndTime:   endTime,
	}
	if startTime == nil {
		stageInfo.SetStartTimeNow()
	}
	stageInfo.Log()
	return stageInfo
}

type StageLogData struct {
	//eg : 'STAGE_INFO|{"stage":"Resource availability","startTime":"2021-01-01T00:00:00Z"}'
	Stage     string     `json:"stage,omitempty"`
	StartTime *time.Time `json:"startTime,omitempty"`
	EndTime   *time.Time `json:"endTime,omitempty"`
	Status    string     `json:"status,omitempty"`
}

func (stageLogData *StageLogData) SetStartTimeNow() {
	currentTime := time.Now()
	stageLogData.StartTime = &currentTime
}

func (stageLogData *StageLogData) SetEndTimeNow() {
	currentTime := time.Now()
	stageLogData.EndTime = &currentTime
}

func (stageLogData *StageLogData) SetStatus(status string) {
	stageLogData.Status = status
}

func (stageLogData *StageLogData) SetStatusEndTimeAndLog(status string) {
	stageLogData.Status = status
	currentTime := time.Now()
	stageLogData.EndTime = &currentTime
	stageLogData.Log()
}

func (stageLogData *StageLogData) SetEndTimeNowAndLog() {
	currentTime := time.Now()
	stageLogData.EndTime = &currentTime
	stageLogData.Log()
}

func (stageLogData *StageLogData) Log() {
	infoLog := fmt.Sprintf("STAGE_INFO|%s\n", stageLogData.String())
	log.SetFlags(0)
	log.Println(infoLog)
	log.SetFlags(log.Ldate | log.Ltime)
}

func (stageLogData *StageLogData) String() string {
	bytes, _ := json.Marshal(stageLogData)
	return string(bytes)
}
