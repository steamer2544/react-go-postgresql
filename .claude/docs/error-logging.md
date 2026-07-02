# Error Handling & Logging

มาตรฐานการจัดการ error และ logging ของทั้งโปรเจกต์ ใช้ร่วมกับ `api-response.md`
(รูปแบบ error ที่ส่ง client) และ `security.md` (ห้าม log/leak ข้อมูลอ่อนไหว)

---

## 1. Backend — error ไหลผ่านชั้นยังไง

หลักการ: **แต่ละชั้นเพิ่มบริบทของตัวเอง แล้วให้ handler เป็นที่เดียวที่แปลงเป็น HTTP response**

```
repository ─ คืน error ดิบ/wrap สั้น ──► service ─ ตีความเป็น domain error ──► handler ─ แปลงเป็น envelope + status
```

- **wrap ด้วย `%w`** เพื่อคงต้นตอ (unwrap ได้):
  ```go
  if err != nil {
      return fmt.Errorf("employeeRepo.Create: %w", err)
  }
  ```
- ตรวจชนิด error ด้วย `errors.Is` / `errors.As` ไม่เทียบข้อความ string

---

## 2. Domain error → error.code + HTTP status (mapping ที่เดียว)

ประกาศ **sentinel/typed error** ระดับโดเมน แล้ว map เป็นโค้ดมาตรฐานที่จุดเดียว (เช่น `pkg/response/` หรือ `pkg/errs/`):

| domain error | error.code | HTTP |
| --- | --- | --- |
| `ErrNotFound` | `NOT_FOUND` | 404 |
| `ErrConflict` (เช่น email ซ้ำ) | `CONFLICT` | 409 |
| `ErrValidation` | `VALIDATION_ERROR` | 400 |
| `ErrUnauthorized` | `UNAUTHORIZED` | 401 |
| `ErrForbidden` | `FORBIDDEN` | 403 |
| อื่น ๆ (ไม่รู้จัก) | `INTERNAL_ERROR` | 500 |

```go
// handler แปลงครั้งเดียว ผ่าน helper กลาง
func (h *EmployeeHandler) Create(c *gin.Context) {
    out, err := h.svc.CreateEmployee(c.Request.Context(), req)
    if err != nil {
        response.Error(c, err) // helper map domain error -> code+status ตาม api-response.md
        return
    }
    response.Created(c, out, "created successfully")
}
```

- **ห้าม**ปั้น JSON error เองในแต่ละ handler — ผ่าน `pkg/response/` เท่านั้น
- error 500 **ห้าม leak** stack/SQL/path ให้ client (log ฝั่ง server, ตอบข้อความกลาง ๆ) — ดู `security.md`

---

## 3. Logging (structured)

- ใช้ **structured logger** (`zap` หรือ `logrus` ตาม `standard-libraries.md`) ไม่ใช่ `fmt.Println`
- log **ที่ขอบระบบ** (middleware, จุดจับ error สุดท้าย) ไม่ใช่โปรยทุกบรรทัด
- ระดับ log: `error` (ต้องมีคนดู) / `warn` (ผิดปกติแต่ไปต่อได้) / `info` (เหตุการณ์สำคัญ) / `debug` (เฉพาะ dev) — ระดับอ่านจาก `LOG_LEVEL` (ดู `config.md`)
- **ห้าม log** password, token, PII — ตัด field เหล่านี้ก่อน log (ย้ำจาก `security.md`)

### Request ID (correlation)
- middleware สร้าง request id (uuid) ต่อ request → ใส่ใน `context` + response header `X-Request-ID` → แนบทุกบรรทัด log ของ request นั้น
- เวลา debug prod: เอา `X-Request-ID` จาก client ไปเทียบ log ได้ตรงคำขอ

### panic
- middleware `recovery` จับ panic → log stack **ฝั่ง server** → ตอบ client 500 `INTERNAL_ERROR` (ไม่มี stack) — server ต้องไม่ล่ม

---

## 4. Frontend — จัดการ error จาก API

- interceptor กลางใน `services/apiClient.js` แกะ `error.response.data.error` (ตาม `api-response.md`) → โยน error ที่มี `code` + `message`
- ชั้น hook/component **เทียบด้วย `error.code`** (ไม่เทียบข้อความ) เพื่อรองรับ i18n
- แสดงให้ผู้ใช้เป็นข้อความที่เตรียมไว้ (เช่น toast ผ่าน `react-toastify`) — **ห้ามโยน raw error object** ให้ผู้ใช้
- error ที่ไม่คาด → ข้อความกลาง ๆ ("เกิดข้อผิดพลาด ลองใหม่อีกครั้ง") ไม่โชว์ detail ภายใน
- อย่า `console.log` token/ข้อมูลอ่อนไหว (หลุดใน devtools/prod)

---

## 5. Checklist ที่ qa-tester ตรวจ

- 🔴 error 500 leak stack/SQL/path ให้ client / log password-token-PII
- 🟠 handler ปั้น JSON error เอง ไม่ผ่าน `pkg/response/` / ไม่ wrap error (สืบต้นตอไม่ได้) / ไม่มี recovery middleware
- 🟡 ไม่มี request id / log ระดับผิด (info ล้น หรือ error เงียบ) / FE เทียบ error ด้วยข้อความแทน code
