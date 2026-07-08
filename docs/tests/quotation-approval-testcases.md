# Test Cases: Quotation Approval Workflow (slug: quotation-approval)

อ้างอิงแผน: `docs/plans/quotation-approval.md`

สถานะ: RED (ทุก test ในเอกสารนี้ต้อง fail ก่อน dev implement — compile error หรือ assertion fail
เพราะ symbol/behavior ยังไม่มี)

> หมายเหตุ scope: repository integration test (TC-REPO-A0x) ต้องใช้ Docker (testcontainers) —
> เครื่องนี้ไม่มี Docker ยืนยัน RED ด้วย `go vet`/`go build`/`go test -short` (compile-time)
> แทนการรันจริงผ่าน container ตามที่ main agent สั่ง

---

## Backend — Repository (integration, ต้องมี Docker; ยืนยัน RED ด้วย compile)

ไฟล์: `backend/internal/repository/quotation_approval_repository_test.go` (package `repository`)

Contract ที่ dev ต้องสร้าง (`backend/internal/repository/quotation_repository.go`):

```go
var ErrStatusConflict = errors.New("status conflict")

type QuotationRepository interface {
    // ...methods เดิม...
    TransitionStatus(ctx context.Context, id uint, fromStatus string, updates map[string]any) error
}
```

Implementation **ต้อง**เป็น atomic conditional update (`WHERE id=? AND status=?`) **ห้ามใช้ `Update()` เดิม**
(full-replace Items/PaymentTerms) — `RowsAffected == 0` → `ErrStatusConflict`

ใช้ helper ที่มีอยู่แล้วในไฟล์เดิม (same package): `testDB` (package var), `setupTx(t)`, `seedUser(t, tx)`
ไม่แก้ไฟล์ `quotation_repository_test.go` เดิม — เพิ่ม helper ใหม่ `seedDraftQuotationForApproval` ในไฟล์ใหม่นี้เท่านั้น

| ID          | อ้างอิง AC                         | ประเภท | Given                                            | When                                                                                                                                                    | Then                                                                                                                                                                                                                     |
| ----------- | ---------------------------------- | ------ | ------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| TC-REPO-A01 | Decision 7 (atomic transition)     | happy  | quotation draft มี 1 item, company="Approval Co" | `TransitionStatus(id, "draft", {status: pending_approval})`                                                                                             | no error; reload: `status=="pending_approval"`, `company=="Approval Co"` (ไม่ถูกแตะ), `len(items)==1` (ไม่ถูกลบ — พิสูจน์ว่าไม่ได้ใช้ `Update()`)                                                                        |
| TC-REPO-A02 | Decision 7 (race condition)        | error  | quotation จริงยังเป็น `draft`                    | `TransitionStatus(id, "pending_approval", {status: approved})` (fromStatus ไม่ตรงของจริง)                                                               | `errors.Is(err, ErrStatusConflict)`; reload: status ยังเป็น `"draft"` (ไม่ถูกแก้)                                                                                                                                        |
| TC-REPO-A03 | Decision 5 (multi-column snapshot) | happy  | quotation `pending_approval`                     | `TransitionStatus(id, "pending_approval", {status, approver_id, approved_at, approved_signee_name, approved_signee_position, approved_signature_path})` | no error; reload: ทุก column ตรงตามที่ส่ง (`approver_id`, `approved_signee_name=="Approver Name"`, `approved_signee_position=="CFO"`, `approved_signature_path=="/uploads/signatures/user_x.png"`, `approved_at != nil`) |

---

## Backend — Service (unit, mock repo/userRepo)

ไฟล์: `backend/internal/service/quotation_approval_service_test.go` (package `service`)

Contract ที่ dev ต้องสร้าง (`backend/internal/service/quotation_service.go`):

```go
func (s *QuotationService) SubmitQuotation(ctx context.Context, userID uint, role string, id uint) (*dto.QuotationResponse, error)
func (s *QuotationService) ApproveQuotation(ctx context.Context, userID uint, role string, id uint) (*dto.QuotationResponse, error)
func (s *QuotationService) RejectQuotation(ctx context.Context, userID uint, role string, id uint) (*dto.QuotationResponse, error)
func (s *QuotationService) GetApprovalSignaturePath(ctx context.Context, id uint) (path string, contentType string, err error)
```

