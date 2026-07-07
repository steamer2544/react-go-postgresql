package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Contract required from dev (internal/config/config.go — BE-1):
//
//	Config gains: SignatureUploadDir string, SignatureMaxBytes int64, SignatureAllowedTypes []string
//	Load() fails fast (os.Exit(1)) when any of these required vars is missing.

func setAllRequiredEnvVars(t *testing.T, overrides map[string]string) {
	t.Helper()
	base := map[string]string{
		"APP_ENV":                 "dev",
		"APP_PORT":                "8080",
		"DB_HOST":                 "localhost",
		"DB_PORT":                 "5432",
		"DB_USER":                 "postgres",
		"DB_PASSWORD":             "postgres",
		"DB_NAME":                 "app_db",
		"DB_SSLMODE":              "disable",
		"JWT_SECRET":              "test-secret",
		"JWT_ACCESS_TTL":          "15m",
		"JWT_REFRESH_TTL":         "168h",
		"CORS_ALLOWED_ORIGINS":    "http://localhost:5173",
		"LOG_LEVEL":               "info",
		"SIGNATURE_UPLOAD_DIR":    "./uploads/signatures",
		"SIGNATURE_MAX_BYTES":     "2097152",
		"SIGNATURE_ALLOWED_TYPES": "image/png,image/jpeg",
	}
	for k, v := range overrides {
		base[k] = v
	}
	for k, v := range base {
		t.Setenv(k, v)
	}
}

func TestConfig_TC16_LoadsSignatureUploadSettingsFromEnv(t *testing.T) {
	// Arrange
	setAllRequiredEnvVars(t, nil)

	// Act
	cfg := Load()

	// Assert (AC16: new keys loaded into typed config)
	assert.Equal(t, "./uploads/signatures", cfg.SignatureUploadDir)
	assert.Equal(t, int64(2097152), cfg.SignatureMaxBytes)
	assert.Equal(t, []string{"image/png", "image/jpeg"}, cfg.SignatureAllowedTypes)
}

// TestConfig_TC16_MissingSignatureUploadDirFailsFast verifies fail-fast behaviour
// by re-executing this same test binary as a subprocess with SIGNATURE_UPLOAD_DIR
// unset — the standard Go idiom for testing os.Exit paths deterministically
// (no reliance on timing/flakiness).
func TestConfig_TC16_MissingSignatureUploadDirFailsFast(t *testing.T) {
	if os.Getenv("BE_CONFIG_TEST_CRASHER") == "1" {
		setAllRequiredEnvVars(t, map[string]string{"SIGNATURE_UPLOAD_DIR": ""})
		os.Unsetenv("SIGNATURE_UPLOAD_DIR")
		Load()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestConfig_TC16_MissingSignatureUploadDirFailsFast")
	cmd.Env = append(os.Environ(), "BE_CONFIG_TEST_CRASHER=1")
	err := cmd.Run()

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok, "expected process to exit non-zero (fail-fast), got err: %v", err)
	assert.False(t, exitErr.Success(), "expected non-zero exit code when SIGNATURE_UPLOAD_DIR is missing")
}

func TestEnvExample_TC16_BackendEnvExampleHasSignatureKeys(t *testing.T) {
	// Arrange
	path := filepath.Join("..", "..", ".env.example")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "expected backend/.env.example to exist")
	content := string(data)

	// Assert (AC16: .env.example synced with new required keys)
	for _, key := range []string{"SIGNATURE_UPLOAD_DIR", "SIGNATURE_MAX_BYTES", "SIGNATURE_ALLOWED_TYPES"} {
		assert.Contains(t, content, key, "backend/.env.example missing key %q", key)
	}
}
