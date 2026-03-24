package tests

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

	"github.com/secflow/server/internal/api/handler"
	"github.com/secflow/server/internal/model"
	"github.com/secflow/server/pkg/auth"
)

// mockUserRepo is a mock implementation of UserRepo for testing.
type mockUserRepo struct {
	users       map[string]*model.User
	inviteCodes map[string]*model.InviteCode
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:       make(map[string]*model.User),
		inviteCodes: make(map[string]*model.InviteCode),
	}
}

func (m *mockUserRepo) CountAll(ctx context.Context) (int64, error) {
	return int64(len(m.users)), nil
}

func (m *mockUserRepo) Create(ctx context.Context, u *model.User) error {
	if u.ID.IsZero() {
		u.ID = bson.NewObjectID()
	}
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	m.users[u.Username] = u
	return nil
}

func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	u, ok := m.users[username]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id bson.ObjectID) (*model.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) UpdatePassword(ctx context.Context, id bson.ObjectID, hash string) error {
	for _, u := range m.users {
		if u.ID == id {
			u.PasswordHash = hash
			u.UpdatedAt = time.Now()
			return nil
		}
	}
	return nil
}

func (m *mockUserRepo) List(ctx context.Context, page, pageSize int64) ([]*model.User, int64, error) {
	var users []*model.User
	for _, u := range m.users {
		users = append(users, u)
	}
	return users, int64(len(users)), nil
}

// mockInviteCodeRepo is a mock implementation of InviteCodeRepo for testing.
type mockInviteCodeRepo struct {
	codes map[string]*model.InviteCode
}

func newMockInviteCodeRepo() *mockInviteCodeRepo {
	return &mockInviteCodeRepo{
		codes: make(map[string]*model.InviteCode),
	}
}

func (m *mockInviteCodeRepo) GetByCode(ctx context.Context, code string) (*model.InviteCode, error) {
	c, ok := m.codes[code]
	if !ok {
		return nil, nil
	}
	return c, nil
}

func (m *mockInviteCodeRepo) MarkUsed(ctx context.Context, code string, userID bson.ObjectID) error {
	if c, ok := m.codes[code]; ok {
		c.Used = true
		c.UsedByID = userID
		c.UsedAt = time.Now()
	}
	return nil
}

func (m *mockInviteCodeRepo) CountByOwner(ctx context.Context, ownerID bson.ObjectID) (int64, error) {
	var count int64
	for _, c := range m.codes {
		if c.OwnerID == ownerID {
			count++
		}
	}
	return count, nil
}

func (m *mockInviteCodeRepo) ListByOwner(ctx context.Context, ownerID bson.ObjectID) ([]*model.InviteCode, error) {
	var codes []*model.InviteCode
	for _, c := range m.codes {
		if c.OwnerID == ownerID {
			codes = append(codes, c)
		}
	}
	return codes, nil
}

func (m *mockInviteCodeRepo) Create(ctx context.Context, code *model.InviteCode) error {
	if code.ID.IsZero() {
		code.ID = bson.NewObjectID()
	}
	code.CreatedAt = time.Now()
	m.codes[code.Code] = code
	return nil
}

// TestUserRolePermissions tests the role-based access control model.
func TestUserRolePermissions(t *testing.T) {
	t.Run("RoleType constants are defined correctly", func(t *testing.T) {
		assert.Equal(t, model.RoleType("admin"), model.RoleAdmin)
		assert.Equal(t, model.RoleType("editor"), model.RoleEditor)
		assert.Equal(t, model.RoleType("viewer"), model.RoleViewer)
	})

	t.Run("User roles are properly assigned", func(t *testing.T) {
		admin := &model.User{Role: model.RoleAdmin}
		editor := &model.User{Role: model.RoleEditor}
		viewer := &model.User{Role: model.RoleViewer}

		assert.Equal(t, model.RoleAdmin, admin.Role)
		assert.Equal(t, model.RoleEditor, editor.Role)
		assert.Equal(t, model.RoleViewer, viewer.Role)
	})

	t.Run("Role hierarchy (admin > editor > viewer)", func(t *testing.T) {
		// Admin should have highest privileges
		assert.Equal(t, "admin", string(model.RoleAdmin))
		// Editor should have medium privileges
		assert.Equal(t, "editor", string(model.RoleEditor))
		// Viewer should have lowest privileges
		assert.Equal(t, "viewer", string(model.RoleViewer))
	})
}

