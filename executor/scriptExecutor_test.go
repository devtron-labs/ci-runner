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
	"fmt"
	"github.com/devtron-labs/ci-runner/helper"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestRunScripts(t *testing.T) {
	t.SkipNow()
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

func Test_buildDockerRunCommand(t *testing.T) {
	type args struct {
		executionConf *executionConf
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "all_single",
			args: args{executionConf: &executionConf{
				DockerImage:         "alpine:latest",
				EnvInputFileName:    "/tmp/ci-test/abc.env",
				EntryScriptFileName: "/tmp/code-location/_entry.sh",
				EnvOutFileName:      "/tmp/ci-test/_env.out",
				ExtraVolumeMounts:   []*helper.MountPath{{SrcPath: "/src", DstPath: "/des"}},
				SourceCodeMount:     &helper.MountPath{SrcPath: "/tmp/code-location", DstPath: "/tmp/code-mount-location"},
				CustomScriptMount:   &helper.MountPath{SrcPath: "/tmp/custom-script-location", DstPath: "/tmp/script-mount-location"},
				ExposedPorts:        map[int]int{80: 8080},
			}},
			wantErr: false,
			want:    "docker run --network host \\\n--env-file /tmp/ci-test/abc.env \\\n-v /tmp/code-location/_entry.sh:/devtron_script/_entry.sh \\\n-v /tmp/ci-test/_env.out:/devtron_script/_out.env \\\n-v /tmp/code-location:/tmp/code-mount-location \\\n-v /src:/des \\\n-v /tmp/custom-script-location:/tmp/script-mount-location \\\n-p 80:8080 \\alpine:latest \\\n/bin/sh /devtron_script/_entry.sh\n",
		},
		{name: "all_multi",
			args: args{executionConf: &executionConf{
				DockerImage:         "alpine:latest",
				EnvInputFileName:    "/tmp/ci-test/abc.env",
				EntryScriptFileName: "/tmp/code-location/_entry.sh",
				EnvOutFileName:      "/tmp/ci-test/_env.out",
				ExtraVolumeMounts:   []*helper.MountPath{{SrcPath: "/src", DstPath: "/des"}, {SrcPath: "/src2", DstPath: "/des2"}},
				SourceCodeMount:     &helper.MountPath{SrcPath: "/tmp/code-location", DstPath: "/tmp/code-mount-location"},
				CustomScriptMount:   &helper.MountPath{SrcPath: "/tmp/custom-script-location", DstPath: "/tmp/script-mount-location"},
				ExposedPorts:        map[int]int{80: 8080, 90: 9090},
			}},
			wantErr: false,
			want:    "docker run --network host \\\n--env-file /tmp/ci-test/abc.env \\\n-v /tmp/code-location/_entry.sh:/devtron_script/_entry.sh \\\n-v /tmp/ci-test/_env.out:/devtron_script/_out.env \\\n-v /tmp/code-location:/tmp/code-mount-location \\\n-v /src:/des \\\n-v /src2:/des2 \\\n-v /tmp/custom-script-location:/tmp/script-mount-location \\\n-p 80:8080 \\\n-p 90:9090 \\alpine:latest \\\n/bin/sh /devtron_script/_entry.sh\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildDockerRunCommand(tt.args.executionConf)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildDockerRunCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("buildDockerRunCommand() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildDockerEntryScript(t *testing.T) {
	type args struct {
		command    string
		args       []string
		outputVars []string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{{name: "hello",
		args:    args{command: "ls"},
		wantErr: false,
		want:    "#!/bin/sh\nset -e\nls \n> /devtron_script/_out.env\n"},
		{name: "ls_dir",
			args:    args{command: "ls", args: []string{"\\tmp"}},
			wantErr: false,
			want:    "#!/bin/sh\nset -e\nls \\tmp\n> /devtron_script/_out.env\n"},
		{name: "ls_dir_with_out",
			args:    args{command: "ls", args: []string{"\\tmp"}, outputVars: []string{"HOME"}},
			wantErr: false,
			want:    "#!/bin/sh\nset -e\nls \\tmp\n> /devtron_script/_out.env\nprintf \"\\nHOME=%s\" \"$HOME\" >> /devtron_script/_out.env\n"},
		{name: "ls_dir_with_out_multi",
			args:    args{command: "ls", args: []string{"\\tmp"}, outputVars: []string{"HOME", "USER"}},
			wantErr: false,
			want:    "#!/bin/sh\nset -e\nls \\tmp\n> /devtron_script/_out.env\nprintf \"\\nHOME=%s\" \"$HOME\" >> /devtron_script/_out.env\nprintf \"\\nUSER=%s\" \"$USER\" >> /devtron_script/_out.env\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildDockerEntryScript(tt.args.command, tt.args.args, tt.args.outputVars)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildDockerEntryScript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("buildDockerEntryScript() got = %v, want %v", got, tt.want)
			}
		})
	}
	fmt.Println("coverage:", testing.CoverMode(), testing.Coverage())
}

func TestRunScriptsInDocker(t *testing.T) {
	t.SkipNow()
	type args struct {
		executionConf *executionConf
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		{name: "hello",
			args: args{
				executionConf: &executionConf{
					Script:            "ls",
					EnvInputVars:      map[string]string{"KIND": "TEST"},
					ExposedPorts:      map[int]int{80: 8080, 90: 9090},
					OutputVars:        []string{"HOME", "PWD", "NAME", "KIND"},
					DockerImage:       "alpine:latest",
					SourceCodeMount:   &helper.MountPath{SrcPath: "/tmp/code-location", DstPath: "/tmp/code-mount-location"},
					CustomScriptMount: &helper.MountPath{SrcPath: "/tmp/custom-script-location", DstPath: "/tmp/script-mount-location"},
					command:           "/bin/sh",
					args:              []string{"-c", "ls;sleep 1;export NAME=from-script;echo done;"},
					scriptFileName:    "",
					workDirectory:     "/tmp/ci-test",
				},
			},
			want:    map[string]string{"HOME": "/root", "PWD": "/", "NAME": "from-script", "KIND": "TEST"},
			wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RunScriptsInDocker(tt.args.executionConf)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunScriptsInDocker() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RunScriptsInDocker() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_writeToEnvFile(t *testing.T) {
	type args struct {
		envMap   map[string]string
		filename string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{{
		name: "empty_env_map",
		args: args{
			envMap:   map[string]string{},
			filename: "test.env",
		},
		wantErr: false,
	}, {
		name: "single_env_var",
		args: args{
			envMap:   map[string]string{"FOO": "BAR"},
			filename: "test.env",
		},
		wantErr: false,
	}, {
		name: "multiple_env_vars",
		args: args{
			envMap:   map[string]string{"FOO": "BAR", "BAR": "FOO"},
			filename: "test.env",
		},
		wantErr: false,
	}, {
		name: "error_creating_file",
		args: args{
			envMap:   map[string]string{"FOO": "BAR"},
			filename: "/dev/null/abcd",
		},
		wantErr: true,
	},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := writeToEnvFile(tt.args.envMap, tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeToEnvFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				// Check the contents of the file
				file, err := os.Open(tt.args.filename)
				if err != nil {
					t.Errorf("Error opening file: %v", err)
					return
				}
				defer file.Close()
				contents, err := ioutil.ReadAll(file)
				if err != nil {
					t.Errorf("Error reading file contents: %v", err)
					return
				}
				for k, v := range tt.args.envMap {
					if !strings.Contains(string(contents), fmt.Sprintf("%s=%s", k, v)) {
						t.Errorf("Expected to find env var %s=%s in file, but it was not found", k, v)
					}
				}
			}
		})
	}
}
