// Package handlers handles HTTP requests for authentication and user management.
package handlers

import (
	"brd-shapify/internal/adapters/storage"
	"brd-shapify/internal/core/domain"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	userAdapter *storage.UserAdapter
}

func NewAuthHandler(ua *storage.UserAdapter) *AuthHandler {
	return &AuthHandler{userAdapter: ua}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	ip := c.IP()
	var req domain.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Username, email and password are required",
		})
	}

	user, err := h.userAdapter.Register(req, ip)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "duplicate key") || strings.Contains(errMsg, "unique") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Email already registered",
			})
		}
		if strings.Contains(errMsg, "context canceled") || strings.Contains(errMsg, "context deadline exceeded") {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "Service temporarily unavailable, please try again",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": errMsg,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(domain.UserResponse{
		Success: true,
		User:    user,
	})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req domain.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email and password are required",
		})
	}

	token, user, err := h.userAdapter.Login(req)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "invalid credentials") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid email or password",
			})
		}
		if strings.Contains(errMsg, "account disabled") {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Account is disabled",
			})
		}
		if strings.Contains(errMsg, "context") {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "Service temporarily unavailable, please try again",
			})
		}
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": errMsg,
		})
	}

	return c.JSON(domain.AuthResponse{
		Success: true,
		Token:   token,
		Message: "Login successful",
		User:    user,
	})
}

func (h *AuthHandler) CreateAPIKey(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization header required",
		})
	}

	token := authHeader[7:]
	user, err := h.userAdapter.ValidateToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	var req domain.CreateKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	apiKey, err := h.userAdapter.CreateKeyForUser(user.ID, req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(apiKey)
}

func (h *AuthHandler) ListAPIKeys(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization header required",
		})
	}

	token := authHeader[7:]
	user, err := h.userAdapter.ValidateToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	keys, total, err := h.userAdapter.GetUserKeys(user.ID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	totalPages := (total + limit - 1) / limit

	return c.JSON(fiber.Map{
		"success":    true,
		"keys":       keys,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": totalPages,
	})
}

func (h *AuthHandler) ListImages(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization header required",
		})
	}

	token := authHeader[7:]
	user, err := h.userAdapter.ValidateToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	images, total, err := h.userAdapter.GetUserImages(user.ID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	totalPages := (total + limit - 1) / limit

	return c.JSON(fiber.Map{
		"success":    true,
		"images":     images,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": totalPages,
	})
}

func (h *AuthHandler) DeleteAPIKey(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization header required",
		})
	}

	token := authHeader[7:]
	user, err := h.userAdapter.ValidateToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	keyID := c.Params("id")
	if keyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Key ID is required",
		})
	}

	err = h.userAdapter.DeleteKey(keyID, user.ID)
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

func (h *AuthHandler) DeleteAPIKeysBatch(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization header required",
		})
	}

	token := authHeader[7:]
	user, err := h.userAdapter.ValidateToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	var req struct {
		KeyIDs []string `json:"key_ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if len(req.KeyIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "key_ids array is required",
		})
	}

	deleted, err := h.userAdapter.DeleteKeysBatch(req.KeyIDs, user.ID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"deleted": deleted,
		"message": "Keys deleted",
	})
}
