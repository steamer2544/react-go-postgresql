# Plan: Payment Terms สำหรับใบเสนอราคา (slug: quotation-payment-terms)

> ต่อยอดจาก `quotation-crud` (commit 25082a6) — ไม่สร้าง endpoint ใหม่, ไม่แก้ calc engine เดิม
> เพิ่มเฉพาะ: model `PaymentTerm`, ฝัง payload ใน create/update quotation เดิม (full-replace เหมือน `Items`),
> validation ผลรวมด้วย integer cents (`int64`) ตาม pattern `quotation_calc.go` ที่มีอยู่แล้ว

## เป้าหมาย / Definition of Done

- ใบเสนอราคา 1 ใบ แบ่งงวดชำระเงินได้ N งวด (`PaymentTerm`), ผูกกับ quotation ผ่าน FK `quotation_id`
- ผลรวม `amount` ของทุกงวด ต้อง**เท่ากับ** `total` ของ quotation แบบ exact (เทียบเป็น satang/int64 กันปัญหา float) มิฉะนั้น 400 `VALIDATION_ERROR`
- แก้ไข payment terms ได้เฉพาะตอน quotation `status = 'draft'` (reuse กติกาเดิมของ `UpdateQuotation`)
- ฟอร์ม quotation (frontend) มีส่วน Payment Term: เพิ่ม/ลบงวด, กรอกยอดต่องวด, โชว์ผลรวม vs total พร้อมเตือนถ้าไม่ตรง
- ออกแบบ schema ให้ `PaymentTerm.ID` เป็น stable PK ที่ future `invoice` table (slug 5) จะอ้างอิงได้ (ไม่ต้องสร้างอะไรเพิ่มตอนนี้)
- `cd backend && gofmt -l . && go vet ./... && go build ./... && go test ./...` ผ่าน
- `cd frontend && npm run lint && npm run build && npm test` ผ่าน

## ขอบเขต

- **Backend**: migration ใหม่ (`000004_create_payment_terms`), `model.PaymentTerm` + association บน `model.Quotation`,
  `dto` เพิ่ม field ใน request/response เดิม (ไม่สร้าง DTO endpoint ใหม่), `service/quotation_calc.go` เพิ่ม helper validate,
  `service/quotation_service.go` wire เข้า Create/Update, `repository/quotation_repository.go` preload + full-replace เหมือน Items
  **ไม่แก้ handler/router** (ไม่มี endpoint ใหม่)
- **Frontend**: `QuotationFormPage.jsx` เพิ่มส่วน Payment Term, `utils/calcQuotation.js` เพิ่ม helper คำนวณ/เทียบผลรวม
  **ไม่แก้** `quotationService.js` / hooks (payload เดิมส่ง object เดียวผ่านอยู่แล้ว แค่เพิ่ม field)
- **Database**: ตารางใหม่ `payment_terms` (FK `quotations.id` ON DELETE CASCADE)

## การตัดสินใจที่พินไว้ (กัน qwen เดา)

1. **ไม่มี endpoint แยก** — `payment_terms` เป็น field ใหม่ใน `CreateQuotationRequest`/`UpdateQuotationRequest` เดิม
   (`json:"payment_terms"`, เป็น `[]PaymentTermInput`, **optional** — ไม่ส่งมาหรือส่ง `[]` = ไม่มีงวด, ข้าม validation ผลรวม)
   เหตุผล: `UpdateQuotationRequest = CreateQuotationRequest` (full-replace PUT) อยู่แล้ว และ items ก็ฝังแบบเดียวกัน
   — ทำตาม pattern เดิมเพื่อไม่ต้องเปิด endpoint ใหม่/ทรานแซกชันแยก

2. **term_no และ sort_order ไม่รับจาก client** — server **derive จากลำดับใน array ที่ส่งมา** เสมอ (index+1)
   `PaymentTermInput` มีแค่ `description` + `amount` (ไม่มี `term_no`/`sort_order` ใน input)
   เหตุผล: term_no มีความหมายทางธุรกิจ (งวด 1 ต้องมาก่อนงวด 2 จริง ๆ ตามลำดับจ่ายเงิน) ต่างจาก `QuotationItem.SortOrder`
   ที่แค่จัดเรียงการแสดงผล — ถ้าปล่อยให้ client ส่ง term_no เอง จะเกิดช่องโหว่ค่าไม่ต่อเนื่อง/ซ้ำที่ต้อง validate เพิ่ม
   การ derive จาก index ตัดปัญหานี้ทั้งหมด และง่ายต่อการ implement/ทดสอบ

