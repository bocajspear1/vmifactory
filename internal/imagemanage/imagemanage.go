package imagemanage

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GetAvailableImages returns a list of images
func GetAvailableImages(path string) []string {
	images := make([]string, 0)

	listing, err := ioutil.ReadDir(path)
	if err == nil {
		for _, item := range listing {
			dirName := item.Name()
			checkPath := path + "/" + dirName + "/" + dirName + ".json"
			_, err := os.Stat(checkPath)
			if err == nil {
				images = append(images, dirName)
			}
		}
	}

	return images
}

type BuilderConfig struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Login       map[string]string `json:"login"`
	Source      map[string]string `json:"source"`
	Out         map[string]string `json:"out"`
	Metadata    map[string]string `json:"metadata"`
}

// VMImage represents an image and its config.
type VMImage struct {
	ImageName    string
	ImageRootDir string
	Config       *BuilderConfig
}

// Private functions for VMImage

func (v VMImage) parseJSON() (*BuilderConfig, error) {
	configFile, ferr := ioutil.ReadFile(v.GetConfigPath())
	if ferr != nil {
		return nil, errors.New("Could not parse config file: File not found")
	}

	var config BuilderConfig
	jerr := json.Unmarshal(configFile, &(config))
	if jerr != nil {
		return nil, jerr
	}

	return &config, nil
}

func (v VMImage) saveJSON() error {
	newConfig, merr := json.MarshalIndent(v.Config, "", "    ")
	if merr != nil {
		return merr
	}
	err := ioutil.WriteFile(v.GetConfigPath(), newConfig, 0644)
	return err
}

// Generate the config
func (v VMImage) generatePackerConfig() (string, error) {

	var str strings.Builder

	builderString, ok := v.Config.Source["hypervisor"]

	if !ok {
		return "", errors.New("Required key 'source'/'hypervisor' not found")
	}

	builderOut := make(map[string]string)

	if builderString == "vbox" {
		builderOut["type"] = "virtualbox-ovf"
		builderOut["guest_additions_mode"] = "attach"
		builderOut["format"] = "ova"
		builderOut["source_path"], _ = filepath.Abs(v.GetWorkDirPath() + "/original.ova")
	} else {
		return "", errors.New("Builder is not supported")
	}

	builderOut["output_directory"], _ = filepath.Abs(v.GetWorkDirPath() + "/packer-out")

	// Login creds
	// TODO: check
	builderOut["ssh_username"] = v.Config.Login["username"]
	builderOut["ssh_password"] = v.Config.Login["password"]

	builderOut["vm_name"] = v.ImageName + "-vmifactory"

	// TODO: check
	str.Reset()
	str.WriteString("echo '")
	// TODO: check
	str.WriteString(v.Config.Login["sudo_password"])
	str.WriteString("' | sudo -p '' -S poweroff")

	builderOut["shutdown_command"] = str.String()

	fullConfig := make(map[string][]interface{})

	fullConfig["builders"] = append(fullConfig["builders"], builderOut)

	// Add the scripts

	provisionersOut := make(map[string]interface{})

	runScripts, err := ioutil.ReadDir(v.GetRunPath())
	if err != nil {
		return "", errors.New("Could not list the run directory for the image")
	}

	runOnceScripts, err := ioutil.ReadDir(v.GetRunOncePath())
	if err != nil {
		return "", errors.New("Could not list the runOnce directory for the image")
	}

	runOnceCount := len(runOnceScripts) - 1

	scripts := make([]string, (len(runScripts) + runOnceCount))

	for i := 0; i < len(runOnceScripts); i++ {
		if !(runOnceScripts[i].IsDir()) {
			scripts[i] = v.GetRunPath() + "/" + runOnceScripts[i].Name()
		}
	}

	for i := 0; i < len(runScripts); i++ {
		scripts[i] = v.GetRunPath() + "/" + runScripts[i].Name()
	}

	provisionersOut["type"] = "shell"
	provisionersOut["scripts"] = scripts

	str.Reset()
	str.WriteString("echo '")
	// TODO: check
	str.WriteString(v.Config.Login["sudo_password"])
	str.WriteString("' | sudo -p '' -S env {{ .Vars }} {{ .Path }}")

	provisionersOut["execute_command"] = str.String()

	fullConfig["provisioners"] = append(fullConfig["provisioners"], provisionersOut)

	// Output the full Packer configurationj
	outJSONBytes, oerr := json.MarshalIndent(fullConfig, "", "    ")

	return string(outJSONBytes), oerr
}

