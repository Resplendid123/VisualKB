package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var allowedOrigins = []string{
	"http://localhost:5173",
	"http://127.0.0.1:5173",
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && isAllowed(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, X-User-Id, Last-Event-ID, Last-Seq")
			c.Header("Access-Control-Expose-Headers", "Last-Event-ID, Last-Seq")
			c.Header("Access-Control-Max-Age", "86400")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func isAllowed(origin string) bool {
	for _, o := range allowedOrigins {
		if strings.EqualFold(o, origin) {
			return true
		}
	}
	return false
}
