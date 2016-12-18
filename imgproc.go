package main

import (
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/nfnt/resize"
	"github.com/shurcooL/sanitized_anchor_name"
)

const (
	imagePath  = "img"
	thumbPath  = "img/thumb"
	uploadPath = "uploads"
	origPath   = "orig"
)

func writeJpeg(img image.Image, fullPath string) error {
	out, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	jpeg.Encode(out, img, nil)
	out.Close()
	return nil
}

func ptr(s string) *string {
	return &s
}

func processOne(upload *StoredImage) error {
	fileName := path.Base(*upload.UploadPath)
	upload.OrigPath = ptr(filepath.Join(origPath, fileName))
	upload.DisplayPath = ptr(filepath.Join(imagePath, fileName))
	upload.ThumbPath = ptr(filepath.Join(thumbPath, fileName))
	// open source image
	srcImgFile, err := os.Open(*upload.UploadPath)
	if err != nil {
		return err
	}
	// decode jpeg into image.Image
	// TODO: support other image formats
	fullSizeImg, err := jpeg.Decode(srcImgFile)
	if err != nil {
		return err
	}
	srcImgFile.Close()
	im := resize.Thumbnail(960, 720, fullSizeImg, resize.Lanczos3)
	th := resize.Thumbnail(348, 464, fullSizeImg, resize.Lanczos3)
	err = writeJpeg(im, *upload.DisplayPath)
	if err != nil {
		return err
	}
	err = writeJpeg(th, *upload.ThumbPath)
	if err != nil {
		return err
	}
	err = os.Rename(*upload.UploadPath, *upload.OrigPath)
	if err != nil {
		return err
	}
	upload.UploadPath = nil
	now := time.Now()
	upload.ProcessedAt = &now
	tx := newTx(db)
	if tx.db.Error != nil {
		return tx.db.Error
	}
	defer tx.rollback()
	err = db.Save(upload).Error
	if err != nil {
		return err
	}
	var album Album
	err = db.Where("name = ?", "default").Find(&album).Error
	if err != nil {
		return err
	}
	image := Image{
		Name:        fileName,
		URLName:     sanitized_anchor_name.Create(fileName),
		Description: fileName,
		StoreID:     upload.ID,
		AlbumID:     album.ID,
	}
	err = db.Save(&image).Error
	if err != nil {
		return err
	}
	tx.commit()
	// TODO: clean up uploads/<uuid> dir after done
	return nil
}

func imgProcJob() {
	var uploads []*StoredImage
	db.Where("processed_at is ?", gorm.Expr("NULL")).Find(&uploads)
	println(len(uploads))
	for _, upload := range uploads {
		err := processOne(upload)
		if err != nil {
			logger.Printf("Error ingesting %s: %s", *upload.UploadPath, err.Error())
		}
	}
}

func ingestImages(src, img, thumb string) error {
	files, err := ioutil.ReadDir(src)
	if err != nil {
		log.Fatalf("read dir: %s\n", err)
	}
	for _, file := range files {
		if !file.IsDir() {
			log.Printf("Processing %s...", file.Name())
			up := path.Join(src, file.Name())
			err = processOne(&StoredImage{
				UploadPath: &up,
			})
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	return nil
}
