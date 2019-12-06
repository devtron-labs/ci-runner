package main

import (
	"os"
	"path/filepath"
)

var tmpArtifactLocation = "./job-artifact"

func UploadArtifact(artifactFiles map[string]string, s3Location string) error {
	//collect in a dir
	err := os.Mkdir(tmpArtifactLocation, os.ModeDir)
	if err!=nil{
		return err
	}
	for key, val:=range artifactFiles{
		err:=os.Mkdir(filepath.Join(tmpArtifactLocation, key), os.ModeDir)
		if err!=nil{
			return err
		}
		os.
	}
	//zip
	//push
	return nil
}
