package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/secflow/server/internal/api/middleware"
	"github.com/secflow/server/internal/model"
	"github.com/secflow/server/internal/repository"
	"github.com/secflow/server/pkg/auth"

	"github.com/google/uuid"
)

// AuthHandler handles all authentication-related endpoints.
type AuthHandler struct {
	userRepo    *repository.UserRepo
	inviteRepo  *repository.InviteCodeRepo
	auditRepo   *repository.AuditLogRepository
	authSvc     *auth.Service
}

func NewAuthHandler(ur *repository.UserRepo, ir *repository.InviteCodeRepo, ar *repository.AuditLogRepository, as *auth.Service) *AuthHandler {
	return &AuthHandler{userRepo: ur, inviteRepo: ir, auditRepo: ar, authSvc: as}
}

// recordAudit logs an action to the audit trail.
func (h *AuthHandler) recordAudit(c *gin.Context, action, resource string) {
	if h.auditRepo == nil {
		return
	}
	log := &model.AuditLog{
		Username:  middleware.GetUsername(c),
		Action:    action,
		Resource:  resource,
		IP:        c.ClientIP(),
	}
	_ = h.auditRepo.Insert(c.Request.Context(), log)
}

// RegisterRequest is the body for POST /auth/register.
type RegisterRequest struct {
	Username   string `json:"username"    binding:"required,min=3,max=32"`
	Password   string `json:"password"    binding:"required,min=8"`
	Email      string `json:"email"       binding:"required,email"`
	InviteCode string `json:"invite_code" binding:"omitempty"` // optional only on first-ever registration
}

// Register creates a new user account using an invite code.
// Special case: if no users exist yet, the first registration creates an admin
// without requiring an invite code.
//
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	// Bootstrap check: is this the very first user?
	count, err := h.userRepo.CountAll(c)
	if err != nil {
		fail(c, http.StatusInternalServerError, "internal error")
		return
	}
	isBootstrap := count == 0

	var targetRole model.RoleType = model.RoleViewer
	var inviteCode *model.InviteCode

	if isBootstrap {
		// First user becomes admin — no invite code required.
		targetRole = model.RoleAdmin
	} else {
		// Validate invite code.
		if req.InviteCode == "" {
			fail(c, http.StatusBadRequest, "invite_code is required")
			return
		}
		var codeErr error
		inviteCode, codeErr = h.inviteRepo.GetByCode(c, req.InviteCode)
		if codeErr != nil || inviteCode == nil {
			fail(c, http.StatusBadRequest, "invalid invite code")
			return
		}
		if inviteCode.Used {
			fail(c, http.StatusBadRequest, "invite code has already been used")
			return
		}
		if !inviteCode.ExpiresAt.IsZero() && time.Now().After(inviteCode.ExpiresAt) {
			fail(c, http.StatusBadRequest, "invite code has expired")
			return
		}
		if inviteCode.IsAdmin {
			targetRole = model.RoleAdmin
		}
	}

	// Check username uniqueness.
	existing, err := h.userRepo.GetByUsername(c, req.Username)
	if err != nil {
		fail(c, http.StatusInternalServerError, "internal error")
		return
	}
	if existing != nil {
		fail(c, http.StatusConflict, "username already taken")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		fail(c, http.StatusInternalServerError, "internal error")
		return
	}

	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         targetRole,
		InviteCode:   req.InviteCode,
		Active:       true,
	}
	if err = h.userRepo.Create(c, user); err != nil {
		fail(c, http.StatusInternalServerError, "failed to create user")
		return
	}

	// Mark invite code as used (skip for bootstrap registration).
	if !isBootstrap && inviteCode != nil {
		_ = h.inviteRepo.MarkUsed(c, req.InviteCode, user.ID)
	}

	// Record audit log
	h.recordAudit(c, "register", fmt.Sprintf("user:%s", user.Username))

	token, _ := h.authSvc.GenerateToken(user.ID.Hex(), user.Username, string(user.Role))
	ok(c, gin.H{"token": token, "user": sanitiseUser(user)})
}

