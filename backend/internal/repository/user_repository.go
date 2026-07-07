// Package repository defines the data-access interface and its GORM implementation.
package repository

import (
	"context"

	"imaxx-backend/internal/model"

	"gorm.io/gorm"
)

// UserRepository is the interface for user data access.
type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id uint) (*model.User, error)
	UpdateProfile(ctx context.Context, id uint, fullName string, position string) error
	UpdateSignaturePath(ctx context.Context, id uint, path string) error
}

type gormUserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a UserRepository backed by the given GORM db.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &gormUserRepository{db: db}
}

func (r *gormUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *gormUserRepository) FindByID(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *gormUserRepository) UpdateProfile(ctx context.Context, id uint, fullName string, position string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"full_name": fullName,
		"position":  position,
	}).Error
}

func (r *gormUserRepository) UpdateSignaturePath(ctx context.Context, id uint, path string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("signature_image_path", path).Error
}
