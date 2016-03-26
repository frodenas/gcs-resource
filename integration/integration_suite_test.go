package integration_test

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/frodenas/gcs-resource"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
}

var jsonKey = os.Getenv("GCS_RESOURCE_JSON_KEY")
var bucketName = os.Getenv("GCS_RESOURCE_BUCKET_NAME")
var versionedBucketName = os.Getenv("GCS_RESOURCE_VERSIONED_BUCKET_NAME")
var gcsClient gcsresource.GCSClient

var checkPath string
var inPath string
var outPath string

type suiteData struct {
	CheckPath string
	InPath    string
	OutPath   string
}

var _ = SynchronizedBeforeSuite(func() []byte {
	cp, err := gexec.Build("github.com/frodenas/gcs-resource/cmd/check")
	Expect(err).ToNot(HaveOccurred())
	ip, err := gexec.Build("github.com/frodenas/gcs-resource/cmd/in")
	Expect(err).ToNot(HaveOccurred())
	op, err := gexec.Build("github.com/frodenas/gcs-resource/cmd/out")
	Expect(err).ToNot(HaveOccurred())

	data, err := json.Marshal(suiteData{
		CheckPath: cp,
		InPath:    ip,
		OutPath:   op,
	})
	Expect(err).ToNot(HaveOccurred())

	return data
}, func(data []byte) {
	var sd suiteData
	err := json.Unmarshal(data, &sd)
	Expect(err).ToNot(HaveOccurred())

	checkPath = sd.CheckPath
	inPath = sd.InPath
	outPath = sd.OutPath

	Expect(bucketName).ToNot(BeEmpty(), "must specify $GCS_TESTING_BUCKET")
	Expect(versionedBucketName).ToNot(BeEmpty(), "must specify $GCS_VERSIONED_TESTING_BUCKET")

	gcsClient, err = gcsresource.NewGCSClient(
		ioutil.Discard,
		jsonKey,
	)
	Expect(err).ToNot(HaveOccurred())
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})

func TestIn(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}