`dto.QuotationResponse` ต้องเพิ่ม: `ApproverID *uint`, `ApprovedAt *string` (RFC3339 เช่น `"2026-07-15T10:00:00Z"`),
`ApprovedSigneeName *string`, `ApprovedSigneePosition *string`, `HasApprovedSignature bool`
(**ห้าม** เพิ่ม field ดิบของ `approved_signature_path` ลง DTO — Decision 5)

ไฟล์นี้เพิ่มเมธอด `TransitionStatus` ให้ `mockQuotationRepository` (struct ประกาศใน `quotation_mocks_test.go`
เดิม — เพิ่มเมธอดในไฟล์ใหม่ได้ ไม่ต้องแก้ไฟล์เดิม) และประกาศ `approvalFixedClock()` (เวลาคงที่แยกจาก `fixedClock()`
เดิมเพื่อความชัดเจน แต่ค่าเดียวกัน: `2026-07-15 10:00:00 UTC`)

| ID         | อ้างอิง AC                         | ประเภท | Given                                                                                                                                                      | When                                          | Then                                                                                                                                                                                                                                                                                                    |
| ---------- | ---------------------------------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| TC-SVC-A01 | AC1                                | happy  | quotation `draft`, `CreatedBy=7`                                                                                                                           | `SubmitQuotation(ctx, 7, "creator", 42)`      | no error; `resp.Status == "pending_approval"`; repo ถูกเรียก `TransitionStatus(42, "draft", {status: pending_approval})`                                                                                                                                                                                |
| TC-SVC-A02 | AC2                                | error  | quotation `draft`, `CreatedBy=99`                                                                                                                          | `SubmitQuotation(ctx, 7, "creator", 42)`      | `errors.Is(err, ErrForbidden)`; `TransitionStatus` ไม่ถูกเรียก                                                                                                                                                                                                                                          |
| TC-SVC-A03 | Decision 3 (admin bypass submit)   | edge   | quotation `draft`, `CreatedBy=99`                                                                                                                          | `SubmitQuotation(ctx, 7, "admin", 42)`        | no error; `resp.Status == "pending_approval"`                                                                                                                                                                                                                                                           |
| TC-SVC-A04 | AC2b                               | error  | quotation `pending_approval`, owner ตรง                                                                                                                    | `SubmitQuotation(ctx, 7, "creator", 42)`      | `errors.Is(err, ErrConflict)`; `TransitionStatus` ไม่ถูกเรียก                                                                                                                                                                                                                                           |
| TC-SVC-A05 | Decision 7 (race)                  | edge   | `FindByID` คืน `draft` (stale read) แต่ `TransitionStatus` คืน `repository.ErrStatusConflict`                                                              | `SubmitQuotation(ctx, 7, "creator", 42)`      | `errors.Is(err, ErrConflict)` (แปลจาก repo sentinel)                                                                                                                                                                                                                                                    |
| TC-SVC-A06 | AC3                                | error  | role `creator`                                                                                                                                             | `ApproveQuotation(ctx, 7, "creator", 42)`     | `errors.Is(err, ErrForbidden)`; `repo.FindByID` ไม่ถูกเรียก (role check มาก่อน)                                                                                                                                                                                                                         |
| TC-SVC-A07 | AC3 + Decision 3 (no admin bypass) | error  | role `admin`                                                                                                                                               | `ApproveQuotation(ctx, 7, "admin", 42)`       | `errors.Is(err, ErrForbidden)`; `repo.FindByID` ไม่ถูกเรียก                                                                                                                                                                                                                                             |
| TC-SVC-A08 | AC4                                | happy  | quotation `pending_approval`; approver (userID 9) มี `signature_image_path`, `FullName="Approver Name"`, `Position="CFO"`                                  | `ApproveQuotation(ctx, 9, "approver", 42)`    | no error; `resp.Status=="approved"`; `*resp.ApproverID==9`; `*resp.ApprovedAt=="2026-07-15T10:00:00Z"`; `*resp.ApprovedSigneeName=="Approver Name"`; `*resp.ApprovedSigneePosition=="CFO"`; `resp.HasApprovedSignature==true`; `TransitionStatus` ถูกเรียกด้วย map ที่มีทุก key ตรงค่า snapshot ตอนนั้น |
| TC-SVC-A09 | AC4 (snapshot ไม่ live-lookup)     | edge   | quotation `approved` มี snapshot fields ที่ตั้งไว้แล้วในอดีต (`ApprovedSigneeName="Old Name At Approval Time"` ฯลฯ) — **ไม่ stub `userRepo.FindByID` เลย** | `GetQuotation(ctx, 42)`                       | no error; `resp.ApprovedSigneeName`/`Position` ตรงกับ snapshot ที่เก็บไว้ (ไม่ใช่ query ใหม่); `userRepo.AssertNotCalled("FindByID", ...)` — ถ้า service live-lookup จะ panic เพราะไม่มี `.On(...)` ตั้งไว้                                                                                             |
| TC-SVC-A10 | AC5                                | error  | approver `signature_image_path == nil`                                                                                                                     | `ApproveQuotation(ctx, 9, "approver", 42)`    | `errors.Is(err, ErrValidation)`; `TransitionStatus` ไม่ถูกเรียก                                                                                                                                                                                                                                         |
| TC-SVC-A11 | AC5b                               | error  | quotation `draft` (ไม่ใช่ `pending_approval`)                                                                                                              | `ApproveQuotation(ctx, 9, "approver", 42)`    | `errors.Is(err, ErrConflict)`; `userRepo.FindByID` (ของ approver) ไม่ถูกเรียก (status check มาก่อน signature check)                                                                                                                                                                                     |
| TC-SVC-A12 | AC5b                               | error  | quotation `approved` แล้ว                                                                                                                                  | `RejectQuotation(ctx, 9, "approver", 42)`     | `errors.Is(err, ErrConflict)`; `TransitionStatus` ไม่ถูกเรียก                                                                                                                                                                                                                                           |
| TC-SVC-A13 | AC6                                | happy  | quotation `pending_approval`                                                                                                                               | `RejectQuotation(ctx, 9, "approver", 42)`     | no error; `resp.Status == "rejected"`                                                                                                                                                                                                                                                                   |
| TC-SVC-A14 | AC3                                | error  | role `creator`                                                                                                                                             | `RejectQuotation(ctx, 7, "creator", 42)`      | `errors.Is(err, ErrForbidden)`; `repo.FindByID` ไม่ถูกเรียก                                                                                                                                                                                                                                             |
| TC-SVC-A15 | AC7 (regression)                   | error  | quotation `pending_approval` (ค่า enum ใหม่แทน `sent`)                                                                                                     | `UpdateQuotation(ctx, 7, "creator", 42, req)` | `errors.Is(err, ErrForbidden)` (guard เดิม `!= draft`)                                                                                                                                                                                                                                                  |
| TC-SVC-A16 | AC7 (regression)                   | error  | quotation `rejected`                                                                                                                                       | `DeleteQuotation(ctx, 7, "creator", 42)`      | `errors.Is(err, ErrForbidden)`                                                                                                                                                                                                                                                                          |
| TC-SVC-A17 | AC8                                | happy  | quotation `approved`, `ApprovedSignaturePath="/uploads/signatures/user_9.png"`                                                                             | `GetApprovalSignaturePath(ctx, 42)`           | no error; `path=="/uploads/signatures/user_9.png"`; `contentType=="image/png"`                                                                                                                                                                                                                          |
| TC-SVC-A18 | AC8b                               | error  | quotation `pending_approval`                                                                                                                               | `GetApprovalSignaturePath(ctx, 42)`           | `errors.Is(err, ErrNotFound)`                                                                                                                                                                                                                                                                           |
| TC-SVC-A19 | AC8b (edge)                        | error  | quotation `approved` แต่ `ApprovedSignaturePath == nil`                                                                                                    | `GetApprovalSignaturePath(ctx, 42)`           | `errors.Is(err, ErrNotFound)`                                                                                                                                                                                                                                                                           |
| TC-SVC-A20 | regression (not-found translation) | error  | `FindByID` คืน `gorm.ErrRecordNotFound`                                                                                                                    | `SubmitQuotation(ctx, 7, "creator", 999)`     | `errors.Is(err, ErrNotFound)`                                                                                                                                                                                                                                                                           |

