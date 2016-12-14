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
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/goods/httpbuf"
	"github.com/gorilla/pat"
	"github.com/gorilla/sessions"
	"github.com/nfnt/resize"
	"github.com/rtfb/bark"
)

const (
	imagePath = "./img"
	thumbPath = "./img/thumb"
)

var (
	ingestPath string
	logger     *bark.Logger
)

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

func indexHandler(w http.ResponseWriter, r *http.Request) error {
	data := map[string]interface{}{
		"entries": collectImages(imagePath, thumbPath),
	}
	tmpl, err := template.New("index.html").ParseFiles("./tmpl/index.html")
	var out bytes.Buffer
	err = tmpl.Execute(&out, data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(w, out.String())
	return nil
}

func uploadHandler(w http.ResponseWriter, r *http.Request) error {
	tmpl, err := template.New("upload.html").ParseFiles("./tmpl/upload.html")
	var out bytes.Buffer
	err = tmpl.Execute(&out, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprint(w, out.String())
	return nil
}

func uploadFileHandler(w http.ResponseWriter, r *http.Request) error {
	fmt.Fprint(w, "TODO")
	return nil
}

type handlerFunc func(http.ResponseWriter, *http.Request) error

type handler struct {
	h     handlerFunc
	logRq bool
}

func (h handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	startTime := time.Now().UTC()
	if h.logRq {
		defer logger.LogRq(req, startTime)
	}
	// We're using httpbuf here to satisfy an unobvious requirement:
	// sessions.Save() *must* be called before anything is written to
	// ResponseWriter. So we pass this buffer in place of writer here, then
	// call Save() and finally apply the buffer to the real writer.
	buf := new(httpbuf.Buffer)
	err := h.h(buf, req)
	if err != nil {
		internalError(w, req, err, "Error in handler")
		return
	}
	//save the session
	if err = sessions.Save(req, w); err != nil {
		internalError(w, req, err, "Session save err")
		return
	}
	buf.Apply(w)
}

func internalError(w http.ResponseWriter, req *http.Request, err error, prefix string) error {
	logger.Printf("%s: %s", prefix, err.Error())
	return performStatus(w, req, http.StatusInternalServerError)
}

//PerformStatus runs the passed in status on the request and calls the appropriate block
func performStatus(w http.ResponseWriter, req *http.Request, status int) error {
	return performSimpleStatus(w, status)
}

func performSimpleStatus(w http.ResponseWriter, status int) error {
	w.Write([]byte(fmt.Sprintf("HTTP Error %d\n", status)))
	return nil
}

func initRoutes() *pat.Router {
	const (
		G = "GET"
		P = "POST"
	)
	r := pat.New()
	mkHandler := func(f handlerFunc) *handler {
		return &handler{h: f, logRq: true}
	}
	r.Add(G, "/img/", http.FileServer(http.Dir("."))).Name("img")
	r.Add(G, "/bower_components/", http.FileServer(http.Dir("."))).Name("bower_components")
	r.Add(G, "/up", mkHandler(uploadHandler)).Name("upload")
	r.Add(P, "/uploadFile", mkHandler(uploadFileHandler)).Name("upload_file")
	r.Add(G, "/", mkHandler(indexHandler)).Name("home_page")
	return r
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
	pwd, err := os.Getwd()
	if err != nil {
		panic("wtf")
	}
	println(pwd)
	logger = bark.AppendFile("pho.log")
	addr := ":8080"
	logger.Printf("The server is listening on %s...", addr)
	logger.LogIf(http.ListenAndServe(addr, initRoutes()))
}
