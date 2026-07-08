// Package repository defines data-access interfaces and implementations.
package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgconn"
	"imaxx-backend/internal/dto"
	"imaxx-backend/internal/model"

	"gorm.io/gorm"
)

// ErrDuplicateReferenceNo is returned when a quotation insert violates the
// unique index on reference_no (concurrent creation in the same month).
var ErrDuplicateReferenceNo = errors.New("duplicate reference_no")

// ErrStatusConflict is returned by TransitionStatus when the row's current status
// no longer matches fromStatus (concurrent transition already happened).
var ErrStatusConflict = errors.New("status conflict")

// QuotationRepository is the interface for quotation data access.
type QuotationRepository interface {
	Create(ctx context.Context, q *model.Quotation) error
	FindByID(ctx context.Context, id uint) (*model.Quotation, error)
	Update(ctx context.Context, q *model.Quotation) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, query dto.ListQuotationQuery) ([]model.Quotation, int64, error)
	NextReferenceNo(ctx context.Context, prefix string) (string, error)
	TransitionStatus(ctx context.Context, id uint, fromStatus string, updates map[string]any) error
}

type gormQuotationRepository struct {
	db *gorm.DB
}

// NewQuotationRepository creates a QuotationRepository backed by the given GORM db.
func NewQuotationRepository(db *gorm.DB) QuotationRepository {
	return &gormQuotationRepository{db: db}
}

// Create inserts a quotation and its items in one transaction.
// Returns ErrDuplicateReferenceNo if the insert violates the unique index on reference_no.
func (r *gormQuotationRepository) Create(ctx context.Context, q *model.Quotation) error {
	var txErr error
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txErr = tx.Create(q).Error
		return txErr
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateReferenceNo
		}
		return err
	}
	return txErr
}

// FindByID loads a quotation by ID with its items preloaded ordered by sort_order.
// Returns gorm.ErrRecordNotFound if not found.
func (r *gormQuotationRepository) FindByID(ctx context.Context, id uint) (*model.Quotation, error) {
	var q model.Quotation
	err := r.db.WithContext(ctx).
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order asc")
		}).
		Preload("PaymentTerms", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order asc")
		}).
		First(&q, id).Error
	if err != nil {
		return nil, err
	}
	return &q, nil
}

// Update replaces the quotation header fields and fully replaces the Items slice
// (delete old items, insert new items) inside one transaction.
func (r *gormQuotationRepository) Update(ctx context.Context, q *model.Quotation) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Delete all existing items for this quotation.
		if err := tx.Where("quotation_id = ?", q.ID).Delete(&model.QuotationItem{}).Error; err != nil {
			return err
		}
		// 2. Set QuotationID on all new items.
		for i := range q.Items {
			q.Items[i].QuotationID = q.ID
		}
		// 2b. Delete all existing payment terms for this quotation.
		if err := tx.Where("quotation_id = ?", q.ID).Delete(&model.PaymentTerm{}).Error; err != nil {
			return err
		}
		// 2c. Set QuotationID on all new payment terms.
		for i := range q.PaymentTerms {
			q.PaymentTerms[i].QuotationID = q.ID
		}
		// 3. Update header fields (all columns except Items, PaymentTerms and CreatedAt).
		if err := tx.Model(&model.Quotation{}).
			Where("id = ?", q.ID).
			Select("*").
			Omit("Items", "PaymentTerms", "CreatedAt").
			Updates(q).Error; err != nil {
			return err
		}
		// 4. Insert all new items.
		if len(q.Items) > 0 {
			if err := tx.Create(&q.Items).Error; err != nil {
				return err
			}
		}
		// 5. Insert all new payment terms.
		if len(q.PaymentTerms) > 0 {
			if err := tx.Create(&q.PaymentTerms).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Delete removes a quotation by ID. FK ON DELETE CASCADE handles item deletion.
func (r *gormQuotationRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Quotation{}, id).Error
}

// List returns quotations with pagination, sorting, and filtering.
func (r *gormQuotationRepository) List(ctx context.Context, query dto.ListQuotationQuery) ([]model.Quotation, int64, error) {
	// Sort whitelist.
	sortWhitelist := map[string]string{
		"created_at":    "created_at",
		"-created_at":   "-created_at",
		"date":          "date",
		"-date":         "-date",
		"total":         "total",
		"-total":        "-total",
		"reference_no":  "reference_no",
		"-reference_no": "-reference_no",
	}

	db := r.db.WithContext(ctx).Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order asc")
	}).Preload("PaymentTerms", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order asc")
	})

	// Filter.
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.CreatedBy != 0 {
		db = db.Where("created_by = ?", query.CreatedBy)
	}
	if query.DateGte != "" {
		db = db.Where("date >= ?", query.DateGte)
	}
	if query.DateLte != "" {
		db = db.Where("date <= ?", query.DateLte)
	}
	if query.Q != "" {
		q := query.Q
		db = db.Where("reference_no ILIKE ? OR company ILIKE ? OR attention ILIKE ? OR project ILIKE ?",
			"%"+q+"%", "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}

	// Count total before pagination.
	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Sort.
	if sortClause, ok := sortWhitelist[query.Sort]; ok {
		db = db.Order(sortClause)
	} else {
		db = db.Order("created_at DESC")
	}

	// Pagination.
	db = db.
		Offset((query.Page - 1) * query.PageSize).
		Limit(query.PageSize)

	var results []model.Quotation
	if err := db.Find(&results).Error; err != nil {
		return nil, 0, err
	}
	return results, total, nil
}

// NextReferenceNo computes "prefix" + zero-padded 3-digit running number
// based on MAX(reference_no) LIKE prefix || '%'.
func (r *gormQuotationRepository) NextReferenceNo(ctx context.Context, prefix string) (string, error) {
	var maxRef *string
	err := r.db.WithContext(ctx).
		Table("quotations").
		Select("reference_no").
		Where("reference_no LIKE ?", prefix+"%").
		Order("reference_no DESC").
		Pluck("reference_no", &maxRef).Error
	if err != nil {
		return "", err
	}

	var nextNum int
	if maxRef == nil || *maxRef == "" {
		nextNum = 1
	} else {
		suffix := (*maxRef)[len(prefix):]
		n, err := strconv.Atoi(suffix)
		if err != nil {
			// Fallback: treat as 0.
			nextNum = 1
		} else {
			nextNum = n + 1
		}
	}

	return fmt.Sprintf("%s%03d", prefix, nextNum), nil
}

// TransitionStatus performs an atomic conditional update: only rows matching both
// id and fromStatus are updated. If no row matched (status changed concurrently),
// it returns ErrStatusConflict. It must NOT be implemented via Update() (which
// fully replaces Items/PaymentTerms) — this only touches the given columns.
func (r *gormQuotationRepository) TransitionStatus(ctx context.Context, id uint, fromStatus string, updates map[string]any) error {
	result := r.db.WithContext(ctx).Model(&model.Quotation{}).
		Where("id = ? AND status = ?", id, fromStatus).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrStatusConflict
	}
	return nil
}
