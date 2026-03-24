// Package auth provides JWT-based authentication and bcrypt password hashing.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Claims is the payload embedded in every JWT.
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Service handles JWT generation/validation and password operations.
type Service struct {
	secret []byte
	expire time.Duration
}

// New creates an auth Service.
func New(secret string, expire time.Duration) *Service {
	return &Service{
		secret: []byte(secret),
		expire: expire,
	}
}

// GenerateToken issues a signed JWT for the given user.
func (s *Service) GenerateToken(userID, username, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expire)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.secret)
}

// ParseToken validates a token string and returns the embedded claims.
func (s *Service) ParseToken(tokenStr string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := t.Claims.(*Claims)
	if !ok || !t.Valid {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

// HashPassword returns a bcrypt hash of the plaintext password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

// CheckPassword compares a bcrypt hash against a plaintext password.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// ClaimsFromCtx extracts JWT claims injected by the JWTAuth middleware.
// It uses a gin.Context interface subset to stay decoupled from gin imports.
type claimsKeyer interface {
	GetString(key string) string
}

// ClaimsFromCtx retrieves the *Claims stored in a gin.Context by the middleware.
// Import this in handlers via: claims := auth.ClaimsFromCtx(c)
func ClaimsFromCtx(c interface{ Get(string) (any, bool) }) *Claims {
	v, _ := c.Get("claims")
	if claims, ok := v.(*Claims); ok {
		return claims
	}
	return &Claims{}
}