3. **Validation ผลรวมใช้ integer satang (`int64`)** — ไม่เทียบ float ตรง ๆ
   - reuse `roundHalfUpCents(amount float64) int64` ที่มีอยู่แล้วใน `backend/internal/service/quotation_calc.go`
   - เพิ่มฟังก์ชันใหม่ในไฟล์เดียวกัน:
     ```go
     // validatePaymentTermsCents checks that payment-term amounts sum exactly to
     // totalCents (in satang). Empty/nil termAmounts is always valid (feature is
     // optional — a quotation may have zero payment terms).
     func validatePaymentTermsCents(termAmounts []float64, totalCents int64) error {
         if len(termAmounts) == 0 {
             return nil
         }
         var sumCents int64
         for _, a := range termAmounts {
             sumCents += roundHalfUpCents(a)
         }
         if sumCents != totalCents {
             return ErrValidation
         }
         return nil
     }
     ```
   - ตัวอย่างที่ **ต้องผ่าน**: `33.33 + 33.33 + 33.34 = 100.00` → cents `3333+3333+3334 = 10000` ✅ (ตรงกับ `totalCents=10000`)
     ถ้าเทียบด้วย float ตรง ๆ (`33.33+33.33+33.34 !== 100` ใน floating point) จะพังข้อนี้ — ห้ามทำแบบนั้นเด็ดขาด
   - เรียก `validatePaymentTermsCents` **หลัง** `calcTotals(...)` คืนค่า `totalCents` สำเร็จ (ก่อน snapshot signee /
     generate reference no) ทั้งใน `CreateQuotation` และ `UpdateQuotation` — fail fast แบบเดียวกับ discount validation เดิม

4. **แต่ละงวด `amount` ต้อง > 0 เท่านั้น** (ห้าม 0 หรือติดลบ)
   `PaymentTermInput.Amount float64 \`json:"amount" binding:"gt=0"\``(ไม่ใส่`required`เพราะ`gt=0`ตัดค่า 0 อยู่แล้ว
— ตัดปัญหา required-vs-zero-value ของ go-playground/validator ที่ทำให้`0`ผ่าน required ไม่ได้อยู่แล้วเป็นทุนเดิม)`PaymentTermInput.Description string \`json:"description" binding:"required"\``

5. **Draft-only reuse ของเดิมทั้งหมด** — `UpdateQuotation` เช็ค `existing.Status != "draft" → ErrForbidden` อยู่แล้วก่อนแตะ
   field ใด ๆ ในเพย์โหลด เพราะ payment_terms ฝังอยู่ใน payload เดียวกัน จึงถูกบล็อกโดยอัตโนมัติ — **ไม่ต้องเขียนเช็คซ้ำ**
   `CreateQuotation` ไม่ต้องเช็ค (สร้างใหม่ status เป็น `draft` เสมอ)

6. **Full-replace ใน Update** เหมือน `Items` ทุกประการ: ลบ `payment_terms` เดิมทั้งหมดของ quotation นั้นแล้ว insert ใหม่
   ในทรานแซกชันเดียวกับที่ replace items (ดู `repository.Update` ปัจจุบัน) — เพิ่ม `"PaymentTerms"` เข้า `Omit(...)` ตอน
   `Updates(q)` แบบเดียวกับที่ `"Items"` ถูก omit อยู่แล้ว (เพราะเป็น association ไม่ใช่คอลัมน์ตรงของตาราง `quotations`)

7. **Preload** — `FindByID` และ `List` เพิ่ม `Preload("PaymentTerms", order by sort_order asc)` คู่กับ `Preload("Items", ...)` ที่มีอยู่แล้ว

8. **Invoice foresight (ยังไม่ build)** — ไม่ต้องเพิ่ม field พิเศษใด ๆ ตอนนี้ `PaymentTerm.ID` (PK ปกติ) คือสิ่งที่
   future `invoice.payment_term_id` (slug 5) จะอ้างอิง — ไม่ต้องเผื่อ column เพิ่มรอบนี้ (YAGNI ตาม scope ที่ระบุ)

## ขอบเขต

- Frontend: `frontend/src/features/quotation/pages/QuotationFormPage.jsx`, `frontend/src/features/quotation/utils/calcQuotation.js`
- Backend: `backend/internal/model/quotation.go`, `backend/internal/dto/quotation_dto.go`,
  `backend/internal/service/quotation_calc.go`, `backend/internal/service/quotation_service.go`,
  `backend/internal/repository/quotation_repository.go`, `backend/migrations/000004_create_payment_terms.{up,down}.sql`
- Database: ตาราง `payment_terms` ใหม่

