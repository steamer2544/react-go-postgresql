# i-MAXX Quotation System

ระบบใบเสนอราคา (Quotation) + Approval workflow แบบ full-stack แยก frontend / backend ชัดเจน

| ชั้น     | เทคโนโลยี                                                     |
| -------- | ------------------------------------------------------------- |
| Frontend | React 19 (Vite) + Bootstrap 5 + React Query + React Hook Form |
| Backend  | Go 1.25 + Gin + GORM                                          |
| Database | PostgreSQL                                                    |
| Auth     | JWT + bcrypt + RBAC (admin / creator / approver)              |

> เอกสารมาตรฐาน/สถาปัตยกรรมเชิงลึกอยู่ใน `.claude/docs/` · กฎการเขียนโค้ดใน `.claude/rules/` · `CLAUDE.md` = ภาพรวมโปรเจกต์

---

## ฟีเจอร์ (roadmap 1–4 เสร็จแล้ว)

1. **user-auth** — login (JWT), RBAC middleware, โปรไฟล์ + อัปโหลดรูปลายเซ็น
2. **quotation-crud** — CRUD ใบเสนอราคา + line items + calc engine (integer satang กัน float precision), draft-only edit
3. **quotation-payment-terms** — แบ่งงวดชำระเงิน + validate ผลรวมงวด = total
4. **quotation-approval** — state machine `draft → pending_approval → approved | rejected` + สแตมป์ลายเซ็นอัตโนมัติจากโปรไฟล์ผู้อนุมัติ
5. **invoice** — _(ยังไม่ทำ — เลื่อนเป็นรอบถัดไป)_

---

## โครงสร้างโปรเจกต์

```
.
├── backend/            # Go + Gin (handler → service → repository → model)
│   ├── cmd/api/        # entry point (main.go)
│   ├── internal/       # model, dto, repository, service, handler, middleware, router, config
│   ├── pkg/            # response (envelope มาตรฐาน), database, logger
│   ├── migrations/     # golang-migrate (source of truth ของ schema)
│   └── .env.example
├── frontend/           # React (Vite)
│   └── src/            # features/, components/, hooks/, services/, contexts/, routes/
│       └── ...         # services (axios) → hooks (react-query) → pages
├── docs/               # artifact ต่อฟีเจอร์: plans/ · tests/ · reports/
├── CLAUDE.md           # คู่มือโปรเจกต์สำหรับ AI agent
└── FEATURE_PROMPTS.md  # roadmap + prompt แต่ละฟีเจอร์
```

---

## Prerequisites

- **Go** ≥ 1.25
- **Node.js** ≥ 18 + npm
- **PostgreSQL** ≥ 14 (รันอยู่ที่ localhost:5432 หรือปรับใน `.env`)
- **Docker** — เฉพาะตอนรัน repository integration test (testcontainers)

---

## ติดตั้ง & รัน

### 1. ฐานข้อมูล

สร้าง database ให้ตรงกับ `.env` (ค่า default = `app_db`):

```bash
createdb app_db     # หรือ: psql -c "CREATE DATABASE app_db;"
```

Schema ใช้ `migrations/` เป็น source of truth (staging/prod ใช้ `golang-migrate`);
ตอน dev backend รัน `GORM AutoMigrate` ให้อัตโนมัติเมื่อ start

### 2. Backend

```bash
cd backend
cp .env.example .env          # แล้วเติมค่า DB_PASSWORD, JWT_SECRET (ห้าม commit .env)
go mod download
go run ./cmd/api              # ฟังที่ http://localhost:8080
```

ตรวจว่ารันแล้ว: `GET http://localhost:8080/healthz`

### 3. Frontend

```bash
cd frontend
cp .env.example .env          # VITE_API_URL ชี้ไป backend
npm install
npm run dev                   # เปิดที่ http://localhost:5173
```

---

## Environment Variables

**Backend** (`backend/.env`) — ดูรายละเอียดใน `.claude/docs/config.md`

