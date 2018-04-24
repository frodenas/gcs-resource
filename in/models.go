package in

import (
	"github.com/frodenas/gcs-resource"
)

type InRequest struct {
	Source  gcsresource.Source  `json:"source"`
	Version gcsresource.Version `json:"version"`
	Params  Params              `json:"params"`
}

type Params struct {
	Unpack bool `json:"unpack"`
}

type InResponse struct {
	Version  gcsresource.Version        `json:"version"`
	Metadata []gcsresource.MetadataPair `json:"metadata"`
}
