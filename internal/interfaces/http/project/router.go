package project

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, h *ProjectHandler) {
	projects := rg.Group("/projects")
	{
		projects.POST("", h.Create)
		projects.GET("", h.List)
		projects.GET("/:project_id", h.Get)
		projects.PATCH("/:project_id", h.Rename)
		projects.POST("/:project_id/archive", h.Archive)
	}

	rg.GET("/conversations/:conversation_id/active-project", h.GetActive)
	rg.PUT("/conversations/:conversation_id/active-project", h.SetActive)
}
