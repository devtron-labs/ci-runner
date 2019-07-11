package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"log"
	"os"
	"os/exec"
)

func GetCache(ciRequest *CiRequest) error {
	//ciCacheLocation := ciRequest.CiCacheLocation + ciRequest.CiCacheFileName

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(ciRequest.AwsRegion),
	}))

	file, err := os.Create("/"+ciRequest.CiCacheFileName)

	if err != nil {
		log.Fatal(err)
		return err
	}
	defer file.Close()

	downloader := s3manager.NewDownloader(sess)

	numBytes, err := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(ciRequest.CiCacheLocation),
			Key:    aws.String(ciRequest.CiCacheFileName),
		})
	if err != nil {
		log.Println("couldn't download cache file")
		return nil
	}
	fmt.Println("Downloaded", file.Name(), numBytes, "bytes")


	/*po, err := svc.PutObjectWithContext(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(ciRequest.CiCacheLocation),
		Key:    aws.String(ciRequest.CiCacheFileName),
		Body:   os.Stdin,
	})

	cmd := exec.Command("aws", "s3", "cp", ciCacheLocation, ".")
	log.Println("Downloading pipeline cache")
	err := cmd.Run()
	if err != nil {
		log.Println("Could not get cache", err)
	} else {
		log.Println("Downloaded cache")
	}*/

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
