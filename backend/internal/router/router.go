// Package router sets up the gin engine with middleware and routes.
package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"imaxx-backend/internal/config"
	"imaxx-backend/internal/middleware"
)

// NewRouter creates a gin.Engine with recovery, request logger, and CORS middleware,
// then registers the health-check endpoint.
func NewRouter(cfg *config.Config, logger *slog.Logger) *gin.Engine {
	engine := gin.New()

	engine.Use(middleware.Recovery(logger))
	engine.Use(middleware.RequestLogger(logger))
	engine.Use(middleware.CORSMiddleware(cfg.CORSAllowedOrigins))

	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return engine
}