## Tasks (เรียงตาม dependency)

1. **[BE] Migration** — สร้าง `backend/migrations/000004_create_payment_terms.up.sql`:

   ```sql
   CREATE TABLE IF NOT EXISTS payment_terms (
       id BIGSERIAL PRIMARY KEY,
       quotation_id BIGINT NOT NULL REFERENCES quotations(id) ON DELETE CASCADE,
       term_no INTEGER NOT NULL,
       description TEXT,
       amount NUMERIC(12,2) NOT NULL,
       sort_order INTEGER NOT NULL DEFAULT 0,
       UNIQUE (quotation_id, term_no)
   );
   ```

   และ `000004_create_payment_terms.down.sql` (`DROP TABLE IF EXISTS payment_terms;`)

2. **[BE] Model** — `backend/internal/model/quotation.go`:
   - เพิ่ม struct `PaymentTerm` (field: `ID uint`, `QuotationID uint`, `TermNo int`, `Description string`, `Amount float64`, `SortOrder int`)
   - เพิ่ม field บน `Quotation`: `PaymentTerms []PaymentTerm \`gorm:"foreignKey:QuotationID;constraint:OnDelete:CASCADE"\``

3. **[BE] DTO** — `backend/internal/dto/quotation_dto.go`:
   - `PaymentTermInput { Description string \`json:"description" binding:"required"\`; Amount float64 \`json:"amount" binding:"gt=0"\` }`
   - เพิ่ม `PaymentTerms []PaymentTermInput \`json:"payment_terms"\``ใน`CreateQuotationRequest`(ไม่มี`binding:"required"`— optional)`UpdateQuotationRequest` ได้ field นี้อัตโนมัติ (type alias เดิม)
   - `PaymentTermResponse { ID uint; TermNo int; Description string; Amount float64; SortOrder int }` (json tags snake_case ตามเดิม)
   - เพิ่ม `PaymentTerms []PaymentTermResponse \`json:"payment_terms"\``ใน`QuotationResponse`

4. **[BE] Calc validation** — `backend/internal/service/quotation_calc.go`: เพิ่ม `validatePaymentTermsCents` (ดูโค้ดในหัวข้อ
   การตัดสินใจข้อ 3 ด้านบน — copy ไปใช้ตรง ๆ)

5. **[BE] Service** — `backend/internal/service/quotation_service.go`:
   - helper `buildPaymentTerms(terms []dto.PaymentTermInput) []model.PaymentTerm` — วน index `i`, ตั้ง
     `TermNo: i+1`, `SortOrder: i+1`, `Description: t.Description`, `Amount: t.Amount` (ไม่ round ตอนเก็บ — เก็บ
     ค่า major-unit ตรงจาก input เหมือนที่ `UnitPrice`/`DiscountAmount` เก็บแบบ float ปกติ, cents ใช้แค่ตอน validate)
   - ใน `CreateQuotation`: หลัง `calcTotals(...)` สำเร็จ (มี `totalCents` แล้ว) → ดึง `amounts := extract amount จาก req.PaymentTerms`
     → เรียก `validatePaymentTermsCents(amounts, totalCents)` → ถ้า error return ทันที (ก่อน snapshot user/generate ref no)
     → ตั้ง `candidate.PaymentTerms = buildPaymentTerms(req.PaymentTerms)`
   - ใน `UpdateQuotation`: จุดเดียวกัน (หลัง `calcTotals`, ก่อน parse customer signee date ก็ได้ — ขอแค่หลัง totalCents พร้อม)
     → validate → ตั้ง `q.PaymentTerms = buildPaymentTerms(req.PaymentTerms)`
   - `mapQuotationResponse`: เพิ่ม loop แปลง `q.PaymentTerms` → `[]dto.PaymentTermResponse` (เหมือน loop ของ Items ทุกประการ)

6. **[BE] Repository** — `backend/internal/repository/quotation_repository.go`:
   - `FindByID`: เพิ่ม `.Preload("PaymentTerms", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order asc") })` ต่อจาก Preload Items เดิม
   - `List`: เพิ่ม preload เดียวกัน
   - `Update`: หลังลบ/insert Items เดิม (ในทรานแซกชันเดียวกัน) เพิ่มขั้นตอนเดียวกันสำหรับ `PaymentTerms`:
     ลบ `payment_terms` เดิม (`tx.Where("quotation_id = ?", q.ID).Delete(&model.PaymentTerm{})`), ตั้ง `QuotationID`
     บนทุก element ของ `q.PaymentTerms`, แล้ว `tx.Create(&q.PaymentTerms)` ถ้า `len(q.PaymentTerms) > 0`
   - `Update`: เพิ่ม `"PaymentTerms"` เข้า `.Omit("Items", "CreatedAt")` → `.Omit("Items", "PaymentTerms", "CreatedAt")`
   - `Create`: ไม่ต้องแก้ (GORM auto-save nested association ตอน `tx.Create(q)` เหมือนที่ `Items` ทำงานอยู่แล้วตอนนี้)

