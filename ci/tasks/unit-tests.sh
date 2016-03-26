#!/bin/bash

set -e

go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/gomega

export GOPATH=$PWD/go
ginkgo -r -p -skipPackage integration,vendor
