package main

import (
	"github.com/otiai10/copy"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

var tmpArtifactLocation = "./job-artifact"

func UploadArtifact(artifactFiles map[string]string, s3Location string) error {
	//collect in a dir
	err := os.Mkdir(tmpArtifactLocation, os.ModeDir)
	if err != nil {
		return err
	}
	for key, val := range artifactFiles {
		loc := filepath.Join(tmpArtifactLocation, key)
		err := os.Mkdir(loc, os.ModeDir)
		if err != nil {
			return err
		}
		err = copy.Copy(val, loc)
		if err != nil {
			return err
		}
	}
	zipFile := "job-artifact.zip"
	zipCmd := exec.Command("zip", "-r", zipFile, tmpArtifactLocation)
	err = RunCommand(zipCmd)
	if err != nil {
		return err
	}
	log.Println(devtron, " artifact upload to ", zipFile, s3Location)
	artifactPush := exec.Command("aws", "s3", "cp", zipFile, s3Location)
	return RunCommand(artifactPush)
}
