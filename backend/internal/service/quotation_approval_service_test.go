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

// Contract required from dev (internal/service/quotation_service.go), on top
// of the existing QuotationService:
//
//	func (s *QuotationService) SubmitQuotation(ctx context.Context, userID uint, role string, id uint) (*dto.QuotationResponse, error)
//	func (s *QuotationService) ApproveQuotation(ctx context.Context, userID uint, role string, id uint) (*dto.QuotationResponse, error)
//	func (s *QuotationService) RejectQuotation(ctx context.Context, userID uint, role string, id uint) (*dto.QuotationResponse, error)
//	func (s *QuotationService) GetApprovalSignaturePath(ctx context.Context, id uint) (path string, contentType string, err error)
//
// internal/repository/quotation_repository.go must add:
//	var ErrStatusConflict = errors.New("status conflict")
//	TransitionStatus(ctx context.Context, id uint, fromStatus string, updates map[string]any) error
//
// internal/dto/quotation_dto.go QuotationResponse must add:
//	ApproverID             *uint   `json:"approver_id"`
//	ApprovedAt             *string `json:"approved_at"`  // RFC3339, e.g. "2026-07-15T10:00:00Z"
//	ApprovedSigneeName     *string `json:"approved_signee_name"`
//	ApprovedSigneePosition *string `json:"approved_signee_position"`
//	HasApprovedSignature   bool    `json:"has_approved_signature"`
//
// This file adds a TransitionStatus method to mockQuotationRepository (struct
// declared in quotation_mocks_test.go — do not duplicate the struct here).

// approvalFixedClock is a deterministic clock (same wall-clock value as
// fixedClock() in quotation_mocks_test.go, declared separately here for
// this file's self-containment).
func approvalFixedClock() time.Time {
	return time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
}

func (m *mockQuotationRepository) TransitionStatus(ctx context.Context, id uint, fromStatus string, updates map[string]any) error {
	args := m.Called(ctx, id, fromStatus, updates)
	return args.Error(0)
}

// ─── AC1: Submit happy path ──────────────────────────────────────────────

func TestQuotationService_TC_SVC_A01_SubmitHappyAC1(t *testing.T) {
	// Arrange
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	existing := &model.Quotation{ID: 42, Status: "draft", CreatedBy: 7}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	repo.On("TransitionStatus", mock.Anything, uint(42), "draft", map[string]any{"status": "pending_approval"}).Return(nil)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	resp, err := svc.SubmitQuotation(context.Background(), 7, "creator", 42)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "pending_approval", resp.Status)
}

// ─── AC2: Submit forbidden — non-owner creator ───────────────────────────

