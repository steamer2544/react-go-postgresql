---
paths:
  - "frontend/**"
---

# Frontend Rules (React)

> กฎนี้เป็น **path-scoped**: โหลดเข้า context เฉพาะตอน Claude แตะไฟล์ใต้ `frontend/`
> (ประหยัด context ตอนทำงานฝั่ง backend) — `naming-conventions.md` ไม่ scope จึงโหลดทุก session

กฎเฉพาะฝั่ง React ใช้ร่วมกับ `naming-conventions.md`

## หลักการเขียนโค้ด
- ใช้ **function component + hooks** เท่านั้น ไม่ใช้ class component
- 1 ไฟล์ = 1 component หลัก (ไฟล์ตั้งชื่อตาม component)
- แยก logic ที่ใช้ซ้ำออกเป็น **custom hook** (`use...`)
- เรียก API ผ่านชั้น `services/` เท่านั้น ห้ามยิง axios ตรงใน component
- ค่าคงที่/endpoint เก็บใน `constants/` หรือ `.env` ห้าม hardcode ในคอมโพเนนต์

## การเรียก API (services + react-query)
แบ่งหน้าที่ให้ชัด อย่าปน:
- **`services/`** = ฟังก์ชันยิง axios ล้วน ๆ (เช่น `employeeService.getAll()`) คืน data ดิบ
- **`hooks/`** = custom hook ครอบด้วย react-query (`useQuery`/`useMutation`) เรียก service ข้างใน จัดการ cache/loading/error
- **component** = เรียก hook เท่านั้น ไม่แตะ axios/service ตรง
- interceptor กลางใน `services/apiClient.js` แกะ response ตาม `.claude/docs/api-response.md` (อ่าน `data`/`error` มาตรฐาน)

## Styling
- ใช้ **Bootstrap 5** เป็นฐาน แล้วเสริมด้วย Custom CSS
- Custom CSS class ตั้งชื่อ **kebab-case** (`employee-card`)
- ไฟล์ css วางคู่ component หรือรวมใน `styles/` ตามขนาดงาน
- Icon ใช้ **Font Awesome** ผ่าน `@fortawesome/react-fontawesome`

## ไลบรารีที่กำหนด (ห้ามใช้ตัวอื่นแทนโดยไม่ตกลงกับทีม)
- Table → **TanStack Table** (`@tanstack/react-table`)
- Dropdown → **react-select**
- Datepicker → **react-datepicker**
- Test → **Vitest + React Testing Library**
- รายการเต็ม → `.claude/docs/standard-libraries.md`

## Security (สรุป — เต็มใน `.claude/docs/security.md`)
- ห้าม hardcode secret ในโค้ด frontend (มองเห็นได้ใน bundle)
- ตกลงวิธีเก็บ token วิธีเดียวทั้งทีม; ระวัง `dangerouslySetInnerHTML` (sanitize ก่อน)

## โครงสร้างไฟล์
ดู `.claude/docs/frontend-structure.md`

## ตรวจก่อน commit
```bash
cd frontend && npm run lint && npm run build
```
