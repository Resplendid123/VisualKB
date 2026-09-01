package http

import (
	"learn/internal/interfaces/http/auth"
	"learn/internal/interfaces/http/conversation"
	"learn/internal/interfaces/http/document"
	"learn/internal/interfaces/http/middleware"
	"learn/internal/interfaces/http/project"
	"learn/internal/interfaces/http/user"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires routes and auth.
func RegisterRoutes(api *gin.RouterGroup, handlers *Handlers, verifier middleware.TokenVerifier) {
	auth.RegisterRoutes(api, handlers.Auth)

	protected := api.Group("")
	protected.Use(middleware.Auth(verifier))
	{
		user.RegisterRoutes(protected, handlers.User)
		conversation.RegisterRoutes(protected, handlers.Conversation)
		project.RegisterRoutes(protected, handlers.Project)
		document.RegisterRoutes(protected, handlers.Document, handlers.Tree)
	}
}
