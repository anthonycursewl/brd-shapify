package services

import (
	"brd-shapify/internal/core/domain"
	"brd-shapify/internal/core/ports"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
)

type MediaService struct {
	processor ports.ImageProcessor
	storage   ports.StorageRepository
	cache     ports.CacheRepository
}

func NewMediaService(p ports.ImageProcessor, s ports.StorageRepository, c ports.CacheRepository) *MediaService {
	return &MediaService{
		processor: p,
		storage:   s,
		cache:     c,
	}
}

func (s *MediaService) Resize(img image.Image, width, height int) (image.Image, error) {
	return s.processor.Resize(img, width, height)
}

func (s *MediaService) ProcessAvatar(img image.Image, id string, req domain.ImageProcessingRequest) error {
	resized, err := s.processor.Resize(img, req.Width, req.Height)
	if err != nil {
		return fmt.Errorf("resize failed: %w", err)
	}

	var processedData []byte
	if req.Format != "" {
		processedData, err = s.processor.Convert(resized, req.Format)
	} else {
		processedData, err = s.processor.Compress(resized, req.Compression)
	}

	if err != nil {
		return fmt.Errorf("processing failed: %w", err)
	}

	return s.storage.Save(id, processedData)
}

func (s *MediaService) Save(id string, data []byte) error {
	return s.storage.Save(id, data)
}

func (s *MediaService) Load(id string) ([]byte, error) {
	return s.storage.Load(id)
}

func (s *MediaService) ConvertToFormat(img image.Image, format string) ([]byte, error) {
	if format == "" || format == "jpg" || format == "jpeg" {
		return s.processor.Compress(img, 85)
	}
	return s.processor.Convert(img, format)
}

func (s *MediaService) Compress(img image.Image, quality int) ([]byte, error) {
	return s.processor.Compress(img, quality)
}

func (s *MediaService) GetCached(key string) ([]byte, error) {
	if s.cache == nil {
		return nil, fmt.Errorf("cache not configured")
	}
	return s.cache.Get(key)
}

func (s *MediaService) SetCached(key string, data []byte) error {
	if s.cache == nil {
		return nil
	}
	return s.cache.Set(key, data)
}

func GenerateCacheKey(imgData []byte, width, height int, format string) string {
	hash := sha256.Sum256(imgData)
	hashStr := hex.EncodeToString(hash[:8])
	return fmt.Sprintf("%s_%dx%d_%s", hashStr, width, height)
}
