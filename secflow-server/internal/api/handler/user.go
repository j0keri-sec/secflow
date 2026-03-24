package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/secflow/server/internal/api/middleware"
	"github.com/secflow/server/internal/model"
	"github.com/secflow/server/internal/repository"
)

// UserHandler manages admin user operations.
type UserHandler struct {
	userRepo *repository.UserRepo
}

func NewUserHandler(ur *repository.UserRepo) *UserHandler {
	return &UserHandler{userRepo: ur}
}

// List returns the full user list (admin only).
//
// GET /api/v1/users
func (h *UserHandler) List(c *gin.Context) {
	// Return all users for admin panel (no pagination needed at small scale).
	users, _, err := h.userRepo.List(c, 1, 500)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	sanitised := make([]gin.H, 0, len(users))
	for _, u := range users {
		sanitised = append(sanitised, sanitiseUser(u))
	}
	ok(c, sanitised)
}

// UpdatePatch is the body for PATCH /users/:id (role and/or active).
type UpdatePatch struct {
	Role   model.RoleType `json:"role"`
	Active *bool          `json:"active"`
}

// UpdateRole handles PATCH /api/v1/users/:id
// It accepts any combination of role and active fields.
func (h *UserHandler) UpdateRole(c *gin.Context) {
	var patch UpdatePatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	id, err := objectIDFromHex(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid user id")
		return
	}
	if middleware.GetUserID(c) == c.Param("id") {
		fail(c, http.StatusBadRequest, "cannot modify your own account here")
		return
	}

	set := bson.M{"updated_at": time.Now().UTC()}
	if patch.Role != "" {
		set["role"] = patch.Role
	}
	if patch.Active != nil {
		set["active"] = *patch.Active
	}

	_, err = h.userRepo.DB().Collection(model.CollUsers).UpdateOne(c,
		bson.M{"_id": id},
		bson.M{"$set": set},
	)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, nil)
}

// Delete removes a user account (admin only).
//
// DELETE /api/v1/users/:id
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := objectIDFromHex(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid user id")
		return
	}
	if middleware.GetUserID(c) == c.Param("id") {
		fail(c, http.StatusBadRequest, "cannot delete yourself")
		return
	}
	if err = h.userRepo.Delete(c, id); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, nil)
}
