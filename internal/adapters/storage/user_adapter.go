package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"brd-shapify/internal/core/domain"
	"brd-shapify/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("brd-shapify-secret-key-change-in-production")

func generateID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return "sk_" + hex.EncodeToString(b)
}

type UserAdapter struct {
	userColl *mongo.Collection
	keyColl  *mongo.Collection
	maxKeys  int
	ctx      context.Context
}

func NewUserAdapter(mongoURI, dbName string, maxKeys int, timeout int) (*UserAdapter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(dbName)
	userColl := db.Collection("users")
	keyColl := db.Collection("api_keys")

	userColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	keyColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "key", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	return &UserAdapter{
		userColl: userColl,
		keyColl:  keyColl,
		maxKeys:  maxKeys,
		ctx:      ctx,
	}, nil
}

func (a *UserAdapter) Register(req domain.RegisterRequest, ip string) (*domain.User, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	userID := generateID()

	user := &domain.User{
		ID:        userID,
		Username:  req.Username,
		Email:     req.Email,
		Password:  string(hashed),
		Role:      "user",
		IPCreated: ip,
		Active:    true,
		MaxKeys:   a.maxKeys,
		CreatedAt: time.Now(),
	}

	_, err = a.userColl.InsertOne(a.ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (a *UserAdapter) Login(req domain.LoginRequest) (string, *domain.User, error) {
	var user domain.User
	err := a.userColl.FindOne(a.ctx, bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	if !user.Active {
		return "", nil, errors.New("account disabled")
	}

	token, _, _ := utils.GenerateToken(user.ID, user.Email, user.Role)

	a.userColl.UpdateOne(a.ctx, bson.M{"_id": user.ID}, bson.M{
		"$set": bson.M{"last_login": time.Now()},
	})

	return token, &user, nil
}

func (a *UserAdapter) ValidateToken(tokenString string) (*domain.User, error) {
	user, err := utils.ValidateToken(tokenString, jwtSecret)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (a *UserAdapter) CreateKeyForUser(userID string, req domain.CreateKeyRequest) (*domain.APIKey, error) {
	count, err := a.keyColl.CountDocuments(a.ctx, bson.M{"created_by": userID})
	if err != nil {
		return nil, err
	}
	if count >= 30 {
		return nil, errors.New("maximum API keys reached")
	}

	key := generateKey()

	res, err := a.keyColl.InsertOne(a.ctx, bson.M{
		"key":        key,
		"name":       req.Name,
		"role":       req.Role,
		"active":     true,
		"rate_limit": req.RateLimit,
		"created_at": time.Now(),
		"created_by": userID,
	})
	if err != nil {
		return nil, err
	}

	insertedID := res.InsertedID.(primitive.ObjectID)
	keyID := insertedID.Hex()

	apiKey := &domain.APIKey{
		ID:        keyID,
		Key:       key,
		Name:      req.Name,
		Role:      req.Role,
		Active:    true,
		RateLimit: req.RateLimit,
		CreatedAt: time.Now(),
		CreatedBy: userID,
	}

	log.Printf("[CREATE_KEY] Updating user %s keys_used counter", userID)
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		log.Printf("[CREATE_KEY] Invalid user ObjectID: %v", err)
	} else {
		updateResult, err := a.userColl.UpdateOne(a.ctx, bson.M{"_id": userObjID}, bson.M{
			"$inc": bson.M{"keys_used": 1},
		})
		if err != nil {
			log.Printf("[CREATE_KEY] ERROR updating keys_used: %v", err)
		} else {
			log.Printf("[CREATE_KEY] keys_used update result: matched=%d modified=%d", updateResult.MatchedCount, updateResult.ModifiedCount)
		}
	}

	return apiKey, nil
}

func (a *UserAdapter) GetAPIKey(key string) (*domain.APIKey, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	var apiKey domain.APIKey
	err := a.keyColl.FindOne(ctx, bson.M{"key": key}).Decode(&apiKey)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("key not found")
		}
		return nil, err
	}

	return &apiKey, nil
}

func (a *UserAdapter) DeleteKey(keyID string, userID string) error {
	log.Printf("[DELETE_KEY] keyID=%s userID=%s", keyID, userID)

	objID, err := primitive.ObjectIDFromHex(keyID)
	if err != nil {
		log.Printf("[DELETE_KEY] Invalid ObjectID format: %v", err)
		return errors.New("invalid key ID format")
	}

	result, err := a.keyColl.DeleteOne(a.ctx, bson.M{"_id": objID, "created_by": userID})
	if err != nil {
		log.Printf("[DELETE_KEY] ERROR: %v", err)
		return err
	}
	log.Printf("[DELETE_KEY] Deleted count: %d", result.DeletedCount)

	if result.DeletedCount == 0 {
		return errors.New("key not found")
	}

	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		log.Printf("[DELETE_KEY] Invalid user ObjectID: %v", err)
	} else {
		updateResult, err := a.userColl.UpdateOne(a.ctx, bson.M{"_id": userObjID}, bson.M{
			"$inc": bson.M{"keys_used": -1},
		})
		if err != nil {
			log.Printf("[DELETE_KEY] ERROR updating keys_used: %v", err)
		} else {
			log.Printf("[DELETE_KEY] keys_used update result: matched=%d modified=%d", updateResult.MatchedCount, updateResult.ModifiedCount)
		}
	}

	return nil
}