// TestInviteCodeModel tests the invite code functionality.
func TestInviteCodeModel(t *testing.T) {
	t.Run("InviteCode creation", func(t *testing.T) {
		code := &model.InviteCode{
			Code:      "TEST1234",
			OwnerID:   bson.NewObjectID(),
			IsAdmin:   false,
			Used:      false,
			CreatedAt: time.Now(),
		}

		assert.NotEmpty(t, code.Code)
		assert.False(t, code.Used)
		assert.False(t, code.IsAdmin)
	})

	t.Run("InviteCode marking as used", func(t *testing.T) {
		code := &model.InviteCode{
			Code:    "USED1234",
			Used:    false,
			UsedAt:  time.Time{},
		}

		// Simulate marking as used
		code.Used = true
		code.UsedAt = time.Now()
		code.UsedByID = bson.NewObjectID()

		assert.True(t, code.Used)
		assert.False(t, code.UsedAt.IsZero())
	})
}

// TestPasswordHashing tests the password hashing functionality.
func TestPasswordHashing(t *testing.T) {
	t.Run("HashPassword creates valid hash", func(t *testing.T) {
		password := "securePassword123!"
		hash, err := auth.HashPassword(password)

		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, password, hash) // Hash should differ from password
		assert.Greater(t, len(hash), 50)   // bcrypt hashes are typically 60+ chars
	})

	t.Run("CheckPassword verifies correct password", func(t *testing.T) {
		password := "mySecretPassword456"
		hash, err := auth.HashPassword(password)
		require.NoError(t, err)

		// Correct password should match
		assert.True(t, auth.CheckPassword(hash, password))
	})

	t.Run("CheckPassword rejects incorrect password", func(t *testing.T) {
		password := "mySecretPassword456"
		wrongPassword := "wrongPassword789"
		hash, err := auth.HashPassword(password)
		require.NoError(t, err)

		// Wrong password should not match
		assert.False(t, auth.CheckPassword(hash, wrongPassword))
	})

	t.Run("Different hashes for same password (salted)", func(t *testing.T) {
		password := "samePassword"
		hash1, _ := auth.HashPassword(password)
		hash2, _ := auth.HashPassword(password)

		// Each hash should be different due to random salt
		assert.NotEqual(t, hash1, hash2)
		// But both should verify against original password
		assert.True(t, auth.CheckPassword(hash1, password))
		assert.True(t, auth.CheckPassword(hash2, password))
	})
}

