// Package dto defines request and response data transfer objects.
package dto

// QuotationItemInput is the input for a single quotation line item.
type QuotationItemInput struct {
	ServiceType string  `json:"service_type" binding:"required"`
	Description string  `json:"description" binding:"required"`
	UnitPrice   float64 `json:"unit_price" binding:"required,gte=0"`
	Qty         int     `json:"qty" binding:"required,gte=1"`
	SortOrder   int     `json:"sort_order"`
}

// CreateQuotationRequest is the payload for creating a quotation.
type CreateQuotationRequest struct {
	Attention              string               `json:"attention" binding:"required"`
	Company                string               `json:"company" binding:"required"`
	Project                string               `json:"project"`
	Telephone              string               `json:"telephone"`
	Email                  string               `json:"email" binding:"required,email"`
	Date                   string               `json:"date" binding:"required"`
	ValidUntil             string               `json:"valid_until" binding:"required"`
	DiscountAmount         float64              `json:"discount_amount" binding:"gte=0"`
	CustomerSigneeName     *string              `json:"customer_signee_name"`
	CustomerSigneePosition *string              `json:"customer_signee_position"`
	CustomerSigneeDate     *string              `json:"customer_signee_date"`
	Items                  []QuotationItemInput `json:"items" binding:"required,min=1,dive"`
	PaymentTerms           []PaymentTermInput   `json:"payment_terms" binding:"omitempty,dive"`
}

// UpdateQuotationRequest has the same shape as CreateQuotationRequest (full-replace PUT).
type UpdateQuotationRequest CreateQuotationRequest

// PaymentTermInput is the input for a single payment term.
type PaymentTermInput struct {
	Description string  `json:"description" binding:"required"`
	Amount      float64 `json:"amount" binding:"gt=0"`
}

// QuotationItemResponse is the output for a single quotation line item.
type QuotationItemResponse struct {
	ServiceType string  `json:"service_type"`
	Description string  `json:"description"`
	UnitPrice   float64 `json:"unit_price"`
	Qty         int     `json:"qty"`
	LineTotal   float64 `json:"line_total"`
	SortOrder   int     `json:"sort_order"`
}

// PaymentTermResponse is the output for a single payment term.
type PaymentTermResponse struct {
	ID          uint    `json:"id"`
	TermNo      int     `json:"term_no"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	SortOrder   int     `json:"sort_order"`
}

// QuotationResponse is the success payload for a quotation detail endpoint.
type QuotationResponse struct {
	ID                     uint                    `json:"id"`
	ReferenceNo            string                  `json:"reference_no"`
	Status                 string                  `json:"status"`
	Attention              string                  `json:"attention"`
	Company                string                  `json:"company"`
	Project                string                  `json:"project"`
	Telephone              string                  `json:"telephone"`
	Email                  string                  `json:"email"`
	Date                   string                  `json:"date"`
	ValidUntil             string                  `json:"valid_until"`
	Subtotal               float64                 `json:"subtotal"`
	DiscountAmount         float64                 `json:"discount_amount"`
	VatAmount              float64                 `json:"vat_amount"`
	Total                  float64                 `json:"total"`
	Items                  []QuotationItemResponse `json:"items"`
	PaymentTerms           []PaymentTermResponse   `json:"payment_terms"`
	CustomerSigneeName     *string                 `json:"customer_signee_name"`
	CustomerSigneePosition *string                 `json:"customer_signee_position"`
	CustomerSigneeDate     *string                 `json:"customer_signee_date"`
	CompanySigneeName      string                  `json:"company_signee_name"`
	CompanySigneePosition  string                  `json:"company_signee_position"`
	CreatedBy              uint                    `json:"created_by"`
}

// ListQuotationQuery carries list-query parameters for the quotations list endpoint.
type ListQuotationQuery struct {
	Page      int    `form:"page,default=1" binding:"min=1"`
	PageSize  int    `form:"page_size,default=20" binding:"min=1,max=100"`
	Sort      string `form:"sort"`
	Status    string `form:"status" binding:"omitempty,oneof=draft sent approved rejected"`
	CreatedBy uint   `form:"created_by"`
	DateGte   string `form:"date_gte" binding:"omitempty,datetime=2006-01-02"`
	DateLte   string `form:"date_lte" binding:"omitempty,datetime=2006-01-02"`
	Q         string `form:"q"`
}
