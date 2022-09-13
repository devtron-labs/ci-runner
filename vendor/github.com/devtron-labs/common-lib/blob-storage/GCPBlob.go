package blob_storage

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"io"
	"os"
)

type GCPBlob struct {
}

func (impl *GCPBlob) UploadBlob(request *BlobStorageRequest) error {
	ctx := context.Background()
	file, err := os.Open(request.SourceKey)
	if err != nil {
		return err
	}
	defer file.Close()
	err, gcpObject := getGcpObject(request, ctx)
	if err != nil {
		return err
	}
	objectWriter := gcpObject.NewWriter(ctx)
	_, err = io.Copy(objectWriter, file)
	if err := objectWriter.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %v", err)
	}
	return err
}

func getGcpObject(request *BlobStorageRequest, ctx context.Context) (error, *storage.ObjectHandle) {
	config := request.GcpBlobBaseConfig
	storageClient, err := createGcpClient(ctx, request)
	if err != nil {
		return err, nil
	}
	objects := storageClient.Bucket(config.BucketName).Objects(ctx, &storage.Query{
		Versions: false,
		Prefix:   request.DestinationKey,
	})
	for {
		objectAttrs, err := objects.Next()
		if err == iterator.Done {
			break
		}
		fmt.Println(objectAttrs)
	}
	gcpObject := storageClient.Bucket(config.BucketName).Object(request.DestinationKey)
	return err, gcpObject
}

func createGcpClient(ctx context.Context, request *BlobStorageRequest) (*storage.Client, error) {
	config := request.GcpBlobBaseConfig
	fmt.Println("going to create gcp client")
	storageClient, err := storage.NewClient(ctx, option.WithCredentialsFile(config.CredentialFileJsonData))
	if err != nil {
		return nil, err
	}
	return storageClient, err
}

func (impl *GCPBlob) DownloadBlob(request *BlobStorageRequest, file *os.File) (bool, int64, error) {
	ctx := context.Background()
	err, gcpObject := getGcpObject(request, ctx)
	if err != nil {
		return false, 0, err
	}
	objectReader, err := gcpObject.NewReader(ctx)
	if err != nil {
		return false, 0, err
	}
	defer objectReader.Close()
	writtenBytes, err := io.Copy(file, objectReader)
	return err != nil, writtenBytes, err
}