// NewVMImage creates new vmimage structs
func NewVMImage(path string, imageName string) (*VMImage, error) {
	// TODO: Do some sanity checking (no .. for the sake of it)
	p := new(VMImage)
	p.ImageName = imageName
	p.ImageRootDir = path + "/" + imageName

	config, cerr := p.parseJSON()
	p.Config = config

	if cerr != nil {
		return nil, cerr
	}

	return p, nil
}

// Exists checks if the image actually exists
func (v VMImage) Exists() bool {
	_, err := os.Stat(v.GetConfigPath())
	return err == nil
}

// GetConfigPath returns the path to the config file
func (v VMImage) GetConfigPath() string {
	return v.ImageRootDir + "/" + v.ImageName + ".json"
}

// GetWorkDirPath returns the path to the work directory
func (v VMImage) GetWorkDirPath() string {
	return v.ImageRootDir + "/work"
}

// GetRunPath returns the path to the run script directory
func (v VMImage) GetRunPath() string {
	return v.ImageRootDir + "/run"
}

// GetRunOncePath returns the path to the run script directory
func (v VMImage) GetRunOncePath() string {
	return v.ImageRootDir + "/runonce"
}

// GetCommitFlag returns the path to commit flag
func (v VMImage) GetCommitFlag() string {
	return v.GetWorkDirPath() + "/commit"
}

// CommitFlagExists checks if the image commit flag exists
func (v VMImage) CommitFlagExists() bool {
	_, err := os.Stat(v.GetCommitFlag())
	return err == nil
}

func (v VMImage) EnableCommitFlag() bool {
	newFile, err := os.Create(v.GetCommitFlag())
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer newFile.Close()
	newFile.WriteString("commit!")
	return true
}

func (v VMImage) DisableCommitFlag() bool {
	if v.CommitFlagExists() {
		os.Remove(v.GetCommitFlag())
		return true
	}
	return false
}

// PrepareBuild prepares the image for an update build
func (v VMImage) PrepareBuild() bool {
	workDir := v.GetWorkDirPath()
	_, err := os.Stat(workDir)

	// Remove the old work directory
	if err == nil {
		log.Println("Removing old work directory...")
		os.RemoveAll(workDir)
	}

	os.Mkdir(workDir, 0777)

	return true
}

// RunBuild runs the build
func (v VMImage) RunBuild(skipBuild bool) error {

	// Generate the Packer config
	config, cerr := v.generatePackerConfig()
	if cerr != nil {
		return cerr
	}
	log.Println("Packer config generated...")

	configFile := v.GetWorkDirPath() + "/builtpacker.json"
	ioutil.WriteFile(configFile, []byte(config), 0755)

	imagefilePath, ok := v.Config.Source["imagefile"]
	if !ok {
		return errors.New("Required key source'/'imagefile' not found")
	}

	builderString, ok := v.Config.Source["hypervisor"]
	if !ok {
		return errors.New("Required key 'source'/'hypervisor' not found")
	}

	var copyerr error

	// Copy in the current image file into the work directory
	if builderString == "vbox" {
		copyerr = copyFile(v.ImageRootDir+"/"+imagefilePath, v.GetWorkDirPath()+"/original.ova")
	} else {
		return errors.New(builderString + " not currently supported")
	}

	if copyerr != nil {
		return copyerr
	}

	// Check if we want to sckip the build, usually for testing
	if !skipBuild {
		log.Println("Starting Packer build...")
		// Run the build
		cwd, _ := os.Getwd()

		cmd := exec.Command(cwd+"/packer", "build", configFile)
		output, err := cmd.Output()
		ferr := ioutil.WriteFile(v.GetWorkDirPath()+"/packer.log", output, 0644)
		if err != nil {
			return err
		}
		if ferr != nil {
			return ferr
		}
		fmt.Printf("%s", output)

	} else {
		log.Println("!!! - Faking Packer build...")
		// Manually create and fill the output directory
		os.Mkdir(v.GetWorkDirPath()+"/packer-out", 0777)
		if builderString == "vbox" {
			originalPath := v.GetWorkDirPath() + "/original.ova"
			fakingPath := v.GetWorkDirPath() + "/packer-out/" + v.ImageName + "-vmifactory.ova"
			copyerr = copyFile(originalPath, fakingPath)
		}
	}

	// Convert the outputs
	if builderString == "vbox" {
		// Copy our OVA
		tarFile := v.GetWorkDirPath() + "/convert.tar"
		copyerr = copyFile(v.GetWorkDirPath()+"/packer-out/"+v.ImageName+"-vmifactory.ova", tarFile)

		reader, err := os.Open(tarFile)
		defer reader.Close()
		if err != nil {
			return err
		}

		disksDir := v.GetWorkDirPath() + "/ova-disks"

		// Extract the disk(s)
		tarReader := tar.NewReader(reader)
		os.Mkdir(disksDir, 0777)

		for {
			tarHeader, err := tarReader.Next()
			if err == io.EOF {
				break // End of archive
			} else if err != nil {
				return err
			}

			if strings.Contains(tarHeader.Name, ".vmdk") {
				destination, cerr := os.Create(disksDir + "/" + tarHeader.Name)
				if cerr != nil {
					return cerr
				}
				_, werr := io.Copy(destination, tarReader)
				if werr != nil {
					return werr
				}
			}
		}

		ovaDisks, err := ioutil.ReadDir(disksDir)
		if err != nil {
			return err
		}

		// Do conversion for KVM
		kvmName, ok := v.Config.Out["kvm"]
		if ok && kvmName != "" {
			log.Println("Doing KVM conversion...")
			convertedList := make([]string, len(ovaDisks))
			log.Println("(KVM) Converting disks...")
			// For each disk, make a QCOW2 copy
			for i, diskFile := range ovaDisks {
				newName := strings.ReplaceAll(diskFile.Name(), ".vmdk", ".qcow2")
				convertedList[i] = disksDir + "/" + newName
				cmd := exec.Command("qemu-img", "convert", "-O", "qcow2", disksDir+"/"+diskFile.Name(), disksDir+"/"+newName)
				convertOut, err := cmd.Output()
				if err != nil {
					return err
				}
				fmt.Printf("%s", convertOut)
			}
			log.Println("(KVM) Building Gzipped Tar...")
			// Tar and gzip the QCOW2 files
			packErr := tarAndGzipFiles(convertedList, v.GetWorkDirPath()+"/"+kvmName)
			if packErr != nil {
				return packErr
			}
		}

	}
	log.Println("Conversions completed...")

	realName, ok := v.Config.Source["imagefile"]
	if !ok {
		return errors.New("Required key 'source'.'imagefile' not found")
	}
	os.Rename(v.GetWorkDirPath()+"/original.ova", v.GetWorkDirPath()+"/"+realName)
	log.Println("Renamed original file...")
	return nil
}

