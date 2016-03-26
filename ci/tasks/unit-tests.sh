#!/bin/bash

set -e

export GOPATH=$PWD/go
export PATH=$GOPATH/bin:$PATH

cd $GOPATH/src/github.com/frodenas/gcs-resource

go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/gomega

ginkgo -r -p -skipPackage integration,vendor
