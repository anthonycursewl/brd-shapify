// Package middleware provides middleware for the Fiber framework.
package middleware

import (
	"context"
	"strings"
	"sync"
	"time"

	"brd-shapify/internal/adapters/storage"
	"brd-shapify/internal/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

type KeyAuthMiddleware struct {
	userAdapter *storage.UserAdapter
	cache       *redis.Client
	fallback    map[string]bool
	mu          sync.RWMutex
}

func NewKeyAuth(userAdapter *storage.UserAdapter, fallback []string, cache *redis.Client) *KeyAuthMiddleware {
	validKeys := make(map[string]bool)
	for _, key := range fallback {
		validKeys[key] = true
	}
	return &KeyAuthMiddleware{
		userAdapter: userAdapter,
		cache:       cache,
		fallback:    validKeys,
	}
}

func (k *KeyAuthMiddleware) Handler(c *fiber.Ctx) error {
	key := c.Get("X-API-Key")
	if key == "" {
		key = c.Query("api_key")
	}

	logger.Info("[KEY_AUTH] Received key: %s", key)

	if key == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing API key",
		})
	}

	if k.cache != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		cached, err := k.cache.Get(ctx, "key:"+key).Result()
		if err == nil && cached == "valid" {
			logger.Info("[KEY_AUTH] Key found in cache: %s", key)
			return c.Next()
		}
	}

	if k.userAdapter != nil {
		logger.Info("[KEY_AUTH] Checking MongoDB for key: %s", key)
		apiKey, err := k.userAdapter.GetAPIKey(key)
		logger.Info("[KEY_AUTH] MongoDB result: key=%+v, error=%v", apiKey, err)
		if err == nil && apiKey.Active && !apiKey.IsExpired() {
			logger.Info("[KEY_AUTH] Key is valid: %s", key)
			k.userAdapter.UpdateKeyUsage(key)
			if k.cache != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
				defer cancel()
				k.cache.Set(ctx, "key:"+key, "valid", 24*time.Hour)
			}
			c.Locals("api_key", apiKey)
			c.Locals("user_id", apiKey.CreatedBy)
			return c.Next()
		}
	}

	k.mu.RLock()
	valid := k.fallback[key]
	k.mu.RUnlock()

	logger.Info("[KEY_AUTH] Valid in fallback: %v", valid)

	if !valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid API key",
		})
	}

	logger.Info("[KEY_AUTH] Key is valid: %s", key)
	c.Locals("api_key", key)
	return c.Next()
}

type RateLimiterMiddleware struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiterMiddleware {
	rl := &RateLimiterMiddleware{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

func (r *RateLimiterMiddleware) cleanup() {
	ticker := time.NewTicker(r.window)
	for range ticker.C {
		r.mu.Lock()
		now := time.Now()
		windowStart := now.Add(-r.window)
		for ip, times := range r.requests {
			var valid []time.Time
			for _, t := range times {
				if t.After(windowStart) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(r.requests, ip)
			} else {
				r.requests[ip] = valid
			}
		}
		r.mu.Unlock()
	}
}

func (r *RateLimiterMiddleware) Handler(c *fiber.Ctx) error {
	logger.Info("[RATE_LIMITER] Starting check")
	ip := c.IP()
	if xff := c.Get("X-Forwarded-For"); xff != "" {
		ip = strings.Split(xff, ",")[0]
	}
	logger.Info("[RATE_LIMITER] IP: %s", ip)

	r.mu.Lock()
	logger.Info("[RATE_LIMITER] Lock acquired")
	now := time.Now()
	windowStart := now.Add(-r.window)

	requests := r.requests[ip]
	var validRequests []time.Time
	for _, t := range requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}
	logger.Info("[RATE_LIMITER] Valid requests: %d/%d", len(validRequests), r.limit)

	if len(validRequests) >= r.limit {
		r.mu.Unlock()
		logger.Info("[RATE_LIMITER] Rate limit exceeded")
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "Rate limit exceeded",
		})
	}

	r.requests[ip] = append(validRequests, now)
	r.mu.Unlock()
	logger.Info("[RATE_LIMITER] Passed")

	return c.Next()
}

func ValidateMIME(contentType string) bool {
	allowed := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
		"image/gif":  true,
	}
	ct := strings.TrimSpace(strings.ToLower(contentType))
	return allowed[ct]
}

func ValidateImageSize(maxBytes int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Response().StatusCode() != 0 {
			return c.Next()
		}
		contentLength := int64(c.Response().Header.ContentLength())
		if contentLength > int64(maxBytes) && contentLength != -1 {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
				"error": "Image too large",
			})
		}
		return c.Next()
	}
}

func GetMIME(filename string) string {
	idx := strings.LastIndex(filename, ".")
	if idx == -1 {
		return "application/octet-stream"
	}
	ext := strings.ToLower(filename[idx+1:])
	mimes := map[string]string{
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"webp": "image/webp",
		"gif":  "image/gif",
	}
	if mime, ok := mimes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

type CacheAdapter struct {
	client *redis.Client
	ttl    time.Duration
}

func NewCacheAdapter(addr string, password string, db int) (*CacheAdapter, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  200 * time.Millisecond,
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 200 * time.Millisecond,
		PoolSize:     10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &CacheAdapter{
		client: client,
		ttl:    24 * time.Hour,
	}, nil
}

func (c *CacheAdapter) Get(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (c *CacheAdapter) Set(key string, data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	return c.client.Set(ctx, key, data, c.ttl).Err()
}

func (c *CacheAdapter) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	return c.client.Del(ctx, key).Err()
}

func (c *CacheAdapter) GenerateCacheKey(imageHash string, width, height int, format string) string {
	return strings.Join([]string{imageHash, string(rune(width)), string(rune(height)), format}, "_")
}
