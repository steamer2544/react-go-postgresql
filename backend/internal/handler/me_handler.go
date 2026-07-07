package handler

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"imaxx-backend/internal/dto"
	"imaxx-backend/internal/service"
	"imaxx-backend/pkg/response"
)

// MeServicer handles profile-related business logic.
type MeServicer interface {
	GetMe(ctx context.Context, userID uint) (*dto.MeResponse, error)
	UpdateProfile(ctx context.Context, userID uint, req dto.UpdateProfileRequest) error
	SaveSignature(ctx context.Context, userID uint, upload service.SignatureUpload) (string, error)
	GetSignaturePath(ctx context.Context, userID uint) (string, string, error)
}

// MeHandler handles HTTP requests for the current user's profile.
type MeHandler struct {
	svc MeServicer
}

// NewMeHandler creates a new MeHandler.
func NewMeHandler(svc MeServicer) *MeHandler {
	return &MeHandler{svc: svc}
}

// GetMe handles GET /me.
func (h *MeHandler) GetMe(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	me, err := h.svc.GetMe(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, me, "ok")
}

// UpdateProfile handles PUT /me/profile.
func (h *MeHandler) UpdateProfile(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, service.ErrValidation)
		return
	}
	if err := h.svc.UpdateProfile(c.Request.Context(), userID, req); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, nil, "updated")
}

// UploadSignature handles POST /me/signature.
func (h *MeHandler) UploadSignature(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	file, fileHeader, err := c.Request.FormFile("signature")
	if err != nil {
		response.Fail(c, service.ErrValidation)
		return
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	upload := service.SignatureUpload{
		ContentType: contentType,
		Size:        fileHeader.Size,
		Reader:      file,
	}

	path, err := h.svc.SaveSignature(c.Request.Context(), userID, upload)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, gin.H{"path": path}, "uploaded")
}

// GetSignature handles GET /me/signature.
func (h *MeHandler) GetSignature(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	path, contentType, err := h.svc.GetSignaturePath(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	// Stream the file with correct content-type
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	default:
		contentType = "image/png"
	}

	c.Header("Content-Type", contentType)
	c.File(path)
}
