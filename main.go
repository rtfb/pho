package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goods/httpbuf"
	"github.com/gorilla/pat"
	"github.com/gorilla/sessions"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/rtfb/bark"
	uuid "github.com/satori/go.uuid"
)

const (
	maxFileSize = 50 * 1024 * 1024
)

var (
	ingestPath string
	logger     *bark.Logger
	db         *gorm.DB
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

func collectImagesDB() []imageEntry {
	var dbImages []StoredImage
	err := db.Where("processed_at is not null").Find(&dbImages).Error
	logger.LogIf(err)
	var entries []imageEntry
	for _, im := range dbImages {
		entries = append(entries, imageEntry{
			Image: *im.DisplayPath,
			Thumb: *im.ThumbPath,
		})
	}
	return entries
}

func indexHandler(w http.ResponseWriter, r *http.Request) error {
	data := map[string]interface{}{
		"entries": collectImagesDB(),
	}
	tmpl, err := template.New("index.html").ParseFiles("./tmpl/index.html")
	var out bytes.Buffer
	err = tmpl.Execute(&out, data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprint(w, out.String())
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
	mr, err := r.MultipartReader()
	if err != nil {
		return logger.LogIf(err)
	}
	files := ""
	uploadRoot := filepath.Join(uploadPath, uuid.NewV4().String())
	err = os.MkdirAll(uploadRoot, 0766)
	if err != nil {
		return logger.LogIf(err)
	}
	part, err := mr.NextPart()
	for err == nil {
		if name := part.FormName(); name != "" {
			if part.FileName() != "" {
				files += fmt.Sprintf("[foo]: /%s", part.FileName())
				handleUpload(r, part, uploadRoot)
			}
		}
		part, err = mr.NextPart()
	}
	w.Write([]byte(files))
	return nil
}

func handleUpload(r *http.Request, p *multipart.Part, root string) {
	defer func() {
		if rec := recover(); rec != nil {
			logger.Println(rec)
		}
	}()
	lr := &io.LimitedReader{R: p, N: maxFileSize + 1}
	filename := filepath.Join(root, p.FileName())
	fo, err := os.Create(filename)
	if err != nil {
		logger.Printf("err writing %q!, err = %s\n", filename, err.Error())
	}
	defer fo.Close()
	w := bufio.NewWriter(fo)
	_, err = io.Copy(w, lr)
	if err != nil {
		logger.Printf("err writing %q!, err = %s\n", filename, err.Error())
	}
	if err = w.Flush(); err != nil {
		logger.Printf("err flushing writer for %q!, err = %s\n", filename, err.Error())
	}
	store := StoredImage{
		UploadPath: &filename,
		UploadedAt: time.Now(),
	}
	err = db.Save(&store).Error
	if err != nil {
		logger.Printf("err inserting DB record for %q!, err = %s\n", filename, err.Error())
	}
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
	r.Add(P, "/upload-file", mkHandler(uploadFileHandler)).Name("upload-file")
	r.Add(G, "/", mkHandler(indexHandler)).Name("home_page")
	return r
}

func initDB() *gorm.DB {
	dialect := "postgres"
	conn := "dbname=pho sslmode=disable user=tstusr password=tstpwd"
	logDbConn(dialect, conn)
	db, err := gorm.Open(dialect, conn)
	if err != nil {
		panic(err)
	}
	err = db.DB().Ping()
	if err != nil {
		panic(err)
	}
	// db.LogMode(conf.LogSQL)
	db.LogMode(true)
	db.SingularTable(true)
	return db
}

func logDbConn(dialect, conn string) {
	if dialect == "postgres" {
		conn = censorPostgresConnStr(conn)
	}
	logger.Printf("Connecting to %q DB via conn %q\n", dialect, conn)
}

func censorPostgresConnStr(conn string) string {
	parts := strings.Split(conn, " ")
	var newParts []string
	for _, part := range parts {
		if strings.HasPrefix(part, "password=") {
			newParts = append(newParts, "password=***")
		} else {
			newParts = append(newParts, part)
		}
	}
	return strings.Join(newParts, " ")
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
	logger = bark.AppendFile("pho.log")
	db = initDB()
	imgProcJob()
	addr := ":8080"
	logger.Printf("The server is listening on %s...", addr)
	logger.LogIf(http.ListenAndServe(addr, initRoutes()))
}
