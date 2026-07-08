# QA Report: Quotation Approval Workflow (slug: quotation-approval)

อ้างอิง: docs/plans/quotation-approval.md · docs/tests/quotation-approval-testcases.md
รอบที่: 1 | วันที่: 2026-07-08

## ผลรวม: PASS

- test ผ่าน 45/45 executed (48 defined; 3 repository integration tests SKIP — no Docker on this machine, expected/documented in test doc; verified via `go build`/`go vet` compile-check + manual code review against contract — see หมายเหตุด้านล่าง)
- AC ครบ 9/9 (primary AC1–AC9) + sub-AC 2/2 (AC2b, AC5b) — ครบทุกข้อตาม AC coverage summary ในเอกสาร test case

### คำสั่งที่รันจริง

```
cd backend && gofmt -l .            -> (no output, clean)
cd backend && go vet ./...          -> (no output, clean)
cd backend && go build ./...        -> (no output, clean)
cd backend && go test -short -v ./...   -> ok ทุก package (repo integration SKIP ตามคาด, ไม่มี Docker)
cd frontend && npm run lint         -> 0 errors, 5 pre-existing warnings (ไม่เกี่ยวกับ feature นี้)
cd frontend && npm test -- --run    -> 44/44 test files/tests PASS (รวม 10 test approval-specific)
cd frontend && npm run build        -> success
docker info                          -> ไม่มี Docker บนเครื่องนี้ (ยืนยันเหตุผลที่ TC-REPO-A0x ต้อง SKIP)
```

## ผล test case

### Backend — Repository (integration; ต้อง Docker)

| ID          | คาด                                                   | ได้จริง                                                                                                                                                                                   | สถานะ                                   |
| ----------- | ----------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------- |
| TC-REPO-A01 | atomic partial update, ไม่แตะ Company/Items           | ไม่ได้รันจริง (no Docker) — โค้ด `TransitionStatus` ยืนยันด้วย code review: ใช้ `Model(&model.Quotation{}).Where("id=? AND status=?").Updates(map)` ไม่ใช่ `Update()` เต็ม — ตรง contract | ⚠️ ยังไม่รันจริง (ต้อง CI ที่มี Docker) |
| TC-REPO-A02 | fromStatus ไม่ตรง → `ErrStatusConflict`, แถวไม่ถูกแก้ | โค้ดตรง `RowsAffected==0 → ErrStatusConflict` ตาม spec                                                                                                                                    | ⚠️ ยังไม่รันจริง (ต้อง CI)              |
| TC-REPO-A03 | multi-column snapshot ถูกบันทึกครบ                    | โค้ดส่ง `updates map[string]any` ทุก key ผ่าน `Updates()` ตรง contract                                                                                                                    | ⚠️ ยังไม่รันจริง (ต้อง CI)              |

> ทั้ง 3 เคสนี้ compile ผ่าน (`go vet`/`go build`/`go test -short` ยืนยัน RED→GREEN เชิง compile ตามที่ test-case-writer ระบุไว้) และ implementation ตรง spec 1:1 จาก code review แต่ **ยังไม่มีการรันจริงผ่าน testcontainers เพราะเครื่องนี้ไม่มี Docker** — ต้องให้ CI pipeline ที่มี Docker ยืนยันอีกครั้งก่อนถือว่า verify ครบ 100%

### Backend — Service (unit, mock)

