// Package config handles the application configuration loading and validation.
package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	MongoURI        string
	MongoDB         string
	MongoMaxPoolSize    int
	MongoMinPoolSize    int
	MongoMaxConnIdleTime int
	RedisHost       string
	RedisPort       string
	RedisPassword   string
	RedisDB         int
	JWTSecret       string
	Port            string
	WatermarkPath   string
	Environment     string
}

func init() {
	loadEnvFile()
}

func loadEnvFile() {
	data, err := os.ReadFile(".env")
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if idx := strings.Index(line, "="); idx > 0 {
			key := line[:idx]
			value := line[idx+1:]
			os.Setenv(key, value)
		}
	}
}

func Load() *Config {
	return &Config{
		MongoURI:       getEnv("MONGO_URI", ""),
		MongoDB:        getEnv("MONGO_DB", "shapify"),
		MongoMaxPoolSize:    getEnvInt("MONGO_MAX_POOL_SIZE", 100),
		MongoMinPoolSize:    getEnvInt("MONGO_MIN_POOL_SIZE", 10),
		MongoMaxConnIdleTime: getEnvInt("MONGO_MAX_CONN_IDLE_TIME", 300),
		RedisHost:      getEnv("REDIS_HOST", ""),
		RedisPort:      getEnv("REDIS_PORT", "6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		RedisDB:        getEnvInt("REDIS_DB", 0),
		JWTSecret:      getEnv("JWT_SECRET", ""),
		Port:           getEnv("PORT", "8080"),
		WatermarkPath:  getEnv("WATERMARK_PATH", ""),
		Environment:    getEnv("ENVIRONMENT", "development"),
	}
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intVal int
		fmt.Sscanf(value, "%d", &intVal)
		return intVal
	}
	return defaultValue
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
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	return nil
}
