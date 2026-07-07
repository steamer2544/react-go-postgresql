package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"imaxx-backend/pkg/response"
	"log/slog"
)

// Recovery returns a gin middleware that recovers from panics, logs them, and returns
// a standardized 500 error response.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID := uuid.New().String()
				c.Set("request_id", requestID)

				logger.Error("panic recovered",
					slog.String("request_id", requestID),
					slog.Any("error", err),
				)

				response.Error(c, 500, "INTERNAL_ERROR", "internal server error", nil)
				c.Abort()
			}
		}()
		c.Next()
	}
}
