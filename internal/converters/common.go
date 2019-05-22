package converters

import "os/exec"

func DiskToQCOW2(initPath string, newPath string) (string, error) {
	cmd := exec.Command("qemu-img", "convert", "-O", "qcow2", initPath, newPath)
	convertOut, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(convertOut), nil
}
