#!/bin/bash

set -e

go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/gomega

export GOPATH=$PWD/go
CGO_ENABLED=1 ginkgo -race -r -p
