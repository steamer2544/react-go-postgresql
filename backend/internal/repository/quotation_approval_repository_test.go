package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"imaxx-backend/internal/model"
)

// ─── Contract required from dev (internal/repository/quotation_repository.go) ──
//
//	var ErrStatusConflict = errors.New("status conflict")
//	type QuotationRepository interface {
//	    ... existing methods ...
//	    TransitionStatus(ctx context.Context, id uint, fromStatus string, updates map[string]any) error
//	}
//	Implementation MUST use an atomic conditional UPDATE (WHERE id = ? AND status = ?)
//	and MUST NOT use the existing Update() method (which fully replaces
//	Items/PaymentTerms). RowsAffected == 0 => return ErrStatusConflict.
//
// This file reuses testDB, setupTx(t), seedUser(t, tx) already declared in
// quotation_repository_test.go (same package) — do not duplicate them here.
// ─────────────────────────────────────────────────────────────────────────────

// seedDraftQuotationForApproval inserts a fresh draft quotation (with one
// item) owned by userID and returns its ID.
func seedDraftQuotationForApproval(t *testing.T, tx *gorm.DB, userID uint) uint {
	t.Helper()
	now := time.Now()
	q := &model.Quotation{
		ReferenceNo:    "QTAPR" + now.Format("150405.000000"),
		Status:         "draft",
		Attention:      "Test",
		Company:        "Approval Co",
		Email:          "approval@test.com",
		Date:           now,
		ValidUntil:     now.AddDate(0, 1, 0),
		CreatedBy:      userID,
		DiscountAmount: 0,
		Subtotal:       100,
		VatAmount:      7,
		Total:          107,
		Items: []model.QuotationItem{
			{ServiceType: "A", Description: "a", UnitPrice: 100, Qty: 1, LineTotal: 100, SortOrder: 1},
		},
	}
	require.NoError(t, tx.Create(q).Error)
	return q.ID
}

// TC-REPO-A01: TransitionStatus performs an atomic conditional update and
// leaves untouched columns (e.g. Company, Items) exactly as they were —
// proving the implementation does NOT use the full-replace Update() path.
func TestQuotationRepo_TC_REPO_A01_TransitionStatusHappyPartialUpdate(t *testing.T) {
	// Arrange
	ctx := context.Background()
	tx := setupTx(t)
	repo := NewQuotationRepository(tx)
	userID := seedUser(t, tx)
	id := seedDraftQuotationForApproval(t, tx, userID)

	// Act
	err := repo.TransitionStatus(ctx, id, "draft", map[string]any{"status": "pending_approval"})

	// Assert
	require.NoError(t, err)
	got, err := repo.FindByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "pending_approval", got.Status)
	require.Equal(t, "Approval Co", got.Company, "TransitionStatus must not touch unrelated columns")
	require.Len(t, got.Items, 1, "TransitionStatus must not delete/replace Items (that's Update()'s job, forbidden here)")
}

// TC-REPO-A02: calling TransitionStatus with a fromStatus that no longer
// matches the row's current status (simulating a concurrent transition)
// affects 0 rows and must return ErrStatusConflict.
func TestQuotationRepo_TC_REPO_A02_TransitionStatusConflictOnStatusMismatch(t *testing.T) {
	// Arrange
	ctx := context.Background()
	tx := setupTx(t)
	repo := NewQuotationRepository(tx)
	userID := seedUser(t, tx)
	id := seedDraftQuotationForApproval(t, tx, userID) // actual status = "draft"

	// Act — pretend caller's stale read said "pending_approval"
	err := repo.TransitionStatus(ctx, id, "pending_approval", map[string]any{"status": "approved"})

	// Assert
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrStatusConflict))

	// The row must remain unchanged (still "draft") since the update affected 0 rows.
	got, findErr := repo.FindByID(ctx, id)
	require.NoError(t, findErr)
	require.Equal(t, "draft", got.Status)
}

// TC-REPO-A03: TransitionStatus with a multi-column updates map (the
// approve-transition shape) persists every key given.
func TestQuotationRepo_TC_REPO_A03_TransitionStatusMultiColumnApprove(t *testing.T) {
	// Arrange
	ctx := context.Background()
	tx := setupTx(t)
	repo := NewQuotationRepository(tx)
	userID := seedUser(t, tx)
	approverID := seedUser(t, tx)
	id := seedDraftQuotationForApproval(t, tx, userID)
	require.NoError(t, repo.TransitionStatus(ctx, id, "draft", map[string]any{"status": "pending_approval"}))

	now := time.Now().UTC().Truncate(time.Second)
	updates := map[string]any{
		"status":                   "approved",
		"approver_id":              approverID,
		"approved_at":              now,
		"approved_signee_name":     "Approver Name",
		"approved_signee_position": "CFO",
		"approved_signature_path":  "/uploads/signatures/user_x.png",
	}

	// Act
	err := repo.TransitionStatus(ctx, id, "pending_approval", updates)

	// Assert
	require.NoError(t, err)
	got, err := repo.FindByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "approved", got.Status)
	require.NotNil(t, got.ApproverID)
	require.Equal(t, approverID, *got.ApproverID)
	require.NotNil(t, got.ApprovedSigneeName)
	require.Equal(t, "Approver Name", *got.ApprovedSigneeName)
	require.NotNil(t, got.ApprovedSigneePosition)
	require.Equal(t, "CFO", *got.ApprovedSigneePosition)
	require.NotNil(t, got.ApprovedSignaturePath)
	require.Equal(t, "/uploads/signatures/user_x.png", *got.ApprovedSignaturePath)
	require.NotNil(t, got.ApprovedAt)
}
