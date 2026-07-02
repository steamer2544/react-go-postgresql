# API Response Contract

รูปแบบ JSON response มาตรฐานของทั้งโปรเจกต์ **ทุก endpoint ต้องตอบตามนี้**
เพื่อให้ frontend เขียน handler/interceptor ได้ที่เดียว helper อยู่ที่ `pkg/response/`

---

## 1. Success

```json
{
  "data": { "id": 1, "first_name": "Somchai" },
  "message": "created successfully"
}
```

- `data` — payload หลัก (object หรือ array); ถ้าไม่มีข้อมูลส่งกลับให้เป็น `null`
- `message` — ข้อความสั้น ๆ (optional สำหรับ GET, ควรมีสำหรับ create/update/delete)

## 2. Error

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "email is required",
    "details": [
      { "field": "email", "issue": "required" }
    ]
  }
}
```

- `error.code` — โค้ดคงที่แบบ `UPPER_SNAKE` ให้ frontend เทียบได้ (ไม่ผูกกับข้อความ)
- `error.message` — ข้อความอ่านรู้เรื่อง (ปลอดภัยพอที่จะโชว์ให้ user); **ห้ามหลุด stack/SQL/path**
- `error.details` — optional; ใช้กับ validation หลาย field

## 3. List + Pagination

```json
{
  "data": [ { "id": 1 }, { "id": 2 } ],
  "meta": { "page": 1, "page_size": 20, "total": 137 }
}
```

---

## HTTP status ที่ใช้
| สถานะ | ใช้เมื่อ |
| --- | --- |
| 200 | สำเร็จ (GET/PUT/PATCH) |
| 201 | สร้างสำเร็จ (POST) |
| 204 | สำเร็จ ไม่มี body (DELETE บางกรณี) |
| 400 | input ผิด / validation fail (`VALIDATION_ERROR`) |
| 401 | ไม่ได้ login / token ไม่ถูก (`UNAUTHORIZED`) |
| 403 | login แล้วแต่ไม่มีสิทธิ์ (`FORBIDDEN`) |
| 404 | ไม่พบข้อมูล (`NOT_FOUND`) |
| 409 | ชนกับข้อมูลเดิม เช่น email ซ้ำ (`CONFLICT`) |
| 500 | error ภายใน (`INTERNAL_ERROR`) — ห้าม leak รายละเอียด |

---

## ฝั่ง Frontend
- axios instance กลาง (`services/apiClient.js`) response interceptor:
  - success → คืน `res.data.data` ให้ layer บน
  - error → อ่าน `error.response.data.error` มาตรฐานนี้ แล้วโยนเป็น error ที่มี `code` + `message`
- เทียบด้วย `error.code` (ไม่เทียบข้อความ) เพื่อรองรับ i18n/เปลี่ยนข้อความภายหลัง
