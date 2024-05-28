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

package executor

import (
	"github.com/devtron-labs/ci-runner/helper"
	"reflect"
	"testing"
)

func Test_deduceVariables(t *testing.T) {
	type args struct {
		desiredVars          []*helper.VariableObject
		globalVars           map[string]string
		preeCiStageVariable  map[int]map[string]*helper.VariableObject
		postCiStageVariables map[int]map[string]*helper.VariableObject
	}
	tests := []struct {
		name    string
		args    args
		want    []*helper.VariableObject
		wantErr bool
	}{
		{name: "only value type",
			args: args{
				desiredVars: []*helper.VariableObject{&helper.VariableObject{Name: "age", Value: "20", VariableType: helper.VALUE, Format: helper.NUMBER},
					&helper.VariableObject{Name: "name", Value: "test", VariableType: helper.VALUE, Format: helper.STRING},
					&helper.VariableObject{Name: "status", Value: "true", VariableType: helper.VALUE, Format: helper.BOOL}},
				globalVars:           nil,
				preeCiStageVariable:  nil,
				postCiStageVariables: nil,
			},
			wantErr: false,
			want: []*helper.VariableObject{&helper.VariableObject{Name: "age", Value: "20", VariableType: helper.VALUE, Format: helper.NUMBER},
				&helper.VariableObject{Name: "name", Value: "test", VariableType: helper.VALUE, Format: helper.STRING},
				&helper.VariableObject{Name: "status", Value: "true", VariableType: helper.VALUE, Format: helper.BOOL}},
		}, {name: "from global",
			args: args{
				desiredVars: []*helper.VariableObject{&helper.VariableObject{Name: "age", VariableType: helper.REF_GLOBAL, Format: helper.NUMBER, ReferenceVariableName: "age"},
					&helper.VariableObject{Name: "name", VariableType: helper.REF_GLOBAL, Format: helper.STRING, ReferenceVariableName: "my-name"},
					&helper.VariableObject{Name: "status", VariableType: helper.REF_GLOBAL, Format: helper.BOOL, ReferenceVariableName: "status"}},
				globalVars:           map[string]string{"age": "20", "my-name": "test", "status": "true"},
				preeCiStageVariable:  nil,
				postCiStageVariables: nil,
			},
			wantErr: false,
			want: []*helper.VariableObject{&helper.VariableObject{Name: "age", Value: "20", VariableType: helper.REF_GLOBAL, Format: helper.NUMBER, TypedValue: float64(20), ReferenceVariableName: "age"},
				&helper.VariableObject{Name: "name", Value: "test", VariableType: helper.REF_GLOBAL, Format: helper.STRING, TypedValue: "test", ReferenceVariableName: "my-name"},
				&helper.VariableObject{Name: "status", Value: "true", VariableType: helper.REF_GLOBAL, Format: helper.BOOL, TypedValue: true, ReferenceVariableName: "status"}},
		}, {name: "REF_PRE_CI",
			args: args{
				desiredVars: []*helper.VariableObject{&helper.VariableObject{Name: "age", VariableType: helper.REF_PRE_CI, Format: helper.NUMBER, ReferenceVariableName: "age", ReferenceVariableStepIndex: 1},
					&helper.VariableObject{Name: "name", VariableType: helper.REF_PRE_CI, Format: helper.STRING, ReferenceVariableName: "my-name", ReferenceVariableStepIndex: 1},
					&helper.VariableObject{Name: "status", VariableType: helper.REF_PRE_CI, Format: helper.BOOL, ReferenceVariableName: "status", ReferenceVariableStepIndex: 1}},
				globalVars: map[string]string{"age": "22", "my-name": "test1", "status": "false"},
				preeCiStageVariable: map[int]map[string]*helper.VariableObject{1: {"age": &helper.VariableObject{Name: "age", Value: "20"},
					"my-name": &helper.VariableObject{Name: "my-name", Value: "test"},
					"status":  &helper.VariableObject{Name: "status", Value: "true"},
				}},
				postCiStageVariables: nil,
			},
			wantErr: false,
			want: []*helper.VariableObject{&helper.VariableObject{Name: "age", VariableType: helper.REF_PRE_CI, Format: helper.NUMBER, ReferenceVariableName: "age", Value: "20", TypedValue: float64(20), ReferenceVariableStepIndex: 1},
				&helper.VariableObject{Name: "name", VariableType: helper.REF_PRE_CI, Format: helper.STRING, ReferenceVariableName: "my-name", Value: "test", TypedValue: "test", ReferenceVariableStepIndex: 1},
				&helper.VariableObject{Name: "status", VariableType: helper.REF_PRE_CI, Format: helper.BOOL, ReferenceVariableName: "status", Value: "true", TypedValue: true, ReferenceVariableStepIndex: 1}},
		}, {name: "REF_POST_CI",
			args: args{
				desiredVars: []*helper.VariableObject{&helper.VariableObject{Name: "age", VariableType: helper.REF_POST_CI, Format: helper.NUMBER, ReferenceVariableName: "age", ReferenceVariableStepIndex: 1},
					&helper.VariableObject{Name: "name", VariableType: helper.REF_POST_CI, Format: helper.STRING, ReferenceVariableName: "my-name", ReferenceVariableStepIndex: 1},
					&helper.VariableObject{Name: "status", VariableType: helper.REF_POST_CI, Format: helper.BOOL, ReferenceVariableName: "status", ReferenceVariableStepIndex: 1}},
				globalVars: map[string]string{"age": "22", "my-name": "test1", "status": "false"},
				postCiStageVariables: map[int]map[string]*helper.VariableObject{1: {"age": &helper.VariableObject{Name: "age", Value: "20"},
					"my-name": &helper.VariableObject{Name: "my-name", Value: "test"},
					"status":  &helper.VariableObject{Name: "status", Value: "true"},
				}},
				preeCiStageVariable: nil,
			},
			wantErr: false,
			want: []*helper.VariableObject{&helper.VariableObject{Name: "age", VariableType: helper.REF_POST_CI, Format: helper.NUMBER, ReferenceVariableName: "age", Value: "20", TypedValue: float64(20), ReferenceVariableStepIndex: 1},
				&helper.VariableObject{Name: "name", VariableType: helper.REF_POST_CI, Format: helper.STRING, ReferenceVariableName: "my-name", Value: "test", TypedValue: "test", ReferenceVariableStepIndex: 1},
				&helper.VariableObject{Name: "status", VariableType: helper.REF_POST_CI, Format: helper.BOOL, ReferenceVariableName: "status", Value: "true", TypedValue: true, ReferenceVariableStepIndex: 1}},
		},

		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := deduceVariables(tt.args.desiredVars, tt.args.globalVars, tt.args.preeCiStageVariable, tt.args.postCiStageVariables, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("deduceVariables() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("deduceVariables() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunCiSteps(t *testing.T) {
	type args struct {
		stageType                  helper.StepType
		req                        *helper.CommonWorkflowRequest
		globalEnvironmentVariables map[string]string
		preeCiStageVariable        map[int]map[string]*helper.VariableObject
	}
	tests := []struct {
		name                       string
		args                       args
		wantPreeCiStageVariableOut map[int]map[string]*helper.VariableObject
		wantPostCiStageVariable    map[int]map[string]*helper.VariableObject
		wantErr                    bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		stageExecutor := NewStageExecutorImpl()
		t.Run(tt.name, func(t *testing.T) {
			gotPreeCiStageVariableOut, gotPostCiStageVariable, err := stageExecutor.RunCiCdSteps(tt.args.stageType, nil, tt.args.req.PreCiSteps, nil, tt.args.globalEnvironmentVariables, tt.args.preeCiStageVariable)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunCiCdSteps() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotPreeCiStageVariableOut, tt.wantPreeCiStageVariableOut) {
				t.Errorf("RunCiCdSteps() gotPreeCiStageVariableOut = %v, want %v", gotPreeCiStageVariableOut, tt.wantPreeCiStageVariableOut)
			}
			if !reflect.DeepEqual(gotPostCiStageVariable, tt.wantPostCiStageVariable) {
				t.Errorf("RunCiCdSteps() gotPostCiStageVariable = %v, want %v", gotPostCiStageVariable, tt.wantPostCiStageVariable)
			}
		})
	}
}
