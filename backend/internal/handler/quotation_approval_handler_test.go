package handler

import (
	"context"
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
	"imaxx-backend/internal/middleware"
	"imaxx-backend/internal/service"
)

// Contract required from dev (internal/handler/quotation_handler.go), added to
// the existing QuotationServicer interface:
//
//	SubmitQuotation(ctx context.Context, userID uint, role string, id uint) (*dto.QuotationResponse, error)
//	ApproveQuotation(ctx context.Context, userID uint, role string, id uint) (*dto.QuotationResponse, error)
//	RejectQuotation(ctx context.Context, userID uint, role string, id uint) (*dto.QuotationResponse, error)
//	GetApprovalSignaturePath(ctx context.Context, id uint) (path string, contentType string, err error)
//
// New handler methods, same request/response pattern as Update/Delete:
//	func (h *QuotationHandler) Submit(c *gin.Context)   // POST /quotations/:id/submit  -> 200
//	func (h *QuotationHandler) Approve(c *gin.Context)  // POST /quotations/:id/approve -> 200
//	func (h *QuotationHandler) Reject(c *gin.Context)   // POST /quotations/:id/reject  -> 200
//	func (h *QuotationHandler) GetApprovalSignature(c *gin.Context) // GET /quotations/:id/approval-signature -> stream image (pattern identical to MeHandler.GetSignature)
//
// This file adds SubmitQuotation/ApproveQuotation/RejectQuotation/GetApprovalSignaturePath
// methods to mockQuotationService (struct declared in quotation_handler_test.go —
// do not duplicate the struct here). Reuses withUser, withUserID, decodeJSONBody,
// fakeVerifier already declared in existing handler test files.

func (m *mockQuotationService) SubmitQuotation(ctx context.Context, userID uint, role string, id uint) (*dto.QuotationResponse, error) {
	args := m.Called(ctx, userID, role, id)
	resp, _ := args.Get(0).(*dto.QuotationResponse)
	return resp, args.Error(1)
}

func (m *mockQuotationService) ApproveQuotation(ctx context.Context, userID uint, role string, id uint) (*dto.QuotationResponse, error) {
	args := m.Called(ctx, userID, role, id)
	resp, _ := args.Get(0).(*dto.QuotationResponse)
	return resp, args.Error(1)
}

func (m *mockQuotationService) RejectQuotation(ctx context.Context, userID uint, role string, id uint) (*dto.QuotationResponse, error) {
	args := m.Called(ctx, userID, role, id)
	resp, _ := args.Get(0).(*dto.QuotationResponse)
	return resp, args.Error(1)
}

func (m *mockQuotationService) GetApprovalSignaturePath(ctx context.Context, id uint) (string, string, error) {
	args := m.Called(ctx, id)
	return args.String(0), args.String(1), args.Error(2)
}

// ─── AC1: Submit happy ────────────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_A01_SubmitHappy200AC1(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	svc.On("SubmitQuotation", mock.Anything, uint(7), "creator", uint(1)).
		Return(&dto.QuotationResponse{ID: 1, Status: "pending_approval"}, nil)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.POST("/quotations/:id/submit", withUser(7, "creator"), h.Submit)

	req := httptest.NewRequest(http.MethodPost, "/quotations/1/submit", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := decodeJSONBody(t, w)
	data := body["data"].(map[string]any)
	require.Equal(t, "pending_approval", data["status"])
}

// ─── AC2: Submit forbidden ─────────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_A02_SubmitForbidden403AC2(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	svc.On("SubmitQuotation", mock.Anything, uint(7), "creator", uint(1)).
		Return((*dto.QuotationResponse)(nil), service.ErrForbidden)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.POST("/quotations/:id/submit", withUser(7, "creator"), h.Submit)

	req := httptest.NewRequest(http.MethodPost, "/quotations/1/submit", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "FORBIDDEN", errObj["code"])
}

// ─── AC2b: Submit conflict ──────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_A03_SubmitConflict409AC2b(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	svc.On("SubmitQuotation", mock.Anything, uint(7), "creator", uint(1)).
		Return((*dto.QuotationResponse)(nil), service.ErrConflict)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.POST("/quotations/:id/submit", withUser(7, "creator"), h.Submit)

	req := httptest.NewRequest(http.MethodPost, "/quotations/1/submit", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusConflict, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "CONFLICT", errObj["code"])
}

