package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"brd-shapify/internal/adapters/imaging"
	"brd-shapify/internal/core/services"
	"github.com/gofiber/fiber/v2"
)

type VideoHandler struct {
	processor *imaging.VideoProcessorAdapter
	service   *services.MediaService
}

func NewVideoHandler(v *imaging.VideoProcessorAdapter, s *services.MediaService) *VideoHandler {
	return &VideoHandler{
		processor: v,
		service:   s,
	}
}

type VideoRequest struct {
	Video     string `json:"video"`
	Format    string `json:"format,omitempty"`
	Quality   int    `json:"quality,omitempty"`
	Start     string `json:"start,omitempty"`
	Duration  string `json:"duration,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type VideoResponse struct {
	Success bool   `json:"success"`
	ID      string `json:"id,omitempty"`
}

func (h *VideoHandler) ExtractThumbnail(c *fiber.Ctx) error {
	var req VideoRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
	}

	videoData, err := base64.StdEncoding.DecodeString(req.Video)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid video data"})
	}

	thumb, err := h.processor.ExtractThumbnail(videoData, req.Timestamp)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	id := fmt.Sprintf("%d_thumb.jpg", time.Now().UnixNano())
	if err := h.service.Save(id, thumb); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(VideoResponse{Success: true, ID: id})
}

func (h *VideoHandler) Convert(c *fiber.Ctx) error {
	var req VideoRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
	}

	videoData, _ := base64.StdEncoding.DecodeString(req.Video)
	format := req.Format
	if format == "" {
		format = "mp4"
	}

	result, err := h.processor.ConvertVideo(videoData, format, req.Quality)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	id := fmt.Sprintf("%d_video.%s", time.Now().UnixNano(), format)
	if err := h.service.Save(id, result); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(VideoResponse{Success: true, ID: id})
}

func (h *VideoHandler) ExtractAudio(c *fiber.Ctx) error {
	var req VideoRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
	}

	videoData, _ := base64.StdEncoding.DecodeString(req.Video)
	format := req.Format
	if format == "" {
		format = "mp3"
	}

	audio, err := h.processor.ExtractAudio(videoData, format)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	id := fmt.Sprintf("%d_audio.%s", time.Now().UnixNano(), format)
	if err := h.service.Save(id, audio); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(VideoResponse{Success: true, ID: id})
}

func (h *VideoHandler) Trim(c *fiber.Ctx) error {
	var req VideoRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
	}

	videoData, _ := base64.StdEncoding.DecodeString(req.Video)

	result, err := h.processor.TrimVideo(videoData, req.Start, req.Duration)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	id := fmt.Sprintf("%d_trim.mp4", time.Now().UnixNano())
	if err := h.service.Save(id, result); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(VideoResponse{Success: true, ID: id})
}
