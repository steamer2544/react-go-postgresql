# Plan: ระบบ Authentication + Authorization + โปรไฟล์/ลายเซ็น (slug: user-auth)

> อ้างอิงมาตรฐาน: `.claude/docs/auth.md`, `api-response.md`, `security.md`, `error-logging.md`,
> `config.md`, `backend-structure.md`, `frontend-structure.md`, `testing.md`, `standard-libraries.md`
> และ `.claude/rules/{naming-conventions,backend,frontend}.md`
> Backend Go module = `imaxx-backend` (ดู router/main ที่มีอยู่)

---

## เป้าหมาย / Definition of Done

ผู้ใช้ login ได้ด้วย email/password (password ถูก hash ด้วย bcrypt เสมอ), ได้ JWT access token,
เรียก endpoint ที่ป้องกันด้วย middleware ตรวจ JWT + RBAC (role: `admin` | `creator` | `approver`),
ดูข้อมูลตัวเอง (`GET /me`), แก้โปรไฟล์ (`PUT /me/profile`), และอัปโหลดรูปลายเซ็น
(`POST /me/signature`) โดยไฟล์ถูกเก็บลง disk และ path ถูกบันทึกใน `signature_image_path`
พร้อมดึงกลับมาแสดงได้. ฝั่ง Frontend มีหน้า Login, หน้า Profile (แก้ชื่อ/ตำแหน่ง + อัปโหลดลายเซ็น
พร้อม preview), route guard ตาม role, และ axios แนบ `Authorization` header อัตโนมัติ.

DoD ถือว่าเสร็จเมื่อ:

- ทุก Acceptance Criteria (AC) ด้านล่างมี test ครอบและ PASS
- `cd backend && gofmt -l . && go vet ./... && go build ./... && go test ./...` ผ่าน
- `cd frontend && npm run lint && npm run build && npm test` ผ่าน
- ไม่มี password/token/secret หลุดใน log หรือ response (ตรวจตาม `security.md` + `error-logging.md`)
- `.env.example` ทั้งสองฝั่ง sync กับ env var ใหม่ทุกตัว (ตาม `config.md`)

---

## ขอบเขต

### Frontend (`frontend/src/`)

- `services/apiClient.js` — เพิ่ม **request interceptor** แนบ `Authorization: Bearer <token>`; จัดการ 401 (เคลียร์ token + redirect login)
- `features/auth/services/authService.js` — `login`, `getMe`, `updateProfile`, `uploadSignature`, `getSignatureUrl`
- `features/auth/hooks/` — `useLogin`, `useMe`, `useUpdateProfile`, `useUploadSignature` (react-query)
- `contexts/AuthContext.jsx` + `hooks/useAuth.js` — เก็บ token + user ปัจจุบัน, `login()/logout()`
- `features/auth/pages/LoginPage.jsx`, `features/auth/pages/ProfilePage.jsx`
- `components/common/` — `RequireAuth.jsx`, `RequireRole.jsx` (route guard)
- `routes/AppRoutes.jsx` (+ ต่อใน `App.jsx`) — ผูก route + guard
- `constants/apiEndpoints.js` — path ของ auth/me endpoints
- ใช้ `react-hook-form` + `zod`, `react-toastify` (error), Bootstrap 5 (ตาม `standard-libraries.md`)

### Backend (`backend/`)

- `internal/config/config.go` — เพิ่ม config การอัปโหลดลายเซ็น (dir, max bytes, allowed types)
- `internal/model/user.go` — GORM model `User`
- `internal/dto/auth_dto.go` — request/response DTO (login, profile update, me response)
- `internal/repository/user_repository.go` — interface `UserRepository` + gorm impl
- `internal/service/auth_service.go` — login + ออก JWT; `internal/service/profile_service.go` — me/update/signature
- `internal/service/token_service.go` (หรือ `pkg/token/`) — ออก/ตรวจ JWT (ตรวจ signing method)
- `internal/service/errors.go` — sentinel error (`ErrInvalidCredentials`, `ErrNotFound`, `ErrValidation`, `ErrUnauthorized`, `ErrForbidden`, `ErrUnsupportedFileType`, `ErrFileTooLarge`)
- `internal/middleware/auth.go` (verify JWT → set `userID`,`role`) + `internal/middleware/require_role.go`
- `internal/handler/auth_handler.go` (Login) + `internal/handler/me_handler.go` (GetMe, UpdateProfile, UploadSignature, GetSignature)
- `pkg/response/response.go` — เพิ่ม helper **map domain error → code+status** (ดู task BE-11)
- `internal/router/router.go` — wiring route + middleware
- `backend/migrations/*_create_users.{up,down}.sql`
- `backend/.env.example` — เพิ่ม env var ใหม่

