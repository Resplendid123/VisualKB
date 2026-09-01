package user

import (
	"strconv"

	app "learn/internal/application/user"
	"learn/internal/interfaces/http/middleware"
	"learn/internal/interfaces/http/response"

	"github.com/gin-gonic/gin"
)

// UserHandler parses HTTP and calls services.
type UserHandler struct {
	userSvc *app.UserService
}

func NewUserHandler(userSvc *app.UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

func (h *UserHandler) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	u, err := h.userSvc.Create(c.Request.Context(), req.Name, req.Email)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toUserResponse(u))
}

func (h *UserHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || id <= 0 {
		response.AbortBadRequest(c, "invalid user id")
		return
	}
	u, err := h.userSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toUserResponse(u))
}

func (h *UserHandler) GetPortrait(c *gin.Context) {
	userID := middleware.IdentityFrom(c).UserID
	u, err := h.userSvc.GetByID(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toPortraitResponse(u))
}

// UpdateImmutable writes the user's immutable portrait.
func (h *UserHandler) UpdateImmutable(c *gin.Context) {
	var req UpdateImmutableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	if err := h.userSvc.UpdateImmutable(c.Request.Context(), userID, req.Immutable); err != nil {
		response.Fail(c, err)
		return
	}
	h.loadPortrait(c, userID)
}

func (h *UserHandler) UpdateMutable(c *gin.Context) {
	var req UpdateMutableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	userID := middleware.IdentityFrom(c).UserID
	if err := h.userSvc.UpdateMutable(c.Request.Context(), userID, req.Mutable); err != nil {
		response.Fail(c, err)
		return
	}
	h.loadPortrait(c, userID)
}

// loadPortrait reads back the saved portrait.
func (h *UserHandler) loadPortrait(c *gin.Context, userID int64) {
	u, err := h.userSvc.GetByID(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toPortraitResponse(u))
}
