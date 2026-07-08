# Plan: Quotation Approval Workflow (slug: quotation-approval)

ต่อยอดจาก `quotation-crud` + `quotation-payment-terms` (merged, commit `6480018`). อ้างอิงโค้ดจริงที่อ่านแล้ว:
`backend/internal/{model,dto,service,repository,handler}/quotation*.go`, `router/router.go`,
`middleware/{auth,require_role}.go`, `service/{profile_service,token_service}.go`,
`migrations/000002-000004_*.sql`, `model/user.go`, `frontend/src/features/quotation/**`,
`frontend/src/features/auth/**`, `frontend/src/contexts/AuthContext.jsx`.

---

## เป้าหมาย / Definition of Done

เพิ่ม single-step approval state machine ให้ Quotation: `draft → pending_approval → approved | rejected`
พร้อม RBAC ต่อ transition, สแตมป์ลายเซ็นอัตโนมัติจากโปรไฟล์ผู้อนุมัติ (snapshot ณ เวลาอนุมัติ),
บังคับ immutability ของเอกสารที่ไม่ใช่ draft (ใช้ guard เดิมที่มีอยู่แล้ว), และเตรียม UI ปุ่ม/สถานะ/ส่วนแสดงลายเซ็น
โดยไม่ทำ invoice จริง (แค่บันทึกสถาปัตยกรรมที่ตั้งใจไว้ในเอกสาร ไม่แก้ schema — ดูเหตุผลในหัวข้อ Decision 6)

---

## Pinned Decisions (สำคัญ — กัน qwen เดา)

### 1. Status enum: เปลี่ยน `'sent'` → `'pending_approval'`

โค้ดปัจจุบัน (`migrations/000002_create_quotations.up.sql`, `dto.ListQuotationQuery.Status`,
FE `QuotationListPage.jsx` STATUS_OPTIONS) มีค่า `'sent'` อยู่ใน CHECK constraint/whitelist
แต่**ไม่มี logic ไหนตั้งค่านี้จริง** (`CreateQuotation` ตั้ง `"draft"` เท่านั้น, ไม่มี endpoint อื่นแก้ status) —
เป็นค่าตายที่ทิ้งไว้จากตอน scaffold ปลอดภัยที่จะเปลี่ยนชื่อ

**Transition table ที่ถูกต้องทั้งหมด:**

| จาก                | ไป                 | ใคร trigger                        | endpoint                       |
| ------------------ | ------------------ | ---------------------------------- | ------------------------------ |
| `draft`            | `pending_approval` | creator เจ้าของเอกสาร (หรือ admin) | `POST /quotations/:id/submit`  |
| `pending_approval` | `approved`         | role `approver` เท่านั้น           | `POST /quotations/:id/approve` |
| `pending_approval` | `rejected`         | role `approver` เท่านั้น           | `POST /quotations/:id/reject`  |

**`rejected` เป็น terminal state** (ไม่มี transition กลับไป `draft` หรือที่ไหนในสโคปนี้) — เหตุผล: โจทย์/AC ไม่ระบุ
resubmit flow, การเพิ่ม transition ที่ไม่ถูกทดสอบเสี่ยง over-engineer (YAGNI) ผู้ใช้ที่โดน reject
ต้องสร้างใบเสนอราคาใหม่ (draft ใหม่) แทน — ถ้าต้องการ resubmit ในอนาคตค่อยเปิด slug ใหม่ (ระบุไว้ใน "ความเสี่ยง")

**Edit/Delete ได้เฉพาะ `draft`** — ใช้ guard เดิมที่มีอยู่แล้วใน `QuotationService.UpdateQuotation`/`DeleteQuotation`
(`if existing.Status != "draft" { return ErrForbidden }`) **ไม่ต้องแก้โค้ดจุดนี้เลย** เพราะ guard ใช้ `!= "draft"`
อยู่แล้ว ครอบทั้ง `pending_approval`, `approved`, `rejected` โดยอัตโนมัติทันทีที่ status enum เปลี่ยน —
งานของ dev คือ**เพิ่ม regression test** ยืนยันว่ายังเป็นแบบนี้ (ดู AC6) ไม่ใช่แก้ logic

### 2. Error code ต่อ transition: แยก "ผิดสิทธิ์" (403) กับ "ผิดสถานะ" (409) ให้ชัด

