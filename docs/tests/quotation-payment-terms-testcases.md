# Test Cases: Payment Terms สำหรับใบเสนอราคา (slug: quotation-payment-terms)

อ้างอิงแผน: `docs/plans/quotation-payment-terms.md`

> ต่อยอด `quotation-crud` — **ห้ามแก้** `backend/internal/service/quotation_calc_test.go`,
> `quotation_service_test.go`, `quotation_mocks_test.go`, `mocks_test.go`,
> `backend/internal/handler/quotation_handler_test.go`,
> `frontend/.../calcQuotation.test.js`, `frontend/.../QuotationFormPage.test.jsx`
> **ยกเว้นข้อเดียว** (จำเป็นทางเทคนิค): `backend/internal/repository/quotation_repository_test.go`
> `TestMain` ต้องเพิ่ม `&model.PaymentTerm{}` เข้า `AutoMigrate(...)` (บรรทัดเดียว, additive เท่านั้น)
> มิฉะนั้นตาราง `payment_terms` จะไม่ถูกสร้างและ integration test payment-terms ใหม่จะรันไม่ได้ —
> ห้ามแก้ส่วนอื่นของไฟล์นั้น

Base fixture ที่ใช้ร่วมกันหลายเคส (ตรงกับฐานตัวอย่างใน plan):

- 2 items: `unit_price=1000.00,qty=2` (line 2000.00) + `unit_price=500.00,qty=1` (line 500.00)
  → `subtotal=2500.00`, `discount=0` → `base=2500.00` → `vat=175.00` → **`total=2675.00`** (`totalCents=267500`)
- payment terms 3 งวดตรง: `[891.67, 891.67, 891.66]` → cents `[89167,89167,89166]` sum `267500` = `totalCents` ✅
- payment terms mismatch: `[1000.00, 1000.00, 1000.00]` → sum `300000 ≠ 267500` ❌
- float-precision case (AC2, ใช้ total อิสระ 100.00): `[33.33, 33.33, 33.34]` → cents `[3333,3333,3334]` sum `10000` = `totalCents(100.00)` ✅

---

## 1. Test case table

