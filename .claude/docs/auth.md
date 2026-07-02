# Auth / Authorization

มาตรฐานการยืนยันตัวตน (authentication) และการให้สิทธิ์ (authorization) ของทั้งโปรเจกต์
ใช้ร่วมกับ `security.md` (checklist ขั้นต่ำ) และ `api-response.md` (โค้ด error/สถานะ)

---

## 1. ภาพรวม flow

```
Register  ──> hash password (bcrypt) ──> เก็บ user ใน DB
Login     ──> ตรวจ password ──> ออก access token (สั้น) + refresh token (ยาว)
Request   ──> ส่ง access token (header) ──> middleware verify ──> เข้า handler
Expire    ──> access หมดอายุ ──> เรียก /auth/refresh ด้วย refresh token ──> ได้ access ใหม่
Logout    ──> เพิกถอน refresh token (ลบ/ขึ้นบัญชีดำ)
```

- **access token** — อายุสั้น (เช่น 15 นาที) แนบทุก request ที่ต้องล็อกอิน
- **refresh token** — อายุยาว (เช่น 7 วัน) ใช้ขอ access ใหม่เท่านั้น เพิกถอนได้

---

## 2. Token: เก็บอะไร / เก็บที่ไหน

**JWT claims (access token)** — ใส่เฉพาะข้อมูลที่ไม่อ่อนไหว (payload อ่านได้ ไม่ได้เข้ารหัส):

| claim | ความหมาย |
| --- | --- |
| `sub` | user id |
| `role` | บทบาท (เช่น `admin`, `user`) ใช้ทำ authorization |
| `exp` | เวลาหมดอายุ (บังคับมีทุก token) |
| `iat` | เวลาออก token |

- **ห้าม**ใส่ password, เลขบัตร, PII ลงใน JWT
- verify ต้อง**ตรวจ signing method** (กัน `alg: none`) — ดู `security.md`
- secret อยู่ใน `.env` (`JWT_SECRET`), อายุ token อ่านจาก config (`JWT_ACCESS_TTL`, `JWT_REFRESH_TTL`) → ดู `config.md`

**ฝั่ง Frontend เก็บ token ที่ไหน** (ตกลงวิธีเดียวทั้งทีม):
- แนะนำ **httpOnly cookie** สำหรับ refresh token (กัน XSS อ่านไปได้) — access token เก็บใน memory
- ถ้าใช้ `localStorage` ต้องรับความเสี่ยง XSS และ sanitize ทุกจุด (ดู `security.md`)

---

## 3. Backend — middleware + layering

โครงตามชั้นเดิม (`handler → service → repository`) เพิ่ม middleware 2 ตัวใน `internal/middleware/`:

| ไฟล์ | หน้าที่ |
| --- | --- |
| `auth.go` | ดึง `Authorization: Bearer <token>` → verify JWT → เซ็ต `userID`, `role` ลง `gin.Context` → ถ้าพัง 401 `UNAUTHORIZED` |
| `require_role.go` | middleware แบบ factory `RequireRole("admin")` → เช็ค role ใน context → ไม่พอสิทธิ์ 403 `FORBIDDEN` |

```go
// ใช้ตอนลงทะเบียน route (internal/router/router.go)
admin := r.Group("/admin", middleware.Auth(jwtSvc), middleware.RequireRole("admin"))
admin.GET("/employees", employeeHandler.List)
```

- logic ออก/ตรวจ token อยู่ใน **service** (เช่น `AuthService` / `TokenService`) ไม่ใช่ใน middleware
- middleware ทำแค่ "ดึง–เรียก service–ตัดสิน" ไม่ยัด business logic

---

## 4. Endpoint มาตรฐานของ auth

| Method | Path | ผล |
| --- | --- | --- |
| POST | `/auth/register` | สร้าง user (hash password) → 201 |
| POST | `/auth/login` | ตรวจ credential → 200 + `{ access_token, refresh_token }` (หรือ set cookie) |
| POST | `/auth/refresh` | refresh token ถูกต้อง → 200 + access ใหม่ |
| POST | `/auth/logout` | เพิกถอน refresh token → 204 |
| GET | `/auth/me` | (ต้องล็อกอิน) คืนข้อมูล user ปัจจุบันจาก token |

ทุก response ตอบตาม `api-response.md` (envelope `data`/`error`)

---

## 5. Authorization model

- เริ่มจาก **role-based (RBAC)** ง่าย ๆ: `user.role` 1 ค่า, เช็คด้วย `RequireRole`
- ถ้าโตขึ้นค่อยขยับเป็น permission-based (role → หลาย permission) — ตัดสินใจร่วมทีมก่อน อย่าเพิ่ง over-engineer
- **ห้าม**ตัดสินสิทธิ์จากค่าที่ client ส่งมา (เช่น `?role=admin`) — ยึดจาก token ที่ verify แล้วเท่านั้น
- ตรวจสิทธิ์ระดับข้อมูล (เช่น "แก้ได้เฉพาะของตัวเอง") ทำใน **service** ไม่ใช่แค่ route group

---

## 6. รหัสผ่าน (ย้ำจาก security.md)

- hash ด้วย `golang.org/x/crypto/bcrypt` (cost ≥ 10) — **ห้าม** plaintext
- เทียบด้วย `bcrypt.CompareHashAndPassword` เท่านั้น
- reset password: ออก token แบบใช้ครั้งเดียว มีอายุ ส่งผ่านช่องทางที่ควบคุมได้ ไม่ส่ง password เดิม/ใหม่เป็น plaintext ทาง log/response

---

## 7. Checklist ที่ qa-tester ตรวจ

- 🔴 password ไม่ได้ hash / JWT ไม่ตั้ง `exp` / verify ไม่เช็ค signing method / ตัดสินสิทธิ์จาก input ของ client
- 🟠 access token อายุยาวผิดปกติ / ไม่มี route protection บน endpoint ที่ควรล็อกอิน / refresh เพิกถอนไม่ได้
- 🟡 เก็บ token ฝั่ง FE ไม่ตรงที่ตกลง / error auth หลุด detail ภายใน