// CommitBuild updates the image files and metadata
func (v VMImage) CommitBuild() error {
	v.EnableCommitFlag()
	// Ensure our source image file is the same as its out file name
	imagefileName, ok := v.Config.Source["imagefile"]
	if !ok {
		return errors.New("Required key 'source'.'imagefile' not found")
	}
	sourceType, ok := v.Config.Source["hypervisor"]
	if !ok {
		return errors.New("Required key 'source'.'hypervisor' not found")
	}
	v.Config.Out[sourceType] = imagefileName

	log.Println("Updating metadata and moving files...")

	for hypervisor, outFileName := range v.Config.Out {
		if outFileName != "" {

			oldImagefilePath := v.ImageRootDir + "/Old-" + outFileName
			currentImagefilePath := v.ImageRootDir + "/" + outFileName
			newImagefilePath := v.GetWorkDirPath() + "/" + outFileName

			currentHash, ok := v.Config.Metadata[hypervisor+"_current_hash"]
			if ok {
				v.Config.Metadata[hypervisor+"_last_hash"] = currentHash
			} else {
				v.Config.Metadata[hypervisor+"_last_hash"] = ""
			}
			currentBuildDate, ok := v.Config.Metadata[hypervisor+"_current_date"]
			if ok {
				v.Config.Metadata[hypervisor+"_last_date"] = currentBuildDate
			} else {
				v.Config.Metadata[hypervisor+"_last_date"] = ""
			}
			fileHash, err := getFileSHA256(newImagefilePath)
			if err != nil {
				return err
			}
			v.Config.Metadata[hypervisor+"_current_hash"] = fileHash
			dt := time.Now()
			//
			v.Config.Metadata[hypervisor+"_current_date"] = dt.Format("2006-01-02 15:04:05")

			// Remove the old image if it exists
			_, err = os.Stat(oldImagefilePath)
			if err == nil {
				os.Remove(oldImagefilePath)
			}

			// Move the existing to the Old- name
			_, err = os.Stat(currentImagefilePath)
			if err == nil {
				os.Rename(currentImagefilePath, oldImagefilePath)
			}

			// Move the new to the current path
			os.Rename(newImagefilePath, currentImagefilePath)

		}
		v.saveJSON()
		v.DisableCommitFlag()
	}

	return nil
}