func TestQuotationService_TC_SVC_A02_SubmitForbiddenNonOwnerAC2(t *testing.T) {
	// Arrange
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	existing := &model.Quotation{ID: 42, Status: "draft", CreatedBy: 99}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	_, err := svc.SubmitQuotation(context.Background(), 7, "creator", 42)

	// Assert
	require.ErrorIs(t, err, ErrForbidden)
	repo.AssertNotCalled(t, "TransitionStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// ─── Submit admin bypass ownership (Decision 3) ──────────────────────────

func TestQuotationService_TC_SVC_A03_SubmitAdminBypassesOwnershipDecision3(t *testing.T) {
	// Arrange
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	existing := &model.Quotation{ID: 42, Status: "draft", CreatedBy: 99}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	repo.On("TransitionStatus", mock.Anything, uint(42), "draft", map[string]any{"status": "pending_approval"}).Return(nil)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	resp, err := svc.SubmitQuotation(context.Background(), 7, "admin", 42)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "pending_approval", resp.Status)
}

// ─── AC2b: Submit conflict — wrong status ────────────────────────────────

func TestQuotationService_TC_SVC_A04_SubmitConflictWrongStatusAC2b(t *testing.T) {
	// Arrange
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	existing := &model.Quotation{ID: 42, Status: "pending_approval", CreatedBy: 7}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	_, err := svc.SubmitQuotation(context.Background(), 7, "creator", 42)

	// Assert
	require.ErrorIs(t, err, ErrConflict)
	repo.AssertNotCalled(t, "TransitionStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// ─── Race condition: repository-level ErrStatusConflict translates to ErrConflict ──

func TestQuotationService_TC_SVC_A05_SubmitRaceConditionTranslatesToErrConflict(t *testing.T) {
	// Arrange — service's stale read still says "draft" but another request
	// already transitioned the row; the atomic UPDATE affects 0 rows.
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	existing := &model.Quotation{ID: 42, Status: "draft", CreatedBy: 7}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	repo.On("TransitionStatus", mock.Anything, uint(42), "draft", map[string]any{"status": "pending_approval"}).
		Return(repository.ErrStatusConflict)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	_, err := svc.SubmitQuotation(context.Background(), 7, "creator", 42)

	// Assert
	require.ErrorIs(t, err, ErrConflict)
}

// ─── AC3: Approve forbidden — non-approver roles ─────────────────────────

func TestQuotationService_TC_SVC_A06_ApproveForbiddenCreatorAC3(t *testing.T) {
	// Arrange
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	_, err := svc.ApproveQuotation(context.Background(), 7, "creator", 42)

	// Assert
	require.ErrorIs(t, err, ErrForbidden)
	repo.AssertNotCalled(t, "FindByID", mock.Anything, mock.Anything)
}

func TestQuotationService_TC_SVC_A07_ApproveForbiddenAdminNoBypassAC3(t *testing.T) {
	// Arrange — Decision 3: admin does NOT bypass on approve/reject (unlike submit).
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	_, err := svc.ApproveQuotation(context.Background(), 7, "admin", 42)

	// Assert
	require.ErrorIs(t, err, ErrForbidden)
	repo.AssertNotCalled(t, "FindByID", mock.Anything, mock.Anything)
}

// ─── AC4: Approve happy — snapshot stamp ─────────────────────────────────

func TestQuotationService_TC_SVC_A08_ApproveHappySnapshotAC4(t *testing.T) {
	// Arrange
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	existing := &model.Quotation{ID: 42, Status: "pending_approval", CreatedBy: 7}
	sigPath := "/uploads/signatures/user_9.png"
	approver := &model.User{ID: 9, FullName: "Approver Name", Position: "CFO", SignatureImagePath: &sigPath}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	userRepo.On("FindByID", mock.Anything, uint(9)).Return(approver, nil)
	repo.On("TransitionStatus", mock.Anything, uint(42), "pending_approval", mock.MatchedBy(func(u map[string]any) bool {
		return u["status"] == "approved" &&
			u["approver_id"] == uint(9) &&
			u["approved_signee_name"] == "Approver Name" &&
			u["approved_signee_position"] == "CFO" &&
			u["approved_signature_path"] == sigPath &&
			u["approved_at"] == approvalFixedClock()
	})).Return(nil)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	resp, err := svc.ApproveQuotation(context.Background(), 9, "approver", 42)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "approved", resp.Status)
	require.NotNil(t, resp.ApproverID)
	assert.Equal(t, uint(9), *resp.ApproverID)
	require.NotNil(t, resp.ApprovedAt)
	assert.Equal(t, "2026-07-15T10:00:00Z", *resp.ApprovedAt)
	require.NotNil(t, resp.ApprovedSigneeName)
	assert.Equal(t, "Approver Name", *resp.ApprovedSigneeName)
	require.NotNil(t, resp.ApprovedSigneePosition)
	assert.Equal(t, "CFO", *resp.ApprovedSigneePosition)
	assert.True(t, resp.HasApprovedSignature)
}

// ─── AC4 snapshot semantics: GetQuotation must NOT re-lookup the approver ──

func TestQuotationService_TC_SVC_A09_GetQuotationUsesSnapshotNotLiveLookupAC4(t *testing.T) {
	// Arrange — stored snapshot values differ from what a "live" lookup would
	// return (there is no userRepo stub at all: if the service performed a
	// live lookup it would panic on an unexpected mock.Called).
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	approverID := uint(9)
	snapName := "Old Name At Approval Time"
	snapPos := "Old Position At Approval Time"
	sigPath := "/uploads/signatures/user_9.png"
	existing := &model.Quotation{
		ID: 42, Status: "approved", CreatedBy: 7,
		ApproverID: &approverID, ApprovedSigneeName: &snapName,
		ApprovedSigneePosition: &snapPos, ApprovedSignaturePath: &sigPath,
	}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	resp, err := svc.GetQuotation(context.Background(), 42)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, resp.ApprovedSigneeName)
	assert.Equal(t, snapName, *resp.ApprovedSigneeName)
	require.NotNil(t, resp.ApprovedSigneePosition)
	assert.Equal(t, snapPos, *resp.ApprovedSigneePosition)
	assert.True(t, resp.HasApprovedSignature)
	userRepo.AssertNotCalled(t, "FindByID", mock.Anything, mock.Anything)
}

// ─── AC5: Approve validation — approver has no signature uploaded ────────

func TestQuotationService_TC_SVC_A10_ApproveValidationErrorNoSignatureAC5(t *testing.T) {
	// Arrange
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	existing := &model.Quotation{ID: 42, Status: "pending_approval", CreatedBy: 7}
	approver := &model.User{ID: 9, FullName: "Approver Name", Position: "CFO", SignatureImagePath: nil}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	userRepo.On("FindByID", mock.Anything, uint(9)).Return(approver, nil)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	_, err := svc.ApproveQuotation(context.Background(), 9, "approver", 42)

	// Assert
	require.ErrorIs(t, err, ErrValidation)
	repo.AssertNotCalled(t, "TransitionStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// ─── AC5b: Approve/Reject conflict — wrong status ────────────────────────

func TestQuotationService_TC_SVC_A11_ApproveConflictWrongStatusAC5b(t *testing.T) {
	// Arrange
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	existing := &model.Quotation{ID: 42, Status: "draft", CreatedBy: 7}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	_, err := svc.ApproveQuotation(context.Background(), 9, "approver", 42)

	// Assert
	require.ErrorIs(t, err, ErrConflict)
	userRepo.AssertNotCalled(t, "FindByID", mock.Anything, mock.Anything)
}

func TestQuotationService_TC_SVC_A12_RejectConflictWrongStatusAC5b(t *testing.T) {
	// Arrange
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	existing := &model.Quotation{ID: 42, Status: "approved", CreatedBy: 7}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	_, err := svc.RejectQuotation(context.Background(), 9, "approver", 42)

	// Assert
	require.ErrorIs(t, err, ErrConflict)
	repo.AssertNotCalled(t, "TransitionStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// ─── AC6: Reject happy path ───────────────────────────────────────────────

func TestQuotationService_TC_SVC_A13_RejectHappyAC6(t *testing.T) {
	// Arrange
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	existing := &model.Quotation{ID: 42, Status: "pending_approval", CreatedBy: 7}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	repo.On("TransitionStatus", mock.Anything, uint(42), "pending_approval", map[string]any{"status": "rejected"}).Return(nil)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	resp, err := svc.RejectQuotation(context.Background(), 9, "approver", 42)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "rejected", resp.Status)
}

// ─── AC3: Reject forbidden — non-approver ────────────────────────────────

func TestQuotationService_TC_SVC_A14_RejectForbiddenCreatorAC3(t *testing.T) {
	// Arrange
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	_, err := svc.RejectQuotation(context.Background(), 7, "creator", 42)

	// Assert
	require.ErrorIs(t, err, ErrForbidden)
	repo.AssertNotCalled(t, "FindByID", mock.Anything, mock.Anything)
}

// ─── AC7 (regression): edit/delete guard still blocks the new enum values ─

func TestQuotationService_TC_SVC_A15_UpdateForbiddenPendingApprovalStatusAC7(t *testing.T) {
	// Arrange
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	existing := &model.Quotation{ID: 42, Status: "pending_approval", CreatedBy: 7}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)
	req := dto.UpdateQuotationRequest(sampleCreateRequest())

	// Act
	_, err := svc.UpdateQuotation(context.Background(), 7, "creator", 42, req)

	// Assert
	require.ErrorIs(t, err, ErrForbidden)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestQuotationService_TC_SVC_A16_DeleteForbiddenRejectedStatusAC7(t *testing.T) {
	// Arrange
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	existing := &model.Quotation{ID: 42, Status: "rejected", CreatedBy: 7}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	err := svc.DeleteQuotation(context.Background(), 7, "creator", 42)

	// Assert
	require.ErrorIs(t, err, ErrForbidden)
	repo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

// ─── AC8/AC8b: GetApprovalSignaturePath ──────────────────────────────────

func TestQuotationService_TC_SVC_A17_GetApprovalSignaturePathHappyAC8(t *testing.T) {
	// Arrange
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	sigPath := "/uploads/signatures/user_9.png"
	existing := &model.Quotation{ID: 42, Status: "approved", ApprovedSignaturePath: &sigPath}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	path, contentType, err := svc.GetApprovalSignaturePath(context.Background(), 42)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, sigPath, path)
	assert.Equal(t, "image/png", contentType)
}

func TestQuotationService_TC_SVC_A18_GetApprovalSignaturePathNotApprovedAC8b(t *testing.T) {
	// Arrange
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	existing := &model.Quotation{ID: 42, Status: "pending_approval"}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	_, _, err := svc.GetApprovalSignaturePath(context.Background(), 42)

	// Assert
	require.ErrorIs(t, err, ErrNotFound)
}

func TestQuotationService_TC_SVC_A19_GetApprovalSignaturePathApprovedButNoPathAC8b(t *testing.T) {
	// Arrange — edge: status says approved but snapshot path is nil (should not happen
	// in practice since AC5 blocks it, but the accessor must stay defensive).
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	existing := &model.Quotation{ID: 42, Status: "approved", ApprovedSignaturePath: nil}
	repo.On("FindByID", mock.Anything, uint(42)).Return(existing, nil)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	_, _, err := svc.GetApprovalSignaturePath(context.Background(), 42)

	// Assert
	require.ErrorIs(t, err, ErrNotFound)
}

// ─── Not-found translation regression (same pattern as existing CRUD tests) ──

func TestQuotationService_TC_SVC_A20_SubmitNotFoundTranslatesToErrNotFound(t *testing.T) {
	// Arrange
	repo := new(mockQuotationRepository)
	userRepo := new(mockUserRepository)
	repo.On("FindByID", mock.Anything, uint(999)).Return((*model.Quotation)(nil), gorm.ErrRecordNotFound)
	svc := NewQuotationService(repo, userRepo, approvalFixedClock)

	// Act
	_, err := svc.SubmitQuotation(context.Background(), 7, "creator", 999)

	// Assert
	require.ErrorIs(t, err, ErrNotFound)
}
