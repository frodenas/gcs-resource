package integration_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"github.com/frodenas/gcs-resource"
	"github.com/frodenas/gcs-resource/in"
	"github.com/nu7hatch/gouuid"
)

var _ = Describe("in", func() {
	var (
		err     error
		command *exec.Cmd
		stdin   *bytes.Buffer
		session *gexec.Session
		destDir string

		expectedExitStatus int
	)

	BeforeEach(func() {
		destDir, err = ioutil.TempDir("", "gcs_in_integration_test")
		Expect(err).ToNot(HaveOccurred())

		stdin = &bytes.Buffer{}
		expectedExitStatus = 0

		command = exec.Command(inPath, destDir)
		command.Stdin = stdin
	})

	AfterEach(func() {
		err := os.RemoveAll(destDir)
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
			inRequest in.InRequest
		)

		BeforeEach(func() {
			inRequest = in.InRequest{
				Source: gcsresource.Source{
					JSONKey: jsonKey,
					Bucket:  bucketName,
				},
			}

			expectedExitStatus = 1
		})

		Context("when the bucket is not set", func() {
			BeforeEach(func() {
				inRequest.Source.Bucket = ""

				err := json.NewEncoder(stdin).Encode(inRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				Expect(session.Err).To(gbytes.Say("please specify the bucket"))
			})
		})

		Context("when the regexp and versioned_file are both set", func() {
			BeforeEach(func() {
				inRequest.Source.Regexp = "file-to-*"
				inRequest.Source.VersionedFile = "files/version"

				err := json.NewEncoder(stdin).Encode(inRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				Expect(session.Err).To(gbytes.Say("please specify either regexp or versioned_file"))
			})
		})
	})

	Describe("with regexp", func() {
		var (
			inRequest       in.InRequest
			inResponse      in.InResponse
			directoryPrefix string
		)

		BeforeEach(func() {
			guid, err := uuid.NewV4()
			Expect(err).ToNot(HaveOccurred())
			directoryPrefix = "in-request-files-" + guid.String()

			tempFile, err := ioutil.TempFile("", "file-to-upload")
			Expect(err).ToNot(HaveOccurred())
			tempFile.Close()

			err = ioutil.WriteFile(tempFile.Name(), []byte("file-to-download-1"), 0755)
			Expect(err).ToNot(HaveOccurred())

			_, err = gcsClient.UploadFile(bucketName, filepath.Join(directoryPrefix, "file-to-download-1"), tempFile.Name())
			Expect(err).ToNot(HaveOccurred())

			err = ioutil.WriteFile(tempFile.Name(), []byte("file-to-download-2"), 0755)
			Expect(err).ToNot(HaveOccurred())

			_, err = gcsClient.UploadFile(bucketName, filepath.Join(directoryPrefix, "file-to-download-2"), tempFile.Name())
			Expect(err).ToNot(HaveOccurred())

			err = ioutil.WriteFile(tempFile.Name(), []byte("file-to-download-3"), 0755)
			Expect(err).ToNot(HaveOccurred())

			_, err = gcsClient.UploadFile(bucketName, filepath.Join(directoryPrefix, "file-to-download-3"), tempFile.Name())
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			err = gcsClient.DeleteObject(bucketName, filepath.Join(directoryPrefix, "file-to-download-1"), 0)
			Expect(err).ToNot(HaveOccurred())

			err = gcsClient.DeleteObject(bucketName, filepath.Join(directoryPrefix, "file-to-download-2"), 0)
			Expect(err).ToNot(HaveOccurred())

			err = gcsClient.DeleteObject(bucketName, filepath.Join(directoryPrefix, "file-to-download-3"), 0)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when there is a path", func() {
			Context("when the path exists", func() {
				BeforeEach(func() {
					inRequest = in.InRequest{
						Source: gcsresource.Source{
							JSONKey: jsonKey,
							Bucket:  bucketName,
							Regexp:  filepath.Join(directoryPrefix, "file-to-download-(.*)"),
						},
						Version: gcsresource.Version{
							Path: filepath.Join(directoryPrefix, "file-to-download-1"),
						},
					}

					err = json.NewEncoder(stdin).Encode(inRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				It("downloads the file and outputs the response", func() {
					reader := bytes.NewBuffer(session.Out.Contents())
					err = json.NewDecoder(reader).Decode(&inResponse)
					Expect(err).ToNot(HaveOccurred())

					url, err := gcsClient.URL(bucketName, filepath.Join(directoryPrefix, "file-to-download-1"), int64(0))
					Expect(err).ToNot(HaveOccurred())

					Expect(inResponse).To(Equal(in.InResponse{
						Version: gcsresource.Version{
							Path: filepath.Join(directoryPrefix, "file-to-download-1"),
						},
						Metadata: []gcsresource.MetadataPair{
							{
								Name:  "filename",
								Value: "file-to-download-1",
							},
							{
								Name:  "url",
								Value: url,
							},
						},
					}))

					Expect(filepath.Join(destDir, "file-to-download-1")).To(BeARegularFile())
					contents, err := ioutil.ReadFile(filepath.Join(destDir, "file-to-download-1"))
					Expect(err).ToNot(HaveOccurred())
					Expect(contents).To(Equal([]byte("file-to-download-1")))

					Expect(filepath.Join(destDir, "version")).To(BeARegularFile())
					versionContents, err := ioutil.ReadFile(filepath.Join(destDir, "version"))
					Expect(err).ToNot(HaveOccurred())
					Expect(versionContents).To(Equal([]byte("1")))

					Expect(filepath.Join(destDir, "url")).To(BeARegularFile())
					urlContents, err := ioutil.ReadFile(filepath.Join(destDir, "url"))
					Expect(err).ToNot(HaveOccurred())
					Expect(urlContents).To(Equal([]byte(url)))
				})
			})

			Context("when the path does not exists", func() {
				BeforeEach(func() {
					inRequest = in.InRequest{
						Source: gcsresource.Source{
							JSONKey: jsonKey,
							Bucket:  bucketName,
							Regexp:  filepath.Join(directoryPrefix, "file-to-download-(.*)"),
						},
						Version: gcsresource.Version{
							Path: filepath.Join(directoryPrefix, "file-to-download-missing"),
						},
					}

					expectedExitStatus = 1

					err = json.NewEncoder(stdin).Encode(inRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns an error", func() {
					Expect(session.Err).To(gbytes.Say("error running command: googleapi:"))
				})
			})
		})

		Context("when there is not a path", func() {
			Context("when there are matches", func() {
				BeforeEach(func() {
					inRequest = in.InRequest{
						Source: gcsresource.Source{
							JSONKey: jsonKey,
							Bucket:  bucketName,
							Regexp:  filepath.Join(directoryPrefix, "file-to-download-(.*)"),
						},
					}

					err = json.NewEncoder(stdin).Encode(inRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				It("downloads the latest file version and outputs the response", func() {
					reader := bytes.NewBuffer(session.Out.Contents())
					err = json.NewDecoder(reader).Decode(&inResponse)
					Expect(err).ToNot(HaveOccurred())

					url, err := gcsClient.URL(bucketName, filepath.Join(directoryPrefix, "file-to-download-3"), int64(0))
					Expect(err).ToNot(HaveOccurred())

					Expect(inResponse).To(Equal(in.InResponse{
						Version: gcsresource.Version{
							Path: filepath.Join(directoryPrefix, "file-to-download-3"),
						},
						Metadata: []gcsresource.MetadataPair{
							{
								Name:  "filename",
								Value: "file-to-download-3",
							},
							{
								Name:  "url",
								Value: url,
							},
						},
					}))

					Expect(filepath.Join(destDir, "file-to-download-3")).To(BeARegularFile())
					contents, err := ioutil.ReadFile(filepath.Join(destDir, "file-to-download-3"))
					Expect(err).ToNot(HaveOccurred())
					Expect(contents).To(Equal([]byte("file-to-download-3")))

					Expect(filepath.Join(destDir, "version")).To(BeARegularFile())
					versionContents, err := ioutil.ReadFile(filepath.Join(destDir, "version"))
					Expect(err).ToNot(HaveOccurred())
					Expect(versionContents).To(Equal([]byte("3")))

					Expect(filepath.Join(destDir, "url")).To(BeARegularFile())
					urlContents, err := ioutil.ReadFile(filepath.Join(destDir, "url"))
					Expect(err).ToNot(HaveOccurred())
					Expect(urlContents).To(Equal([]byte(url)))

				})
			})

			Context("when there are no matches", func() {
				BeforeEach(func() {
					inRequest = in.InRequest{
						Source: gcsresource.Source{
							JSONKey: jsonKey,
							Bucket:  bucketName,
							Regexp:  filepath.Join(directoryPrefix, "file-to-upload-(.*)"),
						},
					}

					expectedExitStatus = 1

					err = json.NewEncoder(stdin).Encode(inRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns an error", func() {
					Expect(session.Err).To(gbytes.Say("no extractions could be found - is your regexp correct?"))
				})
			})
		})
	})

	Describe("with versioned_file", func() {
		var (
			inRequest       in.InRequest
			inResponse      in.InResponse
			directoryPrefix string
			generation1     int64
			generation2     int64
			generation3     int64
		)

		BeforeEach(func() {
			guid, err := uuid.NewV4()
			Expect(err).ToNot(HaveOccurred())
			directoryPrefix = "out-request-files-" + guid.String()
		})

		Context("when the bucket is versioned", func() {
			Context("when the versioned file exists", func() {
				BeforeEach(func() {
					tempFile, err := ioutil.TempFile("", "file-to-upload")
					Expect(err).ToNot(HaveOccurred())
					tempFile.Close()

					err = ioutil.WriteFile(tempFile.Name(), []byte("generation-1"), 0755)
					Expect(err).ToNot(HaveOccurred())

					generation1, err = gcsClient.UploadFile(versionedBucketName, filepath.Join(directoryPrefix, "version"), tempFile.Name())
					Expect(err).ToNot(HaveOccurred())

					err = ioutil.WriteFile(tempFile.Name(), []byte("generation-2"), 0755)
					Expect(err).ToNot(HaveOccurred())

					generation2, err = gcsClient.UploadFile(versionedBucketName, filepath.Join(directoryPrefix, "version"), tempFile.Name())
					Expect(err).ToNot(HaveOccurred())

					err = ioutil.WriteFile(tempFile.Name(), []byte("generation-3"), 0755)
					Expect(err).ToNot(HaveOccurred())

					generation3, err = gcsClient.UploadFile(versionedBucketName, filepath.Join(directoryPrefix, "version"), tempFile.Name())
					Expect(err).ToNot(HaveOccurred())

					inRequest = in.InRequest{
						Source: gcsresource.Source{
							JSONKey:       jsonKey,
							Bucket:        versionedBucketName,
							VersionedFile: filepath.Join(directoryPrefix, "version"),
						},
						Version: gcsresource.Version{
							Generation: generation2,
						},
					}

					err = json.NewEncoder(stdin).Encode(inRequest)
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

				It("downloads the file and outputs the response", func() {
					reader := bytes.NewBuffer(session.Out.Contents())
					err = json.NewDecoder(reader).Decode(&inResponse)
					Expect(err).ToNot(HaveOccurred())

					url, err := gcsClient.URL(versionedBucketName, filepath.Join(directoryPrefix, "version"), generation2)
					Expect(err).ToNot(HaveOccurred())

					Expect(inResponse).To(Equal(in.InResponse{
						Version: gcsresource.Version{
							Generation: generation2,
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

					Expect(filepath.Join(destDir, "version")).To(BeARegularFile())
					contents, err := ioutil.ReadFile(filepath.Join(destDir, "version"))
					Expect(err).ToNot(HaveOccurred())
					Expect(contents).To(Equal([]byte("generation-2")))

					Expect(filepath.Join(destDir, "generation")).To(BeARegularFile())
					versionContents, err := ioutil.ReadFile(filepath.Join(destDir, "generation"))
					Expect(err).ToNot(HaveOccurred())
					Expect(versionContents).To(Equal([]byte(strconv.FormatInt(generation2, 10))))

					Expect(filepath.Join(destDir, "url")).To(BeARegularFile())
					urlContents, err := ioutil.ReadFile(filepath.Join(destDir, "url"))
					Expect(err).ToNot(HaveOccurred())
					Expect(urlContents).To(Equal([]byte(url)))
				})
			})

			Context("when the versioned file does not exists", func() {
				BeforeEach(func() {
					inRequest = in.InRequest{
						Source: gcsresource.Source{
							JSONKey:       jsonKey,
							Bucket:        versionedBucketName,
							VersionedFile: filepath.Join(directoryPrefix, "missing"),
						},
						Version: gcsresource.Version{
							Generation: generation2,
						},
					}

					expectedExitStatus = 1

					err = json.NewEncoder(stdin).Encode(inRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns an error", func() {
					Expect(session.Err).To(gbytes.Say("error running command: googleapi:"))
				})
			})
		})

		Context("when the bucket is not versioned", func() {
			Context("when the versioned file exists", func() {
				BeforeEach(func() {
					inRequest = in.InRequest{
						Source: gcsresource.Source{
							JSONKey:       jsonKey,
							Bucket:        bucketName,
							VersionedFile: filepath.Join(directoryPrefix, "version"),
						},
						Version: gcsresource.Version{
							Generation: int64(12345),
						},
					}

					expectedExitStatus = 1

					err = json.NewEncoder(stdin).Encode(inRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns an error", func() {
					Expect(session.Err).To(gbytes.Say("bucket is not versioned"))
				})
			})

			Context("when the versioned file does not exists", func() {
				BeforeEach(func() {
					inRequest = in.InRequest{
						Source: gcsresource.Source{
							JSONKey:       jsonKey,
							Bucket:        bucketName,
							VersionedFile: filepath.Join(directoryPrefix, "missing"),
						},
						Version: gcsresource.Version{
							Generation: 0,
						},
					}

					expectedExitStatus = 1

					err = json.NewEncoder(stdin).Encode(inRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns an error", func() {
					Expect(session.Err).To(gbytes.Say("error running command: googleapi:"))
				})
			})
		})
	})
})
