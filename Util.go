/*
 * Copyright (c) 2020 Devtron Labs
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package main

import (
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

const (
	SSH_PRIVATE_KEY_DIR = "/ssh-keys/"
	SSH_PRIVATE_KEY_FILE_NAME = "ssh_pvt_key"
)

func CreateSshPrivateKeyOnDisk(fileId int, sshPrivateKeyContent string) (privateKeyPath string, err error) {
	sshPrivateKeyFolderPath := path.Join(SSH_PRIVATE_KEY_DIR, strconv.Itoa(fileId))
	sshPrivateKeyFilePath := path.Join(sshPrivateKeyFolderPath, SSH_PRIVATE_KEY_FILE_NAME)

	// create dirs
	err = os.MkdirAll(sshPrivateKeyFolderPath, os.ModeDir)
	if err != nil {
		return "", err
	}

	// create file with content
	err = ioutil.WriteFile(sshPrivateKeyFilePath, []byte(sshPrivateKeyContent), 0600)
	if err != nil {
		return "", err
	}

	return sshPrivateKeyFilePath, nil
}


func CreateGitCredentialFileAndPutData(data string) error {

	userHomeDirectory, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	fileName := path.Join(userHomeDirectory, ".git-credentials")

	// if file exists then delete file
	if _, err := os.Stat(fileName); os.IsExist(err) {
		os.Remove(fileName)
	}

	// create file with content
	err = ioutil.WriteFile(fileName, []byte(data), 0600)
	if err != nil {
		return  err
	}

	return nil
}