// LoginRequest is the body for POST /auth/login.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login authenticates a user and returns a JWT.
//
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.userRepo.GetByUsername(c, req.Username)
	if err != nil || user == nil || !auth.CheckPassword(user.PasswordHash, req.Password) {
		fail(c, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if !user.Active {
		fail(c, http.StatusForbidden, "account disabled")
		return
	}

	token, err := h.authSvc.GenerateToken(user.ID.Hex(), user.Username, string(user.Role))
	if err != nil {
		fail(c, http.StatusInternalServerError, "token generation failed")
		return
	}

	// Record audit log for successful login
	h.recordAudit(c, "login", "system")

	ok(c, gin.H{"token": token, "user": sanitiseUser(user)})
}

// Me returns the authenticated user's profile.
//
// GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	oid, err := objectIDFromHex(userID)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid user id")
		return
	}
	user, err := h.userRepo.GetByID(c, oid)
	if err != nil {
		fail(c, http.StatusNotFound, "user not found")
		return
	}
	ok(c, sanitiseUser(user))
}

// ChangePasswordRequest is the body for PUT /auth/password.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword updates the authenticated user's password.
//
// PUT /api/v1/auth/password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	userID := middleware.GetUserID(c)
	oid, _ := objectIDFromHex(userID)
	user, err := h.userRepo.GetByID(c, oid)
	if err != nil {
		fail(c, http.StatusNotFound, "user not found")
		return
	}
	if !auth.CheckPassword(user.PasswordHash, req.OldPassword) {
		fail(c, http.StatusUnauthorized, "incorrect current password")
		return
	}
	hash, _ := auth.HashPassword(req.NewPassword)
	if err = h.userRepo.UpdatePassword(c, oid, hash); err != nil {
		fail(c, http.StatusInternalServerError, "failed to update password")
		return
	}
	ok(c, nil)
}

// GenerateInviteCode creates a new invite code for the caller.
// Admins have no limit; regular users are capped at 5.
//
// POST /api/v1/auth/invite
func (h *AuthHandler) GenerateInviteCode(c *gin.Context) {
	userID := middleware.GetUserID(c)
	role := middleware.GetRole(c)
	oid, _ := objectIDFromHex(userID)

	const userLimit = 5
	if role != string(model.RoleAdmin) {
		count, err := h.inviteRepo.CountByOwner(c, oid)
		if err != nil {
			fail(c, http.StatusInternalServerError, "failed to check invite code count")
			return
		}
		if count >= userLimit {
			fail(c, http.StatusForbidden, fmt.Sprintf("invite code limit reached (%d)", userLimit))
			return
		}
	}

	code := &model.InviteCode{
		Code:      generateCode(),
		OwnerID:   oid,
		IsAdmin:   role == string(model.RoleAdmin),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days expiration
	}
	if err := h.inviteRepo.Create(c, code); err != nil {
		fail(c, http.StatusInternalServerError, "failed to create invite code")
		return
	}
	ok(c, code)
}

// ListInviteCodes returns all invite codes owned by the caller.
//
// GET /api/v1/auth/invite
func (h *AuthHandler) ListInviteCodes(c *gin.Context) {
	userID := middleware.GetUserID(c)
	oid, err := objectIDFromHex(userID)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid user id")
		return
	}
	codes, err := h.inviteRepo.ListByOwner(c, oid)
	if err != nil {
		fail(c, http.StatusInternalServerError, "database error")
		return
	}
	ok(c, codes)
}

// sanitiseUser strips sensitive fields before returning a user object.
func sanitiseUser(u *model.User) gin.H {
	return gin.H{
		"id":         u.ID.Hex(),
		"username":   u.Username,
		"email":      u.Email,
		"role":       u.Role,
		"avatar":     u.Avatar,
		"active":     u.Active,
		"created_at": u.CreatedAt,
	}
}

// generateCode creates a random 8-character uppercase invite code.
func generateCode() string {
	return uuid.New().String()[:8]
}
