package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/secflow/server/internal/model"
	"github.com/secflow/server/internal/repository"
	"github.com/secflow/server/pkg/auth"
	"github.com/secflow/server/pkg/notify"
)

// minResponseTime is the minimum time to wait before responding
// to prevent timing attacks on email enumeration.
const minResponseTime = 100 * time.Millisecond

// PasswordResetHandler handles password reset requests.
// It implements a secure flow where users can request a password reset
// by providing their email, and receive a time-limited reset token.
type PasswordResetHandler struct {
	userRepo       *repository.UserRepo
	resetTokenRepo *repository.PasswordResetTokenRepo
	tokenExpire    time.Duration // Token validity duration (default: 15 minutes)
	emailSender    *notify.EmailSender
	baseURL        string // Base URL for password reset link (e.g., https://secflow.example.com)
}

// NewPasswordResetHandler creates a new PasswordResetHandler instance.
//
// Parameters:
//   - userRepo: User repository for user data access
//   - resetTokenRepo: Repository for password reset tokens
//   - emailSender: Email sender for sending reset tokens (can be nil in dev mode)
//   - baseURL: Base URL for password reset link
//
// Returns:
//   - *PasswordResetHandler: Configured handler instance
func NewPasswordResetHandler(userRepo *repository.UserRepo, resetTokenRepo *repository.PasswordResetTokenRepo, emailSender *notify.EmailSender, baseURL string) *PasswordResetHandler {
	return &PasswordResetHandler{
		userRepo:       userRepo,
		resetTokenRepo: resetTokenRepo,
		tokenExpire:    15 * time.Minute, // Default 15 minutes validity
		emailSender:    emailSender,
		baseURL:        baseURL,
	}
}

// RequestResetRequest is the body for POST /auth/reset/request.
// Users provide their email to request a password reset.
type RequestResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// RequestReset initiates a password reset flow for the given email.
// If the email exists, a secure reset token is generated and stored.
// The actual sending of the email is handled externally via the configured email provider.
//
// POST /api/v1/auth/reset/request
//
// Security features:
//   - Timing-safe response (always returns success to prevent email enumeration)
//   - Cryptographically secure token generation (32 bytes)
//   - Token hashed before storage (SHA-256)
//   - Time-limited token (15 minutes default)
func (h *PasswordResetHandler) RequestReset(c *gin.Context) {
	var req RequestResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid email format")
		return
	}

	ctx := c.Request.Context()
	requestStart := time.Now()

	// Find user by email (case-insensitive)
	user, err := h.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		// Prevent timing attacks: always take minimum time regardless of user existence
		elapsed := time.Since(requestStart)
		if elapsed < minResponseTime {
			time.Sleep(minResponseTime - elapsed)
		}
		// Don't reveal whether email exists for security
		// Return success anyway to prevent email enumeration attacks
		log.Info().Str("email", req.Email).Msg("password reset requested for non-existent user")
		ok(c, gin.H{"message": "if the email exists, a reset link has been sent"})
		return
	}

	// Invalidate all existing reset tokens for this user to prevent token accumulation
	if err := h.resetTokenRepo.InvalidateAllForUser(ctx, user.ID); err != nil {
		log.Warn().Err(err).Str("user_id", user.ID.Hex()).Msg("failed to invalidate old reset tokens")
		// Continue anyway - this is not a fatal error
	}

	// Generate cryptographically secure reset token (32 bytes = 64 hex chars)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		log.Error().Err(err).Msg("failed to generate password reset token")
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
		log.Error().Err(err).Str("user_id", user.ID.Hex()).Msg("failed to store password reset token")
		fail(c, http.StatusInternalServerError, "failed to create reset token")
		return
	}

	// Send password reset email if email sender is configured
	if h.emailSender != nil {
		resetURL := fmt.Sprintf("%s/reset-password?token=%s", h.baseURL, token)
		if err := h.emailSender.SendPasswordReset(ctx, user.Email, user.Username, token, resetURL); err != nil {
			log.Error().Err(err).
				Str("user_id", user.ID.Hex()).
				Str("email", user.Email).
				Msg("failed to send password reset email")
			// Don't fail the request - token is still valid
			// User can contact support if they don't receive the email
		} else {
			log.Info().
				Str("user_id", user.ID.Hex()).
				Str("email", user.Email).
				Msg("password reset email sent successfully")
		}
	} else {
		// Development mode: log the token (masked) for testing
		log.Info().
			Str("user_id", user.ID.Hex()).
			Str("username", user.Username).
			Str("token_prefix", token[:8]+"...").
			Msg("password reset token generated (email not configured)")
	}

	// Ensure minimum response time to prevent timing attacks
	elapsed := time.Since(requestStart)
	if elapsed < minResponseTime {
		time.Sleep(minResponseTime - elapsed)
	}

	ok(c, gin.H{"message": "if the email exists, a reset link has been sent"})
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
//
// Security features:
//   - Token hashed before lookup (constant-time comparison)
//   - One-time use (token marked as used after successful reset)
//   - Time-limited (tokens expire after 15 minutes)
//   - Password strength validation (minimum 8 characters)
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
		log.Warn().Str("token", req.Token[:8]+"...").Msg("invalid password reset token used")
		fail(c, http.StatusBadRequest, "invalid reset token")
		return
	}

	// Check if token already used
	if resetDoc.Used {
		log.Warn().Str("token_id", resetDoc.ID.Hex()).Msg("attempt to reuse expired password reset token")
		fail(c, http.StatusBadRequest, "reset token has already been used")
		return
	}

	// Check if token expired
	if time.Now().After(resetDoc.ExpiresAt) {
		log.Warn().Str("token_id", resetDoc.ID.Hex()).Msg("attempt to use expired password reset token")
		fail(c, http.StatusBadRequest, "reset token has expired")
		return
	}

	// Get the user
	user, err := h.userRepo.GetByID(ctx, resetDoc.UserID)
	if err != nil || user == nil {
		log.Error().Str("token_id", resetDoc.ID.Hex()).Msg("user not found for password reset token")
		fail(c, http.StatusNotFound, "user not found")
		return
	}

	// Hash new password and update user
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		log.Error().Err(err).Str("user_id", user.ID.Hex()).Msg("failed to hash new password")
		fail(c, http.StatusInternalServerError, "failed to hash password")
		return
	}

	if err := h.userRepo.UpdatePassword(ctx, user.ID, newHash); err != nil {
		log.Error().Err(err).Str("user_id", user.ID.Hex()).Msg("failed to update password")
		fail(c, http.StatusInternalServerError, "failed to update password")
		return
	}

	// Mark token as used (one-time only)
	if err := h.resetTokenRepo.MarkUsed(ctx, resetDoc.ID); err != nil {
		// Log but don't fail - password already changed
		log.Warn().Err(err).Str("token_id", resetDoc.ID.Hex()).Msg("failed to mark token as used after password reset")
	}

	// Log successful password reset
	log.Info().
		Str("user_id", user.ID.Hex()).
		Str("username", user.Username).
		Msg("password reset completed successfully")

	ok(c, gin.H{"message": "password has been reset successfully"})
}

// hashToken creates a SHA-256 hash of a token for secure storage.
// The raw token is sent to user's email, but we only store the hash.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
