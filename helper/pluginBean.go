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
	"encoding/json"
	"fmt"
)

type RefPluginObject struct {
	Id    int           `json:"id"`
	Steps []*StepObject `json:"steps"`
}

const STEP_TYPE_INLINE = "INLINE"

//const STEP_TYPE_REF_PLUGIN = "REF_PLUGIN"

/*script string,
envInputVars map[string]string,
outputVars []string
trigger/skip ConditionObject
success/fail condition
ArtifactPaths
*/

type StepObject struct {
	Name                     string             `json:"name"`
	Index                    int                `json:"index"`
	StepType                 string             `json:"stepType"`     // REF_PLUGIN or INLINE
	ExecutorType             ExecutorType       `json:"executorType"` //continer_image/ shell
	RefPluginId              int                `json:"refPluginId"`
	Script                   string             `json:"script"`
	InputVars                []*VariableObject  `json:"inputVars"`
	ExposedPorts             map[int]int        `json:"exposedPorts"` //map of host:container
	OutputVars               []*VariableObject  `json:"outputVars"`
	TriggerSkipConditions    []*ConditionObject `json:"triggerSkipConditions"`
	SuccessFailureConditions []*ConditionObject `json:"successFailureConditions"`
	DockerImage              string             `json:"dockerImage"`
	Command                  string             `json:"command"`
	Args                     []string           `json:"args"`
	CustomScriptMount        *MountPath         `json:"customScriptMount"` // destination path - storeScriptAt
	SourceCodeMount          *MountPath         `json:"sourceCodeMount"`   // destination path - mountCodeToContainerPath
	ExtraVolumeMounts        []*MountPath       `json:"extraVolumeMounts"` // filePathMapping
	ArtifactPaths            []string           `json:"artifactPaths"`
	TriggerIfParentStageFail bool               `json:"triggerIfParentStageFail"`
}

type MountPath struct {
	SrcPath string `json:"sourcePath"`
	DstPath string `json:"destinationPath"`
}

// ------------
type Format int

const (
	STRING Format = iota
	NUMBER
	BOOL
	DATE
)

func (d Format) ValuesOf(format string) (Format, error) {
	if format == "NUMBER" || format == "number" {
		return NUMBER, nil
	} else if format == "BOOL" || format == "bool" || format == "boolean" {
		return BOOL, nil
	} else if format == "STRING" || format == "string" {
		return STRING, nil
	} else if format == "DATE" || format == "date" {
		return DATE, nil
	}
	return STRING, fmt.Errorf("invalid Format: %s", format)
}

func (d Format) String() string {
	return [...]string{"NUMBER", "BOOL", "STRING", "DATE"}[d]
}

func (t Format) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *Format) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	format, err := t.ValuesOf(s)
	if err != nil {
		return err
	}
	*t = format
	return nil
}

// ---------------
type ExecutorType int

const (
	CONTAINER_IMAGE ExecutorType = iota
	SHELL
	PLUGIN // Added to avoid un-marshaling error in REF_PLUGIN type steps, otherwise this value won't be used
)

func (d ExecutorType) ValueOf(executorType string) (ExecutorType, error) {
	if executorType == "CONTAINER_IMAGE" {
		return CONTAINER_IMAGE, nil
	} else if executorType == "SHELL" {
		return SHELL, nil
	} else if executorType == "PLUGIN" {
		return PLUGIN, nil
	}
	return SHELL, fmt.Errorf("invalid executorType:  %s", executorType)
}
func (d ExecutorType) String() string {
	return [...]string{"CONTAINER_IMAGE", "SHELL"}[d]
}
func (t ExecutorType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *ExecutorType) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	execType, err := t.ValueOf(s)
	if err != nil {
		return err
	}
	*t = execType
	return nil
}

// -----
type VariableType int

const (
	VALUE VariableType = iota
	REF_PRE_CI
	REF_POST_CI
	REF_GLOBAL
	REF_PLUGIN
)

func (d VariableType) ValueOf(variableType string) (VariableType, error) {
	if variableType == "VALUE" {
		return VALUE, nil
	} else if variableType == "REF_PRE_CI" {
		return REF_PRE_CI, nil
	} else if variableType == "REF_POST_CI" {
		return REF_POST_CI, nil
	} else if variableType == "REF_GLOBAL" {
		return REF_GLOBAL, nil
	} else if variableType == "REF_PLUGIN" {
		return REF_PLUGIN, nil
	}
	return VALUE, fmt.Errorf("invalid variableType %s", variableType)
}
func (d VariableType) String() string {
	return [...]string{"VALUE", "REF_PRE_CI", "REF_POST_CI", "REF_GLOBAL", "REF_PLUGIN"}[d]
}
func (t VariableType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *VariableType) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	variableType, err := t.ValueOf(s)
	if err != nil {
		return err
	}
	*t = variableType
	return nil
}

// ---------------
type VariableObject struct {
	Name   string `json:"name"`
	Format Format `json:"format"`
	//only for input type
	Value string `json:"value"`
	//	GlobalVarName              string       `json:"globalVarName"`
	ReferenceVariableName      string       `json:"referenceVariableName"`
	VariableType               VariableType `json:"variableType"`
	ReferenceVariableStepIndex int          `json:"referenceVariableStepIndex"`
	VariableStepIndexInPlugin  int          `json:"variableStepIndexInPlugin"`
	TypedValue                 interface{}  `json:"-"` //typeCased and deduced
}

func (v *VariableObject) TypeCheck() error {
	typedValue, err := TypeConverter(v.Value, v.Format)
	if err != nil {
		return err
	}
	v.TypedValue = typedValue
	return nil
}

// ----------
type ConditionType int

const (
	TRIGGER = iota
	SKIP
	PASS
	FAIL
)

func (d ConditionType) ValueOf(conditionType string) (ConditionType, error) {
	if conditionType == "TRIGGER" {
		return TRIGGER, nil
	} else if conditionType == "SKIP" {
		return SKIP, nil
	} else if conditionType == "PASS" {
		return PASS, nil
	} else if conditionType == "FAIL" {
		return FAIL, nil
	}
	return PASS, fmt.Errorf("invalid conditionType: %s", conditionType)
}
func (d ConditionType) String() string {
	return [...]string{"TRIGGER", "SKIP", "PASS", "FAIL"}[d]
}

func (t ConditionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *ConditionType) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	executorType, err := t.ValueOf(s)
	if err != nil {
		return err
	}
	*t = executorType
	return nil
}

//------
