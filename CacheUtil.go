package main

import (
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
		Region: aws.String(ciRequest.CiCacheRegion),
	}))
	file, err := os.Create("/" + ciRequest.CiCacheFileName)
	if err != nil {
		log.Fatal(err)
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
				log.Println(aerr.Error())
			}
		} else {
			log.Println(err.Error())
		}
		return err
	}

	var version *string
	var size int64
	for _, v := range result.Versions {
		if *v.IsLatest && *v.Key == ciRequest.CiCacheFileName {
			version = v.VersionId
			log.Println(devtron, " selected version ", v.VersionId, " last modified ", v.LastModified)
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
		log.Println("Couldn't download cache file")
		return nil
	}
	log.Println(devtron, " downloaded ", file.Name(), numBytes, " bytes ")

	if numBytes != size {
		log.Println(devtron, " cache sizes don't match, skipping step ", " version cache size ", size, " downloaded size ", numBytes)
		return nil
	}

	if numBytes >= ciRequest.CacheLimit {
		log.Println(devtron, " cache upper limit exceeded, ignoring old cache")
		return nil
	}

	// Extract cache
	if err == nil {
		extractCmd := exec.Command("tar", "-xvzf", ciRequest.CiCacheFileName)
		extractCmd.Dir = "/"
		err = extractCmd.Run()
		if err != nil {
			log.Fatal(" Could not extract cache blob ", err)
		}
	}
	return nil
}

func SyncCache(ciRequest *CiRequest) error {
	err := os.Chdir("/")
	if err != nil {
		log.Println(err)
		return err
	}
	DeleteFile(ciRequest.CiCacheFileName)
	// Generate new cache
	log.Println("Generating new cache")
	tarCmd := exec.Command("tar", "-cvzf", ciRequest.CiCacheFileName, "/var/lib/docker")
	tarCmd.Dir = "/"
	err = tarCmd.Run()
	if err != nil {
		log.Fatal("Could not compress cache", err)
	}

	//aws s3 cp cache.tar.gz s3://ci-caching/
	log.Println(devtron, " -----> pushing new cache")
	cachePush := exec.Command("aws", "s3", "cp", ciRequest.CiCacheFileName, "s3://"+ciRequest.CiCacheLocation+"/"+ciRequest.CiCacheFileName)
	return RunCommand(cachePush)
}