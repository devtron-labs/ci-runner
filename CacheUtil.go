package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"log"
	"os"
	"os/exec"
)

func GetCache(ciRequest *CiRequest) error {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(ciRequest.AwsRegion),
	}))
	file, err := os.Create("/" + ciRequest.CiCacheFileName)
	if err != nil {
		log.Fatal(err)
		return err
	}

	svc := s3.New(sess)
	input := &s3.ListObjectVersionsInput{
		Bucket: aws.String(ciRequest.CiCacheLocation),
		Prefix: aws.String(ciRequest.CiCacheFileName),
	}
	result, err := svc.ListObjectVersions(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return err
	}

	var version *string
	var size int64
	for _, v := range result.Versions {
		if *v.IsLatest && *v.Key == ciRequest.CiCacheFileName {
			version = v.VersionId
			fmt.Println("selected version", v.VersionId, "last modified", v.LastModified)
			size = *v.Size
			break
		}
	}

	downloader := s3manager.NewDownloader(sess)
	numBytes, err := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket:    aws.String(ciRequest.CiCacheLocation),
			Key:       aws.String(ciRequest.CiCacheFileName),
			VersionId: version,
		})
	if err != nil {
		log.Println("couldn't download cache file")
		return nil
	}
	fmt.Println("Downloaded", file.Name(), numBytes, "bytes")

	if numBytes != size {
		fmt.Println("cache sizes don't match. Skipping step", "version cache size", size, "downloaded size", numBytes)
		return nil
	}

	// Extract cache
	if err == nil {
		extractCmd := exec.Command("tar", "-xvf", ciRequest.CiCacheFileName)
		extractCmd.Dir = "/"
		err = extractCmd.Run()
		if err != nil {
			log.Fatal("Could not extract cache blob", err)
			return err
		}
	}
	return nil
}

func SyncCache(ciRequest *CiRequest) error {
	DeleteFile(ciRequest.CiCacheFileName)
	// Generate new cache
	log.Println("------> generating new cache")
	tarCmd := exec.Command("tar", "-cf", ciRequest.CiCacheFileName, "/var/lib/docker")
	tarCmd.Dir = "/"
	err := RunCommand(tarCmd)
	if err != nil {
		fmt.Println(err)
		return err
	}

	//aws s3 cp cache.tar.gz s3://ci-caching/
	log.Println("------> pushing new cache")
	cachePush := exec.Command("aws", "s3", "cp", ciRequest.CiCacheFileName, "s3://" + ciRequest.CiCacheLocation + "/" + ciRequest.CiCacheFileName)
	return RunCommand(cachePush)
}