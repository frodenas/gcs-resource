default: test

# Build gcs-resource binaries
build:
	go build -o assets/in ./cmd/in
	go build -o assets/out ./cmd/out
	go build -o assets/check ./cmd/check

# Get all dependencies
get-deps:
	# Go tools
	go get github.com/golang/lint/golint

	# Test tools
	go get github.com/onsi/ginkgo/ginkgo
	go get github.com/onsi/gomega

# Cleans up
clean:
	go clean ./...

# Run gofmt
fmt:
	ls -d */ | grep -v vendor | xargs -L 1 gofmt -l -w

# Run golint
lint:
	ls -d */ | grep -v vendor | xargs -L 1 golint

# Vet code
vet:
	go tool vet $$(ls -d */ | grep -v vendor)

# Run unit tests
unit-tests:
	ginkgo -r -p -skipPackage integration,vendor

# Run integration tests
integration-tests:
	ginkgo -r -p integration

# Runs the unit tests with coverage
test: get-deps clean fmt lint vet unit-tests build
