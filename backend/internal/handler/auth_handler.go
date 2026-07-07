package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"imaxx-backend/internal/dto"
	"imaxx-backend/internal/service"
	"imaxx-backend/pkg/response"
)

// AuthServicer handles authentication business logic.
type AuthServicer interface {
	Login(ctx context.Context, email, password string) (string, error)
}

// AuthHandler handles HTTP requests for authentication.
type AuthHandler struct {
	svc AuthServicer
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(svc AuthServicer) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, service.ErrValidation)
		return
	}
	token, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, dto.LoginResponse{AccessToken: token}, "login successful")
}
