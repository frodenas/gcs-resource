#!/bin/bash

set -e -x

BUILD_DIR=$PWD/built-resource

export GOPATH=$PWD/go
export PATH=$GOPATH/bin:$PATH

cd $GOPATH/src/github.com/frodenas/gcs-resource

mkdir -p assets
go build -o assets/in ./cmd/in
go build -o assets/out ./cmd/out
go build -o assets/check ./cmd/check

cp -a assets/ Dockerfile $BUILD_DIR
