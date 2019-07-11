package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
)

func GetCache(ciRequest *CiRequest) error {
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
	/*if err == nil {
		extractCmd := exec.Command("tar", "-xvf", ciRequest.CiCacheFileName)
		extractCmd.Dir = "/"
		err = extractCmd.Run()
		if err != nil {
			log.Println("Could not extract cache blob", err)
			log.Fatal(err)
			return err
		}
	}*/

	f, err := os.Open(ciRequest.CiCacheFileName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	tr := tar.NewReader(gzf)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Contents of %s:\n", hdr.Name)
		if _, err := io.Copy(os.Stdout, tr); err != nil {
			log.Fatal(err)
		}
		fmt.Println()
	}
	return nil
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
		return err
	} else {
		log.Println("Pushed cache")
	}

	err = os.RemoveAll("/var/lib/docker/*")
	if err == nil {
		log.Println("removed /var/lib/docker")
	} else {
		log.Println("err", err)
	}
	return err
}