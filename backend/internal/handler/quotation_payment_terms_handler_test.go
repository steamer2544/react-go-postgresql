package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"imaxx-backend/internal/dto"
)

// Contract required from dev (internal/handler/quotation_handler.go):
//
//	func (h *QuotationHandler) Create(c *gin.Context)  // POST /quotations -> 201

func TestPaymentTermsHandler_TC_HDL_PT01_BindingAmountZeroAC4(t *testing.T) {
	// Arrange — AC4: payment_terms with amount=0 should fail binding (gt=0)
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	h := NewQuotationHandler(svc)
	router := gin.New()
	router.POST("/quotations", withUserID(7), h.Create)

	// Build JSON body with invalid payment term (amount=0) using the real DTO struct.
	reqBody, err := json.Marshal(dto.CreateQuotationRequest{
		Attention:      "Mr. Somchai",
		Company:        "Acme Corp",
		Email:          "somchai@acme.com",
		Date:           "2026-07-01",
		ValidUntil:     "2026-07-31",
		DiscountAmount: 0,
		Items: []dto.QuotationItemInput{
			{ServiceType: "Design", Description: "Website design", UnitPrice: 1000.00, Qty: 2, SortOrder: 1},
		},
		PaymentTerms: []dto.PaymentTermInput{
			{Description: "Bad", Amount: 0},
		},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/quotations", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert — binding should reject before reaching service
	require.Equal(t, http.StatusBadRequest, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "VALIDATION_ERROR", errObj["code"])
	svc.AssertNotCalled(t, "CreateQuotation", mock.Anything, mock.Anything, mock.Anything)
}

func TestPaymentTermsHandler_TC_HDL_PT02_BindingNegativeAmountAC4(t *testing.T) {
	// Arrange — AC4: payment_terms with amount=-50 should fail binding (gt=0)
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	h := NewQuotationHandler(svc)
	router := gin.New()
	router.POST("/quotations", withUserID(7), h.Create)

	reqBody, err := json.Marshal(dto.CreateQuotationRequest{
		Attention:      "Mr. Somchai",
		Company:        "Acme Corp",
		Email:          "somchai@acme.com",
		Date:           "2026-07-01",
		ValidUntil:     "2026-07-31",
		DiscountAmount: 0,
		Items: []dto.QuotationItemInput{
			{ServiceType: "Design", Description: "Website design", UnitPrice: 1000.00, Qty: 2, SortOrder: 1},
		},
		PaymentTerms: []dto.PaymentTermInput{
			{Description: "Bad", Amount: -50},
		},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/quotations", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	router.ServeHTTP(w, req)

	// Assert
	require.Equal(t, http.StatusBadRequest, w.Code)
	errObj := decodeJSONBody(t, w)["error"].(map[string]any)
	require.Equal(t, "VALIDATION_ERROR", errObj["code"])
	svc.AssertNotCalled(t, "CreateQuotation", mock.Anything, mock.Anything, mock.Anything)
}

func TestPaymentTermsHandler_TC_HDL_PT03_PassthroughResponseAC1(t *testing.T) {
	// Arrange — AC1: svc returns QuotationResponse with 3 PaymentTerms, handler passthrough
	gin.SetMode(gin.TestMode)
	svc := new(mockQuotationService)
	svc.On("CreateQuotation", mock.Anything, uint(7), mock.AnythingOfType("dto.CreateQuotationRequest")).Return(&dto.QuotationResponse{
		ID:          1,
		ReferenceNo: "QT2607001",
		Status:      "draft",
		PaymentTerms: []dto.PaymentTermResponse{
			{ID: 1, TermNo: 1, Description: "Deposit", Amount: 891.67, SortOrder: 1},
			{ID: 2, TermNo: 2, Description: "Progress", Amount: 891.67, SortOrder: 2},
			{ID: 3, TermNo: 3, Description: "Final", Amount: 891.66, SortOrder: 3},
		},
	}, nil)

	h := NewQuotationHandler(svc)
	router := gin.New()
	router.POST("/quotations", withUserID(7), h.Create)

	// POST body without payment_terms (service mock handles the response)
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

	// Act
	router.ServeHTTP(w, req)

	// Assert
	require.Equal(t, http.StatusCreated, w.Code)
	data := decodeJSONBody(t, w)["data"].(map[string]any)
	require.Equal(t, "QT2607001", data["reference_no"])
	require.Equal(t, "draft", data["status"])

	paymentTerms := data["payment_terms"].([]any)
	require.Len(t, paymentTerms, 3)

	item0 := paymentTerms[0].(map[string]any)
	require.Equal(t, float64(1), item0["term_no"])
	require.Equal(t, "Deposit", item0["description"])
	require.Equal(t, 891.67, item0["amount"])
	require.Equal(t, float64(1), item0["sort_order"])

	item1 := paymentTerms[1].(map[string]any)
	require.Equal(t, float64(2), item1["term_no"])
	require.Equal(t, "Progress", item1["description"])
	require.Equal(t, 891.67, item1["amount"])
	require.Equal(t, float64(2), item1["sort_order"])

	item2 := paymentTerms[2].(map[string]any)
	require.Equal(t, float64(3), item2["term_no"])
	require.Equal(t, "Final", item2["description"])
	require.Equal(t, 891.66, item2["amount"])
	require.Equal(t, float64(3), item2["sort_order"])
}
