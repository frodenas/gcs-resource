package gcsresource

import (
	"encoding/base64"
	"strconv"
)

type Source struct {
	JSONKey              string `json:"json_key"`
	Bucket               string `json:"bucket"`
	Regexp               string `json:"regexp"`
	VersionedFile        string `json:"versioned_file"`
	SkipDownload         bool   `json:"skip_download"`
	InitialVersion       string `json:"initial_version"`
	InitialVersionNumber int64
	InitialPath          string `json:"initial_path"`
	InitialContentText   string `json:"initial_content_text"`
	InitialContentBinary string `json:"initial_content_binary"`
}

func (source *Source) IsValid() (bool, string) {
	if source.Bucket == "" {
		return false, "please specify the bucket"
	}

	if source.Regexp != "" && source.VersionedFile != "" {
		return false, "please specify either regexp or versioned_file"
	}

	if source.InitialVersion != "" {
		version, err := strconv.ParseInt(source.InitialVersion, 10, 64)
		if err != nil {
			return false, "if set, initial_version must be an int64"
		}
		source.InitialVersionNumber = version
	}

	if source.Regexp != "" && source.InitialVersion != "" {
		return false, "use initial_path when regexp is set"
	}

	if source.VersionedFile != "" && source.InitialPath != "" {
		return false, "use initial_version when versioned_file is set"
	}

	if source.InitialContentText != "" && source.InitialContentBinary != "" {
		return false, "use initial_content_text or initial_content_binary but not both"
	}

	_, err := base64.StdEncoding.DecodeString(source.InitialContentBinary)
	if err != nil {
		return false, "initial_content_binary could not be decoded to base64"
	}

	hasInitialContent := source.InitialContentText != "" || source.InitialContentBinary != ""
	if hasInitialContent && source.InitialVersion == "" && source.InitialPath == "" {
		return false, "use initial_version or initial_path when initial content is set"
	}

	return true, ""
}

func (source *Source) GetContents() []byte {
	var contents []byte

	if source.InitialContentText != "" {
		contents = []byte(source.InitialContentText)
	} else if source.InitialContentBinary != "" {
		contents, _ = base64.StdEncoding.DecodeString(source.InitialContentBinary)
	}

	return contents
}

type Version struct {
	Path       string `json:"path,omitempty"`
	Generation string `json:"generation,omitempty"`
}

func (v Version) GenerationValue() (int64, error) {
	i, err := strconv.ParseInt(v.Generation, 10, 64)
	if err != nil {
		return 0, err
	}
	return i, nil
}

type MetadataPair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
