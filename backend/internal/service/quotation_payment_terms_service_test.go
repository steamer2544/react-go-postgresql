package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"imaxx-backend/internal/dto"
	"imaxx-backend/internal/model"
)

// Contract required from dev (internal/service/quotation_service.go):
//
//	func NewQuotationService(repo repository.QuotationRepository, userRepo repository.UserRepository, clock func() time.Time) *QuotationService
//	func (s *QuotationService) CreateQuotation(ctx context.Context, userID uint, req dto.CreateQuotationRequest) (*dto.QuotationResponse, error)
//	func (s *QuotationService) UpdateQuotation(ctx context.Context, userID uint, role string, id uint, req dto.UpdateQuotationRequest) (*dto.QuotationResponse, error)

// samplePaymentTermsItems returns the 2-item base fixture from the testcases.md plan:
// unit_price=1000 qty=2 (line 2000) + unit_price=500 qty=1 (line 500) => subtotal=2500, vat=175, total=2675.
func samplePaymentTermsItems() []dto.QuotationItemInput {
	return []dto.QuotationItemInput{
		{ServiceType: "Design", Description: "Website design", UnitPrice: 1000.00, Qty: 2, SortOrder: 1},
		{ServiceType: "Development", Description: "Backend dev", UnitPrice: 500.00, Qty: 1, SortOrder: 2},
	}
}

// samplePaymentTermsRequest builds a CreateQuotationRequest with the base 2 items and
// payment terms [891.67, 891.67, 891.66] (Deposit/Progress/Final).
func samplePaymentTermsRequest() dto.CreateQuotationRequest {
	return dto.CreateQuotationRequest{
		Attention:      "Mr. Somchai",
		Company:        "Acme Corp",
		Email:          "somchai@acme.com",
		Date:           "2026-07-01",
		ValidUntil:     "2026-07-31",
		DiscountAmount: 0,
		Items:          samplePaymentTermsItems(),
		PaymentTerms: []dto.PaymentTermInput{
			{Description: "Deposit", Amount: 891.67},
			{Description: "Progress", Amount: 891.67},
			{Description: "Final", Amount: 891.66},
		},
	}
}

func TestPaymentTermsService_TC_SVC_PT01_HappyAC1(t *testing.T) {
	// Arrange — AC1: Create with 2 base items + 3 payment terms [891.67, 891.67, 891.66]
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	userRepo.On("FindByID", mock.Anything, uint(7)).Return(&model.User{
		ID: 7, FullName: "Somchai Dev", Position: "Sales Manager",
	}, nil)

	repo.On("NextReferenceNo", mock.Anything, "QT2607").Return("QT2607001", nil)
	repo.On("Create", mock.Anything, mock.MatchedBy(func(q *model.Quotation) bool {
		// Verify PaymentTerms were set on the model before repo.Create
		if len(q.PaymentTerms) != 3 {
			return false
		}
		if q.PaymentTerms[0].TermNo != 1 || q.PaymentTerms[0].Description != "Deposit" {
			return false
		}
		if q.PaymentTerms[1].TermNo != 2 || q.PaymentTerms[1].Description != "Progress" {
			return false
		}
		if q.PaymentTerms[2].TermNo != 3 || q.PaymentTerms[2].Description != "Final" {
			return false
		}
		return true
	})).Return(nil).Once()

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := samplePaymentTermsRequest()

	// Act
	resp, err := svc.CreateQuotation(context.Background(), 7, req)

	// Assert
	require.NoError(t, err)
	assert.Len(t, resp.PaymentTerms, 3)
	assert.Equal(t, 1, resp.PaymentTerms[0].TermNo)
	assert.Equal(t, "Deposit", resp.PaymentTerms[0].Description)
	assert.Equal(t, 891.67, resp.PaymentTerms[0].Amount)
	assert.Equal(t, 1, resp.PaymentTerms[0].SortOrder)

	assert.Equal(t, 2, resp.PaymentTerms[1].TermNo)
	assert.Equal(t, "Progress", resp.PaymentTerms[1].Description)
	assert.Equal(t, 891.67, resp.PaymentTerms[1].Amount)
	assert.Equal(t, 2, resp.PaymentTerms[1].SortOrder)

	assert.Equal(t, 3, resp.PaymentTerms[2].TermNo)
	assert.Equal(t, "Final", resp.PaymentTerms[2].Description)
	assert.Equal(t, 891.66, resp.PaymentTerms[2].Amount)
	assert.Equal(t, 3, resp.PaymentTerms[2].SortOrder)
}

func TestPaymentTermsService_TC_SVC_PT02_MismatchAC3(t *testing.T) {
	// Arrange — AC3: Create with mismatched terms [1000, 1000, 1000] sum=3000 != total=2675
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := dto.CreateQuotationRequest{
		Attention:      "Mr. Somchai",
		Company:        "Acme Corp",
		Email:          "somchai@acme.com",
		Date:           "2026-07-01",
		ValidUntil:     "2026-07-31",
		DiscountAmount: 0,
		Items:          samplePaymentTermsItems(),
		PaymentTerms: []dto.PaymentTermInput{
			{Description: "Term1", Amount: 1000.00},
			{Description: "Term2", Amount: 1000.00},
			{Description: "Term3", Amount: 1000.00},
		},
	}

	// Act
	_, err := svc.CreateQuotation(context.Background(), 7, req)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, ErrValidation)
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestPaymentTermsService_TC_SVC_PT03_NilTermsAC5(t *testing.T) {
	// Arrange — AC5: Create without PaymentTerms (nil) => should pass normally
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	userRepo.On("FindByID", mock.Anything, uint(7)).Return(&model.User{
		ID: 7, FullName: "Somchai Dev", Position: "Sales Manager",
	}, nil)

	repo.On("NextReferenceNo", mock.Anything, "QT2607").Return("QT2607001", nil)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := dto.CreateQuotationRequest{
		Attention:      "Mr. Somchai",
		Company:        "Acme Corp",
		Email:          "somchai@acme.com",
		Date:           "2026-07-01",
		ValidUntil:     "2026-07-31",
		DiscountAmount: 0,
		Items:          samplePaymentTermsItems(),
		// PaymentTerms not set (nil)
	}

	// Act
	resp, err := svc.CreateQuotation(context.Background(), 7, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, resp.PaymentTerms)
	assert.Len(t, resp.PaymentTerms, 0)
}

