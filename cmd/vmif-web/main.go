package main

import (
	"flag"
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

// Handles getting downloading images
func getHandler(w http.ResponseWriter, r *http.Request) {
	// Rudimentary security protections
	reqPath := strings.ReplaceAll(r.URL.Path, "..", "")

	// Parse the incoming URL
	sections := strings.Split(reqPath[len("/get/"):], "/")

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

	// Check if the file actually exists
	fileData, serr := os.Stat(imagePath)
	if serr != nil {
		log.Panicln("Could not find file " + imagePath)
		fmt.Fprintln(w, "The image file was not found, please contact the administrator")
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+fileData.Name())

	contentType, err := getFileContentType(imagePath)
	if err != nil {
		log.Panicln("Failed to get filetype of " + imagePath)
		fmt.Fprintln(w, "Could not get image file type")
		return
	}

	// Set more headers for the file
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(int64(fileData.Size()), 10))

	log.Println("Downloading " + imagePath)

	outFile, oerr := os.Open(imagePath)
	if oerr != nil {
		fmt.Fprintln(w, "Could not open image file")
		return
	}

	io.Copy(w, outFile)

}

// Handles the index page
func mainHandler(w http.ResponseWriter, r *http.Request) {

	pageTemplate, err := template.ParseFiles("./web/templates/template.html")
	if err != nil {
		fmt.Fprintf(w, "Template failed")
		return
	}
	log.Println("Accessed index")

	// Prepare image data for the template
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

	// Fill out the template and return
	pageTemplate.Execute(w, imageDataList)
}

func main() {

	// Parse options
	var listenAt = flag.String("listen", ":8080", "Address:port to listen at")
	var logFilePath = flag.String("logfile", "./vmif-web.log", "File to log to")

	flag.Parse()

	// Setup logging to Stdout and file
	logFile, err := os.OpenFile(*logFilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0660)
	if err != nil {
		panic(err)
	}

	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	// Setup the web server
	fs := http.FileServer(http.Dir("web/static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/get/", getHandler)
	http.HandleFunc("/", mainHandler)

	// Start the web server
	log.Println("Starting server, listening at " + *listenAt)
	log.Fatal(http.ListenAndServe(*listenAt, nil))
}
