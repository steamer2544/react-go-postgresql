# Backend Folder Structure (Go + Gin + GORM)

โครงสร้างอิง **Standard Go Project Layout** ผสม layered architecture

```
backend/
├── cmd/
│   └── api/
│       └── main.go             # entry point: โหลด config, ต่อ DB, start Gin
│
├── internal/                   # โค้ดภายในแอป (ห้าม import จากภายนอก module)
│   ├── config/                 # โหลด/parse config จาก .env (config.go)
│   ├── model/                  # GORM model = ตารางใน DB (employee.go)
│   ├── dto/                    # request/response struct (employee_dto.go)
│   ├── repository/             # ชั้นคุยกับ DB ผ่าน GORM (employee_repository.go)
│   ├── service/                # business logic (employee_service.go)
│   ├── handler/                # Gin handler / controller (employee_handler.go)
│   ├── middleware/             # auth, cors, logger, recovery
│   └── router/                 # ลงทะเบียน route ทั้งหมด (router.go)
│
├── pkg/                        # แพ็กเกจที่ reuse ได้ (ให้ภายนอก import ได้)
│   ├── database/               # สร้าง connection GORM (postgres.go)
│   ├── logger/                 # ตั้งค่า logger
│   └── response/               # helper รูปแบบ JSON response มาตรฐาน (ดู docs/api-response.md)
│
├── migrations/                 # ไฟล์ SQL migration (golang-migrate) — source of truth ของ schema บน prod
│
├── .env                        # ตัวแปร environment (DB_HOST, DB_PORT ...)
├── .env.example                # ตัวอย่าง env สำหรับ commit
├── go.mod
├── go.sum
└── Makefile                    # คำสั่งย่อ: run, build, migrate, test
```

## หน้าที่แต่ละชั้น (ไหลทางเดียว)

```
HTTP Request
   │
   ▼
handler   → validate + แปลง DTO   (ไม่มี business logic)
   │
   ▼
service   → business logic ทั้งหมด
   │
   ▼
repository → query/insert/update ผ่าน GORM
   │
   ▼
model / PostgreSQL
```

## หลักการวางไฟล์

- **`internal/`** ใช้เก็บโค้ดหลักของแอป Go ป้องกันไม่ให้ module อื่น import โดยไม่ตั้งใจ
- **`pkg/`** เก็บของที่อยากให้ reuse ได้จริง (db connection, response helper)
- **`pkg/response/`** เป็นที่เดียวที่ประกอบ JSON response — ทุก handler ตอบผ่าน helper นี้ตาม `.claude/docs/api-response.md`
- 1 โดเมน = 1 ไฟล์ต่อชั้น: `employee_handler.go`, `employee_service.go`, `employee_repository.go`
- ไฟล์ตั้งชื่อ **snake_case**, package ตั้งชื่อ **ตัวเล็กคำเดียว** (`handler`, `repository`)
- struct model แยกจาก DTO เสมอ (อย่าเอา model ไป bind JSON ตรง ๆ)

## นโยบาย Migration
- **dev**: ใช้ `GORM AutoMigrate` ตอน start ได้ (สะดวก)
- **staging / prod**: ใช้ `golang-migrate` เท่านั้น รันจากไฟล์ใน `migrations/` (`*.up.sql`/`*.down.sql`)
- ห้ามผสมสองวิธีใน environment เดียวกัน (กัน schema drift) — ดู `standard-libraries.md`

## ตัวอย่าง model + DTO

```go
// internal/model/employee.go
package model

type Employee struct {
    ID        uint   `gorm:"primaryKey"`
    FirstName string `gorm:"column:first_name"`
    Email     string `gorm:"uniqueIndex"`
}

// internal/dto/employee_dto.go
package dto

type CreateEmployeeRequest struct {
    FirstName string `json:"first_name" binding:"required"`
    Email     string `json:"email" binding:"required,email"`
}
```

## ไฟล์ test
- วางคู่ไฟล์ที่ทดสอบ: `employee_service_test.go` อยู่ package เดียวกับ `employee_service.go`
- ใช้ `testify` (assert/require/mock) ตั้งชื่อ test อ้าง TC id เช่น `TestCreateEmployee_TC01_Happy`
