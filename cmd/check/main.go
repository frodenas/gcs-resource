package main

import (
	"encoding/json"
	"os"

	"github.com/frodenas/gcs-resource"
	"github.com/frodenas/gcs-resource/check"
)

func main() {
	var request check.CheckRequest
	inputRequest(&request)

	gcsClient, err := gcsresource.NewGCSClient(
		os.Stderr,
		request.Source.Project,
		request.Source.JSONKey,
	)
	if err != nil {
		gcsresource.Fatal("building GCS client", err)
	}

	command := check.NewCheckCommand(gcsClient)
	response, err := command.Run(request)
	if err != nil {
		gcsresource.Fatal("running command", err)
	}

	outputResponse(response)
}

func inputRequest(request *check.CheckRequest) {
	if err := json.NewDecoder(os.Stdin).Decode(request); err != nil {
		gcsresource.Fatal("reading request from stdin", err)
	}
}

func outputResponse(response check.CheckResponse) {
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		gcsresource.Fatal("writing response to stdout", err)
	}
}