---

## Backend — Handler (unit, mock service; รวม RBAC ผ่าน middleware จริง)

ไฟล์: `backend/internal/handler/quotation_approval_handler_test.go` (package `handler`)

Contract ที่ dev ต้องสร้าง (`backend/internal/handler/quotation_handler.go`):

```go
// เพิ่มใน QuotationServicer:
SubmitQuotation(ctx context.Context, userID uint, role string, id uint) (*dto.QuotationResponse, error)
ApproveQuotation(ctx context.Context, userID uint, role string, id uint) (*dto.QuotationResponse, error)
RejectQuotation(ctx context.Context, userID uint, role string, id uint) (*dto.QuotationResponse, error)
GetApprovalSignaturePath(ctx context.Context, id uint) (path string, contentType string, err error)

func (h *QuotationHandler) Submit(c *gin.Context)            // POST /quotations/:id/submit  -> 200
func (h *QuotationHandler) Approve(c *gin.Context)           // POST /quotations/:id/approve -> 200
func (h *QuotationHandler) Reject(c *gin.Context)            // POST /quotations/:id/reject  -> 200
func (h *QuotationHandler) GetApprovalSignature(c *gin.Context) // GET /quotations/:id/approval-signature -> stream (pattern = MeHandler.GetSignature)
```

