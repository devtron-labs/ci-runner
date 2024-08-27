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

package bean

import (
	"encoding/base64"
	"encoding/json"
	"github.com/docker/cli/cli/config/types"
)

const (
	YamlSeparator string = "---\n"
)

type DockerAuthConfig struct {
	Username string
	Password string
}

func (r *DockerAuthConfig) GetEncodedRegistryAuth() (string, error) {
	// Create and encode the auth config
	authConfig := types.AuthConfig{
		Username: r.Username,
		Password: r.Password,
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encodedJSON), nil
}
