# Prompt Pack — ระบบ Quotation + Approval (i-MAXX)

> ร่างจากผลการ scrutinize: **อย่ารัน `/feature` ก้อนเดียว** — repo ยังว่าง (ไม่มี `frontend/`, `backend/`)
> และ scope ใหญ่เกิน 1 slug ให้ทำตามลำดับด้านล่าง ทีละขั้น commit เป็น baseline ก่อนไปขั้นถัดไป
>
> **การตัดสินใจ business logic ที่พินแล้ว:**
> - **ลายเซ็น** = pre-stored profile: อัปโหลดรูปลายเซ็น + ชื่อ + ตำแหน่ง ไว้ที่โปรไฟล์ user ล่วงหน้า ตอน approve ระบบดึงมาสแตมป์อัตโนมัติ
> - **Calc** = Discount ระดับบิล, VAT 7% คิด**หลัง**หักส่วนลด: `Base = Subtotal − Discount` → `VAT = round(Base × 0.07, 2)` → `Total = Base + VAT` (ทศนิยม 2 ตำแหน่ง, ปัดครึ่งขึ้น)
> - **Auth** = สร้างเป็น slug แรก (JWT + bcrypt + RBAC ตาม `auth.md`)

---

## ลำดับการทำงาน (dependency order)

```
0. scaffold-baseline   (งานธรรมดา ไม่ใช่ /feature — ใช้ qwen-agent ได้)
        │
1. /feature user-auth              ← ต้องมีก่อน เพราะ approval ต้องใช้ user/role + profile ลายเซ็น
        │
2. /feature quotation-crud         ← header + line items + draft/แก้ไข + calc engine + sign-off UI
        │
3. /feature quotation-payment-terms ← หลายงวด (Term 1..N) + ยอดต่องวด
        │
4. /feature quotation-approval     ← submit → approver → approved + สแตมป์ลายเซ็น + เผื่อ FK invoice
        │
5. (ภายหลัง) /feature invoice       ← 1 quotation → N invoice ตามงวด (ยังไม่ทำรอบนี้)
```

---

## ขั้นที่ 0 — Scaffold baseline (ทำก่อน ไม่ผ่าน `/feature`)

> เหตุผล: `/feature` ถูกออกแบบมา "ต่อยอดของเดิม" — dev agent สมมติว่ามี `services/` axios, `pkg/response/`,
> handler→service→repository→model, DB connection อยู่แล้ว ต้อง bootstrap โครงเหล่านี้ให้ build เปล่าผ่านก่อน

**Prompt:**

```
สร้างโครง scaffold ของโปรเจกต์ให้ตรงกับ .claude/docs/backend-structure.md และ
.claude/docs/frontend-structure.md โดยยังไม่ต้องมี business logic ใด ๆ:

Backend (backend/):
- go mod init, ตั้ง Gin + GORM + PostgreSQL driver ตาม standard-libraries.md
- โครงชั้น handler → service → repository → model (โฟลเดอร์เปล่า/ตัวอย่าง)
- pkg/response/ ตาม .claude/docs/api-response.md (helper ตอบ JSON มาตรฐาน)
- โหลด config จาก .env ตาม .claude/docs/config.md + สร้าง .env.example (ห้าม hardcode)
- structured logging + request id ตาม error-logging.md (middleware กลาง)
- DB connection + health-check endpoint GET /healthz
- go vet ./... && go build ./... ต้องผ่าน

Frontend (frontend/):
- Vite + React + Bootstrap 5 ตาม tech-stack.md
- axios instance กลางใน services/ (base URL จาก .env)
- โครงโฟลเดอร์ตาม frontend-structure.md + หน้า placeholder เดียว
- ใช้โทน/สไตล์ตาม .claude/docs/VISUAL_DESIGN_GUIDE.md
- npm run lint && npm run build ต้องผ่าน

เสร็จแล้ว commit เป็น "chore: scaffold frontend + backend baseline"
```

> งานนี้ mechanical — ส่งให้ **qwen-agent** ได้ (ประหยัด token) แต่ให้ Claude review ผล build ก่อน commit

---

## ขั้นที่ 1 — `/feature user-auth`

**คำสั่ง:** `/feature user-auth`

**Description:**