ไฟล์นี้เพิ่มเมธอด `SubmitQuotation`/`ApproveQuotation`/`RejectQuotation`/`GetApprovalSignaturePath` ให้
`mockQuotationService` (struct ประกาศใน `quotation_handler_test.go` เดิม — เพิ่มเมธอดในไฟล์ใหม่ ไม่แก้ไฟล์เดิม)
ใช้ helper ที่มีอยู่แล้ว: `withUser`, `withUserID`, `decodeJSONBody`, `fakeVerifier`

| ID         | อ้างอิง AC                | ประเภท       | Given                                                                                                                    | When                                      | Then                                                                                                                                                                                                                         |
| ---------- | ------------------------- | ------------ | ------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| TC-HDL-A01 | AC1                       | happy        | svc.SubmitQuotation → `{status: pending_approval}`                                                                       | `POST /quotations/1/submit` (creator)     | 200; `data.status=="pending_approval"`                                                                                                                                                                                       |
| TC-HDL-A02 | AC2                       | error        | svc.SubmitQuotation → `ErrForbidden`                                                                                     | `POST /quotations/1/submit`               | 403; `error.code=="FORBIDDEN"`                                                                                                                                                                                               |
| TC-HDL-A03 | AC2b                      | error        | svc.SubmitQuotation → `ErrConflict`                                                                                      | `POST /quotations/1/submit`               | 409; `error.code=="CONFLICT"`                                                                                                                                                                                                |
| TC-HDL-A04 | AC3                       | error (RBAC) | route จริงผ่าน `middleware.RequireRole("approver")`, token role `creator`                                                | `POST /quotations/1/approve`              | 403 `FORBIDDEN`; `svc.ApproveQuotation` ไม่ถูกเรียก                                                                                                                                                                          |
| TC-HDL-A05 | AC3 + Decision 3          | error (RBAC) | เหมือนบน แต่ token role `admin`                                                                                          | `POST /quotations/1/approve`              | 403 `FORBIDDEN` (admin ไม่ bypass)                                                                                                                                                                                           |
| TC-HDL-A06 | AC4                       | happy        | svc.ApproveQuotation → response ครบ 5 field (approver_id, approved_at, signee name/position, has_approved_signature)     | `POST /quotations/1/approve` (approver)   | 200; `data.status=="approved"`, `data.approver_id==9`, `data.approved_at=="2026-07-15T10:00:00Z"`, `data.approved_signee_name=="Approver Name"`, `data.approved_signee_position=="CFO"`, `data.has_approved_signature==true` |
| TC-HDL-A07 | AC5                       | error        | svc.ApproveQuotation → `ErrValidation`                                                                                   | `POST /quotations/1/approve`              | 400 `VALIDATION_ERROR`                                                                                                                                                                                                       |
| TC-HDL-A08 | AC5b                      | error        | svc.ApproveQuotation → `ErrConflict`                                                                                     | `POST /quotations/1/approve`              | 409 `CONFLICT`                                                                                                                                                                                                               |
| TC-HDL-A09 | AC6                       | happy        | svc.RejectQuotation → `{status: rejected}`                                                                               | `POST /quotations/1/reject` (approver)    | 200; `data.status=="rejected"`                                                                                                                                                                                               |
| TC-HDL-A10 | AC3                       | error (RBAC) | token role `creator`                                                                                                     | `POST /quotations/1/reject`               | 403 `FORBIDDEN`; `svc.RejectQuotation` ไม่ถูกเรียก                                                                                                                                                                           |
| TC-HDL-A11 | AC8                       | happy        | svc.GetApprovalSignaturePath → path ไฟล์ png จริง (เขียนผ่าน `os.WriteFile` ใน `t.TempDir()`), `contentType="image/png"` | `GET /quotations/1/approval-signature`    | 200; `Content-Type` ขึ้นต้น `image/`; body ไม่ว่าง                                                                                                                                                                           |
| TC-HDL-A12 | AC8b                      | error        | svc.GetApprovalSignaturePath → `ErrNotFound`                                                                             | `GET /quotations/1/approval-signature`    | 404 `NOT_FOUND`                                                                                                                                                                                                              |
| TC-HDL-A13 | Decision 5 (no path leak) | regression   | svc.GetQuotation → `{status: approved, has_approved_signature: true}` (ไม่มี raw path field ใน DTO เลย)                  | `GET /quotations/1`                       | 200; body **ไม่มี** substring `"approved_signature_path"`; `data.has_approved_signature==true`                                                                                                                               |
| TC-HDL-A14 | Decision 1 (oneof update) | happy        | svc.ListQuotations รับ query ที่ `status=="pending_approval"`                                                            | `GET /quotations?status=pending_approval` | 200                                                                                                                                                                                                                          |
| TC-HDL-A15 | Decision 1 (oneof update) | error        | ไม่ stub service (ต้องไม่ถูกเรียก)                                                                                       | `GET /quotations?status=sent`             | 400 `VALIDATION_ERROR` (ค่าเก่าไม่อยู่ใน oneof อีกต่อไป); `ListQuotations` ไม่ถูกเรียก                                                                                                                                       |