- **ผิดสิทธิ์/ไม่ใช่เจ้าของ** → `ErrForbidden` → 403 `FORBIDDEN`
- **สิทธิ์ถูก แต่ status ปัจจุบันไม่ตรง precondition ของ transition** (เช่น submit ใบที่ไม่ใช่ draft,
  approve/reject ใบที่ไม่ใช่ pending_approval) → `ErrConflict` → 409 `CONFLICT`
  (สอดคล้องความหมายเดิมของ `ErrConflict` ในโปรเจกต์นี้ = "ชนกับสถานะ/ข้อมูลเดิม" ไม่ต้องเพิ่ม sentinel ใหม่)

> หมายเหตุ inconsistency ที่ทราบและ**ไม่แก้ในสโคปนี้**: guard เดิมของ Update/Delete (`status != draft`)
> ตอบ 403 ไม่ใช่ 409 — คงพฤติกรรมเดิมไว้ (ไม่มี test เดิมให้พังและไม่ใช่ scope คำขอนี้) ระบุไว้ใน "ความเสี่ยง"

### 3. RBAC ต่อ transition — asymmetric ตั้งใจ

- **Submit**: route group เดิม `middleware.RequireRole("admin", "creator")` (คือ `quotationsWrite` ที่มีอยู่แล้ว)
  - service ตรวจ ownership แบบเดียวกับ `UpdateQuotation`/`DeleteQuotation`:
    `if role != "admin" && existing.CreatedBy != userID { return ErrForbidden }`
    → admin bypass ownership ได้ (สอดคล้อง pattern เดิมของ CRUD ทั้งหมด)
- **Approve/Reject**: route group ใหม่ `middleware.RequireRole("approver")` **เท่านั้น — ไม่มี admin bypass**
  เหตุผล: โจทย์ระบุชัดเจน "approve/reject = role approver เท่านั้น" ต่างจาก CRUD ownership ที่ admin
  มีสิทธิ์บริหารจัดการเอกสารทุกใบอยู่แล้ว การอนุมัติเป็น**สิทธิ์แยกตามบทบาทเพื่อ segregation of duties**
  ไม่ใช่สิทธิ์บริหารเอกสาร — ถ้าต้องการให้ admin อนุมัติได้ด้วยในอนาคตต้องเป็นการตัดสินใจ business แยกต่างหาก
  (ระบุเป็นคำถามค้าง)
  - service **ตรวจ role ซ้ำอีกชั้น** (defense-in-depth ตาม `auth.md` §5: "ตรวจสิทธิ์ระดับข้อมูลทำใน service"):
    `if role != "approver" { return ErrForbidden }`

### 4. Approve: business rule เพิ่มเติมนอกเหนือ 4 AC เดิมในโจทย์ — "ผู้อนุมัติต้องมีลายเซ็นอัปโหลดไว้ก่อน"

โจทย์บอกให้สแตมป์ `signature_image + full_name + position + approved_at` จากโปรไฟล์ผู้อนุมัติ
ถ้า `user.SignatureImagePath == nil` (ยังไม่เคยอัปโหลดลายเซ็น) จะสแตมป์ไม่ได้จริง — เพิ่ม guard:
`if approver.SignatureImagePath == nil { return ErrValidation }` → 400 `VALIDATION_ERROR`
เป็น business rule ที่สมเหตุผลและ testable ชัดเจน — ใส่เป็น AC (AC5) แม้ไม่ได้อยู่ใน prompt เดิมตรง ๆ

### 5. Snapshot คืออะไร + ทำไมต้อง snapshot

Field ใหม่บน `Quotation` (migration `000005`):

| field                      | type                     | ความหมาย                                                                                |
| -------------------------- | ------------------------ | --------------------------------------------------------------------------------------- |
| `approver_id`              | `*uint` FK → `users(id)` | ใครอนุมัติ (อ้างอิง user ปัจจุบัน ไม่ snapshot)                                         |
| `approved_at`              | `*time.Time`             | เวลาอนุมัติ (server clock, ผ่าน `s.clock()` ที่ inject ไว้แล้วเพื่อ deterministic test) |
| `approved_signee_name`     | `*string`                | **snapshot** ของ `user.FullName` ณ เวลาอนุมัติ                                          |
| `approved_signee_position` | `*string`                | **snapshot** ของ `user.Position` ณ เวลาอนุมัติ                                          |
| `approved_signature_path`  | `*string`                | **snapshot** ของ `user.SignatureImagePath` ณ เวลาอนุมัติ                                |

