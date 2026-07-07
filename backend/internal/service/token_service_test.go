package service

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Contract required from dev (internal/service/token_service.go):
//
//	type Claims struct { UserID uint; Role string; IssuedAt time.Time; ExpiresAt time.Time }
//	type TokenService interface {
//	    Issue(userID uint, role string) (string, error)
//	    Verify(tokenString string) (Claims, error)
//	}
//	func NewTokenService(secret string, ttl time.Duration) TokenService

func TestTokenService_TC01_IssueThenVerifyRoundTripsClaims(t *testing.T) {
	// Arrange
	svc := NewTokenService("test-secret", time.Hour)

	// Act
	tokenString, issueErr := svc.Issue(42, "admin")

	// Assert
	require.NoError(t, issueErr)
	require.NotEmpty(t, tokenString)

	claims, verifyErr := svc.Verify(tokenString)
	require.NoError(t, verifyErr)
	assert.Equal(t, uint(42), claims.UserID)
	assert.Equal(t, "admin", claims.Role)
	assert.False(t, claims.ExpiresAt.IsZero(), "exp must always be set")
	assert.False(t, claims.IssuedAt.IsZero(), "iat must always be set")
	assert.True(t, claims.ExpiresAt.After(claims.IssuedAt))
}

func TestTokenService_TC06_RejectsAlgNoneToken(t *testing.T) {
	// Arrange: craft a token with alg:none — must never be accepted regardless of payload.
	svc := NewTokenService("test-secret", time.Hour)
	noneClaims := jwt.MapClaims{
		"sub":  "1",
		"role": "admin",
		"exp":  time.Now().Add(time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, noneClaims)
	tokenString, signErr := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, signErr)

	// Act
	_, verifyErr := svc.Verify(tokenString)

	// Assert (AC6: signing method must be checked, alg:none rejected -> maps to 401 UNAUTHORIZED)
	assert.Error(t, verifyErr)
}

func TestTokenService_TC06_RejectsExpiredToken(t *testing.T) {
	// Arrange: craft an already-expired HS256 token signed with the same secret.
	svc := NewTokenService("test-secret", time.Hour)
	expiredClaims := jwt.MapClaims{
		"sub":  "1",
		"role": "admin",
		"exp":  time.Now().Add(-time.Minute).Unix(),
		"iat":  time.Now().Add(-time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	tokenString, signErr := tok.SignedString([]byte("test-secret"))
	require.NoError(t, signErr)

	// Act
	_, verifyErr := svc.Verify(tokenString)

	// Assert (AC6: exp already passed -> Verify must reject)
	assert.Error(t, verifyErr)
}
