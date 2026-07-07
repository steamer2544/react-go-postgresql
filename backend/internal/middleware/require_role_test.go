package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Contract required from dev (internal/middleware/require_role.go):
//
//	func RequireRole(roles ...string) gin.HandlerFunc
//	  - reads role set on gin.Context by Auth middleware (key "role")
//	  - role not in roles -> abort with 403 FORBIDDEN
//	  - role in roles -> calls next
//	  - MUST NOT consider any client-supplied value (e.g. query string) — see AC7

func newRequireRoleTestRouter(contextRole string, allowed ...string) (*gin.Engine, *bool) {
	gin.SetMode(gin.TestMode)
	nextCalled := false
	r := gin.New()
	r.GET("/admin/ping", func(c *gin.Context) {
		c.Set("role", contextRole)
	}, RequireRole(allowed...), func(c *gin.Context) {
		nextCalled = true
		c.Status(http.StatusOK)
	})
	return r, &nextCalled
}

func TestRequireRole_TC07_RoleNotAllowedReturns403(t *testing.T) {
	// Arrange (creator token trying to reach an admin-only route)
	router, nextCalled := newRequireRoleTestRouter("creator", "admin")
	req := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, *nextCalled)
}

func TestRequireRole_TC07_RoleAllowedReturns200(t *testing.T) {
	// Arrange
	router, nextCalled := newRequireRoleTestRouter("admin", "admin")
	req := httptest.NewRequest(http.MethodGet, "/admin/ping", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, *nextCalled)
}

func TestRequireRole_TC07_ClientSuppliedRoleQueryParamHasNoEffect(t *testing.T) {
	// Arrange: token role is "creator" (set into context, simulating Auth middleware);
	// attacker tries to escalate via ?role=admin on the request itself.
	router, nextCalled := newRequireRoleTestRouter("creator", "admin")
	req := httptest.NewRequest(http.MethodGet, "/admin/ping?role=admin", nil)
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert (AC7: authorization decided from token/context only, never from client input)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, *nextCalled)
}
