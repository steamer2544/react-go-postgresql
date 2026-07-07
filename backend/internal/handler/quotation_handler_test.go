package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"imaxx-backend/internal/dto"
	"imaxx-backend/internal/middleware"
	"imaxx-backend/internal/service"
)

// ─────────────────────────────────────────────
// Contract required from dev (internal/handler/quotation_handler.go):
//
//	type QuotationServicer interface {
//	    CreateQuotation(ctx context.Context, userID uint, req dto.CreateQuotationRequest) (*dto.QuotationResponse, error)
//	    UpdateQuotation(ctx context.Context, userID uint, role string, id uint, req dto.UpdateQuotationRequest) (*dto.QuotationResponse, error)
//	    DeleteQuotation(ctx context.Context, userID uint, role string, id uint) error
//	    GetQuotation(ctx context.Context, id uint) (*dto.QuotationResponse, error)
//	    ListQuotations(ctx context.Context, query dto.ListQuotationQuery) ([]dto.QuotationResponse, int64, error)
//	}
//	func NewQuotationHandler(svc QuotationServicer) *QuotationHandler
//	func (h *QuotationHandler) Create(c *gin.Context)  // POST /quotations -> 201
//	func (h *QuotationHandler) List(c *gin.Context)    // GET /quotations -> 200 list envelope
//	func (h *QuotationHandler) Get(c *gin.Context)     // GET /quotations/:id -> 200
//	func (h *QuotationHandler) Update(c *gin.Context)  // PUT /quotations/:id -> 200
//	func (h *QuotationHandler) Delete(c *gin.Context)  // DELETE /quotations/:id -> 204

// ─────────────────────────────────────────────
// mockQuotationService — implements QuotationServicer via testify/mock
// ─────────────────────────────────────────────

type mockQuotationService struct {
	mock.Mock
}

func (m *mockQuotationService) CreateQuotation(ctx context.Context, userID uint, req dto.CreateQuotationRequest) (*dto.QuotationResponse, error) {
	args := m.Called(ctx, userID, req)
	resp, _ := args.Get(0).(*dto.QuotationResponse)
	return resp, args.Error(1)
}

func (m *mockQuotationService) UpdateQuotation(ctx context.Context, userID uint, role string, id uint, req dto.UpdateQuotationRequest) (*dto.QuotationResponse, error) {
	args := m.Called(ctx, userID, role, id, req)
	resp, _ := args.Get(0).(*dto.QuotationResponse)
	return resp, args.Error(1)
}

func (m *mockQuotationService) DeleteQuotation(ctx context.Context, userID uint, role string, id uint) error {
	args := m.Called(ctx, userID, role, id)
	return args.Error(0)
}

func (m *mockQuotationService) GetQuotation(ctx context.Context, id uint) (*dto.QuotationResponse, error) {
	args := m.Called(ctx, id)
	resp, _ := args.Get(0).(*dto.QuotationResponse)
	return resp, args.Error(1)
}

func (m *mockQuotationService) ListQuotations(ctx context.Context, query dto.ListQuotationQuery) ([]dto.QuotationResponse, int64, error) {
	args := m.Called(ctx, query)
	resp, _ := args.Get(0).([]dto.QuotationResponse)
	total, _ := args.Get(1).(int64)
	return resp, total, args.Error(2)
}

// ─────────────────────────────────────────────
// fakeVerifier — implements middleware.TokenVerifier for RBAC wiring tests
// ─────────────────────────────────────────────

type fakeVerifier struct{ claims map[string]service.Claims }

func (f *fakeVerifier) Verify(tokenString string) (service.Claims, error) {
	c, ok := f.claims[tokenString]
	if !ok {
		return service.Claims{}, service.ErrUnauthorized
	}
	return c, nil
}

// withUser sets both userID and role in the gin context, mirroring what
// middleware.Auth does in production (auth.go: c.Set("userID"...), c.Set("role"...)).
// Needed by role-dependent handler tests (Update/Delete) that bypass the full
// auth stack; the shared withUserID helper sets userID only.
func withUser(userID uint, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("role", role)
	}
}

