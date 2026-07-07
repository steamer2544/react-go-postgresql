package middleware

import (
	"github.com/gin-gonic/gin"
	"imaxx-backend/internal/service"
	"imaxx-backend/pkg/response"
)

// RequireRole returns a gin middleware that checks the role set on the context.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		for _, r := range roles {
			if r == role {
				c.Next()
				return
			}
		}
		response.Fail(c, service.ErrForbidden)
		c.Abort()
	}
}
