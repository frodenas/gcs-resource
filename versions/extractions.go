package versions

import (
	"github.com/blang/semver"
)

type Extractions []Extraction

func (e Extractions) Len() int {
	return len(e)
}

func (e Extractions) Less(i int, j int) bool {
	return e[i].Version.LT(e[j].Version)
}

func (e Extractions) Swap(i int, j int) {
	e[i], e[j] = e[j], e[i]
}

type Extraction struct {
	// path to gcs object in bucket
	Path string

	// parsed semantic version
	Version semver.Version

	// the raw version match
	VersionNumber string
}
