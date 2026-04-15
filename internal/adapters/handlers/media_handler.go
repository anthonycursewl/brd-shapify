// Package handlers handles HTTP requests for media processing.
package handlers

import (
	"brd-shapify/internal/adapters/storage"
	"brd-shapify/internal/core/domain"
	"brd-shapify/internal/core/middleware"
	"brd-shapify/internal/core/services"
	"brd-shapify/internal/logger"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type MediaHandler struct {
	service     *services.MediaService
	userAdapter *storage.UserAdapter
}

func NewMediaHandler(s *services.MediaService, ua *storage.UserAdapter) *MediaHandler {
	return &MediaHandler{service: s, userAdapter: ua}
}

type ImageRequest struct {
	Image    string `json:"image"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Format   string `json:"format,omitempty"`
	Quality  int    `json:"quality,omitempty"`
	Compress bool   `json:"compress,omitempty"`
}

type ResizeResponse struct {
	Success        bool    `json:"success"`
	ID             string  `json:"id,omitempty"`
	OriginalSize   int     `json:"original_size,omitempty"`
	CompressedSize int     `json:"new_compressed_size,omitempty"`
	ChangePercent  float64 `json:"change_percent,omitempty"`
}

type ImageResponse struct {
	Success        bool   `json:"success"`
	ID             string `json:"id"`
	Image          string `json:"image"` // base64
	OriginalSize   int    `json:"original_size"`
	CompressedSize int    `json:"new_compressed_size"`
}

func (h *MediaHandler) Resize(c *fiber.Ctx) error {
	logger.Info("[RESIZE] Starting request")
	contentType := c.Get("Content-Type")

	body := c.Body()
	logger.Info("[RESIZE] Body length: %d, Content-Type: %s", len(body), contentType)

	if strings.Contains(contentType, "application/json") {
		logger.Info("[RESIZE] Processing JSON request")
		var req ImageRequest
		if err := json.Unmarshal(body, &req); err != nil {
			logger.Info("[RESIZE] JSON parse error: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		logger.Info("[RESIZE] Request: width=%d, height=%d, format=%s, hasImage=%v",
			req.Width, req.Height, req.Format, req.Image != "")

		if req.Image == "" {
			logger.Info("[RESIZE] No image provided")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "image is required"})
		}

		imgBytes, err := base64.StdEncoding.DecodeString(req.Image)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid base64 image"})
		}

		originalSize := len(imgBytes)

		img, _, err := image.Decode(bytes.NewReader(imgBytes))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to decode image"})
		}

		var processed image.Image
		var fileData []byte

		shouldCompress := req.Compress || req.Quality > 0
		if shouldCompress && (req.Quality < 0 || req.Quality > 100) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "quality must be between 1 and 100"})
		}

		if shouldCompress {
			quality := req.Quality
			if quality <= 0 {
				quality = 85
			}
			processed = img
			if req.Width > 0 && req.Height > 0 {
				processed, err = h.service.Resize(img, req.Width, req.Height)
				if err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
				}
			}
			fileData, err = h.service.Compress(processed, quality)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
		} else if req.Width > 0 && req.Height > 0 {
			processed, err = h.service.Resize(img, req.Width, req.Height)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
			fileData, err = h.service.ConvertToFormat(processed, req.Format)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "width/height or compress/quality required"})
		}

		compressedSize := len(fileData)
		var changePercent float64
		if originalSize > 0 {
			changePercent = float64(compressedSize-originalSize) / float64(originalSize) * 100
		}

		id := generateID(req.Format)
		if err := h.service.Save(id, fileData); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		if userID, ok := c.Locals("user_id").(string); ok && userID != "" && h.userAdapter != nil {
			imgRecord := &domain.ProcessedImage{
				UserID:         userID,
				ImageID:        id,
				Format:         req.Format,
				Width:          req.Width,
				Height:         req.Height,
				OriginalSize:   originalSize,
				CompressedSize: compressedSize,
				ChangePercent:  changePercent,
			}
			go h.userAdapter.SaveProcessedImage(imgRecord)
		}

		return c.JSON(ResizeResponse{
			Success:        true,
			ID:             id,
			OriginalSize:   originalSize,
			CompressedSize: compressedSize,
			ChangePercent:  changePercent,
		})
	}

	if !middleware.ValidateMIME(contentType) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid Content-Type. Use application/json or image/*",
		})
	}

	originalSize := len(body)

	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to decode image"})
	}

	width := c.QueryInt("width", 256)
	height := c.QueryInt("height", 256)
	format := c.Query("format", "")
	quality := c.QueryInt("quality", 0)

	if quality < 0 || quality > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "quality must be between 1 and 100"})
	}

	var fileData []byte
	if quality > 0 {
		var processed image.Image
		if width > 0 && height > 0 {
			processed, err = h.service.Resize(img, width, height)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
		} else {
			processed = img
		}
		fileData, err = h.service.Compress(processed, quality)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	} else {
		processed, err := h.service.Resize(img, width, height)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		fileData, err = h.service.ConvertToFormat(processed, format)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	compressedSize := len(fileData)
	var changePercent float64
	if originalSize > 0 {
		changePercent = float64(compressedSize-originalSize) / float64(originalSize) * 100
	}

	id := generateID(format)
	if err := h.service.Save(id, fileData); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if userID, ok := c.Locals("user_id").(string); ok && userID != "" && h.userAdapter != nil {
		imgRecord := &domain.ProcessedImage{
			UserID:         userID,
			ImageID:        id,
			Format:         format,
			Width:          width,
			Height:         height,
			OriginalSize:   originalSize,
			CompressedSize: compressedSize,
			ChangePercent:  changePercent,
		}
		go h.userAdapter.SaveProcessedImage(imgRecord)
	}

	return c.JSON(fiber.Map{
		"success":             true,
		"id":                  id,
		"original_size":       originalSize,
		"new_compressed_size": compressedSize,
		"change_percent":      changePercent,
	})
}

func (h *MediaHandler) Convert(c *fiber.Ctx) error {
	contentType := c.Get("Content-Type")
	body := c.Body()

	if strings.Contains(contentType, "application/json") {
		var req ImageRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		if req.Image == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "image is required"})
		}

		imgBytes, err := base64.StdEncoding.DecodeString(req.Image)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid base64 image"})
		}

		img, _, err := image.Decode(bytes.NewReader(imgBytes))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to decode image"})
		}

		format := strings.ToLower(req.Format)
		if format != "jpg" && format != "jpeg" && format != "png" && format != "webp" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid format. Allowed: jpg, png, webp",
			})
		}

		processed, err := h.service.ConvertToFormat(img, format)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		id := generateID(format)
		if err := h.service.Save(id, processed); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(ResizeResponse{Success: true, ID: id})
	}

	if !middleware.ValidateMIME(contentType) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Content-Type"})
	}

	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to decode image"})
	}

	format := strings.ToLower(c.FormValue("format", "png"))
	processed, err := h.service.ConvertToFormat(img, format)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	id := generateID(format)
	if err := h.service.Save(id, processed); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(ResizeResponse{Success: true, ID: id})
}

func (h *MediaHandler) GetImage(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID is required"})
	}

	data, err := h.service.Load(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Image not found"})
	}

	ext := filepath.Ext(id)
	mime := middleware.GetMIME(ext)
	c.Set("Content-Type", mime)
	return c.Send(data)
}

func generateID(format string) string {
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
