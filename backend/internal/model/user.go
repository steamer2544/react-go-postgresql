// Package model defines GORM models for the application.
package model

import "time"

// Role represents a user role in the system.
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleCreator  Role = "creator"
	RoleApprover Role = "approver"
)

// User is the application user model mapped to the "users" table.
type User struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	Email              string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash       string    `gorm:"column:password_hash;not null" json:"-"`
	Role               Role      `gorm:"type:varchar(20);not null" json:"role"`
	FullName           string    `gorm:"column:full_name" json:"full_name"`
	Position           string    `gorm:"type:varchar(100)" json:"position"`
	SignatureImagePath *string   `gorm:"column:signature_image_path" json:"signature_image_path,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