func (a *UserAdapter) DeleteKeysBatch(keyIDs []string, userID string) (int, error) {
	log.Printf("[DELETE_KEYS_BATCH] count=%d userID=%s", len(keyIDs), userID)

	if len(keyIDs) == 0 {
		return 0, errors.New("no key IDs provided")
	}

	var objIDs []primitive.ObjectID
	for _, keyID := range keyIDs {
		objID, err := primitive.ObjectIDFromHex(keyID)
		if err != nil {
			log.Printf("[DELETE_KEYS_BATCH] Invalid ObjectID %s: %v", keyID, err)
			continue
		}
		objIDs = append(objIDs, objID)
	}

	if len(objIDs) == 0 {
		return 0, errors.New("no valid key IDs")
	}

	result, err := a.keyColl.DeleteMany(a.ctx, bson.M{
		"_id":        bson.M{"$in": objIDs},
		"created_by": userID,
	})
	if err != nil {
		log.Printf("[DELETE_KEYS_BATCH] ERROR: %v", err)
		return 0, err
	}

	deletedCount := int(result.DeletedCount)
	log.Printf("[DELETE_KEYS_BATCH] Deleted count: %d", deletedCount)

	if deletedCount > 0 {
		log.Printf("[DELETE_KEYS_BATCH] Updating user %s keys_used counter", userID)
		userObjID, err := primitive.ObjectIDFromHex(userID)
		if err != nil {
			log.Printf("[DELETE_KEYS_BATCH] Invalid user ObjectID: %v", err)
		} else {
			updateResult, err := a.userColl.UpdateOne(a.ctx, bson.M{"_id": userObjID}, bson.M{
				"$inc": bson.M{"keys_used": -deletedCount},
			})
			if err != nil {
				log.Printf("[DELETE_KEYS_BATCH] ERROR updating keys_used: %v", err)
			} else {
				log.Printf("[DELETE_KEYS_BATCH] keys_used update result: matched=%d modified=%d", updateResult.MatchedCount, updateResult.ModifiedCount)
			}
		}
	}

	return deletedCount, nil
}

func (a *UserAdapter) GetUserKeys(userID string, page, limit int) ([]*domain.APIKey, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	skip := (page - 1) * limit

	total, err := a.keyColl.CountDocuments(a.ctx, bson.M{"created_by": userID})
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(skip)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := a.keyColl.Find(a.ctx, bson.M{"created_by": userID}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(a.ctx)

	var keys []*domain.APIKey
	if err := cursor.All(a.ctx, &keys); err != nil {
		return nil, 0, err
	}

	return keys, int(total), nil
}

func (a *UserAdapter) UpdateKeyUsage(key string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	now := time.Now()
	_, err := a.keyColl.UpdateOne(
		ctx,
		bson.M{"key": key},
		bson.M{
			"$set": bson.M{"last_used": now},
			"$inc": bson.M{"request_count": 1},
		},
	)
	return err
}
