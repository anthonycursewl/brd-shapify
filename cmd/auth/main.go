package main

import (
	"brd-shapify/internal/adapters/handlers"
	"brd-shapify/internal/adapters/storage"
	"brd-shapify/internal/core/middleware"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	app := fiber.New()
	app.Use(recover.New())

	mongoURI := os.Getenv("MONGO_URI")
	mongoDB := os.Getenv("MONGO_DB")

	if mongoURI == "" {
		log.Fatal("MONGO_URI required")
	}

	userAdapter, err := storage.NewUserAdapter(mongoURI, mongoDB, 5, 30)
	if err != nil {
		log.Fatalf("MongoDB error: %v", err)
	}
	log.Println("MongoDB connected")

	authHandler := handlers.NewAuthHandler(userAdapter)
	keyAuth := middleware.NewKeyAuth(nil, []string{"dev-key"}, nil)

	app.Post("/auth/register", authHandler.Register)
	app.Post("/auth/login", authHandler.Login)

	protected := app.Group("/api", keyAuth.Handler)
	protected.Post("/keys", authHandler.CreateAPIKey)
	protected.Get("/keys", authHandler.ListAPIKeys)

	log.Println("Auth server on :8081")
	app.Listen(":8081")
}
