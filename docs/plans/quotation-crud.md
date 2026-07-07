# Plan: CRUD ใบเสนอราคา + Calc Engine (slug: quotation-crud)

> ต่อยอดจาก `user-auth` (มีแล้ว ไม่ทำซ้ำ): `internal/model/user.go` (User/Role), JWT
> (`internal/service/token_service.go`), middleware `internal/middleware/auth.go` +
> `require_role.go` (set `userID`/`role` ใน `gin.Context`), `pkg/response/` (`Success`,
> `List`, `Fail` + `mapError` มีอยู่แล้ว — เพิ่ม case ใหม่ที่ต้องใช้เท่านั้น),
> `services/apiClient.js` (interceptor แนบ token + แกะ envelope), `AuthContext`/`useAuth`,
> `RequireAuth`/`RequireRole`. Backend Go module = `imaxx-backend`.
>
> อ้างอิงมาตรฐาน: `.claude/docs/{api-response,list-query,auth,error-logging,security,config,
backend-structure,frontend-structure,testing,standard-libraries,VISUAL_DESIGN_GUIDE}.md`
> และ `.claude/rules/{naming-conventions,backend,frontend}.md`

---

## เป้าหมาย / Definition of Done

ผู้ใช้ที่ login แล้ว (role `admin`/`creator`) สร้าง/แก้ไข/ลบใบเสนอราคา (Quotation) พร้อมรายการ
สินค้า/บริการ (QuotationItem) ได้ตราบเท่าที่สถานะยังเป็น `draft`; ระบบคำนวณ subtotal/discount/
VAT 7%/total ที่ backend เป็น source of truth (ปัดเศษครึ่งขึ้น ทศนิยม 2 ตำแหน่ง, คำนวณด้วย
เลขจำนวนเต็ม "สตางค์" ล้วน ไม่ใช้ float คูณ/หารตรง ๆ เพื่อกัน floating-point error);
ทุก role ที่ login แล้ว (`admin`/`creator`/`approver`) ดู list/detail ได้ตาม `list-query.md`;
ฟอร์ม frontend คำนวณ preview ด้วยสูตรเดียวกัน, มีตารางรายการแบบเพิ่ม/ลบแถว, และหน้าตาตาม
`VISUAL_DESIGN_GUIDE.md` (หัวกระดาษโลโก้ i-MAXX + สไตล์ enterprise calm)

DoD ถือว่าเสร็จเมื่อ:

- ทุก Acceptance Criteria (AC) ด้านล่างมี test ครอบและ PASS
- `cd backend && gofmt -l . && go vet ./... && go build ./... && go test ./...` ผ่าน
- `cd frontend && npm run lint && npm run build && npm test` ผ่าน
- calc engine คำนวณตรงตามตัวเลขที่ระบุใน AC1–AC2 เป๊ะ (ไม่ใช่ "ใกล้เคียง")
- ไม่มี PII (email/telephone/ชื่อลูกค้า) หรือ token หลุดใน log (ตาม `security.md`/`error-logging.md`)
- ไม่ต้องเพิ่ม env var ใหม่ (numbering ทำใน DB/service ล้วน) — ถ้าพบว่าจำเป็นต้องเพิ่มจริง
  ให้ sync `config.md` + `.env.example` ตามกติกา

---

## ขอบเขต

### Backend (`backend/`)

- `internal/model/quotation.go` — GORM model `Quotation`, `QuotationItem`
- `internal/dto/quotation_dto.go` — `CreateQuotationRequest`, `UpdateQuotationRequest`,
  `QuotationItemInput`, `QuotationResponse`, `QuotationItemResponse`, `ListQuotationQuery`
- `internal/service/quotation_calc.go` — calc engine ล้วน (pure function, ไม่แตะ DB) — ใช้ int64
  "สตางค์" ภายใน
- `internal/service/quotation_service.go` — business logic: gen reference_no, validate
  draft-only, ownership, orchestrate repository + calc
- `internal/repository/quotation_repository.go` — interface + gorm impl (`Create`, `FindByID`,
  `Update` (replace items ใน transaction), `Delete`, `List` ด้วย GORM scopes)
- `internal/handler/quotation_handler.go` — `Create`, `List`, `Get`, `Update`, `Delete`
- `internal/router/router.go` — เพิ่ม route group `/quotations`
- `pkg/response/response.go` — เพิ่ม case mapping ใน `mapError` เฉพาะถ้ามี sentinel ใหม่ (ดู task)
- `backend/migrations/000X_create_quotations.{up,down}.sql`,
  `000X_create_quotation_items.{up,down}.sql`
