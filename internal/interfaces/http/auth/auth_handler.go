package auth

import (
	"net/http"

	app "learn/internal/application/auth"
	"learn/internal/interfaces/http/response"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authSvc *app.AuthService
	cookie  AuthCookie
}

func NewAuthHandler(authSvc *app.AuthService, cookie AuthCookie) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, cookie: cookie}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	u, token, err := h.authSvc.Register(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.setAccessCookie(c, token)
	response.OK(c, authResponse{
		AccessToken: token,
		User:        authUserPayload{ID: u.ID, Name: u.Name, Email: u.Email},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	u, token, err := h.authSvc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.setAccessCookie(c, token)
	response.OK(c, authResponse{
		AccessToken: token,
		User:        authUserPayload{ID: u.ID, Name: u.Name, Email: u.Email},
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	raw, err := c.Cookie("access_token")
	if err != nil || raw == "" {
		var req RefreshRequest
		if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
			response.Fail(c, bindErr)
			return
		}
		raw = req.AccessToken
	}
	u, token, err := h.authSvc.Refresh(c.Request.Context(), raw)
	if err != nil {
		response.Fail(c, err)
		return
	}
	h.setAccessCookie(c, token)
	response.OK(c, authResponse{
		AccessToken: token,
		User:        authUserPayload{ID: u.ID, Name: u.Name, Email: u.Email},
	})
}

// Logout clears the HttpOnly cookie.
func (h *AuthHandler) Logout(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     h.cookie.Name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: http.SameSiteLaxMode,
	})
	response.OK(c, nil)
}

func (h *AuthHandler) setAccessCookie(c *gin.Context, token string) {
	maxAge := max(0, int(h.cookie.TTL.Seconds()))
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     h.cookie.Name,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}
