package out

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/frodenas/gcs-resource"
)

type OutCommand struct {
	gcsClient gcsresource.GCSClient
}

func NewOutCommand(gcsClient gcsresource.GCSClient) *OutCommand {
	return &OutCommand{
		gcsClient: gcsClient,
	}
}

func (command *OutCommand) Run(sourceDir string, request OutRequest) (OutResponse, error) {
	if ok, message := request.Source.IsValid(); !ok {
		return OutResponse{}, errors.New(message)
	}

	if ok, message := request.Params.IsValid(); !ok {
		return OutResponse{}, errors.New(message)
	}

	localPath, err := command.localPath(request, sourceDir)
	if err != nil {
		return OutResponse{}, err
	}

	objectPath := command.objectPath(request, localPath)

	objectContentType := command.objectContentType(request)

	bucketName := request.Source.Bucket
	generation, err := command.gcsClient.UploadFile(bucketName, objectPath, objectContentType, localPath, request.Params.PredefinedACL)
	if err != nil {
		return OutResponse{}, err
	}

	var url string
	version := gcsresource.Version{}
	if request.Source.Regexp != "" {
		version.Path = objectPath
		url, _ = command.gcsClient.URL(bucketName, objectPath, 0)
	} else {
		version.Generation = fmt.Sprintf("%d", generation)
		url, _ = command.gcsClient.URL(bucketName, objectPath, generation)
	}

	return OutResponse{
		Version:  version,
		Metadata: command.metadata(objectPath, url),
	}, nil
}

func (command *OutCommand) localPath(request OutRequest, sourceDir string) (string, error) {
	pattern := request.Params.File
	matches, err := filepath.Glob(filepath.Join(sourceDir, pattern))
	if err != nil {
		return "", err
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no matches found for pattern: %s", pattern)
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("more than one match found for pattern: %s\n%v", pattern, matches)
	}

	return matches[0], nil
}

func (command *OutCommand) objectPath(request OutRequest, localPath string) string {
	if request.Source.Regexp != "" {
		return filepath.Join(parentDir(request.Source.Regexp), filepath.Base(localPath))
	} else {
		return request.Source.VersionedFile
	}
}

func (command *OutCommand) objectContentType(request OutRequest) string {
	return request.Params.ContentType
}

func parentDir(regexp string) string {
	return regexp[:strings.LastIndex(regexp, "/")+1]
}

func (command *OutCommand) metadata(objectPath string, url string) []gcsresource.MetadataPair {
	objectFilename := filepath.Base(objectPath)

	metadata := []gcsresource.MetadataPair{
		gcsresource.MetadataPair{
			Name:  "filename",
			Value: objectFilename,
		},
		gcsresource.MetadataPair{
			Name:  "url",
			Value: url,
		},
	}

	return metadata
}
