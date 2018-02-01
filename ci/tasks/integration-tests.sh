#!/bin/bash

set -e

: ${GCS_RESOURCE_JSON_KEY:?}
: ${GCS_RESOURCE_BUCKET_NAME:?}
: ${GCS_RESOURCE_VERSIONED_BUCKET_NAME:?}

export GOPATH=$PWD/go
export PATH=$GOPATH/bin:$PATH

cd $GOPATH/src/github.com/frodenas/gcs-resource
make integration-tests
