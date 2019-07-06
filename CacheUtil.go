package main

import (
	"log"
	"os/exec"
)

func GetCache(ciRequest *CiRequest) {
	ciCacheLocation := ciRequest.CiCacheLocation + ciRequest.CiCacheFileName
	cmd := exec.Command("aws", "s3", "cp", ciCacheLocation, ".")
	log.Println("Downloading pipeline cache")
	err := cmd.Run()
	if err != nil {
		log.Println("Could not get cache", err)
	} else {
		log.Println("Downloaded cache")
	}

	// Extract cache
	if err == nil {
		extractCmd := exec.Command("tar", "-xvf", ciRequest.CiCacheFileName)
		extractCmd.Dir = "/"
		err = extractCmd.Run()
		if err != nil {
			log.Println("Could not extract cache blob", err)
		}
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
	err := cachePush.Run()
	if err != nil {
		log.Println("Could not push new cache", err)
	} else {
		log.Println("Pushed cache")
	}
	return err
}