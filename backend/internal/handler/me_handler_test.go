package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"imaxx-backend/internal/dto"
	"imaxx-backend/internal/service"
)

// Contract required from dev (internal/handler/me_handler.go):
//
//	type MeServicer interface {
//	    GetMe(ctx context.Context, userID uint) (*dto.MeResponse, error)
//	    UpdateProfile(ctx context.Context, userID uint, req dto.UpdateProfileRequest) error
//	    SaveSignature(ctx context.Context, userID uint, upload service.SignatureUpload) (path string, err error)
//	    GetSignaturePath(ctx context.Context, userID uint) (path string, contentType string, err error)
//	}
//	func NewMeHandler(svc MeServicer) *MeHandler
//	Methods: GetMe (GET /me), UpdateProfile (PUT /me/profile),
//	         UploadSignature (POST /me/signature, multipart field "signature"), GetSignature (GET /me/signature)
//	All handlers read the authenticated userID from gin.Context key "userID" (set by middleware.Auth).

type mockMeService struct {
	mock.Mock
}

func (m *mockMeService) GetMe(ctx context.Context, userID uint) (*dto.MeResponse, error) {
	args := m.Called(ctx, userID)
	resp, _ := args.Get(0).(*dto.MeResponse)
	return resp, args.Error(1)
}

func (m *mockMeService) UpdateProfile(ctx context.Context, userID uint, req dto.UpdateProfileRequest) error {
	args := m.Called(ctx, userID, req)
	return args.Error(0)
}

func (m *mockMeService) SaveSignature(ctx context.Context, userID uint, upload service.SignatureUpload) (string, error) {
	args := m.Called(ctx, userID, upload)
	return args.String(0), args.Error(1)
}

func (m *mockMeService) GetSignaturePath(ctx context.Context, userID uint) (string, string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.String(1), args.Error(2)
}

func withUserID(userID uint) gin.HandlerFunc {
	return func(c *gin.Context) { c.Set("userID", userID) }
}

func multipartFileRequest(t *testing.T, fieldName, fileName, contentType string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="` + fieldName + `"; filename="` + fileName + `"`},
		"Content-Type":        {contentType},
	})
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return body, writer.FormDataContentType()
}

func TestMeHandler_TC08_GetMeDoesNotLeakPasswordFields(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	svc := new(mockMeService)
	sigPath := "/uploads/signatures/user_5.png"
	svc.On("GetMe", mock.Anything, uint(5)).Return(&dto.MeResponse{
		ID: 5, Email: "user@example.com", Role: "creator", FullName: "Somchai", Position: "Staff",
		SignatureImagePath: &sigPath,
	}, nil)
	h := NewMeHandler(svc)
	router := gin.New()
	router.GET("/me", withUserID(5), h.GetMe)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert (AC8: 200, expected fields present, password/password_hash absent anywhere in body)
	require.Equal(t, http.StatusOK, w.Code)
	lowered := strings.ToLower(w.Body.String())
	require.NotContains(t, lowered, "password")
	body := decodeJSONBody(t, w)
	data := body["data"].(map[string]any)
	require.Equal(t, "user@example.com", data["email"])
	require.Equal(t, "creator", data["role"])
	require.Equal(t, "Somchai", data["full_name"])
	require.Equal(t, "Staff", data["position"])
	require.Equal(t, sigPath, data["signature_image_path"])
}

func TestMeHandler_TC09_UpdateProfileCallsServiceWithAuthenticatedUserID(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	svc := new(mockMeService)
	svc.On("UpdateProfile", mock.Anything, uint(5), dto.UpdateProfileRequest{FullName: "New Name", Position: "Lead"}).Return(nil)
	h := NewMeHandler(svc)
	router := gin.New()
	router.PUT("/me/profile", withUserID(5), h.UpdateProfile)

	reqBody, _ := json.Marshal(map[string]string{"full_name": "New Name", "position": "Lead"})
	req := httptest.NewRequest(http.MethodPut, "/me/profile", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert (AC9: updates only the authenticated user, identified from context/token)
	require.Equal(t, http.StatusOK, w.Code)
	svc.AssertCalled(t, "UpdateProfile", mock.Anything, uint(5), dto.UpdateProfileRequest{FullName: "New Name", Position: "Lead"})
}

func TestMeHandler_TC10_UploadValidPngReturns200(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	svc := new(mockMeService)
	svc.On("SaveSignature", mock.Anything, uint(5), mock.AnythingOfType("service.SignatureUpload")).
		Return("/uploads/signatures/user_5.png", nil)
	h := NewMeHandler(svc)
	router := gin.New()
	router.POST("/me/signature", withUserID(5), h.UploadSignature)

	reqBody, contentType := multipartFileRequest(t, "signature", "sig.png", "image/png", []byte{0x89, 'P', 'N', 'G'})
	req := httptest.NewRequest(http.MethodPost, "/me/signature", reqBody)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert (AC10)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMeHandler_TC12_UploadUnsupportedFileTypeReturns400ValidationError(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	svc := new(mockMeService)
	svc.On("SaveSignature", mock.Anything, uint(5), mock.AnythingOfType("service.SignatureUpload")).
		Return("", service.ErrUnsupportedFileType)
	h := NewMeHandler(svc)
	router := gin.New()
	router.POST("/me/signature", withUserID(5), h.UploadSignature)

	reqBody, contentType := multipartFileRequest(t, "signature", "malware.pdf", "application/pdf", []byte("%PDF-1.4"))
	req := httptest.NewRequest(http.MethodPost, "/me/signature", reqBody)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert (AC12)
	require.Equal(t, http.StatusBadRequest, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "VALIDATION_ERROR", errObj["code"])
}

func TestMeHandler_TC13_UploadTooLargeReturns400ValidationError(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	svc := new(mockMeService)
	svc.On("SaveSignature", mock.Anything, uint(5), mock.AnythingOfType("service.SignatureUpload")).
		Return("", service.ErrFileTooLarge)
	h := NewMeHandler(svc)
	router := gin.New()
	router.POST("/me/signature", withUserID(5), h.UploadSignature)

	reqBody, contentType := multipartFileRequest(t, "signature", "big.png", "image/png", bytes.Repeat([]byte{0xFF}, 64))
	req := httptest.NewRequest(http.MethodPost, "/me/signature", reqBody)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert (AC13)
	require.Equal(t, http.StatusBadRequest, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "VALIDATION_ERROR", errObj["code"])
}

func TestMeHandler_TC11_GetSignatureStreamsFileWithImageContentType(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	svc := new(mockMeService)
	tmpFile := filepath.Join(t.TempDir(), "user_5.png")
	require.NoError(t, os.WriteFile(tmpFile, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, 0o600))
	svc.On("GetSignaturePath", mock.Anything, uint(5)).Return(tmpFile, "image/png", nil)
	h := NewMeHandler(svc)
	router := gin.New()
	router.GET("/me/signature", withUserID(5), h.GetSignature)

	req := httptest.NewRequest(http.MethodGet, "/me/signature", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert (AC11)
	require.Equal(t, http.StatusOK, w.Code)
	require.True(t, strings.HasPrefix(w.Header().Get("Content-Type"), "image/"))
	require.NotEmpty(t, w.Body.Bytes())
}
