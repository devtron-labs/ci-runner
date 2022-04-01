package main

import (
	"reflect"
	"testing"
)

func TestRunScripts(t *testing.T) {
	type args struct {
		workDirectory  string
		scriptFileName string
		script         string
		envVars        map[string]string
		outputVars     []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    map[string]string
	}{
		{name: "simple_success",
			args:    args{workDirectory: "/tmp/ci-test/", scriptFileName: "test", script: "echo hello", envVars: map[string]string{}, outputVars: nil},
			wantErr: false,
			want:    map[string]string{}},
		{name: "simple_script_fail",
			args:    args{workDirectory: "/tmp/ci-test/", scriptFileName: "test1", script: "err_cmd hello", envVars: map[string]string{}, outputVars: nil},
			wantErr: true,
			want:    nil},
		{name: "env_input_out",
			args:    args{workDirectory: "/tmp/ci-test/", scriptFileName: "test_2", script: "echo hello $name_1 \n export name_2=test_name2 \n echo $name_2", envVars: map[string]string{"name_1": "i am from env"}, outputVars: []string{"name_1", "name_2"}},
			wantErr: false,
			want: map[string]string{
				"name_1": "i am from env",
				"name_2": "test_name2",
			},
		},
		{name: "empty_env_out",
			args:    args{workDirectory: "/tmp/ci-test/", scriptFileName: "test_3", script: "echo hello $name_1 \n export name_2=test_name2 \n echo $name_2", envVars: map[string]string{"name_1": "i am from env"}, outputVars: []string{"name_1", "empty_key", "name_2"}},
			wantErr: false,
			want: map[string]string{
				"name_1":    "i am from env",
				"name_2":    "test_name2",
				"empty_key": "",
			},
		},
		{name: "outValContains_specialChar",
			args:    args{workDirectory: "/tmp/ci-test/", scriptFileName: "test_4", script: "echo hello $name_1 \n export name_2=test_name2 \n echo $name_2", envVars: map[string]string{"name_1": "i am from \"env", "specialCharVal": "a=b"}, outputVars: []string{"name_1", "specialCharVal", "name_2"}},
			wantErr: false,
			want: map[string]string{
				"name_1":         "i am from \"env",
				"name_2":         "test_name2",
				"specialCharVal": "a=b",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RunScripts(tt.args.workDirectory, tt.args.scriptFileName, tt.args.script, tt.args.envVars, tt.args.outputVars)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunScripts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RunScripts() got = %v, want %v", got, tt.want)
			}
		})
	}
}
