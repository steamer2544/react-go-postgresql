// Package main is the entry point for the API server.
package main

import (
	"log"

	"imaxx-backend/internal/config"
	"imaxx-backend/internal/router"
	"imaxx-backend/pkg/database"
	"imaxx-backend/pkg/logger"
)

func main() {
	// Load configuration (fail-fast on missing/invalid env vars).
	cfg := config.Load()

	// Build structured logger.
	slogLogger := logger.New(cfg.LogLevel)

	// Connect to database.
	_, err := database.NewPostgresConnection(cfg)
	if err != nil {
		slogLogger.Error("failed to connect to database", "error", err)
		log.Fatal(err)
	}

	// Build router with middleware and routes.
	engine := router.NewRouter(cfg, slogLogger)

	// Start the server.
	addr := ":" + cfg.AppPort
	slogLogger.Info("starting server", "addr", addr)
	if err := engine.Run(addr); err != nil {
		slogLogger.Error("failed to start server", "error", err)
		log.Fatal(err)
	}
}
