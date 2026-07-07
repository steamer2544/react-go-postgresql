package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"imaxx-backend/internal/service"
)

// Contract required from dev (pkg/response/response.go — BE-11):
//
//	func Fail(c *gin.Context, err error) // maps sentinel domain error -> code + HTTP status
//
// Existing Success/Error/List signatures must be left unchanged.

func ginTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request = req
	return c, w
}

func decodeErrorBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok, "expected top-level \"error\" object in response body")
	return errObj
}

func TestFail_TC_NotFoundMapsTo404NotFound(t *testing.T) {
	// Arrange
	c, w := ginTestContext()

	// Act
	Fail(c, service.ErrNotFound)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "NOT_FOUND", decodeErrorBody(t, w)["code"])
}

func TestFail_TC_ConflictMapsTo409Conflict(t *testing.T) {
	c, w := ginTestContext()

	Fail(c, service.ErrConflict)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "CONFLICT", decodeErrorBody(t, w)["code"])
}

func TestFail_TC03_ValidationMapsTo400ValidationError(t *testing.T) {
	c, w := ginTestContext()

	Fail(c, service.ErrValidation)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "VALIDATION_ERROR", decodeErrorBody(t, w)["code"])
}

func TestFail_TC02_InvalidCredentialsMapsTo401Unauthorized(t *testing.T) {
	c, w := ginTestContext()

	Fail(c, service.ErrInvalidCredentials)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "UNAUTHORIZED", decodeErrorBody(t, w)["code"])
}

func TestFail_TC05_UnauthorizedMapsTo401Unauthorized(t *testing.T) {
	c, w := ginTestContext()

	Fail(c, service.ErrUnauthorized)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "UNAUTHORIZED", decodeErrorBody(t, w)["code"])
}

func TestFail_TC07_ForbiddenMapsTo403Forbidden(t *testing.T) {
	c, w := ginTestContext()

	Fail(c, service.ErrForbidden)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, "FORBIDDEN", decodeErrorBody(t, w)["code"])
}

func TestFail_TC12_UnsupportedFileTypeMapsTo400ValidationError(t *testing.T) {
	c, w := ginTestContext()

	Fail(c, service.ErrUnsupportedFileType)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "VALIDATION_ERROR", decodeErrorBody(t, w)["code"])
}

func TestFail_TC13_FileTooLargeMapsTo400ValidationError(t *testing.T) {
	c, w := ginTestContext()

	Fail(c, service.ErrFileTooLarge)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "VALIDATION_ERROR", decodeErrorBody(t, w)["code"])
}

func TestFail_TC15_UnknownErrorMapsTo500InternalErrorWithoutLeakingDetail(t *testing.T) {
	// Arrange: an internal error carrying a detail that must never reach the client.
	c, w := ginTestContext()
	internalErr := errors.New("pq: syntax error near SELECT * FROM secret_table at /var/app/internal/db.go:42")

	// Act
	Fail(c, internalErr)

	// Assert (AC15: 500 INTERNAL_ERROR, generic message, no SQL/path leak)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	errObj := decodeErrorBody(t, w)
	assert.Equal(t, "INTERNAL_ERROR", errObj["code"])
	message, _ := errObj["message"].(string)
	assert.NotContains(t, message, "SELECT")
	assert.NotContains(t, message, "secret_table")
	assert.NotContains(t, message, "/var/app")
}
