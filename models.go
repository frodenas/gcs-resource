package gcsresource

import "strconv"

type Source struct {
	JSONKey       string `json:"json_key"`
	Bucket        string `json:"bucket"`
	Regexp        string `json:"regexp"`
	VersionedFile string `json:"versioned_file"`
	SkipDownload  bool   `json:"skip_download"`
}

func (source Source) IsValid() (bool, string) {
	if source.Bucket == "" {
		return false, "please specify the bucket"
	}

	if source.Regexp != "" && source.VersionedFile != "" {
		return false, "please specify either regexp or versioned_file"
	}

	return true, ""
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