### Database

- ตาราง `users`: `id` (PK), `email` (unique, not null), `password_hash` (not null), `role`
  (enum/check: `admin|creator|approver`), `full_name`, `position`, `signature_image_path` (nullable),
  `created_at`, `updated_at`
- dev = `GORM AutoMigrate` (ลงทะเบียน model ใน main/db init); staging/prod = ไฟล์ `migrations/` (source of truth)

---

## Tasks (เรียงตาม dependency)

### Backend

1. **[BE] Config อัปโหลดไฟล์** — เพิ่มใน `Config`: `SignatureUploadDir` (เช่น `./uploads/signatures`),
   `SignatureMaxBytes` (เช่น `2097152` = 2MB), `SignatureAllowedTypes` (`image/png,image/jpeg`).
   โหลดแบบ fail-fast ตาม pattern เดิม (`loadString`/parse). อัปเดต `backend/.env.example`
   (คีย์ใหม่: `SIGNATURE_UPLOAD_DIR`, `SIGNATURE_MAX_BYTES`, `SIGNATURE_ALLOWED_TYPES`).
2. **[BE] Model `User`** (`internal/model/user.go`) — field PascalCase + `gorm` tag; `Email` `uniqueIndex`;
   `Role` เป็น type string ที่มีค่าคงที่ `RoleAdmin/RoleCreator/RoleApprover`; `PasswordHash` **ไม่มี** json tag ที่ serialize ออก (หรือ `json:"-"`); `SignatureImagePath *string`.
3. **[BE] Migration** — `migrations/000X_create_users.up.sql` / `.down.sql` (สร้าง/ลบตาราง + unique index บน email + check constraint role). ลงทะเบียน AutoMigrate(`&model.User{}`) ตอน start (dev).
4. **[BE] DTO** (`internal/dto/auth_dto.go`) — `LoginRequest{Email,Password}` (`binding:"required,email"` / `required`),
   `LoginResponse{AccessToken}` (+ `RefreshToken` ถ้าตัดสินใจทำ — ดูคำถามค้าง),
   `MeResponse{ID,Email,Role,FullName,Position,SignatureImagePath}` (json snake_case),
   `UpdateProfileRequest{FullName,Position}` (`binding:"required"`). ห้ามมี field password ใน response DTO.
5. **[BE] Token service** — `Issue(userID, role) (string, error)` ใส่ claims `sub`,`role`,`exp`,`iat`
   (อายุจาก `cfg.JWTAccessTTL`), sign HS256 ด้วย `cfg.JWTSecret`; `Verify(token) (claims, error)`
   ต้อง**ตรวจ signing method** (กัน `alg:none`) และ `exp`. ใช้ `github.com/golang-jwt/jwt/v5`.
6. **[BE] Password util** — hash ด้วย `golang.org/x/crypto/bcrypt` (cost ≥ 10); เทียบด้วย `bcrypt.CompareHashAndPassword`. ไม่ log ค่า password/hash.
7. **[BE] Repository** — interface `UserRepository` (`FindByEmail`, `FindByID`, `UpdateProfile`,
   `UpdateSignaturePath`) + gorm impl; ทุก method รับ `context.Context`; query แบบ parameterized (`Where("email = ?", email)`).
8. **[BE] AuthService.Login** — หา user by email → เทียบ bcrypt → ถ้าไม่พบ/ผิด คืน `ErrInvalidCredentials`
   (map เป็น 401 `UNAUTHORIZED`, ข้อความกลาง ๆ ไม่บอกว่า email หรือ password ผิด) → ออก JWT.
9. **[BE] ProfileService** — `GetMe(ctx, userID)`, `UpdateProfile(ctx, userID, req)`,
   `SaveSignature(ctx, userID, file)`: validate content-type ∈ allowed + size ≤ max
   (ไม่ผ่าน → `ErrUnsupportedFileType`/`ErrFileTooLarge` → 400 `VALIDATION_ERROR`),
   เขียนไฟล์ลง `SignatureUploadDir` ด้วยชื่อ deterministic (เช่น `user_<id>.<ext>`) → บันทึก path ลง DB.