- `cmd/api/main.go` — เพิ่ม `AutoMigrate(&model.Quotation{}, &model.QuotationItem{})` (dev)

### Frontend (`frontend/src/`)

- `package.json` — เพิ่ม dependency ที่ยังไม่มี: `@tanstack/react-table`, `react-select`,
  `react-datepicker`, `dayjs` (มีระบุใน `standard-libraries.md`/`frontend.md` แต่ยังไม่ได้ติดตั้ง)
- `constants/apiEndpoints.js` — เพิ่ม `QUOTATIONS = '/quotations'`
- `features/quotation/services/quotationService.js` — `list`, `getById`, `create`, `update`, `remove`
- `features/quotation/hooks/` — `useQuotations` (list), `useQuotation` (detail),
  `useCreateQuotation`, `useUpdateQuotation`, `useDeleteQuotation` (react-query)
- `features/quotation/utils/calcQuotation.js` — calc preview ฝั่ง client (สูตร/การปัดเศษ
  ต้องตรงกับ backend เป๊ะ — ดู Decisions)
- `features/quotation/components/` — `QuotationHeaderFields.jsx` (attention/company/project/
  telephone/email + react-datepicker สำหรับ date/valid_until), `QuotationItemsTable.jsx`
  (TanStack Table แบบแก้ไขได้ในแถว + เพิ่ม/ลบแถว), `QuotationSummary.jsx` (Sub Total/Discount/
  VAT/Total), `SignOffSection.jsx` (ฝั่งลูกค้า/บริษัท), `QuotationPageHeader.jsx` (โลโก้ i-MAXX)
- `features/quotation/pages/` — `QuotationListPage.jsx`, `QuotationFormPage.jsx` (create+edit
  ใช้ฟอร์มเดียวกัน), `QuotationDetailPage.jsx`
- `routes/AppRoutes.jsx` — เพิ่ม `/quotations`, `/quotations/new`, `/quotations/:id`,
  `/quotations/:id/edit` (ครอบด้วย `RequireAuth`; create/edit ครอบเพิ่ม `RequireRole(['admin','creator'])`)

### Database

- ตาราง `quotations`: ฟิลด์ตามโจทย์ + sign-off fields (ดู task 3) + `created_by` (FK → `users.id`)
- ตาราง `quotation_items`: FK → `quotations.id` (`ON DELETE CASCADE`), `sort_order` คุมลำดับแถว
- dev = `GORM AutoMigrate`; staging/prod = ไฟล์ `migrations/` (source of truth ตาม
  `standard-libraries.md`)

---

## Decisions ที่พินไว้แล้ว (ลดคำถามค้าง)

1. **Calc engine ต้องคำนวณด้วยจำนวนเต็ม "สตางค์" (int64), ห้ามคูณ/หาร float64 ตรง ๆ แล้วปัด**
   เพราะ binary float มี representation error ที่ทำให้ tie-case (`x.xx5`) ปัดผิดฝั่งได้
   (เช่น `10.50 * 0.07 = 0.735` แต่ float64 เก็บเป็น `0.7349999...` ทำให้ `math.Round` ปัดลงผิด)
   วิธี: `priceCents := round(unitPrice*100)`; เพราะ **qty เป็นจำนวนเต็ม (int, ≥ 1)** — ตามภาพ
   reference ตาราง Item/Qty เป็นจำนวนนับ ไม่ใช่ทศนิยม — `lineTotalCents := priceCents * qty`
   (คูณ int กับ int แม่นยำ 100%, ไม่มีเศษต้องปัด); `subtotalCents := Σ lineTotalCents`;
   `baseCents := subtotalCents - discountCents`; **VAT ปัดครึ่งขึ้นด้วยตัวเลขล้วน**:
   `vatCents := (baseCents*7 + 50) / 100` (integer division ปัดครึ่งขึ้นเสมอเพราะค่าเป็นบวกเสมอ);
   `totalCents := baseCents + vatCents`. แปลงกลับเป็น `float64(cents)/100` ตอน map เป็น DTO ตอบ
   response เท่านั้น (ห้ามคำนวณต่อจากค่า float ที่แปลงแล้ว)
2. **qty เป็นจำนวนเต็มบวก (`int`, `binding:"required,gte=1"`)** ไม่รองรับ qty ทศนิยมในเฟสนี้
   (ลดความซับซ้อนของ floating-point ที่จุดคูณ; ถ้าต้องการ qty ทศนิยมในอนาคตต้องออกแบบ calc
   engine ใหม่เป็นทศนิยมคงที่หลายตำแหน่ง)
