package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// Contract required from dev (internal/service/password.go):
//
//	func HashPassword(password string) (string, error)
//	func ComparePassword(hash, password string) error

func TestHashPassword_TC04_ProducesBcryptHashWithMinCost10(t *testing.T) {
	// Arrange
	plain := "Sup3rSecret!"

	// Act
	hash, err := HashPassword(plain)

	// Assert (AC4: bcrypt hash, $2a$/$2b$ prefix, cost >= 10, never plaintext)
	require.NoError(t, err)
	require.NotEqual(t, plain, hash)
	assert.True(t, strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$"),
		"expected bcrypt hash prefix, got %q", hash)
	cost, costErr := bcrypt.Cost([]byte(hash))
	require.NoError(t, costErr)
	assert.GreaterOrEqual(t, cost, 10)
}

func TestComparePassword_TC04_CorrectPasswordMatchesViaBcryptCompare(t *testing.T) {
	// Arrange
	hash, err := HashPassword("Sup3rSecret!")
	require.NoError(t, err)

	// Act
	err = ComparePassword(hash, "Sup3rSecret!")

	// Assert
	assert.NoError(t, err)
}

func TestComparePassword_TC04_WrongPasswordDoesNotMatch(t *testing.T) {
	// Arrange
	hash, err := HashPassword("Sup3rSecret!")
	require.NoError(t, err)

	// Act
	err = ComparePassword(hash, "totally-wrong-password")

	// Assert
	assert.Error(t, err)
}
