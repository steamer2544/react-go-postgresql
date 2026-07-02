---
paths:
  - "backend/**"
---

# Backend Rules (Go + Gin)

> กฎนี้เป็น **path-scoped**: โหลดเข้า context เฉพาะตอน Claude แตะไฟล์ใต้ `backend/`
> (ประหยัด context ตอนทำงานฝั่ง frontend) — `naming-conventions.md` ไม่ scope จึงโหลดทุก session

กฎเฉพาะฝั่ง Go ใช้ร่วมกับ `naming-conventions.md`

## สถาปัตยกรรม (layered)
แยกความรับผิดชอบเป็นชั้น ไหลทางเดียว:

```
handler → service → repository → model (DB)
```

- **handler** — รับ request, validate, แปลง DTO, เรียก service (ไม่มี business logic)
- **service** — business logic ทั้งหมด
- **repository** — คุยกับ DB ผ่าน GORM เท่านั้น
- **model** — struct ที่ map กับตาราง (GORM)
- **dto** — struct สำหรับ request/response แยกจาก model

## หลักการเขียนโค้ด
- ทุกฟังก์ชันที่คืน error ต้องเช็ค error เสมอ ห้ามละเลย
- ใช้ `context.Context` ส่งผ่านทุกชั้นที่ติดต่อ DB/external
- config อ่านจาก `.env` ผ่าน viper/godotenv ห้าม hardcode
- validate input ด้วย tag ของ `go-playground/validator` (มากับ Gin binding)
- struct ที่ bind JSON: field เป็น PascalCase + `json:"snake_case"` + `binding:"required"`

## Response มาตรฐาน (บังคับ)
- ทุก endpoint ตอบตามสัญญาใน `.claude/docs/api-response.md` (envelope `data`/`error` + HTTP status)
- ใช้ helper กลางจาก `pkg/response/` เท่านั้น อย่าปั้น JSON เองแต่ละที่
- error ที่ส่ง client ห้าม leak stack/SQL/path

## Security (บังคับ — checklist เต็มใน `.claude/docs/security.md`)
- รหัสผ่าน hash ด้วย `bcrypt` เท่านั้น ห้ามเก็บ plaintext
- query ผ่าน GORM แบบ parameterized (`Where("x = ?", v)`) ห้ามต่อ SQL string จาก user
- ห้าม log ข้อมูลอ่อนไหว (password, token, PII); CORS ระบุ origin จริง ไม่เปิด `*` ใน prod

## Migration
- dev = `GORM AutoMigrate` ได้; staging/prod = `golang-migrate` เท่านั้น (ไฟล์ `migrations/` = source of truth)

## ไลบรารีที่กำหนด
- Framework → **Gin** (`github.com/gin-gonic/gin`)
- ORM → **GORM** (`gorm.io/gorm`) + driver `gorm.io/driver/postgres`
- รายการเต็ม → `.claude/docs/standard-libraries.md`

## โครงสร้างไฟล์
ดู `.claude/docs/backend-structure.md`

## ตรวจก่อน commit
```bash
cd backend && gofmt -l . && go vet ./... && go build ./...
```
> ใช้ `gofmt`/`goimports` จัดฟอร์แมตอัตโนมัติ ไม่ต้องจัด indent เอง
