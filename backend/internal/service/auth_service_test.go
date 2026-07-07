package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"imaxx-backend/internal/model"
)

// Contract required from dev (internal/service/auth_service.go, internal/service/errors.go):
//
//	var ErrInvalidCredentials = errors.New(...)
//	func NewAuthService(repo repository.UserRepository, tokenSvc TokenService) *AuthService
//	func (s *AuthService) Login(ctx context.Context, email, password string) (string, error)

func TestAuthService_TC01_LoginHappyPathReturnsJWTWithRequiredClaims(t *testing.T) {
	// Arrange
	hash, err := HashPassword("Sup3rSecret!")
	require.NoError(t, err)
	repo := new(mockUserRepository)
	repo.On("FindByEmail", mock.Anything, "user@example.com").
		Return(&model.User{ID: 7, Email: "user@example.com", PasswordHash: hash, Role: model.RoleCreator}, nil)
	svc := NewAuthService(repo, NewTokenService("test-secret", time.Hour))

	// Act
	accessToken, loginErr := svc.Login(context.Background(), "user@example.com", "Sup3rSecret!")

	// Assert (AC1: 200 equivalent — valid token whose claims decode to sub/role/exp/iat)
	require.NoError(t, loginErr)
	require.NotEmpty(t, accessToken)

	parsed, parseErr := jwt.Parse(accessToken, func(*jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	require.NoError(t, parseErr)
	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, strconv.Itoa(7), claims["sub"])
	assert.Equal(t, string(model.RoleCreator), claims["role"])
	assert.NotNil(t, claims["exp"])
	assert.NotNil(t, claims["iat"])
}

func TestAuthService_TC02_WrongPasswordReturnsGenericInvalidCredentials(t *testing.T) {
	// Arrange
	hash, err := HashPassword("correct-password")
	require.NoError(t, err)
	repo := new(mockUserRepository)
	repo.On("FindByEmail", mock.Anything, "user@example.com").
		Return(&model.User{ID: 1, Email: "user@example.com", PasswordHash: hash, Role: model.RoleCreator}, nil)
	svc := NewAuthService(repo, NewTokenService("test-secret", time.Hour))

	// Act
	_, loginErr := svc.Login(context.Background(), "user@example.com", "wrong-password")

	// Assert (AC2: 401 UNAUTHORIZED, generic — same error as unknown email)
	require.ErrorIs(t, loginErr, ErrInvalidCredentials)
}

func TestAuthService_TC02_UnknownEmailReturnsSameGenericInvalidCredentials(t *testing.T) {
	// Arrange
	repo := new(mockUserRepository)
	repo.On("FindByEmail", mock.Anything, "missing@example.com").
		Return(nil, ErrNotFound)
	svc := NewAuthService(repo, NewTokenService("test-secret", time.Hour))

	// Act
	_, loginErr := svc.Login(context.Background(), "missing@example.com", "whatever")

	// Assert (AC2: must NOT be distinguishable from wrong-password case — user enumeration guard)
	require.ErrorIs(t, loginErr, ErrInvalidCredentials)
}

func TestAuthService_TC14_ErrorNeverLeaksRawPasswordValue(t *testing.T) {
	// Arrange
	repo := new(mockUserRepository)
	repo.On("FindByEmail", mock.Anything, "user@example.com").Return(nil, ErrNotFound)
	svc := NewAuthService(repo, NewTokenService("test-secret", time.Hour))

	// Act
	_, loginErr := svc.Login(context.Background(), "user@example.com", "TopSecretPass123")

	// Assert (AC14: no sensitive value should ever appear in an error that could be logged)
	require.Error(t, loginErr)
	assert.NotContains(t, loginErr.Error(), "TopSecretPass123")
}
