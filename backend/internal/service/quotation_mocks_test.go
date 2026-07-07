package service

import (
	"context"

	"github.com/stretchr/testify/mock"

	"imaxx-backend/internal/dto"
	"imaxx-backend/internal/model"
	"imaxx-backend/internal/repository"
)

// mockQuotationRepository is a hand-written testify mock implementing
// repository.QuotationRepository. It keeps the quotation service unit tests
// independent from the real GORM repository.
type mockQuotationRepository struct {
	mock.Mock
}

// Compile-time contract check: dev must define repository.QuotationRepository
// with exactly this method set for QuotationService to depend on.
var _ repository.QuotationRepository = (*mockQuotationRepository)(nil)

func (m *mockQuotationRepository) Create(ctx context.Context, q *model.Quotation) error {
	args := m.Called(ctx, q)
	return args.Error(0)
}

func (m *mockQuotationRepository) FindByID(ctx context.Context, id uint) (*model.Quotation, error) {
	args := m.Called(ctx, id)
	q, _ := args.Get(0).(*model.Quotation)
	return q, args.Error(1)
}

func (m *mockQuotationRepository) Update(ctx context.Context, q *model.Quotation) error {
	args := m.Called(ctx, q)
	return args.Error(0)
}

func (m *mockQuotationRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockQuotationRepository) List(ctx context.Context, query dto.ListQuotationQuery) ([]model.Quotation, int64, error) {
	args := m.Called(ctx, query)
	result, _ := args.Get(0).([]model.Quotation)
	return result, args.Get(1).(int64), args.Error(2)
}

func (m *mockQuotationRepository) NextReferenceNo(ctx context.Context, prefix string) (string, error) {
	args := m.Called(ctx, prefix)
	return args.String(0), args.Error(1)
}
