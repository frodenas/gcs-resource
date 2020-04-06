package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	//"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"github.com/frodenas/gcs-resource"
	"github.com/frodenas/gcs-resource/check"
	"github.com/nu7hatch/gouuid"
)

var _ = Describe("check", func() {
	var (
		err     error
		command *exec.Cmd
		stdin   *bytes.Buffer
		session *gexec.Session

		expectedExitStatus int
	)

	BeforeEach(func() {
		stdin = &bytes.Buffer{}
		expectedExitStatus = 0

		command = exec.Command(checkPath)
		command.Stdin = stdin
	})

	JustBeforeEach(func() {
		session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		<-session.Exited
		Expect(session.ExitCode()).To(Equal(expectedExitStatus))
	})

	Describe("when the request is invalid", func() {
		var (
			checkRequest check.CheckRequest
		)

		BeforeEach(func() {
			checkRequest = check.CheckRequest{
				Source: gcsresource.Source{
					JSONKey: jsonKey,
					Bucket:  bucketName,
				},
			}

			expectedExitStatus = 1
		})

		Context("when the bucket is not set", func() {
			BeforeEach(func() {
				checkRequest.Source.Bucket = ""

				err := json.NewEncoder(stdin).Encode(checkRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				Expect(session.Err).To(gbytes.Say("please specify the bucket"))
			})
		})

		Context("when the regexp and versioned_file are both set", func() {
			BeforeEach(func() {
				checkRequest.Source.Regexp = "file-to-*"
				checkRequest.Source.VersionedFile = "files/version"

				err := json.NewEncoder(stdin).Encode(checkRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				Expect(session.Err).To(gbytes.Say("please specify either regexp or versioned_file"))
			})
		})
	})

	Describe("with regexp", func() {
		var (
			checkRequest    check.CheckRequest
			checkResponse   check.CheckResponse
			directoryPrefix string
		)

		BeforeEach(func() {
			guid, err := uuid.NewV4()
			Expect(err).ToNot(HaveOccurred())
			directoryPrefix = "check-request-files-" + guid.String()
		})

		Context("when the bucket does not exist", func() {
			BeforeEach(func() {
				checkRequest = check.CheckRequest{
					Source: gcsresource.Source{
						JSONKey: jsonKey,
						Bucket:  directoryPrefix,
						Regexp:  filepath.Join(directoryPrefix, "missing-(.*).tgz"),
					},
				}

				expectedExitStatus = 1

				err = json.NewEncoder(stdin).Encode(checkRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				Expect(session.Err).To(gbytes.Say("error listing objects: storage: bucket doesn't exist"))
			})
		})

		Context("when the regexp does not match anything", func() {
			BeforeEach(func() {
				checkRequest = check.CheckRequest{
					Source: gcsresource.Source{
						JSONKey: jsonKey,
						Bucket:  bucketName,
						Regexp:  filepath.Join(directoryPrefix, "missing-(.*).tgz"),
					},
				}
			})

			Context("when no initial_path is set", func() {
				BeforeEach(func() {
					err = json.NewEncoder(stdin).Encode(checkRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns an empty check response", func() {
					reader := bytes.NewBuffer(session.Out.Contents())
					err = json.NewDecoder(reader).Decode(&checkResponse)
					Expect(err).ToNot(HaveOccurred())

					Expect(checkResponse).To(Equal(check.CheckResponse{}))
				})
			})

			Context("when initial_path is set", func() {
				BeforeEach(func() {
					checkRequest.Source.InitialPath = filepath.Join(directoryPrefix, "missing-0.0.0.tgz")
					err = json.NewEncoder(stdin).Encode(checkRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns the specified initial path", func() {
					reader := bytes.NewBuffer(session.Out.Contents())
					err = json.NewDecoder(reader).Decode(&checkResponse)
					Expect(err).ToNot(HaveOccurred())

					expected := check.CheckResponse{
						{
							Path: filepath.Join(directoryPrefix, "missing-0.0.0.tgz"),
						},
					}

					Expect(checkResponse).To(Equal(expected))
				})
			})
		})

		Context("when the regexp matches something", func() {
			BeforeEach(func() {
				tempFile, err := ioutil.TempFile("", directoryPrefix)
				Expect(err).ToNot(HaveOccurred())
				tempFile.Close()

				err = ioutil.WriteFile(tempFile.Name(), []byte("file-to-check-1"), 0755)
				Expect(err).ToNot(HaveOccurred())

				_, err = gcsClient.UploadFile(bucketName, filepath.Join(directoryPrefix, "file-to-check-1"), "", tempFile.Name(), "", "")
				Expect(err).ToNot(HaveOccurred())

				err = ioutil.WriteFile(tempFile.Name(), []byte("file-to-check-3"), 0755)
				Expect(err).ToNot(HaveOccurred())

				_, err = gcsClient.UploadFile(bucketName, filepath.Join(directoryPrefix, "file-to-check-3"), "", tempFile.Name(), "", "")
				Expect(err).ToNot(HaveOccurred())

				err = ioutil.WriteFile(tempFile.Name(), []byte("file-to-check-5"), 0755)
				Expect(err).ToNot(HaveOccurred())

				_, err = gcsClient.UploadFile(bucketName, filepath.Join(directoryPrefix, "file-to-check-5"), "", tempFile.Name(), "", "")
				Expect(err).ToNot(HaveOccurred())

				err = os.Remove(tempFile.Name())
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				err = gcsClient.DeleteObject(bucketName, filepath.Join(directoryPrefix, "file-to-check-1"), 0)
				Expect(err).ToNot(HaveOccurred())

				err = gcsClient.DeleteObject(bucketName, filepath.Join(directoryPrefix, "file-to-check-3"), 0)
				Expect(err).ToNot(HaveOccurred())

				err = gcsClient.DeleteObject(bucketName, filepath.Join(directoryPrefix, "file-to-check-5"), 0)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when we do not provide a previous version", func() {
				BeforeEach(func() {
					checkRequest = check.CheckRequest{
						Source: gcsresource.Source{
							JSONKey: jsonKey,
							Bucket:  bucketName,
							Regexp:  filepath.Join(directoryPrefix, "file-to-check-(.*)"),
						},
					}

					err = json.NewEncoder(stdin).Encode(checkRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns the latest version", func() {
					reader := bytes.NewBuffer(session.Out.Contents())
					err = json.NewDecoder(reader).Decode(&checkResponse)
					Expect(err).ToNot(HaveOccurred())

					Expect(checkResponse).To(Equal(check.CheckResponse{
						{
							Path: filepath.Join(directoryPrefix, "file-to-check-5"),
						},
					}))
				})
			})

			Context("when we provide a previous version", func() {
				Context("and the version exists", func() {
					BeforeEach(func() {
						checkRequest = check.CheckRequest{
							Source: gcsresource.Source{
								JSONKey: jsonKey,
								Bucket:  bucketName,
								Regexp:  filepath.Join(directoryPrefix, "file-to-check-(.*)"),
							},
							Version: gcsresource.Version{
								Path: filepath.Join(directoryPrefix, "file-to-check-1"),
							},
						}

						err = json.NewEncoder(stdin).Encode(checkRequest)
						Expect(err).ToNot(HaveOccurred())
					})

					It("returns the most recent versions", func() {
						reader := bytes.NewBuffer(session.Out.Contents())
						err = json.NewDecoder(reader).Decode(&checkResponse)
						Expect(err).ToNot(HaveOccurred())

						Expect(checkResponse).To(Equal(check.CheckResponse{
							{
								Path: filepath.Join(directoryPrefix, "file-to-check-3"),
							},
							{
								Path: filepath.Join(directoryPrefix, "file-to-check-5"),
							},
						}))
					})
				})

				Context("and the version does not exists", func() {
					Context("but there are greater versions", func() {
						BeforeEach(func() {
							checkRequest = check.CheckRequest{
								Source: gcsresource.Source{
									JSONKey: jsonKey,
									Bucket:  bucketName,
									Regexp:  filepath.Join(directoryPrefix, "file-to-check-(.*)"),
								},
								Version: gcsresource.Version{
									Path: filepath.Join(directoryPrefix, "file-to-check-2"),
								},
							}

							err = json.NewEncoder(stdin).Encode(checkRequest)
							Expect(err).ToNot(HaveOccurred())
						})

						It("returns the most recent versions", func() {
							reader := bytes.NewBuffer(session.Out.Contents())
							err = json.NewDecoder(reader).Decode(&checkResponse)
							Expect(err).ToNot(HaveOccurred())

							Expect(checkResponse).To(Equal(check.CheckResponse{
								{
									Path: filepath.Join(directoryPrefix, "file-to-check-3"),
								},
								{
									Path: filepath.Join(directoryPrefix, "file-to-check-5"),
								},
							}))
						})
					})

					Context("and there are not greater versions", func() {
						BeforeEach(func() {
							checkRequest = check.CheckRequest{
								Source: gcsresource.Source{
									JSONKey: jsonKey,
									Bucket:  bucketName,
									Regexp:  filepath.Join(directoryPrefix, "file-to-check-(.*)"),
								},
								Version: gcsresource.Version{
									Path: filepath.Join(directoryPrefix, "file-to-check-6"),
								},
							}

							err = json.NewEncoder(stdin).Encode(checkRequest)
							Expect(err).ToNot(HaveOccurred())
						})

						It("returns an empty check response", func() {
							reader := bytes.NewBuffer(session.Out.Contents())
							err = json.NewDecoder(reader).Decode(&checkResponse)
							Expect(err).ToNot(HaveOccurred())

							Expect(checkResponse).To(Equal(check.CheckResponse{}))
						})
					})
				})
			})
		})
	})

	Describe("with a versioned_file", func() {
		var (
			checkRequest    check.CheckRequest
			checkResponse   check.CheckResponse
			directoryPrefix string
			generation1     int64
			generation2     int64
			generation3     int64
		)

		BeforeEach(func() {
			guid, err := uuid.NewV4()
			Expect(err).ToNot(HaveOccurred())
			directoryPrefix = "check-request-files-" + guid.String()
		})

		Context("when the bucket is not versioned", func() {
			BeforeEach(func() {
				checkRequest = check.CheckRequest{
					Source: gcsresource.Source{
						JSONKey:       jsonKey,
						Bucket:        bucketName,
						VersionedFile: filepath.Join(directoryPrefix, "version"),
					},
				}

				expectedExitStatus = 1

				err = json.NewEncoder(stdin).Encode(checkRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				Expect(session.Err).To(gbytes.Say("bucket is not versioned"))
			})
		})

		Context("when the bucket does not exist", func() {
			BeforeEach(func() {
				checkRequest = check.CheckRequest{
					Source: gcsresource.Source{
						JSONKey:       jsonKey,
						Bucket:        directoryPrefix,
						VersionedFile: filepath.Join(directoryPrefix, "version"),
					},
				}

				expectedExitStatus = 1

				err = json.NewEncoder(stdin).Encode(checkRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				Expect(session.Err).To(gbytes.Say("error running command: storage: bucket doesn't exist"))
			})
		})

		Context("when the file does not exist", func() {
			BeforeEach(func() {
				checkRequest = check.CheckRequest{
					Source: gcsresource.Source{
						JSONKey:       jsonKey,
						Bucket:        versionedBucketName,
						VersionedFile: filepath.Join(directoryPrefix, "version"),
					},
				}
			})

			Context("and no initial_version is set", func() {
				BeforeEach(func() {
					err = json.NewEncoder(stdin).Encode(checkRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns an empty check response", func() {
					reader := bytes.NewBuffer(session.Out.Contents())
					err = json.NewDecoder(reader).Decode(&checkResponse)
					Expect(err).ToNot(HaveOccurred())

					Expect(checkResponse).To(Equal(check.CheckResponse{}))
				})
			})

			Context("when initial_version is set", func() {
				BeforeEach(func() {
					checkRequest.Source.InitialVersion = "100"
					err = json.NewEncoder(stdin).Encode(checkRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns the specified initial version", func() {
					reader := bytes.NewBuffer(session.Out.Contents())
					err = json.NewDecoder(reader).Decode(&checkResponse)
					Expect(err).ToNot(HaveOccurred())

					expected := check.CheckResponse{
						{
							Generation: fmt.Sprintf("%d", 100),
						},
					}

					Expect(checkResponse).To(Equal(expected))
				})
			})
		})

		Context("when the file exists", func() {
			BeforeEach(func() {
				tempFile, err := ioutil.TempFile("", directoryPrefix)
				Expect(err).ToNot(HaveOccurred())
				tempFile.Close()

				err = ioutil.WriteFile(tempFile.Name(), []byte("generation-1"), 0755)
				Expect(err).ToNot(HaveOccurred())

				generation1, err = gcsClient.UploadFile(versionedBucketName, filepath.Join(directoryPrefix, "version"), "", tempFile.Name(), "", "")
				Expect(err).ToNot(HaveOccurred())

				err = ioutil.WriteFile(tempFile.Name(), []byte("generation-2"), 0755)
				Expect(err).ToNot(HaveOccurred())

				generation2, err = gcsClient.UploadFile(versionedBucketName, filepath.Join(directoryPrefix, "version"), "", tempFile.Name(), "", "")
				Expect(err).ToNot(HaveOccurred())

				err = ioutil.WriteFile(tempFile.Name(), []byte("generation-3"), 0755)
				Expect(err).ToNot(HaveOccurred())

				generation3, err = gcsClient.UploadFile(versionedBucketName, filepath.Join(directoryPrefix, "version"), "", tempFile.Name(), "", "")
				Expect(err).ToNot(HaveOccurred())

				err = os.Remove(tempFile.Name())
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

			Context("when we do not provide a previous version", func() {
				BeforeEach(func() {
					checkRequest = check.CheckRequest{
						Source: gcsresource.Source{
							JSONKey:        jsonKey,
							Bucket:         versionedBucketName,
							VersionedFile:  filepath.Join(directoryPrefix, "version"),
							InitialVersion: "100",
						},
					}

					err = json.NewEncoder(stdin).Encode(checkRequest)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns the latest version", func() {
					reader := bytes.NewBuffer(session.Out.Contents())
					err = json.NewDecoder(reader).Decode(&checkResponse)
					Expect(err).ToNot(HaveOccurred())

					Expect(checkResponse).To(Equal(check.CheckResponse{
						{
							Generation: fmt.Sprintf("%d", generation3),
						},
					}))
				})
			})

			Context("when we provide a previous version", func() {
				Context("and the version exists", func() {
					BeforeEach(func() {
						checkRequest = check.CheckRequest{
							Source: gcsresource.Source{
								JSONKey:       jsonKey,
								Bucket:        versionedBucketName,
								VersionedFile: filepath.Join(directoryPrefix, "version"),
							},
							Version: gcsresource.Version{
								Generation: fmt.Sprintf("%d", generation1),
							},
						}

						err = json.NewEncoder(stdin).Encode(checkRequest)
						Expect(err).ToNot(HaveOccurred())
					})

					It("returns the most recent versions", func() {
						reader := bytes.NewBuffer(session.Out.Contents())
						err = json.NewDecoder(reader).Decode(&checkResponse)
						Expect(err).ToNot(HaveOccurred())

						Expect(checkResponse).To(Equal(check.CheckResponse{
							{
								Generation: fmt.Sprintf("%d", generation2),
							},
							{
								Generation: fmt.Sprintf("%d", generation3),
							},
						}))
					})
				})

				Context("and the version does not exists", func() {
					Context("but there are greater versions", func() {
						BeforeEach(func() {
							checkRequest = check.CheckRequest{
								Source: gcsresource.Source{
									JSONKey:       jsonKey,
									Bucket:        versionedBucketName,
									VersionedFile: filepath.Join(directoryPrefix, "version"),
								},
								Version: gcsresource.Version{
									Generation: fmt.Sprintf("%d", generation2+1),
								},
							}

							err = json.NewEncoder(stdin).Encode(checkRequest)
							Expect(err).ToNot(HaveOccurred())
						})

						It("returns the most recent versions", func() {
							reader := bytes.NewBuffer(session.Out.Contents())
							err = json.NewDecoder(reader).Decode(&checkResponse)
							Expect(err).ToNot(HaveOccurred())

							Expect(checkResponse).To(Equal(check.CheckResponse{
								{
									Generation: fmt.Sprintf("%d", generation3),
								},
							}))
						})
					})

					Context("and there are not greater versions", func() {
						BeforeEach(func() {
							checkRequest = check.CheckRequest{
								Source: gcsresource.Source{
									JSONKey:       jsonKey,
									Bucket:        versionedBucketName,
									VersionedFile: filepath.Join(directoryPrefix, "version"),
								},
								Version: gcsresource.Version{
									Generation: fmt.Sprintf("%d", generation3+1),
								},
							}

							err = json.NewEncoder(stdin).Encode(checkRequest)
							Expect(err).ToNot(HaveOccurred())
						})

						It("returns an empty check response", func() {
							reader := bytes.NewBuffer(session.Out.Contents())
							err = json.NewDecoder(reader).Decode(&checkResponse)
							Expect(err).ToNot(HaveOccurred())

							Expect(checkResponse).To(Equal(check.CheckResponse{}))
						})
					})
				})
			})
		})
	})
})
