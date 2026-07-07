// Package service provides business logic for quotations.
package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"imaxx-backend/internal/dto"
	"imaxx-backend/internal/model"
	"imaxx-backend/internal/repository"
)

// translateNotFound maps the repository's gorm.ErrRecordNotFound into the domain
// sentinel ErrNotFound (which pkg/response maps to 404). Any other error passes
// through unchanged so genuine failures still surface as 500 without leaking.
func translateNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

// QuotationService orchestrates quotation business logic: validation, calc engine,
// reference-no generation, and repository calls.
type QuotationService struct {
	repo     repository.QuotationRepository
	userRepo repository.UserRepository
	clock    func() time.Time
}

// NewQuotationService creates a QuotationService with the given dependencies.
// clock is injected for deterministic reference-no generation in tests.
func NewQuotationService(
	repo repository.QuotationRepository,
	userRepo repository.UserRepository,
	clock func() time.Time,
) *QuotationService {
	return &QuotationService{
		repo:     repo,
		userRepo: userRepo,
		clock:    clock,
	}
}

// CreateQuotation validates input, computes totals via the calc engine,
// generates a unique reference number (with retry on conflict), snapshots
// the creator's signee info, and persists the quotation.
func (s *QuotationService) CreateQuotation(ctx context.Context, userID uint, req dto.CreateQuotationRequest) (*dto.QuotationResponse, error) {
	// 1. Parse dates.
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, ErrValidation
	}
	validUntil, err := time.Parse("2006-01-02", req.ValidUntil)
	if err != nil {
		return nil, ErrValidation
	}

	// 2. Validate validUntil >= date.
	if validUntil.Before(date) {
		return nil, ErrValidation
	}

	// 3. Compute line-item cents.
	lineItemCents := make([]int64, 0, len(req.Items))
	for _, item := range req.Items {
		lineItemCents = append(lineItemCents, calcLineTotalCents(item.UnitPrice, item.Qty))
	}

	// 4. Compute totals (validates discount).
	subtotalCents, _, vatCents, totalCents, err := calcTotals(lineItemCents, req.DiscountAmount)
	if err != nil {
		return nil, err
	}

	// 5. Snapshot creator's signee info.
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 5b. Parse optional customer signee date (validates format).
	customerSigneeDate, err := parseDatePtr(req.CustomerSigneeDate)
	if err != nil {
		return nil, err
	}

	// 6. Generate reference number with retry.
	prefix := "QT" + s.clock().Format("0601")
	var q *model.Quotation
	created := false
	for attempt := 0; attempt < 5; attempt++ {
		refNo, err := s.repo.NextReferenceNo(ctx, prefix)
		if err != nil {
			return nil, err
		}
		candidate := &model.Quotation{
			ReferenceNo:            refNo,
			Attention:              req.Attention,
			Company:                req.Company,
			Project:                req.Project,
			Telephone:              req.Telephone,
			Email:                  req.Email,
			Date:                   date,
			ValidUntil:             validUntil,
			DiscountAmount:         req.DiscountAmount,
			Subtotal:               float64(subtotalCents) / 100.0,
			VatAmount:              float64(vatCents) / 100.0,
			Total:                  float64(totalCents) / 100.0,
			CustomerSigneeName:     req.CustomerSigneeName,
			CustomerSigneePosition: req.CustomerSigneePosition,
			CustomerSigneeDate:     customerSigneeDate,
			CompanySigneeName:      user.FullName,
			CompanySigneePosition:  user.Position,
			CreatedBy:              userID,
			Status:                 "draft",
			Items:                  buildItems(req.Items),
		}
		err = s.repo.Create(ctx, candidate)
		if err == nil {
			q = candidate
			created = true
			break
		}
		if errors.Is(err, repository.ErrDuplicateReferenceNo) {
			continue
		}
		return nil, err
	}
	if !created {
		return nil, ErrConflict
	}

	return mapQuotationResponse(q), nil
}

// UpdateQuotation loads an existing quotation, validates draft-only and ownership,
// recomputes totals, and persists changes. Signee fields are carried over unchanged.
func (s *QuotationService) UpdateQuotation(ctx context.Context, userID uint, role string, id uint, req dto.UpdateQuotationRequest) (*dto.QuotationResponse, error) {
	// 1. Load existing.
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, translateNotFound(err)
	}

	// 2. Draft-only check.
	if existing.Status != "draft" {
		return nil, ErrForbidden
	}

	// 3. Ownership check.
	if role != "admin" && existing.CreatedBy != userID {
		return nil, ErrForbidden
	}

	// 4. Parse dates + validate.
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, ErrValidation
	}
	validUntil, err := time.Parse("2006-01-02", req.ValidUntil)
	if err != nil {
		return nil, ErrValidation
	}
	if validUntil.Before(date) {
		return nil, ErrValidation
	}

	// 5. Compute totals.
	lineItemCents := make([]int64, 0, len(req.Items))
	for _, item := range req.Items {
		lineItemCents = append(lineItemCents, calcLineTotalCents(item.UnitPrice, item.Qty))
	}
	subtotalCents, _, vatCents, totalCents, err := calcTotals(lineItemCents, req.DiscountAmount)
	if err != nil {
		return nil, err
	}

	// 5b. Parse optional customer signee date (validates format).
	customerSigneeDate, err := parseDatePtr(req.CustomerSigneeDate)
	if err != nil {
		return nil, err
	}

	// 6. Build updated quotation — signee fields carried from existing (never re-derived).
	q := &model.Quotation{
		ID:                     id,
		ReferenceNo:            existing.ReferenceNo,
		Attention:              req.Attention,
		Company:                req.Company,
		Project:                req.Project,
		Telephone:              req.Telephone,
		Email:                  req.Email,
		Date:                   date,
		ValidUntil:             validUntil,
		DiscountAmount:         req.DiscountAmount,
		Subtotal:               float64(subtotalCents) / 100.0,
		VatAmount:              float64(vatCents) / 100.0,
		Total:                  float64(totalCents) / 100.0,
		CustomerSigneeName:     req.CustomerSigneeName,
		CustomerSigneePosition: req.CustomerSigneePosition,
		CustomerSigneeDate:     customerSigneeDate,
		CompanySigneeName:      existing.CompanySigneeName,
		CompanySigneePosition:  existing.CompanySigneePosition,
		CreatedBy:              existing.CreatedBy,
		Status:                 existing.Status,
		Items:                  buildItems(req.Items),
	}

	if err := s.repo.Update(ctx, q); err != nil {
		return nil, err
	}

	q.ID = id
	return mapQuotationResponse(q), nil
}

