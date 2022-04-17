package main

import (
	"reflect"
	"testing"
)

func Test_deduceVariables(t *testing.T) {
	type args struct {
		desiredVars          []*VariableObject
		globalVars           map[string]string
		preeCiStageVariable  map[int]map[string]*VariableObject
		postCiStageVariables map[int]map[string]*VariableObject
	}
	tests := []struct {
		name    string
		args    args
		want    []*VariableObject
		wantErr bool
	}{
		{name: "only value type",
			args: args{
				desiredVars: []*VariableObject{&VariableObject{Name: "age", Value: "20", VariableType: VALUE, Format: NUMBER},
					&VariableObject{Name: "name", Value: "test", VariableType: VALUE, Format: STRING},
					&VariableObject{Name: "status", Value: "true", VariableType: VALUE, Format: BOOL}},
				globalVars:           nil,
				preeCiStageVariable:  nil,
				postCiStageVariables: nil,
			},
			wantErr: false,
			want: []*VariableObject{&VariableObject{Name: "age", Value: "20", VariableType: VALUE, Format: NUMBER},
				&VariableObject{Name: "name", Value: "test", VariableType: VALUE, Format: STRING},
				&VariableObject{Name: "status", Value: "true", VariableType: VALUE, Format: BOOL}},
		}, {name: "from global",
			args: args{
				desiredVars: []*VariableObject{&VariableObject{Name: "age", VariableType: REF_GLOBAL, Format: NUMBER, ReferenceVariableName: "age"},
					&VariableObject{Name: "name", VariableType: REF_GLOBAL, Format: STRING, ReferenceVariableName: "my-name"},
					&VariableObject{Name: "status", VariableType: REF_GLOBAL, Format: BOOL, ReferenceVariableName: "status"}},
				globalVars:           map[string]string{"age": "20", "my-name": "test", "status": "true"},
				preeCiStageVariable:  nil,
				postCiStageVariables: nil,
			},
			wantErr: false,
			want: []*VariableObject{&VariableObject{Name: "age", Value: "20", VariableType: REF_GLOBAL, Format: NUMBER, TypedValue: float64(20), ReferenceVariableName: "age"},
				&VariableObject{Name: "name", Value: "test", VariableType: REF_GLOBAL, Format: STRING, TypedValue: "test", ReferenceVariableName: "my-name"},
				&VariableObject{Name: "status", Value: "true", VariableType: REF_GLOBAL, Format: BOOL, TypedValue: true, ReferenceVariableName: "status"}},
		}, {name: "REF_PREE_CI",
			args: args{
				desiredVars: []*VariableObject{&VariableObject{Name: "age", VariableType: REF_PREE_CI, Format: NUMBER, ReferenceVariableName: "age", ReferenceVariableStepIndex: 1},
					&VariableObject{Name: "name", VariableType: REF_PREE_CI, Format: STRING, ReferenceVariableName: "my-name", ReferenceVariableStepIndex: 1},
					&VariableObject{Name: "status", VariableType: REF_PREE_CI, Format: BOOL, ReferenceVariableName: "status", ReferenceVariableStepIndex: 1}},
				globalVars: map[string]string{"age": "22", "my-name": "test1", "status": "false"},
				preeCiStageVariable: map[int]map[string]*VariableObject{1: {"age": &VariableObject{Name: "age", Value: "20"},
					"my-name": &VariableObject{Name: "my-name", Value: "test"},
					"status":  &VariableObject{Name: "status", Value: "true"},
				}},
				postCiStageVariables: nil,
			},
			wantErr: false,
			want: []*VariableObject{&VariableObject{Name: "age", VariableType: REF_PREE_CI, Format: NUMBER, ReferenceVariableName: "age", Value: "20", TypedValue: float64(20), ReferenceVariableStepIndex: 1},
				&VariableObject{Name: "name", VariableType: REF_PREE_CI, Format: STRING, ReferenceVariableName: "my-name", Value: "test", TypedValue: "test", ReferenceVariableStepIndex: 1},
				&VariableObject{Name: "status", VariableType: REF_PREE_CI, Format: BOOL, ReferenceVariableName: "status", Value: "true", TypedValue: true, ReferenceVariableStepIndex: 1}},
		}, {name: "REF_POST_CI",
			args: args{
				desiredVars: []*VariableObject{&VariableObject{Name: "age", VariableType: REF_POST_CI, Format: NUMBER, ReferenceVariableName: "age", ReferenceVariableStepIndex: 1},
					&VariableObject{Name: "name", VariableType: REF_POST_CI, Format: STRING, ReferenceVariableName: "my-name", ReferenceVariableStepIndex: 1},
					&VariableObject{Name: "status", VariableType: REF_POST_CI, Format: BOOL, ReferenceVariableName: "status", ReferenceVariableStepIndex: 1}},
				globalVars: map[string]string{"age": "22", "my-name": "test1", "status": "false"},
				postCiStageVariables: map[int]map[string]*VariableObject{1: {"age": &VariableObject{Name: "age", Value: "20"},
					"my-name": &VariableObject{Name: "my-name", Value: "test"},
					"status":  &VariableObject{Name: "status", Value: "true"},
				}},
				preeCiStageVariable: nil,
			},
			wantErr: false,
			want: []*VariableObject{&VariableObject{Name: "age", VariableType: REF_POST_CI, Format: NUMBER, ReferenceVariableName: "age", Value: "20", TypedValue: float64(20), ReferenceVariableStepIndex: 1},
				&VariableObject{Name: "name", VariableType: REF_POST_CI, Format: STRING, ReferenceVariableName: "my-name", Value: "test", TypedValue: "test", ReferenceVariableStepIndex: 1},
				&VariableObject{Name: "status", VariableType: REF_POST_CI, Format: BOOL, ReferenceVariableName: "status", Value: "true", TypedValue: true, ReferenceVariableStepIndex: 1}},
		},

		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := deduceVariables(tt.args.desiredVars, tt.args.globalVars, tt.args.preeCiStageVariable, tt.args.postCiStageVariables)
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
		stageType                  string
		req                        *CiRequest
		globalEnvironmentVariables map[string]string
		preeCiStageVariable        map[int]map[string]*VariableObject
	}
	tests := []struct {
		name                       string
		args                       args
		wantPreeCiStageVariableOut map[int]map[string]*VariableObject
		wantPostCiStageVariable    map[int]map[string]*VariableObject
		wantErr                    bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPreeCiStageVariableOut, gotPostCiStageVariable, err := RunCiSteps(tt.args.stageType, tt.args.req, tt.args.globalEnvironmentVariables, tt.args.preeCiStageVariable)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunCiSteps() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotPreeCiStageVariableOut, tt.wantPreeCiStageVariableOut) {
				t.Errorf("RunCiSteps() gotPreeCiStageVariableOut = %v, want %v", gotPreeCiStageVariableOut, tt.wantPreeCiStageVariableOut)
			}
			if !reflect.DeepEqual(gotPostCiStageVariable, tt.wantPostCiStageVariable) {
				t.Errorf("RunCiSteps() gotPostCiStageVariable = %v, want %v", gotPostCiStageVariable, tt.wantPostCiStageVariable)
			}
		})
	}
}