10. **[BE] Middleware** — `auth.go`: ดึง `Authorization: Bearer` → `tokenService.Verify` → set `userID`,`role` ลง `gin.Context`; พังทุกกรณี → 401 `UNAUTHORIZED`. `require_role.go`: factory `RequireRole(roles...)` → เทียบ role ใน context → ไม่พอ 403 `FORBIDDEN`. **ตัดสินสิทธิ์จาก token เท่านั้น** ห้ามจาก input client.
11. **[BE] Response error mapping** — เพิ่ม helper กลางใน `pkg/response/` (เช่น `Fail(c, err)`) ที่ map
    sentinel error → `error.code` + HTTP ตามตารางใน `error-logging.md` (`ErrNotFound→404 NOT_FOUND`,
    `ErrConflict→409 CONFLICT`, `ErrValidation→400 VALIDATION_ERROR`, `ErrUnauthorized/ErrInvalidCredentials→401 UNAUTHORIZED`,
    `ErrForbidden→403 FORBIDDEN`, อื่น ๆ→500 `INTERNAL_ERROR` ไม่ leak detail). handler ทุกตัวตอบผ่าน helper นี้ + helper `Success` เดิม.
12. **[BE] Handler** — `auth_handler.Login` (POST, public); `me_handler`: `GetMe` (GET `/me`),
    `UpdateProfile` (PUT `/me/profile`), `UploadSignature` (POST `/me/signature`, multipart form field `signature`),
    `GetSignature` (GET `/me/signature` stream ไฟล์ด้วย content-type ถูกต้อง). handler validate/bind + เรียก service + ตอบผ่าน `pkg/response/`.
13. **[BE] Router wiring** — public: `POST /auth/login`. protected group ใช้ `middleware.Auth(tokenService)`:
    `GET /me`, `PUT /me/profile`, `POST /me/signature`, `GET /me/signature`. เพิ่ม route ตัวอย่างที่ป้องกันด้วย
    `RequireRole("admin")` เพื่อพิสูจน์ RBAC (เช่น `GET /admin/ping`). wire service/repo/handler ใน `main.go` (ส่ง `*gorm.DB`, `cfg`).
14. **[BE] User seeding (สำหรับ dev/test)** — ช่องทางสร้าง user เริ่มต้น (เช่น seed admin/creator/approver
    ตอน start เมื่อ `APP_ENV=dev`, password จาก env/const dev เท่านั้น) เพื่อให้ login ทดสอบได้ — **ไม่มี**
    endpoint `/auth/register` แบบ public (ดูคำถามค้าง).

### Frontend

15. **[FE] constants/apiEndpoints.js** — `AUTH_LOGIN='/auth/login'`, `ME='/me'`, `ME_PROFILE='/me/profile'`, `ME_SIGNATURE='/me/signature'`.
16. **[FE] apiClient interceptor** — request interceptor แนบ `Authorization: Bearer <token>` (อ่าน token จาก
    auth store); response error 401 → เคลียร์ token + ส่งสัญญาณ logout. คง response-unwrap เดิมไว้.
17. **[FE] AuthContext + useAuth** — state: `token`, `user`, `isAuthenticated`; `login(credentials)`,
    `logout()`; เก็บ token ตามที่ตกลง (ดูคำถามค้าง — ค่าเริ่มต้น: memory + persist สำหรับ guard ตอน reload).
18. **[FE] authService.js** — ยิง axios ล้วนผ่าน `apiClient`: `login`, `getMe`, `updateProfile(payload)`,
    `uploadSignature(file)` (ส่ง `FormData` field `signature`, header multipart), `getSignatureUrl()`.
19. **[FE] react-query hooks** — `useLogin` (mutation → เซ็ต token+user), `useMe` (query),
    `useUpdateProfile` (mutation → invalidate `me`), `useUploadSignature` (mutation → invalidate `me`). `retry:false`.
20. **[FE] LoginPage** — form (`react-hook-form`+`zod`: email required+email, password required); submit → `useLogin`;
    error 401 → toast ข้อความกลาง (เทียบด้วย `error.code === 'UNAUTHORIZED'`); สำเร็จ → redirect หน้าโปรไฟล์/หน้าแรก.
21. **[FE] ProfilePage** — แสดง `useMe`; ฟอร์มแก้ `full_name`/`position` → `useUpdateProfile`;
    ส่วนอัปโหลดลายเซ็น: เลือกไฟล์ → **preview** (URL.createObjectURL) → submit `useUploadSignature`;
    แสดงลายเซ็นปัจจุบันจาก `signature_image_path` (ผ่าน `GET /me/signature`).
