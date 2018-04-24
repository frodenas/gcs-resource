package in_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gexec"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

var inPath string

var _ = BeforeSuite(func() {
	var err error

	inPath, err = gexec.Build("github.com/frodenas/gcs-resource/cmd/in")
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

func TestIn(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "In Suite")
}

type gcsDownloadTask func(bucketName string, objectPath string, generation int64) (string, error)

func gcsDownloadTaskStub(name string) gcsDownloadTask {
	return func(bucketName string, objectPath string, generation int64) (string, error) {
		sourcePath := filepath.Join("fixtures", name)
		Expect(sourcePath).To(BeAnExistingFile())

		from, err := os.Open(sourcePath)
		Expect(err).NotTo(HaveOccurred())
		defer from.Close()

		tempDir, err := ioutil.TempDir("", "in_command_test_")
		Expect(err).NotTo(HaveOccurred())

		to, err := os.OpenFile(filepath.Join(tempDir, name), os.O_RDWR|os.O_CREATE, 0600)
		defer to.Close()

		_, err = io.Copy(to, from)
		Expect(err).NotTo(HaveOccurred())
		return to.Name(), nil
	}
}