**เหตุผลที่ต้อง snapshot ชื่อ/ตำแหน่ง**: โปรไฟล์ผู้อนุมัติ (`full_name`, `position`) แก้ไขได้ภายหลังผ่าน
`PUT /me/profile` — ถ้าไม่ snapshot เอกสารที่อนุมัติไปแล้วในอดีตจะเปลี่ยนหน้าตาไปตามโปรไฟล์ปัจจุบันของ user
(เช่น เปลี่ยนตำแหน่งงาน) ซึ่งผิดหลัก "เอกสารอนุมัติแล้วต้องคงสภาพเดิม" — ต้อง freeze ค่า ณ เวลานั้น

**ข้อจำกัดที่รู้อยู่และไม่แก้ในสโคปนี้ (YAGNI, ระบุใน "ความเสี่ยง")**: `approved_signature_path` snapshot
เป็น**ตำแหน่งไฟล์** ไม่ใช่การ copy ไฟล์จริง เนื่องจาก `ProfileService.SaveSignature` ตั้งชื่อไฟล์คงที่ตาม
userID เสมอ (`user_<id>.<ext>`) — ถ้า approver คนเดิม re-upload ลายเซ็นใหม่ (นามสกุลไฟล์เดิม) ไฟล์ที่ path เดิม
จะถูกทับ ทำให้เอกสารเก่าที่อนุมัติไปแล้วแสดงรูปใหม่แทน (ชื่อ/ตำแหน่งยัง freeze ถูกต้อง แต่รูปภาพไม่ freeze จริง)
ยอมรับ trade-off นี้เพราะไม่มีอยู่ใน AC และการทำ versioned file copy เป็นงานเพิ่มที่ over-engineer สำหรับตอนนี้

**ไม่ expose `approved_signature_path` แบบ raw ใน JSON response** (ต่างจาก `dto.MeResponse` เดิมที่ expose
`signature_image_path` ตรง ๆ ซึ่งเป็น path ของเจ้าของ token เอง ความเสี่ยงต่ำกว่า) — ที่นี่ผู้ดูใบเสนอราคา
อาจเป็นใครก็ได้ที่ authenticated ไม่ใช่แค่เจ้าของ ให้ expose แค่ `has_approved_signature bool` ใน
`QuotationResponse` แล้วให้ frontend ดึงรูปจริงผ่าน endpoint สตรีมแยก (ดู task 4) — ป้องกัน path ไฟล์ server
หลุดออกไป (`security.md`: ห้าม leak internal detail)

### 6. Invoice foresight — **ไม่แก้ schema ในสโคปนี้**

โจทย์บอกว่า "ใส่แค่ FK/field เผื่อ" แต่ตาราง `invoices` **ยังไม่มีอยู่จริง** (deferred ไปขั้นที่ 5) การเพิ่ม column
ให้ `quotations`/`payment_terms` เผื่อความสัมพันธ์กับตารางที่ยังไม่มีอยู่จะไม่มีจุดใช้งานจริง (dead column,
ผิดหลัก YAGNI ที่โจทย์เองก็เตือนไว้) **Pinned decision: ไม่มี schema change สำหรับ invoice ในสโคปนี้**
ความพร้อมสำหรับ invoice ในอนาคตมาจาก field ที่มีอยู่แล้วตามธรรมชาติ:

- `quotations.status = 'approved'` คือเงื่อนไขที่ future `InvoiceService` ต้องเช็คก่อนสร้าง invoice
  (business rule จะ enforce ใน service layer ตอนสร้าง invoice ในอนาคต ไม่ใช่ DB constraint)
- `payment_terms.id` (มีอยู่แล้ว) คือ FK ที่ future `invoices.payment_term_id` จะอ้างอิง (1 term → N invoice
  รองรับ partial invoicing ถ้าต้องการในอนาคต)
- future `invoices.quotation_id` จะ FK ไปที่ `quotations.id` ตรง ๆ

ไม่มี task/migration สำหรับข้อนี้ในสโคปนี้ — บันทึกไว้เป็นเอกสารอ้างอิงสำหรับตอนเปิด `/feature invoice` เท่านั้น

### 7. Concurrency guard ของ transition (race condition)

