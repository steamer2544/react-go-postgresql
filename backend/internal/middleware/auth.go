package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"imaxx-backend/internal/service"
	"imaxx-backend/pkg/response"
)

// TokenVerifier can verify a JWT token string and return claims.
type TokenVerifier interface {
	Verify(tokenString string) (service.Claims, error)
}

// Auth returns a gin middleware that verifies a Bearer JWT and sets userID/role.
func Auth(verifier TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Fail(c, service.ErrUnauthorized)
			c.Abort()
			return
		}
		if !strings.HasPrefix(authHeader, "Bearer ") {
			response.Fail(c, service.ErrUnauthorized)
			c.Abort()
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := verifier.Verify(tokenString)
		if err != nil {
			response.Fail(c, service.ErrUnauthorized)
			c.Abort()
			return
		}
		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}
