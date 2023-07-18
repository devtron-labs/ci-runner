package main

import (
	"github.com/devtron-labs/ci-runner/helper"
	"reflect"
	"testing"
)

func Test_deduceVariables(t *testing.T) {
	type args struct {
		desiredVars             []*helper.VariableObject
		globalVars              map[string]string
		preeCiStageVariable     map[int]map[string]*helper.VariableObject
		postCiStageVariables    map[int]map[string]*helper.VariableObject
		refPluginStageVariables map[int]map[string]*helper.VariableObject
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
				globalVars:              nil,
				preeCiStageVariable:     nil,
				postCiStageVariables:    nil,
				refPluginStageVariables: nil,
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
				globalVars:              map[string]string{"age": "20", "my-name": "test", "status": "true"},
				preeCiStageVariable:     nil,
				postCiStageVariables:    nil,
				refPluginStageVariables: nil,
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
				postCiStageVariables:    nil,
				refPluginStageVariables: nil,
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
				preeCiStageVariable:     nil,
				refPluginStageVariables: nil,
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
			got, err := deduceVariables(tt.args.desiredVars, tt.args.globalVars, tt.args.preeCiStageVariable, tt.args.postCiStageVariables, tt.args.refPluginStageVariables)
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
		stepType                   StepType
		steps                      []*helper.StepObject
		refStageMap                map[int][]*helper.StepObject
		globalEnvironmentVariables map[string]string
		preeCiStageVariable        map[int]map[string]*helper.VariableObject
	}
	tests := []struct {
		name           string
		args           args
		wantOutVars    map[int]map[string]*helper.VariableObject
		wantFailedStep *helper.StepObject
		wantErr        bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOutVars, gotFailedStep, err := RunCiSteps(tt.args.stepType, tt.args.steps, tt.args.refStageMap, tt.args.globalEnvironmentVariables, tt.args.preeCiStageVariable)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunCiSteps() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotOutVars, tt.wantOutVars) {
				t.Errorf("RunCiSteps() gotOutVars = %v, want %v", gotOutVars, tt.wantOutVars)
			}
			if !reflect.DeepEqual(gotFailedStep, tt.wantFailedStep) {
				t.Errorf("RunCiSteps() gotFailedStep = %v, want %v", gotFailedStep, tt.wantFailedStep)
			}
		})
	}
}