Approve/Submit/Reject ต้องเป็น **atomic conditional update** (`WHERE id = ? AND status = ?`) ไม่ใช่
load-then-save แบบ `Update()` เดิม (ซึ่งจะ overwrite ทั้ง Items/PaymentTerms ด้วย — ห้ามใช้ `Update()` เดิมกับ
transition เด็ดขาด เพราะ `Update()` ทำ full-replace items/payment-terms) ถ้า `RowsAffected == 0` แปลว่า status
เปลี่ยนไปแล้วโดย request อื่นระหว่างนั้น (เช่น อนุมัติซ้อนกัน) → ตอบ `ErrConflict` (409) — ดู spec repository
ใน task 2

---

## ขอบเขต

- **Database**: migration `000005` — เปลี่ยน CHECK constraint สถานะ + เพิ่ม 5 column บน `quotations`
- **Backend**: model, dto, repository (`TransitionStatus` ใหม่ + sentinel error), service (3 method transition
  ใหม่ + 1 method ดึง signature path), handler (3 endpoint transition + 1 endpoint stream รูป), router
  (3 route ใหม่ + 1 route ใหม่), ไม่แตะ auth/middleware (reuse ของเดิมทั้งหมด)
- **Frontend**: `quotationService.js` (4 ฟังก์ชันใหม่), hooks ใหม่ (`useSubmitQuotation`,
  `useApproveQuotation`, `useRejectQuotation`, `useApprovalSignatureUrl`), `QuotationDetailPage.jsx`
  (ปุ่ม Submit/Approve/Reject ตามสิทธิ์+สถานะ, ส่วนแสดงข้อมูล approved), `QuotationListPage.jsx`
  (แก้ label `sent` → `pending_approval` ใน STATUS_OPTIONS)

---

## Tasks (เรียงตาม dependency)

### 1. [DB] Migration `000005_add_approval_fields_to_quotations`

`backend/migrations/000005_add_approval_fields_to_quotations.up.sql`:

```sql
-- Backfill safety: no code path ever set 'sent', but guard against stray prod rows
-- before tightening the CHECK constraint.
UPDATE quotations SET status = 'pending_approval' WHERE status = 'sent';

ALTER TABLE quotations DROP CONSTRAINT quotations_status_check;
ALTER TABLE quotations ADD CONSTRAINT quotations_status_check
    CHECK (status IN ('draft', 'pending_approval', 'approved', 'rejected'));

ALTER TABLE quotations ADD COLUMN approver_id BIGINT REFERENCES users(id);
ALTER TABLE quotations ADD COLUMN approved_at TIMESTAMPTZ;
ALTER TABLE quotations ADD COLUMN approved_signee_name TEXT;
ALTER TABLE quotations ADD COLUMN approved_signee_position TEXT;
ALTER TABLE quotations ADD COLUMN approved_signature_path TEXT;
```

`.down.sql`:

```sql
ALTER TABLE quotations DROP COLUMN IF EXISTS approved_signature_path;
ALTER TABLE quotations DROP COLUMN IF EXISTS approved_signee_position;
ALTER TABLE quotations DROP COLUMN IF EXISTS approved_signee_name;
ALTER TABLE quotations DROP COLUMN IF EXISTS approved_at;
ALTER TABLE quotations DROP COLUMN IF EXISTS approver_id;

ALTER TABLE quotations DROP CONSTRAINT quotations_status_check;
ALTER TABLE quotations ADD CONSTRAINT quotations_status_check
    CHECK (status IN ('draft', 'sent', 'approved', 'rejected'));
```

> ชื่อ constraint `quotations_status_check` มาจาก naming convention default ของ Postgres สำหรับ inline
> CHECK ใน `CREATE TABLE` (`<table>_<column>_check`) — ตรงกับที่นิยามใน `000002_create_quotations.up.sql`

### 2. [BE] Model + Repository

`backend/internal/model/quotation.go` — เพิ่ม field บน struct `Quotation` (ต่อจาก `CreatedBy`):

```go
ApproverID             *uint      `gorm:"column:approver_id"`
ApprovedAt             *time.Time `gorm:"column:approved_at"`
ApprovedSigneeName     *string    `gorm:"column:approved_signee_name"`
ApprovedSigneePosition *string    `gorm:"column:approved_signee_position"`
ApprovedSignaturePath  *string    `gorm:"column:approved_signature_path"`
```

`backend/internal/repository/quotation_repository.go`:

