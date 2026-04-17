package ports

import (
	"brd-shapify/internal/core/domain"
	"image"
)

type ImageProcessor interface {
	Process(img image.Image, opts domain.ProcessOptions) ([]byte, error)
	Resize(img image.Image, width, height int) (image.Image, error)
	Compress(img image.Image, quality int) ([]byte, error)
	Convert(img image.Image, format string) ([]byte, error)
	Watermark(img image.Image, cfg domain.WatermarkConfig) (image.Image, error)
	AutoRotate(img image.Image, orientation int) image.Image
}

type ExifReader interface {
	GetOrientation(data []byte) (int, error)
	GetMetadata(data []byte) (*domain.ExifData, error)
}

type PreviewGenerator interface {
	GenerateBlurHash(img image.Image) (string, error)
}

type StorageRepository interface {
	Save(id string, data []byte) error
	Load(id string) ([]byte, error)
	Delete(id string) error
}

type CacheRepository interface {
	Get(key string) ([]byte, error)
	Set(key string, data []byte) error
	Delete(key string) error
}

type VideoProcessor interface {
	ExtractThumbnail(videoPath string, timestamp string) ([]byte, error)
	ConvertVideo(videoPath string, format string, quality int) ([]byte, error)
	ExtractAudio(videoPath string, format string) ([]byte, error)
	TrimVideo(videoPath string, start, duration string) ([]byte, error)
}
