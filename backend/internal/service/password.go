package service

import "golang.org/x/crypto/bcrypt"

// HashPassword hashes a plain-text password using bcrypt with DefaultCost (10).
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// ComparePassword compares a bcrypt hash against a plain-text password.
func ComparePassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
