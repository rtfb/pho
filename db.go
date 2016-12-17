package main

import (
	"time"

	"github.com/jinzhu/gorm"
)

// Image describes an image to be displayed.
type Image struct {
	ID          string `gorm:"column:id"`
	Name        string `gorm:"column:name"`
	URLName     string `gorm:"column:url_name"`
	Description string `gorm:"column:description"`
	AlbumID     string `gorm:"column:album_id"`
	StoreID     string `gorm:"column:store_id"`
}

// StoredImage contains image details. Initially, an image is uploaded to
// UploadPath and lives there until it gets picked up by the image processor.
// The processor then picks it up, makes downscaled copies for thumbnail and
// display (stores them in ThumbPath and DisplayPath, accordingly), moves the
// original to OrigPath and sets ProcessedAt.
type StoredImage struct {
	ID          string     `gorm:"column:id"`
	UploadPath  *string    `gorm:"column:upload_path"`
	OrigPath    *string    `gorm:"column:orig_path"`
	ThumbPath   *string    `gorm:"column:thumb_path"`
	DisplayPath *string    `gorm:"column:display_path"`
	UploadedAt  time.Time  `gorm:"column:uploaded_at"`
	ProcessedAt *time.Time `gorm:"column:processed_at"`
}

// Album stores collections of images.
type Album struct {
	ID   string `gorm:"column:id"`
	Name string `gorm:"column:name"`
	URL  string `gorm:"column:url"`
}

// Tx wraps Gorm transaction
type Tx struct {
	db   *gorm.DB
	open bool
}

func (t *Tx) commit() {
	if t.open {
		t.db.Commit()
		t.open = false
	}
}

func (t *Tx) rollback() {
	if t.open {
		t.db.Rollback()
		t.open = false
	}
}

func newTx(db *gorm.DB) *Tx {
	tx := db.Begin()
	return &Tx{
		db:   tx,
		open: true,
	}
}
