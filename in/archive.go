package in

import (
	"archive/zip"
	"bufio"
	"io"
	"os"

	"archive/tar"
	"compress/gzip"
	"github.com/h2non/filetype"
	"path/filepath"
)

const (
	mimeTypeZip  = "application/zip"
	mimeTypeTar  = "application/x-tar"
	mimeTypeGzip = "application/gzip"
)

func isSupportedMimeType(mimeType string) bool {
	return mimeType == mimeTypeZip ||
		mimeType == mimeTypeTar ||
		mimeType == mimeTypeGzip
}

func getMimeType(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	bs, err := bufio.NewReader(f).Peek(512)
	if err != nil && err != io.EOF {
		return "", err
	}

	kind, err := filetype.Match(bs)
	if err != nil {
		return "", err
	}

	return kind.MIME.Value, nil
}

func unpack(mimeType, sourcePath string) error {
	for mimeType == mimeTypeGzip {
		var err error
		sourcePath, err = unpackGzip(sourcePath)
		if err != nil {
			return err
		}

		mimeType, err = getMimeType(sourcePath)
		if err != nil {
			return err
		}
	}

	destinationDir := filepath.Dir(sourcePath)

	switch mimeType {
	case mimeTypeZip:
		return unpackZip(sourcePath, destinationDir)
	case mimeTypeTar:
		return unpackTar(sourcePath, destinationDir)
	}

	return nil
}

func unpackZip(sourcePath, destinationDir string) error {
	r, err := zip.OpenReader(sourcePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		fpath := filepath.Join(destinationDir, f.Name)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return err
			}
		} else {
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return err
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}

			_, err = io.Copy(outFile, rc)
			outFile.Close()

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func unpackGzip(sourcePath string) (string, error) {
	reader, err := os.Open(sourcePath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	archive, err := gzip.NewReader(reader)
	if err != nil {
		return "", err
	}
	defer archive.Close()

	var destinationPath string
	if archive.Name != "" {
		destinationPath = filepath.Join(filepath.Dir(sourcePath), archive.Name)
	} else {
		destinationPath = sourcePath + ".uncompressed"
	}

	writer, err := os.Create(destinationPath)
	if err != nil {
		return "", err
	}
	defer writer.Close()

	_, err = io.Copy(writer, archive)
	return destinationPath, err
}

func unpackTar(sourcePath, destinationDir string) error {
	reader, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(destinationDir, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}

			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}

	return nil
}
