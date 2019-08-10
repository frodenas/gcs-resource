package check

import (
	"errors"
	"fmt"
	"github.com/frodenas/gcs-resource"
	"github.com/frodenas/gcs-resource/versions"
)

type CheckCommand struct {
	gcsClient gcsresource.GCSClient
}

func NewCheckCommand(gcsClient gcsresource.GCSClient) *CheckCommand {
	return &CheckCommand{
		gcsClient: gcsClient,
	}
}

func (command *CheckCommand) Run(request CheckRequest) (CheckResponse, error) {
	if ok, message := request.Source.IsValid(); !ok {
		return CheckResponse{}, errors.New(message)
	}

	if request.Source.Regexp != "" {
		return command.checkByRegex(request), nil
	} else {
		return command.checkByVersionedFile(request)
	}
}

func (command *CheckCommand) checkByRegex(request CheckRequest) CheckResponse {
	extractions := versions.GetBucketObjectVersions(command.gcsClient, request.Source)

	if request.Source.InitialPath != "" {
		extraction, ok := versions.Extract(request.Source.InitialPath, request.Source.Regexp)
		if ok {
			extractions = append([]versions.Extraction{extraction}, extractions...)
		}
	}

	if len(extractions) == 0 {
		return CheckResponse{}
	}

	lastVersion, matched := versions.Extract(request.Version.Path, request.Source.Regexp)
	if !matched {
		return latestVersion(extractions)
	} else {
		return newerVersions(lastVersion, extractions)
	}
}

func (command *CheckCommand) checkByVersionedFile(request CheckRequest) (CheckResponse, error) {
	response := CheckResponse{}

	generations, err := command.gcsClient.ObjectGenerations(request.Source.Bucket, request.Source.VersionedFile)
	if err != nil {
		return response, err
	}

	if request.Source.InitialVersion != "" {
		generations = append(generations, request.Source.InitialVersionNumber)
	}

	if len(generations) == 0 {
		return response, nil
	}

	if request.Version.Generation != "" {
		for _, generation := range generations {
			requestGeneration, err := request.Version.GenerationValue()
			if err != nil {
				return nil, err
			}

			if generation > requestGeneration {
				version := gcsresource.Version{
					Generation: fmt.Sprintf("%d", generation),
				}
				response = append(response, version)
			}
		}
	} else {
		maxGeneration := generations[0]
		for _, generation := range generations {
			if generation > maxGeneration {
				maxGeneration = generation
			}
		}

		version := gcsresource.Version{
			Generation: fmt.Sprintf("%d", maxGeneration),
		}
		response = append(response, version)
	}

	return response, nil
}

func latestVersion(extractions versions.Extractions) CheckResponse {
	lastExtraction := extractions[len(extractions)-1]
	return []gcsresource.Version{{Path: lastExtraction.Path}}
}

func newerVersions(lastVersion versions.Extraction, extractions versions.Extractions) CheckResponse {
	response := CheckResponse{}

	for _, extraction := range extractions {
		if extraction.Version.Compare(lastVersion.Version) > 0 {
			version := gcsresource.Version{
				Path: extraction.Path,
			}
			response = append(response, version)
		}
	}

	return response
}