// ─────────────────────────────────────────────
// TC-HDL-01  (AC12, GET /quotations/:id)
// ─────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_01(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)

	custName := "Customer Name"
	custPos := "CEO"
	svc.On("GetQuotation", mock.Anything, uint(42)).Return(&dto.QuotationResponse{
		ID:             42,
		ReferenceNo:    "QT2607001",
		Status:         "draft",
		Subtotal:       2751.50,
		DiscountAmount: 151.50,
		VatAmount:      182.00,
		Total:          2782.00,
		Items: []dto.QuotationItemResponse{
			{ServiceType: "Design", Description: "Website design", UnitPrice: 1000.00, Qty: 2, LineTotal: 2000.00, SortOrder: 1},
		},
		CompanySigneeName:      "Somchai Dev",
		CompanySigneePosition:  "Sales Manager",
		CustomerSigneeName:     &custName,
		CustomerSigneePosition: &custPos,
		CreatedBy:              7,
	}, nil)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.GET("/quotations/:id", withUserID(7), h.Get)

	req := httptest.NewRequest(http.MethodGet, "/quotations/42", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := decodeJSONBody(t, w)
	data := body["data"].(map[string]any)
	require.Equal(t, "QT2607001", data["reference_no"])
	require.Equal(t, "draft", data["status"])
	require.Equal(t, 2751.50, data["subtotal"])
	require.Equal(t, 151.50, data["discount_amount"])
	require.Equal(t, 182.00, data["vat_amount"])
	require.Equal(t, 2782.00, data["total"])
	require.Equal(t, float64(7), data["created_by"])

	items := data["items"].([]any)
	require.Len(t, items, 1)
	item0 := items[0].(map[string]any)
	require.Equal(t, "Design", item0["service_type"])
	require.Equal(t, 1000.00, item0["unit_price"])
	require.Equal(t, float64(2), item0["qty"])
	require.Equal(t, 2000.00, item0["line_total"])
	require.Equal(t, float64(1), item0["sort_order"])

	require.Equal(t, "Somchai Dev", data["company_signee_name"])
	require.Equal(t, "Sales Manager", data["company_signee_position"])

	_, hasKey := data["customer_signee_name"]
	require.True(t, hasKey)
}

// ─────────────────────────────────────────────
// TC-HDL-02  (POST /quotations happy 201)
// ─────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_02(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	svc.On("CreateQuotation", mock.Anything, uint(7), mock.AnythingOfType("dto.CreateQuotationRequest")).Return(&dto.QuotationResponse{
		ID:          1,
		ReferenceNo: "QT2607001",
		Status:      "draft",
	}, nil)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.POST("/quotations", withUserID(7), h.Create)

	reqBody, _ := json.Marshal(dto.CreateQuotationRequest{
		Attention:      "Mr. Somchai",
		Company:        "Acme Corp",
		Email:          "somchai@acme.com",
		Date:           "2026-07-07",
		ValidUntil:     "2026-08-07",
		DiscountAmount: 0,
		Items: []dto.QuotationItemInput{
			{ServiceType: "Design", Description: "Website design", UnitPrice: 1000.00, Qty: 2, SortOrder: 1},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/quotations", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	body := decodeJSONBody(t, w)
	data := body["data"].(map[string]any)
	require.Equal(t, "QT2607001", data["reference_no"])
	require.NotEmpty(t, body["message"])
}

// ─────────────────────────────────────────────
// TC-HDL-03  (POST /quotations, ErrValidation -> 400)
// ─────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_03(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	svc.On("CreateQuotation", mock.Anything, uint(7), mock.AnythingOfType("dto.CreateQuotationRequest")).Return((*dto.QuotationResponse)(nil), service.ErrValidation)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.POST("/quotations", withUserID(7), h.Create)

	reqBody, _ := json.Marshal(dto.CreateQuotationRequest{
		Attention:      "Mr. Somchai",
		Company:        "Acme Corp",
		Email:          "somchai@acme.com",
		Date:           "2026-07-07",
		ValidUntil:     "2026-08-07",
		DiscountAmount: 0,
		Items: []dto.QuotationItemInput{
			{ServiceType: "Design", Description: "Website design", UnitPrice: 1000.00, Qty: 2, SortOrder: 1},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/quotations", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "VALIDATION_ERROR", errObj["code"])
}

// ─────────────────────────────────────────────
// TC-HDL-04  (PUT /quotations/:id, ErrForbidden -> 403)
// ─────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_04(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	svc.On("UpdateQuotation", mock.Anything, uint(7), "creator", uint(1), mock.AnythingOfType("dto.UpdateQuotationRequest")).Return((*dto.QuotationResponse)(nil), service.ErrForbidden)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.PUT("/quotations/:id", withUser(7, "creator"), h.Update)

	reqBody, _ := json.Marshal(dto.UpdateQuotationRequest{
		Attention:      "Mr. Somchai",
		Company:        "Acme Corp",
		Email:          "somchai@acme.com",
		Date:           "2026-07-07",
		ValidUntil:     "2026-08-07",
		DiscountAmount: 0,
		Items: []dto.QuotationItemInput{
			{ServiceType: "Design", Description: "Website design", UnitPrice: 1000.00, Qty: 2, SortOrder: 1},
		},
	})
	req := httptest.NewRequest(http.MethodPut, "/quotations/1", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "FORBIDDEN", errObj["code"])
}

// ─────────────────────────────────────────────
// TC-HDL-05  (DELETE /quotations/:id happy -> 204)
// ─────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_05(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	svc.On("DeleteQuotation", mock.Anything, uint(7), "creator", uint(1)).Return(nil)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.DELETE("/quotations/:id", withUser(7, "creator"), h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/quotations/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, 0, w.Body.Len())
}

// ─────────────────────────────────────────────
// TC-HDL-06  (DELETE /quotations/:id, ErrForbidden -> 403)
// ─────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_06(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	svc.On("DeleteQuotation", mock.Anything, uint(7), "creator", uint(1)).Return(service.ErrForbidden)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.DELETE("/quotations/:id", withUser(7, "creator"), h.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/quotations/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "FORBIDDEN", errObj["code"])
}

// ─────────────────────────────────────────────
// TC-HDL-07  (GET /quotations list envelope)
// ─────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_07(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	svc.On("ListQuotations", mock.Anything, mock.MatchedBy(func(q dto.ListQuotationQuery) bool {
		return q.Page == 1 && q.PageSize == 20 && q.Sort == "-created_at" && q.Status == "draft"
	})).Return([]dto.QuotationResponse{
		{ID: 1, ReferenceNo: "QT2607001"},
		{ID: 2, ReferenceNo: "QT2607002"},
	}, int64(2), nil)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.GET("/quotations", withUserID(7), h.List)

	req := httptest.NewRequest(http.MethodGet, "/quotations?page=1&page_size=20&sort=-created_at&status=draft", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := decodeJSONBody(t, w)
	data := body["data"].([]any)
	require.Len(t, data, 2)
	meta := body["meta"].(map[string]any)
	require.Equal(t, float64(1), meta["page"])
	require.Equal(t, float64(20), meta["page_size"])
	require.Equal(t, float64(2), meta["total"])
}

