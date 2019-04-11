package imagemanage

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func copyFile(src string, dst string) error {

	source, oerr := os.Open(src)
	if oerr != nil {
		return oerr
	}
	defer source.Close()

	fmt.Println(dst)
	destination, cerr := os.Create(dst)
	if cerr != nil {
		return cerr
	}
	defer destination.Close()
	_, copyerr := io.Copy(destination, source)
	fmt.Println("hi")
	return copyerr
}

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
	Hashes      map[string]string `json:"hashes"`
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
	fmt.Println(config)
	return &config, nil
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

	scripts := make([]string, len(runScripts))

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
	fmt.Println(p.Config)
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

// PrepareBuild prepares the image for an update build
func (v VMImage) PrepareBuild() bool {
	workDir := v.GetWorkDirPath()
	_, err := os.Stat(workDir)

	// Remove the old work directory
	if err == nil {
		fmt.Println("Removing old work directory...")
		os.RemoveAll(workDir)
	}

	os.Mkdir(workDir, 0777)

	return true
}

func (v VMImage) RunBuild() error {

	// Generate the Packer config
	config, cerr := v.generatePackerConfig()
	if cerr != nil {
		return cerr
	}

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
	}

	if copyerr != nil {
		return copyerr
	}

	// Run the build
	cwd, _ := os.Getwd()

	cmd := exec.Command(cwd+"/packer", "build", configFile)
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	fmt.Printf("%s", output)

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
			for _, diskFile := range ovaDisks {
				newName := strings.ReplaceAll(diskFile.Name(), ".vmdk", ".qcow2")
				cmd := exec.Command("qemu-img", "convert", "-O", "qcow2", disksDir+"/"+diskFile.Name(), disksDir+"/"+newName)
				convertOut, err := cmd.Output()
				if err != nil {
					return err
				}
				fmt.Printf("%s", convertOut)
			}
		}

	}

	// os.RemoveAll(workDir)
	return nil
}
