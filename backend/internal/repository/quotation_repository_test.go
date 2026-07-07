// Package repository defines data access interfaces and implementations.
package repository

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"imaxx-backend/internal/model"
)

// ─── Contract that dev must implement (not written here) ────────────────────
//
//	package model
//	type Quotation struct {
//		ID, QuotationID, CreatedBy uint
//		ReferenceNo, Attention, Company, Email, Status string
//		Date, ValidUntil, CreatedAt, UpdatedAt time.Time
//		DiscountAmount, Subtotal, VatAmount, Total float64
//		CompanySigneeName, CompanySigneePosition string
//		Items []QuotationItem
//	}
//	type QuotationItem struct {
//		ID, QuotationID uint
//		ServiceType, Description string
//		UnitPrice, LineTotal float64
//		Qty, SortOrder int
//	}
//
//	package repository
//	// IMPORTANT (layering — see .claude/rules/backend.md "handler -> service ->
//	// repository -> model"): repository must NEVER import internal/service
//	// (service already imports repository for the interface type below — an
//	// import the other way round is a compile-time import cycle). Not-found
//	// is therefore signalled with the well-known gorm.ErrRecordNotFound
//	// sentinel (same convention as the existing gormUserRepository.FindByID),
//	// and a duplicate reference_no (unique-index violation) is signalled with
//	// a NEW repository-level sentinel declared in this package:
//	//   var ErrDuplicateReferenceNo = errors.New("duplicate reference_no")
//	// internal/service (which already imports internal/repository) is the
//	// one that translates ErrDuplicateReferenceNo -> retry -> (after 5
//	// attempts) service.ErrConflict, and gorm.ErrRecordNotFound -> service.ErrNotFound.
//	type QuotationRepository interface {
//		Create(ctx context.Context, q *model.Quotation) error // returns ErrDuplicateReferenceNo on unique-index violation
//		FindByID(ctx context.Context, id uint) (*model.Quotation, error) // returns gorm.ErrRecordNotFound if missing
//		Update(ctx context.Context, q *model.Quotation) error
//		Delete(ctx context.Context, id uint) error
//		List(ctx context.Context, query dto.ListQuotationQuery) ([]model.Quotation, int64, error)
//		NextReferenceNo(ctx context.Context, prefix string) (string, error)
//	}
//	func NewQuotationRepository(db *gorm.DB) QuotationRepository
//	var ErrDuplicateReferenceNo = errors.New("duplicate reference_no")
//
// ─── Package-level state ────────────────────────────────────────────────────

var (
	testDB        *gorm.DB
	testContainer testcontainers.Container
)

func TestMain(m *testing.M) {
	// flag.Parse() must run before testing.Short(): the testing package panics
	// with "Short called before Parse" if the -short flag hasn't been parsed yet.
	// m.Run() would parse it, but we need Short() earlier to decide whether to
	// spin up the Docker container at all.
	flag.Parse()

	if testing.Short() {
		os.Exit(m.Run())
	}

	ctx := context.Background()

	// Skip if Docker is not available
	if !dockerAvailable(ctx) {
		fmt.Fprintln(os.Stderr, "skip: Docker not available — set -short to skip integration tests")
		os.Exit(m.Run())
	}

	var err error
	testContainer, err = tcpostgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		tcpostgres.WithDatabase("quotation_test"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to start postgres container:", err)
		os.Exit(1)
	}

	host, err := testContainer.Host(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to get container host:", err)
		os.Exit(1)
	}
	mappedPort, err := testContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to get container mapped port:", err)
		os.Exit(1)
	}

	dsn := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=quotation_test sslmode=disable", host, mappedPort.Port())

	testDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to connect to postgres:", err)
		os.Exit(1)
	}

	// AutoMigrate models required by quotation tests (User needed for FK created_by → users(id))
	if err := testDB.AutoMigrate(&model.Quotation{}, &model.QuotationItem{}, &model.User{}); err != nil {
		fmt.Fprintln(os.Stderr, "failed to auto-migrate:", err)
		os.Exit(1)
	}

	code := m.Run()

	if err := testContainer.Terminate(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "failed to terminate container:", err)
	}

	os.Exit(code)
}

