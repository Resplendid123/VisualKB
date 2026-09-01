package user

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, h *UserHandler) {
	users := rg.Group("/users")
	{
		users.POST("", h.Create)
		users.GET("/:user_id", h.Get)
		// /me routes resolve identity only via JWT.
		users.GET("/me/portrait", h.GetPortrait)
		users.PUT("/me/portrait/immutable", h.UpdateImmutable)
		users.PUT("/me/portrait/mutable", h.UpdateMutable)
	}
}
