package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/bocajspear1/vmifactory/internal/imagemanage"
)

func runImageBuild(image *imagemanage.VMImage, testBuild bool, noCommit bool) error {
	if !image.Exists() {
		return errors.New("Image '" + image.ImageName + "' does not exist or is not properly configured")
	}
	image.PrepareBuild()
	runerr := image.RunBuild(testBuild)
	if runerr != nil {
		return runerr
	}
	if !noCommit {
		commiterr := image.CommitBuild()
		if commiterr != nil {
			return commiterr
		}
	}

	return nil
}

func main() {

	const IMAGEDIR string = "./images"

	var noCommit = flag.Bool("nocommit", false, "Do not commit the images to be used. Useful for testing the images before distribution.")
	var testBuild = flag.Bool("test", false, "Don't actually do the build, useful for testing post-processing")
	var listImages = flag.Bool("list", false, "List the known available images")
	var runBuild = flag.String("run", "", "Set to run only one build instead of them all")

	var logFilePath = flag.String("logfile", "./vmif-run.log", "File to log to")

	flag.Parse()

	// Setup logging to Stdout and file
	logFile, err := os.OpenFile(*logFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0660)
	if err != nil {
		panic(err)
	}

	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	if *runBuild != "" {
		filteredImage := strings.ReplaceAll(*runBuild, ".", "")
		image, ierr := imagemanage.NewVMImage(IMAGEDIR, filteredImage)
		if ierr != nil {
			fmt.Println("Error finding image '" + filteredImage + "'")
			fmt.Println(ierr)
			return
		}
		fmt.Println("Running '" + image.Config.Name + "'")
		berr := runImageBuild(image, *testBuild, *noCommit)
		if berr != nil {
			fmt.Println(ierr)
			return
		}
		return
	}

	images := imagemanage.GetAvailableImages(IMAGEDIR)

	for _, imagePath := range images {
		image, ierr := imagemanage.NewVMImage(IMAGEDIR, imagePath)
		if ierr != nil {
			fmt.Println(ierr)
			return
		}
		if *listImages {
			fmt.Println(image.ImageName + " - " + image.Config.Name)
		} else {
			runerr := runImageBuild(image, *testBuild, *noCommit)
			if runerr != nil {
				fmt.Println(ierr)
				return
			}
		}
	}

}