// dockerAvailable checks whether the Docker CLI is reachable (prevents hanging on
// machines where testcontainers would block waiting for an engine that will never
// start).
func dockerAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "docker", "info")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func setupTx(t *testing.T) *gorm.DB {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	tx := testDB.Begin()
	t.Cleanup(func() { tx.Rollback() })
	return tx
}

// seedUser inserts a fresh user (unique email) and returns its ID.
func seedUser(t *testing.T, tx *gorm.DB) uint {
	t.Helper()
	ts := time.Now().UTC().Format("20060102150405")
	u := &model.User{
		Email:        fmt.Sprintf("u%s@test.com", ts),
		PasswordHash: "x",
		Role:         model.RoleCreator,
		FullName:     "Test User",
		Position:     "Staff",
	}
	require.NoError(t, tx.Create(u).Error)
	return u.ID
}

// ─── TC-REPO-01: ReferenceNo sequential + Create ───────────────────────────

func TestQuotationRepo_TC_REPO_01_ReferenceNoSequential(t *testing.T) {
	// Arrange
	ctx := context.Background()
	tx := setupTx(t)
	repo := NewQuotationRepository(tx)
	userID := seedUser(t, tx)

	// Act — get first reference number
	ref1, err := repo.NextReferenceNo(ctx, "QT2607")
	require.NoError(t, err)
	require.Regexp(t, `^QT2607\d{3}$`, ref1)

	// Create a quotation with that reference number
	now := time.Now()
	q1 := &model.Quotation{
		ReferenceNo:    ref1,
		Status:         "draft",
		Attention:      "Test",
		Company:        "Test Co",
		Email:          "test@test.com",
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
	require.NoError(t, repo.Create(ctx, q1))

	// Get the next reference number — must differ and be +1
	ref2, err := repo.NextReferenceNo(ctx, "QT2607")
	require.NoError(t, err)
	require.NotEqual(t, ref1, ref2)

	// Verify running numbers are sequential (last 3 digits)
	ref1Num, err := parseRunningNumber(ref1)
	require.NoError(t, err)
	ref2Num, err := parseRunningNumber(ref2)
	require.NoError(t, err)
	require.Equal(t, ref1Num+1, ref2Num, "ref2 running number should be ref1+1")
}

func parseRunningNumber(ref string) (int, error) {
	s := ref[len(ref)-3:]
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// ─── TC-REPO-02: Duplicate reference_no conflict ───────────────────────────

func TestQuotationRepo_TC_REPO_02_DuplicateReferenceNoConflict(t *testing.T) {
	// Arrange
	ctx := context.Background()
	tx := setupTx(t)
	repo := NewQuotationRepository(tx)
	userID := seedUser(t, tx)

	now := time.Now()
	q1 := &model.Quotation{
		ReferenceNo:    "QT2607999",
		Status:         "draft",
		Attention:      "Test",
		Company:        "Test Co",
		Email:          "test2@test.com",
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
	require.NoError(t, repo.Create(ctx, q1))

	// Act — attempt to create a second quotation with the same reference_no
	q2 := &model.Quotation{
		ReferenceNo:    "QT2607999",
		Status:         "draft",
		Attention:      "Test2",
		Company:        "Test Co 2",
		Email:          "test2b@test.com",
		Date:           now,
		ValidUntil:     now.AddDate(0, 1, 0),
		CreatedBy:      userID,
		DiscountAmount: 0,
		Subtotal:       200,
		VatAmount:      14,
		Total:          214,
		Items: []model.QuotationItem{
			{ServiceType: "B", Description: "b", UnitPrice: 200, Qty: 1, LineTotal: 200, SortOrder: 1},
		},
	}
	err := repo.Create(ctx, q2)

	// Assert — must return error matching the repository-level duplicate
	// reference_no sentinel (NOT service.ErrConflict — repository must never
	// import internal/service, see contract comment at top of this file).
	require.Error(t, err)
	require.ErrorIs(t, err, ErrDuplicateReferenceNo)
}

// ─── TC-REPO-03: Delete cascades to items ──────────────────────────────────

func TestQuotationRepo_TC_REPO_03_DeleteCascadesItems(t *testing.T) {
	// Arrange
	ctx := context.Background()
	tx := setupTx(t)
	repo := NewQuotationRepository(tx)
	userID := seedUser(t, tx)

	now := time.Now()
	q := &model.Quotation{
		ReferenceNo:    "QT2607111",
		Status:         "draft",
		Attention:      "Test",
		Company:        "Test Co",
		Email:          "test3@test.com",
		Date:           now,
		ValidUntil:     now.AddDate(0, 1, 0),
		CreatedBy:      userID,
		DiscountAmount: 0,
		Subtotal:       300,
		VatAmount:      21,
		Total:          321,
		Items: []model.QuotationItem{
			{ServiceType: "A", Description: "a", UnitPrice: 100, Qty: 1, LineTotal: 100, SortOrder: 1},
			{ServiceType: "B", Description: "b", UnitPrice: 200, Qty: 1, LineTotal: 200, SortOrder: 2},
		},
	}
	require.NoError(t, repo.Create(ctx, q))

	// Act — delete the quotation
	require.NoError(t, repo.Delete(ctx, q.ID))

	// Assert — FindByID must return gorm's well-known not-found sentinel
	// (same convention as gormUserRepository.FindByID — repository does not
	// import internal/service, see contract comment at top of this file).
	_, err := repo.FindByID(ctx, q.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Assert — quotation_items for this ID must be 0 (CASCADE)
	var itemCount int64
	tx.Model(&model.QuotationItem{}).Where("quotation_id = ?", q.ID).Count(&itemCount)
	require.Equal(t, int64(0), itemCount)
}

// ─── TC-REPO-04: Update replaces items in transaction ──────────────────────

func TestQuotationRepo_TC_REPO_04_UpdateReplacesItemsInTransaction(t *testing.T) {
	// Arrange
	ctx := context.Background()
	tx := setupTx(t)
	repo := NewQuotationRepository(tx)
	userID := seedUser(t, tx)

	now := time.Now()
	q := &model.Quotation{
		ReferenceNo:    "QT2607222",
		Status:         "draft",
		Attention:      "Test",
		Company:        "Test Co",
		Email:          "test4@test.com",
		Date:           now,
		ValidUntil:     now.AddDate(0, 1, 0),
		CreatedBy:      userID,
		DiscountAmount: 0,
		Subtotal:       300,
		VatAmount:      21,
		Total:          321,
		Items: []model.QuotationItem{
			{ServiceType: "A", Description: "a", UnitPrice: 100, Qty: 1, LineTotal: 100, SortOrder: 1},
			{ServiceType: "B", Description: "b", UnitPrice: 200, Qty: 1, LineTotal: 200, SortOrder: 2},
		},
	}
	require.NoError(t, repo.Create(ctx, q))

	// Act — update with completely different items
	q.Items = []model.QuotationItem{
		{ServiceType: "C", Description: "c", UnitPrice: 300, Qty: 1, LineTotal: 300, SortOrder: 1},
		{ServiceType: "D", Description: "d", UnitPrice: 400, Qty: 2, LineTotal: 800, SortOrder: 2},
		{ServiceType: "E", Description: "e", UnitPrice: 500, Qty: 1, LineTotal: 500, SortOrder: 3},
	}
	require.NoError(t, repo.Update(ctx, q))

	// Assert — FindByID should return the quotation with exactly 3 new items
	got, err := repo.FindByID(ctx, q.ID)
	require.NoError(t, err)
	require.Len(t, got.Items, 3)

	serviceTypes := make([]string, 0, len(got.Items))
	for _, it := range got.Items {
		serviceTypes = append(serviceTypes, it.ServiceType)
	}
	// A, B must be gone; only C, D, E remain
	require.ElementsMatch(t, []string{"C", "D", "E"}, serviceTypes)
}