- เพิ่ม sentinel `var ErrStatusConflict = errors.New("status conflict")` (คู่กับ `ErrDuplicateReferenceNo` ที่มีอยู่)
- เพิ่มเมธอดใหม่ในอินเทอร์เฟซ `QuotationRepository`:
  ```go
  TransitionStatus(ctx context.Context, id uint, fromStatus string, updates map[string]any) error
  ```
- Implementation (**ห้ามใช้ `Update()` เดิม** — มันจะ full-replace Items/PaymentTerms):
  ```go
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
  ```
  `updates` เป็น `map[string]any` เพราะแต่ละ transition อัปเดต column ต่างกัน (submit อัปเดตแค่ `status`,
  approve อัปเดต 5 column) — ใช้ `Updates(map)` ของ GORM จะอัปเดตเฉพาะ key ที่ส่งมา ไม่แตะ column อื่น

### 3. [BE] Service — 3 transition method + 1 signature-path method

`backend/internal/service/quotation_service.go`:

```go
func (s *QuotationService) SubmitQuotation(ctx context.Context, userID uint, role string, id uint) (*dto.QuotationResponse, error) {
    existing, err := s.repo.FindByID(ctx, id)
    if err != nil { return nil, translateNotFound(err) }
    if role != "admin" && existing.CreatedBy != userID { return nil, ErrForbidden }
    if existing.Status != "draft" { return nil, ErrConflict }
    if err := s.repo.TransitionStatus(ctx, id, "draft", map[string]any{"status": "pending_approval"}); err != nil {
        if errors.Is(err, repository.ErrStatusConflict) { return nil, ErrConflict }
        return nil, err
    }
    existing.Status = "pending_approval"
    return mapQuotationResponse(existing), nil
}

func (s *QuotationService) ApproveQuotation(ctx context.Context, userID uint, role string, id uint) (*dto.QuotationResponse, error) {
    if role != "approver" { return nil, ErrForbidden } // defense-in-depth, route already RequireRole("approver")
    existing, err := s.repo.FindByID(ctx, id)
    if err != nil { return nil, translateNotFound(err) }
    if existing.Status != "pending_approval" { return nil, ErrConflict }
    approver, err := s.userRepo.FindByID(ctx, userID)
    if err != nil { return nil, err }
    if approver.SignatureImagePath == nil { return nil, ErrValidation }
    now := s.clock()
    updates := map[string]any{
        "status":                    "approved",
        "approver_id":               userID,
        "approved_at":               now,
        "approved_signee_name":      approver.FullName,
        "approved_signee_position":  approver.Position,
        "approved_signature_path":   *approver.SignatureImagePath,
    }
    if err := s.repo.TransitionStatus(ctx, id, "pending_approval", updates); err != nil {
        if errors.Is(err, repository.ErrStatusConflict) { return nil, ErrConflict }
        return nil, err
    }
    existing.Status = "approved"
    existing.ApproverID = &userID
    existing.ApprovedAt = &now
    existing.ApprovedSigneeName = &approver.FullName
    existing.ApprovedSigneePosition = &approver.Position
    existing.ApprovedSignaturePath = approver.SignatureImagePath
    return mapQuotationResponse(existing), nil
}

func (s *QuotationService) RejectQuotation(ctx context.Context, userID uint, role string, id uint) (*dto.QuotationResponse, error) {
    if role != "approver" { return nil, ErrForbidden }
    existing, err := s.repo.FindByID(ctx, id)
    if err != nil { return nil, translateNotFound(err) }
    if existing.Status != "pending_approval" { return nil, ErrConflict }
    if err := s.repo.TransitionStatus(ctx, id, "pending_approval", map[string]any{"status": "rejected"}); err != nil {
        if errors.Is(err, repository.ErrStatusConflict) { return nil, ErrConflict }
        return nil, err
    }
    existing.Status = "rejected"
    return mapQuotationResponse(existing), nil
}

// GetApprovalSignaturePath returns the stored (path, content-type) of the approval
// stamp signature. Returns ErrNotFound if the quotation isn't approved or has no
// snapshotted signature path.
func (s *QuotationService) GetApprovalSignaturePath(ctx context.Context, id uint) (string, string, error) {
    existing, err := s.repo.FindByID(ctx, id)
    if err != nil { return "", "", translateNotFound(err) }
    if existing.Status != "approved" || existing.ApprovedSignaturePath == nil {
        return "", "", ErrNotFound
    }
    // pathToContentType is already defined (unexported) in profile_service.go,
    // same package `service` — reuse it, do NOT duplicate.
    return *existing.ApprovedSignaturePath, pathToContentType(*existing.ApprovedSignaturePath), nil
}
```

