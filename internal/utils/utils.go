package utils

import (
	"errors"
	"strings"
	"time"
	"unicode"

	"brd-shapify/internal/core/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("brd-shapify-secret-key-change-in-production")

func Split(s, sep string) []string {
	if s == "" {
		return []string{}
	}

	result := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i = start - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func Trim(s string) string {
	start, end := 0, len(s)
	for start < end && unicode.IsSpace(rune(s[start])) {
		start++
	}
	for end > start && unicode.IsSpace(rune(s[end-1])) {
		end--
	}
	return s[start:end]
}

func ParseAPIKeys(keysStr string) []string {
	if keysStr == "" {
		return []string{}
	}

	parts := Split(keysStr, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = Trim(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func StringInSlice(s string, slice []string) bool {
	for _, item := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

func GenerateToken(userID, email, role string) (string, *domain.User, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", nil, err
	}

	user := &domain.User{
		ID:    userID,
		Email: email,
		Role:  role,
	}

	return tokenString, user, nil
}

func ValidateToken(tokenString string, jwtSecret []byte) (*domain.User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	email, _ := claims["email"].(string)
	userID, _ := claims["user_id"].(string)
	role, _ := claims["role"].(string)

	return &domain.User{
		ID:    userID,
		Email: email,
		Role:  role,
	}, nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
