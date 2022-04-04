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
		{name: "dev",
			args: args{executionConf: &executionConf{
				DockerImage:             "alpine:latest",
				EnvInputFileName:        "/tmp/ci-test/abc.env",
				MountCode:               true,
				SourceCodeLocation:      "/tmp/code-location",
				SourceCodeMountLocation: "/tmp/code-mount-location",
				ScriptLocation:          "/tmp/custom-script-location",
				ScriptMountLocation:     "/tmp/script-mount-location",
				EntryScriptFileName:     "/tmp/code-location/_entry.sh",
				ExposedPorts:            map[int]int{80: 8080, 90: 9090},
			}},
			wantErr: false,
			want:    "docker run -it \\\n--env-file /tmp/ci-test/abc.env \\\n-v /tmp/code-location/_entry.sh:/devtron_script/_entry.sh \\\n-v /tmp/code-location:/tmp/code-mount-location \\\n-v /tmp/custom-script-location:/tmp/script-mount-location \\\n-p 80:8080 \\\n-p 90:9090 \\\nalpine:latest \\\n/bin/sh /devtron_script/_entry.sh\n",
		},

		// TODO: Add test cases.
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
		command        string
		args           []string
		outputVars     []string
		envOutFileName string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{{name: "hello",
		args:    args{command: "ls", envOutFileName: "out.env"},
		wantErr: false,
		want:    "#!/bin/sh\nset -e\nset -o pipefail\nls \n> out.env\n"},
		{name: "ls_dir",
			args:    args{command: "ls", envOutFileName: "out.env", args: []string{"\\tmp"}},
			wantErr: false,
			want:    "#!/bin/sh\nset -e\nset -o pipefail\nls \\tmp\n> out.env\n"},
		{name: "ls_dir_with_out",
			args:    args{command: "ls", envOutFileName: "out.env", args: []string{"\\tmp"}, outputVars: []string{"HOME"}},
			wantErr: false,
			want:    "#!/bin/sh\nset -e\nset -o pipefail\nls \\tmp\n> out.env\nprintf \"\\nHOME=%s\" \"$HOME\" >> out.env\n"},
		{name: "ls_dir_with_out_multi",
			args:    args{command: "ls", envOutFileName: "out.env", args: []string{"\\tmp"}, outputVars: []string{"HOME", "USER"}},
			wantErr: false,
			want:    "#!/bin/sh\nset -e\nset -o pipefail\nls \\tmp\n> out.env\nprintf \"\\nHOME=%s\" \"$HOME\" >> out.env\nprintf \"\\nUSER=%s\" \"$USER\" >> out.env\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildDockerEntryScript(tt.args.command, tt.args.args, tt.args.outputVars, tt.args.envOutFileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildDockerEntryScript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("buildDockerEntryScript() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunScriptsInDocker(t *testing.T) {
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
					Script:                  "ls",
					ScriptLocation:          "/tmp/custom-script-location",
					ScriptMountLocation:     "/tmp/script-mount-location",
					EnvInputVars:            nil,
					ExposedPorts:            map[int]int{80: 8080, 90: 9090},
					OutputVars:              []string{"HOME", "PWD", "NAME"},
					DockerImage:             "alpine:latest",
					MountCode:               true,
					SourceCodeLocation:      "/tmp/code-location",
					SourceCodeMountLocation: "/tmp/code-mount-location",
					command:                 "/bin/sh",
					args:                    []string{"-c", "ls;sleep 1;export NAME=nishant;echo done;"},
					scriptFileName:          "",
					workDirectory:           "/tmp/ci-test",
				},
			},
			want:    nil,
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