> หมายเหตุ: `mapQuotationResponse` ต้องเพิ่ม field ใหม่ (`ApproverID`, `ApprovedAt` แปลงด้วย helper ใหม่
> `timePtrToRFC3339` เพราะเป็น timestamp เต็ม ไม่ใช่ date-only แบบ `formatTime`/`timePtrToString`,
> `ApprovedSigneeName`, `ApprovedSigneePosition`, และ `HasApprovedSignature bool` = `q.ApprovedSignaturePath != nil`
> — **ห้าม** map `ApprovedSignaturePath` ดิบลง JSON ตาม Decision 5)

`backend/internal/dto/quotation_dto.go`:

- `QuotationResponse` เพิ่ม field: `ApproverID *uint json:"approver_id"`, `ApprovedAt *string json:"approved_at"`,
  `ApprovedSigneeName *string json:"approved_signee_name"`, `ApprovedSigneePosition *string json:"approved_signee_position"`,
  `HasApprovedSignature bool json:"has_approved_signature"`
- `ListQuotationQuery.Status` แก้ `binding:"omitempty,oneof=draft sent approved rejected"` →
  `binding:"omitempty,oneof=draft pending_approval approved rejected"`

### 4. [BE] Handler + Router

`backend/internal/handler/quotation_handler.go`:

- เพิ่มเมธอดในอินเทอร์เฟซ `QuotationServicer`: `SubmitQuotation`, `ApproveQuotation`, `RejectQuotation`
  (signature ตรงกับ service ด้านบน) และ `GetApprovalSignaturePath(ctx, id) (string, string, error)`
- เพิ่ม handler method `Submit`, `Approve`, `Reject` — pattern เดียวกับ `Update`/`Delete` เดิม (`c.MustGet("userID")`,
  `c.GetString("role")`, parse `:id`, เรียก service, `response.Success(c, http.StatusOK, resp, "<submitted|approved|rejected>")`)
- เพิ่ม handler method `GetApprovalSignature` — pattern เดียวกับ `MeHandler.GetSignature` เป๊ะ
  (`c.Header("Content-Type", contentType); c.File(path)`)

`backend/internal/router/router.go`:

```go
quotationsWrite.POST("/:id/submit", quotationHandler.Submit) // ใน group เดิม RequireRole("admin","creator")

quotations.GET("/:id/approval-signature", quotationHandler.GetApprovalSignature) // ใน group อ่านเดิม (Auth เท่านั้น)

quotationsApproval := engine.Group("/quotations", middleware.Auth(tokenSvc), middleware.RequireRole("approver"))
{
    quotationsApproval.POST("/:id/approve", quotationHandler.Approve)
    quotationsApproval.POST("/:id/reject", quotationHandler.Reject)
}
```

### 5. [FE] Service + Hooks

`frontend/src/features/quotation/services/quotationService.js` — เพิ่ม:

```js
export async function submit(id) {
  return apiClient.post(`${QUOTATIONS}/${id}/submit`);
}
export async function approve(id) {
  return apiClient.post(`${QUOTATIONS}/${id}/approve`);
}
export async function reject(id) {
  return apiClient.post(`${QUOTATIONS}/${id}/reject`);
}
export async function getApprovalSignatureUrl(id) {
  const response = await apiClient.get(
    `${QUOTATIONS}/${id}/approval-signature`,
    { responseType: "arraybuffer" },
  );
  const contentType = response.headers["content-type"] || "image/png";
  const blob = new Blob([response.data], { type: contentType });
  return URL.createObjectURL(blob);
}
```

(pattern ก็อปจาก `authService.getSignatureUrl` ที่มีอยู่แล้ว)

Hooks ใหม่ (`frontend/src/features/quotation/hooks/`) — pattern ก็อปจาก `useDeleteQuotation.js`:

- `useSubmitQuotation.js`, `useApproveQuotation.js`, `useRejectQuotation.js` — `useMutation` ที่ `onSuccess`
  ทำ `queryClient.invalidateQueries({ queryKey: ['quotation', id] })` **และ** `['quotations']`
  (list ก็ filter ตาม status ได้ ต้อง invalidate ทั้งคู่)
