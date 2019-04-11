package main

import (
	"fmt"

	"github.com/bocajspear1/vmifactory/internal/imagemanage"
)

func main() {
	images := imagemanage.GetAvailableImages("./images")
	fmt.Println(images)
	image, ierr := imagemanage.NewVMImage("./images", "test-image")
	if ierr != nil {
		fmt.Println(ierr)
		return
	}
	fmt.Println(image.Exists())
	image.PrepareBuild()
	runerr := image.RunBuild()
	if runerr != nil {
		fmt.Println(runerr)
		return
	}
}
