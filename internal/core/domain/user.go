package domain

import "time"

type User struct {
	ID        string     `bson:"_id" json:"id"`
	Username  string     `bson:"username" json:"username"`
	Email     string     `bson:"email" json:"email"`
	Password  string     `bson:"password" json:"-"`
	Role      string     `bson:"role" json:"role"`
	IPCreated string     `bson:"ip_created" json:"ip_created"`
	Active    bool       `bson:"active" json:"active"`
	KeysUsed  int        `bson:"keys_used" json:"keys_used"`
	MaxKeys   int        `bson:"max_keys" json:"max_keys"`
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	LastLogin *time.Time `bson:"last_login,omitempty" json:"last_login,omitempty"`
}

func (u *User) CanCreateKey() bool {
	return u.KeysUsed < u.MaxKeys
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	Message string `json:"message,omitempty"`
	User    *User  `json:"user,omitempty"`
}

type UserResponse struct {
	Success bool  `json:"success"`
	User    *User `json:"user,omitempty"`
}
