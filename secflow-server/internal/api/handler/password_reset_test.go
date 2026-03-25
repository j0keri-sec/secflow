package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/secflow/server/internal/model"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockPasswordResetTokenRepo is a mock implementation for testing
type mockPasswordResetTokenRepo struct {
	tokens map[string]*model.PasswordResetToken
}

func newMockPasswordResetTokenRepo() *mockPasswordResetTokenRepo {
	return &mockPasswordResetTokenRepo{
		tokens: make(map[string]*model.PasswordResetToken),
	}
}

func (m *mockPasswordResetTokenRepo) Create(ctx context.Context, token *model.PasswordResetToken) error {
	if token.ID.IsZero() {
		token.ID = bson.NewObjectID()
	}
	token.CreatedAt = time.Now()
	m.tokens[token.TokenHash] = token
	return nil
}

func (m *mockPasswordResetTokenRepo) GetByTokenHash(ctx context.Context, hash string) (*model.PasswordResetToken, error) {
	token, ok := m.tokens[hash]
	if !ok {
		return nil, nil
	}
	return token, nil
}

func (m *mockPasswordResetTokenRepo) MarkUsed(ctx context.Context, id bson.ObjectID) error {
	return nil
}

// mockUserRepoForReset is a minimal mock for password reset tests
type mockUserRepoForReset struct {
	users map[string]*model.User
}

func newMockUserRepoForReset() *mockUserRepoForReset {
	return &mockUserRepoForReset{
		users: make(map[string]*model.User),
	}
}

func (m *mockUserRepoForReset) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepoForReset) GetByID(ctx context.Context, id bson.ObjectID) (*model.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepoForReset) UpdatePassword(ctx context.Context, id bson.ObjectID, hash string) error {
	return nil
}

func TestHashToken(t *testing.T) {
	token := "abc123def456"
	hash := hashToken(token)
	
	// Should be SHA-256 hex string (64 characters)
	assert.Len(t, hash, 64)
	
	// Same input should produce same hash
	assert.Equal(t, hash, hashToken(token))
	
	// Different input should produce different hash
	assert.NotEqual(t, hash, hashToken("different-token"))
}

func TestPasswordResetHandler_RequestReset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("invalid email format", func(t *testing.T) {
		h := &PasswordResetHandler{}
		
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/", bytes.NewBufferString(`{"email":"invalid"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		
		h.RequestReset(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	
	t.Run("missing email", func(t *testing.T) {
		h := &PasswordResetHandler{}
		
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/", bytes.NewBufferString(`{}`))
		c.Request.Header.Set("Content-Type", "application/json")
		
		h.RequestReset(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestPasswordResetHandler_ConfirmReset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("missing token", func(t *testing.T) {
		h := &PasswordResetHandler{}
		
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/", bytes.NewBufferString(`{"token":"","new_password":"password123"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		
		h.ConfirmReset(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	
	t.Run("token too short", func(t *testing.T) {
		h := &PasswordResetHandler{}
		
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/", bytes.NewBufferString(`{"token":"short","new_password":"password123"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		
		h.ConfirmReset(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	
	t.Run("password too short", func(t *testing.T) {
		h := &PasswordResetHandler{}
		
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/", bytes.NewBufferString(`{"token":"1234567890123456789012345678901234567890123456789012345678901234","new_password":"short"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		
		h.ConfirmReset(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestResponseHelpers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	t.Run("ok response", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		ok(c, gin.H{"message": "success"})
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "ok", resp["message"])
	})
	
	t.Run("fail response", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		
		fail(c, http.StatusBadRequest, "validation error")
		
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "validation error", resp["message"])
	})
}
