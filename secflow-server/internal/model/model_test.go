package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestRoleTypes(t *testing.T) {
	assert.Equal(t, RoleType("admin"), RoleAdmin)
	assert.Equal(t, RoleType("editor"), RoleEditor)
	assert.Equal(t, RoleType("viewer"), RoleViewer)
}

func TestUserStruct(t *testing.T) {
	user := &User{
		ID:           bson.NewObjectID(),
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         RoleAdmin,
		Active:       true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, RoleAdmin, user.Role)
	assert.True(t, user.Active)
}

func TestInviteCodeStruct(t *testing.T) {
	code := &InviteCode{
		ID:        bson.NewObjectID(),
		Code:      "TESTCODE123",
		OwnerID:   bson.NewObjectID(),
		Used:      false,
		IsAdmin:   false,
		CreatedAt: time.Now(),
	}

	assert.Equal(t, "TESTCODE123", code.Code)
	assert.False(t, code.Used)
	assert.False(t, code.IsAdmin)
}

func TestPasswordResetTokenStruct(t *testing.T) {
	token := &PasswordResetToken{
		ID:        bson.NewObjectID(),
		UserID:    bson.NewObjectID(),
		TokenHash: "somehash",
		ExpiresAt: time.Now().Add(15 * time.Minute),
		Used:      false,
		CreatedAt: time.Now(),
	}

	assert.Equal(t, "somehash", token.TokenHash)
	assert.False(t, token.Used)
	assert.True(t, token.ExpiresAt.After(time.Now()))
}

func TestNodeStatus(t *testing.T) {
	assert.Equal(t, NodeStatus("online"), NodeOnline)
	assert.Equal(t, NodeStatus("offline"), NodeOffline)
	assert.Equal(t, NodeStatus("busy"), NodeBusy)
}
