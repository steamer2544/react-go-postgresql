// Package database provides Postgres connection helpers.
package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"imaxx-backend/internal/config"
)

// NewPostgresConnection builds a DSN from cfg and opens a GORM postgres connection.
// Returns the error on failure; never panics.
func NewPostgresConnection(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBSSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	return db, nil
}
