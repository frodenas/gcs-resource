package integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GCSclient", func() {
	var (
		err             error
		tempDir         string
		tempFile        *os.File
		tempVerDir      string
		tempVerFile     *os.File
		runtime         string
		directoryPrefix string
	)

	BeforeEach(func() {
		directoryPrefix = "gcsclient-tests"
		runtime = fmt.Sprintf("%d", time.Now().Unix())
	})

	Describe("with a non versioned bucket", func() {
		BeforeEach(func() {
			tempDir, err = ioutil.TempDir("", "gcs_client_integration_test")
			Expect(err).ToNot(HaveOccurred())

			tempFile, err = ioutil.TempFile(tempDir, "file-to-upload")
			Expect(err).ToNot(HaveOccurred())

			tempFile.Write([]byte("hello-" + runtime))
		})

		AfterEach(func() {
			err := os.RemoveAll(tempDir)
			Expect(err).ToNot(HaveOccurred())

			err = gcsClient.DeleteObject(bucketName, filepath.Join(directoryPrefix, "file-to-upload-1"), 0)
			Expect(err).ToNot(HaveOccurred())

			err = gcsClient.DeleteObject(bucketName, filepath.Join(directoryPrefix, "file-to-upload-2"), 0)
			Expect(err).ToNot(HaveOccurred())

			err = gcsClient.DeleteObject(bucketName, filepath.Join(directoryPrefix, "zip-to-upload.zip"), 0)
			Expect(err).ToNot(HaveOccurred())
		})

		It("can interact with buckets", func() {
			_, err := gcsClient.UploadFile(bucketName, filepath.Join(directoryPrefix, "file-to-upload-1"), "", tempFile.Name(), "")
			Expect(err).ToNot(HaveOccurred())

			_, err = gcsClient.UploadFile(bucketName, filepath.Join(directoryPrefix, "file-to-upload-2"), "", tempFile.Name(), "")
			Expect(err).ToNot(HaveOccurred())

			_, err = gcsClient.UploadFile(bucketName, filepath.Join(directoryPrefix, "file-to-upload-2"), "", tempFile.Name(), "")
			Expect(err).ToNot(HaveOccurred())

			_, err = gcsClient.UploadFile(bucketName, filepath.Join(directoryPrefix, "zip-to-upload.zip"), "application/zip", tempFile.Name(), "")
			Expect(err).ToNot(HaveOccurred())

			fakeZipFileObject, err := gcsClient.GetBucketObjectInfo(bucketName, filepath.Join(directoryPrefix, "zip-to-upload.zip"))
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeZipFileObject.ContentType).To(Equal("application/zip"))

			files, err := gcsClient.BucketObjects(bucketName, directoryPrefix)
			Expect(err).ToNot(HaveOccurred())
			Expect(files).To(ConsistOf([]string{filepath.Join(directoryPrefix, "file-to-upload-1"), filepath.Join(directoryPrefix, "file-to-upload-2"), filepath.Join(directoryPrefix, "zip-to-upload.zip")}))

			_, err = gcsClient.ObjectGenerations(bucketName, filepath.Join(directoryPrefix, "file-to-upload-1"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("bucket is not versioned"))

			fileOneURL, err := gcsClient.URL(bucketName, filepath.Join(directoryPrefix, "file-to-upload-1"), 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(fileOneURL).To(Equal(fmt.Sprintf("gs://%s/%s", bucketName, filepath.Join(directoryPrefix, "file-to-upload-1"))))

			_, err = gcsClient.ObjectGenerations(bucketName, filepath.Join(directoryPrefix, "file-to-upload-2"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("bucket is not versioned"))

			fileTwoURL, err := gcsClient.URL(bucketName, filepath.Join(directoryPrefix, "file-to-upload-2"), 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(fileTwoURL).To(Equal(fmt.Sprintf("gs://%s/%s", bucketName, filepath.Join(directoryPrefix, "file-to-upload-2"))))

			err = gcsClient.DownloadFile(bucketName, filepath.Join(directoryPrefix, "file-to-upload-1"), 0, filepath.Join(tempDir, "downloaded-file"))
			Expect(err).ToNot(HaveOccurred())

			read, err := ioutil.ReadFile(filepath.Join(tempDir, "downloaded-file"))
			Expect(err).ToNot(HaveOccurred())
			Expect(read).To(Equal([]byte("hello-" + runtime)))
		})
	})

	Describe("with a versioned bucket", func() {
		BeforeEach(func() {
			tempVerDir, err = ioutil.TempDir("", "gcs-versioned-upload-dir")
			Expect(err).ToNot(HaveOccurred())

			tempVerFile, err = ioutil.TempFile(tempVerDir, "file-to-upload")
			Expect(err).ToNot(HaveOccurred())

			tempVerFile.Write([]byte("hello-" + runtime))
		})

		AfterEach(func() {
			err := os.RemoveAll(tempVerDir)
			Expect(err).ToNot(HaveOccurred())

			fileOneGenerations, err := gcsClient.ObjectGenerations(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-1"))
			Expect(err).ToNot(HaveOccurred())

			for _, fileOneGeneration := range fileOneGenerations {
				err := gcsClient.DeleteObject(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-1"), fileOneGeneration)
				Expect(err).ToNot(HaveOccurred())
			}

			fileTwoGenerations, err := gcsClient.ObjectGenerations(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-2"))
			Expect(err).ToNot(HaveOccurred())

			for _, fileTwoGeneration := range fileTwoGenerations {
				err := gcsClient.DeleteObject(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-2"), fileTwoGeneration)
				Expect(err).ToNot(HaveOccurred())
			}

			fakeZipFileGenerations, err := gcsClient.ObjectGenerations(versionedBucketName, filepath.Join(directoryPrefix, "zip-to-upload.zip"))
			Expect(err).ToNot(HaveOccurred())

			for _, fakeZipFileGeneration := range fakeZipFileGenerations {
				err := gcsClient.DeleteObject(versionedBucketName, filepath.Join(directoryPrefix, "zip-to-upload.zip"), fakeZipFileGeneration)
				Expect(err).ToNot(HaveOccurred())
			}
		})

		It("can interact with buckets", func() {
			fileOneGeneration, err := gcsClient.UploadFile(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-1"), "", tempVerFile.Name(), "")
			Expect(err).ToNot(HaveOccurred())

			fileTwoGeneration1, err := gcsClient.UploadFile(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-2"), "", tempVerFile.Name(), "")
			Expect(err).ToNot(HaveOccurred())

			fileTwoGeneration2, err := gcsClient.UploadFile(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-2"), "", tempVerFile.Name(), "")
			Expect(err).ToNot(HaveOccurred())

			fakeZipFileGeneration, err := gcsClient.UploadFile(versionedBucketName, filepath.Join(directoryPrefix, "zip-to-upload.zip"), "application/zip", tempVerFile.Name(), "")
			Expect(err).ToNot(HaveOccurred())

			fakeZipFileObject, err := gcsClient.GetBucketObjectInfo(versionedBucketName, filepath.Join(directoryPrefix, "zip-to-upload.zip"))
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeZipFileObject.ContentType).To(Equal("application/zip"))
			Expect(fakeZipFileGeneration).To(Equal(fakeZipFileObject.Generation))

			files, err := gcsClient.BucketObjects(versionedBucketName, directoryPrefix)
			Expect(err).ToNot(HaveOccurred())
			Expect(files).To(ConsistOf([]string{filepath.Join(directoryPrefix, "file-to-upload-1"), filepath.Join(directoryPrefix, "file-to-upload-2"), filepath.Join(directoryPrefix, "zip-to-upload.zip")}))

			fileOneGenerations, err := gcsClient.ObjectGenerations(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-1"))
			Expect(err).ToNot(HaveOccurred())
			Expect(fileOneGenerations).To(ConsistOf([]int64{fileOneGeneration}))

			fileOneGenerationsObject, err := gcsClient.GetBucketObjectInfo(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-1"))
			Expect(err).ToNot(HaveOccurred())
			Expect(fileOneGenerations).To(ConsistOf([]int64{fileOneGenerationsObject.Generation}))

			fileOneURL, err := gcsClient.URL(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-1"), 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(fileOneURL).To(Equal(fmt.Sprintf("gs://%s/%s", versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-1"))))

			fileOneURLGeneration, err := gcsClient.URL(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-1"), fileOneGeneration)
			Expect(err).ToNot(HaveOccurred())
			Expect(fileOneURLGeneration).To(Equal(fmt.Sprintf("gs://%s/%s#%d", versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-1"), fileOneGeneration)))

			fileTwoGenerations, err := gcsClient.ObjectGenerations(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-2"))
			Expect(err).ToNot(HaveOccurred())
			Expect(fileTwoGenerations).To(ConsistOf([]int64{fileTwoGeneration1, fileTwoGeneration2}))

			fileTwoURL, err := gcsClient.URL(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-2"), 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(fileTwoURL).To(Equal(fmt.Sprintf("gs://%s/%s", versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-2"))))

			fileTwoURLGeneration1, err := gcsClient.URL(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-2"), fileTwoGeneration1)
			Expect(err).ToNot(HaveOccurred())
			Expect(fileTwoURLGeneration1).To(Equal(fmt.Sprintf("gs://%s/%s#%d", versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-2"), fileTwoGeneration1)))

			fileTwoURLGeneration2, err := gcsClient.URL(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-2"), fileTwoGeneration2)
			Expect(err).ToNot(HaveOccurred())
			Expect(fileTwoURLGeneration2).To(Equal(fmt.Sprintf("gs://%s/%s#%d", versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-2"), fileTwoGeneration2)))

			err = gcsClient.DownloadFile(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload-1"), 0, filepath.Join(tempVerDir, "downloaded-file"))
			Expect(err).ToNot(HaveOccurred())

			read, err := ioutil.ReadFile(filepath.Join(tempVerDir, "downloaded-file"))
			Expect(err).ToNot(HaveOccurred())
			Expect(read).To(Equal([]byte("hello-" + runtime)))
		})
	})
})