7. **[FE] Calc util** — `frontend/src/features/quotation/utils/calcQuotation.js`: เพิ่ม (export ใหม่ ไม่แก้ของเดิม)

   ```js
   export function paymentTermsSumMatchesTotal(terms, total) {
     if (!terms || terms.length === 0) return true; // optional feature: no terms = always valid
     const sumCents = terms.reduce(
       (sum, t) => sum + roundHalfUpCents(Number(t.amount) || 0),
       0,
     );
     const totalCents = roundHalfUpCents(Number(total) || 0);
     return sumCents === totalCents;
   }
   ```

   (`roundHalfUpCents` มีอยู่แล้วในไฟล์นี้แต่ไม่ได้ export — ต้อง `export` เพิ่มเพื่อเรียกจากฟังก์ชันใหม่/จาก test ได้ ห้ามเขียน
   ลอจิกปัดเศษซ้ำเป็นชุดที่สอง)

8. **[FE] Form page** — `frontend/src/features/quotation/pages/QuotationFormPage.jsx`:
   - เพิ่ม default `payment_terms: []` ใน `useForm` defaultValues
   - `watchedPaymentTerms = watch('payment_terms')`
   - ส่วน UI ใหม่ (วางถัดจาก Summary block, ก่อน submit button):
     - หัวข้อ "Payment Terms"
     - loop `watchedPaymentTerms.map((term, index) => ...)` แต่ละแถว (`data-testid="payment-term-row"`):
       description input (`id="payment-term-description-{index}"`), amount input type number step 0.01
       (`id="payment-term-amount-{index}"`, `{...register(\`payment_terms.${index}.amount\`, { valueAsNumber: true })}`),
ปุ่มลบแถว (`disabled={isLocked}`)
     - ปุ่ม "Add Term" (`disabled={isLocked}`) — push `{ description: '', amount: 0 }` ด้วย `setValue` (pattern เดียวกับ `addItem`)
     - แสดงผลรวมงวด (`data-testid="payment-terms-sum"`) เทียบกับ `totals.total`
     - ถ้า `!paymentTermsSumMatchesTotal(watchedPaymentTerms, totals.total)` **และ** `watchedPaymentTerms.length > 0`
       → โชว์ `data-testid="payment-terms-warning"` ("ผลรวมงวดไม่เท่ากับยอดรวม")
   - `onSubmit`: เพิ่ม `payment_terms: (data.payment_terms || []).map((t) => ({ description: t.description, amount: Number(t.amount) || 0 }))`
     เข้า `payload`
   - ปุ่ม Save: เพิ่มเงื่อนไข disable เมื่อ payment terms mismatch (รวมกับเงื่อนไข `DISCOUNT_EXCEEDS_SUBTOTAL`/`isPending` เดิม)
   - ทุก input ในส่วนนี้ต้อง `disabled={isLocked}` เหมือน field อื่นในฟอร์ม (non-draft ล็อกทั้งฟอร์มอยู่แล้วผ่าน `isLocked` เดิม)

## Acceptance Criteria

ตัวอย่างฐาน: quotation มี 2 items — `unit_price=1000.00, qty=2` (line=2000.00) และ `unit_price=500.00, qty=1` (line=500.00)
→ `subtotal=2500.00`, `discount_amount=0` → `base=2500.00` → `vat_amount=175.00` (2500×0.07) → **`total=2675.00`**

- **AC1 (happy path, sum ตรง)**: ส่ง `payment_terms` 3 งวด `[891.67, 891.67, 891.66]` (รวม cents `89167+89167+89166=267500`
  = `totalCents 267500`) → POST/PUT ผ่าน (201/200), response มี `payment_terms` 3 รายการ `term_no` เรียง `1,2,3`
- **AC2 (float precision case ที่ห้ามพัง)**: total สมมติ `100.00`, งวด `[33.33, 33.33, 33.34]` → ผลรวม cents
  `3333+3333+3334=10000` ตรงกับ `totalCents=10000` → ต้องผ่าน (unit test ระดับ `validatePaymentTermsCents` /
  `paymentTermsSumMatchesTotal` โดยตรง ไม่ต้องพึ่ง quotation จริง)
  = **spec เดียวกับที่ main agent ระบุ: ห้ามเทียบ float ตรง ๆ**