22. **[FE] Route guard** — `RequireAuth` (ไม่มี token → redirect `/login`), `RequireRole(allowed)` (role ไม่อยู่ใน
    allowed → redirect/หน้า 403). ผูกใน `AppRoutes.jsx` + ต่อ `App.jsx` (ต้องมี `QueryClientProvider`, `AuthProvider`, `BrowserRouter`).

### Test (สำหรับ test-case-writer — ระบุที่วางไฟล์)

23. Backend: `*_test.go` คู่ไฟล์ (unit service ด้วย `testify/mock` ของ `UserRepository`; integration repo ตาม `testing.md` ถ้าจำเป็น). ตั้งชื่อ `TestX_TC0N_...`.
24. Frontend: Vitest + RTL + MSW (mock HTTP ที่ขอบ) สำหรับ LoginPage/ProfilePage/guard/service.

---

## Acceptance Criteria (ทดสอบได้)

**Auth / Login**

- **AC1**: `POST /auth/login` ด้วย email+password ที่ถูกต้อง → 200, body `data.access_token` เป็น JWT ที่ถอด claims ได้ `sub`,`role`,`exp`,`iat` (มี `exp` เสมอ).
- **AC2**: `POST /auth/login` ด้วย password ผิด **หรือ** email ไม่มีในระบบ → 401 body `error.code === "UNAUTHORIZED"` และ **ข้อความไม่ระบุ**ว่า email หรือ password ผิด (กัน user enumeration).
- **AC3**: `POST /auth/login` ไม่มี email/password (validation fail) → 400 `error.code === "VALIDATION_ERROR"`.
- **AC4**: password ทุกตัวถูกเก็บเป็น bcrypt hash (`$2a$`/`$2b$` prefix, cost ≥ 10) — ไม่มี plaintext ใน DB; login เทียบด้วย `bcrypt.CompareHashAndPassword`.

**Middleware / RBAC**

- **AC5**: เรียก endpoint ที่ป้องกัน (`GET /me`) โดย**ไม่มี** `Authorization` header → 401 `UNAUTHORIZED`.
- **AC6**: เรียกด้วย token ที่ signing method ไม่ถูก (`alg:none`) หรือ token หมดอายุ (`exp` ผ่านแล้ว) → 401 `UNAUTHORIZED` (Verify ปฏิเสธ).
- **AC7**: role ที่ไม่มีสิทธิ์เรียก endpoint ที่ต้อง `RequireRole("admin")` (เช่น `creator`) → 403 `FORBIDDEN`; role `admin` → 200. สิทธิ์ตัดสินจาก claim ใน token เท่านั้น (ส่ง `?role=admin` ไม่มีผล).

**Me / Profile**

- **AC8**: `GET /me` ด้วย token ถูกต้อง → 200, `data` มี `email`,`role`,`full_name`,`position`,`signature_image_path` และ **ไม่มี** `password`/`password_hash`.
- **AC9**: `PUT /me/profile` ด้วย `{full_name, position}` ถูกต้อง → 200 และค่าใน DB ของ user นั้นอัปเดต; `GET /me` ครั้งถัดไปคืนค่าใหม่. userID มาจาก token (แก้ได้เฉพาะของตัวเอง).

**Signature upload**

