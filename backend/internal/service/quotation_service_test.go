package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"imaxx-backend/internal/dto"
	"imaxx-backend/internal/model"
	"imaxx-backend/internal/repository"
)

// Contract required from dev (internal/service/quotation_service.go):
//
//	func NewQuotationService(repo repository.QuotationRepository, userRepo repository.UserRepository, clock func() time.Time) *QuotationService
//	func (s *QuotationService) CreateQuotation(ctx context.Context, userID uint, req dto.CreateQuotationRequest) (*dto.QuotationResponse, error)
//	func (s *QuotationService) UpdateQuotation(ctx context.Context, userID uint, role string, id uint, req dto.UpdateQuotationRequest) (*dto.QuotationResponse, error)
//	func (s *QuotationService) DeleteQuotation(ctx context.Context, userID uint, role string, id uint) error
//	func (s *QuotationService) GetQuotation(ctx context.Context, id uint) (*dto.QuotationResponse, error)
//	func (s *QuotationService) ListQuotations(ctx context.Context, query dto.ListQuotationQuery) ([]dto.QuotationResponse, int64, error)

// fixedClock returns a deterministic time: 2026-07-15 10:00:00 UTC.
// Used across all service tests to produce predictable reference_no values.
func fixedClock() time.Time {
	return time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
}

// sampleItemsAC1 returns the item set from AC1 used in many tests.
func sampleItemsAC1() []dto.QuotationItemInput {
	return []dto.QuotationItemInput{
		{ServiceType: "Design", Description: "Website design", UnitPrice: 1000.00, Qty: 2, SortOrder: 1},
		{ServiceType: "Development", Description: "Backend dev", UnitPrice: 250.50, Qty: 3, SortOrder: 2},
	}
}

func sampleCreateRequest() dto.CreateQuotationRequest {
	date := "2026-07-01"
	validUntil := "2026-07-31"
	return dto.CreateQuotationRequest{
		Attention:      "Mr. John",
		Company:        "Acme Corp",
		Email:          "john@acme.com",
		Date:           date,
		ValidUntil:     validUntil,
		DiscountAmount: 151.50,
		Items:          sampleItemsAC1(),
	}
}

func TestQuotationService_TC01_CreateHappyAC1AC5AC13(t *testing.T) {
	// Arrange — AC1+AC5+AC13: happy create with 2 items + discount
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	userRepo.On("FindByID", mock.Anything, uint(7)).Return(&model.User{
		ID: 7, FullName: "Somchai Dev", Position: "Sales Manager",
	}, nil)

	repo.On("NextReferenceNo", mock.Anything, "QT2607").Return("QT2607001", nil)
	repo.On("Create", mock.Anything, mock.MatchedBy(func(q *model.Quotation) bool {
		return q.CreatedBy == 7 && q.Status == "draft" && q.ReferenceNo == "QT2607001"
	})).Return(nil).Once()

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := sampleCreateRequest()

	// Act
	resp, err := svc.CreateQuotation(context.Background(), 7, req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "QT2607001", resp.ReferenceNo)
	assert.Equal(t, "draft", resp.Status)
	assert.Equal(t, uint(7), resp.CreatedBy)
	assert.Equal(t, "Somchai Dev", resp.CompanySigneeName)
	assert.Equal(t, "Sales Manager", resp.CompanySigneePosition)
	assert.InDelta(t, 2751.50, resp.Subtotal, 0.001)
	assert.InDelta(t, 151.50, resp.DiscountAmount, 0.001)
	assert.InDelta(t, 182.00, resp.VatAmount, 0.001)
	assert.InDelta(t, 2782.00, resp.Total, 0.001)
}

