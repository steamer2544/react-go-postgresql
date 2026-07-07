package service

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"imaxx-backend/internal/dto"
	"imaxx-backend/internal/repository"
)

// SignatureUpload holds an incoming signature file for saving.
type SignatureUpload struct {
	ContentType string
	Size        int64
	Reader      io.Reader
}

// ProfileService handles profile-related business logic.
type ProfileService struct {
	repo         repository.UserRepository
	uploadDir    string
	maxBytes     int64
	allowedTypes []string
}

// NewProfileService creates a new ProfileService.
func NewProfileService(repo repository.UserRepository, uploadDir string, maxBytes int64, allowedTypes []string) *ProfileService {
	return &ProfileService{
		repo:         repo,
		uploadDir:    uploadDir,
		maxBytes:     maxBytes,
		allowedTypes: allowedTypes,
	}
}

// GetMe returns the profile data for a user.
func (s *ProfileService) GetMe(ctx context.Context, userID uint) (*dto.MeResponse, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &dto.MeResponse{
		ID:                 user.ID,
		Email:              user.Email,
		Role:               string(user.Role),
		FullName:           user.FullName,
		Position:           user.Position,
		SignatureImagePath: user.SignatureImagePath,
	}, nil
}

// UpdateProfile updates a user's full name and position.
func (s *ProfileService) UpdateProfile(ctx context.Context, userID uint, req dto.UpdateProfileRequest) error {
	return s.repo.UpdateProfile(ctx, userID, req.FullName, req.Position)
}

// SaveSignature validates and saves a signature image file.
func (s *ProfileService) SaveSignature(ctx context.Context, userID uint, upload SignatureUpload) (string, error) {
	if !contains(s.allowedTypes, upload.ContentType) {
		return "", ErrUnsupportedFileType
	}
	if upload.Size > s.maxBytes {
		return "", ErrFileTooLarge
	}
	ext := contentTypeToExt(upload.ContentType)
	filename := "user_" + strconv.FormatUint(uint64(userID), 10) + "." + ext
	path := filepath.Join(s.uploadDir, filename)
	if err := os.MkdirAll(s.uploadDir, 0o755); err != nil {
		return "", err
	}
	// Defense-in-depth: limit reads to maxBytes so a client that lies about
	// Size/Content-Length cannot exhaust server memory.
	data, err := io.ReadAll(io.LimitReader(upload.Reader, s.maxBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > int(s.maxBytes) {
		return "", ErrFileTooLarge
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	if err := s.repo.UpdateSignaturePath(ctx, userID, path); err != nil {
		return "", err
	}
	return path, nil
}

// GetSignaturePath returns the stored signature file path and its content-type.
func (s *ProfileService) GetSignaturePath(ctx context.Context, userID uint) (string, string, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return "", "", err
	}
	if user.SignatureImagePath == nil {
		return "", "", ErrNotFound
	}
	contentType := pathToContentType(*user.SignatureImagePath)
	return *user.SignatureImagePath, contentType, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func contentTypeToExt(ct string) string {
	switch ct {
	case "image/png":
		return "png"
	case "image/jpeg":
		return "jpg"
	default:
		return "png"
	}
}

func pathToContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "image/png"
	}
}
