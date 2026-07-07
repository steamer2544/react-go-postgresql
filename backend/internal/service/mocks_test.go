package service

import (
	"context"

	"github.com/stretchr/testify/mock"

	"imaxx-backend/internal/model"
	"imaxx-backend/internal/repository"
)

// mockUserRepository is a hand-written testify mock implementing
// repository.UserRepository (see .claude/docs/testing.md — "testify/mock
// เขียนเอง" is the default strategy for small interfaces). It keeps the
// internal/service unit tests independent from the real GORM repository.
type mockUserRepository struct {
	mock.Mock
}

// Compile-time contract check: dev must define repository.UserRepository
// with exactly this method set for AuthService/ProfileService to depend on.
var _ repository.UserRepository = (*mockUserRepository)(nil)

func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	user, _ := args.Get(0).(*model.User)
	return user, args.Error(1)
}

func (m *mockUserRepository) FindByID(ctx context.Context, id uint) (*model.User, error) {
	args := m.Called(ctx, id)
	user, _ := args.Get(0).(*model.User)
	return user, args.Error(1)
}

func (m *mockUserRepository) UpdateProfile(ctx context.Context, id uint, fullName string, position string) error {
	args := m.Called(ctx, id, fullName, position)
	return args.Error(0)
}

func (m *mockUserRepository) UpdateSignaturePath(ctx context.Context, id uint, path string) error {
	args := m.Called(ctx, id, path)
	return args.Error(0)
}