func TestQuotationService_TC02_CreateRetryConflictAC5(t *testing.T) {
	// Arrange — AC5 retry: NextReferenceNo returns sequential values; Create fails twice then succeeds
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	userRepo.On("FindByID", mock.Anything, uint(7)).Return(&model.User{
		ID: 7, FullName: "Somchai Dev", Position: "Sales Manager",
	}, nil)

	repo.On("NextReferenceNo", mock.Anything, "QT2607").Return("QT2607001", nil).Once()
	repo.On("NextReferenceNo", mock.Anything, "QT2607").Return("QT2607002", nil).Once()
	repo.On("NextReferenceNo", mock.Anything, "QT2607").Return("QT2607003", nil).Once()

	// repo.Create simulates the real repository contract: on a unique-index
	// violation it returns repository.ErrDuplicateReferenceNo (NOT
	// service.ErrConflict directly — repository never imports internal/service,
	// see backend/internal/repository/quotation_repository_test.go contract
	// comment). The service is expected to retry on this specific sentinel.
	repo.On("Create", mock.Anything, mock.Anything).Return(repository.ErrDuplicateReferenceNo).Once()
	repo.On("Create", mock.Anything, mock.Anything).Return(repository.ErrDuplicateReferenceNo).Once()
	repo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := sampleCreateRequest()

	// Act
	resp, err := svc.CreateQuotation(context.Background(), 7, req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "QT2607003", resp.ReferenceNo)
	repo.AssertNumberOfCalls(t, "Create", 3)
	repo.AssertNumberOfCalls(t, "NextReferenceNo", 3)
}

func TestQuotationService_TC03_CreateRetryExhaustedDecision3(t *testing.T) {
	// Arrange — Decision #3: all 5 Create attempts hit conflict
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	userRepo.On("FindByID", mock.Anything, uint(7)).Return(&model.User{
		ID: 7, FullName: "Somchai Dev", Position: "Sales Manager",
	}, nil)

	repo.On("NextReferenceNo", mock.Anything, "QT2607").Return("QT2607001", nil)
	// Every attempt hits the repository-level duplicate sentinel (see TC02 comment above).
	repo.On("Create", mock.Anything, mock.Anything).Return(repository.ErrDuplicateReferenceNo)

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := sampleCreateRequest()

	// Act
	_, err := svc.CreateQuotation(context.Background(), 7, req)

	// Assert — after exhausting retries the service translates the repository
	// signal into its own domain sentinel, service.ErrConflict (409 CONFLICT).
	require.Error(t, err)
	require.ErrorIs(t, err, ErrConflict)
	repo.AssertNumberOfCalls(t, "Create", 5)
}

func TestQuotationService_TC04_CreateTieVatAC2(t *testing.T) {
	// Arrange — AC2 full flow: base=10.50, discount=0 => vat=0.74, total=11.24
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	userRepo.On("FindByID", mock.Anything, uint(7)).Return(&model.User{
		ID: 7, FullName: "Somchai Dev", Position: "Sales Manager",
	}, nil)

	repo.On("NextReferenceNo", mock.Anything, "QT2607").Return("QT2607001", nil)
	repo.On("Create", mock.Anything, mock.MatchedBy(func(q *model.Quotation) bool {
		return q.ReferenceNo == "QT2607001"
	})).Return(nil).Once()

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := dto.CreateQuotationRequest{
		Attention:      "Mr. Test",
		Company:        "Test Co",
		Email:          "test@test.com",
		Date:           "2026-07-01",
		ValidUntil:     "2026-07-31",
		DiscountAmount: 0,
		Items: []dto.QuotationItemInput{
			{ServiceType: "Consulting", Description: "1 hour", UnitPrice: 10.50, Qty: 1, SortOrder: 1},
		},
	}

	// Act
	resp, err := svc.CreateQuotation(context.Background(), 7, req)

	// Assert
	require.NoError(t, err)
	assert.InDelta(t, 0.74, resp.VatAmount, 0.001)
	assert.InDelta(t, 11.24, resp.Total, 0.001)
}