// ─── AC3: Approve RBAC forbidden (route-level, non-approver role) ────────

func TestQuotationHandler_TC_HDL_A04_ApproveForbiddenRBAC403AC3(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	verifier := &fakeVerifier{claims: map[string]service.Claims{
		"creator-token": {UserID: 7, Role: "creator"},
	}}
	h := NewQuotationHandler(svc)

	router := gin.New()
	protected := router.Group("", middleware.Auth(verifier))
	approvalOnly := protected.Group("", middleware.RequireRole("approver"))
	approvalOnly.POST("/quotations/:id/approve", h.Approve)

	req := httptest.NewRequest(http.MethodPost, "/quotations/1/approve", nil)
	req.Header.Set("Authorization", "Bearer creator-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "FORBIDDEN", errObj["code"])
	svc.AssertNotCalled(t, "ApproveQuotation", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// Admin must ALSO be forbidden by the route-level RBAC (Decision 3: no admin bypass).
func TestQuotationHandler_TC_HDL_A05_ApproveForbiddenAdminRBAC403AC3(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	verifier := &fakeVerifier{claims: map[string]service.Claims{
		"admin-token": {UserID: 1, Role: "admin"},
	}}
	h := NewQuotationHandler(svc)

	router := gin.New()
	protected := router.Group("", middleware.Auth(verifier))
	approvalOnly := protected.Group("", middleware.RequireRole("approver"))
	approvalOnly.POST("/quotations/:id/approve", h.Approve)

	req := httptest.NewRequest(http.MethodPost, "/quotations/1/approve", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	svc.AssertNotCalled(t, "ApproveQuotation", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// ─── AC4: Approve happy ───────────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_A06_ApproveHappy200AC4(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	approverID := uint(9)
	approvedAt := "2026-07-15T10:00:00Z"
	signeeName := "Approver Name"
	signeePos := "CFO"
	svc.On("ApproveQuotation", mock.Anything, uint(9), "approver", uint(1)).
		Return(&dto.QuotationResponse{
			ID: 1, Status: "approved",
			ApproverID: &approverID, ApprovedAt: &approvedAt,
			ApprovedSigneeName: &signeeName, ApprovedSigneePosition: &signeePos,
			HasApprovedSignature: true,
		}, nil)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.POST("/quotations/:id/approve", withUser(9, "approver"), h.Approve)

	req := httptest.NewRequest(http.MethodPost, "/quotations/1/approve", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := decodeJSONBody(t, w)
	data := body["data"].(map[string]any)
	require.Equal(t, "approved", data["status"])
	require.Equal(t, float64(9), data["approver_id"])
	require.Equal(t, "2026-07-15T10:00:00Z", data["approved_at"])
	require.Equal(t, "Approver Name", data["approved_signee_name"])
	require.Equal(t, "CFO", data["approved_signee_position"])
	require.Equal(t, true, data["has_approved_signature"])
}

// ─── AC5: Approve validation error ─────────────────────────────────────────

func TestQuotationHandler_TC_HDL_A07_ApproveValidation400AC5(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	svc.On("ApproveQuotation", mock.Anything, uint(9), "approver", uint(1)).
		Return((*dto.QuotationResponse)(nil), service.ErrValidation)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.POST("/quotations/:id/approve", withUser(9, "approver"), h.Approve)

	req := httptest.NewRequest(http.MethodPost, "/quotations/1/approve", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "VALIDATION_ERROR", errObj["code"])
}

// ─── AC5b: Approve conflict ─────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_A08_ApproveConflict409AC5b(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	svc.On("ApproveQuotation", mock.Anything, uint(9), "approver", uint(1)).
		Return((*dto.QuotationResponse)(nil), service.ErrConflict)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.POST("/quotations/:id/approve", withUser(9, "approver"), h.Approve)

	req := httptest.NewRequest(http.MethodPost, "/quotations/1/approve", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusConflict, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "CONFLICT", errObj["code"])
}

// ─── AC6: Reject happy ─────────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_A09_RejectHappy200AC6(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	svc.On("RejectQuotation", mock.Anything, uint(9), "approver", uint(1)).
		Return(&dto.QuotationResponse{ID: 1, Status: "rejected"}, nil)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.POST("/quotations/:id/reject", withUser(9, "approver"), h.Reject)

	req := httptest.NewRequest(http.MethodPost, "/quotations/1/reject", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := decodeJSONBody(t, w)
	data := body["data"].(map[string]any)
	require.Equal(t, "rejected", data["status"])
}

// ─── AC3: Reject RBAC forbidden (route-level) ─────────────────────────────

func TestQuotationHandler_TC_HDL_A10_RejectForbiddenRBAC403AC3(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	verifier := &fakeVerifier{claims: map[string]service.Claims{
		"creator-token": {UserID: 7, Role: "creator"},
	}}
	h := NewQuotationHandler(svc)

	router := gin.New()
	protected := router.Group("", middleware.Auth(verifier))
	approvalOnly := protected.Group("", middleware.RequireRole("approver"))
	approvalOnly.POST("/quotations/:id/reject", h.Reject)

	req := httptest.NewRequest(http.MethodPost, "/quotations/1/reject", nil)
	req.Header.Set("Authorization", "Bearer creator-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	svc.AssertNotCalled(t, "RejectQuotation", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// ─── AC8: GetApprovalSignature streams image ──────────────────────────────

func TestQuotationHandler_TC_HDL_A11_GetApprovalSignatureStreams200AC8(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	tmpFile := filepath.Join(t.TempDir(), "approved_1.png")
	require.NoError(t, os.WriteFile(tmpFile, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, 0o600))
	svc.On("GetApprovalSignaturePath", mock.Anything, uint(1)).Return(tmpFile, "image/png", nil)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.GET("/quotations/:id/approval-signature", withUserID(7), h.GetApprovalSignature)

	req := httptest.NewRequest(http.MethodGet, "/quotations/1/approval-signature", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.True(t, strings.HasPrefix(w.Header().Get("Content-Type"), "image/"))
	require.NotEmpty(t, w.Body.Bytes())
}

// ─── AC8b: GetApprovalSignature not found ────────────────────────────────

func TestQuotationHandler_TC_HDL_A12_GetApprovalSignatureNotFound404AC8b(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	svc.On("GetApprovalSignaturePath", mock.Anything, uint(1)).Return("", "", service.ErrNotFound)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.GET("/quotations/:id/approval-signature", withUserID(7), h.GetApprovalSignature)

	req := httptest.NewRequest(http.MethodGet, "/quotations/1/approval-signature", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "NOT_FOUND", errObj["code"])
}

// ─── AC8: does not leak the raw file path in the JSON of GET /quotations/:id ──
// (regression guard for Decision 5 — QuotationResponse must expose only the
// boolean has_approved_signature, never approved_signature_path)

func TestQuotationHandler_TC_HDL_A13_GetQuotationDoesNotLeakSignaturePathDecision5(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	svc.On("GetQuotation", mock.Anything, uint(1)).Return(&dto.QuotationResponse{
		ID: 1, Status: "approved", HasApprovedSignature: true,
	}, nil)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.GET("/quotations/:id", withUserID(7), h.Get)

	req := httptest.NewRequest(http.MethodGet, "/quotations/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotContains(t, w.Body.String(), "approved_signature_path")
	body := decodeJSONBody(t, w)
	data := body["data"].(map[string]any)
	require.Equal(t, true, data["has_approved_signature"])
}

// ─── Decision 1: ListQuotationQuery.Status oneof updated (sent -> pending_approval) ──

func TestQuotationHandler_TC_HDL_A14_ListStatusFilterAcceptsPendingApprovalDecision1(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	svc.On("ListQuotations", mock.Anything, mock.MatchedBy(func(q dto.ListQuotationQuery) bool {
		return q.Status == "pending_approval"
	})).Return([]dto.QuotationResponse{}, int64(0), nil)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.GET("/quotations", withUserID(7), h.List)

	req := httptest.NewRequest(http.MethodGet, "/quotations?status=pending_approval", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestQuotationHandler_TC_HDL_A15_ListStatusFilterRejectsOldSentValueDecision1(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	h := NewQuotationHandler(svc)
	router := gin.New()
	router.GET("/quotations", withUserID(7), h.List)

	req := httptest.NewRequest(http.MethodGet, "/quotations?status=sent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "VALIDATION_ERROR", errObj["code"])
	svc.AssertNotCalled(t, "ListQuotations", mock.Anything, mock.Anything)
}
