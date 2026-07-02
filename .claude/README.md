# โฟลเดอร์ `.claude/`

โฟลเดอร์นี้เก็บ config และเอกสารมาตรฐานของโปรเจกต์ สำหรับให้ Claude Code (และทีมงาน)
อ่านและทำตาม **commit เข้า git** เพื่อให้ทุกคนในทีมได้มาตรฐานเดียวกัน

## โครงสร้าง

```
.claude/
├── README.md                  # ไฟล์นี้
├── settings.json              # hooks + permissions (commit เข้า git — บังคับแบบ deterministic)
├── hooks/                     # สคริปต์ของ hook
│   ├── guard-secrets.sh       # PreToolUse: บล็อกอ่าน/แก้ .env จริง, *.key, *.pem, secrets/
│   └── format.sh              # PostToolUse: auto gofmt (.go) / prettier (frontend)
├── commands/                  # slash command (orchestrate pipeline ด้วยคำสั่งเดียว)
│   ├── feature.md             # /feature — รันทั้งสาย planner→test→dev→qa วนจน PASS (resume ได้)
│   ├── plan.md                # /plan — planner อย่างเดียว หยุดให้ review
│   └── qa.md                  # /qa — qa-tester อย่างเดียว (verify + เขียนรายงาน)
├── rules/                     # กฎที่โหลดเข้า context
│   ├── naming-conventions.md  # ไม่ scope → โหลดทุก session
│   ├── frontend.md            # path-scoped: frontend/**
│   └── backend.md             # path-scoped: backend/**
├── agents/                    # subagent ทำงานเป็น pipeline
│   ├── planner.md
│   ├── test-case-writer.md
│   ├── dev.md
│   └── qa-tester.md
└── docs/                      # เอกสารมาตรฐาน "คงที่" โหลดเฉพาะเมื่อต้องใช้ (ไม่กิน context ทุก session)
    ├── tech-stack.md
    ├── frontend-structure.md
    ├── backend-structure.md
    ├── standard-libraries.md
    ├── api-response.md        # รูปแบบ JSON response มาตรฐาน (สัญญา FE/BE)
    ├── list-query.md          # สัญญา query param ของ list (page/sort/filter)
    ├── auth.md                # auth/authorization (JWT, bcrypt, middleware, RBAC)
    ├── error-logging.md       # error ข้ามชั้น + structured logging + request id
    ├── config.md              # env var schema + กติกา config
    ├── testing.md             # กลยุทธ์ test: unit/mock, integration DB, frontend
    └── security.md            # checklist ความปลอดภัยขั้นต่ำ
```

> หมายเหตุ: ผลงานที่ agent ผลิตต่อฟีเจอร์ (plan/test/report) ไป **ที่ root `docs/`** ไม่ใช่ `.claude/docs/`
> `docs/plans/<slug>.md` (planner), `docs/tests/<slug>-testcases.md` (test-case-writer),
> `docs/reports/<slug>-qa.md` (qa-tester)

## `rules/` vs `docs/` ต่างกันอย่างไร

| | `rules/` | `docs/` |
| --- | --- | --- |
| การโหลด | เข้า context ตอน session (ดู path-scope ด้านล่าง) | เฉพาะเมื่อถูกอ้างถึง/ต้องใช้ |
| กิน context | ทุกครั้ง (ถ้าไม่ scope) | เฉพาะตอนอ่าน |
| เหมาะกับ | กฎสั้น ๆ ที่ต้องบังคับเสมอ (naming, style) | เอกสารยาว (โครงสร้าง, รายชื่อ lib, สัญญา API, security) |

> หลักการ: อะไรที่ต้อง "บังคับทุก session" ให้อยู่ใน `rules/` (เขียนให้สั้น)
> อะไรที่เป็น "อ้างอิงตอนต้องใช้" ให้อยู่ใน `docs/` เพื่อไม่ให้เปลือง context

### path-scoped rules (ประหยัด context)
rule ที่ใส่ frontmatter `paths:` จะโหลด**เฉพาะตอน Claude แตะไฟล์ที่ match** — ที่นี่ `frontend.md`
(scope `frontend/**`) และ `backend.md` (scope `backend/**`) จึงไม่กิน context ตอนทำงานอีกฝั่ง
ส่วน `naming-conventions.md` **ไม่** scope เพราะต้องใช้ตอน "สร้างไฟล์ใหม่" ด้วย

```yaml
---
paths:
  - "frontend/**"
---
```

> ⚠️ ข้อจำกัดที่ต้องรู้: path-scoped rule โหลด "เมื่ออ่าน/เข้าถึงไฟล์ที่ match" และอาจหลุดหลัง `/compact`
> กฎที่ต้องมีตอน **สร้างไฟล์ใหม่แน่ ๆ** ควรปล่อยไม่ scope (เหมือน naming) หรือย้ายเข้า CLAUDE.md
> ตรวจว่าอะไรโหลดอยู่จริงด้วย `/context` หรือ `/memory`

### การบังคับที่ "แน่นอน" ใช้ hooks/permissions ไม่ใช่ rules
`rules/` และ `CLAUDE.md` เป็น **context ไม่ใช่การบังคับ** (Claude อาจพลาดได้ตอน session ยาว/เจอ
prompt injection) อะไรที่ "ต้องเกิดขึ้นแน่ ๆ" ให้ทำเป็น **hook** หรือ **permission** ใน `settings.json`:
- **auto-format** (`format.sh`) — gofmt/prettier หลังแก้ไฟล์ ไม่ต้องหวังให้ Claude จำไปรัน
- **บล็อก secret** (`guard-secrets.sh` + `permissions.deny`) — กันอ่าน/แก้ `.env` จริง (ยอมให้ `.env.example`)