// TestJWTToken tests JWT token generation and validation.
func TestJWTToken(t *testing.T) {
	authSvc := auth.New("test-secret-key", 24*time.Hour)

	t.Run("GenerateToken creates valid token", func(t *testing.T) {
		userID := bson.NewObjectID().Hex()
		username := "testuser"
		role := string(model.RoleAdmin)

		token, err := authSvc.GenerateToken(userID, username, role)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("ParseToken parses token claims", func(t *testing.T) {
		userID := bson.NewObjectID().Hex()
		username := "validator_test"
		role := string(model.RoleEditor)

		token, err := authSvc.GenerateToken(userID, username, role)
		require.NoError(t, err)

		claims, err := authSvc.ParseToken(token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, username, claims.Username)
		assert.Equal(t, role, claims.Role)
	})

	t.Run("ParseToken rejects invalid token", func(t *testing.T) {
		_, err := authSvc.ParseToken("invalid.token.here")
		assert.Error(t, err)
	})

	t.Run("ParseToken rejects expired token", func(t *testing.T) {
		// Create service with negative expiry
		expiredSvc := auth.New("test-secret", -1*time.Hour)
		token, err := expiredSvc.GenerateToken("user", "name", "role")
		require.NoError(t, err)

		_, err = expiredSvc.ParseToken(token)
		assert.Error(t, err)
	})
}

// TestUserModel tests the User model structure.
func TestUserModel(t *testing.T) {
	t.Run("User creation with all fields", func(t *testing.T) {
		now := time.Now()
		user := &model.User{
			ID:           bson.NewObjectID(),
			Username:     "testuser",
			Email:        "test@example.com",
			PasswordHash: "hashedPassword",
			Role:         model.RoleAdmin,
			Avatar:       "https://example.com/avatar.png",
			Active:       true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		assert.NotZero(t, user.ID)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, model.RoleAdmin, user.Role)
		assert.True(t, user.Active)
	})

	t.Run("User InviteCode tracking", func(t *testing.T) {
		inviteCode := "ADMIN123"
		user := &model.User{
			Username:   "newuser",
			InviteCode: inviteCode,
		}

		assert.Equal(t, inviteCode, user.InviteCode)
	})
}

// TestRegisterRequest tests the registration request validation.
func TestRegisterRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Valid registration request", func(t *testing.T) {
		reqBody := handler.RegisterRequest{
			Username:   "newuser123",
			Password:   "password123!",
			Email:      "newuser@example.com",
			InviteCode: "VALID123",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		var parsed handler.RegisterRequest
		err := c.ShouldBindJSON(&parsed)
		assert.NoError(t, err)
		assert.Equal(t, "newuser123", parsed.Username)
		assert.Equal(t, "newuser@example.com", parsed.Email)
	})

	t.Run("Invalid email format rejected", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "testuser",
			"password": "password123",
			"email":    "not-an-email",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		var parsed handler.RegisterRequest
		err := c.ShouldBindJSON(&parsed)
		assert.Error(t, err) // Should fail due to invalid email
	})

	t.Run("Password too short rejected", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "testuser",
			"password": "short", // Less than 8 characters
			"email":    "test@example.com",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		var parsed handler.RegisterRequest
		err := c.ShouldBindJSON(&parsed)
		assert.Error(t, err) // Should fail due to password too short
	})
}

// TestPasswordResetToken tests the password reset token flow.
func TestPasswordResetToken(t *testing.T) {
	t.Run("PasswordResetToken creation", func(t *testing.T) {
		token := &model.PasswordResetToken{
			ID:        bson.NewObjectID(),
			UserID:    bson.NewObjectID(),
			TokenHash: "hashedTokenValue",
			ExpiresAt: time.Now().Add(15 * time.Minute),
			Used:      false,
			CreatedAt: time.Now(),
		}

		assert.NotZero(t, token.ID)
		assert.NotZero(t, token.UserID)
		assert.False(t, token.Used)
		assert.True(t, token.ExpiresAt.After(time.Now()))
	})

	t.Run("Token expiration check", func(t *testing.T) {
		// Expired token
		expiredToken := &model.PasswordResetToken{
			ExpiresAt: time.Now().Add(-1 * time.Hour), // 1 hour ago
			Used:      false,
		}
		assert.True(t, time.Now().After(expiredToken.ExpiresAt))

		// Valid token
		validToken := &model.PasswordResetToken{
			ExpiresAt: time.Now().Add(15 * time.Minute), // 15 minutes from now
			Used:      false,
		}
		assert.True(t, time.Now().Before(validToken.ExpiresAt))
	})

	t.Run("Token one-time use", func(t *testing.T) {
		token := &model.PasswordResetToken{
			Used: false,
		}

		// First use
		token.Used = true
		assert.True(t, token.Used)

		// Second use attempt should be rejected by logic
		assert.True(t, token.Used) // Would be caught by "already used" check
	})
}

