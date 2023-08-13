package resthandlers

import (
	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/util"
	"log"
	"net/http"
)

type BuildxCleanerHandlerImpl struct {
	ciRequest *helper.CiRequest
}

type BuildxCleanerHandler interface {
	CleanBuildxK8sDriver(w http.ResponseWriter, r *http.Request)
}

func NewBuildxCleanerHandler(ciRequest *helper.CiRequest) BuildxCleanerHandler {
	return BuildxCleanerHandlerImpl{
		ciRequest: ciRequest,
	}
}

func (impl BuildxCleanerHandlerImpl) CleanBuildxK8sDriver(w http.ResponseWriter, r *http.Request) {
	if !helper.ValidBuildxK8sDriverOptions(impl.ciRequest) {
		w.WriteHeader(http.StatusOK)
		return
	}

	err := helper.CleanBuildxK8sDriver(impl.ciRequest.CiBuildConfig.DockerBuildConfig.BuildxK8sDriverOptions)
	if err != nil {
		log.Println(util.DEVTRON, "error occurred in performing buildx driver cleanup")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
