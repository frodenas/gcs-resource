package out_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/frodenas/gcs-resource"
	"github.com/frodenas/gcs-resource/fakes"

	. "github.com/frodenas/gcs-resource/out"
)

var _ = Describe("Out Command", func() {
	Describe("running the command", func() {
		var (
			err       error
			tmpPath   string
			sourceDir string
			request   OutRequest

			gcsClient *fakes.FakeGCSClient
			command   *OutCommand
		)

		BeforeEach(func() {
			tmpPath, err = ioutil.TempDir("", "out_command")
			Expect(err).ToNot(HaveOccurred())

			sourceDir = filepath.Join(tmpPath, "source")
			err = os.MkdirAll(sourceDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			request = OutRequest{
				Source: gcsresource.Source{
					Bucket: "bucket-name",
				},
				Params: Params{
					File: "files/file*.tgz",
				},
			}

			gcsClient = &fakes.FakeGCSClient{}
			command = NewOutCommand(gcsClient)
		})

		AfterEach(func() {
			err := os.RemoveAll(tmpPath)
			Expect(err).ToNot(HaveOccurred())
		})

		createFile := func(path string) {
			fullPath := filepath.Join(sourceDir, path)
			err := os.MkdirAll(filepath.Dir(fullPath), 0755)
			Expect(err).ToNot(HaveOccurred())

			file, err := os.Create(fullPath)
			Expect(err).ToNot(HaveOccurred())
			file.Close()
		}

		Describe("when the request is invalid", func() {
			Context("when the bucket is not set", func() {
				BeforeEach(func() {
					request.Source.Bucket = ""
				})

				It("returns an error", func() {
					_, err := command.Run(sourceDir, request)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("please specify the bucket"))
				})
			})

			Context("when the regexp and versioned_file are both set", func() {
				BeforeEach(func() {
					request.Source.Regexp = "folder/file-(.*).tgz"
					request.Source.VersionedFile = "files/version"
				})

				It("returns an error", func() {
					_, err := command.Run(sourceDir, request)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("please specify either regexp or versioned_file"))
				})
			})

			Context("when the file is not set", func() {
				BeforeEach(func() {
					request.Params.File = ""
				})

				It("returns an error", func() {
					_, err := command.Run(sourceDir, request)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("please specify the file"))
				})
			})
		})

		Describe("finding the local file to upload", func() {
			It("does not error if there is a single match", func() {
				createFile("files/file.tgz")
				createFile("files/test.tgz")

				_, err := command.Run(sourceDir, request)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error if there are no matches", func() {
				createFile("files/test.tgz")

				_, err := command.Run(sourceDir, request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no matches found for pattern"))
			})

			It("returns an error if there are more than one match", func() {
				createFile("files/file1.tgz")
				createFile("files/file2.tgz")

				_, err := command.Run(sourceDir, request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("more than one match found for pattern"))
			})
		})

		Describe("with regexp", func() {
			BeforeEach(func() {
				request.Source.Regexp = "folder/file-(.*).tgz"
				createFile("files/file.tgz")
			})

			It("uploads the file", func() {
				_, err := command.Run(sourceDir, request)
				Expect(err).ToNot(HaveOccurred())

				Expect(gcsClient.UploadFileCallCount()).To(Equal(1))
				bucketName, objectPath, localPath, predefinedACL := gcsClient.UploadFileArgsForCall(0)

				Expect(bucketName).To(Equal("bucket-name"))
				Expect(objectPath).To(Equal("folder/file.tgz"))
				Expect(localPath).To(Equal(filepath.Join(sourceDir, "files/file.tgz")))
				Expect(predefinedACL).To(BeEmpty())
			})

			It("returns a response", func() {
				gcsClient.UploadFileReturns(int64(12345), nil)
				gcsClient.URLReturns("gs://bucket-name/folder/file.tgz", nil)

				response, err := command.Run(sourceDir, request)
				Expect(err).ToNot(HaveOccurred())

				Expect(gcsClient.URLCallCount()).To(Equal(1))
				bucketName, objectPath, generation := gcsClient.URLArgsForCall(0)
				Expect(bucketName).To(Equal("bucket-name"))
				Expect(objectPath).To(Equal("folder/file.tgz"))
				Expect(generation).To(Equal(int64(0)))

				Expect(response.Version.Path).To(Equal("folder/file.tgz"))
				Expect(response.Version.Generation).To(Equal(int64(0)))

				Expect(response.Metadata[0].Name).To(Equal("filename"))
				Expect(response.Metadata[0].Value).To(Equal("file.tgz"))

				Expect(response.Metadata[1].Name).To(Equal("url"))
				Expect(response.Metadata[1].Value).To(Equal("gs://bucket-name/folder/file.tgz"))
			})

			It("returns an error if upload fails", func() {
				gcsClient.UploadFileReturns(int64(0), errors.New("error uploading file"))

				_, err := command.Run(sourceDir, request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error uploading file"))
			})

			It("does not return an error if url fails", func() {
				gcsClient.URLReturns("", errors.New("error url"))

				_, err := command.Run(sourceDir, request)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when PredefinedACL param is set", func() {
				BeforeEach(func() {
					request.Params.PredefinedACL = "publicRead"
				})

				It("uploads the file", func() {
					_, err := command.Run(sourceDir, request)
					Expect(err).ToNot(HaveOccurred())

					Expect(gcsClient.UploadFileCallCount()).To(Equal(1))
					bucketName, objectPath, localPath, predefinedACL := gcsClient.UploadFileArgsForCall(0)

					Expect(bucketName).To(Equal("bucket-name"))
					Expect(objectPath).To(Equal("folder/file.tgz"))
					Expect(localPath).To(Equal(filepath.Join(sourceDir, "files/file.tgz")))
					Expect(predefinedACL).To(Equal("publicRead"))
				})
			})
		})

		Describe("with versioned_file", func() {
			BeforeEach(func() {
				request.Source.VersionedFile = "folder/version"
				createFile("files/file.tgz")
			})

			It("uploads the file", func() {
				_, err := command.Run(sourceDir, request)
				Expect(err).ToNot(HaveOccurred())

				Expect(gcsClient.UploadFileCallCount()).To(Equal(1))
				bucketName, objectPath, localPath, predefinedACL := gcsClient.UploadFileArgsForCall(0)

				Expect(bucketName).To(Equal("bucket-name"))
				Expect(objectPath).To(Equal("folder/version"))
				Expect(localPath).To(Equal(filepath.Join(sourceDir, "files/file.tgz")))
				Expect(predefinedACL).To(BeEmpty())
			})

			It("returns a response", func() {
				gcsClient.UploadFileReturns(int64(12345), nil)
				gcsClient.URLReturns("gs://bucket-name/folder/file.tgz#12345", nil)

				response, err := command.Run(sourceDir, request)
				Expect(err).ToNot(HaveOccurred())

				Expect(gcsClient.URLCallCount()).To(Equal(1))
				bucketName, objectPath, generation := gcsClient.URLArgsForCall(0)
				Expect(bucketName).To(Equal("bucket-name"))
				Expect(objectPath).To(Equal("folder/version"))
				Expect(generation).To(Equal(int64(12345)))

				Expect(response.Version.Path).To(BeEmpty())
				Expect(response.Version.Generation).To(Equal(int64(12345)))

				Expect(response.Metadata[0].Name).To(Equal("filename"))
				Expect(response.Metadata[0].Value).To(Equal("version"))

				Expect(response.Metadata[1].Name).To(Equal("url"))
				Expect(response.Metadata[1].Value).To(Equal("gs://bucket-name/folder/file.tgz#12345"))
			})

			It("returns an error if upload fails", func() {
				gcsClient.UploadFileReturns(int64(0), errors.New("error uploading file"))

				_, err := command.Run(sourceDir, request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error uploading file"))
			})

			It("does not return an error if url fails", func() {
				gcsClient.URLReturns("", errors.New("error url"))

				_, err := command.Run(sourceDir, request)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when PredefinedACL param is set", func() {
				BeforeEach(func() {
					request.Params.PredefinedACL = "publicRead"
				})

				It("uploads the file", func() {
					_, err := command.Run(sourceDir, request)
					Expect(err).ToNot(HaveOccurred())

					Expect(gcsClient.UploadFileCallCount()).To(Equal(1))
					bucketName, objectPath, localPath, predefinedACL := gcsClient.UploadFileArgsForCall(0)

					Expect(bucketName).To(Equal("bucket-name"))
					Expect(objectPath).To(Equal("folder/version"))
					Expect(localPath).To(Equal(filepath.Join(sourceDir, "files/file.tgz")))
					Expect(predefinedACL).To(Equal("publicRead"))
				})
			})
		})
	})
})
