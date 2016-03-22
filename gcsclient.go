package gcsresource

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/cheggaaa/pb"
	"golang.org/x/oauth2"
	oauthgoogle "golang.org/x/oauth2/google"
	"google.golang.org/api/storage/v1"
)

type GCSClient interface {
	BucketObjects(bucketName string, prefix string) ([]string, error)
	ObjectGenerations(bucketName string, objectPath string) ([]int64, error)
	DownloadFile(bucketName string, objectPath string, generation int64, localPath string) error
	UploadFile(bucketName string, objectPath string, localPath string) (int64, error)
	URL(bucketName string, objectPath string, generation int64) (string, error)
}

type gcsclient struct {
	client         *storage.Service
	progressOutput io.Writer
}

func NewGCSClient(
	progressOutput io.Writer,
	project string,
	jsonKey string,
) (GCSClient, error) {
	var err error
	var storageClient *http.Client
	var userAgent = "gcs-resource/0.0.1"

	if jsonKey != "" {
		storageJwtConf, err := oauthgoogle.JWTConfigFromJSON([]byte(jsonKey), storage.DevstorageFullControlScope)
		if err != nil {
			return &gcsclient{}, err
		}
		storageClient = storageJwtConf.Client(oauth2.NoContext)
	} else {
		storageClient, err = oauthgoogle.DefaultClient(oauth2.NoContext, storage.DevstorageFullControlScope)
		if err != nil {
			return &gcsclient{}, err
		}
	}

	storageService, err := storage.New(storageClient)
	if err != nil {
		return &gcsclient{}, err
	}
	storageService.UserAgent = userAgent

	return &gcsclient{
		client:         storageService,
		progressOutput: progressOutput,
	}, nil
}

func (client *gcsclient) BucketObjects(bucketName string, prefix string) ([]string, error) {
	bucketObjects, err := client.getBucketObjects(bucketName, prefix)
	if err != nil {
		return []string{}, err
	}

	return bucketObjects, nil
}

func (client *gcsclient) ObjectGenerations(bucketName string, objectPath string) ([]int64, error) {
	isBucketVersioned, err := client.getBucketVersioning(bucketName)
	if err != nil {
		return []int64{}, err
	}

	if !isBucketVersioned {
		return []int64{}, errors.New("bucket is not versioned")
	}

	objectGenerations, err := client.getObjectGenerations(bucketName, objectPath)
	if err != nil {
		return []int64{}, err
	}

	return objectGenerations, nil
}

func (client *gcsclient) DownloadFile(bucketName string, objectPath string, generation int64, localPath string) error {
	getCall := client.client.Objects.Get(bucketName, objectPath)
	if generation != 0 {
		getCall = getCall.Generation(generation)
	}

	object, err := getCall.Do()
	if err != nil {
		return err
	}

	localFile, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer localFile.Close()

	progress := client.newProgressBar(int64(object.Size))
	progress.Start()
	defer progress.Finish()

	// TODO
	_, err = getCall.Download()
	if err != nil {
		return err
	}

	return nil
}

func (client *gcsclient) UploadFile(bucketName string, objectPath string, localPath string) (int64, error) {
	stat, err := os.Stat(localPath)
	if err != nil {
		return 0, err
	}

	localFile, err := os.Open(localPath)
	if err != nil {
		return 0, err
	}
	defer localFile.Close()

	progress := client.newProgressBar(stat.Size())
	progress.Start()
	defer progress.Finish()

	object := &storage.Object{
		Name: objectPath,
	}

	uploadedObject, err := client.client.Objects.Insert(bucketName, object).Media(progress.NewProxyReader(localFile)).Do()
	if err != nil {
		return 0, err
	}

	return uploadedObject.Generation, nil
}

func (client *gcsclient) URL(bucketName string, objectPath string, generation int64) (string, error) {
	getCall := client.client.Objects.Get(bucketName, objectPath)
	if generation != 0 {
		getCall = getCall.Generation(generation)
	}

	object, err := getCall.Do()
	if err != nil {
		return "", err
	}

	var url string
	if object.MediaLink != "" {
		url = object.MediaLink
	} else {
		if generation != 0 {
			url = fmt.Sprintf("gs://%s/%s#%d", bucketName, objectPath, generation)
		} else {
			url = fmt.Sprintf("gs://%s/%s", bucketName, objectPath)
		}
	}

	return url, nil
}

func (client *gcsclient) getBucketObjects(bucketName string, prefix string) ([]string, error) {
	var bucketObjects []string

	pageToken := ""
	for {
		listCall := client.client.Objects.List(bucketName)
		listCall = listCall.PageToken(pageToken)
		listCall = listCall.Prefix(prefix)
		listCall = listCall.Versions(false)

		objects, err := listCall.Do()
		if err != nil {
			return bucketObjects, err
		}

		for _, object := range objects.Items {
			bucketObjects = append(bucketObjects, object.Name)
		}

		if objects.NextPageToken != "" {
			pageToken = objects.NextPageToken
		} else {
			break
		}
	}

	return bucketObjects, nil
}

func (client *gcsclient) getBucketVersioning(bucketName string) (bool, error) {
	bucket, err := client.client.Buckets.Get(bucketName).Do()
	if err != nil {
		return false, err
	}

	return bucket.Versioning.Enabled, nil
}

func (client *gcsclient) getObjectGenerations(bucketName string, objectPath string) ([]int64, error) {
	var objectGenerations []int64

	pageToken := ""
	for {
		listCall := client.client.Objects.List(bucketName)
		listCall = listCall.PageToken(pageToken)
		listCall = listCall.Prefix(objectPath)
		listCall = listCall.Versions(true)

		objects, err := listCall.Do()
		if err != nil {
			return objectGenerations, err
		}

		for _, object := range objects.Items {
			if object.Name == objectPath {
				objectGenerations = append(objectGenerations, object.Generation)
			}
		}

		if objects.NextPageToken != "" {
			pageToken = objects.NextPageToken
		} else {
			break
		}
	}

	return objectGenerations, nil
}

func (client *gcsclient) newProgressBar(total int64) *pb.ProgressBar {
	progress := pb.New64(total)

	progress.Output = client.progressOutput
	progress.ShowSpeed = true
	progress.Units = pb.U_BYTES
	progress.NotPrint = true

	return progress.SetWidth(80)
}
