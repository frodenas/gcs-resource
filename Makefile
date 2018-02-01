GO     := GO15VENDOREXPERIMENT=1 go
GINKGO := ginkgo
pkgs   = $(shell $(GO) list ./... | grep -v /vendor/)

DOCKER_IMAGE_NAME ?= gcs-resource
DOCKER_IMAGE_TAG  ?= $(subst /,-,$(shell git rev-parse --abbrev-ref HEAD))

default: format build unit-tests

deps:
	@$(GO) get github.com/onsi/ginkgo/ginkgo
	@$(GO) get github.com/onsi/gomega

format:
	@echo ">> formatting code"
	@$(GO) fmt $(pkgs)

style:
	@echo ">> checking code style"
	@! gofmt -d $(shell find . -path ./vendor -prune -o -name '*.go' -print) | grep '^'

vet:
	@echo ">> vetting code"
	@$(GO) vet $(pkgs)

build:
	@echo ">> building binaries"
	@$(GO) build -o assets/in ./cmd/in
	@$(GO) build -o assets/out ./cmd/out
	@$(GO) build -o assets/check ./cmd/check

unit-tests: deps
	@echo ">> running unit tests"
	@$(GINKGO) version
	@$(GINKGO) -r -race -p -skipPackage integration,vendor

integration-tests: deps
	@echo ">> running integration tests"
	@$(GINKGO) version
	@$(GINKGO) -r -p integration

docker:
	@echo ">> building docker image"
	@docker build -t "$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)" .

.PHONY: default deps format style vet build unit-tests integration-tests docker
