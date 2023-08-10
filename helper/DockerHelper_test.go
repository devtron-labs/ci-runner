package helper

import (
	"fmt"
	"os/exec"
	"testing"
)

func TestCreateBuildXK8sDriver(t *testing.T) {
	buildxOpts := make([]map[string]string, 0)
	buildxOpts = append(buildxOpts, map[string]string{"node": "builder-amd64", "driverOptions": "namespace=devtron-ci,nodeselector=kubernetes.io/arch:amd64"})
	buildxOpts = append(buildxOpts, map[string]string{"node": "builder-amd64-test", "driverOptions": "namespace=devtron-ci,nodeselector=kubernetes.io/arch:amd64"})
	err := CreateBuildXK8sDriver(buildxOpts)
	t.Cleanup(func() {
		buildxDelete := fmt.Sprintf("docker buildx rm %s", BUILDX_K8S_DRIVER_NAME)
		builderRemoveCmd := exec.Command("/bin/sh", "-c", buildxDelete)
		builderRemoveCmd.Run()
	})
	if err != nil {
		fmt.Println(err.Error())
		t.Fail()
	}
}
