// Package model defines GORM models for the application.
package model

import "time"

// Quotation represents a quotation document mapped to the "quotations" table.
type Quotation struct {
	ID                     uint   `gorm:"primaryKey"`
	ReferenceNo            string `gorm:"uniqueIndex;column:reference_no"`
	Attention              string
	Company                string
	Project                string
	Telephone              string
	Email                  string
	Date                   time.Time
	ValidUntil             time.Time
	Status                 string `gorm:"type:varchar(20);default:draft"`
	DiscountAmount         float64
	Subtotal               float64
	VatAmount              float64
	Total                  float64
	CustomerSigneeName     *string
	CustomerSigneePosition *string
	CustomerSigneeDate     *time.Time
	CompanySigneeName      string
	CompanySigneePosition  string
	CreatedBy              uint            `gorm:"column:created_by;not null"`
	Items                  []QuotationItem `gorm:"foreignKey:QuotationID;constraint:OnDelete:CASCADE"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// QuotationItem represents a line item within a quotation.
type QuotationItem struct {
	ID          uint `gorm:"primaryKey"`
	QuotationID uint `gorm:"not null"`
	ServiceType string
	Description string
	UnitPrice   float64
	Qty         int
	LineTotal   float64
	SortOrder   int
}
