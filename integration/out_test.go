package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"github.com/frodenas/gcs-resource"
	"github.com/frodenas/gcs-resource/out"
	"github.com/nu7hatch/gouuid"
)

var _ = Describe("out", func() {
	var (
		err       error
		command   *exec.Cmd
		stdin     *bytes.Buffer
		session   *gexec.Session
		sourceDir string

		expectedExitStatus int
	)

	BeforeEach(func() {
		sourceDir, err = ioutil.TempDir("", "gcs_out_integration_test")
		Expect(err).ToNot(HaveOccurred())

		stdin = &bytes.Buffer{}
		expectedExitStatus = 0

		command = exec.Command(outPath, sourceDir)
		command.Stdin = stdin
	})

	AfterEach(func() {
		err := os.RemoveAll(sourceDir)
		Expect(err).ToNot(HaveOccurred())
	})

	JustBeforeEach(func() {
		session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		<-session.Exited
		Expect(session.ExitCode()).To(Equal(expectedExitStatus))
	})

	Describe("when the request is invalid", func() {
		var (
			outRequest out.OutRequest
		)

		BeforeEach(func() {
			outRequest = out.OutRequest{
				Source: gcsresource.Source{
					JSONKey: jsonKey,
					Bucket:  bucketName,
				},
				Params: out.Params{
					File: "files/file*.tgz",
				},
			}

			expectedExitStatus = 1
		})

		Context("when the bucket is not set", func() {
			BeforeEach(func() {
				outRequest.Source.Bucket = ""

				err := json.NewEncoder(stdin).Encode(outRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				Expect(session.Err).To(gbytes.Say("please specify the bucket"))
			})
		})

		Context("when the regexp and versioned_file are both set", func() {
			BeforeEach(func() {
				outRequest.Source.Regexp = "file-to-*"
				outRequest.Source.VersionedFile = "files/version"

				err := json.NewEncoder(stdin).Encode(outRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				Expect(session.Err).To(gbytes.Say("please specify either regexp or versioned_file"))
			})
		})

		Context("when the file is not set", func() {
			BeforeEach(func() {
				outRequest.Params.File = ""

				err := json.NewEncoder(stdin).Encode(outRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				Expect(session.Err).To(gbytes.Say("please specify the file"))
			})
		})
	})

	Describe("when finding the local file to upload", func() {
		var (
			outRequest out.OutRequest
		)

		BeforeEach(func() {
			outRequest = out.OutRequest{
				Source: gcsresource.Source{
					JSONKey: jsonKey,
					Bucket:  bucketName,
				},
			}

			expectedExitStatus = 1
		})

		Context("when there are no matches", func() {
			BeforeEach(func() {
				outRequest.Params.File = "file-to-upload"

				err = json.NewEncoder(stdin).Encode(outRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				Expect(session.Err).To(gbytes.Say("no matches found for pattern: file-to-upload"))
			})
		})

		Context("when there are more than one match", func() {
			BeforeEach(func() {
				err = ioutil.WriteFile(filepath.Join(sourceDir, "file-to-upload-1"), []byte("contents"), 0755)
				Expect(err).ToNot(HaveOccurred())

				err = ioutil.WriteFile(filepath.Join(sourceDir, "file-to-upload-2"), []byte("contents"), 0755)
				Expect(err).ToNot(HaveOccurred())

				outRequest.Params.File = "file-to-upload-*"

				err = json.NewEncoder(stdin).Encode(outRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				Expect(session.Err).To(gbytes.Say("more than one match found for pattern: file-to-upload-*"))
			})
		})
	})

	Describe("with regexp", func() {
		var (
			outRequest      out.OutRequest
			outResponse     out.OutResponse
			directoryPrefix string
		)

		BeforeEach(func() {
			guid, err := uuid.NewV4()
			Expect(err).ToNot(HaveOccurred())
			directoryPrefix = "out-request-files-" + guid.String()

			err = ioutil.WriteFile(filepath.Join(sourceDir, "file-to-upload"), []byte("contents"), 0755)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when the bucket is not versioned", func() {
			BeforeEach(func() {
				outRequest = out.OutRequest{
					Source: gcsresource.Source{
						JSONKey: jsonKey,
						Bucket:  bucketName,
						Regexp:  filepath.Join(directoryPrefix, "file-to-*"),
					},
					Params: out.Params{
						File:          "file-to-*",
						PredefinedACL: "publicRead",
					},
				}

				err = json.NewEncoder(stdin).Encode(outRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				err := gcsClient.DeleteObject(bucketName, filepath.Join(directoryPrefix, "file-to-upload"), int64(0))
				Expect(err).ToNot(HaveOccurred())
			})

			It("uploads the file and outputs the response", func() {
				files, err := gcsClient.BucketObjects(bucketName, directoryPrefix)
				Expect(err).ToNot(HaveOccurred())
				Expect(files).To(ConsistOf(filepath.Join(directoryPrefix, "file-to-upload")))

				reader := bytes.NewBuffer(session.Out.Contents())
				err = json.NewDecoder(reader).Decode(&outResponse)
				Expect(err).ToNot(HaveOccurred())

				url, err := gcsClient.URL(bucketName, filepath.Join(directoryPrefix, "file-to-upload"), int64(0))
				Expect(err).ToNot(HaveOccurred())

				Expect(outResponse).To(Equal(out.OutResponse{
					Version: gcsresource.Version{
						Path: filepath.Join(directoryPrefix, "file-to-upload"),
					},
					Metadata: []gcsresource.MetadataPair{
						{
							Name:  "filename",
							Value: "file-to-upload",
						},
						{
							Name:  "url",
							Value: url,
						},
					},
				}))
			})
		})

		Context("when the bucket is versioned", func() {
			BeforeEach(func() {
				outRequest = out.OutRequest{
					Source: gcsresource.Source{
						JSONKey: jsonKey,
						Bucket:  versionedBucketName,
						Regexp:  filepath.Join(directoryPrefix, "file-to-*"),
					},
					Params: out.Params{
						File: "file-to-*",
					},
				}

				err = json.NewEncoder(stdin).Encode(outRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				generations, err := gcsClient.ObjectGenerations(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload"))
				Expect(err).ToNot(HaveOccurred())
				for _, generation := range generations {
					err := gcsClient.DeleteObject(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload"), generation)
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("uploads the file and outputs the response", func() {
				files, err := gcsClient.BucketObjects(versionedBucketName, directoryPrefix)
				Expect(err).ToNot(HaveOccurred())
				Expect(files).To(ConsistOf(filepath.Join(directoryPrefix, "file-to-upload")))

				reader := bytes.NewBuffer(session.Out.Contents())
				err = json.NewDecoder(reader).Decode(&outResponse)
				Expect(err).ToNot(HaveOccurred())

				url, err := gcsClient.URL(versionedBucketName, filepath.Join(directoryPrefix, "file-to-upload"), int64(0))
				Expect(err).ToNot(HaveOccurred())

				Expect(outResponse).To(Equal(out.OutResponse{
					Version: gcsresource.Version{
						Path: filepath.Join(directoryPrefix, "file-to-upload"),
					},
					Metadata: []gcsresource.MetadataPair{
						{
							Name:  "filename",
							Value: "file-to-upload",
						},
						{
							Name:  "url",
							Value: url,
						},
					},
				}))
			})
		})

		Context("when the bucket does not exists", func() {
			BeforeEach(func() {
				outRequest = out.OutRequest{
					Source: gcsresource.Source{
						JSONKey: jsonKey,
						Bucket:  directoryPrefix,
						Regexp:  filepath.Join(directoryPrefix, "file-to-*"),
					},
					Params: out.Params{
						File: "file-to-*",
					},
				}

				expectedExitStatus = 1

				err = json.NewEncoder(stdin).Encode(outRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				Expect(session.Err).To(gbytes.Say("error running command: googleapi:"))
			})
		})
	})

	Describe("with versioned_file", func() {
		var (
			outRequest      out.OutRequest
			outResponse     out.OutResponse
			directoryPrefix string
		)

		BeforeEach(func() {
			guid, err := uuid.NewV4()
			Expect(err).ToNot(HaveOccurred())
			directoryPrefix = "out-request-files-" + guid.String()

			err = ioutil.WriteFile(filepath.Join(sourceDir, "file-to-upload"), []byte("contents"), 0755)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when the bucket is not versioned", func() {
			BeforeEach(func() {
				outRequest = out.OutRequest{
					Source: gcsresource.Source{
						JSONKey:       jsonKey,
						Bucket:        bucketName,
						VersionedFile: filepath.Join(directoryPrefix, "version"),
					},
					Params: out.Params{
						File: "file-to-*",
					},
				}

				err = json.NewEncoder(stdin).Encode(outRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				err := gcsClient.DeleteObject(bucketName, filepath.Join(directoryPrefix, "version"), int64(0))
				Expect(err).ToNot(HaveOccurred())
			})

			It("uploads the file and outputs the response", func() {
				generations, err := gcsClient.ObjectGenerations(versionedBucketName, filepath.Join(directoryPrefix, "version"))
				Expect(err).ToNot(HaveOccurred())
				Expect(len(generations)).To(Equal(0))

				reader := bytes.NewBuffer(session.Out.Contents())
				err = json.NewDecoder(reader).Decode(&outResponse)
				Expect(err).ToNot(HaveOccurred())

				url, err := gcsClient.URL(bucketName, filepath.Join(directoryPrefix, "version"), int64(0))
				Expect(err).ToNot(HaveOccurred())

				Expect(outResponse).To(Equal(out.OutResponse{
					Version: gcsresource.Version{
						Generation: "0",
					},
					Metadata: []gcsresource.MetadataPair{
						{
							Name:  "filename",
							Value: "version",
						},
						{
							Name:  "url",
							Value: url,
						},
					},
				}))
			})
		})

		Context("when the bucket is versioned", func() {
			BeforeEach(func() {
				outRequest = out.OutRequest{
					Source: gcsresource.Source{
						JSONKey:       jsonKey,
						Bucket:        versionedBucketName,
						VersionedFile: filepath.Join(directoryPrefix, "version"),
					},
					Params: out.Params{
						File:          "file-to-*",
						PredefinedACL: "publicRead",
					},
				}

				err = json.NewEncoder(stdin).Encode(outRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				generations, err := gcsClient.ObjectGenerations(versionedBucketName, filepath.Join(directoryPrefix, "version"))
				Expect(err).ToNot(HaveOccurred())
				for _, generation := range generations {
					err := gcsClient.DeleteObject(versionedBucketName, filepath.Join(directoryPrefix, "version"), generation)
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("uploads the file and outputs the response", func() {
				generations, err := gcsClient.ObjectGenerations(versionedBucketName, filepath.Join(directoryPrefix, "version"))
				Expect(err).ToNot(HaveOccurred())
				Expect(len(generations)).To(Equal(1))

				reader := bytes.NewBuffer(session.Out.Contents())
				err = json.NewDecoder(reader).Decode(&outResponse)
				Expect(err).ToNot(HaveOccurred())

				url, err := gcsClient.URL(versionedBucketName, filepath.Join(directoryPrefix, "version"), generations[0])
				Expect(err).ToNot(HaveOccurred())

				Expect(outResponse).To(Equal(out.OutResponse{
					Version: gcsresource.Version{
						Generation: fmt.Sprintf("%d", generations[0]),
					},
					Metadata: []gcsresource.MetadataPair{
						{
							Name:  "filename",
							Value: "version",
						},
						{
							Name:  "url",
							Value: url,
						},
					},
				}))
			})
		})

		Context("when the bucket does not exists", func() {
			BeforeEach(func() {
				outRequest = out.OutRequest{
					Source: gcsresource.Source{
						JSONKey:       jsonKey,
						Bucket:        directoryPrefix,
						VersionedFile: filepath.Join(directoryPrefix, "version"),
					},
					Params: out.Params{
						File: "file-to-*",
					},
				}

				expectedExitStatus = 1

				err = json.NewEncoder(stdin).Encode(outRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				Expect(session.Err).To(gbytes.Say("error running command: googleapi:"))
			})
		})
	})
})
