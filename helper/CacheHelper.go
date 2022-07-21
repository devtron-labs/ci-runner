/*
 *  Copyright 2020 Devtron Labs
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package helper

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/devtron-labs/ci-runner/util"
)

func DownLoadFromS3(file *os.File, ciRequest *CiRequest, sess *session.Session) (success bool, err error) {
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
		return false, err
	}

	var version *string
	var size int64
	for _, v := range result.Versions {
		if *v.IsLatest && *v.Key == ciRequest.CiCacheFileName {
			version = v.VersionId
			log.Println(util.DEVTRON, " selected version ", v.VersionId, " last modified ", v.LastModified)
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
		return false, nil
	}
	log.Println(util.DEVTRON, " downloaded ", file.Name(), numBytes, " bytes ")

	if numBytes != size {
		log.Println(util.DEVTRON, " cache sizes don't match, skipping step ", " version cache size ", size, " downloaded size ", numBytes)
		return false, nil
	}

	if numBytes >= ciRequest.CacheLimit {
		log.Println(util.DEVTRON, " cache upper limit exceeded, ignoring old cache")
		return false, nil
	}
	return true, nil
}

func GetCache(ciRequest *CiRequest) error {
	if ciRequest.InvalidateCache {
		log.Println("ignoring cache ... ")
		return nil
	}
	log.Println("setting build cache ...............")
	file, err := os.Create("/" + ciRequest.CiCacheFileName)
	defer file.Close()
	if err != nil {
		log.Fatal(err)
	}
	//----------download file
	downloadSuccess := false
	switch ciRequest.CloudProvider {
	case BLOB_STORAGE_S3:
		sess := session.Must(session.NewSession(&aws.Config{
			Region: aws.String(ciRequest.CiCacheRegion),
		}))
		downloadSuccess, err = DownLoadFromS3(file, ciRequest, sess)
	case BLOB_STORAGE_MINIO:
		sess := session.Must(session.NewSession(&aws.Config{
			Region:           aws.String("us-west-2"),
			Endpoint:         aws.String(ciRequest.MinioEndpoint),
			DisableSSL:       aws.Bool(true),
			S3ForcePathStyle: aws.Bool(true),
		}))
		downloadSuccess, err = DownLoadFromS3(file, ciRequest, sess)
	case BLOB_STORAGE_AZURE:
		b := AzureBlob{}
		downloadSuccess, err = b.DownloadBlob(context.Background(), ciRequest.CiCacheFileName, ciRequest.AzureBlobConfig, file)
	default:
		return fmt.Errorf("cloudprovider %s not supported", ciRequest.CloudProvider)
	}
	///---------download file end
	// Extract cache
	if err == nil && downloadSuccess {
		extractCmd := exec.Command("tar", "-xvzf", ciRequest.CiCacheFileName)
		extractCmd.Dir = "/"
		err = extractCmd.Run()
		if err != nil {
			log.Fatal(" Could not extract cache blob ", err)
		}
	} else if err != nil {
		log.Println(util.DEVTRON, "build cache  error", err.Error())
	}
	return nil
}

func SyncCache(ciRequest *CiRequest) error {
	err := os.Chdir("/")
	if err != nil {
		log.Println(err)
		return err
	}
	util.DeleteFile(ciRequest.CiCacheFileName)
	// Generate new cache
	log.Println("Generating new cache")
	var cachePath string
	if ciRequest.DockerBuildTargetPlatform != "" {
		cachePath = util.LOCAL_BUILDX_CACHE_LOCATION
	} else {
		cachePath = "/var/lib/docker"
	}

	tarCmd := exec.Command("tar", "-cvzf", ciRequest.CiCacheFileName, cachePath)
	tarCmd.Dir = "/"
	err = tarCmd.Run()
	if err != nil {
		log.Fatal("Could not compress cache", err)
	}

	//aws s3 cp cache.tar.gz s3://ci-caching/
	//----------upload file
	log.Println(util.DEVTRON, " -----> pushing new cache")
	switch ciRequest.CloudProvider {
	case BLOB_STORAGE_S3:
		cachePush := exec.Command("aws", "s3", "cp", ciRequest.CiCacheFileName, "s3://"+ciRequest.CiCacheLocation+"/"+ciRequest.CiCacheFileName)
		err = util.RunCommand(cachePush)
	case BLOB_STORAGE_MINIO:
		cachePush := exec.Command("aws", "--endpoint-url", ciRequest.MinioEndpoint, "s3", "cp", ciRequest.CiCacheFileName, "s3://"+ciRequest.CiCacheLocation+"/"+ciRequest.CiCacheFileName)
		err = util.RunCommand(cachePush)
	case BLOB_STORAGE_AZURE:
		b := AzureBlob{}
		err = b.UploadBlob(context.Background(), ciRequest.CiCacheFileName, ciRequest.AzureBlobConfig, ciRequest.CiCacheFileName, ciRequest.AzureBlobConfig.BlobContainerCiCache)
	default:
		return fmt.Errorf("cloudprovider %s not supported", ciRequest.CloudProvider)
	}
	///---------upload file end
	if err != nil {
		log.Println(util.DEVTRON, " -----> push err", err)
	}
	return err
}

//--------------------
type AzureBlob struct {
}

func (impl *AzureBlob) getSharedCredentials(accountName, accountKey string) (*azblob.SharedKeyCredential, error) {
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		log.Println(util.DEVTRON, "Invalid credentials with error: "+err.Error())
	}
	return credential, err
}

func (impl *AzureBlob) getTokenCredentials() (azblob.TokenCredential, error) {
	msiEndpoint, err := adal.GetMSIEndpoint()
	if err != nil {
		return nil, fmt.Errorf("failed to get the managed service identity endpoint: %v", err)
	}

	token, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, azure.PublicCloud.ResourceIdentifiers.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to create the managed service identity token: %v", err)
	}
	err = token.Refresh()
	if err != nil {
		return nil, fmt.Errorf("failure refreshing token from MSI endpoint %w", err)
	}

	credential := azblob.NewTokenCredential(token.Token().AccessToken, impl.defaultTokenRefreshFunction(token))
	return credential, err
}

func (impl *AzureBlob) buildContainerUrl(config *AzureBlobConfig, container string) (*azblob.ContainerURL, error) {
	var credential azblob.Credential
	var err error
	if len(config.AccountKey) > 0 {
		credential, err = impl.getSharedCredentials(config.AccountName, config.AccountKey)
		if err != nil {
			return nil, fmt.Errorf("failed in getting credentials: %v", err)
		}
	} else {
		credential, err = impl.getTokenCredentials()
		if err != nil {
			return nil, fmt.Errorf("failed in getting credentials: %v", err)
		}
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})

	// From the Azure portal, get your storage account blob service URL endpoint.
	URL, _ := url.Parse(
		fmt.Sprintf("https://%s.blob.core.windows.net/%s", config.AccountName, container))

	// Create a ContainerURL object that wraps the container URL and a request
	// pipeline to make requests.
	containerURL := azblob.NewContainerURL(*URL, p)
	return &containerURL, nil
}

func (impl *AzureBlob) DownloadBlob(context context.Context, blobName string, config *AzureBlobConfig, file *os.File) (success bool, err error) {
	containerURL, err := impl.buildContainerUrl(config, config.BlobContainerCiCache)
	if err != nil {
		return false, err
	}
	res, err := containerURL.ListBlobsFlatSegment(context, azblob.Marker{}, azblob.ListBlobsSegmentOptions{
		Details: azblob.BlobListingDetails{
			Versions: false,
		},
		Prefix: blobName,
	})
	if err != nil {
		return false, err
	}
	var latestVersion string
	for _, s := range res.Segment.BlobItems {
		if *s.IsCurrentVersion {
			latestVersion = *s.VersionID
			break
		}
	}
	log.Println(util.DEVTRON, " latest version", latestVersion)
	blobURL := containerURL.NewBlobURL(blobName).WithVersionID(latestVersion)
	err = azblob.DownloadBlobToFile(context, blobURL, 0, azblob.CountToEnd, file, azblob.DownloadFromBlobOptions{})
	return true, err
}

func (impl *AzureBlob) UploadBlob(context context.Context, blobName string, config *AzureBlobConfig, inputFileName string, container string) error {
	containerURL, err := impl.buildContainerUrl(config, container)
	if err != nil {
		return err
	}
	blobURL := containerURL.NewBlockBlobURL(blobName)
	log.Println(util.DEVTRON, "upload blob url ", blobURL, "file", inputFileName)

	file, err := os.Open(inputFileName)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = azblob.UploadFileToBlockBlob(context, file, blobURL, azblob.UploadToBlockBlobOptions{})
	return err
}

func (impl *AzureBlob) defaultTokenRefreshFunction(spToken *adal.ServicePrincipalToken) func(credential azblob.TokenCredential) time.Duration {
	return func(credential azblob.TokenCredential) time.Duration {
		err := spToken.Refresh()
		if err != nil {
			return 0
		}
		expiresIn, err := strconv.ParseInt(string(spToken.Token().ExpiresIn), 10, 64)
		if err != nil {
			return 0
		}
		credential.SetToken(spToken.Token().AccessToken)
		return time.Duration(expiresIn-300) * time.Second
	}
}
