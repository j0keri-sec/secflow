package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/secflow/server/internal/model"
	"github.com/secflow/server/internal/repository"
	"github.com/secflow/server/pkg/auth"
)

// PasswordResetHandler handles password reset requests.
// It implements a secure flow where users can request a password reset
// by providing their email, and receive a time-limited reset token.
type PasswordResetHandler struct {
	userRepo       *repository.UserRepo
	resetTokenRepo *repository.PasswordResetTokenRepo
	tokenExpire    time.Duration // Token validity duration (default: 15 minutes)
}

// NewPasswordResetHandler creates a new PasswordResetHandler instance.
//
// Parameters:
//   - userRepo: User repository for user data access
//   - resetTokenRepo: Repository for password reset tokens
//
// Returns:
//   - *PasswordResetHandler: Configured handler instance
func NewPasswordResetHandler(userRepo *repository.UserRepo, resetTokenRepo *repository.PasswordResetTokenRepo) *PasswordResetHandler {
	return &PasswordResetHandler{
		userRepo:       userRepo,
		resetTokenRepo: resetTokenRepo,
		tokenExpire:    15 * time.Minute, // Default 15 minutes validity
	}
}

// RequestResetRequest is the body for POST /auth/reset/request.
// Users provide their email to request a password reset.
type RequestResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// RequestReset initiates a password reset flow for the given email.
// If the email exists, a secure reset token is generated and stored.
// The actual sending of the email is handled externally (SMTP integration pending).
//
// POST /api/v1/auth/reset/request
func (h *PasswordResetHandler) RequestReset(c *gin.Context) {
	var req RequestResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid email format")
		return
	}

	ctx := c.Request.Context()

	// Find user by email (case-insensitive)
	user, err := h.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		// Don't reveal whether email exists for security
		// Return success anyway to prevent email enumeration attacks
		ok(c, gin.H{"message": "if the email exists, a reset link has been sent"})
		return
	}

	// Generate cryptographically secure reset token (32 bytes = 64 hex chars)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		fail(c, http.StatusInternalServerError, "failed to generate reset token")
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// Store hashed token in database
	hashedToken := hashToken(token)
	resetDoc := &model.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: hashedToken,
		ExpiresAt: time.Now().Add(h.tokenExpire),
		Used:      false,
		CreatedAt: time.Now(),
	}

	if err := h.resetTokenRepo.Create(ctx, resetDoc); err != nil {
		fail(c, http.StatusInternalServerError, "failed to create reset token")
		return
	}

	// TODO: Send email with reset link
	// For now, we log the token (in production, email this to the user)
	// In production: sendEmail(user.Email, buildResetLink(token))
	fmt.Printf("[DEBUG] Password reset token for %s: %s\n", user.Email, token)

	// Record audit log (skip for now - would need auditRepo passed in)
	_ = fmt.Sprintf("password_reset_request for user %s", user.Username)

	ok(c, gin.H{
		"message": "if the email exists, a reset link has been sent",
		// DEBUG ONLY - remove in production!
		"debug_token": token,
	})
}

// ConfirmResetRequest is the body for POST /auth/reset/confirm.
// Users provide the reset token and new password to complete the reset.
type ConfirmResetRequest struct {
	Token       string `json:"token" binding:"required,len=64"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ConfirmReset completes the password reset flow using a valid token.
// The token is invalidated after use (one-time only).
//
// POST /api/v1/auth/reset/confirm
func (h *PasswordResetHandler) ConfirmReset(c *gin.Context) {
	var req ConfirmResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request: token must be 64 characters, password minimum 8 characters")
		return
	}

	ctx := c.Request.Context()

	// Find and validate token
	hashedToken := hashToken(req.Token)
	resetDoc, err := h.resetTokenRepo.GetByTokenHash(ctx, hashedToken)
	if err != nil || resetDoc == nil {
		fail(c, http.StatusBadRequest, "invalid reset token")
		return
	}

	// Check if token already used
	if resetDoc.Used {
		fail(c, http.StatusBadRequest, "reset token has already been used")
		return
	}

	// Check if token expired
	if time.Now().After(resetDoc.ExpiresAt) {
		fail(c, http.StatusBadRequest, "reset token has expired")
		return
	}

	// Get the user
	user, err := h.userRepo.GetByID(ctx, resetDoc.UserID)
	if err != nil || user == nil {
		fail(c, http.StatusNotFound, "user not found")
		return
	}

	// Hash new password and update user
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		fail(c, http.StatusInternalServerError, "failed to hash password")
		return
	}

	if err := h.userRepo.UpdatePassword(ctx, user.ID, newHash); err != nil {
		fail(c, http.StatusInternalServerError, "failed to update password")
		return
	}

	// Mark token as used (one-time only)
	if err := h.resetTokenRepo.MarkUsed(ctx, resetDoc.ID); err != nil {
		// Log but don't fail - password already changed
		fmt.Printf("warning: failed to mark token as used: %v\n", err)
	}

	// Record audit log (skip for now)
	_ = fmt.Sprintf("password_reset_complete for user %s", user.Username)

	ok(c, gin.H{"message": "password has been reset successfully"})
}

// hashToken creates a SHA-256 hash of a token for secure storage.
// The raw token is sent to user's email, but we only store the hash.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
