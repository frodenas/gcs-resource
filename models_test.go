package gcsresource_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/frodenas/gcs-resource"
)

var _ = Describe("GCS Resource model", func() {
	Describe("IsValid", func() {
		var source gcsresource.Source

		BeforeEach(func() {
			source = gcsresource.Source{}
		})

		Context("when bucket is not specified", func() {
			JustBeforeEach(func() {
				source.VersionedFile = "some/file"
			})

			It("returns false with a useful message", func() {
				valid, message := source.IsValid()
				Expect(valid).To(BeFalse())
				Expect(message).To(Equal("please specify the bucket"))
			})
		})

		Context("when both regexp and versioned_file are provided", func() {
			JustBeforeEach(func() {
				source.Regexp = "missing-(.*).tgz"
				source.VersionedFile = "some/file"
			})

			It("returns false with a useful message", func() {
				valid, message := source.IsValid()
				Expect(valid).To(BeFalse())
				Expect(message).To(Equal("please specify the bucket"))
			})
		})

		DescribeTable("initial versions",
			func(source gcsresource.Source, expectedValid bool, expectedMessage string) {
				valid, message := source.IsValid()
				Expect(valid).To(Equal(expectedValid))
				Expect(message).To(Equal(expectedMessage))
			},
			Entry("initial version not an int64",
				gcsresource.Source{Bucket: "some-bucket", InitialVersion: "zero"},
				false,
				"if set, initial_version must be an int64",
			),
			Entry("text and binary both set",
				gcsresource.Source{Bucket: "some-bucket", InitialContentText: "foo", InitialContentBinary: "Zm9v"},
				false,
				"use initial_content_text or initial_content_binary but not both",
			),
			Entry("initial_content_binary not base64",
				gcsresource.Source{Bucket: "some-bucket", InitialContentBinary: "not base64"},
				false,
				"initial_content_binary could not be decoded to base64",
			),
			Entry("regexp and initial_version",
				gcsresource.Source{Bucket: "some-bucket", Regexp: "foo-(.*).tgz", InitialVersion: "100"},
				false,
				"use initial_path when regexp is set",
			),
			Entry("versioned_file and initial_path",
				gcsresource.Source{Bucket: "some-bucket", VersionedFile: "foo.tgz", InitialPath: "foo-100.tgz"},
				false,
				"use initial_version when versioned_file is set",
			),
			Entry("text but not regexp or versioned_file",
				gcsresource.Source{Bucket: "some-bucket", InitialContentText: "foo"},
				false,
				"use initial_version or initial_path when initial content is set",
			),
			Entry("binary but not regexp or versioned_file",
				gcsresource.Source{Bucket: "some-bucket", InitialContentBinary: "Zm9v"},
				false,
				"use initial_version or initial_path when initial content is set",
			),
		)
	})
})
