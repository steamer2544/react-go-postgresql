# Test Cases: CRUD ใบเสนอราคา + Calc Engine (slug: quotation-crud)

อ้างอิงแผน: `docs/plans/quotation-crud.md`
อ้างอิงสัญญา: `.claude/docs/{api-response,list-query,auth,error-logging,testing,standard-libraries}.md`,
`.claude/rules/{naming-conventions,backend,frontend}.md`

> เอกสารนี้เป็น **spec ที่ qwen (`claude-9arm`) ต้องพิมพ์ไฟล์ test ตามเป๊ะ ๆ** — ห้ามเดาตัวเลข/ชื่อ field
> ทุกค่าที่ปรากฏด้านล่างมาจาก AC ในแผนหรือถูก pin ไว้ชัดเจนโดย test-case-writer (ระบุเหตุผลกำกับ)

---

## 0. Contract ที่ dev ต้องสร้าง (สัญญา API สำหรับให้ test เรียกได้)

### 0.1 Backend — `internal/service/quotation_calc.go` (pure, ไม่แตะ DB)

```go
package service

// roundHalfUpCents converts a monetary amount in major units (e.g. THB, "250.50")
// to integer cents (satang), rounding half-up (ties away from zero). Used to
// convert unitPrice into an exact integer before any further arithmetic so that
// float64 representation error never propagates into qty multiplication or VAT.
func roundHalfUpCents(amount float64) int64

// calcLineTotalCents returns qty * roundHalfUpCents(unitPrice) — qty is an exact
// integer so this multiplication is 100% exact (no rounding needed here).
func calcLineTotalCents(unitPrice float64, qty int) int64

// calcTotals sums lineItemCents (already-computed line totals in cents),
// applies discountAmount (major units, converted via roundHalfUpCents), and
// computes VAT 7% with half-up rounding using pure integer arithmetic:
//   subtotalCents = sum(lineItemCents)
//   discountCents = roundHalfUpCents(discountAmount)
//   baseCents     = subtotalCents - discountCents
//   vatCents      = (baseCents*7 + 50) / 100   // integer division, half-up
//   totalCents    = baseCents + vatCents
// Returns service.ErrValidation if discountCents < 0 or discountCents > subtotalCents
// (per Decision #8: 0 ≤ discount_amount ≤ subtotal). On error, the four returned
// cents values are the zero value (0) — callers must check err first.
func calcTotals(lineItemCents []int64, discountAmount float64) (subtotalCents, baseCents, vatCents, totalCents int64, err error)
```

### 0.2 Backend — `internal/repository/quotation_repository.go`

> **แก้ไขหลัง review (สำคัญ — layering bug ที่พบตอนตรวจไฟล์ที่ qwen เขียน):** ห้าม repository
> import `internal/service` เด็ดขาด เพราะ `internal/service` import `internal/repository`
> อยู่แล้ว (สำหรับ interface นี้เอง) — import ย้อนกลับจะเกิด **import cycle** (compile fail
> ทันทีทั้ง build ไม่ใช่แค่ RED ที่ตั้งใจ) ดังนั้น:
>
> - not-found ใช้ sentinel ของ gorm ตรง ๆ (`gorm.ErrRecordNotFound`) — สอดคล้องกับ
>   `gormUserRepository.FindByID` ที่มีอยู่แล้วซึ่ง**คืน raw gorm error โดยไม่แปล**
>   (ดู `backend/internal/repository/user_repository.go`)
> - unique-index violation (reference_no ซ้ำ) ใช้ sentinel **ใหม่ระดับ repository package เอง**:
>   `var ErrDuplicateReferenceNo = errors.New("duplicate reference_no")`
> - หน้าที่แปลเป็น `service.ErrNotFound`/`service.ErrConflict` (409/404 ตาม api-response.md)
>   เป็นของ **service layer** เท่านั้น (service import repository ได้ทางเดียว ไม่ใช่กลับกัน)

```go
package repository

var ErrDuplicateReferenceNo = errors.New("duplicate reference_no")

type QuotationRepository interface {
    // Create inserts a quotation + its items in one GORM transaction. If the
    // insert violates the unique index on reference_no (concurrent creation
    // in the same month), Create returns ErrDuplicateReferenceNo (this package's
    // own sentinel — errors.Is-compatible) so the service layer can detect it
    // and retry with a fresh reference_no.
    Create(ctx context.Context, q *model.Quotation) error
    FindByID(ctx context.Context, id uint) (*model.Quotation, error) // gorm.ErrRecordNotFound if missing
    // Update replaces the quotation header fields AND fully replaces the Items
    // slice (delete old items, insert new items) inside one transaction.
    Update(ctx context.Context, q *model.Quotation) error
    Delete(ctx context.Context, id uint) error // cascades to quotation_items via FK ON DELETE CASCADE
    List(ctx context.Context, query dto.ListQuotationQuery) ([]model.Quotation, int64, error)
    // NextReferenceNo computes "prefix" + zero-padded 3-digit running number
    // for the given prefix (e.g. "QT2607"), based on MAX(reference_no) LIKE prefix%.
    NextReferenceNo(ctx context.Context, prefix string) (string, error)
}
```

