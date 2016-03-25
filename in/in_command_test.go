package in_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/frodenas/gcs-resource"
	"github.com/frodenas/gcs-resource/fakes"

	. "github.com/frodenas/gcs-resource/in"
)

var _ = Describe("In Command", func() {
	Describe("running the command", func() {
		var (
			err     error
			tmpPath string
			destDir string
			request InRequest

			gcsClient *fakes.FakeGCSClient
			command   *InCommand
		)

		BeforeEach(func() {
			tmpPath, err = ioutil.TempDir("", "in_command")
			Expect(err).ToNot(HaveOccurred())

			destDir = filepath.Join(tmpPath, "destination")

			request = InRequest{
				Source: gcsresource.Source{
					Bucket: "bucket-name",
				},
			}

			gcsClient = &fakes.FakeGCSClient{}
			command = NewInCommand(gcsClient)
		})

		AfterEach(func() {
			err := os.RemoveAll(tmpPath)
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("when the request is invalid", func() {
			Context("when the bucket is not set", func() {
				BeforeEach(func() {
					request.Source.Bucket = ""
				})

				It("returns an error", func() {
					_, err := command.Run(destDir, request)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("please specify the bucket"))
				})
			})

			Context("when the regexp and versioned_file are both set", func() {
				BeforeEach(func() {
					request.Source.Regexp = "folder/file-(.*).tgz"
					request.Source.VersionedFile = "folder/version"
				})

				It("returns an error", func() {
					_, err := command.Run(destDir, request)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("please specify either regexp or versioned_file"))
				})
			})
		})

		Describe("with regexp", func() {
			BeforeEach(func() {
				request.Source.Regexp = "folder/file-(.*).tgz"
			})

			Describe("with versions that would fail if lexicographically ordered", func() {
				BeforeEach(func() {
					request.Version.Path = ""

					gcsClient.BucketObjectsReturns([]string{
						"folder/file-1.5.6-build.10.tgz",
						"folder/file-1.5.6-build.100.tgz",
						"folder/file-1.5.6-build.9.tgz",
					}, nil)
				})

				It("scans the bucket for the latest file to download", func() {
					_, err := command.Run(destDir, request)
					Expect(err).ToNot(HaveOccurred())

					Expect(gcsClient.DownloadFileCallCount()).To(Equal(1))
					bucketName, objectPath, generation, localPath := gcsClient.DownloadFileArgsForCall(0)

					Expect(bucketName).To(Equal("bucket-name"))
					Expect(objectPath).To(Equal("folder/file-1.5.6-build.100.tgz"))
					Expect(generation).To(Equal(int64(0)))
					Expect(localPath).To(Equal(filepath.Join(destDir, "file-1.5.6-build.100.tgz")))
				})
			})

			Describe("when there is no existing version in the request", func() {
				BeforeEach(func() {
					request.Version.Path = ""

					gcsClient.BucketObjectsReturns([]string{
						"folder/file-0.0.1.tgz",
						"folder/file-3.53.tgz",
						"folder/file-2.33.333.tgz",
						"folder/file-2.4.3.tgz",
					}, nil)
				})

				It("creates the destination directory", func() {
					Expect(destDir).ToNot(ExistOnFilesystem())

					_, err := command.Run(destDir, request)
					Expect(err).ToNot(HaveOccurred())

					Expect(destDir).To(ExistOnFilesystem())
				})

				It("scans the bucket for the latest file to download", func() {
					_, err := command.Run(destDir, request)
					Expect(err).ShouldNot(HaveOccurred())

					Expect(gcsClient.DownloadFileCallCount()).To(Equal(1))
					bucketName, objectPath, generation, localPath := gcsClient.DownloadFileArgsForCall(0)

					Expect(bucketName).To(Equal("bucket-name"))
					Expect(objectPath).To(Equal("folder/file-3.53.tgz"))
					Expect(generation).To(Equal(int64(0)))
					Expect(localPath).To(Equal(filepath.Join(destDir, "file-3.53.tgz")))
				})

				It("creates a 'version' file that contains the latest version", func() {
					versionFile := filepath.Join(destDir, "version")
					Expect(versionFile).ToNot(ExistOnFilesystem())

					_, err := command.Run(destDir, request)
					Expect(err).ToNot(HaveOccurred())

					Expect(versionFile).To(ExistOnFilesystem())
					contents, err := ioutil.ReadFile(versionFile)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(contents)).To(Equal("3.53"))
				})

				It("creates a 'url' file that contains the URL", func() {
					gcsClient.URLReturns("gs://bucket-name/folder/file-3.53.tgz", nil)

					urlFile := filepath.Join(destDir, "url")
					Expect(urlFile).ToNot(ExistOnFilesystem())

					_, err := command.Run(destDir, request)
					Expect(err).ToNot(HaveOccurred())

					bucketName, objectPath, generation := gcsClient.URLArgsForCall(0)
					Expect(bucketName).To(Equal("bucket-name"))
					Expect(objectPath).To(Equal("folder/file-3.53.tgz"))
					Expect(generation).To(Equal(int64(0)))

					Expect(urlFile).To(ExistOnFilesystem())
					contents, err := ioutil.ReadFile(urlFile)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(contents)).To(Equal("gs://bucket-name/folder/file-3.53.tgz"))
				})

				It("returns a response", func() {
					gcsClient.URLReturns("gs://bucket-name/folder/file-3.53.tgz", nil)

					response, err := command.Run(destDir, request)
					Expect(err).ToNot(HaveOccurred())

					Expect(response.Version.Path).To(Equal("folder/file-3.53.tgz"))
					Expect(response.Version.Generation).To(Equal(int64(0)))

					Expect(response.Metadata[0].Name).To(Equal("filename"))
					Expect(response.Metadata[0].Value).To(Equal("file-3.53.tgz"))

					Expect(response.Metadata[1].Name).To(Equal("url"))
					Expect(response.Metadata[1].Value).To(Equal("gs://bucket-name/folder/file-3.53.tgz"))
				})

				It("returns an error when the regexp has no groups", func() {
					request.Source.Regexp = "folder/file-.*.tgz"

					_, err := command.Run(destDir, request)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("no extractions could be found - is your regexp correct?"))
				})

				It("returns an error if download fails", func() {
					gcsClient.DownloadFileReturns(errors.New("error downloading file"))

					_, err := command.Run(destDir, request)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("error downloading file"))
				})

				It("returns an error if url fails", func() {
					gcsClient.URLReturns("", errors.New("error url"))

					_, err := command.Run(destDir, request)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("error url"))
				})
			})

			Describe("when there is an existing version in the request", func() {
				BeforeEach(func() {
					request.Version.Path = "folder/file-1.3.tgz"
				})

				It("creates the destination directory", func() {
					Expect(destDir).ToNot(ExistOnFilesystem())

					_, err := command.Run(destDir, request)
					Expect(err).ToNot(HaveOccurred())

					Expect(destDir).To(ExistOnFilesystem())
				})

				It("downloads the existing version of the file", func() {
					_, err := command.Run(destDir, request)
					Expect(err).ToNot(HaveOccurred())

					Expect(gcsClient.DownloadFileCallCount()).Should(Equal(1))
					bucketName, objectPath, generation, localPath := gcsClient.DownloadFileArgsForCall(0)

					Expect(bucketName).To(Equal("bucket-name"))
					Expect(objectPath).To(Equal("folder/file-1.3.tgz"))
					Expect(generation).To(Equal(int64(0)))
					Expect(localPath).To(Equal(filepath.Join(destDir, "file-1.3.tgz")))
				})

				It("creates a 'version' file that contains the matched version", func() {
					versionFile := filepath.Join(destDir, "version")
					Expect(versionFile).ToNot(ExistOnFilesystem())

					_, err := command.Run(destDir, request)
					Expect(err).ToNot(HaveOccurred())

					Expect(versionFile).To(ExistOnFilesystem())
					contents, err := ioutil.ReadFile(versionFile)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(contents)).To(Equal("1.3"))
				})

				It("does not creates a 'version' file if it cannot extract the version", func() {
					request.Version.Path = "folder/file.tgz"

					versionFile := filepath.Join(destDir, "version")
					Expect(versionFile).ToNot(ExistOnFilesystem())

					_, err := command.Run(destDir, request)
					Expect(err).ToNot(HaveOccurred())

					Expect(versionFile).ToNot(ExistOnFilesystem())
				})

				It("creates a 'url' file that contains the URL", func() {
					gcsClient.URLReturns("gs://bucket-name/folder/file-1.3.tgz", nil)

					urlFile := filepath.Join(destDir, "url")
					Expect(urlFile).ToNot(ExistOnFilesystem())

					_, err := command.Run(destDir, request)
					Expect(err).ToNot(HaveOccurred())

					bucketName, objectPath, generation := gcsClient.URLArgsForCall(0)
					Expect(bucketName).To(Equal("bucket-name"))
					Expect(objectPath).To(Equal("folder/file-1.3.tgz"))
					Expect(generation).To(Equal(int64(0)))

					Expect(urlFile).To(ExistOnFilesystem())
					contents, err := ioutil.ReadFile(urlFile)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(contents)).To(Equal("gs://bucket-name/folder/file-1.3.tgz"))
				})

				It("returns a response", func() {
					gcsClient.URLReturns("gs://bucket-name/folder/file-1.3.tgz", nil)

					response, err := command.Run(destDir, request)
					Expect(err).ToNot(HaveOccurred())

					Expect(response.Version.Path).To(Equal("folder/file-1.3.tgz"))
					Expect(response.Version.Generation).To(Equal(int64(0)))

					Expect(response.Metadata[0].Name).To(Equal("filename"))
					Expect(response.Metadata[0].Value).To(Equal("file-1.3.tgz"))

					Expect(response.Metadata[1].Name).To(Equal("url"))
					Expect(response.Metadata[1].Value).To(Equal("gs://bucket-name/folder/file-1.3.tgz"))
				})

				It("returns an error if download fails", func() {
					gcsClient.DownloadFileReturns(errors.New("error downloading file"))

					_, err := command.Run(destDir, request)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("error downloading file"))
				})

				It("returns an error if url fails", func() {
					gcsClient.URLReturns("", errors.New("error url"))

					_, err := command.Run(destDir, request)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("error url"))
				})
			})
		})

		Describe("with versioned_file", func() {
			BeforeEach(func() {
				request.Source.VersionedFile = "folder/version"
				request.Version.Generation = int64(12345)
			})

			It("creates the destination directory", func() {
				Expect(destDir).ToNot(ExistOnFilesystem())

				_, err := command.Run(destDir, request)
				Expect(err).ToNot(HaveOccurred())

				Expect(destDir).To(ExistOnFilesystem())
			})

			It("downloads the versioned file", func() {
				_, err := command.Run(destDir, request)
				Expect(err).ShouldNot(HaveOccurred())

				Expect(gcsClient.DownloadFileCallCount()).To(Equal(1))
				bucketName, objectPath, generation, localPath := gcsClient.DownloadFileArgsForCall(0)

				Expect(bucketName).To(Equal("bucket-name"))
				Expect(objectPath).To(Equal("folder/version"))
				Expect(generation).To(Equal(int64(12345)))
				Expect(localPath).To(Equal(filepath.Join(destDir, "version")))
			})

			It("creates a 'generation' file that contains the generation", func() {
				generationFile := filepath.Join(destDir, "generation")
				Expect(generationFile).ToNot(ExistOnFilesystem())

				_, err := command.Run(destDir, request)
				Expect(err).ToNot(HaveOccurred())

				Expect(generationFile).To(ExistOnFilesystem())
				contents, err := ioutil.ReadFile(generationFile)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(contents)).To(Equal("12345"))
			})

			It("creates a 'url' file that contains the URL", func() {
				gcsClient.URLReturns("gs://bucket-name/folder/version#12345", nil)

				urlFile := filepath.Join(destDir, "url")
				Expect(urlFile).ToNot(ExistOnFilesystem())

				_, err := command.Run(destDir, request)
				Expect(err).ToNot(HaveOccurred())

				bucketName, objectPath, generation := gcsClient.URLArgsForCall(0)
				Expect(bucketName).To(Equal("bucket-name"))
				Expect(objectPath).To(Equal("folder/version"))
				Expect(generation).To(Equal(int64(12345)))

				Expect(urlFile).To(ExistOnFilesystem())
				contents, err := ioutil.ReadFile(urlFile)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(contents)).To(Equal("gs://bucket-name/folder/version#12345"))
			})

			It("returns a response", func() {
				gcsClient.URLReturns("gs://bucket-name/folder/version#12345", nil)

				response, err := command.Run(destDir, request)
				Expect(err).ToNot(HaveOccurred())

				Expect(response.Version.Path).To(BeEmpty())
				Expect(response.Version.Generation).To(Equal(int64(12345)))

				Expect(response.Metadata[0].Name).To(Equal("filename"))
				Expect(response.Metadata[0].Value).To(Equal("version"))

				Expect(response.Metadata[1].Name).To(Equal("url"))
				Expect(response.Metadata[1].Value).To(Equal("gs://bucket-name/folder/version#12345"))
			})

			It("returns an error if download fails", func() {
				gcsClient.DownloadFileReturns(errors.New("error downloading file"))

				_, err := command.Run(destDir, request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error downloading file"))
			})

			It("returns an error if url fails", func() {
				gcsClient.URLReturns("", errors.New("error url"))

				_, err := command.Run(destDir, request)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error url"))
			})
		})
	})
})