// DeleteQuotation validates draft-only and ownership, then deletes.
func (s *QuotationService) DeleteQuotation(ctx context.Context, userID uint, role string, id uint) error {
	// 1. Load existing.
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateNotFound(err)
	}

	// 2. Draft-only check.
	if existing.Status != "draft" {
		return ErrForbidden
	}

	// 3. Ownership check.
	if role != "admin" && existing.CreatedBy != userID {
		return ErrForbidden
	}

	return s.repo.Delete(ctx, id)
}

// GetQuotation returns a single quotation by ID (no ownership check).
func (s *QuotationService) GetQuotation(ctx context.Context, id uint) (*dto.QuotationResponse, error) {
	q, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, translateNotFound(err)
	}
	return mapQuotationResponse(q), nil
}

// ListQuotations returns paginated quotations with optional filters.
func (s *QuotationService) ListQuotations(ctx context.Context, query dto.ListQuotationQuery) ([]dto.QuotationResponse, int64, error) {
	items, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	resp := make([]dto.QuotationResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, *mapQuotationResponse(&item))
	}
	return resp, total, nil
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

// buildItems converts DTO item inputs into model QuotationItems with line totals.
func buildItems(items []dto.QuotationItemInput) []model.QuotationItem {
	result := make([]model.QuotationItem, 0, len(items))
	for _, item := range items {
		lineTotalCents := calcLineTotalCents(item.UnitPrice, item.Qty)
		result = append(result, model.QuotationItem{
			ServiceType: item.ServiceType,
			Description: item.Description,
			UnitPrice:   item.UnitPrice,
			Qty:         item.Qty,
			LineTotal:   float64(lineTotalCents) / 100.0,
			SortOrder:   item.SortOrder,
		})
	}
	return result
}

// formatTime formats a time.Time as "2006-01-02".
func formatTime(t time.Time) string {
	return t.Format("2006-01-02")
}

// parseDatePtr parses an optional *string date ("2006-01-02") into *time.Time.
// A nil or empty input means "no date" (nil, nil); a non-empty but malformed
// value is a client error (ErrValidation) rather than being silently dropped.
func parseDatePtr(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, ErrValidation
	}
	return &t, nil
}

// timePtrToString converts *time.Time to *string.
func timePtrToString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := formatTime(*t)
	return &s
}

// mapQuotationResponse converts a model.Quotation to a dto.QuotationResponse.
// Cents are converted back to float64 (divided by 100.0).
func mapQuotationResponse(q *model.Quotation) *dto.QuotationResponse {
	resp := &dto.QuotationResponse{
		ID:                     q.ID,
		ReferenceNo:            q.ReferenceNo,
		Status:                 q.Status,
		Attention:              q.Attention,
		Company:                q.Company,
		Project:                q.Project,
		Telephone:              q.Telephone,
		Email:                  q.Email,
		Date:                   formatTime(q.Date),
		ValidUntil:             formatTime(q.ValidUntil),
		Subtotal:               q.Subtotal,
		DiscountAmount:         q.DiscountAmount,
		VatAmount:              q.VatAmount,
		Total:                  q.Total,
		CustomerSigneeName:     q.CustomerSigneeName,
		CustomerSigneePosition: q.CustomerSigneePosition,
		CustomerSigneeDate:     timePtrToString(q.CustomerSigneeDate),
		CompanySigneeName:      q.CompanySigneeName,
		CompanySigneePosition:  q.CompanySigneePosition,
		CreatedBy:              q.CreatedBy,
	}

	resp.Items = make([]dto.QuotationItemResponse, 0, len(q.Items))
	for _, item := range q.Items {
		resp.Items = append(resp.Items, dto.QuotationItemResponse{
			ServiceType: item.ServiceType,
			Description: item.Description,
			UnitPrice:   item.UnitPrice,
			Qty:         item.Qty,
			LineTotal:   item.LineTotal,
			SortOrder:   item.SortOrder,
		})
	}

	return resp
}