### 0.3 Backend — `internal/service/quotation_service.go`

```go
package service

func NewQuotationService(repo repository.QuotationRepository, userRepo repository.UserRepository, clock func() time.Time) *QuotationService

// CreateQuotation: parses date/valid_until ("2006-01-02"), validates ValidUntil >= Date
// (Decision #9, ErrValidation), computes calcTotals (ErrValidation passthrough on bad
// discount), generates reference_no via repo.NextReferenceNo(ctx, "QT"+YYMM from clock()),
// sets ReferenceNo on the quotation and calls repo.Create; if repo.Create returns an
// error matching errors.Is(err, repository.ErrDuplicateReferenceNo) (the repository's
// own sentinel — NOT service.ErrConflict, see corrected 0.2 above), retries by calling
// NextReferenceNo again and Create again, up to 5 total attempts; after the 5th failed
// attempt, CreateQuotation returns service.ErrConflict (errors.Is(returnedErr, ErrConflict)
// == true) — i.e. the service translates the low-level repository signal into its own
// public domain sentinel only at the point it gives up retrying. Snapshots CompanySigneeName/
// CompanySigneePosition from userRepo.FindByID(ctx, userID) at creation time only
// (Decision #7). Sets Status="draft", CreatedBy=userID.
func (s *QuotationService) CreateQuotation(ctx context.Context, userID uint, req dto.CreateQuotationRequest) (*dto.QuotationResponse, error)

// UpdateQuotation: loads existing via repo.FindByID; Status != "draft" -> ErrForbidden;
// role != "admin" && existing.CreatedBy != userID -> ErrForbidden; otherwise recomputes
// totals (same validation as Create) and calls repo.Update with a full item replace.
// CompanySigneeName/CompanySigneePosition/CompanySigneeDate are carried over UNCHANGED
// from the loaded record (never re-derived from the current caller's profile — Decision #7).
func (s *QuotationService) UpdateQuotation(ctx context.Context, userID uint, role string, id uint, req dto.UpdateQuotationRequest) (*dto.QuotationResponse, error)

// DeleteQuotation: same draft-only + ownership checks as UpdateQuotation, then repo.Delete.
func (s *QuotationService) DeleteQuotation(ctx context.Context, userID uint, role string, id uint) error

func (s *QuotationService) GetQuotation(ctx context.Context, id uint) (*dto.QuotationResponse, error)
func (s *QuotationService) ListQuotations(ctx context.Context, query dto.ListQuotationQuery) ([]dto.QuotationResponse, int64, error)
```

### 0.4 Backend — `internal/handler/quotation_handler.go`

```go
package handler

type QuotationServicer interface {
    CreateQuotation(ctx context.Context, userID uint, req dto.CreateQuotationRequest) (*dto.QuotationResponse, error)
    UpdateQuotation(ctx context.Context, userID uint, role string, id uint, req dto.UpdateQuotationRequest) (*dto.QuotationResponse, error)
    DeleteQuotation(ctx context.Context, userID uint, role string, id uint) error
    GetQuotation(ctx context.Context, id uint) (*dto.QuotationResponse, error)
    ListQuotations(ctx context.Context, query dto.ListQuotationQuery) ([]dto.QuotationResponse, int64, error)
}
func NewQuotationHandler(svc QuotationServicer) *QuotationHandler
// Methods read userID/role from gin.Context (set by middleware.Auth), same as MeHandler:
func (h *QuotationHandler) Create(c *gin.Context)  // POST /quotations   -> 201
func (h *QuotationHandler) List(c *gin.Context)    // GET  /quotations   -> 200 list envelope
func (h *QuotationHandler) Get(c *gin.Context)     // GET  /quotations/:id -> 200
func (h *QuotationHandler) Update(c *gin.Context)  // PUT  /quotations/:id -> 200
func (h *QuotationHandler) Delete(c *gin.Context)  // DELETE /quotations/:id -> 204, no body
```

