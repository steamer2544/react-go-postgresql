# Tech Stack

รายละเอียดเทคโนโลยีทั้งหมดของโปรเจกต์

## ภาพรวม

```
┌─────────────┐     HTTP/JSON     ┌─────────────┐     GORM      ┌──────────────┐
│   Frontend  │  ───────────────> │   Backend   │  ──────────>  │  PostgreSQL  │
│ React (Vite)│  <───────────────  │  Go + Gin   │  <──────────  │              │
└─────────────┘                    └─────────────┘               └──────────────┘
```

## 1. Frontend — React

| ด้าน | เทคโนโลยี | หมายเหตุ |
| --- | --- | --- |
| Library | React | ใช้ function component + hooks |
| Build tool | Vite | ไฟล์ component เป็น `.jsx` |
| Stylesheet | Bootstrap 5 + Custom CSS | Bootstrap เป็นฐาน เสริมด้วย custom |
| Table | TanStack Table (`@tanstack/react-table`) | headless table, ยืดหยุ่นสูง |
| Font / Icon | Font Awesome | ผ่าน `@fortawesome/react-fontawesome` |
| Dropdown | react-select | dropdown ที่ค้นหา/multi-select ได้ |
| Datepicker | react-datepicker | เลือกวันที่ |
| Testing | Vitest + React Testing Library | unit/component test |

## 2. Backend — Go (Golang)

| ด้าน | เทคโนโลยี | หมายเหตุ |
| --- | --- | --- |
| Framework | Gin | HTTP web framework |
| ORM | GORM | ใช้กับ PostgreSQL driver |
| Config | .env | ผ่าน viper หรือ godotenv |
| Auth | JWT + bcrypt | ออก token / hash รหัสผ่าน |

## 3. Database — PostgreSQL

- ใช้ GORM เป็น ORM หลัก
- Driver: `gorm.io/driver/postgres`
- จัดการ schema ตามนโยบาย migration: **dev = AutoMigrate ได้, staging/prod = golang-migrate เท่านั้น**
  (รายละเอียด → `standard-libraries.md` หัวข้อ "นโยบาย Migration")

## เอกสารมาตรฐานที่เกี่ยวข้อง
- `standard-libraries.md` — รายชื่อ library + นโยบาย migration
- `api-response.md` — รูปแบบ JSON response มาตรฐาน (สัญญาระหว่าง FE/BE)
- `security.md` — checklist ความปลอดภัยขั้นต่ำ

## เวอร์ชัน

> เลขด้านล่างเป็น **ตัวอย่าง** ให้ทีมเช็คเวอร์ชันปัจจุบันตอนตั้งโปรเจกต์ แล้วเติมเวอร์ชันจริง
> ที่ตกลงใช้ ให้ทุกคน/ทุก environment ตรงกัน (แหล่งความจริง = `package.json` / `go.mod`)

| ส่วน | เวอร์ชัน (ตัวอย่าง — ยืนยันก่อนใช้) |
| --- | --- |
| Node.js | (เช่น 22 LTS) |
| React | (เช่น 19.x) |
| Go | (เช่น 1.24) |
| PostgreSQL | (เช่น 17) |
