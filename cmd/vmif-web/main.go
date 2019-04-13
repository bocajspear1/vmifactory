package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/bocajspear1/vmifactory/internal/imagemanage"
)

func getFileContentType(filePath string) (string, error) {

	testFile, oerr := os.Open(filePath)
	defer testFile.Close()
	if oerr != nil {
		return "", oerr
	}
	buffer := make([]byte, 512)

	_, rerr := testFile.Read(buffer)
	if rerr != nil {
		return "", rerr
	}

	contentType := http.DetectContentType(buffer)

	return contentType, nil
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	reqPath := strings.ReplaceAll(r.URL.Path, "..", "")
	sections := strings.Split(reqPath[len("/get/"):], "/")
	fmt.Println(len(sections))
	if len(sections) != 2 {
		fmt.Fprintln(w, "Invalid image file requested")
		return
	}
	imagePathName := sections[0]
	imageName := sections[1]
	image, ierr := imagemanage.NewVMImage("./images", imagePathName)
	if ierr != nil {
		fmt.Fprintln(w, "Invalid image file name requested")
		return
	}
	imagePath := image.ImageRootDir + "/" + imageName
	fmt.Println(imagePath)
	fileData, serr := os.Stat(imagePath)
	if serr != nil {
		fmt.Fprintln(w, "The image file was not found, please contact the administrator")
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+fileData.Name())

	contentType, err := getFileContentType(imagePath)
	if err != nil {
		fmt.Println(err)
		fmt.Fprintln(w, "Could not get image file type")
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(int64(fileData.Size()), 10))

	outFile, oerr := os.Open(imagePath)
	if oerr != nil {
		fmt.Fprintln(w, "Could not open image file")
		return
	}

	io.Copy(w, outFile)

}

func mainHandler(w http.ResponseWriter, r *http.Request) {

	pageTemplate, err := template.ParseFiles("./web/templates/template.html")
	if err != nil {
		fmt.Fprintf(w, "Template failed")
		return
	}

	images := imagemanage.GetAvailableImages("./images")

	imageDataList := make([]imagemanage.BuilderConfig, 0)

	for _, imagePath := range images {
		image, ierr := imagemanage.NewVMImage("./images", imagePath)
		if ierr == nil {
			image.Config.Metadata["image_path_name"] = imagePath
			if image.CommitFlagExists() {
				image.Config.Metadata["in_progress"] = "yes"
			} else {
				image.Config.Metadata["in_progress"] = ""
			}

			imageDataList = append(imageDataList, *(image.Config))
		}
	}

	pageTemplate.Execute(w, imageDataList)
}

func main() {
	fs := http.FileServer(http.Dir("web/static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/get/", getHandler)
	http.HandleFunc("/", mainHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
