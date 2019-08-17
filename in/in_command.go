package in

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/frodenas/gcs-resource"
	"github.com/frodenas/gcs-resource/versions"
)

type InCommand struct {
	gcsClient gcsresource.GCSClient
}

func NewInCommand(gcsClient gcsresource.GCSClient) *InCommand {
	return &InCommand{
		gcsClient: gcsClient,
	}
}

func (command *InCommand) Run(destinationDir string, request InRequest) (InResponse, error) {
	if ok, message := request.Source.IsValid(); !ok {
		return InResponse{}, errors.New(message)
	}

	err := command.createDirectory(destinationDir)
	if err != nil {
		return InResponse{}, err
	}

	var skipDownload bool

	if request.Params.SkipDownload != "" {
		skipDownload, err = strconv.ParseBool(request.Params.SkipDownload)
		if err != nil {
			return InResponse{}, fmt.Errorf("invalid skip_download value specified: %s", request.Params.SkipDownload)
		}
	} else {
		skipDownload = request.Source.SkipDownload
	}

	if request.Source.Regexp != "" {
		return command.inByRegex(destinationDir, request, skipDownload)
	} else {
		return command.inByVersionedFile(destinationDir, request, skipDownload)
	}
}

func (command *InCommand) createDirectory(destinationDir string) error {
	return os.MkdirAll(destinationDir, 0755)
}

func (command *InCommand) inByRegex(destinationDir string, request InRequest, skipDownload bool) (InResponse, error) {
	bucketName := request.Source.Bucket

	var url, objectPath string
	var err error

	isInitialVersion := request.Source.InitialPath != "" && request.Version.Path == request.Source.InitialPath
	if isInitialVersion {
		initialFilename := path.Base(request.Version.Path)

		err = ioutil.WriteFile(filepath.Join(destinationDir, initialFilename), request.Source.GetContents(), 0644)
		if err != nil {
			return InResponse{}, err
		}

		objectPath = request.Source.InitialPath
	} else {
		objectPath, err = command.pathToDownload(request)
		if err != nil {
			return InResponse{}, err
		}

		if !skipDownload {
			localPath := filepath.Join(destinationDir, filepath.Base(objectPath))

			if err = command.downloadFile(bucketName, objectPath, 0, localPath); err != nil {
				return InResponse{}, err
			}

			if request.Params.Unpack {
				if err := command.unpackFile(localPath); err != nil {
					return InResponse{}, err
				}
			}
		}

		url, err = command.gcsClient.URL(bucketName, objectPath, 0)
		if err != nil {
			return InResponse{}, err
		}

		if err = command.writeURLFile(url, destinationDir); err != nil {
			return InResponse{}, err
		}
	}

	version, ok := versions.Extract(objectPath, request.Source.Regexp)
	if ok {
		err := command.writeVersionFile(version.VersionNumber, destinationDir)
		if err != nil {
			return InResponse{}, err
		}
	}

	return InResponse{
		Version: gcsresource.Version{
			Path: objectPath,
		},
		Metadata: command.metadata(objectPath, url),
	}, nil
}

func (command *InCommand) pathToDownload(request InRequest) (string, error) {
	if request.Version.Path != "" {
		return request.Version.Path, nil
	}

	extractions := versions.GetBucketObjectVersions(command.gcsClient, request.Source)

	if len(extractions) == 0 {
		return "", errors.New("no extractions could be found - is your regexp correct?")
	}

	lastExtraction := extractions[len(extractions)-1]
	return lastExtraction.Path, nil
}

func (command *InCommand) inByVersionedFile(destinationDir string, request InRequest, skipDownload bool) (InResponse, error) {
	var url string
	bucketName := request.Source.Bucket
	objectPath := request.Source.VersionedFile
	generation, err := request.Version.GenerationValue()
	if err != nil {
		return InResponse{}, err
	}

	isInitialVersion := request.Source.InitialVersion != "" && request.Version.Generation == request.Source.InitialVersion

	if isInitialVersion {
		initialFilename := path.Base(request.Source.VersionedFile)

		err = ioutil.WriteFile(filepath.Join(destinationDir, initialFilename), request.Source.GetContents(), 0644)
		if err != nil {
			return InResponse{}, err
		}

		objectPath = request.Source.VersionedFile
	} else {
		if !skipDownload {
			localPath := filepath.Join(destinationDir, filepath.Base(objectPath))

			if err = command.downloadFile(bucketName, objectPath, generation, localPath); err != nil {
				return InResponse{}, err
			}

			if request.Params.Unpack {
				if err = command.unpackFile(localPath); err != nil {
					return InResponse{}, err
				}
			}
		}

		url, err = command.gcsClient.URL(bucketName, objectPath, generation)
		if err != nil {
			return InResponse{}, err
		}

		if err = command.writeURLFile(url, destinationDir); err != nil {
			return InResponse{}, err
		}
	}

	if err := command.writeGenerationFile(generation, destinationDir); err != nil {
		return InResponse{}, err
	}

	return InResponse{
		Version: gcsresource.Version{
			Generation: fmt.Sprintf("%d", generation),
		},
		Metadata: command.metadata(objectPath, url),
	}, nil
}

func (command *InCommand) writeVersionFile(version string, destinationDir string) error {
	return ioutil.WriteFile(filepath.Join(destinationDir, "version"), []byte(version), 0644)
}

func (command *InCommand) writeGenerationFile(generation int64, destinationDir string) error {
	return ioutil.WriteFile(filepath.Join(destinationDir, "generation"), []byte(strconv.FormatInt(generation, 10)), 0644)
}

func (command *InCommand) writeURLFile(url string, destinationDir string) error {
	return ioutil.WriteFile(filepath.Join(destinationDir, "url"), []byte(url), 0644)
}

func (command *InCommand) downloadFile(bucketName string, objectPath string, generation int64, localPath string) error {
	return command.gcsClient.DownloadFile(
		bucketName,
		objectPath,
		generation,
		localPath,
	)
}

func (command *InCommand) unpackFile(sourcePath string) error {

	var (
		errorMessage = "failed to extract '%s' with the 'params.unpack' option enabled: %s"
		fileName     = filepath.Base(sourcePath)
	)

	mimeType, err := getMimeType(sourcePath)
	if err != nil {
		return fmt.Errorf(errorMessage, fileName, err)
	}

	if !isSupportedMimeType(mimeType) {
		return fmt.Errorf(errorMessage, fileName, "unsupported MIME type "+mimeType)
	}

	if err := unpack(mimeType, sourcePath); err != nil {
		return fmt.Errorf(errorMessage, fileName, err)
	}

	return nil
}
func (command *InCommand) metadata(objectPath string, url string) []gcsresource.MetadataPair {
	objectFilename := filepath.Base(objectPath)

	metadata := []gcsresource.MetadataPair{
		{
			Name:  "filename",
			Value: objectFilename,
		},
	}

	if url != "" {
		metadata = append(metadata, gcsresource.MetadataPair{Name: "url", Value: url})
	}

	return metadata
}
