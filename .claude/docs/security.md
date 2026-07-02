# Security Checklist

มาตรฐานความปลอดภัยขั้นต่ำที่ทุกฟีเจอร์ต้องผ่าน ใช้เป็น checklist ตอนเขียน (dev) และตอนตรวจ (qa-tester)

---

## Backend (Go + Gin + GORM)

### Authentication / Password
- **ห้ามเก็บรหัสผ่านเป็น plaintext เด็ดขาด** — hash ด้วย `golang.org/x/crypto/bcrypt` (cost ≥ 10)
- เปรียบเทียบรหัสผ่านด้วย `bcrypt.CompareHashAndPassword` เท่านั้น ห้ามเทียบ string ตรง ๆ
- JWT: เก็บ secret ใน `.env` (`JWT_SECRET`), ตั้ง `exp` ทุก token, ตรวจ signing method ตอน verify (กัน `alg: none`)
- ไม่ใส่ข้อมูลอ่อนไหว (รหัสผ่าน, เลขบัตร) ลงใน JWT payload (payload อ่านได้ ไม่ได้เข้ารหัส)

### Input / Injection
- ใช้ GORM ผ่าน method ปกติ (`Where("email = ?", email)`) — **ห้ามต่อ SQL string เอง** ด้วยตัวแปรจาก user
- validate ทุก input ด้วย Gin binding tag (`binding:"required,email"` ฯลฯ)
- จำกัดขนาด body/request ที่รับ (กัน payload ใหญ่ผิดปกติ)

### Secret / Config / Logging
- ห้าม hardcode secret/DB credential — อ่านจาก `.env` เท่านั้น, `.env` ต้อง gitignore
- **ห้าม log ข้อมูลอ่อนไหว** (รหัสผ่าน, token, ข้อมูลส่วนบุคคล) — ก่อน log ให้ตัด field เหล่านี้ออก
- error ที่ส่งกลับ client ห้ามหลุด internal detail (stack trace, SQL, path ไฟล์) → ใช้ response มาตรฐาน (ดู `api-response.md`)

### CORS / Middleware
- ตั้ง CORS ให้ระบุ origin ที่อนุญาตจริง (จาก `.env`) — ห้ามเปิด `*` ใน production
- มี middleware `recovery` กัน panic ทำ server ล่ม และไม่ leak stack ให้ client

---

## Frontend (React)

- เก็บ token ให้สอดคล้องกับทีม (httpOnly cookie ปลอดภัยกว่า `localStorage` ต่อ XSS) — ตกลงวิธีเดียวทั้งโปรเจกต์
- ห้าม hardcode secret/endpoint ที่อ่อนไหวในโค้ด frontend (ทุกอย่างใน bundle มองเห็นได้)
- ระวัง `dangerouslySetInnerHTML` — ถ้าจำเป็นต้อง sanitize ก่อน (กัน XSS)
- แสดง error จาก API เป็นข้อความที่เตรียมไว้ ไม่โยน raw error object ให้ user

---

## qa-tester ตรวจอะไรบ้าง (severity)
- 🔴 รหัสผ่านไม่ได้ hash / secret หลุดใน git / ต่อ SQL string เอง / CORS เปิด `*` ใน prod
- 🟠 log ข้อมูลอ่อนไหว / error หลุด internal detail ให้ client / ไม่ validate input
- 🟡 token เก็บใน localStorage โดยไม่ได้ตกลง / ไม่จำกัดขนาด request