- **AC3 (ผลรวมไม่ตรง)**: total `2675.00` เดิม แต่ส่งงวด `[1000.00, 1000.00, 1000.00]` (รวม `3000.00 ≠ 2675.00`)
  → 400 `VALIDATION_ERROR` ตาม `.claude/docs/api-response.md`, quotation **ไม่ถูกบันทึก/ไม่ถูกแก้**
- **AC4 (amount ≤ 0)**: งวดใดงวดหนึ่งส่ง `amount: 0` หรือ `amount: -50` → 400 `VALIDATION_ERROR` (fail ที่ binding
  ก่อนถึง service เลย เพราะ `binding:"gt=0"`)
- **AC5 (ไม่มีงวด = ผ่านปกติ)**: ไม่ส่ง `payment_terms` (หรือส่ง `[]`) → quotation บันทึกผ่านตามปกติเหมือนก่อนมีฟีเจอร์นี้
  (`payment_terms: []` ใน response), ไม่ trigger validation ผลรวมใด ๆ
- **AC6 (non-draft แก้ไม่ได้)**: quotation ที่ `status != 'draft'` (เช่น `sent`) → `PUT /quotations/:id` พร้อม
  `payment_terms` field ใด ๆ → 403 `FORBIDDEN` (reuse เช็คเดิมของ `UpdateQuotation`, ไม่ต้องเขียนโค้ดเพิ่ม) —
  payment terms เดิมในฐานข้อมูล**ไม่ถูกแตะ**
- **AC7 (frontend)**: กรอกฟอร์ม 3 งวดที่ผลรวม = total → ไม่มี `payment-terms-warning`, ปุ่ม Save ไม่ถูก disable
  โดย mismatch; ถ้าแก้ยอดงวดให้ผลรวม ≠ total → ปรากฏ `payment-terms-warning` และปุ่ม Save ถูก disable
- **AC8 (frontend, non-draft)**: โหลด quotation ที่ `status != 'draft'` เข้าฟอร์ม → input ของทุกแถว payment term
  (`payment-term-description-*`, `payment-term-amount-*`) และปุ่ม Add Term ต้อง `disabled`

## ความเสี่ยง / คำถามค้าง

- **payment terms เป็น optional ทั้งฟีเจอร์** (AC5) — โจทย์ไม่ได้ระบุชัดว่าบังคับทุกใบเสนอราคาต้องมีอย่างน้อย 1 งวด
  หรือไม่ ถ้าธุรกิจจริงต้องการบังคับ (เช่น "ทุกใบต้องมีอย่างน้อย 1 งวด") ต้องแจ้งกลับมาก่อน dev เพราะจะเปลี่ยน
  `binding` ของ field และ AC5 จะกลายเป็น error case แทน — **สมมติฐานตอนนี้: optional** (ยึดตามคำในโจทย์ที่พูดถึง
  แค่ validation เรื่องผลรวม ไม่ได้พูดถึง cardinality ขั้นต่ำ)
- **term_no/sort_order derive จาก index เท่านั้น** (การตัดสินใจข้อ 2) — ถ้าในอนาคตต้องการให้ผู้ใช้ลาก reorder
  งวดโดยที่ term_no ไม่เปลี่ยนตามลำดับการแสดงผล (เช่น sort_order ต่างจาก term_no) จะต้องออกแบบใหม่ตรงนี้ —
  นอกขอบเขต slug นี้
  ที่ยังไม่มี column เผื่อ ต้อง revisit ตอนนั้นเพื่อ derive ยอด/สถานะ term จาก invoice — นอกขอบเขต slug นี้ (ตามที่ระบุใน FEATURE_PROMPTS.md)
- migration status enum ปัจจุบัน (`000002_create_quotations.up.sql`) มีแค่ `draft/sent/approved/rejected` — ยังไม่มี
  `pending_approval` (จะมาใน slug 4 `quotation-approval`) — ไม่กระทบ slug นี้ แค่บันทึกไว้กันงง
- ยังไม่มี endpoint/DTO แยกสำหรับดึง payment terms ของ quotation เดี่ยว ๆ (เช่น `GET /quotations/:id/payment-terms`)
  เพราะข้อมูลฝังมาใน `GET /quotations/:id` อยู่แล้ว — ถ้า frontend ต้องการ endpoint ย่อยภายหลัง (เช่นหน้า invoice
  ที่อ้างอิง term เดียว) ต้องเปิด slug ใหม่แยก
