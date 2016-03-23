package check_test

import (
	"errors"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/frodenas/gcs-resource"
	"github.com/frodenas/gcs-resource/fakes"

	. "github.com/frodenas/gcs-resource/check"
)

var _ = Describe("Check Command", func() {
	Describe("running the command", func() {
		var (
			err     error
			tmpPath string
			request CheckRequest

			gcsClient *fakes.FakeGCSClient
			command   *CheckCommand
		)

		BeforeEach(func() {
			tmpPath, err = ioutil.TempDir("", "check_command")
			Expect(err).ToNot(HaveOccurred())

			request = CheckRequest{
				Source: gcsresource.Source{
					Project: "project",
					Bucket:  "bucket-name",
				},
			}

			gcsClient = &fakes.FakeGCSClient{}
			command = NewCheckCommand(gcsClient)
		})

		AfterEach(func() {
			err := os.RemoveAll(tmpPath)
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("when the request is invalid", func() {
			Context("when the project is not set", func() {
				BeforeEach(func() {
					request.Source.Project = ""
				})

				It("returns an error", func() {
					_, err := command.Run(request)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("please specify the project"))
				})
			})

			Context("when the bucket is not set", func() {
				BeforeEach(func() {
					request.Source.Bucket = ""
				})

				It("returns an error", func() {
					_, err := command.Run(request)
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
					_, err := command.Run(request)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("please specify either regexp or versioned_file"))
				})
			})
		})

		Describe("with regexp", func() {
			BeforeEach(func() {
				request.Source.Regexp = "folder/file-(.*).tgz"

				gcsClient.BucketObjectsReturns([]string{
					"folder/file-0.0.1.tgz",
					"folder/file-2.33.333.tgz",
					"folder/file-2.4.3.tgz",
					"folder/file-3.53.tgz",
				}, nil)
			})

			Context("when there is no previous version", func() {
				It("includes the latest version", func() {
					response, err := command.Run(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(response).To(HaveLen(1))
					Expect(response).To(ConsistOf(
						gcsresource.Version{
							Path: "folder/file-3.53.tgz",
						},
					))
				})
			})

			Context("when there is a previous version", func() {
				BeforeEach(func() {
					request.Version.Path = "folder/file-2.4.3.tgz"
				})

				It("includes the latest versions", func() {
					response, err := command.Run(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(response).To(HaveLen(2))
					Expect(response).To(ConsistOf(
						gcsresource.Version{
							Path: "folder/file-2.33.333.tgz",
						},
						gcsresource.Version{
							Path: "folder/file-3.53.tgz",
						},
					))
				})

				Context("when the regex does not match the previous version", func() {
					BeforeEach(func() {
						request.Version.Path = "folder/fake-0.0.1.tgz"
					})

					It("returns the latest version", func() {
						response, err := command.Run(request)
						Expect(err).ToNot(HaveOccurred())

						Expect(response).To(HaveLen(1))
						Expect(response).To(ConsistOf(
							gcsresource.Version{
								Path: "folder/file-3.53.tgz",
							},
						))
					})
				})
			})

			Context("when the bucket does not contains objects", func() {
				BeforeEach(func() {
					gcsClient.BucketObjectsReturns([]string{}, nil)
				})

				It("does not explode", func() {
					response, err := command.Run(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(response).To(HaveLen(0))
				})
			})

			Context("when the regexp does not match anything", func() {
				BeforeEach(func() {
					request.Source.Regexp = "folder/missing-(.*).tgz"
				})

				It("does not explode", func() {
					response, err := command.Run(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(response).To(HaveLen(0))
				})
			})
		})

		Describe("with versioned_file", func() {
			BeforeEach(func() {
				request.Source.VersionedFile = "folder/version"

				gcsClient.ObjectGenerationsReturns([]int64{1234, 789, 123, 456}, nil)
			})

			Context("when there is no previous version", func() {
				It("includes the latest version", func() {
					response, err := command.Run(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(response).To(HaveLen(1))
					Expect(response).To(ConsistOf(
						gcsresource.Version{
							Generation: 1234,
						},
					))
				})
			})

			Context("when there is a previous version", func() {
				BeforeEach(func() {
					request.Version.Generation = 456
				})

				It("includes the latest versions", func() {
					response, err := command.Run(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(response).To(HaveLen(2))
					Expect(response).To(ConsistOf(
						gcsresource.Version{
							Generation: 789,
						},
						gcsresource.Version{
							Generation: 1234,
						},
					))
				})

				Context("when there is not any version greater than the previous version", func() {
					BeforeEach(func() {
						request.Version.Generation = 9999
					})

					It("returns the latest version", func() {
						response, err := command.Run(request)
						Expect(err).ToNot(HaveOccurred())

						Expect(response).To(HaveLen(1))
						Expect(response).To(ConsistOf(
							gcsresource.Version{
								Generation: 1234,
							},
						))
					})
				})
			})

			Context("when the file does not have generations", func() {
				BeforeEach(func() {
					gcsClient.ObjectGenerationsReturns([]int64{}, nil)
				})

				It("does not explode", func() {
					response, err := command.Run(request)
					Expect(err).ToNot(HaveOccurred())

					Expect(response).To(HaveLen(0))
				})
			})

			Context("when object generations fails", func() {
				BeforeEach(func() {
					gcsClient.ObjectGenerationsReturns([]int64{}, errors.New("error object generations"))
				})

				It("returns an error", func() {
					_, err := command.Run(request)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("error object generations"))
				})
			})
		})
	})
})
