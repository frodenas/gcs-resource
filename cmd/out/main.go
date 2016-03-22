package main

import (
	"encoding/json"
	"os"

	"github.com/frodenas/gcs-resource"
	"github.com/frodenas/gcs-resource/out"
)

func main() {
	if len(os.Args) < 2 {
		gcsresource.Sayf("usage: %s <sources directory>\n", os.Args[0])
		os.Exit(1)
	}

	sourceDir := os.Args[1]

	var request out.OutRequest
	inputRequest(&request)

	gcsClient, err := gcsresource.NewGCSClient(
		os.Stderr,
		request.Source.Project,
		request.Source.JSONKey,
	)
	if err != nil {
		gcsresource.Fatal("building GCS client", err)
	}

	command := out.NewOutCommand(gcsClient)
	response, err := command.Run(sourceDir, request)
	if err != nil {
		gcsresource.Fatal("running command", err)
	}

	outputResponse(response)
}

func inputRequest(request *out.OutRequest) {
	if err := json.NewDecoder(os.Stdin).Decode(request); err != nil {
		gcsresource.Fatal("reading request from stdin", err)
	}
}

func outputResponse(response out.OutResponse) {
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		gcsresource.Fatal("writing response to stdout", err)
	}
}
