package gcsresource

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"fmt"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"gopkg.in/cheggaaa/pb.v1"
	"io"
	"os"
)

//go:generate counterfeiter -o fakes/fake_gcsclient.go . GCSClient
type GCSClient interface {
	BucketObjects(bucketName string, prefix string) ([]string, error)
	ObjectGenerations(bucketName string, objectPath string) ([]int64, error)
	DownloadFile(bucketName string, objectPath string, generation int64, localPath string) error
	UploadFile(bucketName string, objectPath string, objectContentType string, localPath string, predefinedACL string, cacheControl string) (int64, error)
	URL(bucketName string, objectPath string, generation int64) (string, error)
	DeleteObject(bucketName string, objectPath string, generation int64) error
	GetBucketObjectInfo(bucketName, objectPath string) (*storage.ObjectAttrs, error)
}

type gcsclient struct {
	storageService *storage.Client
	progressOutput io.Writer
}

func NewGCSClient(
	progressOutput io.Writer,
	jsonKey string,
) (GCSClient, error) {
	var err error
	var userAgent = "gcs-resource/0.0.1"

	var storageService *storage.Client
	if jsonKey == "" {
		ctx := context.Background()
		storageService, err = storage.NewClient(ctx, option.WithUserAgent(userAgent))
		if err != nil {
			return &gcsclient{}, err
		}
	} else {
		ctx := context.Background()
		storageService, err = storage.NewClient(ctx, option.WithUserAgent(userAgent), option.WithCredentialsJSON([]byte(jsonKey)))
		if err != nil {
			return &gcsclient{}, err
		}
	}

	return &gcsclient{
		storageService: storageService,
		progressOutput: progressOutput,
	}, nil
}

func (gcsclient *gcsclient) BucketObjects(bucketName string, prefix string) ([]string, error) {
	bucketObjects, err := gcsclient.getBucketObjects(bucketName, prefix)
	if err != nil {
		return []string{}, err
	}

	return bucketObjects, nil
}

func (gcsclient *gcsclient) ObjectGenerations(bucketName string, objectPath string) ([]int64, error) {
	isBucketVersioned, err := gcsclient.getBucketVersioning(bucketName)
	if err != nil {
		return []int64{}, err
	}

	if !isBucketVersioned {
		return []int64{}, errors.New("bucket is not versioned")
	}

	objectGenerations, err := gcsclient.getObjectGenerations(bucketName, objectPath)
	if err != nil {
		return []int64{}, err
	}

	return objectGenerations, nil
}

func (gcsclient *gcsclient) DownloadFile(bucketName string, objectPath string, generation int64, localPath string) error {
	isBucketVersioned, err := gcsclient.getBucketVersioning(bucketName)
	if err != nil {
		return err
	}

	if !isBucketVersioned && generation != 0 {
		return errors.New("bucket is not versioned")
	}

	attrs, err := gcsclient.GetBucketObjectInfo(bucketName, objectPath)
	if err != nil {
		return err
	}

	progress := gcsclient.newProgressBar(int64(attrs.Size))
	defer progress.Finish()
	progress.Start()

	localFile, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer localFile.Close()
	ctx := context.Background()
	objectHandle := gcsclient.storageService.Bucket(bucketName).Object(objectPath)
	if generation != 0 {
		objectHandle = objectHandle.Generation(generation)
	}
	rc, err := objectHandle.NewReader(ctx)
	if err != nil {
		return err
	}
	defer rc.Close()

	_, err = io.Copy(localFile, rc)
	if err != nil {
		return err
	}

	return nil
}

