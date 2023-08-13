package router

import (
	"fmt"
	"github.com/devtron-labs/ci-runner/helper"
	"github.com/devtron-labs/ci-runner/resthandlers"
	"github.com/devtron-labs/ci-runner/util"
	"github.com/gorilla/mux"
	"net/http"
)

func InitRouter(ciCdRequest *helper.CiCdTriggerEvent) {
	router := mux.NewRouter()
	registerRoutes(ciCdRequest, router)
	http.Handle("/", router)
	err := http.ListenAndServe(":8080", router)
	if err != nil {
		fmt.Println(util.DEVTRON, "error starting server...")
		return
	}
}

func registerRoutes(ciCdRequest *helper.CiCdTriggerEvent, r *mux.Router) {
	buildxCleanerHandler := resthandlers.NewBuildxCleanerHandler(ciCdRequest.CiRequest)
	//this is get method, since it is used in prestop lifecycle hook in the container and the lifecycle allows only GET api
	r.HandleFunc("/cleanK8sDriver", buildxCleanerHandler.CleanBuildxK8sDriver).Methods("GET")
}
