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
	"fmt"
	"github.com/devtron-labs/common-lib/utils/workFlow"
)

type CiStageError struct {
	stageErr         error
	metrics          *CIMetrics
	failureMessage   string
	artifactUploaded bool
}

func NewCiStageError(err error) *CiStageError {
	return &CiStageError{stageErr: err}
}

func (err *CiStageError) WithMetrics(metrics *CIMetrics) *CiStageError {
	err.metrics = metrics
	return err
}

func (err *CiStageError) WithFailureMessage(failureMessage string) *CiStageError {
	err.failureMessage = failureMessage
	return err
}

func (err *CiStageError) WithArtifactUploaded(artifactUploaded bool) *CiStageError {
	err.artifactUploaded = artifactUploaded
	return err
}

func (err *CiStageError) GetMetrics() CIMetrics {
	if err.metrics == nil {
		return CIMetrics{}
	}
	return *err.metrics
}

func (err *CiStageError) GetFailureMessage() string {
	return err.failureMessage
}

func (err *CiStageError) IsArtifactUploaded() bool {
	return err.artifactUploaded
}

func (err *CiStageError) Error() string {
	return err.stageErr.Error()
}

// ErrorMessage returns the error message with the failure message
func (err *CiStageError) ErrorMessage() string {
	if len(err.failureMessage) != 0 && err.failureMessage != workFlow.CiFailed.String() {
		return err.failureMessage
	} else if err.failureMessage == workFlow.CiFailed.String() {
		return fmt.Sprintf("%s. Reason: %s", err.failureMessage, err.stageErr.Error())
	} else {
		return err.stageErr.Error()
	}
}

func (err *CiStageError) Unwrap() error {
	return err.stageErr
}

type CdStageError struct {
	stageErr         error
	failureMessage   string
	artifactUploaded bool
}

func NewCdStageError(err error) *CdStageError {
	return &CdStageError{stageErr: err}
}

func (err *CdStageError) WithFailureMessage(failureMessage string) *CdStageError {
	err.failureMessage = failureMessage
	return err
}

func (err *CdStageError) WithArtifactUploaded(artifactUploaded bool) *CdStageError {
	err.artifactUploaded = artifactUploaded
	return err
}

func (err *CdStageError) GetFailureMessage() string {
	return err.failureMessage
}

func (err *CdStageError) IsArtifactUploaded() bool {
	return err.artifactUploaded
}

func (err *CdStageError) Error() string {
	return err.stageErr.Error()
}

// ErrorMessage returns the error message with the failure message
func (err *CdStageError) ErrorMessage() string {
	if len(err.failureMessage) != 0 {
		return err.failureMessage
	} else {
		return err.stageErr.Error()
	}
}

func (err *CdStageError) Unwrap() error {
	return err.stageErr
}
