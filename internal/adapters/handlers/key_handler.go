package handlers

import (
	"brd-shapify/internal/adapters/storage"
	"brd-shapify/internal/core/domain"

	"github.com/gofiber/fiber/v2"
)

type KeyHandler struct {
	mongo *storage.MongoAdapter
}

func NewKeyHandler(m *storage.MongoAdapter) *KeyHandler {
	return &KeyHandler{mongo: m}
}

type KeyListResponse struct {
	Success bool             `json:"success"`
	Keys    []*domain.APIKey `json:"keys"`
	Count   int              `json:"count"`
}

func (h *KeyHandler) Create(c *fiber.Ctx) error {
	var req domain.CreateKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	apiKey, err := h.mongo.CreateKey(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(domain.CreateKeyResponse{
		Success: true,
		Key:     apiKey.Key,
		ID:      apiKey.ID,
	})
}

func (h *KeyHandler) List(c *fiber.Ctx) error {
	keys, err := h.mongo.ListKeys()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	for _, key := range keys {
		key.Key = ""
	}

	return c.JSON(KeyListResponse{
		Success: true,
		Keys:    keys,
		Count:   len(keys),
	})
}

func (h *KeyHandler) Revoke(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID is required",
		})
	}

	err := h.mongo.RevokeKey(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Key revoked",
	})
}

func (h *KeyHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID is required",
		})
	}

	err := h.mongo.DeleteKey(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Key deleted",
	})
}

func (h *KeyHandler) Get(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID is required",
		})
	}

	key, err := h.mongo.GetKeyByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	key.Key = ""
	return c.JSON(key)
}
