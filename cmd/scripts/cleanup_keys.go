package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	dbName := os.Getenv("MONGO_DB")
	if dbName == "" {
		dbName = "brd_shapify"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect(ctx)

	keyColl := client.Database(dbName).Collection("api_keys")
	userColl := client.Database(dbName).Collection("users")

	fmt.Println("=== API Keys Cleanup Script ===")
	fmt.Println()

	// Find all keys
	cursor, err := keyColl.Find(ctx, bson.M{})
	if err != nil {
		log.Fatalf("Failed to find keys: %v", err)
	}
	defer cursor.Close(ctx)

	var keys []bson.M
	if err := cursor.All(ctx, &keys); err != nil {
		log.Fatalf("Failed to decode keys: %v", err)
	}

	fmt.Printf("Found %d API keys\n\n", len(keys))

	if len(keys) == 0 {
		fmt.Println("No keys to delete.")
		return
	}

	// Show keys with their IDs
	fmt.Println("Keys to delete:")
	for i, k := range keys {
		id := k["_id"]
		key := k["key"]
		name := k["name"]
		createdBy := k["created_by"]
		createdAt := k["created_at"]

		idStr := fmt.Sprintf("%v", id)
		if len(idStr) == 32 {
			fmt.Printf("  [%d] ID(type=string): %s | key: %s | name: %v | created_by: %v | created_at: %v\n",
				i, idStr, key, name, createdBy, createdAt)
		} else {
			fmt.Printf("  [%d] ID(type=ObjectID): %s | key: %s | name: %v | created_by: %v | created_at: %v\n",
				i, idStr, key, name, createdBy, createdAt)
		}
	}
	fmt.Println()

	// Ask for confirmation
	fmt.Print("Delete all these keys? (yes/no): ")
	var confirm string
	fmt.Scanln(&confirm)

	if confirm != "yes" {
		fmt.Println("Cancelled.")
		return
	}

	// Delete all keys
	result, err := keyColl.DeleteMany(ctx, bson.M{})
	if err != nil {
		log.Fatalf("Failed to delete keys: %v", err)
	}

	fmt.Printf("\nDeleted %d API keys\n", result.DeletedCount)

	// Reset keys_used for all users
	userResult, err := userColl.UpdateMany(ctx, bson.M{}, bson.M{
		"$set": bson.M{"keys_used": 0},
	})
	if err != nil {
		log.Fatalf("Failed to reset keys_used: %v", err)
	}

	fmt.Printf("Reset keys_used for %d users\n", userResult.ModifiedCount)
	fmt.Println("\nDone! You can now create new keys with MongoDB ObjectIDs.")
}
