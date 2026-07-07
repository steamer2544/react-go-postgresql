# CLAUDE.md

> ไฟล์นี้ Claude Code จะโหลดเข้า context อัตโนมัติทุกครั้งที่เริ่ม session
> เก็บให้ **สั้น กระชับ ชี้ทาง** ส่วนรายละเอียดยาว ๆ ให้อยู่ใน `.claude/docs/`

---

## 1. โปรเจกต์นี้คืออะไร

ระบบ Web Application แบบ full-stack แยก frontend / backend ชัดเจน

| ชั้น     | เทคโนโลยี                               |
| -------- | --------------------------------------- |
| Frontend | React (Vite) + Bootstrap 5 + Custom CSS |
| Backend  | Go (Golang) + Gin framework             |
| ORM      | GORM                                    |
| Database | PostgreSQL                              |

รายละเอียด tech stack เต็ม → `.claude/docs/tech-stack.md`

---

## 2. โครงสร้างโปรเจกต์ (ระดับบนสุด)

```
project/
├── frontend/          # React app (ดู .claude/docs/frontend-structure.md)
├── backend/           # Go + Gin app (ดู .claude/docs/backend-structure.md)
├── docs/              # artifact ที่ agent สร้างต่อฟีเจอร์
│   ├── plans/         # ผลของ planner  (docs/plans/<slug>.md)
│   ├── tests/         # ผลของ test-case-writer  (docs/tests/<slug>-testcases.md)
│   └── reports/       # ผลของ qa-tester  (docs/reports/<slug>-qa.md)
├── CLAUDE.md          # ไฟล์นี้
└── .claude/
    ├── settings.json  # hooks (auto-format) + permissions (บล็อกอ่าน secret) — commit เข้า git
    ├── hooks/         # สคริปต์ของ hook (guard-secrets.sh, format.sh)
    ├── commands/      # slash command: /feature (orchestrate ทั้งสาย), /plan, /qa
    ├── rules/         # กฎที่โหลดเข้า context (naming = ทุก session; frontend/backend = path-scoped)
    ├── agents/        # subagent: planner → test-case-writer → dev → qa-tester
    └── docs/          # เอกสารมาตรฐาน "คงที่" โหลดเฉพาะตอนต้องใช้
```

> แยกให้ชัด: `.claude/docs/` = มาตรฐานของโปรเจกต์ (คงที่) — `docs/` (ที่ root) = ของที่ agent ผลิตต่อฟีเจอร์ (plan/test)

---

## 3. กฎหลักที่ต้องจำเสมอ (สรุป)

- **Frontend naming**: Component/ไฟล์ Component = `PascalCase`, ตัวแปร/ฟังก์ชัน/state = `camelCase`, state setter = `set` + PascalCase, custom hook = `use` + PascalCase, CSS class = `kebab-case`
- **Backend naming (Go)**: local variable = `camelCase`, identifier ที่ export = `PascalCase` (บังคับโดยภาษา), json tag = `snake_case`, package = lowercase
- ห้าม hardcode ค่า config → ใช้ `.env` ทั้งสองฝั่ง (รายการ var เต็ม → `.claude/docs/config.md`)
- ทุก endpoint ตอบตาม response มาตรฐาน → `.claude/docs/api-response.md`; list ใช้ query มาตรฐาน → `.claude/docs/list-query.md`
- auth/authorization ตามมาตรฐาน (JWT + bcrypt + middleware + RBAC) → `.claude/docs/auth.md`
- error จัดการข้ามชั้น + structured log + request id, ไม่ leak internal → `.claude/docs/error-logging.md`
- ผ่าน security checklist ขั้นต่ำ (hash password, ไม่ต่อ SQL เอง, ไม่ log secret) → `.claude/docs/security.md`
- ใช้เฉพาะ library มาตรฐานสากลที่ทีมกำหนด → `.claude/docs/standard-libraries.md`
- test ตามกลยุทธ์ (unit+mock / integration DB / frontend) → `.claude/docs/testing.md`

กฎแบบละเอียด (`.claude/rules/`):

- `naming-conventions.md` — คอนเวนชันการตั้งชื่อทั้งหมด (โหลด**ทุก session** เพราะต้องใช้ตอนสร้างไฟล์ใหม่)
- `frontend.md` — กฎเฉพาะฝั่ง React (**path-scoped**: โหลดเฉพาะตอนแตะ `frontend/`)
- `backend.md` — กฎเฉพาะฝั่ง Go/Gin (**path-scoped**: โหลดเฉพาะตอนแตะ `backend/`)

> การบังคับแบบ deterministic อยู่ที่ `.claude/settings.json` (hooks + permissions):
> auto `gofmt`/prettier หลังแก้ไฟล์, และบล็อกการอ่าน/แก้ไฟล์ secret (`.env`, `*.key`, `secrets/`)

---

## 4. คำสั่งที่ใช้ตรวจงาน (verify commands)

