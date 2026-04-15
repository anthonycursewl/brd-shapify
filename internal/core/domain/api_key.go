// Package domain handles the core domain entities.
package domain

import "time"

type APIKey struct {
	ID           string     `bson:"_id,omitempty" json:"id"`
	Key          string     `bson:"key" json:"key"`
	Name         string     `bson:"name" json:"name"`
	Role         string     `bson:"role" json:"role"`
	Active       bool       `bson:"active" json:"active"`
	ExpiresAt    *time.Time `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	CreatedAt    time.Time  `bson:"created_at" json:"created_at"`
	LastUsed     *time.Time `bson:"last_used,omitempty" json:"last_used,omitempty"`
	RequestCount int        `bson:"request_count" json:"request_count"`
	RateLimit    int        `bson:"rate_limit" json:"rate_limit"`
	CreatedBy    string     `bson:"created_by" json:"created_by"`
}

func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

func (k *APIKey) IsValid() bool {
	return k.Active && !k.IsExpired()
}

type CreateKeyRequest struct {
	Name      string `json:"name"`
	Role      string `json:"role"`
	RateLimit int    `json:"rate_limit"`
	ExpiresIn int    `json:"expires_in"`
}

type CreateKeyResponse struct {
	Success bool   `json:"success"`
	Key     string `json:"key,omitempty"`
	ID      string `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
}
