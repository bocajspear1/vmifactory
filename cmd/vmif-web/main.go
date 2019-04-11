package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"

	"github.com/bocajspear1/vmifactory/internal/imagemanage"
)

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
			imageDataList = append(imageDataList, *(image.Config))
		}
	}

	pageTemplate.Execute(w, imageDataList)
}

func main() {
	fs := http.FileServer(http.Dir("web/static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", mainHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