```
สร้างระบบ authentication + authorization ตาม .claude/docs/auth.md:

Backend:
- User model: email (unique), password_hash (bcrypt), role (enum: admin | creator | approver),
  full_name, position, signature_image_path (nullable)
- POST /auth/login → คืน JWT; middleware ตรวจ JWT + RBAC ตาม role
- GET /me → ข้อมูล user ปัจจุบัน
- โปรไฟล์: PUT /me/profile (แก้ full_name, position) + POST /me/signature
  (อัปโหลดรูปลายเซ็น เก็บไฟล์ + path ลง signature_image_path) — validate ชนิด/ขนาดไฟล์
- json tag snake_case, exported = PascalCase, ตอบผ่าน pkg/response/
- ห้าม log password/secret (security.md)

Frontend:
- หน้า Login + เก็บ token, axios interceptor แนบ Authorization header
- หน้า Profile: แก้ชื่อ/ตำแหน่ง + อัปโหลดรูปลายเซ็น (preview)
- guard route ตาม role

Acceptance:
- login ผิด → 401 ตาม api-response.md; password ถูก hash ด้วย bcrypt เสมอ
- เฉพาะ role ที่มีสิทธิ์เข้าถึง endpoint ที่ป้องกันได้
- อัปโหลดลายเซ็นแล้ว signature_image_path ถูกบันทึกและดึงกลับมาแสดงได้

เริ่มจากวางแผนใน docs/plans/user-auth.md ก่อน
```

---

## ขั้นที่ 2 — `/feature quotation-crud`

**คำสั่ง:** `/feature quotation-crud`

**Description:**

```
สร้าง CRUD ใบเสนอราคา + calc engine ตามมาตรฐาน CLAUDE.md และ VISUAL_DESIGN_GUIDE.md
(อ้างรูป reference: หัวกระดาษ logo i-MAXX, ตาราง Item/Service Type/Description/Unit Price/Qty/Price,
บล็อกสรุป Sub Total/Discount/VAT 7%/Total, ส่วน Sign-off ล่างสองฝั่ง)

ข้อมูล/Schema:
- Quotation: reference_no (เช่น QT2607001), attention, company, project, telephone, email,
  date, valid_until, status (default 'draft'), discount_amount, subtotal, vat_amount, total,
  created_by (FK user)
- QuotationItem: quotation_id (FK), service_type, description, unit_price, qty, line_total, sort_order
- Sign-off: เก็บชื่อ/ตำแหน่ง/วันที่ ทั้งฝั่งลูกค้าและบริษัท (ฝั่งบริษัทดึงจากโปรไฟล์ผู้สร้าง/ผู้อนุมัติ)

Calc engine (พินแล้ว — ทศนิยม 2 ตำแหน่ง, ปัดครึ่งขึ้น):
- line_total = unit_price × qty
- subtotal   = Σ line_total
- base       = subtotal − discount_amount        (discount ระดับบิล, ต้อง 0 ≤ discount ≤ subtotal)
- vat_amount = round(base × 0.07, 2)
- total      = base + vat_amount
- คำนวณฝั่ง backend เป็น source of truth; frontend คำนวณ preview แบบเดียวกัน

CRUD + Draft:
- สร้าง/แก้ไข/ลบ ได้เฉพาะตอน status = 'draft'
- list ตาม .claude/docs/list-query.md (page/sort/filter)

Frontend:
- ฟอร์มสร้าง/แก้ไข: ตารางรายการเพิ่ม/ลบแถวได้ (TanStack Table), react-select, react-datepicker
- แสดง Sub Total/Discount/VAT/Total อัปเดตสด ตาม calc ด้านบน
- layout + โทนสีตาม VISUAL_DESIGN_GUIDE.md + หัวกระดาษ logo i-MAXX

Acceptance:
- ใส่ 2 รายการ + discount → subtotal/vat/total ตรงสูตร (ทศนิยม 2 ตำแหน่ง)
- discount > subtotal → validation error
- แก้ไขได้เฉพาะ draft; non-draft แก้ไม่ได้ (403/ตาม api-response.md)

เริ่มจากวางแผนใน docs/plans/quotation-crud.md ก่อน
```

---

