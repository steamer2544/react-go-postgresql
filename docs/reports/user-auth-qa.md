# QA Report: Authentication + Authorization + โปรไฟล์/ลายเซ็น (slug: user-auth)

อ้างอิง: docs/plans/user-auth.md · docs/tests/user-auth-testcases.md
รอบที่: 2 | วันที่: 2026-07-07

## ผลรวม: PASS

- test ผ่าน 57/57 (backend 44, frontend 13 — เพิ่ม 2 จากรอบ 1 สำหรับ round-trip ของ AC21) — ทุก test PASS จริง ไม่ใช่ assertion อ่อน/scope ออก
- `gofmt -l .` สะอาด, `go vet ./...` ผ่าน, `go build ./...` ผ่าน
- frontend `npm run lint` ผ่าน (0 error, 1 warning เดิมที่ยอมรับแล้วตั้งแต่รอบ 1), `npm run build` ผ่าน, `npm test` ผ่านทั้ง 6 ไฟล์ 13/13 test
- AC ครบ **21/21** — AC21 ที่เป็น blocker เดียวของรอบ 1 ได้รับการแก้แล้ว: `ProfilePage.jsx` เรียก `useUploadSignature` จริงตอน submit และแสดงลายเซ็นเดิมผ่าน `useSignatureUrl` + `authService.getSignatureUrl()` (ดึงจาก authenticated `GET /me/signature` เป็น blob) พร้อม test พิสูจน์ round-trip ทั้งสองทิศทาง (fetch+display ของเดิม / upload จริงตอน submit)

## ผลรัน test จริง (รอบ 2)

**Backend** (`cd backend && gofmt -l . && go vet ./... && go build ./... && go test ./... -v`)

- `gofmt -l .` → ไม่มี output (สะอาด)
- `go vet ./...` → ผ่าน ไม่มี warning
- `go build ./...` → ผ่าน
- `go test ./... -v` → **PASS ทั้งหมด 44 test** ครบทุกแพ็กเกจ (`internal/config`, `internal/handler`, `internal/middleware`, `internal/service`, `pkg/response`) — ไม่มีการแก้โค้ด backend เพิ่มจากรอบ 1 (ยกเว้น `SaveSignature` ตามข้อ 3 ด้านล่าง) ผลจึงเหมือนเดิมทุกเทส

**Frontend** (`cd frontend && npm run lint && npm run build && npm test`)

- `npm run lint` → 0 error, 1 warning (`AuthContext.jsx` react-refresh/only-export-components — ยอมรับแล้วตั้งแต่รอบ 1 เพราะ export `AuthContext` ตามสัญญา test)
- `npm run build` → ผ่าน (vite build สำเร็จ, 706ms)
- `npm test` → **6 test files passed, 13/13 test PASS** (เพิ่มไฟล์ใหม่ `ProfilePage.signature.test.jsx` 2 tests จากเดิม 11)

## ผล test case