| ID               | อ้างอิง AC            | ประเภท                           | Given                                                                                        | When                                                                                          | Then                                                                                                                                                                                                                          |
| ---------------- | --------------------- | -------------------------------- | -------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| TC-CALC-PT-01    | AC1                   | happy                            | totalCents=267500                                                                            | `validatePaymentTermsCents([891.67,891.67,891.66], 267500)`                                   | nil error                                                                                                                                                                                                                     |
| TC-CALC-PT-02    | AC2                   | edge (float precision)           | totalCents=10000                                                                             | `validatePaymentTermsCents([33.33,33.33,33.34], 10000)`                                       | nil error (ห้าม fail เพราะ float compare ตรง ๆ)                                                                                                                                                                               |
| TC-CALC-PT-03    | AC3                   | error                            | totalCents=267500                                                                            | `validatePaymentTermsCents([1000,1000,1000], 267500)`                                         | `ErrorIs(ErrValidation)`                                                                                                                                                                                                      |
| TC-CALC-PT-04    | AC5                   | edge (optional/empty)            | totalCents=267500                                                                            | `validatePaymentTermsCents(nil, 267500)` และ `validatePaymentTermsCents([]float64{}, 267500)` | ทั้งคู่ nil error                                                                                                                                                                                                             |
| TC-CALC-PT-05    | AC3 (defense)         | edge (off-by-one satang)         | totalCents=267500                                                                            | `validatePaymentTermsCents([891.68,891.67,891.66], 267500)` (sum=267501)                      | `ErrorIs(ErrValidation)` — พิสูจน์ว่า compare แบบ exact ไม่ใช่ tolerance                                                                                                                                                      |
| TC-SVC-PT-01     | AC1                   | happy                            | Create, 2 items ฐาน + terms `[891.67,891.67,891.66]` desc `Deposit/Progress/Final`           | `CreateQuotation`                                                                             | 200-level (no err); `resp.PaymentTerms` len 3, `TermNo` 1,2,3 เรียงตามลำดับ ส่ง, `SortOrder` เท่ากับ `TermNo`, `Description`/`Amount` ตรง; repo.Create ถูกเรียกด้วย `q.PaymentTerms` ที่มีค่าเดียวกัน (ผ่าน `mock.MatchedBy`) |
| TC-SVC-PT-02     | AC3                   | error                            | Create, terms mismatch `[1000,1000,1000]` (sum 3000 ≠ total 2675)                            | `CreateQuotation`                                                                             | `ErrorIs(ErrValidation)`; `repo.AssertNotCalled("Create", ...)`                                                                                                                                                               |
| TC-SVC-PT-03     | AC5                   | edge                             | Create, `PaymentTerms` ไม่ส่ง (nil)                                                          | `CreateQuotation`                                                                             | no error; `resp.PaymentTerms` ไม่ nil และ `len == 0`; `repo.Create` ถูกเรียกปกติ                                                                                                                                              |
| TC-SVC-PT-04     | AC6 (via #5 decision) | error                            | Update, existing `Status="sent"`, req มี `PaymentTerms` ไม่ว่าง                              | `UpdateQuotation`                                                                             | `ErrorIs(ErrForbidden)`; `repo.AssertNotCalled("Update", ...)`                                                                                                                                                                |
| TC-SVC-PT-05     | AC1                   | happy                            | Update, existing `Status="draft"`, terms ตรง `[891.67,891.67,891.66]`                        | `UpdateQuotation`                                                                             | no error; `resp.PaymentTerms` len 3 TermNo 1,2,3; repo.Update ถูกเรียกด้วย `q.PaymentTerms` ตรงตามที่ build (mock.MatchedBy)                                                                                                  |
| TC-SVC-PT-06     | AC3                   | error                            | Update, existing `Status="draft"`, terms mismatch                                            | `UpdateQuotation`                                                                             | `ErrorIs(ErrValidation)`; `repo.AssertNotCalled("Update", ...)`                                                                                                                                                               |
| TC-HDL-PT-01     | AC4                   | error                            | POST /quotations, `payment_terms=[{description:"Bad",amount:0}]`                             | binding จริงผ่าน `ShouldBindJSON`                                                             | `400`, `error.code=="VALIDATION_ERROR"`; `svc.AssertNotCalled("CreateQuotation", ...)`                                                                                                                                        |
| TC-HDL-PT-02     | AC4                   | error                            | POST /quotations, `payment_terms=[{description:"Bad",amount:-50}]`                           | binding จริง                                                                                  | `400 VALIDATION_ERROR`; `svc.AssertNotCalled("CreateQuotation", ...)`                                                                                                                                                         |
| TC-HDL-PT-03     | AC1                   | happy                            | POST /quotations, svc mock คืน `QuotationResponse` ที่มี `PaymentTerms` 3 รายการ             | `Create`                                                                                      | `201`; body `data.payment_terms` เป็น array len 3; `data.payment_terms[0].term_no==1`, `.description=="Deposit"`, `.amount==891.67`, `.sort_order==1` (สัญญา json tag snake_case)                                             |
| TC-REPO-PT-01    | AC1/Decision#7        | happy (integration, ต้อง Docker) | Create quotation พร้อม 3 payment terms (term_no/sort_order 1,2,3)                            | `FindByID`                                                                                    | `got.PaymentTerms` len 3, เรียงตาม `sort_order asc` ตรงกับที่ insert                                                                                                                                                          |
| TC-REPO-PT-02    | Decision#6            | happy (integration)              | Create ด้วย 2 payment terms, แล้ว `Update` ด้วย payment terms ชุดใหม่ 1 รายการ               | `Update` แล้ว `FindByID`                                                                      | เหลือ payment term เดียวตามชุดใหม่ (ของเดิม 2 รายการหายไปทั้งหมด — full replace)                                                                                                                                              |
| TC-REPO-PT-03    | Decision#8/cascade    | happy (integration)              | Create พร้อม payment terms                                                                   | `Delete` quotation                                                                            | `COUNT(payment_terms WHERE quotation_id=?) == 0`                                                                                                                                                                              |
| TC-FE-CALC-PT-01 | AC1                   | happy                            | `paymentTermsSumMatchesTotal([{amount:891.67},{amount:891.67},{amount:891.66}], 2675.00)`    | เรียกฟังก์ชัน                                                                                 | `true`                                                                                                                                                                                                                        |
| TC-FE-CALC-PT-02 | AC2                   | edge (float)                     | `paymentTermsSumMatchesTotal([{amount:33.33},{amount:33.33},{amount:33.34}], 100.00)`        | เรียกฟังก์ชัน                                                                                 | `true`                                                                                                                                                                                                                        |
| TC-FE-CALC-PT-03 | AC3                   | error                            | `paymentTermsSumMatchesTotal([{amount:1000},{amount:1000},{amount:1000}], 2675.00)`          | เรียกฟังก์ชัน                                                                                 | `false`                                                                                                                                                                                                                       |
| TC-FE-CALC-PT-04 | AC5                   | edge                             | `paymentTermsSumMatchesTotal([], 2675.00)` และ `paymentTermsSumMatchesTotal(undefined, 100)` | เรียกฟังก์ชัน                                                                                 | ทั้งคู่ `true`                                                                                                                                                                                                                |
| TC-FE-CALC-PT-05 | support               | happy                            | `roundHalfUpCents` ต้องถูก `export`                                                          | `roundHalfUpCents(1000.00)`, `roundHalfUpCents(0.006)`                                        | `100000`, `1`                                                                                                                                                                                                                 |
| TC-FE-FORM-PT-01 | AC7                   | happy                            | กรอก 2 items ฐาน (total 2675.00) + 3 payment terms ตรง (891.67/891.67/891.66)                | เติมฟอร์ม                                                                                     | ไม่มี `payment-terms-warning`; ปุ่ม Save ไม่ disabled; `payment-terms-sum` แสดง `2,675.00`                                                                                                                                    |
| TC-FE-FORM-PT-02 | AC7                   | error                            | เหมือนบน แต่ payment terms รวม `2000.00` (1000+1000) ≠ total 2675.00                         | เติมฟอร์ม                                                                                     | ปรากฏ `payment-terms-warning`; ปุ่ม Save `disabled`                                                                                                                                                                           |
| TC-FE-FORM-PT-03 | AC8                   | error/edge                       | โหลด quotation แก้ไข `status="sent"` มี 1 payment term (`Deposit`, `891.67`)                 | render edit mode                                                                              | `payment-term-description-0` และ `payment-term-amount-0` เป็น `disabled`; ปุ่ม Add Term (`/add term                                                                                                                           | เพิ่มงวด/i`) เป็น `disabled` |

---

## 2. สัญญา API ที่ dev ต้องสร้าง (ให้ qwen พิมพ์ test อ้างตามนี้เป๊ะ)

### Backend

```go
// backend/internal/model/quotation.go
type PaymentTerm struct {
    ID          uint
    QuotationID uint
    TermNo      int
    Description string
    Amount      float64
    SortOrder   int
}
// Quotation gets a new field:
//   PaymentTerms []PaymentTerm `gorm:"foreignKey:QuotationID;constraint:OnDelete:CASCADE"`

// backend/internal/dto/quotation_dto.go
type PaymentTermInput struct {
    Description string  `json:"description" binding:"required"`
    Amount      float64 `json:"amount" binding:"gt=0"`
}
// CreateQuotationRequest gets: PaymentTerms []PaymentTermInput `json:"payment_terms" binding:"omitempty,dive"`
// UpdateQuotationRequest = CreateQuotationRequest (unchanged alias, gets field automatically)
type PaymentTermResponse struct {
    ID          uint    `json:"id"`
    TermNo      int     `json:"term_no"`
    Description string  `json:"description"`
    Amount      float64 `json:"amount"`
    SortOrder   int     `json:"sort_order"`
}
// QuotationResponse gets: PaymentTerms []PaymentTermResponse `json:"payment_terms"`

// backend/internal/service/quotation_calc.go (package service, unexported — tests in same package call directly)
func validatePaymentTermsCents(termAmounts []float64, totalCents int64) error
```

> หมายเหตุสำคัญให้ qwen: `binding:"omitempty,dive"` (ไม่ใช่แค่ไม่มี tag เลย) — ต้องมี `dive` เพื่อให้
> go-playground/validator ตรวจ tag ภายใน `PaymentTermInput` แต่ละตัว (`gt=0`, `required`) มิฉะนั้น
> TC-HDL-PT-01/02 (AC4) จะไม่มีวันล้มเหลวที่ binding และ RED จะผิดจุด — **ถ้า dev ไม่ใส่ dive ให้ถือว่า
> คือบั๊กที่ dev ต้องแก้ ไม่ใช่ให้ทดสอบอ่อนลง**

### Frontend

```js
// frontend/src/features/quotation/utils/calcQuotation.js
export function roundHalfUpCents(amount) // เปลี่ยนจาก private -> export (ของเดิม ห้ามเปลี่ยน logic)
export function paymentTermsSumMatchesTotal(terms, total)
// terms: {amount:number}[] | undefined | null ; ว่าง/undefined => true เสมอ (optional feature)

// frontend/src/features/quotation/pages/QuotationFormPage.jsx เพิ่ม:
//   - defaultValues.payment_terms = []
//   - แถวละ data-testid="payment-term-row"
//   - input: id=`payment-term-description-${index}`, id=`payment-term-amount-${index}` (number)
//   - แต่ละ input ต้องมี <label htmlFor=...> คู่กัน โดยใช้ข้อความ "Term Description" และ
//     "Term Amount" (ไม่ใช่แค่ "Description"/"Amount" เฉย ๆ) — เพราะฟอร์มมี item rows ที่มี label
//     "Description" อยู่แล้วหลายแถว ถ้าใช้ label ซ้ำจะ query ด้วย getByLabelText ไม่ได้ (ambiguous)
//   - ปุ่ม accessible name matching /add term|เพิ่มงวด/i
//   - data-testid="payment-terms-sum" (ผลรวมงวด, format เหมือน summary-* ด้วย toLocaleString 2 ตำแหน่ง)
//   - data-testid="payment-terms-warning" เมื่อ mismatch && length>0
//   - ทุก input/ปุ่มในส่วนนี้ disabled={isLocked} เหมือน field อื่น
```

---

## 3. AC coverage summary

| AC                                               | Covered by                                                                                                 |
| ------------------------------------------------ | ---------------------------------------------------------------------------------------------------------- |
| AC1 (happy sum, response 3 terms, term_no 1,2,3) | TC-CALC-PT-01, TC-SVC-PT-01, TC-SVC-PT-05, TC-HDL-PT-03, TC-REPO-PT-01, TC-FE-CALC-PT-01, TC-FE-FORM-PT-01 |
| AC2 (float precision)                            | TC-CALC-PT-02, TC-FE-CALC-PT-02                                                                            |
| AC3 (mismatch → 400 VALIDATION_ERROR)            | TC-CALC-PT-03, TC-CALC-PT-05, TC-SVC-PT-02, TC-SVC-PT-06, TC-FE-CALC-PT-03, TC-FE-FORM-PT-02               |
| AC4 (amount ≤ 0 → 400)                           | TC-HDL-PT-01, TC-HDL-PT-02                                                                                 |
| AC5 (ไม่มีงวด = ผ่านปกติ)                        | TC-CALC-PT-04, TC-SVC-PT-03, TC-FE-CALC-PT-04                                                              |
| AC6 (non-draft แก้ไม่ได้)                        | TC-SVC-PT-04                                                                                               |
| AC7 (frontend warning + disable submit)          | TC-FE-FORM-PT-01, TC-FE-FORM-PT-02                                                                         |
| AC8 (frontend non-draft disable)                 | TC-FE-FORM-PT-03                                                                                           |
| Decision#6 (full-replace update)                 | TC-REPO-PT-02                                                                                              |
| Decision#7 (preload order)                       | TC-REPO-PT-01                                                                                              |
| Cascade delete                                   | TC-REPO-PT-03                                                                                              |

**ไม่มี AC ใด descope** — ครอบครบทุกข้อตาม plan ยกเว้นหมายเหตุ: TC-REPO-PT-01..03 เป็น integration test ที่ต้องใช้
Docker (testcontainers) ซึ่งเครื่องพัฒนานี้ไม่มี Docker → verify ได้แค่ `go vet`/`go build` (compile-level RED)
ในรอบนี้ ส่วนการรันจริงจะเกิดใน CI ตามที่ main agent ระบุไว้แต่แรก
