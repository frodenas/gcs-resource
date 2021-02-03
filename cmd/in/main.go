package main

import (
	"encoding/json"
	"os"

	"github.com/frodenas/gcs-resource"
	"github.com/frodenas/gcs-resource/in"
)

func main() {
	if len(os.Args) < 2 {
		gcsresource.Sayf("usage: %s <dest directory>\n", os.Args[0])
		os.Exit(1)
	}

	destinationDir := os.Args[1]

	var request in.InRequest
	inputRequest(&request)

	gcsClient, err := gcsresource.NewGCSClient(
		os.Stderr,
		request.Source.JSONKey,
		request.Source.AccessToken,
	)
	if err != nil {
		gcsresource.Fatal("building GCS client", err)
	}

	command := in.NewInCommand(gcsClient)
	response, err := command.Run(destinationDir, request)
	if err != nil {
		gcsresource.Fatal("running command", err)
	}

	outputResponse(response)
}

func inputRequest(request *in.InRequest) {
	if err := json.NewDecoder(os.Stdin).Decode(request); err != nil {
		gcsresource.Fatal("reading request from stdin", err)
	}
}

func outputResponse(response in.InResponse) {
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		gcsresource.Fatal("writing response to stdout", err)
	}
}
