#!/bin/bash

set -e

export GOPATH=$PWD/go
export PATH=$GOPATH/bin:$PATH

cd $GOPATH/src/github.com/frodenas/gcs-resource
make
