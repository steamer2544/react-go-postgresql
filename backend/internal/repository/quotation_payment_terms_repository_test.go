package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"imaxx-backend/internal/model"
)

// Contract required from dev (internal/repository/quotation_repository.go):
//
//	func (r *gormQuotationRepository) Create(ctx context.Context, q *model.Quotation) error
//	func (r *gormQuotationRepository) FindByID(ctx context.Context, id uint) (*model.Quotation, error)
//	func (r *gormQuotationRepository) Update(ctx context.Context, q *model.Quotation) error
//	func (r *gormQuotationRepository) Delete(ctx context.Context, id uint) error
//	func (r *gormQuotationRepository) NextReferenceNo(ctx context.Context, prefix string) (string, error)

func TestPaymentTermsRepo_TC_REPO_PT01_CreateWithTerms(t *testing.T) {
	// Arrange — AC1/Decision#7: Create quotation with 3 payment terms (term_no/sort_order 1,2,3)
	ctx := context.Background()
	tx := setupTx(t)
	repo := NewQuotationRepository(tx)
	userID := seedUser(t, tx)

	now := time.Now()
	q := &model.Quotation{
		ReferenceNo:    "QT2607PT1",
		Status:         "draft",
		Attention:      "Test",
		Company:        "Test Co",
		Email:          "testpt1@test.com",
		Date:           now,
		ValidUntil:     now.AddDate(0, 1, 0),
		CreatedBy:      userID,
		DiscountAmount: 0,
		Subtotal:       2500.00,
		VatAmount:      175.00,
		Total:          2675.00,
		PaymentTerms: []model.PaymentTerm{
			{TermNo: 1, Description: "Deposit", Amount: 891.67, SortOrder: 1},
			{TermNo: 2, Description: "Progress", Amount: 891.67, SortOrder: 2},
			{TermNo: 3, Description: "Final", Amount: 891.66, SortOrder: 3},
		},
		Items: []model.QuotationItem{
			{ServiceType: "Design", Description: "Website design", UnitPrice: 1000.00, Qty: 2, LineTotal: 2000.00, SortOrder: 1},
			{ServiceType: "Development", Description: "Backend dev", UnitPrice: 500.00, Qty: 1, LineTotal: 500.00, SortOrder: 2},
		},
	}

	// Act
	require.NoError(t, repo.Create(ctx, q))
	got, err := repo.FindByID(ctx, q.ID)

	// Assert
	require.NoError(t, err)
	require.Len(t, got.PaymentTerms, 3)
	require.Equal(t, "Deposit", got.PaymentTerms[0].Description)
	require.Equal(t, "Progress", got.PaymentTerms[1].Description)
	require.Equal(t, "Final", got.PaymentTerms[2].Description)
}

func TestPaymentTermsRepo_TC_REPO_PT02_FullReplaceUpdate(t *testing.T) {
	// Arrange — Decision#6: Create with 2 payment terms, then Update with 1 payment term
	// Old terms should be fully replaced (2 -> 1)
	ctx := context.Background()
	tx := setupTx(t)
	repo := NewQuotationRepository(tx)
	userID := seedUser(t, tx)

	now := time.Now()
	q := &model.Quotation{
		ReferenceNo:    "QT2607PT2",
		Status:         "draft",
		Attention:      "Test",
		Company:        "Test Co",
		Email:          "testpt2@test.com",
		Date:           now,
		ValidUntil:     now.AddDate(0, 1, 0),
		CreatedBy:      userID,
		DiscountAmount: 0,
		Subtotal:       2500.00,
		VatAmount:      175.00,
		Total:          2675.00,
		PaymentTerms: []model.PaymentTerm{
			{TermNo: 1, Description: "Deposit", Amount: 1337.50, SortOrder: 1},
			{TermNo: 2, Description: "Final", Amount: 1337.50, SortOrder: 2},
		},
		Items: []model.QuotationItem{
			{ServiceType: "Design", Description: "Website design", UnitPrice: 1000.00, Qty: 2, LineTotal: 2000.00, SortOrder: 1},
			{ServiceType: "Development", Description: "Backend dev", UnitPrice: 500.00, Qty: 1, LineTotal: 500.00, SortOrder: 2},
		},
	}

	require.NoError(t, repo.Create(ctx, q))

	// Act — replace payment terms with a single term
	q.PaymentTerms = []model.PaymentTerm{
		{TermNo: 1, Description: "OnlyOne", Amount: 999.99, SortOrder: 1},
	}
	require.NoError(t, repo.Update(ctx, q))

	got, err := repo.FindByID(ctx, q.ID)

	// Assert — should have exactly 1 payment term (old 2 removed)
	require.NoError(t, err)
	require.Len(t, got.PaymentTerms, 1)
	require.Equal(t, "OnlyOne", got.PaymentTerms[0].Description)
}

func TestPaymentTermsRepo_TC_REPO_PT03_DeleteCascadesTerms(t *testing.T) {
	// Arrange — Decision#8/cascade: Create quotation with payment terms, then delete
	ctx := context.Background()
	tx := setupTx(t)
	repo := NewQuotationRepository(tx)
	userID := seedUser(t, tx)

	now := time.Now()
	q := &model.Quotation{
		ReferenceNo:    "QT2607PT3",
		Status:         "draft",
		Attention:      "Test",
		Company:        "Test Co",
		Email:          "testpt3@test.com",
		Date:           now,
		ValidUntil:     now.AddDate(0, 1, 0),
		CreatedBy:      userID,
		DiscountAmount: 0,
		Subtotal:       2500.00,
		VatAmount:      175.00,
		Total:          2675.00,
		PaymentTerms: []model.PaymentTerm{
			{TermNo: 1, Description: "Deposit", Amount: 891.67, SortOrder: 1},
			{TermNo: 2, Description: "Progress", Amount: 891.67, SortOrder: 2},
			{TermNo: 3, Description: "Final", Amount: 891.66, SortOrder: 3},
		},
		Items: []model.QuotationItem{
			{ServiceType: "Design", Description: "Website design", UnitPrice: 1000.00, Qty: 2, LineTotal: 2000.00, SortOrder: 1},
			{ServiceType: "Development", Description: "Backend dev", UnitPrice: 500.00, Qty: 1, LineTotal: 500.00, SortOrder: 2},
		},
	}

	require.NoError(t, repo.Create(ctx, q))

	// Act
	require.NoError(t, repo.Delete(ctx, q.ID))

	// Assert — payment_terms count for this quotation_id must be 0 (CASCADE)
	var cnt int64
	tx.Model(&model.PaymentTerm{}).Where("quotation_id = ?", q.ID).Count(&cnt)
	require.Equal(t, int64(0), cnt)
}
