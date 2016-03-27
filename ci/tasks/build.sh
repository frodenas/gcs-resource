#!/bin/bash

set -e -x

BUILD_DIR=$PWD/built-resource

export GOPATH=$PWD/go
export PATH=$GOPATH/bin:$PATH

cd $GOPATH/src/github.com/frodenas/gcs-resource

mkdir -p assets
make build

cp -a assets/ Dockerfile $BUILD_DIR
