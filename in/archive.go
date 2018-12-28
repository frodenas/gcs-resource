package in

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

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
	cmd := exec.Command("unzip", "-P", "", "-d", destinationDir, sourcePath)
	defer os.Remove(sourcePath)

	return cmd.Run()
}

func unpackGzip(sourcePath string) (string, error) {
	cmd := exec.Command("gunzip", sourcePath)
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	destinationDir := filepath.Dir(sourcePath)
	fileInfos, err := ioutil.ReadDir(destinationDir)
	if err != nil {
		return "", fmt.Errorf("failed to read dir: %s", err)
	}
	if len(fileInfos) != 1 {
		return "", fmt.Errorf("%d files found after gunzip; expected 1", len(fileInfos))
	}

	return filepath.Join(destinationDir, fileInfos[0].Name()), err
}

func unpackTar(sourcePath, destinationDir string) error {
	cmd := exec.Command("tar", "xf", sourcePath, "-C", destinationDir)
	defer os.Remove(sourcePath)

	return cmd.Run()
}