func TestPaymentTermsService_TC_SVC_PT04_UpdateForbiddenNonDraftAC6(t *testing.T) {
	// Arrange — AC6: Update on non-draft status is forbidden
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	existing := &model.Quotation{
		ID: 42, Status: "sent", CreatedBy: 7,
		CompanySigneeName: "Somchai Dev", CompanySigneePosition: "Sales Manager",
	}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := dto.UpdateQuotationRequest{
		Attention:      "Mr. Somchai",
		Company:        "Acme Corp",
		Email:          "somchai@acme.com",
		Date:           "2026-07-01",
		ValidUntil:     "2026-07-31",
		DiscountAmount: 0,
		Items:          samplePaymentTermsItems(),
		PaymentTerms: []dto.PaymentTermInput{
			{Description: "Deposit", Amount: 891.67},
			{Description: "Progress", Amount: 891.67},
			{Description: "Final", Amount: 891.66},
		},
	}

	// Act
	_, err := svc.UpdateQuotation(context.Background(), 7, "creator", 42, req)

	// Assert
	require.ErrorIs(t, err, ErrForbidden)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestPaymentTermsService_TC_SVC_PT05_UpdateHappyAC1(t *testing.T) {
	// Arrange — AC1: Update draft quotation with valid payment terms
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	existing := &model.Quotation{
		ID: 42, Status: "draft", CreatedBy: 7,
		CompanySigneeName: "Somchai Dev", CompanySigneePosition: "Sales Manager",
	}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(q *model.Quotation) bool {
		if len(q.PaymentTerms) != 3 {
			return false
		}
		if q.PaymentTerms[0].TermNo != 1 || q.PaymentTerms[0].Description != "Deposit" {
			return false
		}
		if q.PaymentTerms[1].TermNo != 2 || q.PaymentTerms[1].Description != "Progress" {
			return false
		}
		if q.PaymentTerms[2].TermNo != 3 || q.PaymentTerms[2].Description != "Final" {
			return false
		}
		return true
	})).Return(nil)

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := dto.UpdateQuotationRequest{
		Attention:      "Mr. Somchai",
		Company:        "Acme Corp",
		Email:          "somchai@acme.com",
		Date:           "2026-07-01",
		ValidUntil:     "2026-07-31",
		DiscountAmount: 0,
		Items:          samplePaymentTermsItems(),
		PaymentTerms: []dto.PaymentTermInput{
			{Description: "Deposit", Amount: 891.67},
			{Description: "Progress", Amount: 891.67},
			{Description: "Final", Amount: 891.66},
		},
	}

	// Act
	resp, err := svc.UpdateQuotation(context.Background(), 7, "creator", 42, req)

	// Assert
	require.NoError(t, err)
	assert.Len(t, resp.PaymentTerms, 3)
	assert.Equal(t, 1, resp.PaymentTerms[0].TermNo)
	assert.Equal(t, "Deposit", resp.PaymentTerms[0].Description)
	assert.Equal(t, 891.67, resp.PaymentTerms[0].Amount)
	assert.Equal(t, 1, resp.PaymentTerms[0].SortOrder)

	assert.Equal(t, 2, resp.PaymentTerms[1].TermNo)
	assert.Equal(t, "Progress", resp.PaymentTerms[1].Description)
	assert.Equal(t, 891.67, resp.PaymentTerms[1].Amount)
	assert.Equal(t, 2, resp.PaymentTerms[1].SortOrder)

	assert.Equal(t, 3, resp.PaymentTerms[2].TermNo)
	assert.Equal(t, "Final", resp.PaymentTerms[2].Description)
	assert.Equal(t, 891.66, resp.PaymentTerms[2].Amount)
	assert.Equal(t, 3, resp.PaymentTerms[2].SortOrder)
}

func TestPaymentTermsService_TC_SVC_PT06_UpdateMismatchAC3(t *testing.T) {
	// Arrange — AC3: Update draft with mismatched payment terms
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	existing := &model.Quotation{
		ID: 42, Status: "draft", CreatedBy: 7,
		CompanySigneeName: "Somchai Dev", CompanySigneePosition: "Sales Manager",
	}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := dto.UpdateQuotationRequest{
		Attention:      "Mr. Somchai",
		Company:        "Acme Corp",
		Email:          "somchai@acme.com",
		Date:           "2026-07-01",
		ValidUntil:     "2026-07-31",
		DiscountAmount: 0,
		Items:          samplePaymentTermsItems(),
		PaymentTerms: []dto.PaymentTermInput{
			{Description: "Term1", Amount: 1000.00},
			{Description: "Term2", Amount: 1000.00},
			{Description: "Term3", Amount: 1000.00},
		},
	}

	// Act
	_, err := svc.UpdateQuotation(context.Background(), 7, "creator", 42, req)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, ErrValidation)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}
