package config

import (
	"fmt"
	"os"
)

type Config struct {
	MongoURI  string
	MongoDB   string
	RedisHost string
	RedisPort string
	JWTSecret string
	Port      string
}

func Load() *Config {
	return &Config{
		MongoURI:  getEnv("MONGO_URI", ""),
		MongoDB:   getEnv("MONGO_DB", "shapify"),
		RedisHost: getEnv("REDIS_HOST", ""),
		RedisPort: getEnv("REDIS_PORT", "6379"),
		JWTSecret: getEnv("JWT_SECRET", "brd-shapify-secret-key-2024!"),
		Port:      getEnv("PORT", "8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) Validate() error {
	if c.MongoURI == "" {
		return fmt.Errorf("MONGO_URI is required")
	}
	return nil
}
