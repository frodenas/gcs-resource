package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	gcsresource "github.com/frodenas/gcs-resource"
	"github.com/frodenas/gcs-resource/in"
	uuid "github.com/nu7hatch/gouuid"
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

			tempFile, err := ioutil.TempFile("", directoryPrefix)
			Expect(err).ToNot(HaveOccurred())
			tempFile.Close()

			err = ioutil.WriteFile(tempFile.Name(), []byte("file-to-download-1"), 0755)
			Expect(err).ToNot(HaveOccurred())

			_, err = gcsClient.UploadFile(bucketName, filepath.Join(directoryPrefix, "file-to-download-1"), "", tempFile.Name(), "", "")
			Expect(err).ToNot(HaveOccurred())

			err = ioutil.WriteFile(tempFile.Name(), []byte("file-to-download-2"), 0755)
			Expect(err).ToNot(HaveOccurred())

			_, err = gcsClient.UploadFile(bucketName, filepath.Join(directoryPrefix, "file-to-download-2"), "", tempFile.Name(), "", "")
			Expect(err).ToNot(HaveOccurred())

			err = ioutil.WriteFile(tempFile.Name(), []byte("file-to-download-3"), 0755)
			Expect(err).ToNot(HaveOccurred())

			_, err = gcsClient.UploadFile(bucketName, filepath.Join(directoryPrefix, "file-to-download-3"), "", tempFile.Name(), "", "")
			Expect(err).ToNot(HaveOccurred())

			err = os.Remove(tempFile.Name())
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

			Context("when the path matches the initial path", func() {
				BeforeEach(func() {
					inRequest = in.InRequest{
						Source: gcsresource.Source{
							JSONKey:            jsonKey,
							Bucket:             bucketName,
							Regexp:             filepath.Join(directoryPrefix, "file-to-download-(.*)"),
							InitialPath:        filepath.Join(directoryPrefix, "file-to-download-0.0.0"),
							InitialContentText: "foo",
						},
						Version: gcsresource.Version{
							Path: filepath.Join(directoryPrefix, "file-to-download-0.0.0"),
						},
					}

					err = json.NewEncoder(stdin).Encode(inRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				It("uses the initial content", func() {
					reader := bytes.NewBuffer(session.Out.Contents())
					err = json.NewDecoder(reader).Decode(&inResponse)
					Expect(err).ToNot(HaveOccurred())

					Expect(inResponse).To(Equal(in.InResponse{
						Version: gcsresource.Version{
							Path: filepath.Join(directoryPrefix, "file-to-download-0.0.0"),
						},
						Metadata: []gcsresource.MetadataPair{
							{
								Name:  "filename",
								Value: "file-to-download-0.0.0",
							},
						},
					}))

					Expect(filepath.Join(destDir, "file-to-download-0.0.0")).To(BeARegularFile())
					contents, err := ioutil.ReadFile(filepath.Join(destDir, "file-to-download-0.0.0"))
					Expect(err).ToNot(HaveOccurred())
					Expect(contents).To(Equal([]byte("foo")))

					Expect(filepath.Join(destDir, "version")).To(BeARegularFile())
					versionContents, err := ioutil.ReadFile(filepath.Join(destDir, "version"))
					Expect(err).ToNot(HaveOccurred())
					Expect(versionContents).To(Equal([]byte("0.0.0")))

					Expect(filepath.Join(destDir, "url")).ToNot(BeARegularFile())
				})
			})

			Context("when path exists and 'skip_download' is specified as a source param", func() {
				BeforeEach(func() {
					tempDir, err := ioutil.TempDir("", directoryPrefix)
					Expect(err).ToNot(HaveOccurred())

					tempFilePath := filepath.Join(tempDir, "file-to-download.txt")
					tempTarballPath := filepath.Join(tempDir, "file-to-download.tgz")

					err = ioutil.WriteFile(tempFilePath, []byte("file-to-download-4"), 0600)
					Expect(err).ToNot(HaveOccurred())

					command := exec.Command("tar", "czf", tempTarballPath, "-C", tempDir, "file-to-download.txt")
					session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Eventually(session).Should(gexec.Exit(0))

					_, err = gcsClient.UploadFile(bucketName, filepath.Join(directoryPrefix, "file-to-download.tgz"), "", tempTarballPath, "", "")
					Expect(err).ToNot(HaveOccurred())

					err = os.RemoveAll(tempDir)
					Expect(err).NotTo(HaveOccurred())

					inRequest = in.InRequest{
						Source: gcsresource.Source{
							JSONKey:      jsonKey,
							Bucket:       bucketName,
							Regexp:       filepath.Join(directoryPrefix, "file-to-download-(.*)"),
							SkipDownload: true,
						},
						Version: gcsresource.Version{
							Path: filepath.Join(directoryPrefix, "file-to-download.tgz"),
						},
					}

					err = json.NewEncoder(stdin).Encode(inRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					err = gcsClient.DeleteObject(bucketName, filepath.Join(directoryPrefix, "file-to-download.tgz"), 0)
					Expect(err).ToNot(HaveOccurred())
				})

				It("skips the download of the file, but still outputs the response", func() {
					reader := bytes.NewBuffer(session.Out.Contents())
					err = json.NewDecoder(reader).Decode(&inResponse)
					Expect(err).ToNot(HaveOccurred())

					url, err := gcsClient.URL(bucketName, filepath.Join(directoryPrefix, "file-to-download.tgz"), int64(0))
					Expect(err).ToNot(HaveOccurred())

					Expect(inResponse).To(Equal(in.InResponse{
						Version: gcsresource.Version{
							Path: filepath.Join(directoryPrefix, "file-to-download.tgz"),
						},
						Metadata: []gcsresource.MetadataPair{
							{
								Name:  "filename",
								Value: "file-to-download.tgz",
							},
							{
								Name:  "url",
								Value: url,
							},
						},
					}))

					Expect(filepath.Join(destDir, "file-to-download.tgz")).NotTo(BeARegularFile())

					Expect(filepath.Join(destDir, "url")).To(BeARegularFile())
					urlContents, err := ioutil.ReadFile(filepath.Join(destDir, "url"))
					Expect(err).ToNot(HaveOccurred())
					Expect(urlContents).To(Equal([]byte(url)))
				})
			})

			Context("when path exists and 'unpack' is specified", func() {
				BeforeEach(func() {
					tempDir, err := ioutil.TempDir("", directoryPrefix)
					Expect(err).ToNot(HaveOccurred())

					tempFilePath := filepath.Join(tempDir, "file-to-download.txt")
					tempTarballPath := filepath.Join(tempDir, "file-to-download.tgz")

					err = ioutil.WriteFile(tempFilePath, []byte("file-to-download-4"), 0600)
					Expect(err).ToNot(HaveOccurred())

					command := exec.Command("tar", "czf", tempTarballPath, "-C", tempDir, "file-to-download.txt")
					session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Eventually(session).Should(gexec.Exit(0))

					_, err = gcsClient.UploadFile(bucketName, filepath.Join(directoryPrefix, "file-to-download.tgz"), "", tempTarballPath, "", "")
					Expect(err).ToNot(HaveOccurred())

					err = os.RemoveAll(tempDir)
					Expect(err).NotTo(HaveOccurred())

					inRequest = in.InRequest{
						Source: gcsresource.Source{
							JSONKey: jsonKey,
							Bucket:  bucketName,
							Regexp:  filepath.Join(directoryPrefix, "file-to-download-(.*)"),
						},
						Version: gcsresource.Version{
							Path: filepath.Join(directoryPrefix, "file-to-download.tgz"),
						},
						Params: in.Params{
							Unpack: true,
						},
					}

					err = json.NewEncoder(stdin).Encode(inRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					err = gcsClient.DeleteObject(bucketName, filepath.Join(directoryPrefix, "file-to-download.tgz"), 0)
					Expect(err).ToNot(HaveOccurred())
				})

				It("downloads the file, decompresses it, and outputs the response", func() {
					reader := bytes.NewBuffer(session.Out.Contents())
					err = json.NewDecoder(reader).Decode(&inResponse)
					Expect(err).ToNot(HaveOccurred())

					url, err := gcsClient.URL(bucketName, filepath.Join(directoryPrefix, "file-to-download.tgz"), int64(0))
					Expect(err).ToNot(HaveOccurred())

					Expect(inResponse).To(Equal(in.InResponse{
						Version: gcsresource.Version{
							Path: filepath.Join(directoryPrefix, "file-to-download.tgz"),
						},
						Metadata: []gcsresource.MetadataPair{
							{
								Name:  "filename",
								Value: "file-to-download.tgz",
							},
							{
								Name:  "url",
								Value: url,
							},
						},
					}))

					Expect(filepath.Join(destDir, "file-to-download.tgz")).To(BeARegularFile())

					Expect(filepath.Join(destDir, "file-to-download.txt")).To(BeARegularFile())
					contents, err := ioutil.ReadFile(filepath.Join(destDir, "file-to-download.txt"))
					Expect(err).ToNot(HaveOccurred())
					Expect(contents).To(Equal([]byte("file-to-download-4")))

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
					Expect(session.Err).To(gbytes.Say("error running command: storage: object doesn't exist"))
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
			generation2     int64
		)

		BeforeEach(func() {
			guid, err := uuid.NewV4()
			Expect(err).ToNot(HaveOccurred())
			directoryPrefix = "in-request-files-" + guid.String()
		})

		Context("when the bucket is versioned", func() {
			Context("when the versioned file exists", func() {
				BeforeEach(func() {
					tempFile, err := ioutil.TempFile("", directoryPrefix)
					Expect(err).ToNot(HaveOccurred())
					tempFile.Close()

					err = ioutil.WriteFile(tempFile.Name(), []byte("generation-1"), 0755)
					Expect(err).ToNot(HaveOccurred())

					_, err = gcsClient.UploadFile(versionedBucketName, filepath.Join(directoryPrefix, "version"), "", tempFile.Name(), "", "")
					Expect(err).ToNot(HaveOccurred())

					err = ioutil.WriteFile(tempFile.Name(), []byte("generation-2"), 0755)
					Expect(err).ToNot(HaveOccurred())

					generation2, err = gcsClient.UploadFile(versionedBucketName, filepath.Join(directoryPrefix, "version"), "", tempFile.Name(), "", "")
					Expect(err).ToNot(HaveOccurred())

					err = ioutil.WriteFile(tempFile.Name(), []byte("generation-3"), 0755)
					Expect(err).ToNot(HaveOccurred())

					_, err = gcsClient.UploadFile(versionedBucketName, filepath.Join(directoryPrefix, "version"), "", tempFile.Name(), "", "")
					Expect(err).ToNot(HaveOccurred())

					err = os.Remove(tempFile.Name())
					Expect(err).ToNot(HaveOccurred())

					inRequest = in.InRequest{
						Source: gcsresource.Source{
							JSONKey:       jsonKey,
							Bucket:        versionedBucketName,
							VersionedFile: filepath.Join(directoryPrefix, "version"),
						},
						Version: gcsresource.Version{
							Generation: fmt.Sprintf("%d", generation2),
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
							Generation: fmt.Sprintf("%d", generation2),
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

			Context("when the versioned file exists and 'skip_download' is specified as a get param", func() {
				var (
					generation int64
				)

				BeforeEach(func() {
					tempDir, err := ioutil.TempDir("", directoryPrefix)
					Expect(err).ToNot(HaveOccurred())

					tempFilePath := filepath.Join(tempDir, "version.txt")
					tempTarballPath := filepath.Join(tempDir, "version.tgz")

					err = ioutil.WriteFile(tempFilePath, []byte("generation-4"), 0600)
					Expect(err).ToNot(HaveOccurred())

					command := exec.Command("tar", "czf", tempTarballPath, "-C", tempDir, "version.txt")
					session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Eventually(session).Should(gexec.Exit(0))

					generation, err = gcsClient.UploadFile(versionedBucketName, filepath.Join(directoryPrefix, "version.tgz"), "", tempTarballPath, "", "")
					Expect(err).ToNot(HaveOccurred())

					err = os.RemoveAll(tempDir)
					Expect(err).NotTo(HaveOccurred())

					inRequest = in.InRequest{
						Source: gcsresource.Source{
							JSONKey:       jsonKey,
							Bucket:        versionedBucketName,
							VersionedFile: filepath.Join(directoryPrefix, "version.tgz"),
						},
						Version: gcsresource.Version{
							Generation: fmt.Sprintf("%d", generation),
						},
						Params: in.Params{
							SkipDownload: "true",
						},
					}

					err = json.NewEncoder(stdin).Encode(inRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					err := gcsClient.DeleteObject(versionedBucketName, filepath.Join(directoryPrefix, "version.tgz"), generation)
					Expect(err).ToNot(HaveOccurred())
				})

				It("skips the download of the file, but still outputs the response", func() {
					reader := bytes.NewBuffer(session.Out.Contents())
					err = json.NewDecoder(reader).Decode(&inResponse)
					Expect(err).ToNot(HaveOccurred())

					url, err := gcsClient.URL(versionedBucketName, filepath.Join(directoryPrefix, "version.tgz"), generation)
					Expect(err).ToNot(HaveOccurred())

					Expect(inResponse).To(Equal(in.InResponse{
						Version: gcsresource.Version{
							Generation: fmt.Sprintf("%d", generation),
						},
						Metadata: []gcsresource.MetadataPair{
							{
								Name:  "filename",
								Value: "version.tgz",
							},
							{
								Name:  "url",
								Value: url,
							},
						},
					}))

					Expect(filepath.Join(destDir, "version.txt")).NotTo(BeARegularFile())

					Expect(filepath.Join(destDir, "generation")).To(BeARegularFile())
					versionContents, err := ioutil.ReadFile(filepath.Join(destDir, "generation"))
					Expect(err).ToNot(HaveOccurred())
					Expect(versionContents).To(Equal([]byte(strconv.FormatInt(generation, 10))))

					Expect(filepath.Join(destDir, "url")).To(BeARegularFile())
					urlContents, err := ioutil.ReadFile(filepath.Join(destDir, "url"))
					Expect(err).ToNot(HaveOccurred())
					Expect(urlContents).To(Equal([]byte(url)))
				})
			})

			Context("when the versioned file exists and 'unpack' is specified", func() {
				var (
					generation int64
				)

				BeforeEach(func() {
					tempDir, err := ioutil.TempDir("", directoryPrefix)
					Expect(err).ToNot(HaveOccurred())

					tempFilePath := filepath.Join(tempDir, "version.txt")
					tempTarballPath := filepath.Join(tempDir, "version.tgz")

					err = ioutil.WriteFile(tempFilePath, []byte("generation-4"), 0600)
					Expect(err).ToNot(HaveOccurred())

					command := exec.Command("tar", "czf", tempTarballPath, "-C", tempDir, "version.txt")
					session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Eventually(session).Should(gexec.Exit(0))

					generation, err = gcsClient.UploadFile(versionedBucketName, filepath.Join(directoryPrefix, "version.tgz"), "", tempTarballPath, "", "")
					Expect(err).ToNot(HaveOccurred())

					err = os.RemoveAll(tempDir)
					Expect(err).NotTo(HaveOccurred())

					inRequest = in.InRequest{
						Source: gcsresource.Source{
							JSONKey:       jsonKey,
							Bucket:        versionedBucketName,
							VersionedFile: filepath.Join(directoryPrefix, "version.tgz"),
						},
						Version: gcsresource.Version{
							Generation: fmt.Sprintf("%d", generation),
						},
						Params: in.Params{
							Unpack: true,
						},
					}

					err = json.NewEncoder(stdin).Encode(inRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					err := gcsClient.DeleteObject(versionedBucketName, filepath.Join(directoryPrefix, "version.tgz"), generation)
					Expect(err).ToNot(HaveOccurred())
				})

				It("downloads the file, decompresses it, and outputs the response", func() {
					reader := bytes.NewBuffer(session.Out.Contents())
					err = json.NewDecoder(reader).Decode(&inResponse)
					Expect(err).ToNot(HaveOccurred())

					url, err := gcsClient.URL(versionedBucketName, filepath.Join(directoryPrefix, "version.tgz"), generation)
					Expect(err).ToNot(HaveOccurred())

					Expect(inResponse).To(Equal(in.InResponse{
						Version: gcsresource.Version{
							Generation: fmt.Sprintf("%d", generation),
						},
						Metadata: []gcsresource.MetadataPair{
							{
								Name:  "filename",
								Value: "version.tgz",
							},
							{
								Name:  "url",
								Value: url,
							},
						},
					}))

					Expect(filepath.Join(destDir, "version.txt")).To(BeARegularFile())
					contents, err := ioutil.ReadFile(filepath.Join(destDir, "version.txt"))
					Expect(err).ToNot(HaveOccurred())
					Expect(contents).To(Equal([]byte("generation-4")))

					Expect(filepath.Join(destDir, "generation")).To(BeARegularFile())
					versionContents, err := ioutil.ReadFile(filepath.Join(destDir, "generation"))
					Expect(err).ToNot(HaveOccurred())
					Expect(versionContents).To(Equal([]byte(strconv.FormatInt(generation, 10))))

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
							Generation: fmt.Sprintf("%d", generation2),
						},
					}

					expectedExitStatus = 1

					err = json.NewEncoder(stdin).Encode(inRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns an error", func() {
					Expect(session.Err).To(gbytes.Say("error running command: storage: object doesn't exist"))
				})
			})

			Context("when the version matches the initial version", func() {
				BeforeEach(func() {
					inRequest = in.InRequest{
						Source: gcsresource.Source{
							JSONKey:              jsonKey,
							Bucket:               versionedBucketName,
							VersionedFile:        filepath.Join(directoryPrefix, "missing"),
							InitialVersion:       "0",
							InitialContentBinary: "Zm9v",
						},
						Version: gcsresource.Version{
							Generation: "0",
						},
					}

					err = json.NewEncoder(stdin).Encode(inRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				It("uses the writes decoded base64 as initial content", func() {
					reader := bytes.NewBuffer(session.Out.Contents())
					err = json.NewDecoder(reader).Decode(&inResponse)
					Expect(err).ToNot(HaveOccurred())

					Expect(inResponse).To(Equal(in.InResponse{
						Version: gcsresource.Version{
							Generation: "0",
						},
						Metadata: []gcsresource.MetadataPair{
							{
								Name:  "filename",
								Value: "missing",
							},
						},
					}))

					Expect(filepath.Join(destDir, "missing")).To(BeARegularFile())
					contents, err := ioutil.ReadFile(filepath.Join(destDir, "missing"))
					Expect(err).ToNot(HaveOccurred())
					Expect(contents).To(Equal([]byte("foo")), "")

					Expect(filepath.Join(destDir, "generation")).To(BeARegularFile())
					versionContents, err := ioutil.ReadFile(filepath.Join(destDir, "generation"))
					Expect(err).ToNot(HaveOccurred())
					Expect(versionContents).To(Equal([]byte(strconv.FormatInt(0, 10))))

					Expect(filepath.Join(destDir, "url")).ToNot(BeARegularFile())

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
							Generation: "12345",
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
							Generation: "0",
						},
					}

					expectedExitStatus = 1

					err = json.NewEncoder(stdin).Encode(inRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns an error", func() {
					Expect(session.Err).To(gbytes.Say("error running command: storage: object doesn't exist"))
				})
			})
		})
	})
})