| ID         | คาด                                                                    | ได้จริง                               | สถานะ |
| ---------- | ---------------------------------------------------------------------- | ------------------------------------- | ----- |
| TC-SVC-A01 | AC1 submit happy → pending_approval                                    | ตรงตามคาด                             | ✅    |
| TC-SVC-A02 | AC2 non-owner creator → ErrForbidden                                   | ตรงตามคาด                             | ✅    |
| TC-SVC-A03 | admin bypass ownership submit                                          | ตรงตามคาด                             | ✅    |
| TC-SVC-A04 | AC2b wrong status → ErrConflict                                        | ตรงตามคาด                             | ✅    |
| TC-SVC-A05 | race → repo ErrStatusConflict แปลเป็น ErrConflict                      | ตรงตามคาด                             | ✅    |
| TC-SVC-A06 | AC3 creator approve → ErrForbidden, FindByID ไม่ถูกเรียก               | ตรงตามคาด (role check ก่อน DB lookup) | ✅    |
| TC-SVC-A07 | AC3 admin approve → ErrForbidden (no bypass)                           | ตรงตามคาด                             | ✅    |
| TC-SVC-A08 | AC4 approve happy snapshot ครบ 5 field                                 | ตรงตามคาด                             | ✅    |
| TC-SVC-A09 | AC4 GetQuotation ใช้ snapshot ไม่ live-lookup                          | ตรงตามคาด (userRepo ไม่ถูกเรียก)      | ✅    |
| TC-SVC-A10 | AC5 approver ไม่มีลายเซ็น → ErrValidation                              | ตรงตามคาด                             | ✅    |
| TC-SVC-A11 | AC5b approve สถานะผิด → ErrConflict, status check ก่อน signature check | ตรงตามคาด                             | ✅    |
| TC-SVC-A12 | AC5b reject สถานะผิด → ErrConflict                                     | ตรงตามคาด                             | ✅    |
| TC-SVC-A13 | AC6 reject happy → rejected                                            | ตรงตามคาด                             | ✅    |
| TC-SVC-A14 | AC3 creator reject → ErrForbidden                                      | ตรงตามคาด                             | ✅    |
| TC-SVC-A15 | AC7 regression Update บน pending_approval → ErrForbidden               | ตรงตามคาด                             | ✅    |
| TC-SVC-A16 | AC7 regression Delete บน rejected → ErrForbidden                       | ตรงตามคาด                             | ✅    |
| TC-SVC-A17 | AC8 GetApprovalSignaturePath happy                                     | ตรงตามคาด                             | ✅    |
| TC-SVC-A18 | AC8b not approved → ErrNotFound                                        | ตรงตามคาด                             | ✅    |
| TC-SVC-A19 | AC8b approved แต่ path nil → ErrNotFound                               | ตรงตามคาด                             | ✅    |
| TC-SVC-A20 | not-found translation → ErrNotFound                                    | ตรงตามคาด                             | ✅    |

### Backend — Handler (unit, mock service + RBAC ผ่าน middleware จริง)

