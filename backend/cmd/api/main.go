// Package main is the entry point for the API server.
package main

import (
	"log"
	"log/slog"

	"imaxx-backend/internal/config"
	"imaxx-backend/internal/model"
	"imaxx-backend/internal/router"
	"imaxx-backend/internal/service"
	"imaxx-backend/pkg/database"
	"imaxx-backend/pkg/logger"

	"gorm.io/gorm"
)

const devPassword = "Passw0rd!"

func main() {
	// Load configuration (fail-fast on missing/invalid env vars).
	cfg := config.Load()

	// Build structured logger.
	appLogger := logger.New(cfg.LogLevel)

	// Connect to database.
	db, err := database.NewPostgresConnection(cfg)
	if err != nil {
		appLogger.Error("failed to connect to database", "error", err)
		log.Fatal(err)
	}

	// Auto-migrate in dev.
	if cfg.AppEnv == "dev" {
		if err := db.AutoMigrate(&model.User{}); err != nil {
			appLogger.Error("failed to auto-migrate", "error", err)
			log.Fatal(err)
		}
		seedDevUsers(db, appLogger)
	}

	// Build router with middleware and routes.
	engine := router.NewRouter(cfg, appLogger, db)

	// Start the server.
	addr := ":" + cfg.AppPort
	appLogger.Info("starting server", "addr", addr)
	if err := engine.Run(addr); err != nil {
		appLogger.Error("failed to start server", "error", err)
		log.Fatal(err)
	}
}

func seedDevUsers(db *gorm.DB, logger *slog.Logger) {
	passwordHash, err := service.HashPassword(devPassword)
	if err != nil {
		logger.Error("failed to hash dev password", "error", err)
		return
	}

	users := []model.User{
		{Email: "admin@example.com", PasswordHash: passwordHash, Role: model.RoleAdmin, FullName: "Admin User", Position: "Administrator"},
		{Email: "creator@example.com", PasswordHash: passwordHash, Role: model.RoleCreator, FullName: "Creator User", Position: "Creator"},
		{Email: "approver@example.com", PasswordHash: passwordHash, Role: model.RoleApprover, FullName: "Approver User", Position: "Approver"},
	}

	for _, u := range users {
		if err := db.Where("email = ?", u.Email).FirstOrCreate(&u, &u).Error; err != nil {
			logger.Error("failed to seed user", "email", u.Email, "error", err)
		}
	}
}
