// Package router sets up the gin engine with middleware and routes.
package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"imaxx-backend/internal/config"
	"imaxx-backend/internal/handler"
	"imaxx-backend/internal/middleware"
	"imaxx-backend/internal/repository"
	"imaxx-backend/internal/service"
	"imaxx-backend/pkg/response"
)

// NewRouter creates a gin.Engine with middleware and registers all routes.
func NewRouter(cfg *config.Config, logger *slog.Logger, db *gorm.DB) *gin.Engine {
	engine := gin.New()

	engine.Use(middleware.Recovery(logger))
	engine.Use(middleware.RequestLogger(logger))
	engine.Use(middleware.CORSMiddleware(cfg.CORSAllowedOrigins))

	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Wire repositories and services
	userRepo := repository.NewUserRepository(db)
	tokenSvc := service.NewTokenService(cfg.JWTSecret, cfg.JWTAccessTTL)
	authSvc := service.NewAuthService(userRepo, tokenSvc)
	profileSvc := service.NewProfileService(userRepo, cfg.SignatureUploadDir, cfg.SignatureMaxBytes, cfg.SignatureAllowedTypes)

	authHandler := handler.NewAuthHandler(authSvc)
	meHandler := handler.NewMeHandler(profileSvc)

	// Public routes
	engine.POST("/auth/login", authHandler.Login)

	// Protected routes
	me := engine.Group("/me", middleware.Auth(tokenSvc))
	{
		me.GET("", meHandler.GetMe)
		me.PUT("/profile", meHandler.UpdateProfile)
		me.POST("/signature", meHandler.UploadSignature)
		me.GET("/signature", meHandler.GetSignature)
	}

	// RBAC example
	admin := engine.Group("/admin", middleware.Auth(tokenSvc), middleware.RequireRole("admin"))
	{
		admin.GET("/ping", func(c *gin.Context) {
			response.Success(c, 200, nil, "pong")
		})
	}

	return engine
}