func TestQuotationService_TC05_CreateDiscountExceedsSubtotalAC3(t *testing.T) {
	// Arrange — AC3: discount=3000 > subtotal=2751.50
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := sampleCreateRequest()
	req.DiscountAmount = 3000.00

	// Act
	_, err := svc.CreateQuotation(context.Background(), 7, req)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, ErrValidation)
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestQuotationService_TC06_CreateDiscountEqualsSubtotalBoundaryAC4(t *testing.T) {
	// Arrange — AC4: discount == subtotal exactly
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	userRepo.On("FindByID", mock.Anything, uint(7)).Return(&model.User{
		ID: 7, FullName: "Somchai Dev", Position: "Sales Manager",
	}, nil)

	repo.On("NextReferenceNo", mock.Anything, "QT2607").Return("QT2607001", nil)
	repo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := sampleCreateRequest()
	req.DiscountAmount = 2751.50 // equal to subtotal

	// Act
	resp, err := svc.CreateQuotation(context.Background(), 7, req)

	// Assert
	require.NoError(t, err)
	assert.InDelta(t, 0.00, resp.VatAmount, 0.001)
	assert.InDelta(t, 0.00, resp.Total, 0.001)
}

func TestQuotationService_TC07_CreateValidUntilBeforeDate(t *testing.T) {
	// Arrange — Decision #9: valid_until < date is invalid
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := sampleCreateRequest()
	req.Date = "2026-07-31"
	req.ValidUntil = "2026-07-01"

	// Act
	_, err := svc.CreateQuotation(context.Background(), 7, req)

	// Assert
	require.ErrorIs(t, err, ErrValidation)
}

func TestQuotationService_TC08_UpdateHappyAC6(t *testing.T) {
	// Arrange — AC6: update a draft quotation successfully
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	existing := &model.Quotation{
		ID: 42, Status: "draft", CreatedBy: 7,
		CompanySigneeName: "Somchai Dev", CompanySigneePosition: "Sales Manager",
	}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil)

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := dto.UpdateQuotationRequest(sampleCreateRequest())

	// Act
	resp, err := svc.UpdateQuotation(context.Background(), 7, "creator", 42, req)

	// Assert
	require.NoError(t, err)
	assert.InDelta(t, 2751.50, resp.Subtotal, 0.001)
	repo.AssertCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestQuotationService_TC09_UpdateForbiddenNonDraftAC7(t *testing.T) {
	// Arrange — AC7: update on non-draft status is forbidden
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	existing := &model.Quotation{ID: 42, Status: "sent", CreatedBy: 7}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := dto.UpdateQuotationRequest(sampleCreateRequest())

	// Act
	_, err := svc.UpdateQuotation(context.Background(), 7, "creator", 42, req)

	// Assert
	require.ErrorIs(t, err, ErrForbidden)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestQuotationService_TC10_UpdateForbiddenOwnershipAC8(t *testing.T) {
	// Arrange — AC8: creator cannot update another user's quotation
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	existing := &model.Quotation{ID: 42, Status: "draft", CreatedBy: 99}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := dto.UpdateQuotationRequest(sampleCreateRequest())

	// Act
	_, err := svc.UpdateQuotation(context.Background(), 7, "creator", 42, req)

	// Assert
	require.ErrorIs(t, err, ErrForbidden)
}

func TestQuotationService_TC11_UpdateAdminBypassAC8(t *testing.T) {
	// Arrange — AC8: admin bypasses ownership check
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	existing := &model.Quotation{ID: 42, Status: "draft", CreatedBy: 99}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil)

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := dto.UpdateQuotationRequest(sampleCreateRequest())

	// Act
	_, err := svc.UpdateQuotation(context.Background(), 7, "admin", 42, req)

	// Assert
	require.NoError(t, err)
}

func TestQuotationService_TC12_UpdateSigneeImmutableAC13(t *testing.T) {
	// Arrange — AC13: company_signee fields are NOT updated from caller's profile
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	existing := &model.Quotation{
		ID: 42, Status: "draft", CreatedBy: 7,
		CompanySigneeName: "Old Name", CompanySigneePosition: "Old Position",
	}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)

	repo.On("Update", mock.Anything, mock.Anything).Return(nil)

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := dto.UpdateQuotationRequest(sampleCreateRequest())

	// Act
	resp, err := svc.UpdateQuotation(context.Background(), 7, "creator", 42, req)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Old Name", resp.CompanySigneeName)
	assert.Equal(t, "Old Position", resp.CompanySigneePosition)
}

