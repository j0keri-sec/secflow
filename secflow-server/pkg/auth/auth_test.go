package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	svc := New("test-secret", 24*time.Hour)
	assert.NotNil(t, svc)
	assert.Equal(t, "test-secret", string(svc.secret))
}

func TestGenerateToken(t *testing.T) {
	svc := New("test-secret", 24*time.Hour)

	token, err := svc.GenerateToken("user123", "testuser", "admin")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Token should be a valid JWT format (three parts separated by dots)
	parts := 0
	for _, c := range token {
		if c == '.' {
			parts++
		}
	}
	assert.Equal(t, 2, parts, "JWT should have 3 parts (2 dots)")
}

func TestParseToken(t *testing.T) {
	svc := New("test-secret", 24*time.Hour)

	// Generate a token
	token, err := svc.GenerateToken("user123", "testuser", "admin")
	assert.NoError(t, err)

	// Parse the token
	claims, err := svc.ParseToken(token)
	assert.NoError(t, err)
	assert.Equal(t, "user123", claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, "admin", claims.Role)
}

func TestParseToken_Invalid(t *testing.T) {
	svc := New("test-secret", 24*time.Hour)

	// Test with invalid token
	_, err := svc.ParseToken("invalid-token")
	assert.Error(t, err)

	// Test with wrong secret
	svc2 := New("different-secret", 24*time.Hour)
	token, _ := svc2.GenerateToken("user123", "testuser", "admin")
	_, err = svc.ParseToken(token)
	assert.Error(t, err)
}

func TestHashPassword(t *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash, "Hash should differ from original password")

	// Hash should be bcrypt format (starts with $2)
	assert.True(t, len(hash) > 4)
	assert.Equal(t, "$2", hash[:2])
}

func TestCheckPassword(t *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)
	assert.NoError(t, err)

	// Correct password should match
	assert.True(t, CheckPassword(hash, password), "Correct password should match hash")

	// Wrong password should not match
	assert.False(t, CheckPassword(hash, "wrongpassword"), "Wrong password should not match hash")
}

func TestCheckPassword_Empty(t *testing.T) {
	hash, _ := HashPassword("testpassword")
	assert.False(t, CheckPassword(hash, ""))
	assert.False(t, CheckPassword(hash, " longerbutwrong"))
}

func TestClaimsFromCtx(t *testing.T) {
	// This test requires a Gin context, tested via middleware
}
