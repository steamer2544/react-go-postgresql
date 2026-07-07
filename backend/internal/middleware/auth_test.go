package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"imaxx-backend/internal/service"
)

// Contract required from dev (internal/middleware/auth.go):
//
//	type TokenVerifier interface { Verify(tokenString string) (service.Claims, error) }
//	func Auth(verifier TokenVerifier) gin.HandlerFunc
//	  - missing/invalid/expired token -> abort with 401 UNAUTHORIZED
//	  - valid token -> c.Set("userID", claims.UserID), c.Set("role", claims.Role), calls next

type fakeTokenVerifier struct {
	claims service.Claims
	err    error
}

func (f fakeTokenVerifier) Verify(string) (service.Claims, error) {
	return f.claims, f.err
}

func newAuthTestRouter(verifier TokenVerifier, next gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/me", Auth(verifier), next)
	return r
}

func TestAuth_TC05_MissingAuthorizationHeaderReturns401(t *testing.T) {
	// Arrange
	nextCalled := false
	router := newAuthTestRouter(fakeTokenVerifier{}, func(c *gin.Context) { nextCalled = true; c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert (AC5)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, nextCalled, "handler must not run when Authorization header is missing")
}

func TestAuth_TC06_InvalidOrExpiredTokenReturns401(t *testing.T) {
	// Arrange
	nextCalled := false
	router := newAuthTestRouter(fakeTokenVerifier{err: errors.New("signature invalid")}, func(c *gin.Context) {
		nextCalled = true
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer some.invalid.token")
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert (AC6)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, nextCalled)
}

func TestAuth_TC_ValidTokenSetsUserIDAndRoleThenCallsNext(t *testing.T) {
	// Arrange
	var gotUserID uint
	var gotRole string
	router := newAuthTestRouter(fakeTokenVerifier{claims: service.Claims{UserID: 9, Role: "creator"}}, func(c *gin.Context) {
		gotUserID = c.MustGet("userID").(uint)
		gotRole = c.MustGet("role").(string)
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, uint(9), gotUserID)
	assert.Equal(t, "creator", gotRole)
}