| ID         | คาด                                                                   | ได้จริง                    | สถานะ |
| ---------- | --------------------------------------------------------------------- | -------------------------- | ----- |
| TC-HDL-A01 | AC1 submit happy 200                                                  | ตรงตามคาด                  | ✅    |
| TC-HDL-A02 | AC2 submit forbidden 403                                              | ตรงตามคาด                  | ✅    |
| TC-HDL-A03 | AC2b submit conflict 409                                              | ตรงตามคาด                  | ✅    |
| TC-HDL-A04 | AC3 approve creator token → 403 (route RBAC จริง)                     | ตรงตามคาด, svc ไม่ถูกเรียก | ✅    |
| TC-HDL-A05 | AC3+Decision3 approve admin token → 403 (no bypass)                   | ตรงตามคาด                  | ✅    |
| TC-HDL-A06 | AC4 approve happy 200 ครบ 5 field                                     | ตรงตามคาด                  | ✅    |
| TC-HDL-A07 | AC5 approve validation 400                                            | ตรงตามคาด                  | ✅    |
| TC-HDL-A08 | AC5b approve conflict 409                                             | ตรงตามคาด                  | ✅    |
| TC-HDL-A09 | AC6 reject happy 200                                                  | ตรงตามคาด                  | ✅    |
| TC-HDL-A10 | AC3 reject creator token → 403                                        | ตรงตามคาด                  | ✅    |
| TC-HDL-A11 | AC8 stream signature 200, image/*                                     | ตรงตามคาด                  | ✅    |
| TC-HDL-A12 | AC8b not found 404                                                    | ตรงตามคาด                  | ✅    |
| TC-HDL-A13 | Decision5 GET /quotations/:id ไม่มี `approved_signature_path` ใน body | ตรงตามคาด                  | ✅    |
| TC-HDL-A14 | Decision1 status=pending_approval ผ่าน binding                        | ตรงตามคาด                  | ✅    |
| TC-HDL-A15 | Decision1 status=sent (ค่าเก่า) → 400, service ไม่ถูกเรียก            | ตรงตามคาด                  | ✅    |

### Frontend — QuotationDetailPage (AC9)

| ID               | คาด                                                  | ได้จริง   | สถานะ |
| ---------------- | ---------------------------------------------------- | --------- | ----- |
| TC-FE-DETAIL-A01 | Submit แสดงสำหรับ owner creator                      | ตรงตามคาด | ✅    |
| TC-FE-DETAIL-A02 | Submit ไม่แสดงสำหรับ non-owner creator               | ตรงตามคาด | ✅    |
| TC-FE-DETAIL-A03 | Submit แสดงสำหรับ admin (bypass)                     | ตรงตามคาด | ✅    |
| TC-FE-DETAIL-A04 | คลิก Submit → POST submit, status อัปเดตหลัง refetch | ตรงตามคาด | ✅    |
| TC-FE-DETAIL-A05 | Approve+Reject แสดงสำหรับ approver                   | ตรงตามคาด | ✅    |
| TC-FE-DETAIL-A06 | Approve/Reject ไม่แสดงสำหรับ creator                 | ตรงตามคาด | ✅    |
| TC-FE-DETAIL-A07 | คลิก Approve → POST approve                          | ตรงตามคาด | ✅    |
| TC-FE-DETAIL-A08 | error 400 แสดงใน role="alert"                        | ตรงตามคาด | ✅    |
| TC-FE-DETAIL-A09 | ส่วน Approved แสดงชื่อ/ตำแหน่ง/วันที่/รูป blob:      | ตรงตามคาด | ✅    |

### Frontend — QuotationListPage (Decision 1)

| ID             | คาด                                                      | ได้จริง   | สถานะ |
| -------------- | -------------------------------------------------------- | --------- | ----- |
| TC-FE-LIST-A01 | label "Pending Approval" แทน "Sent", query param ถูกต้อง | ตรงตามคาด | ✅    |

## Acceptance Criteria — ตรวจครบ

| AC                                                          | สถานะ | หลักฐาน                                                           |
| ----------------------------------------------------------- | ----- | ----------------------------------------------------------------- |
| AC1 submit happy                                            | ✅    | TC-SVC-A01, TC-HDL-A01, TC-FE-DETAIL-A04                          |
| AC2 submit non-owner forbidden                              | ✅    | TC-SVC-A02, TC-HDL-A02, TC-FE-DETAIL-A02                          |
| AC2b submit wrong status conflict                           | ✅    | TC-SVC-A04, TC-HDL-A03                                            |
| AC3 approve/reject non-approver forbidden (no admin bypass) | ✅    | TC-SVC-A06/A07/A14, TC-HDL-A04/A05/A10, TC-FE-DETAIL-A06          |
| AC4 approve happy + snapshot                                | ✅    | TC-SVC-A08/A09, TC-HDL-A06, TC-FE-DETAIL-A07/A09                  |
| AC5 approver ไม่มีลายเซ็น → 400                             | ✅    | TC-SVC-A10, TC-HDL-A07, TC-FE-DETAIL-A08                          |
| AC5b approve/reject wrong status conflict                   | ✅    | TC-SVC-A11/A12, TC-HDL-A08                                        |
| AC6 reject happy                                            | ✅    | TC-SVC-A13, TC-HDL-A09                                            |
| AC7 regression edit/delete ยัง forbidden บน non-draft ใหม่  | ✅    | TC-SVC-A15/A16                                                    |
| AC8 stream signature happy                                  | ✅    | TC-SVC-A17, TC-HDL-A11, TC-FE-DETAIL-A09 (repo TC-REPO-A03 รอ CI) |
| AC8b stream signature not found                             | ✅    | TC-SVC-A18/A19, TC-HDL-A12                                        |
| AC9 frontend ปุ่ม/ส่วนแสดง                                  | ✅    | TC-FE-DETAIL-A01..A09, TC-FE-LIST-A01                             |

**AC ครบ 9/9 (AC1–AC9) + 2/2 sub-AC (AC2b, AC5b) = ครบทั้งหมดตามแผน**

## Review เชิงลึก (ตรวจโค้ดจริง นอกเหนือจาก test case)

ตรวจตาม checklist ที่ main agent ระบุ ทุกข้อ **ผ่าน**:

1. **RBAC ordering** — `ApproveQuotation`/`RejectQuotation` เช็ค `role != "approver"` **ก่อน** เรียก `s.repo.FindByID` (`backend/internal/service/quotation_service.go:311-317`, `:355-361`) — ยืนยันด้วย mock `AssertNotCalled("FindByID", ...)` ใน TC-SVC-A06/A07/A14 ผ่านจริง ไม่ใช่แค่ mock ยอมให้เรียก
2. **ไม่มี admin bypass สำหรับ approve/reject** — ทั้งระดับ route (`quotationsApproval := engine.Group(..., middleware.RequireRole("approver"))`, `router/router.go:70-74`) และระดับ service (defense-in-depth, `quotation_service.go:311`, `:355`) — TC-HDL-A05/TC-SVC-A07 ยืนยันด้วย token จริง role `admin` ก็ยัง 403
3. **wrong status → 409 ไม่ใช่ 403** — `SubmitQuotation`/`ApproveQuotation`/`RejectQuotation` คืน `ErrConflict` เมื่อ status ไม่ตรง precondition (`quotation_service.go:294-296`, `:318-320`, `:362-364`) แยกชัดจาก ownership/role ที่คืน `ErrForbidden`
4. **atomic TransitionStatus** — `repository/quotation_repository.go:229-240` ใช้ `Model().Where("id=? AND status=?").Updates(map)` ไม่ใช่ `Update()` เดิม, `RowsAffected==0 → ErrStatusConflict` ตรง spec (Decision 7) ยืนยันด้วย TC-SVC-A05 (race translates to ErrConflict) แต่ตัวจริงระดับ DB (TC-REPO-A01/A02) ยังไม่ได้รันเพราะไม่มี Docker
5. **snapshot ไม่ re-derive ตอน read** — `GetQuotation` (`quotation_service.go:263-269`) ไม่เรียก `s.userRepo` เลย อ่านจาก field ที่ persist ไว้ตรงๆ ยืนยันด้วย TC-SVC-A09 ที่ตั้งใจไม่ stub `userRepo.FindByID` เลย (ถ้า live-lookup จะ panic) — grep ยืนยัน `GetQuotation` ไม่มี reference ถึง `userRepo`
6. **ไม่ leak path ลายเซ็น** — grep `approved_signature_path`/`ApprovedSignaturePath` ทั้งโปรเจกต์: มีเฉพาะใน model/repository/service (internal) และ test assertion เชิง regression (`TestQuotationHandler_TC_HDL_A13...` เช็คว่า body ไม่มี substring นี้) — `dto/quotation_dto.go` มีแค่ `HasApprovedSignature bool` ไม่มี raw path field เลย ลายเซ็นเสิร์ฟผ่าน `GetApprovalSignature` (stream) เท่านั้น ตรง Decision 5
7. **approver ไม่มีลายเซ็น → 400** — `if approver.SignatureImagePath == nil { return nil, ErrValidation }` (`quotation_service.go:325-327`) เช็คหลัง status check (ไม่ query โดยไม่จำเป็น) — TC-SVC-A10/A11 ยืนยัน ordering
8. **immutability** — guard เดิม `if existing.Status != "draft" { return ErrForbidden }` ใน `UpdateQuotation`/`DeleteQuotation` (บรรทัด 160-162, 250-252) ไม่ถูกแก้เลย ครอบคลุม `pending_approval`/`approved`/`rejected` โดยอัตโนมัติเพราะ enum ใหม่ยังเข้าเงื่อนไข `!= "draft"` — TC-SVC-A15/A16 (regression) ผ่านจริง
9. **migration 000005** — up: backfill `sent→pending_approval` ก่อน DROP/ADD constraint ใหม่ (`draft,pending_approval,approved,rejected`) + เพิ่ม 5 column ตรง spec; down: DROP 5 column + คืน constraint เดิม (รวม `'sent'`) — reversible ครบ ตรง plan 1:1
10. **convention/response/error** — ทุก error sentinel (`ErrForbidden`/`ErrConflict`/`ErrValidation`/`ErrNotFound`) map ผ่าน `pkg/response` เดิม (ไม่ปั้น JSON เอง), handler ไม่มี business logic (แค่ parse+call service+respond), ไม่มี log statement ใน service ที่แตะ PII/secret (grep ไม่พบ `log.`/`logger.` ใน `quotation_service.go`), naming ตรง convention (Go exported PascalCase, FE hook `use` + PascalCase, component PascalCase)
11. **List/query whitelist** — `ListQuotationQuery.Status` oneof อัปเดตเป็น `draft pending_approval approved rejected` (ตัด `sent` ออก) ตรง Decision 1, sort/query-key whitelist เดิมไม่กระทบ (TC-HDL-A14/A15 ยืนยัน)

## ปัญหาที่พบ

ไม่พบปัญหาระดับ Critical/Warning จาก code review หรือ test run

- 🟡 Suggestion: **TC-REPO-A01/A02/A03 (3 เคส) ยังไม่ได้รันจริงผ่าน Docker/testcontainers** บนเครื่องนี้ (ไม่มี Docker daemon) — ยืนยันได้แค่ระดับ compile (`go vet`/`go build`/`go test -short`) และ manual code review ที่ implementation ตรง contract 1:1 (ดูข้อ 4 ด้านบน) แนะนำให้ CI pipeline ที่มี Docker รัน `go test ./...` (ไม่ใส่ `-short`) อีกครั้งก่อน merge เพื่อยืนยัน 100% — ไม่ใช่ blocker เพราะ logic ผ่านการตรวจโค้ดแล้วและ service-level test (TC-SVC-A05) ครอบคลุม race-condition behavior ทางอ้อมผ่าน mock
- 🟡 Suggestion: ไฟล์ `docs/RESUME-OVERNIGHT.md` (untracked) หลุดอยู่ใน working tree — ไม่เกี่ยวกับ scope ของ feature นี้ (ดูเหมือน artifact ของ dev agent) ไม่กระทบผลตรวจ แต่ควรพิจารณาลบ/ไม่ commit ถ้าไม่ตั้งใจ
- 🟡 Suggestion: ข้อจำกัดที่ plan ระบุไว้แล้วว่ายอมรับ (ไม่ใช่บั๊ก แต่ขอย้ำเพื่อบันทึก): guard เดิม `status != draft` ของ Update/Delete ตอบ 403 (ไม่ใช่ 409 ตาม convention ใหม่ของ transition) — ตามที่ระบุใน Decision 2/ความเสี่ยงข้อ 5 ของแผน เป็นการตัดสินใจที่ pin ไว้แล้ว ไม่ต้องแก้

## สรุปสิ่งที่ต้องให้ dev แก้

ไม่มี — implementation ครบตาม test case + AC ทั้งหมด, convention/security review ผ่านทุกข้อ

รอเพียง: ให้ CI ที่มี Docker รัน `go test ./...` (ไม่ short mode) เพื่อยืนยัน TC-REPO-A01/A02/A03 อีกครั้งก่อน merge ขั้นสุดท้าย (ไม่ block การ PASS รอบนี้เพราะ code review ยืนยัน implementation ตรง spec แล้ว)
