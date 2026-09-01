package auth

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, h *AuthHandler) {
	auth := rg.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/register", h.Register)
		auth.POST("/refresh", h.Refresh)
		auth.POST("/logout", h.Logout)
	}
}
