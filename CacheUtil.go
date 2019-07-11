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
	"path/filepath"
	"strings"
)

func GetCache(ciRequest *CiRequest) error {
	//ciCacheLocation := ciRequest.CiCacheLocation + ciRequest.CiCacheFileName

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(ciRequest.AwsRegion),
	}))
	file, err := os.Create("/" + ciRequest.CiCacheFileName)
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

	/*f, err := os.Open(ciRequest.CiCacheFileName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()*/

	gzf, err := gzip.NewReader(file)
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
	CreateTar(ciRequest.CiCacheFileName, "/var/lib/docker")
	//aws s3 cp cache.tar.gz s3://ci-caching/

	f, err := os.Open(ciRequest.CiCacheFileName)
	if err != nil {
		log.Fatal(err)
	}
	fi, _:=f.Stat()
	fmt.Printf("file size %s",fi)
	log.Println("------> pushing new cache")
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(ciRequest.AwsRegion),
	}))

	uploader := s3manager.NewUploader(sess)

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(ciRequest.CiCacheLocation),
		Key:    aws.String(ciRequest.CiCacheFileName),
		Body:   f,
	})
	if err != nil {
		// Print the error and exit.
		log.Println("file upload fail")
		return err
	} else {

		fmt.Printf("Successfully uploaded %q to %q\n", ciRequest.CiCacheLocation, ciRequest.CiCacheFileName,)

	}

	/*err = os.RemoveAll("/var/lib/docker/*")
	if err == nil {
		log.Println("removed /var/lib/docker")
	} else {
		log.Println("err", err)
	}*/
	return err
}

func CreateTar(destinationfile, sourcedir string)  {

	dir, err := os.Open(sourcedir)

	checkerror(err)

	fmt.Println(dir.Name())
	defer dir.Close()

	files, err := dir.Readdir(0) // grab the files list

	checkerror(err)

	tarfile, err := os.Create(destinationfile)
	defer  tarfile.Close()
	checkerror(err)

	var fileWriter io.WriteCloser = tarfile

	if strings.HasSuffix(destinationfile, ".gz") {
		fileWriter = gzip.NewWriter(tarfile) // add a gzip filter
		defer fileWriter.Close()             // if user add .gz in the destination filename
	}

	tarfileWriter := tar.NewWriter(fileWriter)
	defer tarfileWriter.Close()

	for _, fileInfo := range files {

		if fileInfo.IsDir() {
			continue
		}

		// see https://www.socketloop.com/tutorials/go-file-path-independent-of-operating-system

		file, err := os.Open(dir.Name() + string(filepath.Separator) + fileInfo.Name())

		checkerror(err)

		defer file.Close()

		// prepare the tar header

		header := new(tar.Header)
		header.Name = file.Name()
		header.Size = fileInfo.Size()
		header.Mode = int64(fileInfo.Mode())
		header.ModTime = fileInfo.ModTime()

		err = tarfileWriter.WriteHeader(header)

		checkerror(err)

		_, err = io.Copy(tarfileWriter, file)

		checkerror(err)
	}
}
