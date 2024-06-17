package helper

import (
	"reflect"
	"testing"
)

func Test_getDockerBuildFlagsMap(t *testing.T) {
	type args struct {
		dockerBuildConfig *DockerBuildConfig
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "test with empty",
			args: args{dockerBuildConfig: &DockerBuildConfig{
				Args:               map[string]string{},
				DockerBuildOptions: map[string]string{},
			}},
			want: map[string]string{},
		},
		{
			name: "test with no especial character",
			args: args{dockerBuildConfig: &DockerBuildConfig{
				Args:               map[string]string{"key1": "value1", "key2": "value2"},
				DockerBuildOptions: map[string]string{"key3": "value3", "key4": "value4"},
			}},
			want: map[string]string{"--build-arg key1": "=\"value1\"", "--build-arg key2": "=\"value2\"", "--key3": "=\"value3\"", "--key4": "=\"value4\""},
		},
		{
			name: "test with special characters",
			args: args{dockerBuildConfig: &DockerBuildConfig{
				Args:               map[string]string{"key1": "value1=& abcd", "key2": "value2=&abcd"},
				DockerBuildOptions: map[string]string{"key3": "value3=& abcd", "key4": "value4=& abcd"},
			}},
			want: map[string]string{"--build-arg key1": "=\"value1=& abcd\"", "--build-arg key2": "=\"value2=&abcd\"", "--key3": "=\"value3=& abcd\"", "--key4": "=\"value4=& abcd\""},
		},
		{
			name: "test backward compatibility with already quoted values special characters",
			args: args{dockerBuildConfig: &DockerBuildConfig{
				Args:               map[string]string{"key1": "\"value1=& abcd\"", "key2": "\"value2=&abcd\""},
				DockerBuildOptions: map[string]string{"key3": "\"value3=& abcd\"", "key4": "\"value4=& abcd\""},
			}},
			want: map[string]string{"--build-arg key1": "=\"value1=& abcd\"", "--build-arg key2": "=\"value2=&abcd\"", "--key3": "=\"value3=& abcd\"", "--key4": "=\"value4=& abcd\""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDockerBuildFlagsMap(tt.args.dockerBuildConfig); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getDockerBuildFlagsMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
