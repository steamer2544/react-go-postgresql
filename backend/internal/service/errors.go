// Package service defines domain error sentinels used across all service layers.
package service

import "errors"

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrNotFound            = errors.New("not found")
	ErrConflict            = errors.New("conflict")
	ErrValidation          = errors.New("validation error")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrUnsupportedFileType = errors.New("unsupported file type")
	ErrFileTooLarge        = errors.New("file too large")
)
