package services

import (
	"brd-shapify/internal/core/domain"
	"brd-shapify/internal/core/ports"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"sync"
	"time"
)

type MediaService struct {
	processor      ports.ImageProcessor
	storage        ports.StorageRepository
	cache          ports.CacheRepository
	exifReader     ports.ExifReader
	previewGen     ports.PreviewGenerator
}

func NewMediaService(p ports.ImageProcessor, s ports.StorageRepository, c ports.CacheRepository, e ports.ExifReader, pg ports.PreviewGenerator) *MediaService {
	return &MediaService{
		processor:    p,
		storage:      s,
		cache:        c,
		exifReader:   e,
		previewGen:  pg,
	}
}

func (s *MediaService) Process(img image.Image, opts domain.ProcessOptions) ([]byte, error) {
	return s.processor.Process(img, opts)
}

func (s *MediaService) ProcessWithAutoRotate(imgData []byte, opts domain.ProcessOptions) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	if s.exifReader != nil {
		orientation, _ := s.exifReader.GetOrientation(imgData)
		if orientation != domain.OrientationNormal {
			img = s.processor.AutoRotate(img, orientation)
		}
	}

	return s.processor.Process(img, opts)
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

func (s *MediaService) GenerateBlurHash(img image.Image) (string, error) {
	if s.previewGen == nil {
		return "", fmt.Errorf("preview generator not configured")
	}
	return s.previewGen.GenerateBlurHash(img)
}

type BatchRequest struct {
	Sizes     []BatchSize `json:"sizes"`
	Format    string      `json:"format,omitempty"`
	Quality   int         `json:"quality,omitempty"`
	Watermark *domain.WatermarkConfig `json:"watermark,omitempty"`
}

type BatchSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type BatchResult struct {
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	ImageID     string `json:"image_id"`
	Size        int    `json:"size"`
	ChangePercent float64 `json:"change_percent"`
}

func (s *MediaService) ProcessBatch(imgData []byte, req BatchRequest) ([]BatchResult, error) {
	results := make([]BatchResult, len(req.Sizes))
	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	quality := req.Quality
	if quality <= 0 {
		quality = 85
	}

	for i, size := range req.Sizes {
		wg.Add(1)
		go func(idx int, width, height int) {
			defer wg.Done()

			img, _, err := image.Decode(bytes.NewReader(imgData))
			if err != nil {
				select {
				case errChan <- fmt.Errorf("failed to decode image: %w", err):
				default:
				}
				return
			}

			if s.exifReader != nil {
				orientation, _ := s.exifReader.GetOrientation(imgData)
				if orientation != domain.OrientationNormal {
					img = s.processor.AutoRotate(img, orientation)
				}
			}

			opts := domain.ProcessOptions{
				Width:    width,
				Height:   height,
				Format:   req.Format,
				Quality:  quality,
				Fit:      "fill",
				Watermark: req.Watermark,
			}

			processedData, err := s.processor.Process(img, opts)
			if err != nil {
				select {
				case errChan <- fmt.Errorf("process failed: %w", err):
				default:
				}
				return
			}

			id := generateBatchID(req.Format)
			if err := s.storage.Save(id, processedData); err != nil {
				select {
				case errChan <- fmt.Errorf("save failed: %w", err):
				default:
				}
				return
			}

			originalSize := len(imgData)
			compressedSize := len(processedData)
			var changePercent float64
			if originalSize > 0 {
				changePercent = float64(compressedSize-originalSize) / float64(originalSize) * 100
			}

			results[idx] = BatchResult{
				Width:         width,
				Height:        height,
				ImageID:       id,
				Size:          compressedSize,
				ChangePercent: changePercent,
			}
		}(i, size.Width, size.Height)
	}

	wg.Wait()

	select {
	case err := <-errChan:
		return nil, err
	default:
		return results, nil
	}
}

func generateBatchID(format string) string {
	b := make([]byte, 8)
	rand.Read(b)
	id := hex.EncodeToString(b)
	timestamp := time.Now().Unix()
	switch format {
	case "png":
		return fmt.Sprintf("%d_%s.png", timestamp, id)
	case "webp":
		return fmt.Sprintf("%d_%s.webp", timestamp, id)
	default:
		return fmt.Sprintf("%d_%s.jpg", timestamp, id)
	}
}
