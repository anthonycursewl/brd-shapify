package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"brd-shapify/internal/config"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

func main() {
	log.Println("=== Testing Database Connections ===")

	cfg := config.Load()

	log.Printf("\n--- Configuration ---")
	log.Printf("MongoURI: %s", cfg.MongoURI)
	log.Printf("MongoDB:  %s", cfg.MongoDB)
	log.Printf("RedisHost: %s", cfg.RedisHost)
	log.Printf("RedisPort: %s", cfg.RedisPort)

	testMongo(cfg)

	if cfg.RedisHost != "" {
		testRedis(cfg)
	} else {
		log.Println("\n--- Redis ---")
		log.Println("Redis not configured (REDIS_HOST not set)")
	}

	log.Println("\n=== Test Complete ===")
}

func testMongo(cfg *config.Config) {
	log.Println("\n--- MongoDB ---")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Printf("ERROR: Failed to connect: %v", err)
		return
	}
	defer client.Disconnect(ctx)

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Printf("ERROR: Ping failed: %v", err)
		return
	}
	log.Println("✓ Connected successfully")

	db := client.Database(cfg.MongoDB)

	collections := []string{"users", "api_keys", "sessions"}
	for _, collName := range collections {
		coll := db.Collection(collName)
		count, err := coll.CountDocuments(ctx, bson.M{})
		if err != nil {
			log.Printf("  ✗ Collection '%s': error counting: %v", collName, err)
		} else {
			log.Printf("  ✓ Collection '%s': %d documents", collName, count)
		}
	}

	log.Println("✓ MongoDB test completed")
}

func testRedis(cfg *config.Config) {
	log.Println("\n--- Redis ---")

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisHost + ":" + cfg.RedisPort,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Ping(ctx).Err()
	if err != nil {
		log.Printf("ERROR: Failed to connect: %v", err)
		return
	}
	log.Println("✓ Connected successfully")

	serverInfo, err := client.Info(ctx, "server").Result()
	if err != nil {
		log.Printf("  Warning: Could not get info: %v", err)
	} else {
		log.Printf("  ✓ Server info:\n%s", serverInfo)
	}

	dbsize, err := client.DBSize(ctx).Result()
	if err != nil {
		log.Printf("  Warning: Could not get DB size: %v", err)
	} else {
		log.Printf("  ✓ Database size: %d keys", dbsize)
	}

	log.Println("✓ Redis test completed")
}
