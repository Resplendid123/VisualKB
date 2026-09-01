package middleware

import (
	"context"
	"learn/internal/domain/auth"
	"learn/internal/interfaces/http/response"

	"github.com/gin-gonic/gin"
)

const identityKey = "identity"

const accessTokenCookie = "access_token"

// TokenVerifier validates tokens via application.
type TokenVerifier interface {
	VerifyToken(ctx context.Context, token string) (*auth.Identity, error)
}

func IdentityFrom(c *gin.Context) auth.Identity {
	return c.MustGet(identityKey).(auth.Identity)
}

// Auth verifies the access token.
func Auth(verifier TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := extractToken(c)
		if raw == "" {
			response.AbortUnauthorized(c, "missing credentials")
			return
		}
		identity, err := verifier.VerifyToken(c.Request.Context(), raw)
		if err != nil {
			response.AbortUnauthorized(c, "invalid or expired credentials")
			return
		}
		c.Set(identityKey, *identity)
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	if v, ok := bearer(c.GetHeader("Authorization")); ok {
		return v
	}
	if v, err := c.Cookie(accessTokenCookie); err == nil {
		return v
	}
	return ""
}

func bearer(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || header[:len(prefix)] != prefix {
		return "", false
	}
	return header[len(prefix):], true
}