3. **reference_no**: รูปแบบ `QT` + `YY` + `MM` + running number 3 หลัก reset รายเดือน (เช่น
   `QT2607001`). สร้างที่ **service layer** ผ่าน `Clock` ที่ inject ได้ (`func() time.Time`,
   default `time.Now`) ตาม `testing.md` (พึ่งเวลา → ต้อง mock ได้). implementation: หา
   `MAX(reference_no)` ที่ prefix `QT{YYMM}` ด้วย query แบบ parameterized เพื่อคำนวณเลขถัดไป
   แล้ว insert ใน transaction เดียวกับ header+items; ถ้าเจอ unique-constraint conflict (แข่งกัน
   สร้างพร้อมกัน) ให้ **retry สูงสุด 5 ครั้ง** ด้วยเลขถัดไป (trade-off ที่ยอมรับสำหรับ MVP โหลดต่ำ
   — ดูหัวข้อความเสี่ยงสำหรับแนวทางขยับไป advisory lock ถ้า concurrency สูงขึ้น)
4. **สถานะ (`status`)**: ค่าที่รองรับ `draft` (default), `sent`, `approved`, `rejected` (เก็บไว้ใน
   DB check constraint เผื่ออนาคต) — **สโคปนี้ไม่มี endpoint เปลี่ยนสถานะ** (ฟีเจอร์ approval/
   workflow เป็นสโคปถัดไป, ดู `docs/plans/` ที่จะตามมา). การทดสอบ "แก้ non-draft ไม่ได้" ให้
   test-case-writer seed แถวสถานะ ≠ `draft` ผ่าน repository/DB ตรง ๆ ใน test setup (ไม่ใช่ผ่าน
   public API เพราะยังไม่มี)
5. **สิทธิ์ (RBAC + ownership)**:
   - `POST /quotations` (create), `PUT /quotations/:id`, `DELETE /quotations/:id` →
     `RequireRole("admin","creator")`
   - `GET /quotations`, `GET /quotations/:id` → แค่ `middleware.Auth` (ทุก role ที่ login แล้ว
     อ่านได้ รวม `approver`)
   - **ownership ระดับข้อมูล** (ทำใน service ตาม `auth.md` ข้อ 5): role `creator` แก้/ลบได้เฉพาะ
     quotation ที่ `created_by == userID` (จาก token) เท่านั้น — ไม่ตรง → `ErrForbidden` (403);
     role `admin` ข้าม check นี้ (แก้/ลบได้ทุกใบ)
   - `created_by` ตั้งจาก `userID` ใน context (token) เสมอ **ห้าม**รับจาก client
6. **Draft-only edit/delete violation → sentinel `service.ErrForbidden` (403 `FORBIDDEN`)**
   ตามที่โจทย์ระบุ "403/ตาม api-response.md" (ใช้ sentinel ที่มีอยู่แล้ว ไม่ต้องเพิ่มใหม่)
7. **Sign-off fields**:
   - ฝั่งบริษัท: `company_signee_name`, `company_signee_position` **ดึงจากโปรไฟล์ผู้สร้าง**
     (`User.FullName`/`User.Position`) ตอน **สร้าง** quotation เท่านั้น (snapshot ตอนสร้าง ไม่
     sync ย้อนหลังถ้าโปรไฟล์ user เปลี่ยนทีหลัง — ใบเสนอราคาเป็นเอกสารที่ต้อง fix ค่า ณ เวลาออก
     เอกสาร) — **ไม่มีใน request DTO**, ไม่ให้แก้ผ่าน update ได้; `company_signee_date` = ค่า
     เดียวกับ `date` ของ quotation (ไม่มีคอลัมน์แยก ลดความซ้ำซ้อน)
   - ฝั่งลูกค้า: `customer_signee_name`, `customer_signee_position`, `customer_signee_date`
     เป็น **optional field ใน request** (กรอกได้ตอน create/update ตราบใดที่ยัง draft, ค่าเริ่มต้น
     ว่าง — ผู้ใช้กรอกเองภายหลังเมื่อได้รับการเซ็นจริงจากลูกค้า)
8. **discount_amount**: ระดับบิล (ไม่ใช่ % ต่อ item), validate ที่ **service layer**
   (cross-field, ทำ binding tag ไม่ได้): `0 ≤ discount_amount ≤ subtotal` มิฉะนั้น
   `service.ErrValidation` (400 `VALIDATION_ERROR`) — คำนวณ `subtotal` จาก items ก่อนเช็คเสมอ
