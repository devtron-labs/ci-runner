package main

import (
	"log"
	"os/exec"
)

func GetCache(ciRequest *CiRequest) {
	ciCacheLocation := ciRequest.CiCacheLocation + ciRequest.CiCacheFileName
	cmd := exec.Command("aws", "s3", "cp", ciCacheLocation, ".")
	err := RunCommand(cmd)

	// Extract cache
	if err == nil {
		extractCmd := exec.Command("tar", "-xvf", ciRequest.CiCacheFileName)
		extractCmd.Dir = "/"
		extractCmd.Run()
	}
}


func SyncCache(ciRequest *CiRequest) error {
	DeleteFile(ciRequest.CiCacheFileName)

	// Generate new cache
	log.Println("------> generating new cache")
	tarCmd := exec.Command("tar", "-cf", ciRequest.CiCacheFileName, "/var/lib/docker")
	tarCmd.Dir = "/"
	tarCmd.Run()

	//aws s3 cp cache.tar.gz s3://ci-caching/
	log.Println("------> pushing new cache")
	cachePush := exec.Command("aws", "s3", "cp", ciRequest.CiCacheFileName, ciRequest.CiCacheLocation+ciRequest.CiCacheFileName)
	return RunCommand(cachePush)
}