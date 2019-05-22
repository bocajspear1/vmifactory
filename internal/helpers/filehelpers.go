package helpers

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
)

func CopyFile(src string, dst string) error {

	source, oerr := os.Open(src)
	if oerr != nil {
		return oerr
	}
	defer source.Close()

	destination, cerr := os.Create(dst)
	if cerr != nil {
		return cerr
	}
	defer destination.Close()
	_, copyerr := io.Copy(destination, source)
	return copyerr
}

func TarAndGzipFiles(files []string, outputFilePath string) error {
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	gzipWriter := gzip.NewWriter(outputFile)
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, inputFilePath := range files {
		inputFile, err := os.Open(inputFilePath)
		if err != nil {
			return err
		}
		defer inputFile.Close()
		inputFileInfo, err := os.Stat(inputFilePath)
		header, err := tar.FileInfoHeader(inputFileInfo, inputFileInfo.Name())
		if err != nil {
			return err
		}
		tarWriter.WriteHeader(header)
		_, err = io.Copy(tarWriter, inputFile)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetFileSHA256(filepath string) (string, error) {
	hasher := sha256.New()
	inFile, err := ioutil.ReadFile(filepath)
	hasher.Write(inFile)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
