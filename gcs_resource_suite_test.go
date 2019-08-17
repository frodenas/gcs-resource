package gcsresource_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGcsResource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GcsResource Suite")
}