- `useApprovalSignatureUrl.js` — pattern ก็อปจาก `useSignatureUrl.js`:
  ```js
  export function useApprovalSignatureUrl(id, hasApprovedSignature) {
    return useQuery({
      queryKey: ["quotation", id, "approval-signature"],
      queryFn: () => getApprovalSignatureUrl(id),
      enabled: !!hasApprovedSignature,
      retry: false,
      staleTime: Infinity,
    });
  }
  ```

### 6. [FE] `QuotationDetailPage.jsx` — ปุ่ม + ส่วนแสดง approved

- ใช้ `useMe()` (มีอยู่แล้ว) ดึง `{ id, role }` ของ user ปัจจุบัน — **ไม่แก้ `AuthContext`** (มันเก็บแค่ `role`
  จาก JWT decode ปัจจุบัน ไม่มี user id; ใช้ `useMe()` ที่ยิง `GET /me` แทน ซึ่งมี `id` อยู่แล้วใน `MeResponse`)
- แสดงปุ่ม **Submit** เมื่อ `quotation.status === 'draft' && me && (me.role === 'admin' || me.id === quotation.created_by)`
  → เรียก `useSubmitQuotation().mutate(id)`
- แสดงปุ่ม **Approve** / **Reject** เมื่อ `quotation.status === 'pending_approval' && me?.role === 'approver'`
  → เรียก `useApproveQuotation().mutate(id)` / `useRejectQuotation().mutate(id)`
- แสดง error จาก mutation (`mutation.error?.message`) เป็นข้อความใต้ปุ่ม (ไม่มี toast lib ในโปรเจกต์นี้ —
  ใช้ inline `<p role="alert">{mutation.error.message}</p>` ตาม pattern เรียบง่ายที่มีอยู่ ไม่เพิ่ม dependency ใหม่)
- section **Approved** ใหม่ แสดงเมื่อ `quotation.status === 'approved'`: `approved_signee_name`,
  `approved_signee_position`, `approved_at`, และรูปลายเซ็นผ่าน `useApprovalSignatureUrl(id, quotation.has_approved_signature)`
  (เหมือน `ProfilePage.jsx` แสดง `existingSignatureUrl`)
- badge/ข้อความสถานะทั่วไป (`quotation.status`) แสดงอยู่แล้ว (บรรทัดเดิม) — ไม่ต้องแก้

### 7. [FE] `QuotationListPage.jsx` — แก้ label filter

`STATUS_OPTIONS`: เปลี่ยน `{ value: 'sent', label: 'Sent' }` → `{ value: 'pending_approval', label: 'Pending Approval' }`

---

## Acceptance Criteria

- **AC1**: creator เจ้าของ quotation สถานะ `draft` เรียก `POST /quotations/:id/submit` → 200,
  response `data.status === "pending_approval"`
- **AC2**: user role `creator` ที่**ไม่ใช่**เจ้าของเอกสาร เรียก submit → 403 `FORBIDDEN`
- **AC2b**: submit quotation ที่สถานะไม่ใช่ `draft` (เช่น `pending_approval` อยู่แล้ว) → 409 `CONFLICT`
- **AC3**: user role ที่ไม่ใช่ `approver` (เช่น `creator`, `admin`) เรียก `POST /quotations/:id/approve` → 403 `FORBIDDEN`
- **AC4**: role `approver` เรียก approve บนใบสถานะ `pending_approval` (approver มี `signature_image_path`
  อยู่แล้ว) → 200, `data.status === "approved"`, `data.approver_id`, `data.approved_at`,
  `data.approved_signee_name`, `data.approved_signee_position` ตรงกับโปรไฟล์ผู้อนุมัติ ณ ตอนนั้น,
  `data.has_approved_signature === true`
- **AC5**: role `approver` ที่**ยังไม่เคยอัปโหลดลายเซ็น** (`signature_image_path IS NULL`) เรียก approve →
  400 `VALIDATION_ERROR` (ห้ามอนุมัติสำเร็จโดยไม่มีลายเซ็นสแตมป์)
- **AC5b**: approve/reject บนใบที่สถานะไม่ใช่ `pending_approval` (เช่น `draft` หรือ `approved` อยู่แล้ว) →
  409 `CONFLICT`