9. **valid_until ≥ date**: สมมติฐานทางธุรกิจที่เพิ่มเพื่อคุณภาพ (ป้องกันใบเสนอราคาหมดอายุก่อนวันที่
   ออก) — validate ที่ service, ไม่ผ่าน → 400 `VALIDATION_ERROR` (ถ้าทีมไม่ต้องการ constraint นี้
   สามารถถอดออกได้ง่าย เพราะแยกเป็น validation function เดี่ยว)
10. **List-query whitelist** (ตาม `list-query.md`):
    - `sort` ที่อนุญาต: `created_at`, `date`, `total`, `reference_no` (default `-created_at`)
    - filter เท่ากับ: `status` (`oneof=draft sent approved rejected`), `created_by` (uint)
    - filter ช่วง: `date_gte`, `date_lte` (รูปแบบ `YYYY-MM-DD`)
    - `q`: ค้นหาใน `reference_no`, `company`, `attention`, `project` (ILIKE พร้อม parameterized
      query — ห้าม string-concat)
    - field/param นอก whitelist → 400 `VALIDATION_ERROR` (ไม่เงียบ); `page_size` เกิน 100 →
      400 `VALIDATION_ERROR` (เลือกปฏิเสธ ไม่ cap เงียบ ๆ เพื่อ deterministic)
11. **DELETE** สำเร็จ → 204 ไม่มี body (ลบ quotation cascade ลบ items ด้วย FK `ON DELETE CASCADE`)
12. **ไม่ต้องเพิ่ม env var ใหม่** — reference-no generation และ calc engine ทำงานในโค้ด/DB ล้วน
    ไม่ต้องพึ่ง config เพิ่ม (ตรวจตาม `config.md` ว่ายัง sync ไม่มีอะไรตก)

---

## Tasks (เรียงตาม dependency)

### Backend

1. **[BE] Model** (`internal/model/quotation.go`) — `Quotation` struct: `ID uint`,
   `ReferenceNo string` (`gorm:"uniqueIndex;column:reference_no"`), `Attention`, `Company`,
   `Project`, `Telephone`, `Email string`, `Date time.Time`, `ValidUntil time.Time`,
   `Status string` (`gorm:"type:varchar(20);default:draft"`), `DiscountAmount float64`,
   `Subtotal float64`, `VatAmount float64`, `Total float64`, `CustomerSigneeName *string`,
   `CustomerSigneePosition *string`, `CustomerSigneeDate *time.Time`, `CompanySigneeName string`,
   `CompanySigneePosition string`, `CreatedBy uint` (`gorm:"column:created_by;not null"`),
   `Items []QuotationItem` (`gorm:"foreignKey:QuotationID;constraint:OnDelete:CASCADE"`),
   `CreatedAt`, `UpdatedAt time.Time`. `QuotationItem` struct: `ID uint`,
   `QuotationID uint` (`gorm:"not null"`), `ServiceType`, `Description string`,
   `UnitPrice float64`, `Qty int`, `LineTotal float64`, `SortOrder int`.
2. **[BE] Migration** — `migrations/000X_create_quotations.{up,down}.sql` (ตาราง + unique index
   `reference_no` + check `status in ('draft','sent','approved','rejected')` + FK `created_by`
   → `users(id)`), `000X_create_quotation_items.{up,down}.sql` (FK `quotation_id` →
   `quotations(id)` `ON DELETE CASCADE`). ลงทะเบียน `AutoMigrate` ใน `main.go` (dev)
3. **[BE] DTO** (`internal/dto/quotation_dto.go`) — `QuotationItemInput{ServiceType string
binding:"required"`, `Description string binding:"required"`, `UnitPrice float64
binding:"required,gte=0"`, `Qty int binding:"required,gte=1"`, `SortOrder int}`;
   `CreateQuotationRequest{Attention, Company string binding:"required"`, `Project, Telephone
string`, `Email string binding:"required,email"`, `Date, ValidUntil string
binding:"required"` (parse `YYYY-MM-DD`), `DiscountAmount float64 binding:"gte=0"`,
   `CustomerSigneeName, CustomerSigneePosition, CustomerSigneeDate *string`,
   `Items []QuotationItemInput binding:"required,min=1,dive"}`; `UpdateQuotationRequest`
   เหมือนกัน (PUT แบบ full replace); `QuotationResponse`/`QuotationItemResponse` (json
   snake_case ครบทุก field รวม `subtotal`,`discount_amount`,`vat_amount`,`total`); `ListQuotationQuery`
   ตาม pattern `.claude/docs/list-query.md` §5 (`Page`,`PageSize`,`Sort`,`Status`,`CreatedBy`,
   `DateGte`,`DateLte`,`Q`)
