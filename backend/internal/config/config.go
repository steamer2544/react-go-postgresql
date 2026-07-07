// Package config loads and validates application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	AppEnv             string
	AppPort            string
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	DBSSLMode          string
	JWTSecret          string
	JWTAccessTTL       time.Duration
	JWTRefreshTTL      time.Duration
	CORSAllowedOrigins []string
	LogLevel           string
}

// Load reads configuration from .env file (if present) and environment variables.
// It fails fast: if any required variable is missing or invalid, it calls os.Exit(1).
func Load() *Config {
	// Try loading .env file; ignore error if file doesn't exist (prod uses real env vars).
	_ = godotenv.Load()

	c := &Config{}

	loadString("APP_ENV", &c.AppEnv)
	loadString("APP_PORT", &c.AppPort)
	loadString("DB_HOST", &c.DBHost)
	loadString("DB_PORT", &c.DBPort)
	loadString("DB_USER", &c.DBUser)
	loadString("DB_PASSWORD", &c.DBPassword)
	loadString("DB_NAME", &c.DBName)
	loadString("DB_SSLMODE", &c.DBSSLMode)
	loadString("JWT_SECRET", &c.JWTSecret)
	loadString("LOG_LEVEL", &c.LogLevel)

	c.JWTAccessTTL = mustDuration("JWT_ACCESS_TTL")
	c.JWTRefreshTTL = mustDuration("JWT_REFRESH_TTL")

	// Parse CORS origins from comma-separated string.
	corsRaw := os.Getenv("CORS_ALLOWED_ORIGINS")
	if corsRaw == "" {
		fmt.Fprintf(os.Stderr, "fatal: required environment variable CORS_ALLOWED_ORIGINS is not set\n")
		os.Exit(1)
	}
	c.CORSAllowedOrigins = strings.Split(corsRaw, ",")
	for i, origin := range c.CORSAllowedOrigins {
		c.CORSAllowedOrigins[i] = strings.TrimSpace(origin)
	}

	return c
}

func loadString(envKey string, target *string) {
	*target = os.Getenv(envKey)
	if *target == "" {
		fmt.Fprintf(os.Stderr, "fatal: required environment variable %s is not set\n", envKey)
		os.Exit(1)
	}
}

func mustDuration(envKey string) time.Duration {
	raw := os.Getenv(envKey)
	if raw == "" {
		fmt.Fprintf(os.Stderr, "fatal: required environment variable %s is not set\n", envKey)
		os.Exit(1)
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %s=%q is not a valid duration: %v\n", envKey, raw, err)
		os.Exit(1)
	}
	return d
}