```bash
# frontend
cd frontend && npm run lint && npm run build && npm test

# backend
cd backend && gofmt -l . && go vet ./... && go build ./... && go test ./...
```

---

## 5. Agent Workflow (`.claude/agents/`)

ทีม subagent ทำงานต่อกันเป็นสายพานแบบ **TDD (Red → Green → Refactor → Verify)** 1 ฟีเจอร์ = 1 `slug`:

```
planner ──> test-case-writer ──> dev ──> qa-tester
  │              │                │          │
plan +        test cases +     โค้ด (min)   รัน test + verify +
acceptance    test ที่ FAIL     ทำให้เขียว   รายงาน PASS/FAIL
docs/plans/   = RED             + refactor   docs/reports/
              docs/tests/       = GREEN      = VERIFY
```

> ถ้า qa-tester รายงาน FAIL → main agent ส่งกลับให้ `dev` แก้ แล้ววน qa-tester ใหม่จนกว่าจะ PASS (สูงสุด 3 รอบ)
> รันทั้งสายอัตโนมัติด้วย **`/feature <slug หรือ คำอธิบาย>`** (resume ข้ามเฟสที่ทำเสร็จแล้วให้เอง) — ดู `.claude/commands/`

| Agent              | หน้าที่                                                                                  | เขียนโค้ด?                 | model   |
| ------------------ | ---------------------------------------------------------------------------------------- | -------------------------- | ------- |
| `planner`          | แตกงาน + acceptance criteria                                                             | ไม่ (เฉพาะ plan)           | sonnet¹ |
| `test-case-writer` | ออกแบบ test case + assertion (RED) — **พิมพ์ไฟล์ test delegate ให้ qwen** แล้ว review    | เฉพาะ test (ออกแบบ+review) | sonnet  |
| `dev`              | implement ให้ test ผ่าน (GREEN) + refactor — **พิมพ์โค้ด delegate ให้ qwen** แล้ว verify | ใช่ (delegate→verify)      | sonnet  |
| `qa-tester`        | รัน test + verify + เขียนรายงาน `docs/reports/<slug>-qa.md`                              | เฉพาะรายงาน                | sonnet  |

> **นโยบายประหยัดโทเคน:** งาน "พิมพ์โค้ด" (test + implementation) delegate ให้ qwen (`claude-9arm`) ซึ่งถูกกว่ามาก
> ส่วน Claude ถือเฉพาะงาน "คิด" ที่ผิดแล้วพัง: ออกแบบ spec/test/assertion, review diff, ตัดสิน PASS/FAIL, แก้บั๊กลึก
> (อ่าน/ตรวจ ถูกกว่าเขียนเอง) — qwen รันแบบ **synchronous เท่านั้น** ห้าม background/watcher (กัน agent "จอด" เผาโทเคน)
>
> ¹ `planner` = sonnet เป็นค่าตั้งต้น; ใช้ **opus** เฉพาะฟีเจอร์ที่ต้องตัดสินใจ architecture ใหม่จริง ๆ (สั่งเป็นราย ๆ)

เรียกใช้ (เลือกทางใดทางหนึ่ง):

- **แนะนำ** — `/feature <slug หรือ คำอธิบาย>` : orchestrate ทั้งสายให้อัตโนมัติ (planner→test→dev→qa วนจน PASS); `/plan <โจทย์>` วางแผนอย่างเดียว; `/qa <slug>` verify อย่างเดียว
- หรือสั่งทีละตัว `"ใช้ planner วางแผนฟีเจอร์ X"` แล้วไล่ต่อ หรือปล่อยให้ Claude auto-delegate ตาม description

---

## 6. เอกสารอ้างอิงเชิงลึก (`.claude/docs/`)

| ไฟล์                    | เนื้อหา                                                     |
| ----------------------- | ----------------------------------------------------------- |
| `tech-stack.md`         | รายละเอียด stack + เวอร์ชันที่ใช้                           |
| `frontend-structure.md` | โครงสร้างโฟลเดอร์ frontend เต็ม                             |
| `backend-structure.md`  | โครงสร้างโฟลเดอร์ backend เต็ม                              |
| `standard-libraries.md` | รายชื่อ library มาตรฐาน + นโยบาย migration                  |
| `api-response.md`       | รูปแบบ JSON response มาตรฐาน (สัญญา FE/BE)                  |
| `list-query.md`         | สัญญา query param ของ endpoint แบบ list (page/sort/filter)  |
| `auth.md`               | มาตรฐาน auth/authorization (JWT, bcrypt, middleware, RBAC)  |
| `error-logging.md`      | การจัดการ error ข้ามชั้น + structured logging + request id  |
| `config.md`             | รายการ env var ทั้งหมด (schema) + กติกา config              |
| `testing.md`            | กลยุทธ์ test: unit/mock, integration DB, frontend, coverage |
| `security.md`           | checklist ความปลอดภัยขั้นต่ำ                                |