---

## Frontend — QuotationDetailPage (AC9)

ไฟล์: `frontend/src/features/quotation/pages/QuotationDetailPage.approval.test.jsx`

Contract ที่ dev ต้องสร้าง:

- `QuotationDetailPage.jsx` ใช้ `useMe()` (มีอยู่แล้ว) อ่าน `{ id, role }`
- ปุ่ม **Submit** (accessible name จับด้วย regex `/submit/i`) แสดงเมื่อ
  `quotation.status === 'draft' && (me.role === 'admin' || me.id === quotation.created_by)`
  กดแล้วเรียก `useSubmitQuotation().mutate(id)` → `POST /quotations/:id/submit`
- ปุ่ม **Approve** (`/approve/i`) และ **Reject** (`/reject/i`) แสดงเมื่อ
  `quotation.status === 'pending_approval' && me.role === 'approver'`
- error จาก mutation แสดงเป็น `<p role="alert">{mutation.error.message}</p>`
- ส่วน **Approved**: เมื่อ `quotation.status === 'approved'` แสดง `approved_signee_name`,
  `approved_signee_position`, `approved_at` เป็นข้อความ (raw, ไม่ format เพิ่ม — ตาม pattern
  `customer_signee_date` เดิม) และเมื่อ `has_approved_signature` เป็น true แสดง
  `<img data-testid="approved-signature" src={blobUrl}>` (ผ่าน `useApprovalSignatureUrl`)
- ไฟล์ใหม่ที่ต้องมี: `hooks/useSubmitQuotation.js`, `hooks/useApproveQuotation.js`,
  `hooks/useRejectQuotation.js`, `hooks/useApprovalSignatureUrl.js`; และ
  `quotationService.js` เพิ่ม `submit(id)`, `approve(id)`, `reject(id)`, `getApprovalSignatureUrl(id)`