List query binding: handler MUST reject (400 `VALIDATION_ERROR`, service NOT called) any
request whose raw query string contains a key outside the whitelist
`{page,page_size,sort,status,created_by,date_gte,date_lte,q}`, any `sort` value outside
`{created_at,-created_at,date,-date,total,-total,reference_no,-reference_no}`, and any
`page_size > 100` (Decision #10).

### 0.5 Backend — `internal/dto/quotation_dto.go` (field names — exact, snake_case json tags)

```go
type QuotationItemInput struct {
    ServiceType string  `json:"service_type" binding:"required"`
    Description string  `json:"description" binding:"required"`
    UnitPrice   float64 `json:"unit_price" binding:"required,gte=0"`
    Qty         int     `json:"qty" binding:"required,gte=1"`
    SortOrder   int     `json:"sort_order"`
}
type CreateQuotationRequest struct {
    Attention              string               `json:"attention" binding:"required"`
    Company                string               `json:"company" binding:"required"`
    Project                string               `json:"project"`
    Telephone              string               `json:"telephone"`
    Email                  string               `json:"email" binding:"required,email"`
    Date                   string               `json:"date" binding:"required"`        // "YYYY-MM-DD"
    ValidUntil             string               `json:"valid_until" binding:"required"` // "YYYY-MM-DD"
    DiscountAmount         float64              `json:"discount_amount" binding:"gte=0"`
    CustomerSigneeName     *string              `json:"customer_signee_name"`
    CustomerSigneePosition *string              `json:"customer_signee_position"`
    CustomerSigneeDate     *string              `json:"customer_signee_date"`
    Items                  []QuotationItemInput `json:"items" binding:"required,min=1,dive"`
}
type UpdateQuotationRequest CreateQuotationRequest // same shape (full replace PUT)

type QuotationItemResponse struct {
    ServiceType string  `json:"service_type"`
    Description string  `json:"description"`
    UnitPrice   float64 `json:"unit_price"`
    Qty         int     `json:"qty"`
    LineTotal   float64 `json:"line_total"`
    SortOrder   int     `json:"sort_order"`
}
type QuotationResponse struct {
    ID                     uint                     `json:"id"`
    ReferenceNo            string                   `json:"reference_no"`
    Status                 string                   `json:"status"`
    Attention              string                   `json:"attention"`
    Company                string                   `json:"company"`
    Project                string                   `json:"project"`
    Telephone              string                   `json:"telephone"`
    Email                  string                   `json:"email"`
    Date                   string                   `json:"date"`
    ValidUntil             string                   `json:"valid_until"`
    Subtotal               float64                  `json:"subtotal"`
    DiscountAmount         float64                  `json:"discount_amount"`
    VatAmount              float64                  `json:"vat_amount"`
    Total                  float64                  `json:"total"`
    Items                  []QuotationItemResponse  `json:"items"`
    CustomerSigneeName     *string                  `json:"customer_signee_name"`
    CustomerSigneePosition *string                  `json:"customer_signee_position"`
    CustomerSigneeDate     *string                  `json:"customer_signee_date"`
    CompanySigneeName      string                   `json:"company_signee_name"`
    CompanySigneePosition  string                   `json:"company_signee_position"`
    CreatedBy              uint                     `json:"created_by"`
}
type ListQuotationQuery struct {
    Page      int    `form:"page,default=1" binding:"min=1"`
    PageSize  int    `form:"page_size,default=20" binding:"min=1,max=100"`
    Sort      string `form:"sort"`
    Status    string `form:"status" binding:"omitempty,oneof=draft sent approved rejected"`
    CreatedBy uint   `form:"created_by"`
    DateGte   string `form:"date_gte"`
    DateLte   string `form:"date_lte"`
    Q         string `form:"q"`
}
```

### 0.6 Frontend — `features/quotation/utils/calcQuotation.js`

```js
// calcLineTotal(unitPrice: number, qty: number): number
// calcTotals(items: {unitPrice:number, qty:number}[], discountAmount: number):
//   { subtotal: number, discountAmount: number, vatAmount: number|null, total: number|null, error: string|null }
// error === 'DISCOUNT_EXCEEDS_SUBTOTAL' when discountAmount < 0 or > subtotal
//   (compare by this exact string code, mirrors error.code convention — api-response.md)
// Internally MUST mirror the backend integer-cents algorithm (Math.round(x*100),
// integer VAT formula) — NOT `toFixed` on a float product.
```

### 0.7 Frontend — pages (dev builds under `features/quotation/`)

- `QuotationFormPage.jsx` — item rows with `data-testid="item-row"`, each having
  `unit_price`/`qty` inputs; discount input `data-testid="discount-input"` +
  error `data-testid="discount-error"`; summary block with
  `data-testid="summary-subtotal"`, `summary-discount`, `summary-vat`, `summary-total`
  (formatted `toLocaleString` with 2 decimals, e.g. `2,751.50`); submit button
  accessible name matching `/save|submit|บันทึก/i`.
- `QuotationListPage.jsx` — renders rows with the quotation's `reference_no` text
  visible; issues `GET /quotations` with the current page/sort/status params.

---

## 1. ตารางสรุป Test Case

| ID            | อ้างอิง AC                | ประเภท           | Given                                                                                                        | When                                       | Then                                                                                                                                                |
| ------------- | ------------------------- | ---------------- | ------------------------------------------------------------------------------------------------------------ | ------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| TC-CALC-01    | AC1                       | happy            | unitPrice/qty คู่ที่ 1,2                                                                                     | `calcLineTotalCents`                       | 200000, 75150                                                                                                                                       |
| TC-CALC-02    | AC1                       | happy            | lineItemCents=[200000,75150], discount=151.50                                                                | `calcTotals`                               | subtotal=275150, base=260000, vat=18200, total=278200                                                                                               |
| TC-CALC-03    | AC2                       | edge (tie)       | lineItemCents=[1050], discount=0                                                                             | `calcTotals`                               | vat=74 (ไม่ใช่ 73), total=1124                                                                                                                      |
| TC-CALC-04    | AC3                       | error            | lineItemCents=[275150], discount=300000 (คิดเป็น 3000.00)                                                    | `calcTotals`                               | err = ErrValidation                                                                                                                                 |
| TC-CALC-05    | AC4                       | edge (boundary)  | lineItemCents=[275150], discount=275150 (เท่ากับ subtotal เป๊ะ)                                              | `calcTotals`                               | base=0, vat=0, total=0, err=nil                                                                                                                     |
| TC-CALC-06    | Decision #8               | error            | discount=-0.01 (ติดลบ)                                                                                       | `calcTotals`                               | err = ErrValidation                                                                                                                                 |
| TC-CALC-07    | AC1 (สนับสนุน)            | happy            | amount=250.50 / 1000.00                                                                                      | `roundHalfUpCents`                         | 25050 / 100000                                                                                                                                      |
| TC-CALC-08    | Decision #1               | edge             | amount=0.006                                                                                                 | `roundHalfUpCents`                         | 1 (ปัดขึ้น ไม่ truncate เป็น 0)                                                                                                                     |
| TC-SVC-01     | AC1,AC5,AC13              | happy            | items เหมือน AC1, clock คงที่ 2026-07-15, user profile mock                                                  | `CreateQuotation`                          | ref_no="QT2607001", status="draft", signee snapshot ตรง user, subtotal/discount/vat/total ตรง AC1                                                   |
| TC-SVC-02     | AC5                       | edge             | repo.Create ชน conflict 2 ครั้งแรก                                                                           | `CreateQuotation`                          | สำเร็จครั้งที่ 3, ref_no จากการ retry ล่าสุด, Create ถูกเรียก 3 ครั้ง                                                                               |
| TC-SVC-03     | Decision #3               | error            | repo.Create ชน conflict ทุกครั้ง (5 ครั้ง)                                                                   | `CreateQuotation`                          | คืน err ที่ errors.Is(ErrConflict), Create ถูกเรียกพอดี 5 ครั้ง (ไม่ใช่ 6)                                                                          |
| TC-SVC-04     | AC2                       | edge (full flow) | items ที่ base=10.50, discount=0                                                                             | `CreateQuotation`                          | vat_amount=0.74, total=11.24                                                                                                                        |
| TC-SVC-05     | AC3                       | error            | discount_amount=3000.00 > subtotal 2751.50                                                                   | `CreateQuotation`                          | err=ErrValidation, repo.Create ไม่ถูกเรียก                                                                                                          |
| TC-SVC-06     | AC4                       | edge             | discount_amount == subtotal เป๊ะ                                                                             | `CreateQuotation`                          | สำเร็จ, base/vat/total=0.00                                                                                                                         |
| TC-SVC-07     | Decision #9               | error            | valid_until < date                                                                                           | `CreateQuotation`                          | err=ErrValidation                                                                                                                                   |
| TC-SVC-08     | AC6                       | happy            | quotation เดิม status=draft                                                                                  | `UpdateQuotation`                          | สำเร็จ, repo.Update ถูกเรียกด้วยค่าที่คำนวณใหม่                                                                                                     |
| TC-SVC-09     | AC7                       | error            | quotation เดิม status="sent"                                                                                 | `UpdateQuotation`                          | err=ErrForbidden, repo.Update ไม่ถูกเรียก                                                                                                           |
| TC-SVC-10     | AC8                       | error            | role=creator, CreatedBy=userB, caller=userA                                                                  | `UpdateQuotation`                          | err=ErrForbidden                                                                                                                                    |
| TC-SVC-11     | AC8                       | happy            | role=admin, CreatedBy=userB, caller=userA(admin)                                                             | `UpdateQuotation`                          | สำเร็จ (ข้าม ownership check)                                                                                                                       |
| TC-SVC-12     | AC13                      | edge             | quotation เดิมมี company_signee_name="Old Name"                                                              | `UpdateQuotation`                          | ผลลัพธ์ company_signee_name ยังเป็น "Old Name" ไม่เปลี่ยนตาม profile ปัจจุบัน                                                                       |
| TC-SVC-13     | AC6                       | happy            | quotation เดิม status=draft                                                                                  | `DeleteQuotation`                          | สำเร็จ, repo.Delete ถูกเรียก                                                                                                                        |
| TC-SVC-14     | AC7                       | error            | quotation เดิม status="approved"                                                                             | `DeleteQuotation`                          | err=ErrForbidden, repo.Delete ไม่ถูกเรียก                                                                                                           |
| TC-SVC-15     | AC8                       | error            | role=creator, CreatedBy=userB, caller=userA                                                                  | `DeleteQuotation`                          | err=ErrForbidden                                                                                                                                    |
| TC-SVC-16     | AC8                       | happy            | role=admin                                                                                                   | `DeleteQuotation`                          | สำเร็จ                                                                                                                                              |
| TC-HDL-01     | AC12                      | happy            | mock GetQuotation คืนค่าครบทุก field                                                                         | `GET /quotations/:id`                      | 200, body มีครบทุก key ที่ระบุใน AC12                                                                                                               |
| TC-HDL-02     | AC1                       | happy            | mock CreateQuotation สำเร็จ                                                                                  | `POST /quotations`                         | 201, data.reference_no ตรง mock, message ไม่ว่าง                                                                                                    |
| TC-HDL-03     | AC3                       | error            | mock CreateQuotation คืน ErrValidation                                                                       | `POST /quotations`                         | 400, error.code=VALIDATION_ERROR                                                                                                                    |
| TC-HDL-04     | AC7/AC8                   | error            | mock UpdateQuotation คืน ErrForbidden                                                                        | `PUT /quotations/:id`                      | 403, error.code=FORBIDDEN                                                                                                                           |
| TC-HDL-05     | AC6/AC11                  | happy            | mock DeleteQuotation สำเร็จ                                                                                  | `DELETE /quotations/:id`                   | 204, body ว่าง                                                                                                                                      |
| TC-HDL-06     | AC7/AC8                   | error            | mock DeleteQuotation คืน ErrForbidden                                                                        | `DELETE /quotations/:id`                   | 403, error.code=FORBIDDEN                                                                                                                           |
| TC-HDL-07     | AC10                      | happy            | query `page=1&page_size=20&sort=-created_at&status=draft`                                                    | `GET /quotations`                          | service ถูกเรียกด้วย query ตรงตาม param, 200, meta.page=1, meta.page_size=20, meta.total ตรง mock                                                   |
| TC-HDL-08     | AC11                      | error            | query `sort=unit_price`                                                                                      | `GET /quotations`                          | 400 VALIDATION_ERROR, service ไม่ถูกเรียก                                                                                                           |
| TC-HDL-09     | AC11                      | error            | query `foo=bar`                                                                                              | `GET /quotations`                          | 400 VALIDATION_ERROR, service ไม่ถูกเรียก                                                                                                           |
| TC-HDL-10     | AC11                      | error            | query `page_size=1000`                                                                                       | `GET /quotations`                          | 400 VALIDATION_ERROR, service ไม่ถูกเรียก                                                                                                           |
| TC-HDL-11     | AC9                       | error            | token role=approver                                                                                          | `POST /quotations`                         | 403 FORBIDDEN, service ไม่ถูกเรียก                                                                                                                  |
| TC-HDL-12     | AC9                       | happy            | token role=approver                                                                                          | `GET /quotations`                          | 200                                                                                                                                                 |
| TC-HDL-13     | AC9                       | happy            | token role=approver                                                                                          | `GET /quotations/:id`                      | 200                                                                                                                                                 |
| TC-HDL-14     | AC9                       | error            | ไม่มี header Authorization                                                                                   | `GET /quotations`                          | 401 UNAUTHORIZED                                                                                                                                    |
| TC-HDL-15     | AC9                       | happy            | token role=creator                                                                                           | `POST /quotations`                         | ผ่าน middleware ถึง handler, 201                                                                                                                    |
| TC-HDL-16     | AC19                      | error            | mock service คืน error ทั่วไป (ไม่ใช่ sentinel ที่รู้จัก) มีข้อความหลุด internal เช่น "dsn=... conn refused" | `GET /quotations/:id`                      | 500 INTERNAL_ERROR, body ไม่มีคำว่า "dsn=" หรือ "conn refused"                                                                                      |
| TC-REPO-01    | AC5                       | happy            | DB จริง, เดือน 2026-07, ยังไม่มีแถว                                                                          | `NextReferenceNo` แล้ว `Create` 2 ครั้งติด | ครั้งแรก match `^QT2607\d{3}$`, ครั้งสอง = ครั้งแรก+1, ไม่ error                                                                                    |
| TC-REPO-02    | AC5                       | error            | 2 quotation ที่ reference_no ซ้ำกันเป๊ะ                                                                      | `Create` ครั้งที่สอง                       | err ที่ errors.Is(repository.ErrDuplicateReferenceNo) (sentinel ระดับ repository เอง — แก้หลัง review เพื่อกัน import cycle กับ service, ดูข้อ 0.2) |
| TC-REPO-03    | AC6                       | happy            | quotation + 2 items ถูกสร้างแล้ว                                                                             | `Delete`                                   | quotation หาย (errors.Is(gorm.ErrRecordNotFound)), quotation_items ของ id นั้นเหลือ 0 แถว                                                           |
| TC-REPO-04    | Decision — Update replace | happy            | quotation มี items A,B                                                                                       | `Update` ด้วย items C,D,E                  | FindByID คืน items 3 แถวตรงกับ C,D,E เท่านั้น (A,B หายไป)                                                                                           |
| TC-FE-CALC-01 | AC1                       | happy            | items AC1                                                                                                    | `calcLineTotal` x2                         | 2000.00, 751.50                                                                                                                                     |
| TC-FE-CALC-02 | AC1                       | happy            | items AC1, discount=151.50                                                                                   | `calcTotals`                               | subtotal=2751.50, vatAmount=182.00, total=2782.00, error=null                                                                                       |
| TC-FE-CALC-03 | AC2                       | edge (tie)       | subtotal=10.50, discount=0                                                                                   | `calcTotals`                               | vatAmount=0.74, total=11.24                                                                                                                         |
| TC-FE-CALC-04 | AC3 (mirror)              | error            | discount > subtotal                                                                                          | `calcTotals`                               | error==='DISCOUNT_EXCEEDS_SUBTOTAL', vatAmount=null, total=null                                                                                     |
| TC-FE-CALC-05 | AC4 (mirror)              | edge             | discount === subtotal เป๊ะ                                                                                   | `calcTotals`                               | vatAmount=0, total=0, error=null                                                                                                                    |
| TC-FE-FORM-01 | AC14                      | happy            | กรอกฟอร์ม 2 แถว + discount เหมือน AC1                                                                        | เรนเดอร์ real-time (ไม่ยิง API)            | summary แสดง "2,751.50"/"151.50"/"182.00"/"2,782.00"                                                                                                |
| TC-FE-FORM-02 | AC15                      | error            | กรอก discount > subtotal                                                                                     | เรนเดอร์                                   | `discount-error` ปรากฏ, ปุ่ม submit `disabled`                                                                                                      |
| TC-FE-FORM-03 | AC16                      | edge             | โหลดฟอร์มแก้ไข quotation ที่ status="sent" (mock GET)                                                        | เรนเดอร์                                   | ฟิลด์ attention/discount `disabled`, ไม่มีปุ่ม Save/Delete                                                                                          |
| TC-FE-LIST-01 | AC17                      | happy            | list ว่างตอนแรก (MSW), submit สร้างใหม่สำเร็จ (201)                                                          | สร้างแล้วกลับหน้า list                     | list แสดง reference_no ใหม่ (มาจาก refetch หลัง invalidate)                                                                                         |
| TC-FE-LIST-02 | list-query.md             | happy            | คลิกเปลี่ยนหน้า/sort ในตาราง                                                                                 | ตรวจ request ที่ MSW จับได้                | query string ที่ยิงจริงมี `page`/`sort` ตรงกับ state ที่ผู้ใช้ตั้ง                                                                                  |

รวม **44 test case** ครอบ AC1–AC17, AC19 + Decision #3/#8/#9 (ที่ plan ระบุว่า "ต้องเทส" แม้ไม่ใช่ AC number ตรง ๆ)

---

## 2. Scope ที่ระบุชัดว่า "ไม่ครอบด้วย automated test" (ไม่ใช่ silent descope)

- **AC18** (ห้าม log PII ระดับ info): แผนนี้ **ไม่มี task ใหม่ที่เพิ่ม log call** สำหรับ quotation feature
  (ดู Tasks ในแผน — ไม่มีข้อไหนสั่งให้เพิ่ม logging เฉพาะ quotation) มิดเดิลแวร์ที่มีอยู่แล้ว
  (`RequestLogger`, `Recovery` — อ่านแล้วยืนยันด้านบน) log แค่ `request_id`/`method`/`path`/`status`/
  `latency` หรือ panic value เท่านั้น ไม่แตะ body/PII อยู่แล้ว และมี test ของมิดเดิลแวร์เหล่านั้นอยู่แล้ว
  (นอก scope slug นี้) จึงไม่เพิ่ม test ใหม่ซ้ำ — **ความเสี่ยงที่เหลือ**: ถ้า dev เพิ่ม `log`/`fmt.Println`
  เอง ๆ ใน `quotation_service.go`/`quotation_handler.go` ที่พิมพ์ email/telephone/customer_signee_name
  จะไม่มี test จับได้อัตโนมัติ — แนะนำให้ **qa-tester ตรวจด้วยสายตา/grep** ไฟล์ใหม่ทั้งหมดหา
  `log.`/`fmt.Print`/`logger.` ที่มีตัวแปร email/telephone/signee ก่อนอนุมัติ PASS
- **TC-REPO-\* (integration)**: เขียนและ compile-verify ได้ แต่ **รันจริงกับ Postgres ไม่ได้ในเครื่องนี้**
  เพราะ Docker Desktop engine ไม่ทำงาน (`docker ps` error: `dockerDesktopLinuxEngine` pipe ไม่พบ) —
  ดูหัวข้อ "ผลรัน RED" ด้านล่างสำหรับรายละเอียด และคำแนะนำให้ dev/CI รันซ้ำในเครื่อง/pipeline ที่มี
  Docker ทำงานจริงก่อนเชื่อว่า integration test เขียวจริง

---

## 3. Prompt ที่จะส่งให้ qwen (`claude-9arm`) — สรุปย่อ (รายละเอียดเต็มอยู่ในคำสั่งที่รันจริง)

แบ่งเป็น 4 รอบ (backend calc/service, backend handler, backend repository, frontend) แต่ละรอบใส่:
absolute path ของ `docs/plans/quotation-crud.md`, ไฟล์นี้ (`docs/tests/quotation-crud-testcases.md`),
`.claude/docs/{api-response,list-query,testing}.md`, `.claude/rules/{backend,frontend,naming-conventions}.md`,
ตัวอย่างไฟล์ pattern ที่มีอยู่แล้ว (`internal/service/profile_service_test.go`,
`internal/handler/me_handler_test.go`, `internal/service/mocks_test.go`,
`frontend/src/features/auth/pages/ProfilePage.test.jsx`) และย้ำ:
"เขียนเฉพาะไฟล์ test ห้ามเขียน production code, ห้าม `t.Skip`/`it.skip`/`it.todo` (ยกเว้น
`testing.Short()` guard ตามรูปแบบใน testing.md ข้อ 3), assert ค่าตัวเลข/field name ตามตารางในเอกสารนี้เป๊ะ
ห้ามลดทอนความเข้ม (ห้าม assert.NotNil แทนค่าจริง)"

