package in

import (
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/h2non/filetype"
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
	cmd := exec.Command("unzip", "-d", destinationDir, sourcePath)

	return cmd.Run()
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
	cmd := exec.Command("tar", "xf", sourcePath, "-C", destinationDir)

	return cmd.Run()
}
