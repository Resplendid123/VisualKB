package conversation

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, h *ConversationHandler) {
	conversations := rg.Group("/conversations")
	{
		conversations.POST("", h.Create)
		conversations.GET("", h.List)
		conversations.GET("/:conversation_id", h.Get)
		conversations.GET("/:conversation_id/messages", h.GetMessages)
		conversations.POST("/:conversation_id/archive", h.Archive)
		conversations.GET("/:conversation_id/events", h.Events)
	}
}
