package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

var startTime = time.Now()

func Home(c *fiber.Ctx) error {
	environment := c.Locals("environment")
	env := "development"
	if e, ok := environment.(string); ok {
		env = e
	}

	uptime := time.Since(startTime).Round(time.Second).String()

	return c.JSON(fiber.Map{
		"system_info": fiber.Map{
			"status":      "🟢 All systems operational",
			"environment": env,
			"uptime":      uptime,
			"timestamp":   time.Now().Format(time.RFC3339),
		},
		"api_details": fiber.Map{
			"name":        "Shapify API",
			"version":     "v1.0.0",
			"description": "High-performance image processing API powered by Hexagonal Architecture.",
			"author":      "Anthony Cursewl",
			"ai_partner":  "minimax 2.5",
		},
		"ascii_art": []string{
			`   _____ __  _____    ____  ___________  __ `,
			`  / ___// / / /   |  / __ \/  _/ ____/ \/ / `,
			`  \__ \/ /_/ / /| | / /_/ // // /_    \  /  `,
			` ___/ / __  / ___ |/ ____// // __/    / /   `,
			`/____/_/ /_/_/  |_/_/   /___/_/      /_/    `,
			`                                            `,
			`    Pixel-perfect processing in Go      `,
		},
		"documentation": fiber.Map{
			"base_url":    "pending to post :/",
			"swagger_ui":  "/api/docs",
			"github_repo": "https://github.com/anthonycursewl/brd-shapify",
		},
		"quickstart": fiber.Map{
			"step_1": "Get an API key: POST /api/keys",
			"step_2": "Try your first request:",
			"example_curl": `curl -X POST https://{{PENDING_TO_SET}}/v1/images/resize \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -F "image=@photo.jpg" \
  -F "width=800" \
  -F "format=webp"`,
		},
		"endpoints": fiber.Map{
			"auth & keys": []string{
				"POST   /auth/register",
				"POST   /auth/login",
				"POST   /api/keys",
				"GET    /api/keys",
				"DELETE /api/keys/:id",
				"POST   /api/keys/batch-delete",
			},
			"image_processing": []string{
				"POST   /v1/images/resize    - Resizes keeping aspect ratio",
				"POST   /v1/images/convert   - Converts between JPEG/PNG/WebP",
				"POST   /v1/images/compact   - Compresses & strips EXIF metadata",
				"POST   /v1/images/batch     - Generates srcset sizes concurrently",
				"POST   /v1/images/blurhash  - Generates lightweight blur previews",
				"GET    /v1/images/:id       - Retrieves a processed image",
			},
		},
		"capabilities": fiber.Map{
			"processing": []string{
				"Smart resize & crop",
				"Format conversion (JPEG, PNG, WebP)",
				"Compression with quality control",
				"Auto-rotation (EXIF orientation detection)",
				"Watermarking & Overlays",
				"BlurHash encoding",
			},
			"infrastructure": []string{
				"Batch processing using Goroutines ⚡",
				"Hexagonal Architecture (Ports & Adapters)",
				"Rate limiting & Size limiting protection",
				"API Key authentication (MongoDB + Redis cache)",
			},
},
		"developer_experience": fiber.Map{
			"x_powered_by": "Caffeine, Go routines & Fiber",
			"fun_facts": []string{
				"Made with 💖 by Anthony & Minimax 2.5",
				"Zero CGO dependencies for WebP conversion!",
				"Automatically fixes those upside-down mobile photos.",
			},
			"quote_of_the_day": "Talk is cheap. Show me the code. - Linus Torvalds",
		},
	})
}