func TestQuotationService_TC13_DeleteHappyAC6(t *testing.T) {
	// Arrange — AC6: delete a draft quotation successfully
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	existing := &model.Quotation{ID: 42, Status: "draft", CreatedBy: 7}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	repo.On("Delete", mock.Anything, uint(42)).Return(nil)

	svc := NewQuotationService(repo, userRepo, fixedClock)

	// Act
	err := svc.DeleteQuotation(context.Background(), 7, "creator", 42)

	// Assert
	require.NoError(t, err)
	repo.AssertCalled(t, "Delete", mock.Anything, uint(42))
}

func TestQuotationService_TC14_DeleteForbiddenNonDraftAC7(t *testing.T) {
	// Arrange — AC7: delete on non-draft status is forbidden
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	existing := &model.Quotation{ID: 42, Status: "approved", CreatedBy: 7}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)

	svc := NewQuotationService(repo, userRepo, fixedClock)

	// Act
	err := svc.DeleteQuotation(context.Background(), 7, "creator", 42)

	// Assert
	require.ErrorIs(t, err, ErrForbidden)
	repo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestQuotationService_TC15_DeleteForbiddenOwnershipAC8(t *testing.T) {
	// Arrange — AC8: creator cannot delete another user's quotation
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	existing := &model.Quotation{ID: 42, Status: "draft", CreatedBy: 99}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)

	svc := NewQuotationService(repo, userRepo, fixedClock)

	// Act
	err := svc.DeleteQuotation(context.Background(), 7, "creator", 42)

	// Assert
	require.ErrorIs(t, err, ErrForbidden)
}

func TestQuotationService_TC16_DeleteAdminBypassAC8(t *testing.T) {
	// Arrange — AC8: admin bypasses ownership check on delete
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)

	existing := &model.Quotation{ID: 42, Status: "draft", CreatedBy: 99}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	repo.On("Delete", mock.Anything, uint(42)).Return(nil)

	svc := NewQuotationService(repo, userRepo, fixedClock)

	// Act
	err := svc.DeleteQuotation(context.Background(), 7, "admin", 42)

	// Assert
	require.NoError(t, err)
}

// ─── Not-found translation (AC12/AC19 + api-response.md): the service must map
// the repository's gorm.ErrRecordNotFound into the domain sentinel ErrNotFound
// so the handler returns 404 NOT_FOUND, not a leaked 500. Regression tests for
// the critical defect found in QA round 1. ────────────────────────────────────

func TestQuotationService_TC17_GetNotFoundTranslatesToErrNotFound(t *testing.T) {
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	repo.On("FindByID", mock.Anything, uint(999)).Return((*model.Quotation)(nil), gorm.ErrRecordNotFound)

	svc := NewQuotationService(repo, userRepo, fixedClock)

	_, err := svc.GetQuotation(context.Background(), 999)

	require.ErrorIs(t, err, ErrNotFound)
}

func TestQuotationService_TC18_UpdateNotFoundTranslatesToErrNotFound(t *testing.T) {
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	repo.On("FindByID", mock.Anything, uint(999)).Return((*model.Quotation)(nil), gorm.ErrRecordNotFound)

	svc := NewQuotationService(repo, userRepo, fixedClock)
	req := dto.UpdateQuotationRequest(sampleCreateRequest())

	_, err := svc.UpdateQuotation(context.Background(), 7, "creator", 999, req)

	require.ErrorIs(t, err, ErrNotFound)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestQuotationService_TC19_DeleteNotFoundTranslatesToErrNotFound(t *testing.T) {
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	repo.On("FindByID", mock.Anything, uint(999)).Return((*model.Quotation)(nil), gorm.ErrRecordNotFound)

	svc := NewQuotationService(repo, userRepo, fixedClock)

	err := svc.DeleteQuotation(context.Background(), 7, "creator", 999)

	require.ErrorIs(t, err, ErrNotFound)
	repo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}
