package domain

import "image"
import "time"

type MediaFile struct {
	ID     string
	Data   image.Image
	Format string
	Width  int
	Height int
}

type ImageProcessingRequest struct {
	Width       int
	Height      int
	Compression int
	Format      string
}

type ProcessedImage struct {
	ID             string    `bson:"_id" json:"id"`
	UserID         string    `bson:"user_id" json:"user_id"`
	ImageID        string    `bson:"image_id" json:"image_id"`
	Format         string    `bson:"format" json:"format"`
	Width          int       `bson:"width" json:"width"`
	Height         int       `bson:"height" json:"height"`
	OriginalSize   int       `bson:"original_size" json:"original_size"`
	CompressedSize int       `bson:"compressed_size" json:"compressed_size"`
	ChangePercent  float64   `bson:"change_percent" json:"change_percent"`
	CreatedAt      time.Time `bson:"created_at" json:"created_at"`
}