| ID               | อ้างอิง AC             | ประเภท | Given                                                                              | When         | Then                                                                                                |
| ---------------- | ---------------------- | ------ | ---------------------------------------------------------------------------------- | ------------ | --------------------------------------------------------------------------------------------------- |
| TC-FE-DETAIL-A01 | AC9                    | happy  | me = creator id 7, quotation `draft` `created_by=7`                                | render       | ปุ่ม Submit แสดง                                                                                    |
| TC-FE-DETAIL-A02 | AC9                    | edge   | me = creator id 8, quotation `draft` `created_by=7`                                | render       | ปุ่ม Submit **ไม่**แสดง                                                                             |
| TC-FE-DETAIL-A03 | AC9 (admin bypass)     | edge   | me = admin, quotation `draft` `created_by=7`                                       | render       | ปุ่ม Submit แสดง                                                                                    |
| TC-FE-DETAIL-A04 | AC9                    | happy  | me = creator เจ้าของ, quotation `draft`                                            | คลิก Submit  | เรียก `POST /quotations/1/submit`; หลัง refetch status เปลี่ยนเป็น `pending_approval` บนหน้าจอ      |
| TC-FE-DETAIL-A05 | AC9                    | happy  | me = approver, quotation `pending_approval`                                        | render       | ปุ่ม Approve และ Reject แสดงทั้งคู่                                                                 |
| TC-FE-DETAIL-A06 | AC9                    | edge   | me = creator, quotation `pending_approval`                                         | render       | ปุ่ม Approve/Reject **ไม่**แสดง                                                                     |
| TC-FE-DETAIL-A07 | AC9                    | happy  | me = approver, quotation `pending_approval`                                        | คลิก Approve | เรียก `POST /quotations/1/approve`                                                                  |
| TC-FE-DETAIL-A08 | AC9 (error display)    | error  | approve API ตอบ 400 `VALIDATION_ERROR` message "approver has no signature on file" | คลิก Approve | `role="alert"` แสดงข้อความ error นั้น                                                               |
| TC-FE-DETAIL-A09 | AC9 (approved section) | happy  | quotation `approved` มี signee name/position/date/`has_approved_signature=true`    | render       | เห็นชื่อ/ตำแหน่ง/วันที่เป็นข้อความ; `data-testid="approved-signature"` มี `src` ขึ้นต้นด้วย `blob:` |

---

## Frontend — QuotationListPage (Decision 1 label)

ไฟล์: `frontend/src/features/quotation/pages/QuotationListPage.approval.test.jsx`

Contract: `STATUS_OPTIONS` เปลี่ยน `{ value: 'sent', label: 'Sent' }` →
`{ value: 'pending_approval', label: 'Pending Approval' }`

| ID             | อ้างอิง AC | ประเภท | Given | When                                  | Then                                                                                             |
| -------------- | ---------- | ------ | ----- | ------------------------------------- | ------------------------------------------------------------------------------------------------ |
| TC-FE-LIST-A01 | Decision 1 | happy  | —     | เปิด dropdown filter แล้วเลือก option | ไม่มี label "Sent"; มี label "Pending Approval"; เลือกแล้ว query param `status=pending_approval` |

---

## AC coverage summary

| AC                                                   | covered by                                               |
| ---------------------------------------------------- | -------------------------------------------------------- |
| AC1                                                  | TC-SVC-A01, TC-HDL-A01, TC-FE-DETAIL-A04                 |
| AC2                                                  | TC-SVC-A02, TC-HDL-A02, TC-FE-DETAIL-A02                 |
| AC2b                                                 | TC-SVC-A04, TC-HDL-A03                                   |
| AC3                                                  | TC-SVC-A06/A07/A14, TC-HDL-A04/A05/A10, TC-FE-DETAIL-A06 |
| AC4                                                  | TC-SVC-A08/A09, TC-HDL-A06, TC-FE-DETAIL-A07/A09         |
| AC5                                                  | TC-SVC-A10, TC-HDL-A07, TC-FE-DETAIL-A08                 |
| AC5b                                                 | TC-SVC-A11/A12, TC-HDL-A08                               |
| AC6                                                  | TC-SVC-A13, TC-HDL-A09                                   |
| AC7                                                  | TC-SVC-A15/A16                                           |
| AC8                                                  | TC-SVC-A17, TC-HDL-A11, TC-FE-DETAIL-A09                 |
| AC8b                                                 | TC-SVC-A18/A19, TC-HDL-A12                               |
| AC9                                                  | TC-FE-DETAIL-A01..A09                                    |
| Decision 1 (status enum)                             | TC-HDL-A14/A15, TC-FE-LIST-A01                           |
| Decision 3 (admin bypass submit / no bypass approve) | TC-SVC-A03/A07, TC-HDL-A05                               |
| Decision 5 (snapshot, no path leak)                  | TC-SVC-A09, TC-HDL-A13                                   |
| Decision 7 (atomic transition, race)                 | TC-REPO-A01/A02/A03, TC-SVC-A05                          |

ไม่มี AC ไหน descope — ครบทุก AC หลัก (AC1–AC9) รวม AC2b/AC5b (sub-AC) และ decision สำคัญที่ pin ไว้ในแผน
