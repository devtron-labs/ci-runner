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

package main

import (
	"context"
	"fmt"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func DownLoadFromS3(file *os.File, ciRequest *CiRequest) error {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(ciRequest.CiCacheRegion),
	}))

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
	return nil
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
	switch ciRequest.CloudProvider {
	case CLOUD_PROVIDER_AWS:
		err = DownLoadFromS3(file, ciRequest)
	case CLOUD_PROVIDER_AZURE:
		b := AzureBlob{}
		err = b.DownloadBlob(context.Background(), ciRequest.CiCacheFileName, ciRequest.AzureBlobConfig, file)
	default:
		return fmt.Errorf("cloudprovider %s not supported", ciRequest.CloudProvider)
	}
	///---------download file end
	// Extract cache
	if err == nil {
		extractCmd := exec.Command("tar", "-xvzf", ciRequest.CiCacheFileName)
		extractCmd.Dir = "/"
		err = extractCmd.Run()
		if err != nil {
			log.Fatal(" Could not extract cache blob ", err)
		}
	} else {
		log.Println(devtron, "build cache  error", err)
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
	//----------upload file
	log.Println(devtron, " -----> pushing new cache")
	switch ciRequest.CloudProvider {
	case CLOUD_PROVIDER_AWS:
		cachePush := exec.Command("aws", "s3", "cp", ciRequest.CiCacheFileName, "s3://"+ciRequest.CiCacheLocation+"/"+ciRequest.CiCacheFileName)
		err = RunCommand(cachePush)

	case CLOUD_PROVIDER_AZURE:
		b := AzureBlob{}
		err = b.UploadBlob(context.Background(), ciRequest.CiCacheFileName, ciRequest.AzureBlobConfig, ciRequest.CiCacheFileName)
	default:
		return fmt.Errorf("cloudprovider %s not supported", ciRequest.CloudProvider)
	}
	///---------upload file end
	if err != nil {
		log.Println(devtron, " -----> push err", err)
	}
	return err
}

//--------------------
type AzureBlob struct {
}

func (impl *AzureBlob) getSharedCredentials(accountName, accountKey string) (*azblob.SharedKeyCredential, error) {
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		log.Fatal("Invalid credentials with error: " + err.Error())
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

func (impl *AzureBlob) buildContainerUrl(config *AzureBlobConfig) (*azblob.ContainerURL, error) {
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
		fmt.Sprintf("https://%s.blob.core.windows.net/%s", config.AccountName, config.BlobContainer))

	// Create a ContainerURL object that wraps the container URL and a request
	// pipeline to make requests.
	containerURL := azblob.NewContainerURL(*URL, p)
	return &containerURL, nil
}

func (impl *AzureBlob) DownloadBlob(context context.Context, blobName string, config *AzureBlobConfig, file *os.File) error {
	containerURL, err := impl.buildContainerUrl(config)
	if err != nil {
		return err
	}
	res, err := containerURL.ListBlobsFlatSegment(context, azblob.Marker{}, azblob.ListBlobsSegmentOptions{
		Details: azblob.BlobListingDetails{
			Versions: false,
		},
		Prefix: blobName,
	})
	if err != nil {
		return err
	}
	var latestVersion string
	for _, s := range res.Segment.BlobItems {
		if *s.IsCurrentVersion {
			latestVersion = *s.VersionID
			break
		}
	}
	fmt.Println("latest version" + latestVersion)
	blobURL := containerURL.NewBlobURL(blobName).WithVersionID(latestVersion)
	err = azblob.DownloadBlobToFile(context, blobURL, 0, azblob.CountToEnd, file, azblob.DownloadFromBlobOptions{})
	return err
}

func (impl *AzureBlob) UploadBlob(context context.Context, blobName string, config *AzureBlobConfig, cacheFileName string) error {
	containerURL, err := impl.buildContainerUrl(config)
	if err != nil {
		return err
	}
	blobURL := containerURL.NewBlockBlobURL(blobName)
	file, err := os.Open(cacheFileName)
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
