package versions

import (
	"github.com/cppforlife/go-semi-semantic/version"
)

type Extractions []Extraction

func (e Extractions) Len() int {
	return len(e)
}

func (e Extractions) Less(i int, j int) bool {
	return e[i].Version.IsLt(e[j].Version)
}

func (e Extractions) Swap(i int, j int) {
	e[i], e[j] = e[j], e[i]
}

type Extraction struct {
	// path to gcs object in bucket
	Path string

	// parsed version
	Version version.Version

	// the raw version match
	VersionNumber string
}
