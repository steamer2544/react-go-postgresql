package service

import (
	"context"

	"imaxx-backend/internal/repository"
)

// AuthService handles authentication logic.
type AuthService struct {
	repo     repository.UserRepository
	tokenSvc TokenService
}

// NewAuthService creates a new AuthService.
func NewAuthService(repo repository.UserRepository, tokenSvc TokenService) *AuthService {
	return &AuthService{repo: repo, tokenSvc: tokenSvc}
}

// Login authenticates a user by email and password, returning a JWT on success.
func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return "", ErrInvalidCredentials
	}
	if err := ComparePassword(user.PasswordHash, password); err != nil {
		return "", ErrInvalidCredentials
	}
	token, err := s.tokenSvc.Issue(user.ID, string(user.Role))
	if err != nil {
		return "", ErrInvalidCredentials
	}
	return token, nil
}