// ─────────────────────────────────────────────
// TC-HDL-08  (GET /quotations?sort=unit_price -> 400)
// ─────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_08(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	h := NewQuotationHandler(svc)
	router := gin.New()
	router.GET("/quotations", withUserID(7), h.List)

	req := httptest.NewRequest(http.MethodGet, "/quotations?sort=unit_price", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "VALIDATION_ERROR", errObj["code"])
	svc.AssertNotCalled(t, "ListQuotations", mock.Anything, mock.Anything)
}

// ─────────────────────────────────────────────
// TC-HDL-09  (GET /quotations?foo=bar -> 400)
// ─────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_09(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	h := NewQuotationHandler(svc)
	router := gin.New()
	router.GET("/quotations", withUserID(7), h.List)

	req := httptest.NewRequest(http.MethodGet, "/quotations?foo=bar", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "VALIDATION_ERROR", errObj["code"])
	svc.AssertNotCalled(t, "ListQuotations", mock.Anything, mock.Anything)
}

// ─────────────────────────────────────────────
// TC-HDL-10  (GET /quotations?page_size=1000 -> 400)
// ─────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_10(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	h := NewQuotationHandler(svc)
	router := gin.New()
	router.GET("/quotations", withUserID(7), h.List)

	req := httptest.NewRequest(http.MethodGet, "/quotations?page_size=1000", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "VALIDATION_ERROR", errObj["code"])
	svc.AssertNotCalled(t, "ListQuotations", mock.Anything, mock.Anything)
}

