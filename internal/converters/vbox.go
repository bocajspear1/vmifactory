package converters

import (
	"archive/tar"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/bocajspear1/vmifactory/internal/helpers"
)

const ovaDir = "ova-disks"

// VBoxExtractDisks extracts the disks the produced OVA
// into the directory workDir + /ova-disks
func VBoxExtractDisks(workDir string, ovaPath string) ([]string, error) {

	ovaDisks := make([]string, 0)

	// Copy our OVA
	tarFile := workDir + "/convert.tar"
	copyerr := helpers.CopyFile(ovaPath, tarFile)

	if copyerr != nil {
		return nil, copyerr
	}

	reader, err := os.Open(tarFile)
	defer reader.Close()
	if err != nil {
		return nil, err
	}

	disksDir := workDir + "/" + ovaDir

	// Extract the disk(s)
	tarReader := tar.NewReader(reader)
	os.Mkdir(disksDir, 0777)

	for {
		tarHeader, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		} else if err != nil {
			return nil, err
		}

		if strings.Contains(tarHeader.Name, ".vmdk") {
			outPath := disksDir + "/" + tarHeader.Name
			destination, cerr := os.Create(outPath)
			if cerr != nil {
				return nil, cerr
			}
			_, werr := io.Copy(destination, tarReader)
			if werr != nil {
				return nil, werr
			}
			ovaDisks = append(ovaDisks, outPath)
		}
	}

	log.Println("VBox TAR cleanup...")
	os.Remove(tarFile)

	return ovaDisks, nil
}

// VBoxCleanup cleans up conversion artifacts
func VBoxCleanup(workDir string) {
	disksDir := workDir + "/" + ovaDir
	os.RemoveAll(disksDir)
}

func VBoxToKVM(diskList []string, outputPath string) error {
	convertedList := make([]string, len(diskList))
	log.Println("(KVM) Converting disks...")
	// For each disk, make a QCOW2 copy
	for i, diskFile := range diskList {
		newName := strings.ReplaceAll(diskFile, ".vmdk", ".qcow2")
		convertedList[i] = newName
		output, cerr := DiskToQCOW2(diskFile, newName)
		if cerr != nil {
			return cerr
		}
		fmt.Printf("%s", output)

	}
	log.Println("(KVM) Building Gzipped Tar...")
	// Tar and gzip the QCOW2 files
	packErr := helpers.TarAndGzipFiles(convertedList, outputPath)
	if packErr != nil {
		return packErr
	}
	return nil
}
