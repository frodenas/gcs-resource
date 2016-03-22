package gcsresource

type Source struct {
	Project       string `json:"project"`
	JSONKey       string `json:"json_key"`
	Bucket        string `json:"bucket"`
	Regexp        string `json:"regexp"`
	VersionedFile string `json:"versioned_file"`
}

func (source Source) IsValid() (bool, string) {
	if source.Project == "" {
		return false, "please specify the project"
	}

	if source.JSONKey == "" {
		return false, "please specify the json_key"
	}

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
	Generation int64  `json:"generation,omitempty"`
}

type MetadataPair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
