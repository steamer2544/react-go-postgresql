// Package handler handles HTTP requests for quotations.
package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"imaxx-backend/internal/dto"
	"imaxx-backend/internal/service"
	"imaxx-backend/pkg/response"
)

// QuotationServicer handles quotation business logic.
type QuotationServicer interface {
	CreateQuotation(ctx context.Context, userID uint, req dto.CreateQuotationRequest) (*dto.QuotationResponse, error)
	UpdateQuotation(ctx context.Context, userID uint, role string, id uint, req dto.UpdateQuotationRequest) (*dto.QuotationResponse, error)
	DeleteQuotation(ctx context.Context, userID uint, role string, id uint) error
	GetQuotation(ctx context.Context, id uint) (*dto.QuotationResponse, error)
	ListQuotations(ctx context.Context, query dto.ListQuotationQuery) ([]dto.QuotationResponse, int64, error)
}

// QuotationHandler handles HTTP requests for quotations.
type QuotationHandler struct {
	svc QuotationServicer
}

// NewQuotationHandler creates a new QuotationHandler.
func NewQuotationHandler(svc QuotationServicer) *QuotationHandler {
	return &QuotationHandler{svc: svc}
}

// createQueryKeyWhitelist returns a set of allowed query parameter keys for the list endpoint.
func createQueryKeyWhitelist() map[string]struct{} {
	return map[string]struct{}{
		"page":       {},
		"page_size":  {},
		"sort":       {},
		"status":     {},
		"created_by": {},
		"date_gte":   {},
		"date_lte":   {},
		"q":          {},
	}
}

// createSortWhitelist returns a set of allowed sort values for the list endpoint.
func createSortWhitelist() map[string]struct{} {
	return map[string]struct{}{
		"created_at":    {},
		"-created_at":   {},
		"date":          {},
		"-date":         {},
		"total":         {},
		"-total":        {},
		"reference_no":  {},
		"-reference_no": {},
	}
}

// Create handles POST /quotations.
func (h *QuotationHandler) Create(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req dto.CreateQuotationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, service.ErrValidation)
		return
	}
	resp, err := h.svc.CreateQuotation(c.Request.Context(), userID, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, http.StatusCreated, resp, "created")
}

// Update handles PUT /quotations/:id.
func (h *QuotationHandler) Update(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	role := c.GetString("role")
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Fail(c, service.ErrValidation)
		return
	}
	var req dto.UpdateQuotationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, service.ErrValidation)
		return
	}
	resp, err := h.svc.UpdateQuotation(c.Request.Context(), userID, role, uint(id), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, resp, "updated")
}

// Delete handles DELETE /quotations/:id.
func (h *QuotationHandler) Delete(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	role := c.GetString("role")
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Fail(c, service.ErrValidation)
		return
	}
	if err := h.svc.DeleteQuotation(c.Request.Context(), userID, role, uint(id)); err != nil {
		response.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Get handles GET /quotations/:id.
func (h *QuotationHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Fail(c, service.ErrValidation)
		return
	}
	resp, err := h.svc.GetQuotation(c.Request.Context(), uint(id))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, http.StatusOK, resp, "ok")
}

// List handles GET /quotations with query-string whitelist validation.
func (h *QuotationHandler) List(c *gin.Context) {
	// 1. Whitelist raw query keys.
	queryKeyWhitelist := createQueryKeyWhitelist()
	for key := range c.Request.URL.Query() {
		if _, ok := queryKeyWhitelist[strings.ToLower(key)]; !ok {
			response.Fail(c, service.ErrValidation)
			return
		}
	}

	// 2. Bind query params.
	var query dto.ListQuotationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Fail(c, service.ErrValidation)
		return
	}

	// 3. Whitelist sort value.
	if query.Sort != "" {
		sortWhitelist := createSortWhitelist()
		if _, ok := sortWhitelist[query.Sort]; !ok {
			response.Fail(c, service.ErrValidation)
			return
		}
	}

	data, total, err := h.svc.ListQuotations(c.Request.Context(), query)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.List(c, data, query.Page, query.PageSize, total)
}
