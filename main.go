package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/nfnt/resize"
)

const (
	imagePath = "./img"
	thumbPath = "./img/thumb"
)

var ingestPath string

type imageEntry struct {
	Image string
	Thumb string
}

func init() {
	const usage = "A path to ingest images from"
	flag.StringVar(&ingestPath, "ingest", "", usage)
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

func writeJpeg(img image.Image, fullPath string) error {
	out, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	jpeg.Encode(out, img, nil)
	out.Close()
	return nil
}

func processOne(srcPath, imgPath, thumbPath, fileName string) error {
	// open source image
	srcImgFile, err := os.Open(filepath.Join(srcPath, fileName))
	if err != nil {
		log.Fatalf("open: %s\n", err)
	}
	// decode jpeg into image.Image
	fullSizeImg, err := jpeg.Decode(srcImgFile)
	if err != nil {
		log.Fatalf("decode: %s\n", err)
	}
	srcImgFile.Close()
	im := resize.Thumbnail(960, 720, fullSizeImg, resize.Lanczos3)
	th := resize.Thumbnail(348, 464, fullSizeImg, resize.Lanczos3)
	err = writeJpeg(im, filepath.Join(imgPath, fileName))
	if err != nil {
		log.Fatalf("write image: %s\n", err)
	}
	err = writeJpeg(th, filepath.Join(thumbPath, fileName))
	if err != nil {
		log.Fatalf("write thumbnail: %s\n", err)
	}
	return nil
}

func ingestImages(src, img, thumb string) error {
	files, err := ioutil.ReadDir(src)
	if err != nil {
		log.Fatalf("read dir: %s\n", err)
	}
	for _, file := range files {
		if !file.IsDir() {
			log.Printf("Processing %s...", file.Name())
			err = processOne(src, img, thumb, file.Name())
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	return nil
}

func main() {
	flag.Parse()
	if ingestPath != "" {
		// TODO: expand glob before using
		_, err := os.Stat(ingestPath)
		if err != nil {
			log.Fatal(err)
		}
		if os.IsNotExist(err) {
			fmt.Printf("Path does not exist: %s\n", ingestPath)
			return
		}
		err = ingestImages(ingestPath, imagePath, thumbPath)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
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