---

## 4. ผลรัน RED (ยืนยันแล้วโดย test-case-writer หลัง review + แก้ไข)

### Backend

| Package                                                                                               | คำสั่ง                                            | ผล                                                                                                                                                               |
| ----------------------------------------------------------------------------------------------------- | ------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/service` (`quotation_calc_test.go`, `quotation_mocks_test.go`, `quotation_service_test.go`) | `gofmt -l .` แล้ว `go vet ./internal/service/...` | gofmt สะอาด; vet fail ที่ `undefined: repository.QuotationRepository` (compile fail เพราะยังไม่มี production code — RED ถูกต้อง)                                 |
| `internal/handler` (`quotation_handler_test.go`)                                                      | `go vet ./internal/handler/...`                   | fail ที่ `undefined: dto.CreateQuotationRequest` — RED ถูกต้อง                                                                                                   |
| `internal/repository` (`quotation_repository_test.go`)                                                | `go vet ./internal/repository/...`                | fail ที่ `undefined: model.Quotation` — RED ถูกต้อง                                                                                                              |
| ทั้ง repo                                                                                             | `gofmt -l .` (root) + `go vet ./...`              | ไม่มีไฟล์ format ผิด; error เฉพาะ 3 package ข้างบนเท่านั้น package อื่น (`config`,`middleware`,`model`,`pkg/response`,`router`,`cmd`) ผ่านหมด — ไม่มี regression |

**บั๊กที่พบระหว่าง review แล้วแก้เอง (ก่อนส่งต่อ dev):**

1. `quotation_repository_test.go` ที่ qwen เขียนครั้งแรก **import cycle** (`internal/repository` test import
   `internal/service` ซึ่ง `internal/service` import `internal/repository` อยู่แล้ว) — แก้โดยเปลี่ยนสัญญาให้
   repository ใช้ sentinel ของตัวเอง (`ErrDuplicateReferenceNo`) และ `gorm.ErrRecordNotFound` แทน
   `service.ErrConflict`/`service.ErrNotFound` (ตรงกับ pattern ที่มีอยู่แล้วใน `gormUserRepository`) —
   ปรับ `quotation_service_test.go` (TC-SVC-02/03) ให้ mock ค่า conflict เป็น `repository.ErrDuplicateReferenceNo`
   ให้สอดคล้องกัน (ดูข้อ 0.2 ที่แก้ไขแล้วด้านบน)
2. import alias ชนกัน: `github.com/testcontainers/testcontainers-go/modules/postgres` กับ
   `gorm.io/driver/postgres` ใช้ชื่อ package `postgres` เหมือนกัน — แก้ด้วย alias `tcpostgres`
3. `testContainer.Host(ctx)` ถูกเรียกผิด signature (คืนแค่ host, ไม่คืน port) — แก้ให้เรียก
   `MappedPort(ctx, "5432/tcp")` แยก
4. import `"strings"` ที่ไม่ได้ใช้ในไฟล์ repository test — ลบออก

### Frontend

| ไฟล์                                                                                | คำสั่ง                        | ผล                                                                                                                                                                                                   |
| ----------------------------------------------------------------------------------- | ----------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `calcQuotation.test.js`, `QuotationFormPage.test.jsx`, `QuotationListPage.test.jsx` | `npx eslint <3 ไฟล์>`         | ผ่าน ไม่มี warning/error                                                                                                                                                                             |
| ทั้ง 3 ไฟล์                                                                         | `npx vitest run` (ทั้ง suite) | **3 failed, 6 passed (9 ไฟล์)**, **13 tests เดิมผ่านหมด (ไม่มี regression)**; 3 ไฟล์ใหม่ fail ด้วย `Failed to resolve import "@/features/quotation/..."` — RED ถูกต้อง (component/util ยังไม่มีจริง) |

**บั๊กที่พบระหว่าง review แล้วแก้เอง:**

1. **ตัวอักษรเพี้ยน (encoding corruption จาก qwen):** regex ที่ควรเป็นข้อความไทย "เพิ่มรายการ" ถูกเขียนผิดเป็น
   "เพิ่מרายการ" (มีตัวอักษรฮีบรู מ/ר ปนอยู่) ใน `QuotationFormPage.test.jsx` และ `QuotationListPage.test.jsx`
   — ถ้าไม่แก้ regex จะไม่ match ข้อความไทยจริงที่ dev เขียน ทำให้ test fail แม้ dev implement ถูกต้อง — แก้เป็น
   "เพิ่มรายการ" ที่ถูกต้องแล้ว
2. **selector เปราะ (fragile, ผูก DOM ภายใน):** `row.querySelector('input')` (เลือก input ตัวแรกในแถวแบบเดา
   ตำแหน่ง DOM) แทนที่จะใช้ `getAllByLabelText(/unit price/i)` ตามสัญญาที่ระบุไว้ — แก้ให้ใช้ label ตามสัญญา
3. **`if (...)` ครอบการกรอกฟอร์ม + `try/catch` กลืน error เงียบ ๆ**: ทำให้ test อาจ "ผ่านลวง" แม้ field ที่ควร
   กรอกจริงไม่ถูกเจอ/ไม่ถูกกรอก (ขัดกับกฎ "ไม่มี logic ในตัว test" และเสี่ยง false-positive) — แก้เป็นเรียก
   `getByLabelText(...)` ตรง ๆ ไม่มี branching
4. **wait ที่หลวมเกินไป**: `findByText(/no quotations|empty|loading|data|.../i)` ใช้รอ "อะไรก็ได้ที่มีคำว่า
   data" ซึ่งแทบจะ match ได้เสมอ (ไม่ได้ยืนยันว่า request ที่แท้จริงเกิดขึ้นจริง) — แก้เป็น
   `waitFor(() => expect(capturedURL).not.toBe(''))` เพื่อรอเงื่อนไขที่ต้องการจริง ๆ

**สรุปรวม:** 44 test case ตามที่ออกแบบ, ยืนยัน RED ครบทุกไฟล์ (compile/resolve fail ด้วยเหตุผลที่ถูกต้อง
คือ "ยังไม่มี production code" ไม่ใช่ typo/syntax error), ไม่มี test ใดถูกทำให้ "ผ่านลวง"
