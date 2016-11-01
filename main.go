package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"path/filepath"
)

const (
	imagePath = "./img"
	thumbPath = "./img/thumb"
)

type imageEntry struct {
	Image string
	Thumb string
}

func collectImages(imagePath, thumbPath string) []imageEntry {
	files, err := ioutil.ReadDir(imagePath)
	if err != nil {
		log.Fatal(err)
	}
	var entries []imageEntry
	for _, file := range files {
		if !file.IsDir() {
			entries = append(entries, imageEntry{
				Image: filepath.Join(imagePath, file.Name()),
				Thumb: filepath.Join(thumbPath, file.Name()),
			})
		}
	}
	return entries
}

func main() {
	data := map[string]interface{}{
		"entries": collectImages(imagePath, thumbPath),
	}
	tmpl, err := template.New("index.html").ParseFiles("./tmpl/index.html")
	var out bytes.Buffer
	err = tmpl.Execute(&out, data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out.String())
}
