package helper

import (
	"encoding/json"
	"github.com/devtron-labs/ci-runner/util"
	"io/ioutil"
	"log"
)

func ExtractPluginArtifacts() (*PluginArtifacts, error) {
	exists, err := util.CheckFileExists(util.PluginArtifactsResults)
	if err != nil || !exists {
		log.Println("err", err)
		return nil, err
	}
	file, err := ioutil.ReadFile(util.PluginArtifactsResults)
	if err != nil {
		log.Println("error in reading file", "err", err.Error())
		return nil, err
	}
	pluginArtifacts := &PluginArtifacts{}
	err = json.Unmarshal(file, &pluginArtifacts)
	if err != nil {
		log.Println("error in unmarshalling imageDetailsFromCr results", "err", err.Error())
		return nil, err
	}
	return pluginArtifacts, nil

}
