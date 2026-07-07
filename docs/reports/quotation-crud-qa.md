# QA Report: CRUD ใบเสนอราคา + Calc Engine (slug: quotation-crud)

อ้างอิง: docs/plans/quotation-crud.md · docs/tests/quotation-crud-testcases.md
รอบที่: 1 → 2 | วันที่: 2026-07-08

---

## ✅ รอบที่ 2 — ผลรวม: PASS (verify โดย main agent/reviewer)

ปิดทุก blocker จากรอบ 1 แล้ว รันจริงยืนยัน:

```
BACKEND : gofmt สะอาด · go vet ผ่าน · go build ผ่าน · go test -short → 87 PASS / 4 SKIP / 0 FAIL
FRONTEND: npm run lint 0 error · npm test → 26/26 PASS (11 files) · npm run build สำเร็จ
```

**สิ่งที่แก้ในรอบ 2:**

- 🔴→✅ **Critical (404):** เพิ่ม helper `translateNotFound` แปลง `gorm.ErrRecordNotFound` → `service.ErrNotFound`
  ที่ `GetQuotation`/`UpdateQuotation`/`DeleteQuotation` (quotation_service.go) — ตอนนี้ id ที่ไม่มีคืน **404 NOT_FOUND**
  ไม่ใช่ 500 อีกต่อไป + **เพิ่ม regression test TC17/18/19** (service layer) ที่ปิด gap เดิม (suite ไม่เคยมี not-found case)
