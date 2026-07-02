# Configuration Schema

รายการ environment variable ทั้งหมดของโปรเจกต์ (แหล่งอ้างอิงว่ามี var อะไรบ้าง)
ใช้ร่วมกับ `security.md` (ห้าม hardcode/commit secret) และ `tech-stack.md`

> **แหล่งความจริงของ "มี var อะไร" = `.env.example`** (commit เข้า git, ไม่มีค่าจริง)
> ทุกครั้งที่เพิ่ม var ใหม่ ต้องอัปเดต `.env.example` + เอกสารนี้พร้อมกัน
> ค่าจริงอยู่ใน `.env` (**gitignore**) — บน staging/prod ใช้ secret manager ไม่ commit

---

## 1. Backend (`backend/.env`)

| ตัวแปร | ตัวอย่าง | จำเป็น | หมายเหตุ |
| --- | --- | --- | --- |
| `APP_ENV` | `dev` | ✅ | `dev` / `staging` / `prod` — คุมพฤติกรรม (เช่น AutoMigrate, log level) |
| `APP_PORT` | `8080` | ✅ | พอร์ตที่ Gin ฟัง |
| `DB_HOST` | `localhost` | ✅ | |
| `DB_PORT` | `5432` | ✅ | |
| `DB_USER` | `postgres` | ✅ | |
| `DB_PASSWORD` | `••••••` | ✅ | **secret** |
| `DB_NAME` | `app_db` | ✅ | |
| `DB_SSLMODE` | `disable` | ✅ | dev = `disable`, prod = `require` ขึ้นไป |
| `JWT_SECRET` | `••••••` | ✅ | **secret** — ดู `auth.md` |
| `JWT_ACCESS_TTL` | `15m` | ✅ | อายุ access token |
| `JWT_REFRESH_TTL` | `168h` | ✅ | อายุ refresh token (เช่น 7 วัน) |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173` | ✅ | คั่นด้วย comma; **ห้าม `*` ใน prod** (ดู `security.md`) |
| `LOG_LEVEL` | `info` | ✅ | `debug`/`info`/`warn`/`error` (ดู `error-logging.md`) |

---

## 2. Frontend (`frontend/.env`)

> Vite เปิดให้ client อ่านเฉพาะตัวแปรที่ขึ้นต้น **`VITE_`** เท่านั้น →
> **ห้ามใส่ secret ใด ๆ** ที่นี่ (ทุกอย่างมองเห็นได้ใน bundle — ดู `security.md`)

| ตัวแปร | ตัวอย่าง | จำเป็น | หมายเหตุ |
| --- | --- | --- | --- |
| `VITE_API_URL` | `http://localhost:8080` | ✅ | base URL ของ backend |
| `VITE_ENV` | `dev` | – | ใช้แยกพฤติกรรม UI ตาม environment |

---

## 3. กติกาการใช้ config

- **ห้าม hardcode** ค่าใด ๆ ในโค้ด — อ่านจาก `.env` ทั้งสองฝั่ง
- Backend โหลด config ผ่าน `viper`/`godotenv` เข้า **struct config** ที่ `internal/config/` (typed)
- **fail-fast**: ตอน start ถ้า var ที่ `จำเป็น` หายหรือผิดรูปแบบ ให้ **หยุดทันทีพร้อมข้อความชัด** ไม่ปล่อยรันด้วยค่า default เงียบ ๆ
- `.env` และ `.env.*.local` ต้องอยู่ใน `.gitignore`; commit เฉพาะ `.env.example`
- guard hook (`.claude/hooks/guard-secrets.sh`) จะบล็อกไม่ให้ agent อ่าน/แก้ `.env` จริง แต่ยอมให้แตะ `.env.example`

---

## 4. `.env.example` — ต้องมีและ sync เสมอ

ให้ทั้ง `backend/.env.example` และ `frontend/.env.example` มี **ทุก key** ตามตารางด้านบน โดยใส่ค่าตัวอย่าง/ว่าง (ไม่ใส่ค่า secret จริง) เช่น:

```dotenv
# backend/.env.example
APP_ENV=dev
APP_PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=
DB_NAME=app_db
DB_SSLMODE=disable
JWT_SECRET=
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h
CORS_ALLOWED_ORIGINS=http://localhost:5173
LOG_LEVEL=info
```

---

## 5. Checklist ที่ qa-tester ตรวจ

- 🔴 secret hardcode ในโค้ด / `.env` หลุดเข้า git / `VITE_` มี secret
- 🟠 var จำเป็นหายแล้ว app ยังรันด้วย default เงียบ ๆ (ไม่ fail-fast) / `.env.example` ไม่ sync กับที่โค้ดใช้จริง
- 🟡 ไม่มี `.env.example` / ค่า default ไม่เหมาะกับ prod (เช่น `DB_SSLMODE=disable`)