// ─────────────────────────────────────────────
// TC-HDL-11  (RBAC: approver POST -> 403)
// ─────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_11(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	verifier := &fakeVerifier{claims: map[string]service.Claims{
		"approver-token": {UserID: 7, Role: "approver"},
	}}
	h := NewQuotationHandler(svc)

	router := gin.New()
	protected := router.Group("", middleware.Auth(verifier))
	write := protected.Group("", middleware.RequireRole("admin", "creator"))
	write.POST("/quotations", h.Create)

	reqBody, _ := json.Marshal(dto.CreateQuotationRequest{
		Attention:      "Mr. Somchai",
		Company:        "Acme Corp",
		Email:          "somchai@acme.com",
		Date:           "2026-07-07",
		ValidUntil:     "2026-08-07",
		DiscountAmount: 0,
		Items: []dto.QuotationItemInput{
			{ServiceType: "Design", Description: "Website design", UnitPrice: 1000.00, Qty: 2, SortOrder: 1},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/quotations", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer approver-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "FORBIDDEN", errObj["code"])
	svc.AssertNotCalled(t, "CreateQuotation", mock.Anything, mock.Anything, mock.Anything)
}

// ─────────────────────────────────────────────
// TC-HDL-12  (RBAC: approver GET list -> 200)
// ─────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_12(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	verifier := &fakeVerifier{claims: map[string]service.Claims{
		"approver-token": {UserID: 7, Role: "approver"},
	}}
	svc.On("ListQuotations", mock.Anything, mock.Anything).Return([]dto.QuotationResponse{}, int64(0), nil)

	h := NewQuotationHandler(svc)

	router := gin.New()
	protected := router.Group("", middleware.Auth(verifier))
	protected.GET("/quotations", h.List)

	req := httptest.NewRequest(http.MethodGet, "/quotations", nil)
	req.Header.Set("Authorization", "Bearer approver-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

// ─────────────────────────────────────────────
// TC-HDL-13  (RBAC: approver GET /quotations/:id -> 200)
// ─────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_13(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	verifier := &fakeVerifier{claims: map[string]service.Claims{
		"approver-token": {UserID: 7, Role: "approver"},
	}}
	svc.On("GetQuotation", mock.Anything, uint(1)).Return(&dto.QuotationResponse{ID: 1}, nil)

	h := NewQuotationHandler(svc)

	router := gin.New()
	protected := router.Group("", middleware.Auth(verifier))
	protected.GET("/quotations/:id", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/quotations/1", nil)
	req.Header.Set("Authorization", "Bearer approver-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

// ─────────────────────────────────────────────
// TC-HDL-14  (No auth header -> 401)
// ─────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_14(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	verifier := &fakeVerifier{claims: map[string]service.Claims{
		"approver-token": {UserID: 7, Role: "approver"},
	}}

	h := NewQuotationHandler(svc)

	router := gin.New()
	protected := router.Group("", middleware.Auth(verifier))
	protected.GET("/quotations", h.List)

	req := httptest.NewRequest(http.MethodGet, "/quotations", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "UNAUTHORIZED", errObj["code"])
}

// ─────────────────────────────────────────────
// TC-HDL-15  (RBAC: creator POST -> 201)
// ─────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_15(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	verifier := &fakeVerifier{claims: map[string]service.Claims{
		"creator-token": {UserID: 7, Role: "creator"},
	}}
	svc.On("CreateQuotation", mock.Anything, uint(7), mock.Anything).Return(&dto.QuotationResponse{
		ID:          5,
		ReferenceNo: "QT2607005",
		Status:      "draft",
	}, nil)

	h := NewQuotationHandler(svc)

	router := gin.New()
	protected := router.Group("", middleware.Auth(verifier))
	write := protected.Group("", middleware.RequireRole("admin", "creator"))
	write.POST("/quotations", h.Create)

	reqBody, _ := json.Marshal(dto.CreateQuotationRequest{
		Attention:      "Mr. Somchai",
		Company:        "Acme Corp",
		Email:          "somchai@acme.com",
		Date:           "2026-07-07",
		ValidUntil:     "2026-08-07",
		DiscountAmount: 0,
		Items: []dto.QuotationItemInput{
			{ServiceType: "Design", Description: "Website design", UnitPrice: 1000.00, Qty: 2, SortOrder: 1},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/quotations", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer creator-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
}

// ─────────────────────────────────────────────
// TC-HDL-16  (AC19, 500 no internal leak)
// ─────────────────────────────────────────────

func TestQuotationHandler_TC_HDL_16(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	svc.On("GetQuotation", mock.Anything, uint(1)).Return(
		(*dto.QuotationResponse)(nil),
		strErr("dsn=postgres://user:pass@host db connection refused at repository.go:123"),
	)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.GET("/quotations/:id", withUserID(7), h.Get)

	req := httptest.NewRequest(http.MethodGet, "/quotations/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "INTERNAL_ERROR", errObj["code"])

	lowered := strings.ToLower(w.Body.String())
	require.NotContains(t, lowered, "dsn=")
	require.NotContains(t, lowered, "connection refused")
	require.NotContains(t, lowered, "repository.go")
}

// strErr wraps a string into an error type.
type strErr string

func (e strErr) Error() string { return string(e) }