| ตัวแปร                                                                   | คำอธิบาย                                    |
| ------------------------------------------------------------------------ | ------------------------------------------- |
| `APP_ENV`, `APP_PORT`                                                    | environment + พอร์ต (default 8080)          |
| `DB_HOST/PORT/USER/PASSWORD/NAME/SSLMODE`                                | การเชื่อมต่อ PostgreSQL                     |
| `JWT_SECRET`                                                             | คีย์เซ็น JWT (**ต้องตั้ง** ห้ามว่างใน prod) |
| `JWT_ACCESS_TTL`, `JWT_REFRESH_TTL`                                      | อายุ token                                  |
| `CORS_ALLOWED_ORIGINS`                                                   | origin ที่อนุญาต (ไม่เปิด `*` ใน prod)      |
| `SIGNATURE_UPLOAD_DIR`, `SIGNATURE_MAX_BYTES`, `SIGNATURE_ALLOWED_TYPES` | ตั้งค่าอัปโหลดลายเซ็น                       |

**Frontend** (`frontend/.env`): `VITE_API_URL` (base URL ของ backend), `VITE_ENV`

> ⚠️ `.env` และไฟล์ secret ถูกกันไม่ให้ commit — ใช้ `.env.example` เป็นแม่แบบเท่านั้น

---

## API Endpoints (สรุป)

ทุก response ตามสัญญาใน `.claude/docs/api-response.md` (envelope `data` / `error` + meta สำหรับ list)

| Method | Path                                                                     | สิทธิ์                         |
| ------ | ------------------------------------------------------------------------ | ------------------------------ |
| `GET`  | `/healthz`                                                               | public                         |
| `POST` | `/auth/login`                                                            | public                         |
| `GET`  | `/me` · `PUT /me/profile` · `POST /me/signature` · `GET /me/signature`   | authenticated                  |
| `GET`  | `/quotations` · `/quotations/:id` · `/quotations/:id/approval-signature` | authenticated (approver ดูได้) |
| `POST` | `/quotations` · `PUT /quotations/:id` · `DELETE /quotations/:id`         | admin / creator (owner)        |
| `POST` | `/quotations/:id/submit`                                                 | creator (owner) / admin        |
| `POST` | `/quotations/:id/approve` · `/quotations/:id/reject`                     | approver เท่านั้น              |

**Calc engine** (source of truth ฝั่ง backend, ทศนิยม 2 ตำแหน่ง ปัดครึ่งขึ้น, คำนวณด้วย integer satang):
`line_total = unit_price × qty` · `subtotal = Σ line_total` · `base = subtotal − discount` · `vat = round(base × 0.07)` · `total = base + vat`

---

## ทดสอบ & ตรวจงาน (verify commands)

```bash
# backend
cd backend && gofmt -l . && go vet ./... && go build ./... && go test ./...
#   └─ go test ./...  ต้องมี Docker (repository integration ใช้ testcontainers)
#      ถ้าไม่มี Docker ให้ใช้ go test -short ./...  (ข้าม integration)

# frontend
cd frontend && npm run lint && npm run build && npm test
```

**สถานะ test ปัจจุบัน:** backend 136 pass (+ integration ที่ต้อง Docker) · frontend 44 pass

---

## Development Workflow (AI agent, TDD)

โปรเจกต์นี้พัฒนาด้วยทีม subagent เป็นสายพาน **TDD (Red → Green → Refactor → Verify)** 1 ฟีเจอร์ = 1 `slug`:

```
planner → test-case-writer → dev → qa-tester
(plan)     (failing test/RED)  (GREEN)  (verify + report)
```

- รันทั้งสายด้วย `/feature <slug>` — ดู `FEATURE_PROMPTS.md`
- artifact ต่อฟีเจอร์เก็บใน `docs/{plans,tests,reports}/`
- นโยบายประหยัด token: งานพิมพ์โค้ด/test delegate ให้ qwen; Claude ถือ design spec + review diff + ตัดสิน PASS/FAIL

---

## หมายเหตุ / Known items

- **Integration test** (`TC-REPO-*`) ต้องรันในเครื่อง/CI ที่มี Docker (`go test ./...` แบบไม่มี `-short`)
- **UI polish**: ยังไม่ลง VISUAL_DESIGN_GUIDE styling เต็มรูป + แตก component ย่อย (ฟังก์ชันครบ/test เขียว แต่หน้าตายังเรียบ)
- staging/prod: ใช้ `golang-migrate` รัน `migrations/` (อย่าพึ่ง AutoMigrate)
