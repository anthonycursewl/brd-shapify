package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"brd-shapify/internal/core/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoAdapter struct {
	client     *mongo.Client
	database   string
	collection *mongo.Collection
}

func NewMongoAdapter(uri, dbName, collectionName string) (*MongoAdapter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	collection := client.Database(dbName).Collection(collectionName)

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "key", Value: 1}},
		Options: options.Index().SetUnique(true).SetBackground(true),
	}
	_, err = collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	return &MongoAdapter{
		client:     client,
		database:   dbName,
		collection: collection,
	}, nil
}

func (m *MongoAdapter) generateKey() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return "sk_" + hex.EncodeToString(b), nil
}

func (m *MongoAdapter) CreateKey(req domain.CreateKeyRequest) (*domain.APIKey, error) {
	key, err := m.generateKey()
	if err != nil {
		return nil, err
	}

	role := req.Role
	if role == "" {
		role = "user"
	}

	rateLimit := req.RateLimit
	if rateLimit == 0 {
		rateLimit = 5
	}

	now := time.Now()
	apiKey := &domain.APIKey{
		Key:       key,
		Name:      req.Name,
		Role:      role,
		Active:    true,
		RateLimit: rateLimit,
		CreatedAt: now,
	}

	if req.ExpiresIn > 0 {
		expiresAt := now.AddDate(0, 0, req.ExpiresIn)
		apiKey.ExpiresAt = &expiresAt
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = m.collection.InsertOne(ctx, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create key: %w", err)
	}

	return apiKey, nil
}

func (m *MongoAdapter) GetKey(key string) (*domain.APIKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var apiKey domain.APIKey
	err := m.collection.FindOne(ctx, bson.M{"key": key}).Decode(&apiKey)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("key not found")
		}
		return nil, err
	}

	return &apiKey, nil
}

func (m *MongoAdapter) GetKeyByID(id string) (*domain.APIKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var apiKey domain.APIKey
	err := m.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&apiKey)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("key not found")
		}
		return nil, err
	}

	return &apiKey, nil
}

func (m *MongoAdapter) ListKeys() ([]*domain.APIKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := m.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var keys []*domain.APIKey
	if err := cursor.All(ctx, &keys); err != nil {
		return nil, err
	}

	return keys, nil
}

func (m *MongoAdapter) RevokeKey(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := m.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"active": false}},
	)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("key not found")
	}

	return nil
}

func (m *MongoAdapter) UpdateKeyUsage(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	_, err := m.collection.UpdateOne(
		ctx,
		bson.M{"key": key},
		bson.M{
			"$set": bson.M{"last_used": now},
			"$inc": bson.M{"request_count": 1},
		},
	)
	return err
}

func (m *MongoAdapter) DeleteKey(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := m.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (m *MongoAdapter) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.client.Disconnect(ctx)
}
