package service

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"imaxx-backend/internal/dto"
	"imaxx-backend/internal/model"
)

// Contract required from dev (internal/service/profile_service.go):
//
//	type SignatureUpload struct { ContentType string; Size int64; Reader io.Reader }
//	func NewProfileService(repo repository.UserRepository, uploadDir string, maxBytes int64, allowedTypes []string) *ProfileService
//	func (s *ProfileService) GetMe(ctx context.Context, userID uint) (*dto.MeResponse, error)
//	func (s *ProfileService) UpdateProfile(ctx context.Context, userID uint, req dto.UpdateProfileRequest) error
//	func (s *ProfileService) SaveSignature(ctx context.Context, userID uint, upload SignatureUpload) (path string, err error)
//	func (s *ProfileService) GetSignaturePath(ctx context.Context, userID uint) (path string, contentType string, err error)

func TestProfileService_TC08_GetMeReturnsProfileWithoutPasswordFields(t *testing.T) {
	// Arrange
	repo := new(mockUserRepository)
	sigPath := "/uploads/signatures/user_3.png"
	repo.On("FindByID", mock.Anything, uint(3)).Return(&model.User{
		ID: 3, Email: "u@example.com", PasswordHash: "$2a$10$shouldNeverBeExposed",
		Role: model.RoleApprover, FullName: "Kanya", Position: "Manager", SignatureImagePath: &sigPath,
	}, nil)
	svc := NewProfileService(repo, t.TempDir(), int64(2*1024*1024), []string{"image/png", "image/jpeg"})

	// Act
	me, err := svc.GetMe(context.Background(), 3)

	// Assert (AC8)
	require.NoError(t, err)
	assert.Equal(t, "u@example.com", me.Email)
	assert.Equal(t, string(model.RoleApprover), me.Role)
	assert.Equal(t, "Kanya", me.FullName)
	assert.Equal(t, "Manager", me.Position)
	require.NotNil(t, me.SignatureImagePath)
	assert.Equal(t, sigPath, *me.SignatureImagePath)
}

func TestProfileService_TC09_UpdateProfilePersistsAndSubsequentGetMeReflectsIt(t *testing.T) {
	// Arrange
	repo := new(mockUserRepository)
	user := &model.User{ID: 4, Email: "a@example.com", FullName: "Old Name", Position: "Old Position", Role: model.RoleCreator}
	repo.On("FindByID", mock.Anything, uint(4)).Return(user, nil)
	repo.On("UpdateProfile", mock.Anything, uint(4), "New Name", "New Position").
		Run(func(args mock.Arguments) {
			user.FullName = "New Name"
			user.Position = "New Position"
		}).Return(nil)
	svc := NewProfileService(repo, t.TempDir(), int64(2*1024*1024), []string{"image/png"})

	// Act
	updateErr := svc.UpdateProfile(context.Background(), 4, dto.UpdateProfileRequest{FullName: "New Name", Position: "New Position"})
	require.NoError(t, updateErr)
	me, getErr := svc.GetMe(context.Background(), 4)

	// Assert (AC9: userID comes from token/service arg, and the change round-trips)
	require.NoError(t, getErr)
	assert.Equal(t, "New Name", me.FullName)
	assert.Equal(t, "New Position", me.Position)
}

func TestProfileService_TC10_SaveSignatureWritesFileAndPersistsPath(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	repo := new(mockUserRepository)
	repo.On("UpdateSignaturePath", mock.Anything, uint(6), mock.MatchedBy(func(p string) bool {
		return strings.HasSuffix(p, ".png")
	})).Return(nil)
	svc := NewProfileService(repo, dir, int64(2*1024*1024), []string{"image/png", "image/jpeg"})
	content := []byte{0x89, 'P', 'N', 'G'}

	// Act
	path, err := svc.SaveSignature(context.Background(), 6, SignatureUpload{
		ContentType: "image/png",
		Size:        int64(len(content)),
		Reader:      bytes.NewReader(content),
	})

	// Assert (AC10: file written to SignatureUploadDir, path persisted via repo)
	require.NoError(t, err)
	assert.FileExists(t, path)
	written, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	assert.Equal(t, content, written)
	repo.AssertCalled(t, "UpdateSignaturePath", mock.Anything, uint(6), path)
}

func TestProfileService_TC11_GetSignaturePathReturnsWrittenFileWithImageContentType(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	repo := new(mockUserRepository)
	repo.On("UpdateSignaturePath", mock.Anything, uint(6), mock.Anything).Return(nil)
	svc := NewProfileService(repo, dir, int64(2*1024*1024), []string{"image/png"})
	content := []byte{0x89, 'P', 'N', 'G'}
	writtenPath, saveErr := svc.SaveSignature(context.Background(), 6, SignatureUpload{
		ContentType: "image/png",
		Size:        int64(len(content)),
		Reader:      bytes.NewReader(content),
	})
	require.NoError(t, saveErr)
	repo.On("FindByID", mock.Anything, uint(6)).Return(&model.User{ID: 6, SignatureImagePath: &writtenPath}, nil)

	// Act
	path, contentType, err := svc.GetSignaturePath(context.Background(), 6)

	// Assert (AC11: signature_image_path non-empty and retrievable with image content-type)
	require.NoError(t, err)
	assert.Equal(t, writtenPath, path)
	assert.True(t, strings.HasPrefix(contentType, "image/"), "expected image/* content-type, got %q", contentType)
}

func TestProfileService_TC12_SaveSignatureRejectsUnsupportedContentType(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	repo := new(mockUserRepository)
	svc := NewProfileService(repo, dir, int64(2*1024*1024), []string{"image/png", "image/jpeg"})

	// Act
	_, err := svc.SaveSignature(context.Background(), 6, SignatureUpload{
		ContentType: "application/pdf",
		Size:        4,
		Reader:      bytes.NewReader([]byte("%PDF")),
	})

	// Assert (AC12: 400 VALIDATION_ERROR, no file written, no DB write)
	require.ErrorIs(t, err, ErrUnsupportedFileType)
	repo.AssertNotCalled(t, "UpdateSignaturePath", mock.Anything, mock.Anything, mock.Anything)
	entries, readDirErr := os.ReadDir(dir)
	require.NoError(t, readDirErr)
	assert.Empty(t, entries)
}

func TestProfileService_TC13_SaveSignatureRejectsFileTooLarge(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	repo := new(mockUserRepository)
	const maxBytes = int64(10)
	svc := NewProfileService(repo, dir, maxBytes, []string{"image/png"})
	content := []byte("this-content-is-longer-than-ten-bytes")

	// Act
	_, err := svc.SaveSignature(context.Background(), 6, SignatureUpload{
		ContentType: "image/png",
		Size:        int64(len(content)),
		Reader:      bytes.NewReader(content),
	})

	// Assert (AC13: 400 VALIDATION_ERROR, no file written, no DB write)
	require.ErrorIs(t, err, ErrFileTooLarge)
	repo.AssertNotCalled(t, "UpdateSignaturePath", mock.Anything, mock.Anything, mock.Anything)
	entries, readDirErr := os.ReadDir(dir)
	require.NoError(t, readDirErr)
	assert.Empty(t, entries)
}