4. **[BE] Calc engine** (`internal/service/quotation_calc.go`) — pure function ไม่แตะ DB/HTTP:
   `roundHalfUpCents(cents int64) ...` (ตัวช่วยแปลงหน่วย), `calcLineTotalCents(unitPrice
float64, qty int) int64`, `calcTotals(lineItemCents []int64, discountAmount float64)
(subtotalCents, baseCents, vatCents, totalCents int64, err error)` — คืน `service.ErrValidation`
   ถ้า `discountCents > subtotalCents` หรือ `< 0`. implement ตาม Decision #1 เป๊ะ (ห้ามใช้
   `math.Round(x*100)/100` กับค่าที่ผ่านการคูณ float แล้ว)
5. **[BE] Repository** (`internal/repository/quotation_repository.go`) — interface
   `QuotationRepository{Create, FindByID, Update, Delete, List, NextReferenceNo}` (ทุก method
   รับ `context.Context`); `Create`/`Update` ทำใน **GORM transaction** (`db.Transaction(func(tx
*gorm.DB) error {...})`) — `Update` ลบ items เดิมของ quotation แล้ว insert ใหม่ทั้งชุด (full
   replace ตาม Decision); `List` ประกอบด้วย GORM scopes (pagination/sort/filter) query แบบ
   parameterized ตาม `list-query.md` §5; `NextReferenceNo(ctx, prefix string) (string, error)`
   หา `MAX(reference_no)` ที่ขึ้นต้นด้วย prefix แล้วคำนวณเลขถัดไป