## `agents/` — ทีม subagent (pipeline)

subagent เป็นไฟล์ markdown + YAML frontmatter (`name`, `description`, `tools`, `model`)
แต่ละตัวรันใน context แยก ทำงานเดียว แล้วส่งผลกลับ main agent สายพานของโปรเจกต์นี้เป็นแบบ **TDD**:

```
planner ──> test-case-writer ──> dev ──> qa-tester
            (RED)                (GREEN)   (VERIFY)
```

1. **planner** — แตกฟีเจอร์เป็น task + acceptance criteria → `docs/plans/<slug>.md`
2. **test-case-writer (RED)** — เอา AC มาทำ test case + เขียน test ที่ **fail จริง** แล้วรันยืนยัน RED → `docs/tests/<slug>-testcases.md` + ไฟล์ test
3. **dev (GREEN → Refactor)** — เขียนโค้ดน้อยที่สุดให้ test ผ่านครบ แล้ว refactor โดย test ยังเขียว
4. **qa-tester (VERIFY)** — รัน test + verify เทียบ test case + **เขียนรายงาน `docs/reports/<slug>-qa.md`** + ตอบ PASS/FAIL
   (ถ้า FAIL → main agent ส่งกลับให้ dev แก้ แล้ววน qa-tester ใหม่จน PASS, **สูงสุด 3 รอบ**)

> แก้ไฟล์ agent บนดิสก์แล้วต้อง **restart session** ถึงจะโหลดใหม่
> (ถ้าสร้างผ่านคำสั่ง `/agents` จะมีผลทันที)

## `commands/` — slash command (orchestrate สายพานด้วยคำสั่งเดียว)

`agents/` เป็น "ทีมงาน" ส่วน `commands/` เป็น "ปุ่มสั่งงาน" ที่ร้อย agent เข้าด้วยกันเป็นขั้นตอนตายตัว
เพื่อไม่ต้องพึ่งการสั่ง agent ด้วยมือทีละตัวหรือจำลำดับเอง แต่ละคำสั่งคือไฟล์ markdown (ชื่อไฟล์ = ชื่อคำสั่ง)

| คำสั่ง | ทำอะไร | หมายเหตุ |
| --- | --- | --- |
| `/feature <slug หรือ คำอธิบาย>` | รันทั้งสาย **planner → test-case-writer → dev → qa-tester** แล้ววน dev↔qa จน PASS (สูงสุด 3 รอบ) | **resume ได้**: ข้ามเฟสที่มี artifact อยู่แล้ว เว้นแต่ใส่ `--fresh` |
| `/plan <คำอธิบาย>` | เรียก planner อย่างเดียว แล้วหยุดให้ human review | พอใจแผนแล้วต่อด้วย `/feature <slug>` |
| `/qa <slug>` | เรียก qa-tester อย่างเดียว (verify + เขียนรายงาน ไม่แก้โค้ด) | ใช้ตอนแก้มือแล้วอยาก re-verify |

- ตั้ง `disable-model-invocation: true` ทุกคำสั่ง → **ผู้ใช้เป็นคนกดสั่งเอง** Claude ไม่ auto-fire คำสั่งที่มี side effect (เขียนโค้ด/ไฟล์) เอง — เข้ากับหลัก "บังคับแบบตั้งใจ" ของโปรเจกต์
- `$ARGUMENTS` = ข้อความท้ายคำสั่งทั้งก้อน (ไม่ parse เป็น positional) — คำสั่งจึงตีความ slug/โจทย์เองอย่างยืดหยุ่น
- แก้ไฟล์คำสั่งมีผลทันที ไม่ต้อง restart (ต่างจาก agent) — พิมพ์ `/` เพื่อดูรายการที่มี

> Claude Code รุ่นใหม่นับ `commands/` (ไฟล์เดี่ยว) กับ `skills/` (โฟลเดอร์ + `SKILL.md`) เป็นกลไกเดียวกัน
> เอกสารทางการเรียกไฟล์เดี่ยวว่า "legacy" และแนะนำ skills แต่ไฟล์คำสั่งเดิม**ยังใช้ได้ปกติ** — ถ้าคำสั่งโตจนยาว ค่อยยกเป็น skill (`.claude/skills/feature/SKILL.md`) ที่ frontmatter เหมือนกัน

## ไฟล์อื่นที่ Claude Code รองรับ (เพิ่มได้ภายหลังตามต้องการ)

- `.claude/settings.local.json` — ตั้งค่าส่วนตัวของแต่ละคน เช่น `claudeMdExcludes` (**gitignore**)
- `.claude/skills/` — workflow/procedure ที่เรียกใช้ซ้ำได้ (body โหลดเฉพาะตอน invoke)
- preference ส่วนตัว → ใช้ `~/.claude/` (user-level) หรือ `@import` ในไฟล์ที่ commit ได้

> หมายเหตุ: `CLAUDE.local.md` ถูก **deprecated** แล้ว (ใช้ได้ไม่ดีข้าม git worktree) —
> แทนที่ด้วย import (`@path/to/file.md`) หรือ user-level `~/.claude/CLAUDE.md` สำหรับของส่วนตัว