- 🟠→✅ **Delete ไม่ถึง UI:** ต่อ `useDeleteQuotation` เข้าปุ่ม Delete (draft-only + confirm + invalidate cache) + test
- 🟠→✅ **ไม่มี detail page/route:** เพิ่ม `QuotationDetailPage.jsx` + route `/quotations/:id` (RequireAuth, approver ดูได้ตาม Decision #5) + smoke test
- 🟠→✅ **standard-library ที่บังคับแต่ไม่ได้ใช้:** ใช้ครบ — `@tanstack/react-table` (ตาราง list), `react-select` (status filter), `react-datepicker` (date fields) โดย test เดิมยังเขียว
  - แถม dev จับ **timezone bug**: datepicker เดิมใช้ `toISOString()` (UTC) ทำวันเลื่อน −1 ใน TZ ที่ >UTC (เช่น ไทย +7) → แก้เป็น `dayjs(...).format('YYYY-MM-DD')` (local)
- 🟡→✅ `parseDatePtr` คืน `ErrValidation` เมื่อ date ผิดรูป (เลิก swallow เงียบ)
- 🟡→✅ `date_gte`/`date_lte` ตอนนี้ validate รูปแบบ `datetime=2006-01-02` (binding) → ผิดรูป = 400 ไม่ใช่ 500

**Tech-debt ที่จงใจเลื่อน (บันทึกไว้ ไม่ทิ้งเงียบ) — ไม่กระทบ AC/ความถูกต้อง:**

- ยังไม่ลง VISUAL_DESIGN_GUIDE styling (แบรนด์ i-MAXX สี/เลย์เอาต์) เต็มรูป
- ยังไม่แตก component ย่อย (`QuotationHeaderFields.jsx`, `QuotationItemsTable.jsx` ฯลฯ ตาม plan Tasks 15–19) — ปัจจุบันรวมในหน้า page
- `TC-REPO-01..04` (integration) ยังต้องรันกับ Postgres จริงใน CI ที่มี Docker ก่อนเชื่อว่าเขียวจริง (เครื่อง dev นี้ไม่มี Docker)

> รายละเอียดรอบ 1 (ที่นำไปสู่การแก้ด้านบน) เก็บไว้ด้านล่างเพื่ออ้างอิง

---

## รอบที่ 1 — ผลรวม: FAIL

- test ผ่าน 50/50 ที่รันได้จริงในเครื่องนี้ (backend 40/40 + frontend 10/10) — 4 integration test
  (`TC-REPO-01..04`) **ถูก skip ด้วย `-short`** เพราะ Docker ไม่ทำงานในเครื่องนี้ (ตามที่
  test-case-writer ระบุไว้แล้วว่าคาดหวัง — **ยังไม่ได้ verify จริงกับ Postgres**, ต้องรันซ้ำใน CI/
  เครื่องที่มี Docker ก่อนเชื่อว่าเขียวจริง)
- AC ที่มี test/review ครอบ: 19/19 (AC1–AC19) — **แต่พบบั๊ก Critical 1 จุดจาก code review ที่ไม่มี
  test ใดจับได้ (gap ในชุด test เอง) ซึ่งขัดกับสัญญาที่ระบุไว้ชัดเจนใน test doc §0.2 และ
  `api-response.md`** จึงตัดสิน **FAIL** ทั้งที่ test suite ที่ออกแบบไว้เขียวครบ — เหตุผลอยู่ท้าย
  รายงาน (หัวข้อ "ปัญหาที่พบ" #1)

รันจริงยืนยันแล้ว (ไม่ใช่เชื่อคำรายงานอย่างเดียว):

```
cd backend && gofmt -l . && go vet ./... && go build ./... && go test -short -count=1 ./...
→ gofmt: สะอาด, vet: ผ่าน, build: ผ่าน, test: PASS ทุก package (repo integration SKIP ตามคาด)

cd frontend && npm run lint && npm test -- --run && npm run build
→ lint: 0 error (2 warning ที่มีอยู่ก่อนแล้ว ไม่เกี่ยวกับ feature นี้), test: 9 files / 23 tests PASS
  (10 test ใหม่ของ quotation ผ่านหมด), build: สำเร็จ
```

---

## ผลราย Test Case

### Backend — Calc engine (`internal/service/quotation_calc_test.go`)

| ID         | คาด                                                   | ได้จริง | สถานะ |
| ---------- | ----------------------------------------------------- | ------- | ----- |
| TC-CALC-01 | 200000, 75150                                         | ตรง     | ✅    |
| TC-CALC-02 | subtotal=275150, base=260000, vat=18200, total=278200 | ตรง     | ✅    |
| TC-CALC-03 | vat=74 (tie ปัดขึ้น), total=1124                      | ตรง     | ✅    |
| TC-CALC-04 | err=ErrValidation                                     | ตรง     | ✅    |
| TC-CALC-05 | base=0,vat=0,total=0,err=nil                          | ตรง     | ✅    |
| TC-CALC-06 | err=ErrValidation (discount ติดลบ)                    | ตรง     | ✅    |
| TC-CALC-07 | 25050 / 100000                                        | ตรง     | ✅    |
| TC-CALC-08 | 1 (ปัดขึ้นจาก 0.006)                                  | ตรง     | ✅    |

### Backend — Service (`internal/service/quotation_service_test.go`)

| ID        | คาด                                                      | ได้จริง | สถานะ |
| --------- | -------------------------------------------------------- | ------- | ----- |
| TC-SVC-01 | ref_no=QT2607001, draft, signee snapshot, totals ตรง AC1 | ตรง     | ✅    |
| TC-SVC-02 | สำเร็จครั้งที่ 3, Create ถูกเรียก 3 ครั้ง                | ตรง     | ✅    |
| TC-SVC-03 | ErrConflict, Create ถูกเรียกพอดี 5 ครั้ง                 | ตรง     | ✅    |
| TC-SVC-04 | vat=0.74, total=11.24 (full flow)                        | ตรง     | ✅    |
| TC-SVC-05 | ErrValidation, repo.Create ไม่ถูกเรียก                   | ตรง     | ✅    |
| TC-SVC-06 | สำเร็จ, base/vat/total=0.00                              | ตรง     | ✅    |
| TC-SVC-07 | ErrValidation (valid_until<date)                         | ตรง     | ✅    |
| TC-SVC-08 | สำเร็จ, repo.Update ถูกเรียกด้วยค่าที่คำนวณใหม่          | ตรง     | ✅    |
| TC-SVC-09 | ErrForbidden, repo.Update ไม่ถูกเรียก                    | ตรง     | ✅    |
| TC-SVC-10 | ErrForbidden (ownership, creator≠owner)                  | ตรง     | ✅    |
| TC-SVC-11 | สำเร็จ (admin bypass)                                    | ตรง     | ✅    |
| TC-SVC-12 | company_signee_name ไม่เปลี่ยนตาม profile ปัจจุบัน       | ตรง     | ✅    |
| TC-SVC-13 | สำเร็จ, repo.Delete ถูกเรียก                             | ตรง     | ✅    |
| TC-SVC-14 | ErrForbidden, repo.Delete ไม่ถูกเรียก                    | ตรง     | ✅    |
| TC-SVC-15 | ErrForbidden (ownership)                                 | ตรง     | ✅    |
| TC-SVC-16 | สำเร็จ (admin)                                           | ตรง     | ✅    |

### Backend — Handler (`internal/handler/quotation_handler_test.go`)

| ID        | คาด                                                | ได้จริง | สถานะ |
| --------- | -------------------------------------------------- | ------- | ----- |
| TC-HDL-01 | 200, body ครบทุก key (AC12)                        | ตรง     | ✅    |
| TC-HDL-02 | 201, data.reference_no ตรง mock                    | ตรง     | ✅    |
| TC-HDL-03 | 400 VALIDATION_ERROR                               | ตรง     | ✅    |
| TC-HDL-04 | 403 FORBIDDEN                                      | ตรง     | ✅    |
| TC-HDL-05 | 204 ไม่มี body                                     | ตรง     | ✅    |
| TC-HDL-06 | 403 FORBIDDEN                                      | ตรง     | ✅    |
| TC-HDL-07 | 200, meta ตรง param                                | ตรง     | ✅    |
| TC-HDL-08 | 400 (sort นอก whitelist), service ไม่ถูกเรียก      | ตรง     | ✅    |
| TC-HDL-09 | 400 (filter นอก whitelist)                         | ตรง     | ✅    |
| TC-HDL-10 | 400 (page_size=1000)                               | ตรง     | ✅    |
| TC-HDL-11 | 403 (approver POST)                                | ตรง     | ✅    |
| TC-HDL-12 | 200 (approver GET list)                            | ตรง     | ✅    |
| TC-HDL-13 | 200 (approver GET detail)                          | ตรง     | ✅    |
| TC-HDL-14 | 401 (ไม่มี token)                                  | ตรง     | ✅    |
| TC-HDL-15 | 201 (creator POST ผ่าน)                            | ตรง     | ✅    |
| TC-HDL-16 | 500 INTERNAL_ERROR, ไม่ leak "dsn="/"conn refused" | ตรง     | ✅    |

### Backend — Repository integration (`internal/repository/quotation_repository_test.go`)

| ID         | คาด                               | ได้จริง                           | สถานะ                 |
| ---------- | --------------------------------- | --------------------------------- | --------------------- |
| TC-REPO-01 | ref_no วิ่งต่อเนื่อง              | **SKIP** (`-short`, ไม่มี Docker) | ⏭️ ยังไม่ verify จริง |
| TC-REPO-02 | ErrDuplicateReferenceNo           | **SKIP**                          | ⏭️ ยังไม่ verify จริง |
| TC-REPO-03 | cascade delete items              | **SKIP**                          | ⏭️ ยังไม่ verify จริง |
| TC-REPO-04 | full replace items ใน transaction | **SKIP**                          | ⏭️ ยังไม่ verify จริง |

> Code review: ไฟล์ compile ผ่าน `go vet`/`go build` เต็ม, ใช้ `tcpostgres` alias แก้ import ชน,
> `MappedPort` เรียกถูก signature, ไม่มี unused import — เนื้อหา logic ตรงกับ contract §0.2
> (`ErrDuplicateReferenceNo`, `gorm.ErrRecordNotFound`) แต่ **ต้องรันจริงกับ Postgres ใน CI ก่อนเชื่อ
> ว่าเขียว** (ยังไม่เคย exercise จริงสักครั้งในสภาพแวดล้อมใดเลยตามที่บันทึกไว้ใน test doc)

### Frontend — calc / form / list

| ID            | คาด                                               | ได้จริง                                                                                                                                                                        | สถานะ                                      |
| ------------- | ------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------ |
| TC-FE-CALC-01 | 2000.00, 751.50                                   | ตรง                                                                                                                                                                            | ✅                                         |
| TC-FE-CALC-02 | subtotal=2751.50, vat=182.00, total=2782.00       | ตรง                                                                                                                                                                            | ✅                                         |
| TC-FE-CALC-03 | vat=0.74, total=11.24 (tie)                       | ตรง                                                                                                                                                                            | ✅                                         |
| TC-FE-CALC-04 | error='DISCOUNT_EXCEEDS_SUBTOTAL', vat/total=null | ตรง                                                                                                                                                                            | ✅                                         |
| TC-FE-CALC-05 | vat=0,total=0,error=null                          | ตรง                                                                                                                                                                            | ✅                                         |
| TC-FE-FORM-01 | summary 2,751.50/151.50/182.00/2,782.00           | ตรง                                                                                                                                                                            | ✅                                         |
| TC-FE-FORM-02 | discount-error ปรากฏ + submit disabled            | ตรง                                                                                                                                                                            | ✅                                         |
| TC-FE-FORM-03 | fields disabled, ไม่มีปุ่ม save/delete            | ผ่านทาง assertion แต่ **vacuous สำหรับ "delete"** (ดูปัญหา #3 — ปุ่ม delete ไม่เคยถูกสร้างแม้ตอน draft)                                                                        | ⚠️ ผ่านแบบ technicality                    |
| TC-FE-LIST-01 | list แสดง ref_no ใหม่หลัง invalidate              | ตรง                                                                                                                                                                            | ✅                                         |
| TC-FE-LIST-02 | query string มี page/page_size                    | ตรง (แต่ถูกลดทอนเป็น smoke test ตอน mount เท่านั้น ไม่ได้ทดสอบ "คลิกเปลี่ยนหน้า/sort" ตามที่ตารางเดิมออกแบบไว้ เพราะ UI ไม่มีปุ่ม/dropdown เปลี่ยนหน้า-sort จริง — ดูปัญหา #2) | ⚠️ ผ่านแบบ smoke test อ่อนกว่าที่ออกแบบไว้ |

**สรุป**: backend 40/40 PASS (unit/handler), 4 integration SKIP (ตามคาด ไม่นับเป็น FAIL);
frontend 10/10 PASS. รวมที่รันได้จริง 50/50 PASS.

---

## ปัญหาที่พบ (เรียงตามความรุนแรง)

- 🔴 **Critical — GET/PUT/DELETE `/quotations/:id` ตอบ 500 `INTERNAL_ERROR` แทนที่จะเป็น 404
  `NOT_FOUND` เมื่อ id ไม่มีอยู่จริง**
  ไฟล์: `backend/internal/service/quotation_service.go` บรรทัด 126–129 (`UpdateQuotation`),
  200–203 (`DeleteQuotation`), 220–224 (`GetQuotation`) — ทั้งสามจุดเรียก
  `s.repo.FindByID(ctx, id)` แล้ว `if err != nil { return nil, err }` **ส่ง `gorm.ErrRecordNotFound`
  ดิบ ๆ กลับไปโดยไม่แปล** เมื่อไปถึง `pkg/response.mapError` (`response.go:99-119`) จะไม่ match
  case ไหนเลย (เทียบกับ `service.ErrNotFound` ซึ่งเป็นคนละ error กับ `gorm.ErrRecordNotFound`) จึง
  ตกไปที่ `default` → 500 `INTERNAL_ERROR` ทั้งที่ควรเป็น 404 `NOT_FOUND` ตาม
  `.claude/docs/api-response.md` (ตาราง HTTP status: "404 | ไม่พบข้อมูล") **และขัดกับสัญญาที่
  test-case-writer ระบุไว้ชัดเจนใน `docs/tests/quotation-crud-testcases.md` §0.2**: "หน้าที่แปลเป็น
  `service.ErrNotFound`/`service.ErrConflict` (409/404 ตาม api-response.md) เป็นของ **service
  layer** เท่านั้น" — ไม่มี test case ใดในชุดที่ออกแบบไว้ (ทั้ง TC-SVC-\* และ TC-HDL-\*) ครอบ
  สถานการณ์ "id ไม่พบ" เลย (เป็น gap ของชุด test เอง) จึงไม่มี test จับบั๊กนี้ได้
  **วิธีแก้ที่แนะนำ**: เพิ่ม helper เช่น

  ```go
  if err != nil {
      if errors.Is(err, gorm.ErrRecordNotFound) {
          return nil, ErrNotFound
      }
      return nil, err
  }
  ```

  หลังทุกจุดที่เรียก `s.repo.FindByID` ใน `CreateQuotation`(ไม่มี)/`UpdateQuotation`/
  `DeleteQuotation`/`GetQuotation` แล้วเพิ่ม unit test ใหม่ (mock `FindByID` คืน
  `gorm.ErrRecordNotFound` → assert `errors.Is(err, service.ErrNotFound)`) — เป็นเงื่อนไขที่จะเกิด
  ได้ปกติในโปรดักชันจริง (client ยิง id ที่ลบไปแล้ว/พิมพ์ผิด) ไม่ใช่ edge case แปลก ๆ

- 🟠 **Warning — Frontend scope ตามแผนหายไปมาก แม้ AC ที่ระบุเป็นตัวเลขจะผ่าน test**
  1. ไม่มี `features/quotation/components/` เลย (Task 15–19 ในแผนระบุ `QuotationHeaderFields.jsx`,
     `QuotationItemsTable.jsx`, `QuotationSummary.jsx`, `SignOffSection.jsx`,
     `QuotationPageHeader.jsx`) — โค้ดทั้งหมดถูกอัดรวมไว้ใน `QuotationFormPage.jsx`/
     `QuotationListPage.jsx` เป็น plain `<input>`/`<table>` ล้วน
  2. ไม่มี `QuotationDetailPage.jsx` และไม่มี route `/quotations/:id` เลยใน
     `frontend/src/routes/AppRoutes.jsx` (Task 22–23 ระบุไว้ชัดว่าต้องมี 4 route: list/new/detail/
     edit — ตอนนี้มีแค่ 3 เส้นทาง ไม่มี detail-view)
  3. เพิ่ม dependency `@tanstack/react-table`, `react-select`, `react-datepicker`, `dayjs` ใน
     `package.json` ตามแผน **แต่ไม่ได้ถูก import/ใช้งานจริงที่ไหนเลย** (grep ไม่พบการ import ใน
     `frontend/src/features/quotation/**`) — ขัดกับ `.claude/rules/frontend.md` ที่ระบุว่า
     Table **ต้อง**ใช้ TanStack Table, Datepicker **ต้อง**ใช้ react-datepicker ("ห้ามใช้ตัวอื่นแทน
     โดยไม่ตกลงกับทีม") ตอนนี้ใช้ `<input type="date">`/`<table>` ธรรมดาแทนทั้งหมด
  4. ไม่มีการจัดสไตล์ตาม `VISUAL_DESIGN_GUIDE.md` เลย (ไม่มีโลโก้ i-MAXX, ไม่มีสี `--primary`, ไม่มี
     card/shadow ตามที่ DoD ของแผนระบุไว้ตรง ๆ ในหัวข้อ "เป้าหมาย/Definition of Done")
     ไฟล์: `frontend/src/features/quotation/pages/QuotationFormPage.jsx`,
     `frontend/src/features/quotation/pages/QuotationListPage.jsx`,
     `frontend/src/routes/AppRoutes.jsx`
     **แนะนำ**: เนื่องจากไม่มี AC ตัวเลขไหนระบุชื่อไฟล์ component ตรง ๆ test จึงยังผ่าน แต่ scope ตามแผน
     ไม่ครบ — ควรแจ้ง dev เพิ่มรอบถัดไป (แตกไฟล์ตาม task list, ต่อ TanStack Table/react-datepicker/
     react-select เข้ากับของที่ install ไว้แล้ว, เพิ่ม Detail page + route, ปรับสไตล์ตาม
     VISUAL_DESIGN_GUIDE) ก่อนถือว่า feature เสร็จสมบูรณ์ตามแผน

- 🟠 **Warning — ฟังก์ชันลบใบเสนอราคา (Delete) ไม่มีทางเข้าถึงได้จาก UI เลย (dead hook)**
  ไฟล์: `frontend/src/features/quotation/hooks/useDeleteQuotation.js` ถูกสร้างไว้ครบ (invalidate
  ถูกต้อง) แต่**ไม่ถูก import/เรียกใช้ที่ไหนเลย** ใน `QuotationListPage.jsx`/`QuotationFormPage.jsx`
  — ไม่มีปุ่ม "Delete"/"ลบ" ปรากฏแม้แต่ตอน quotation เป็น `draft` (ที่ควรลบได้ตาม AC6/เป้าหมายของแผน
  "สร้าง/แก้ไข/ลบใบเสนอราคา") ทำให้ `TC-FE-FORM-03` (assert "ไม่มีปุ่ม save/delete" ตอน non-draft)
  ผ่านแบบ **vacuous** (ปุ่ม delete ไม่เคยมีอยู่แม้ตอน draft ก็ตาม จึงไม่ได้พิสูจน์ว่า logic "ซ่อนเมื่อ
  non-draft" ทำงานถูกต้องจริง เพราะไม่มี branch ให้ทดสอบ) Backend DELETE endpoint ทำงานถูกต้อง 100%
  (ยืนยันจาก TC-HDL-05/06 + TC-SVC-13-16) แต่ end-user เข้าถึงไม่ได้ผ่านหน้าเว็บ
  **แนะนำ**: เพิ่มปุ่ม Delete ในหน้า list หรือหน้า form (edit mode, เฉพาะ draft) เรียก
  `useDeleteQuotation` จริง แล้วปรับ `TC-FE-FORM-03`/เพิ่ม test ใหม่ให้ยืนยัน "ปุ่มมีตอน draft,
  หายไปตอน non-draft" แทนที่จะเช็คแค่ "ไม่มีตอน non-draft"

- 🟡 **Suggestion — `parseDatePtr` กลืน parse error เงียบ ๆ**
  ไฟล์: `backend/internal/service/quotation_service.go:266-274` — ถ้า
  `req.CustomerSigneeDate`/`req.CustomerSigneePosition` ส่งค่าที่ parse วันที่ไม่ผ่าน (format ผิด)
  ฟังก์ชันคืน `nil` เงียบ ๆ แทนที่จะ error กลับไปเป็น `ErrValidation` เหมือนจุดอื่น ๆ ที่ parse
  `date`/`valid_until` (บรรทัด 41-48) ทำให้ client ส่งค่าผิด format แล้วได้ค่า `null` แทนที่จะได้รับ
  แจ้งเตือนว่า format ผิด — ไม่ critical เพราะ field นี้ optional แต่ inconsistent กับ validation
  ฟิลด์อื่นในไฟล์เดียวกัน

- 🟡 **Suggestion — `date_gte`/`date_lte` ไม่ validate รูปแบบ `YYYY-MM-DD`**
  ไฟล์: `backend/internal/repository/quotation_repository.go:133-138` — ใช้ค่า string ดิบจาก
  query ต่อเข้า `Where("date >= ?", query.DateGte)` แบบ parameterized (**ไม่มีช่องโหว่ SQL
  injection** เพราะเป็น parameterized query ถูกต้องแล้ว) แต่ถ้า client ส่งค่าที่ไม่ใช่วันที่ (เช่น
  `date_gte=abc`) Postgres จะ error กลายเป็น 500 แทนที่จะเป็น 400 `VALIDATION_ERROR` ที่ควรจะเป็น
  (เข้าเงื่อนไขเดียวกับ AC19 ที่ควร fail ด้วยข้อความ generic แต่ status code ที่ถูกต้องกว่าคือ 400)
  ไม่มี test ครอบเคสนี้ — แนะนำ validate format ที่ handler/DTO ด้วย `binding` custom validator หรือ
  service layer ก่อนส่งเข้า repository

- 🟡 **Suggestion — integration test (TC-REPO-01..04) ยังไม่เคยรันจริงสักครั้ง**
  ตามที่ระบุไว้แล้วใน test doc — Docker ไม่ทำงานในเครื่องนี้ ต้องยืนยันใน CI/เครื่องที่มี Docker
  ก่อน merge ให้มั่นใจว่า unique constraint / cascade delete / transaction replace ทำงานถูกต้องกับ
  Postgres จริง (ไม่ใช่แค่ compile ผ่าน)

---

## จุดที่ตรวจแล้วผ่าน (ไม่มีปัญหา)

- **Calc engine**: ใช้ int64 สตางค์ล้วนตาม Decision #1 เป๊ะ (`roundHalfUpCents`,
  `calcLineTotalCents`, `calcTotals` ใน `quotation_calc.go`) ไม่มีจุดไหนคูณ float แล้วปัดย้อนหลัง —
  ตรงกับ AC1/AC2 ทุกตัวเลข ทั้ง backend และ frontend (`calcQuotation.js` mirror สูตรเดียวกัน)
- **RBAC/ownership**: ตัดสินจาก `userID`/`role` ที่ middleware set ใน `gin.Context` เท่านั้น (ยืนยัน
  จาก handler test ที่ใช้ `middleware.Auth` + `fakeVerifier` จริง ไม่ได้ mock ข้าม role check) — ไม่
  รับ role จาก client input เลย
  ตรงตาม `.claude/docs/auth.md`
- **SQL**: ทุก query ผ่าน GORM parameterized (`Where("x = ?", v)`) ไม่มีจุดต่อ SQL string จาก user
  input เลยแม้แต่ในฟังก์ชัน `List`/`NextReferenceNo` ที่มี `LIKE`/`ILIKE`
- **Response envelope**: ตรง `.claude/docs/api-response.md` ทุก endpoint (`Success`/`List`/`Fail`
  ใช้ผ่าน `pkg/response/` เท่านั้น ไม่มีการปั้น JSON เอง); error 500 ไม่ leak stack/SQL/path
  (AC19 ผ่าน, ยืนยันด้วย TC-HDL-16) — **ยกเว้นเรื่อง status code 500-vs-404 ที่แจ้งไว้ข้างบน (บั๊ก
  Critical) ซึ่งเป็นเรื่อง status code ผิด ไม่ใช่เรื่อง leak ข้อมูล**
- **List query whitelist**: sort/filter/page_size ปฏิเสธด้วย 400 ตาม Decision #10 ครบ (ยืนยัน
  TC-HDL-08/09/10)
- **AC18 (PII ใน log)**: grep ไฟล์ backend ใหม่ทั้งหมด (`quotation_service.go`,
  `quotation_handler.go`, `quotation_repository.go`) ไม่พบ `log.`/`fmt.Print`/`logger.`/`slog.`
  เลยสักบรรทัด — ไม่มีความเสี่ยง PII หลุดจาก log ของ feature นี้
- **naming convention**: ตรวจไฟล์ Go ใหม่ทั้งหมด — struct field PascalCase + json snake_case
  ครบถ้วน, ตัวแปร local camelCase, ไฟล์ snake_case, package lowercase — ไม่พบข้อผิดตาม
  `.claude/rules/naming-conventions.md`
- **config**: ไม่มีการเพิ่ม env var ใหม่ (`git diff` ยืนยันไม่มีการแก้ `.env.example`/`config.md`)
  ตรงตาม Decision #12
- **security**: bcrypt/JWT ไม่ถูกแตะต้อง (นอกสโคป), ไม่มี CORS `*` ใหม่ถูกเพิ่ม, ไม่มี secret
  hardcode ในไฟล์ quotation ใหม่
- **บั๊ก 4 จุดที่ main agent (reviewer) แก้ในไฟล์ test**: ตรวจแล้วยืนยันว่า assertion เดิมยังตรง AC
  จริง (mock ctx arg ครบ, role ถูก set แยก "creator"/"admin" ใน TC-SVC-10/11 ตรงตาม ownership logic,
  `TestMain`+`flag.Parse()` ในไฟล์ repository test ถูกต้องตามรูปแบบ `testing.md` §3, ไม่มี dead var
  หลงเหลือที่กระทบผลลัพธ์)

---

## สรุปสิ่งที่ต้องให้ dev แก้ (เรียงตามลำดับความสำคัญ)

1. **[Critical, ต้องแก้ก่อน sign-off]** แปล `gorm.ErrRecordNotFound` → `service.ErrNotFound` ใน
   `quotation_service.go` (`GetQuotation`/`UpdateQuotation`/`DeleteQuotation`) เพื่อให้ id ที่ไม่มี
   อยู่จริงตอบ 404 แทน 500 — เพิ่ม unit test ใหม่ครอบเคสนี้ด้วย (ไม่มีในชุด TC เดิม)
2. **[Warning]** เพิ่มปุ่ม Delete ที่เรียก `useDeleteQuotation` จริงในหน้า list หรือ form (เฉพาะ
   draft) ไม่ให้ hook เป็น dead code, ปรับ/เพิ่ม test ยืนยันว่าปุ่มมีตอน draft และหายตอน non-draft
3. **[Warning]** พิจารณาทำ Detail page (`/quotations/:id`) + component decomposition ตาม Task 15-23
   ในแผน และต่อ TanStack Table/react-select/react-datepicker ที่ install ไว้แล้วให้ใช้งานจริง หรือ
   ถ้าตัดสินใจเปลี่ยนแนวทาง (พอสำหรับ MVP ด้วย plain HTML) ให้อัปเดตแผนให้ตรงกับของจริงแทน
4. **[Suggestion]** validate format `date_gte`/`date_lte`/`customer_signee_date` ให้คืน 400
   แทนที่จะปล่อยเงียบ/500
5. รัน `TC-REPO-01..04` ใน CI/เครื่องที่มี Docker จริงก่อน merge เพื่อยืนยัน integration ทำงานถูกต้อง