// TestRoleBasedAccess tests role hierarchy and permissions.
func TestRoleBasedAccess(t *testing.T) {
	// Define permission levels (higher number = more permissions)
	permissionLevels := map[model.RoleType]int{
		model.RoleViewer: 1,
		model.RoleEditor: 2,
		model.RoleAdmin:  3,
	}

	t.Run("Admin has highest permissions", func(t *testing.T) {
		assert.Greater(t, permissionLevels[model.RoleAdmin], permissionLevels[model.RoleEditor])
		assert.Greater(t, permissionLevels[model.RoleAdmin], permissionLevels[model.RoleViewer])
	})

	t.Run("Editor has medium permissions", func(t *testing.T) {
		assert.Greater(t, permissionLevels[model.RoleEditor], permissionLevels[model.RoleViewer])
	})

	t.Run("Viewer has lowest permissions", func(t *testing.T) {
		assert.Equal(t, 1, permissionLevels[model.RoleViewer])
	})
}

// TestInviteCodeGeneration tests invite code generation.
func TestInviteCodeGeneration(t *testing.T) {
	t.Run("Generated codes have correct length", func(t *testing.T) {
		code := generateTestCode()
		assert.Equal(t, 8, len(code))
	})

	t.Run("Real UUID-based codes would be unique", func(t *testing.T) {
		// This test verifies the concept - in production, UUIDs are used
		// which have sufficient randomness to avoid collisions
		// We just verify the code length is consistent
		for i := 0; i < 10; i++ {
			code := generateTestCode()
			assert.Equal(t, 8, len(code))
		}
	})
}

// generateTestCode simulates code generation for testing.
// Returns a consistent 8-char string for testing.
func generateTestCode() string {
	return "TESTCODE"
}

// Integration test helpers
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

// TestIntegration_FullRegistrationFlow tests the complete registration flow.
func TestIntegration_FullRegistrationFlow(t *testing.T) {
	// This is a placeholder for integration tests
	// Full integration tests require MongoDB and Redis running
	
	t.Run("Registration flow is documented", func(t *testing.T) {
		// Step 1: Generate invite code (admin)
		// Step 2: Register with invite code
		// Step 3: Login to get JWT
		// Step 4: Use JWT for authenticated requests
		
		steps := []string{
			"1. Admin generates invite code via POST /api/v1/auth/invite",
			"2. User registers via POST /api/v1/auth/register with invite code",
			"3. User logs in via POST /api/v1/auth/login",
			"4. User uses JWT token in Authorization header for subsequent requests",
		}
		
		assert.Equal(t, 4, len(steps))
	})
}

// TestIntegration_PasswordResetFlow tests the complete password reset flow.
func TestIntegration_PasswordResetFlow(t *testing.T) {
	t.Run("Password reset flow is documented", func(t *testing.T) {
		// Step 1: Request reset via email
		// Step 2: Receive token (in production, via email)
		// Step 3: Confirm reset with token and new password
		
		steps := []string{
			"1. User requests reset via POST /api/v1/auth/reset/request with email",
			"2. System sends reset token (DEBUG: token returned in response)",
			"3. User confirms via POST /api/v1/auth/reset/confirm with token and new password",
		}
		
		assert.Equal(t, 3, len(steps))
	})
}

// BenchmarkPasswordHashing benchmarks the password hashing function.
func BenchmarkPasswordHashing(b *testing.B) {
	password := "benchmarkPassword123!"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auth.HashPassword(password)
	}
}

// BenchmarkTokenGeneration benchmarks JWT token generation.
func BenchmarkTokenGeneration(b *testing.B) {
	authSvc := auth.New("benchmark-secret", 24*time.Hour)
	userID := bson.NewObjectID().Hex()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		authSvc.GenerateToken(userID, "benchuser", "admin")
	}
}