- **AC6**: role `approver` เรียก reject บนใบ `pending_approval` → 200, `data.status === "rejected"`
- **AC7 (regression)**: `PUT /quotations/:id` หรือ `DELETE /quotations/:id` บนใบที่สถานะ `approved`
  (หรือ `pending_approval`/`rejected`) → ยังคง 403 `FORBIDDEN` เหมือนเดิม (guard เดิมไม่เปลี่ยน)
- **AC8**: `GET /quotations/:id/approval-signature` บนใบที่ `approved` และมี `has_approved_signature` →
  200, `Content-Type: image/*`, body ไม่ว่าง
- **AC8b**: `GET /quotations/:id/approval-signature` บนใบที่ยังไม่ `approved` → 404 `NOT_FOUND`
- **AC9 (frontend)**: หน้า detail แสดงปุ่ม Submit เฉพาะ creator เจ้าของ+status draft, ปุ่ม Approve/Reject
  เฉพาะ role approver+status pending_approval, และแสดงชื่อ/ตำแหน่ง/วันที่/ลายเซ็นเมื่อ approved

---

## ความเสี่ยง / คำถามค้าง

1. **Reject ไม่มีช่อง "เหตุผล" (reason)** — โจทย์/AC ไม่ได้ระบุให้เก็บเหตุผลการ reject ตัดสินใจไม่ใส่ในสโคปนี้
   (YAGNI) ถ้าต้องการภายหลังต้องเพิ่ม column + field ในอนาคต (ไม่กระทบ schema เดิมเพราะเป็น nullable เพิ่ม)
2. **`rejected` เป็น terminal state ถาวร** — ผู้ใช้ที่โดน reject ต้องสร้างใบใหม่ทั้งหมด ไม่มี resubmit
   ถ้า business ต้องการ resubmit flow ในอนาคต (`rejected → draft`) ต้องเปิดเป็น decision ใหม่แยกต่างหาก
3. **`approved_signature_path` เป็น snapshot ของ "path" ไม่ใช่ "ไฟล์"** — ถ้า approver re-upload ลายเซ็นใหม่
   (นามสกุลไฟล์เดิม) หลังอนุมัติไปแล้ว ไฟล์ที่ path เดิมจะถูกทับ ทำให้เอกสารเก่าที่อนุมัติแล้วแสดงรูปใหม่
   (ชื่อ/ตำแหน่งยัง freeze ถูกต้อง) — ยอมรับ trade-off นี้ตาม YAGNI ไม่มีอยู่ใน AC
4. **Approve/Reject ไม่มี admin bypass** (asymmetric กับ CRUD ownership ที่ admin bypass ได้) — เป็นการตัดสินใจ
   ที่ pin ไว้ตาม literal reading ของโจทย์ ("approver เท่านั้น") ถ้าทีมต้องการให้ admin อนุมัติแทนได้ในกรณีฉุกเฉิน
   ต้องเป็น decision แยกและอาจกระทบความหมายของ segregation of duties
5. **Inconsistency ที่ทราบแต่ไม่แก้**: guard เดิมของ edit/delete (`status != draft`) ตอบ 403 ไม่ใช่ 409
   ต่างจาก convention ใหม่ที่ pin ในสโคปนี้ (403=สิทธิ์, 409=สถานะผิด) — คงพฤติกรรมเดิมไว้เพราะนอกสโคป/ไม่มี
   test เดิมให้แก้ ถ้าจะ normalize ในอนาคตต้องรู้ว่าเป็น breaking change ของ error code ฝั่ง client เดิม
6. **Self-approval ไม่ถูกกันด้วยโค้ดพิเศษ** — เพราะ role เป็นค่าเดียวต่อ user (`creator` xor `approver` xor
   `admin`) ผู้สร้างเอกสารกับผู้อนุมัติจึงเป็นคนละคนโดยธรรมชาติของ RBAC ปัจจุบัน ไม่ต้องเพิ่ม guard แยก
   (ถ้าอนาคตให้ user มีหลาย role พร้อมกันได้ ต้องกลับมาทบทวนจุดนี้)
7. **`invoices` table ยังไม่มี** — Decision 6 ตัดสินใจไม่แตะ schema เลยสำหรับ invoice foresight รอบนี้
   ถ้า main agent อยากให้มี placeholder column บาง field ใน `quotations`/`payment_terms` ไว้ก่อนจริง ๆ
   ต้องสั่งเจาะจงว่าจะใส่ field อะไร (ไม่มีข้อมูลพอจะเดาแบบไม่ over-engineer)