- **AC10**: `POST /me/signature` อัปโหลดไฟล์ png/jpeg ขนาด ≤ limit → 200/201, ไฟล์ถูกเขียนลง `SignatureUploadDir`, `signature_image_path` ใน DB ถูกบันทึก.
- **AC11**: หลังอัปโหลดสำเร็จ `GET /me` คืน `signature_image_path` ที่ไม่ว่าง และ `GET /me/signature` คืนไฟล์รูป (content-type เป็น image/*) ที่ดึงกลับมาแสดงได้.
- **AC12**: อัปโหลดไฟล์ชนิดไม่รองรับ (เช่น `application/pdf`/`text/plain`) → 400 `VALIDATION_ERROR`, ไม่มีการเขียนไฟล์/แก้ DB.
- **AC13**: อัปโหลดไฟล์เกินขนาด limit → 400 `VALIDATION_ERROR`, ไม่มีการเขียนไฟล์/แก้ DB.

**Security / Logging**

- **AC14**: ไม่มี log บรรทัดใดมีค่า password, password_hash, หรือ JWT token (ตรวจตาม `security.md`/`error-logging.md`).
- **AC15**: error 500 ใด ๆ ไม่ leak stack/SQL/path ให้ client (ตอบ `INTERNAL_ERROR` ข้อความกลาง). ทุก response ตอบผ่าน `pkg/response/`.
- **AC16**: `backend/.env.example` มีคีย์ใหม่ครบ (`SIGNATURE_UPLOAD_DIR`,`SIGNATURE_MAX_BYTES`,`SIGNATURE_ALLOWED_TYPES`) และ config fail-fast เมื่อ var จำเป็นหาย.

**Frontend**

- **AC17**: กรอกฟอร์ม Login แล้ว submit สำเร็จ (mock 200 ด้วย MSW) → token ถูกเก็บใน auth store และ redirect ออกจากหน้า login.
- **AC18**: Login ล้มเหลว (mock 401) → แสดง toast/ข้อความ error โดยเทียบจาก `error.code === "UNAUTHORIZED"` (ไม่เทียบข้อความ), ไม่โยน raw error object ให้ผู้ใช้.
- **AC19**: request ที่ยิงหลัง login แนบ header `Authorization: Bearer <token>` (ตรวจผ่าน MSW request assertion).
- **AC20**: เข้าถึง route ที่ป้องกันโดยยังไม่ login → ถูก redirect ไป `/login`; role ไม่ตรง `RequireRole` → ไม่เห็นหน้า/redirect 403.
- **AC21**: ในหน้า Profile เลือกไฟล์รูป → เห็น **preview** ก่อน submit; อัปโหลดสำเร็จ (mock) → หน้าแสดงลายเซ็นจาก `signature_image_path`; แก้ full_name/position แล้วบันทึกสำเร็จ → ค่าใหม่แสดง (invalidate `me`).

---

## ความเสี่ยง / คำถามค้าง

1. **Refresh token / logout**: `auth.md` มาตรฐานมี refresh + logout + `/auth/refresh` แต่โจทย์ระบุแค่ "login → คืน JWT".
   → แผนนี้ทำ **access token อย่างเดียว** ให้ตรงโจทย์. ต้องยืนยัน: ทำ refresh token ตอนนี้ด้วยไหม หรือค่อยเฟสหน้า?
2. **Path ของ endpoint `me`**: โจทย์ใช้ `GET /me`, `PUT /me/profile`, `POST /me/signature` แต่ `auth.md` ตัวอย่างใช้ `/auth/me`.
   → แผนยึดตามโจทย์ (`/me...`). ยืนยันว่าโอเคหรือให้ย้ายไปใต้ `/auth`?
3. **การสร้าง user ครั้งแรก**: ไม่มี `/auth/register` public (role admin/creator/approver ควรถูกสร้างโดย admin/seed).
   → แผนเสนอ **seed dev users** เพื่อทดสอบ. ต้องยืนยันวิธี provisioning จริง (admin สร้าง? seed script? migration?).
4. **การเก็บ token ฝั่ง FE**: `security.md`/`auth.md` แนะนำ httpOnly cookie (refresh) + access ใน memory.
   แต่ไม่มี refresh endpoint และ guard ต้องรอด reload → ต้องเลือก: (a) เก็บ access ใน `localStorage` (เสี่ยง XSS, ทีมต้องรับความเสี่ยง + sanitize) หรือ (b) memory อย่างเดียว (reload = ต้อง login ใหม่).
   → ค่าเริ่มต้นที่เสนอ: **localStorage สำหรับ MVP** พร้อม note ความเสี่ยง — ขอ decision ทีม.
5. **การ serve ไฟล์ลายเซ็น**: เลือก **authenticated `GET /me/signature`** (stream ไฟล์ของ user ตาม token) แทน static public dir
   เพื่อกันการเข้าถึงลายเซ็นคนอื่นโดยตรง. ยืนยันแนวทางนี้ (กระทบว่า FE ใช้ blob URL แทน `<img src=path>` ตรง ๆ).
6. **Signature ควรลบไฟล์เก่าเมื่ออัปโหลดใหม่ไหม**: เสนอใช้ชื่อ deterministic (`user_<id>.<ext>`) แล้ว overwrite เพื่อกันไฟล์ค้าง — ยืนยัน.
7. **pkg/response ปัจจุบัน** มีแค่ `Success/Error/List` (Error รับ args ตรง) ยังไม่มี mapping จาก domain error → ต้องเพิ่ม helper (task BE-11) โดยไม่ทำ signature เดิมพัง.