func (gcsclient *gcsclient) UploadFile(bucketName string, objectPath string, objectContentType string, localPath string, predefinedACL string, cacheControl string) (int64, error) {
	isBucketVersioned, err := gcsclient.getBucketVersioning(bucketName)
	if err != nil {
		return 0, err
	}

	stat, err := os.Stat(localPath)
	if err != nil {
		return 0, err
	}

	localFile, err := os.Open(localPath)
	if err != nil {
		return 0, err
	}
	defer localFile.Close()

	progress := gcsclient.newProgressBar(stat.Size())
	progress.Start()
	defer progress.Finish()

	ctx := context.Background()
	wc := gcsclient.storageService.Bucket(bucketName).Object(objectPath).NewWriter(ctx)
	if _, err = io.Copy(wc, localFile); err != nil {
		return 0, err
	}

	if err := wc.Close(); err != nil {
		return 0, err
	}

	if predefinedACL != "" || cacheControl != "" || objectContentType != "" {
		var cacheControlOption interface{}
		var contentTypeOption interface{}

		if cacheControl != "" {
			cacheControlOption = cacheControl
		}

		if objectContentType != "" {
			contentTypeOption = objectContentType
		}

		attrs := storage.ObjectAttrsToUpdate{
			ContentType:   contentTypeOption,
			CacheControl:  cacheControlOption,
			PredefinedACL: predefinedACL,
		}
		ctx = context.Background()
		_, err = gcsclient.storageService.Bucket(bucketName).Object(objectPath).Update(ctx, attrs)
		if err != nil {
			return 0, nil
		}
	}

	if isBucketVersioned {
		attrs, err := gcsclient.GetBucketObjectInfo(bucketName, objectPath)
		if err != nil {
			return 0, err
		}
		return attrs.Generation, nil
	}
	return 0, nil
}

func (gcsclient *gcsclient) URL(bucketName string, objectPath string, generation int64) (string, error) {
	ctx := context.Background()
	objectHandle := gcsclient.storageService.Bucket(bucketName).Object(objectPath)
	if generation != 0 {
		objectHandle = objectHandle.Generation(generation)
	}
	attrs, err := objectHandle.Attrs(ctx)
	if err != nil {
		return "", err
	}

	var url string
	if generation != 0 {
		url = fmt.Sprintf("gs://%s/%s#%d", bucketName, objectPath, attrs.Generation)
	} else {
		url = fmt.Sprintf("gs://%s/%s", bucketName, objectPath)
	}

	return url, nil
}

func (gcsclient *gcsclient) DeleteObject(bucketName string, objectPath string, generation int64) error {
	var err error
	ctx := context.Background()
	if generation != 0 {
		err = gcsclient.storageService.Bucket(bucketName).Object(objectPath).Generation(generation).Delete(ctx)
	} else {
		err = gcsclient.storageService.Bucket(bucketName).Object(objectPath).Delete(ctx)
	}
	if err != nil {
		return err
	}
	return nil
}

func (gcsclient *gcsclient) GetBucketObjectInfo(bucketName, objectPath string) (*storage.ObjectAttrs, error) {
	ctx := context.Background()
	attrs, err := gcsclient.storageService.Bucket(bucketName).Object(objectPath).Attrs(ctx)

	if err != nil {
		return nil, err
	}
	return attrs, nil
}

func (gcsclient *gcsclient) getBucketObjects(bucketName string, prefix string) ([]string, error) {
	var bucketObjects []string
	ctx := context.Background()
	pageToken := ""
	query := &storage.Query{
		Delimiter: pageToken,
		Prefix:    prefix,
		Versions:  false,
	}
	objectIterator := gcsclient.storageService.Bucket(bucketName).Objects(ctx, query)
	for {
		object, err := objectIterator.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		bucketObjects = append(bucketObjects, object.Name)
	}

	return bucketObjects, nil
}

func (gcsclient *gcsclient) getBucketVersioning(bucketName string) (bool, error) {
	ctx := context.Background()
	bucket, err := gcsclient.storageService.Bucket(bucketName).Attrs(ctx)
	if err != nil {
		return false, err
	}

	return bucket.VersioningEnabled, nil
}

func (gcsclient *gcsclient) getObjectGenerations(bucketName string, objectPath string) ([]int64, error) {
	var objectGenerations []int64
	ctx := context.Background()
	pageToken := ""
	query := &storage.Query{
		Delimiter: pageToken,
		Prefix:    objectPath,
		Versions:  true,
	}
	objectIterator := gcsclient.storageService.Bucket(bucketName).Objects(ctx, query)
	for {
		object, err := objectIterator.Next()

		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if object.Name == objectPath {
			objectGenerations = append(objectGenerations, object.Generation)
		}
		objectIterator.PageInfo()
	}

	return objectGenerations, nil
}

func (gcsclient *gcsclient) newProgressBar(total int64) *pb.ProgressBar {
	progress := pb.New64(total)

	progress.Output = gcsclient.progressOutput
	progress.ShowSpeed = true
	progress.Units = pb.U_BYTES
	progress.NotPrint = true

	return progress.SetWidth(80)
}
