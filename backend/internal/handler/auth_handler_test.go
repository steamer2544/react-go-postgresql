package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"imaxx-backend/internal/service"
)

// Contract required from dev (internal/handler/auth_handler.go):
//
//	type AuthServicer interface { Login(ctx context.Context, email, password string) (string, error) }
//	func NewAuthHandler(svc AuthServicer) *AuthHandler
//	func (h *AuthHandler) Login(c *gin.Context) // POST /auth/login, public

type mockAuthService struct {
	mock.Mock
}

func (m *mockAuthService) Login(ctx context.Context, email, password string) (string, error) {
	args := m.Called(ctx, email, password)
	return args.String(0), args.Error(1)
}

func decodeJSONBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func TestAuthHandler_TC01_LoginHappyPathReturns200WithAccessToken(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	svc := new(mockAuthService)
	svc.On("Login", mock.Anything, "user@example.com", "Sup3rSecret!").Return("signed.jwt.token", nil)
	h := NewAuthHandler(svc)
	router := gin.New()
	router.POST("/auth/login", h.Login)

	reqBody, _ := json.Marshal(map[string]string{"email": "user@example.com", "password": "Sup3rSecret!"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert (AC1)
	require.Equal(t, http.StatusOK, w.Code)
	body := decodeJSONBody(t, w)
	data, ok := body["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "signed.jwt.token", data["access_token"])
}

func TestAuthHandler_TC02_InvalidCredentialsReturns401Generic(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	svc := new(mockAuthService)
	svc.On("Login", mock.Anything, "user@example.com", "wrong-password").Return("", service.ErrInvalidCredentials)
	h := NewAuthHandler(svc)
	router := gin.New()
	router.POST("/auth/login", h.Login)

	reqBody, _ := json.Marshal(map[string]string{"email": "user@example.com", "password": "wrong-password"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert (AC2)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	body := decodeJSONBody(t, w)
	errObj := body["error"].(map[string]any)
	require.Equal(t, "UNAUTHORIZED", errObj["code"])
}

func TestAuthHandler_TC03_MissingEmailReturns400ValidationErrorWithoutCallingService(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	svc := new(mockAuthService)
	h := NewAuthHandler(svc)
	router := gin.New()
	router.POST("/auth/login", h.Login)

	reqBody, _ := json.Marshal(map[string]string{"password": "Sup3rSecret!"}) // email omitted
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert (AC3)
	require.Equal(t, http.StatusBadRequest, w.Code)
	body := decodeJSONBody(t, w)
	errObj := body["error"].(map[string]any)
	require.Equal(t, "VALIDATION_ERROR", errObj["code"])
	svc.AssertNotCalled(t, "Login", mock.Anything, mock.Anything, mock.Anything)
}

func TestAuthHandler_TC03_MissingPasswordReturns400ValidationError(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	svc := new(mockAuthService)
	h := NewAuthHandler(svc)
	router := gin.New()
	router.POST("/auth/login", h.Login)

	reqBody, _ := json.Marshal(map[string]string{"email": "user@example.com"}) // password omitted
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert (AC3)
	require.Equal(t, http.StatusBadRequest, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "VALIDATION_ERROR", errObj["code"])
}