## ขั้นที่ 3 — `/feature quotation-payment-terms`

**คำสั่ง:** `/feature quotation-payment-terms`

**Description:**

```
เพิ่มการแบ่งงวดชำระเงิน (Payment Terms) ให้ใบเสนอราคา:

Schema:
- PaymentTerm: quotation_id (FK), term_no (1,2,3...), description (เช่น "งวดที่ 1 มัดจำ"),
  amount, sort_order
- 1 quotation → N payment terms

กติกา:
- ผลรวม amount ของทุกงวด = total ของใบเสนอราคา (validation; error ถ้าไม่ตรง)
- แก้ไขได้เฉพาะตอน status = 'draft'
- (เผื่ออนาคต) แต่ละงวดคือหน่วยที่จะออก invoice ได้ 1 ใบ — ออกแบบให้ term อ้างอิงได้ภายหลัง

Frontend:
- ในฟอร์ม quotation: ส่วน Payment Term เพิ่ม/ลบงวด, กรอกยอดต่องวด, แสดงผลรวม vs total (เตือนถ้าไม่ตรง)

Acceptance:
- 3 งวดที่ผลรวม = total → บันทึกผ่าน
- ผลรวมงวด ≠ total → validation error
- non-draft แก้งวดไม่ได้

เริ่มจากวางแผนใน docs/plans/quotation-payment-terms.md ก่อน
```

---

## ขั้นที่ 4 — `/feature quotation-approval`

**คำสั่ง:** `/feature quotation-approval`

**Description:**

```
เพิ่ม single-step approval workflow + สแตมป์ลายเซ็นอัตโนมัติ:

State machine:
- draft → (creator กด submit) → pending_approval → (approver) → approved | rejected
- transition ตาม RBAC: submit = creator เจ้าของเอกสาร; approve/reject = role approver เท่านั้น

เมื่อ approved:
- สแตมป์อัตโนมัติจากโปรไฟล์ผู้อนุมัติ (pre-stored): signature_image + full_name + position + approved_at
  ลงในเอกสาร (เก็บ approver_id, approved_at, และ snapshot ชื่อ/ตำแหน่ง/ path ลายเซ็น ณ เวลาอนุมัติ)
- เอกสารที่ approved แล้ว: แก้ไขไม่ได้

Invoice foresight (ยังไม่ build invoice รอบนี้ — แค่เตรียม schema):
- ออกแบบให้ "เฉพาะ quotation ที่ approved เท่านั้น" ถึงจะออก invoice ได้ในอนาคต
- เตรียมความสัมพันธ์ 1 quotation → N invoice (ผูกกับ payment term) — ใส่ได้แค่ FK/field เผื่อไว้
  *ไม่นับเป็น acceptance ที่ต้องผ่าน test รอบนี้*

Frontend:
- ปุ่ม Submit (creator) / Approve-Reject (approver) ตามสถานะ + สิทธิ์
- แสดงสถานะเอกสาร + เมื่อ approved โชว์ลายเซ็น/ชื่อ/ตำแหน่ง/วันที่ในส่วน Approved

Acceptance:
- creator submit → status = pending_approval
- non-approver กด approve → 403
- approver approve → status = approved + ลายเซ็น/ชื่อ/ตำแหน่ง/approved_at ถูกสแตมป์จากโปรไฟล์
- เอกสาร approved แก้ไขไม่ได้

เริ่มจากวางแผนใน docs/plans/quotation-approval.md ก่อน
```

---

## ขั้นที่ 5 — Invoice (ภายหลัง)

ยังไม่ทำรอบนี้ เมื่อ 1–4 ผ่านและ schema เผื่อไว้แล้ว ค่อยเปิด `/feature invoice`
(1 quotation approved → ออก N invoice ตามงวดชำระเงิน)

---

## หมายเหตุประหยัด token

- ขั้น 0 (scaffold) และงาน mechanical (rename, boilerplate, format) → **qwen-agent**
- ทุก slug: ปล่อยให้ pipeline วน dev↔qa เอง แต่ถ้าชน cap 3 รอบ ให้หยุดดูก่อน อย่าฝืนวน
- ทำทีละ slug + commit ก่อนไปตัวถัดไป จะ debug ง่ายและไม่เผา context รวม