6. **[BE] Service** (`internal/service/quotation_service.go`) — `NewQuotationService(repo,
userRepo, clock func() time.Time)`; `CreateQuotation(ctx, userID uint, req)`: parse
   date/valid_until (ผิด format → `ErrValidation`), validate `ValidUntil >= Date` (Decision #9),
   คำนวณ items ผ่าน calc engine, validate discount ผ่าน calc engine, gen reference_no (retry
   สูงสุด 5 ครั้งเมื่อชนกัน unique — Decision #3), ดึงโปรไฟล์ผู้สร้างจาก `userRepo.FindByID` มา
   snapshot เป็น `company_signee_*` (Decision #7), เซ็ต `CreatedBy=userID`, `Status="draft"`,
   บันทึกผ่าน repo; `UpdateQuotation(ctx, userID uint, role string, id uint, req)`: โหลด quotation
   เดิม → ถ้า `Status != "draft"` → `ErrForbidden`; ถ้า `role != "admin" && quotation.CreatedBy !=
userID` → `ErrForbidden`; คำนวณใหม่ทั้งหมดเหมือน create แล้ว replace; `DeleteQuotation(ctx,
userID, role, id)`: เช็ค draft-only + ownership เหมือนกัน แล้วเรียก repo.Delete;
   `ListQuotations(ctx, query)`, `GetQuotation(ctx, id)` (ไม่บังคับ ownership สำหรับอ่าน)
7. **[BE] Handler** (`internal/handler/quotation_handler.go`) — bind DTO + เรียก service +
   ตอบผ่าน `pkg/response/` (`Success`/`List`/`Fail`) ทุก endpoint ตาม `api-response.md`
   (`Create`→201, `List`→200 list envelope, `Get`→200, `Update`→200, `Delete`→204)
8. **[BE] Router wiring** (`internal/router/router.go`) — wire repo/service/handler; route
   group: `quotations := engine.Group("/quotations", middleware.Auth(tokenSvc))`;
   `quotations.GET("", h.List)`, `quotations.GET("/:id", h.Get)`; write group แยก
   `middleware.RequireRole("admin","creator")` ครอบ `POST ""`, `PUT "/:id"`, `DELETE "/:id"`
9. **[BE] Response error mapping** (`pkg/response/response.go`) — ตรวจว่า `mapError` ที่มีอยู่
   ครอบ `ErrValidation`/`ErrForbidden`/`ErrNotFound` พอแล้ว (มีอยู่แล้วจาก user-auth) — **ไม่ต้อง
   เพิ่ม sentinel ใหม่** สำหรับ feature นี้ (ดู Decision #6)

### Frontend

10. **[FE] เพิ่ม dependency** — `npm install @tanstack/react-table react-select react-datepicker
dayjs` (ตาม `standard-libraries.md`/`frontend.md`, ยังไม่มีใน `package.json`)
11. **[FE] constants/apiEndpoints.js** — `QUOTATIONS = '/quotations'`
12. **[FE] quotationService.js** — `list(params)`, `getById(id)`, `create(payload)`,
    `update(id, payload)`, `remove(id)` (ยิงผ่าน `apiClient` ที่มี interceptor อยู่แล้ว)
13. **[FE] calcQuotation.js** (util, ไม่ผูก React) — mirror สูตร backend เป๊ะ (Decision #1–2):
    ทำงานด้วยหน่วยสตางค์ (`Math.round(value * 100)`) ไม่ใช่ `toFixed` ตรง ๆ กับผลคูณ float;
    export `calcLineTotal(unitPrice, qty)`, `calcTotals(items, discountAmount)` คืน
    `{ subtotal, discountAmount, vatAmount, total, error }` (error เมื่อ discount ไม่ผ่าน)
14. **[FE] hooks** — `useQuotations` (query, key รวม params ตาม `list-query.md` §6),
    `useQuotation(id)`, `useCreateQuotation`, `useUpdateQuotation` (invalidate list+detail),
    `useDeleteQuotation` (invalidate list); ทั้งหมด `retry:false`
15. **[FE] QuotationItemsTable.jsx** — TanStack Table, แถวแก้ไข inline (service_type/description/
    unit_price/qty), ปุ่มเพิ่ม/ลบแถว, คำนวณ `line_total` ต่อแถวด้วย `calcLineTotal` สด
16. **[FE] QuotationSummary.jsx** — แสดง Sub Total/Discount/VAT 7%/Total จาก `calcTotals`
    อัปเดตทุกครั้งที่ items/discount เปลี่ยน (controlled จาก parent form state)
17. **[FE] QuotationHeaderFields.jsx** — attention/company/project/telephone/email +
    `react-datepicker` สำหรับ `date`/`valid_until`
18. **[FE] SignOffSection.jsx** — ฝั่งลูกค้า (input ชื่อ/ตำแหน่ง/วันที่ — optional) + ฝั่งบริษัท
    (read-only, แสดงจาก `company_signee_name`/`position`/quotation `date` — ไม่มี input ให้แก้)
19. **[FE] QuotationPageHeader.jsx** — หัวกระดาษโลโก้ i-MAXX ตาม `VISUAL_DESIGN_GUIDE.md`
    (สี `--primary #0061A8`, heading slate `#374557`, การ์ด radius 8px เงานุ่มหลายชั้น)
20. **[FE] QuotationFormPage.jsx** — `react-hook-form` + `zod` schema ที่ mirror validation
    backend (Decision #8, #9); disable ฟิลด์/ซ่อนปุ่ม save เมื่อ `status !== 'draft'`
    (Decision — UI ต้อง sync กับ AC13); submit → `useCreateQuotation`/`useUpdateQuotation`;
    error จาก API เทียบด้วย `error.code` (`VALIDATION_ERROR`/`FORBIDDEN`) แสดง toast
21. **[FE] QuotationListPage.jsx** — TanStack Table + pagination/sort ส่ง param ตาม
    `list-query.md`, react-query key รวม param, filter `status` ด้วย `react-select`
22. **[FE] QuotationDetailPage.jsx** — แสดงเอกสารเต็ม (header/items/summary/sign-off) จัด
    เลย์เอาต์ตามภาพ reference (สองฝั่ง sign-off ด้านล่าง)
23. **[FE] routes/AppRoutes.jsx** — เพิ่ม route ทั้ง 4 เส้นทาง (Decision #5: create/edit ครอบ
    `RequireRole(['admin','creator'])`; list/detail ครอบแค่ `RequireAuth`)

### Test (สำหรับ test-case-writer — ระบุที่วางไฟล์)

24. Backend unit: `internal/service/quotation_calc_test.go` (TC calc engine ล้วน — ใช้ตัวเลขจาก
    AC1/AC2 ตรง ๆ), `internal/service/quotation_service_test.go` (mock `QuotationRepository`
    - `UserRepository` ด้วย `testify/mock`, mock `clock` คืนเวลาคงที่เพื่อ assert reference_no)
25. Backend integration: `internal/repository/quotation_repository_test.go` (DB จริงตาม
    `testing.md` §3 — ยืนยัน unique constraint, cascade delete, transaction replace items)
26. Backend handler: `internal/handler/quotation_handler_test.go` (RBAC + response envelope +
    list-query whitelist reject)
27. Frontend: `features/quotation/utils/calcQuotation.test.js`,
    `features/quotation/pages/QuotationFormPage.test.jsx` (MSW),
    `features/quotation/pages/QuotationListPage.test.jsx` (MSW, param ใน query key)

---

## Acceptance Criteria (ทดสอบได้ — ตัวเลขจริงสำหรับ assertion)

**Calc engine**

- **AC1 (happy path 2 รายการ + discount)**: items = `[{unit_price:1000.00, qty:2},
{unit_price:250.50, qty:3}]`, `discount_amount:151.50` →
  `line_total` = `2000.00`, `751.50` ตามลำดับ; `subtotal = 2751.50`; `base (subtotal-discount)
= 2600.00`; `vat_amount = round(2600.00*0.07,2) = 182.00`; `total = 2782.00` (ทศนิยม 2
  ตำแหน่งเป๊ะทุกค่า)
- **AC2 (rounding tie — ต้องปัดขึ้น ไม่ใช่ float bug ปัดลง)**: `base = 10.50` (เช่น
  `subtotal=10.50, discount_amount=0`) → `vat_amount = 0.74` (จาก `10.50*0.07=0.735` ซึ่งเป็น
  tie ที่ตำแหน่งที่ 3 ต้องปัดขึ้นเป็น `0.74`, **ไม่ใช่** `0.73`); `total = 11.24`. ทดสอบทั้งที่
  unit function ของ calc engine โดยตรง และผ่าน full create flow
- **AC3 (discount เกิน subtotal → error)**: `subtotal=2751.50, discount_amount=3000.00` →
  `POST /quotations` และ `PUT /quotations/:id` ตอบ 400 `error.code === "VALIDATION_ERROR"`;
  ไม่มี record ถูกสร้าง/แก้ไขใน DB
- **AC4 (discount == subtotal พอดี — boundary OK)**: `discount_amount == subtotal` เป๊ะ →
  ไม่ error, `base=0.00`, `vat_amount=0.00`, `total=0.00`

**Reference number**

- **AC5**: สร้าง quotation สำเร็จ (วันที่ระบบอยู่ในเดือน `2026-07`) → `reference_no` ตรง regex
  `^QT2607\d{3}$`; สร้างอีกใบถัดไปในเดือนเดียวกัน (DB/test เดียวกัน) → เลขวิ่งต่อ (เช่น
  `QT2607001` แล้ว `QT2607002`) ไม่ซ้ำกัน (unique constraint ไม่ violate)

**CRUD + Draft restriction**

- **AC6**: `PUT /quotations/:id` และ `DELETE /quotations/:id` บน quotation ที่ `status="draft"`
  → 200/204 สำเร็จ และค่าที่แก้ถูกบันทึก/ถูกลบจริงใน DB
- **AC7**: `PUT`/`DELETE` บน quotation ที่ `status != "draft"` (seed ตรงผ่าน repository ใน test
  setup) → 403 `error.code === "FORBIDDEN"`; DB ไม่ถูกแก้ไข/ไม่ถูกลบ
- **AC8 (ownership)**: user role `creator` (userA) พยายาม `PUT`/`DELETE` quotation ของ `creator`
  อีกคน (userB, `created_by=userB.id`) → 403 `FORBIDDEN`; user role `admin` ทำ `PUT`/`DELETE`
  quotation ของใครก็ได้ → สำเร็จ

**RBAC**

- **AC9**: role `approver` เรียก `POST /quotations` → 403 `FORBIDDEN`; role `approver` เรียก
  `GET /quotations` และ `GET /quotations/:id` → 200 สำเร็จ; ไม่มี token (ไม่ login) เรียก
  endpoint ใดใน `/quotations/*` → 401 `UNAUTHORIZED`

**List query**

- **AC10**: `GET /quotations?page=1&page_size=20&sort=-created_at&status=draft` → 200,
  envelope `{ data: [...], meta: { page, page_size, total } }` ตาม `list-query.md`;
  `meta.total` = จำนวนแถวหลังกรอง `status=draft` เท่านั้น
- **AC11**: `GET /quotations?sort=unit_price` (field นอก whitelist) หรือ
  `GET /quotations?foo=bar` (filter นอก whitelist) → 400 `VALIDATION_ERROR`;
  `GET /quotations?page_size=1000` → 400 `VALIDATION_ERROR`

**Response shape / Sign-off**

- **AC12**: `GET /quotations/:id` → `data` มีครบ: `reference_no`, `status`, `subtotal`,
  `discount_amount`, `vat_amount`, `total`, `items[]` (`service_type`, `description`,
  `unit_price`, `qty`, `line_total`, `sort_order`), `customer_signee_name/position/date`,
  `company_signee_name/position`, `created_by`
- **AC13**: หลัง create, `company_signee_name`/`company_signee_position` ตรงกับ
  `full_name`/`position` ของ user ผู้สร้าง ณ เวลานั้น (snapshot — แก้โปรไฟล์ user ภายหลังไม่
  กระทบ quotation เดิม)

**Frontend**

- **AC14**: กรอกฟอร์มด้วยชุดข้อมูลเดียวกับ AC1 (2 รายการ + discount 151.50) → หน้าจอแสดง
  Sub Total `2,751.50` / Discount `151.50` / VAT `182.00` / Total `2,782.00` แบบ real-time
  (ไม่ต้องเรียก backend) จาก `calcQuotation.js`
- **AC15**: กรอก discount มากกว่า subtotal ในฟอร์ม → ขึ้น validation error ใต้ช่อง discount และ
  ปุ่ม submit ถูก disable (ก่อนยิง API)
- **AC16**: เปิดฟอร์มแก้ไข quotation ที่ `status !== 'draft'` → ฟิลด์ทั้งหมด disabled/read-only,
  ไม่มีปุ่มบันทึก/ลบ
- **AC17**: submit สำเร็จ (mock 201/200 ด้วย MSW) → react-query invalidate list เดิม, list
  แสดงข้อมูลใหม่/อัปเดต

**Security / Logging**

- **AC18**: ไม่มี log บรรทัดใดพิมพ์ `email`/`telephone`/`customer_signee_name` เต็มค่าที่ระดับ
  `info` ขึ้นไป (log เฉพาะ id/status/ผลลัพธ์สั้น ๆ ตาม `error-logging.md`)
- **AC19**: error 500 ใด ๆ (ถ้าเกิด) ไม่ leak stack/SQL/path — ตอบ `INTERNAL_ERROR` ข้อความกลาง
  ผ่าน `pkg/response/` เท่านั้น

---

## ความเสี่ยง / คำถามค้าง

1. **reference_no concurrency**: วิธี retry-on-conflict (สูงสุด 5 ครั้ง) เพียงพอสำหรับ MVP
   โหลดต่ำ แต่ถ้ามีผู้ใช้สร้างพร้อมกันจำนวนมากในเดือนเดียวกัน อาจต้องขยับไปใช้ Postgres advisory
   lock (`pg_advisory_xact_lock(hashtext(prefix))`) เพื่อ serialize การออกเลขจริง — ทำเมื่อพบ
   ปัญหาจริงจาก QA/production ไม่ทำ preemptive ในรอบนี้
2. **status workflow (sent/approved/rejected)**: สโคปนี้เก็บ column ไว้เผื่ออนาคตแต่ไม่มี
   endpoint เปลี่ยนสถานะ — ฟีเจอร์ approval workflow ต้องเป็น slug ถัดไป (เช่น
   `quotation-approval`) เมื่อมีโจทย์ชัดเจนกว่านี้ (ใครอนุมัติได้, ต้องแจ้งเตือนไหม ฯลฯ)
3. **valid_until ≥ date**: เป็นสมมติฐานที่ planner เพิ่มเพื่อคุณภาพข้อมูล ไม่ได้ระบุในโจทย์ตรง ๆ
   — ถ้าทีมไม่ต้องการ constraint นี้ให้แจ้ง dev ถอด validation function นี้ออกได้ทันที (แยกเป็น
   ฟังก์ชันเดี่ยวไม่ปนกับ calc engine)
4. **การพิมพ์เอกสาร (print/PDF)**: โจทย์อ้างอิงภาพ reference สำหรับ layout หน้าจอ (Detail page)
   แต่ไม่ได้ขอฟีเจอร์ export PDF/print stylesheet ชัดเจน — แผนนี้ทำแค่หน้าจอที่เลย์เอาต์ตรงกับภาพ
   ไม่ได้ทำ print-to-PDF (ถ้าต้องการ ต้องเป็นสโคปเพิ่ม/slug ใหม่)
5. **service_type เป็น free-text หรือ dropdown ค่าคงที่**: โจทย์ไม่ได้กำหนดรายการ service type
   ตายตัว — แผนนี้ปฏิบัติเป็น free-text input (validate แค่ required) ถ้าจริง ๆ ต้องการเป็น
   dropdown ปิด (fixed list) ต้องแจ้งรายการค่าที่อนุญาตก่อน dev เริ่ม