| ID                      | คาด                                                                                                 | ได้จริง                                                                                                                                                                                                                                                                         | สถานะ |
| ----------------------- | --------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----- |
| TC-01a/b                | AC1: login ถูกต้อง → JWT มี sub/role/exp/iat, handler 200                                           | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-02a/b/c              | AC2: password/email ผิด → ErrInvalidCredentials เดียวกัน, handler 401 UNAUTHORIZED                  | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-03a/b                | AC3: ขาด email/password → 400 VALIDATION_ERROR, service ไม่ถูกเรียก                                 | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-04a/b/c              | AC4: bcrypt hash cost≥10, compare ถูก/ผิด                                                           | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-05                   | AC5: ไม่มี header → 401                                                                             | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-06a/b/c              | AC6: alg:none / expired ถูกปฏิเสธ, middleware 401                                                   | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-07a/b/c              | AC7: role ไม่พอ→403, พอ→200, query `?role=admin` ไม่มีผล                                            | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-auth-valid           | (สนับสนุน) token ถูกต้อง → set userID/role แล้วเรียก next                                           | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-08a/b                | AC8: GetMe ไม่มี password field                                                                     | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-09a/b                | AC9: update แล้ว GetMe เห็นค่าใหม่, userID มาจาก context เท่านั้น                                   | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-10a/b                | AC10: อัปโหลด png → เขียนไฟล์ + บันทึก path, 200                                                    | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-11                   | AC11: ดึงกลับมาด้วย content-type image/*                                                            | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-12a/b                | AC12: ชนิดไฟล์ไม่รองรับ → 400, ไม่เขียนไฟล์/DB                                                      | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-13a/b                | AC13: ไฟล์เกินขนาด → 400, ไม่เขียนไฟล์/DB                                                           | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-14                   | AC14: error message ไม่มี raw password                                                              | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-15, TC-map-*         | AC15: mapping ครบตาราง (404/409/400/401/403/500 ไม่ leak)                                           | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-16a/b/c              | AC16: config โหลดคีย์ใหม่, fail-fast, `.env.example` sync                                           | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-17                   | AC17: login สำเร็จ → token ใน localStorage + redirect ออกจาก /login                                 | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-18                   | AC18: 401 → ข้อความคงที่ ไม่ใช่ backend message, ไม่มี `[object Object]`                            | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-19a/b/c              | AC19: แนบ header เมื่อมี token / ไม่แนบเมื่อไม่มี / เคลียร์ token เมื่อ 401                         | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-20a/b/c/d            | AC20: RequireAuth/RequireRole redirect ถูกต้อง                                                      | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-21a                  | AC21 (บางส่วน): เลือกไฟล์ → preview ด้วย blob: URL                                                  | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| TC-21b                  | AC21 (บางส่วน): แก้ full_name/position → Save → invalidate `me` แสดงค่าใหม่                         | ตรง                                                                                                                                                                                                                                                                             | ✅    |
| **AC21 (เต็ม) — NEW-1** | ลายเซ็นเดิม (`signature_image_path` ไม่ว่าง) แสดงผลผ่าน `GET /me/signature` → blob: URL ตอนโหลดหน้า | `ProfilePage.signature.test.jsx::"existing signature is fetched and displayed via GET /me/signature on page load"` — mock GET /me คืน `signature_image_path` ไม่ว่าง + mock GET /me/signature คืน PNG bytes จริง → assert `data-testid="current-signature"` src ขึ้นต้น `blob:` | ✅    |
| **AC21 (เต็ม) — NEW-2** | เลือกไฟล์ใหม่ → กด Save → ยิง `POST /me/signature` จริง (ไม่ใช่แค่ preview เฉย ๆ)                   | `ProfilePage.signature.test.jsx::"selecting a new signature file and clicking Save uploads it via POST /me/signature"` — mock POST /me/signature ตั้ง flag `uploadCalled` → `user.upload` ไฟล์ + `user.click` Save → `waitFor(() => expect(uploadCalled).toBe(true))` ผ่าน      | ✅    |

## ตรวจ regression เพิ่มเติม (AC1–AC20)

- รันชุด test เต็มทั้งสองฝั่งซ้ำ (ไม่ใช่แค่ไฟล์ที่เปลี่ยน) → **ผลเท่ากับรอบ 1 ทุกเคส ไม่มีตกถดถอย**
- backend ไม่มีไฟล์ handler/service/middleware/router ใดถูกแก้เพิ่มเทียบรอบ 1 ยกเว้น `profile_service.go::SaveSignature` (ดูข้อ 3 ใน "จุดที่ตรวจแล้วผ่าน" ด้านล่าง) ซึ่งเป็นการปรับ defense-in-depth ไม่กระทบ contract/AC ใด — ยืนยันด้วย `go test ./... -v` PASS ครบ 44 test เดิม
- frontend ไฟล์ที่แก้เพิ่ม: `ProfilePage.jsx`, `authService.js` (เพิ่ม `getSignatureUrl`), `useSignatureUrl.js` (ใหม่), `apiClient.js` (เพิ่ม exception ให้ `responseType: arraybuffer/blob` ข้ามการ unwrap envelope — จำเป็นสำหรับ endpoint ที่ตอบไฟล์ดิบตามแผน task 12 ไม่ใช่ endpoint แบบ envelope ปกติ จึงไม่ขัดกับ `api-response.md`) — ทั้งหมดนี้ไม่กระทบ flow เดิมของ AC17–AC20 (ยืนยันด้วยผลเทสเดิมของ `LoginPage.test.jsx`, `apiClient.test.js`, `RequireAuth.test.jsx`, `RequireRole.test.jsx` ที่ยัง PASS ครบ)

## เช็ค non-blocking items จากรอบ 1

1. **`io.LimitReader` ใน `SaveSignature`** (`backend/internal/service/profile_service.go:75-83`) — **แก้แล้ว**: เปลี่ยนจาก `io.ReadAll(upload.Reader)` ตรง ๆ เป็น `io.ReadAll(io.LimitReader(upload.Reader, s.maxBytes+1))` แล้วเช็ค `len(data) > int(s.maxBytes)` คืน `ErrFileTooLarge` ก่อน `WriteFile` — ป้องกัน client โกหก `Size`/`Content-Length` ได้จริงแล้ว (defense-in-depth ตามที่แนะนำ)
2. **dead `onError` ใน `LoginPage.jsx`** — **แก้แล้ว**: ลบ callback ว่างออก เหลือแค่ `onSuccess: () => navigate('/')` ใน `loginMutation.mutate(...)` (`frontend/src/features/auth/pages/LoginPage.jsx:11-19`) สะอาดขึ้น ไม่มี dead code
3. **`devPassword` hardcode ใน `backend/cmd/api/main.go:18`** — **ยังไม่แก้** (`const devPassword = "Passw0rd!"` คงเดิม) — ตามที่ระบุไว้ในรอบ 1 นี่เป็น suggestion ไม่บังคับ (ถูก guard ด้วย `cfg.AppEnv == "dev"`, ไม่ log) ยังไม่ถือเป็น blocker ในรอบนี้เช่นกัน แต่ยังแนะนำให้ย้ายไปอ่านจาก env var ในอนาคตเพื่อสอดคล้อง `config.md` เคร่งครัดขึ้น

## ปัญหาที่พบ (เรียงตามความรุนแรง)

- 🟡 **Suggestion (ค้างจากรอบ 1 ไม่บล็อก)**: `backend/cmd/api/main.go:18` `devPassword` ยัง hardcode — แนะนำย้ายไปอ่านจาก env var (เช่น `DEV_SEED_PASSWORD` พร้อม default ใน `.env.example`)
- 🟡 **Suggestion**: eslint warning ที่ `frontend/src/contexts/AuthContext.jsx:3` (`react-refresh/only-export-components`) ยังคงอยู่ตามที่ยอมรับไว้ตั้งแต่รอบ 1 (ตั้งใจ export `AuthContext` เพื่อให้ test ฉีด mock ได้) ไม่ใช่บั๊ก
- 🟡 **Suggestion**: `ProfilePage.signature.test.jsx` เทส "selecting a new signature file and clicking Save uploads it" ยืนยันแค่ว่า `POST /me/signature` ถูกยิงจริง (`uploadCalled === true`) แต่ไม่ได้ยืนยันต่อว่าหลังอัปโหลดสำเร็จหน้าจอ re-render แสดงรูปใหม่ (เพราะ mock `GET /me` ในเทสนั้นไม่เปลี่ยน `signature_image_path` หลัง upload) — ส่วนการแสดงลายเซ็นที่มีอยู่แล้วถูกพิสูจน์แยกในเทสแรกของไฟล์เดียวกัน (`existing signature is fetched and displayed...`) รวมสองเทสแล้วครอบคลุมทั้งสองด้านของ AC21 เพียงพอ ไม่ถือเป็น gap ที่ต้องบล็อก แต่ถ้าต้องการความมั่นใจเพิ่มสามารถเพิ่ม assertion ต่อยอด (mock ให้ GET /me คืน path ใหม่หลัง POST สำเร็จ แล้ว assert ว่าเห็น `current-signature` ใหม่) ในรอบถัดไปได้

## จุดที่ตรวจแล้วผ่าน (สะสมจากรอบ 1 + ยืนยันซ้ำรอบ 2)

- bcrypt cost ≥ 10, เปรียบเทียบด้วย `bcrypt.CompareHashAndPassword` เท่านั้น
- ไม่มี log บรรทัดใดพิมพ์ password/hash/token ทั้ง backend/frontend
- RBAC ตัดสินจาก JWT claim ใน context เท่านั้น ไม่มีจุดใดอ่าน role จาก query/body
- `TokenService.Verify` ปฏิเสธ `alg:none` และ token หมดอายุ
- `response.Fail` map error ครบตาราง `error-logging.md`, error ที่ไม่รู้จักตอบ `INTERNAL_ERROR` กลาง ไม่ leak
- `model.User.PasswordHash` มี `json:"-"`; `dto.MeResponse` ไม่มี field password
- CORS อ่าน origin จาก `.env` ไม่เปิด `*`
- `.env.example` ทั้งสองฝั่ง sync ครบ, `config.Load()` fail-fast
- naming convention ถูกต้องทั้งหมดที่ตรวจ (PascalCase export, json snake_case, camelCase local)
- FE เทียบ error ด้วย `error.code` เท่านั้น, apiClient interceptor unwrap envelope ถูกต้อง (ยกเว้น binary responseType ตามที่ตั้งใจ)
- **[ใหม่รอบ 2]** `profile_service.go::SaveSignature` ใช้ `io.LimitReader` ป้องกัน memory exhaustion จาก client ที่โกหกขนาดไฟล์
- **[ใหม่รอบ 2]** `ProfilePage.jsx` เรียก `useUploadSignature().mutate(selectedFile)` จริงใน `onSuccess` ของ `updateMutation` เมื่อมีไฟล์ที่เลือกไว้ (`onSubmit`, บรรทัด 32-51), แสดงลายเซ็นปัจจุบันผ่าน `useSignatureUrl(profile?.signature_image_path)` เมื่อยังไม่ได้เลือกไฟล์ใหม่ (`!previewUrl && existingSignatureUrl`)
- **[ใหม่รอบ 2]** `authService.getSignatureUrl()` ยิง `GET /me/signature` ผ่าน `apiClient` (แนบ Authorization header อัตโนมัติจาก interceptor เดิม) ด้วย `responseType: 'arraybuffer'` แล้วสร้าง `Blob`+`URL.createObjectURL` ตาม content-type จริงจาก response header — ไม่ใช่ static `<img src=path>` ตรง ๆ (ตรงตามการตัดสินใจใน "คำถามค้าง" ข้อ 5 ของแผน)
- **[ใหม่รอบ 2]** `useUploadSignature`/`useUpdateProfile` invalidate `queryKey: ['me']` ซึ่งครอบคลุม `useSignatureUrl` (`queryKey: ['me','signature']`) ด้วยตาม prefix-matching ของ TanStack Query — ทำให้ลายเซ็นใหม่ถูก refetch อัตโนมัติหลังอัปโหลดสำเร็จ

## สรุปสิ่งที่ต้องให้ dev แก้

ไม่มีข้อบังคับเหลือ — **PASS** ปิดงานได้

รายการไม่บังคับ (ทำเมื่อสะดวก ไม่ต้องวนรอบ QA ใหม่เพื่อเรื่องเหล่านี้):

1. ย้าย `devPassword` ใน `backend/cmd/api/main.go:18` ไปอ่านจาก env var
2. (ถ้าต้องการ coverage เพิ่ม) เพิ่ม assertion ต่อยอดใน `ProfilePage.signature.test.jsx` ให้ยืนยันว่าหลัง upload สำเร็จหน้าจอแสดงลายเซ็นใหม่จริง (ปัจจุบันยืนยันแค่ request ถูกยิง)
