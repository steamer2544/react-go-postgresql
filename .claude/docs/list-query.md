# List Query Contract

สัญญากลางของ **query parameter สำหรับ endpoint แบบ list** (GET หลายรายการ)
ให้ทุกหน้า list ใช้รูปแบบเดียวกัน ต่อยอดจาก `api-response.md` (envelope `data` + `meta`)

---

## 1. พารามิเตอร์มาตรฐาน

| กลุ่ม | param | ตัวอย่าง | ความหมาย |
| --- | --- | --- | --- |
| Pagination | `page` | `page=2` | หน้า (เริ่มที่ 1) |
| | `page_size` | `page_size=20` | จำนวนต่อหน้า |
| Sort | `sort` | `sort=-created_at,last_name` | เรียงหลาย field, `-` นำหน้า = จากมากไปน้อย (desc) |
| Filter (เท่ากับ) | `<field>` | `status=active&department_id=3` | กรองแบบ exact |
| Filter (ช่วง) | `<field>_gte` / `<field>_lte` | `created_at_gte=2026-01-01` | ช่วงค่า (วันที่/ตัวเลข) |
| ค้นหา | `q` | `q=somchai` | free-text search ตาม field ที่กำหนด |

ตัวอย่างเต็ม:
```
GET /employees?page=1&page_size=20&sort=-created_at&status=active&q=som
```

---

## 2. ค่าเริ่มต้น + เพดาน (บังคับ กัน abuse)

| param | default | เพดาน / กติกา |
| --- | --- | --- |
| `page` | 1 | ต้อง ≥ 1 (ผิด → 400 `VALIDATION_ERROR`) |
| `page_size` | 20 | **สูงสุด 100** (ขอเกินให้ cap ที่ 100 หรือ 400 ตามที่ทีมตกลง) |
| `sort` | ตามแต่ละ resource (เช่น `-created_at`) | เฉพาะ field ใน whitelist |

---

## 3. Whitelisting (สำคัญด้านความปลอดภัย)

- **อนุญาต sort/filter เฉพาะ field ที่กำหนดไว้ล่วงหน้า** ต่อ resource
- field นอก whitelist → ปฏิเสธ 400 `VALIDATION_ERROR` (อย่าเงียบแล้วข้าม)
- เหตุผล: กัน SQL injection ทางชื่อคอลัมน์ และกันเดา/หลุดโครงสร้างตาราง — สอดคล้อง `security.md`

---

## 4. Response (ตาม api-response.md)

```json
{
  "data": [ { "id": 1 }, { "id": 2 } ],
  "meta": { "page": 1, "page_size": 20, "total": 137 }
}
```
- `meta.total` = จำนวนทั้งหมด**หลังกรอง** (ก่อนแบ่งหน้า) — FE เอาไปคำนวณจำนวนหน้า

---

## 5. Backend — วิธี implement (Go + GORM)

- parse + validate query เป็น **DTO** ที่ handler (เช่น `ListEmployeeQuery`) → ส่งเข้า service → repository
- repository ประกอบ query ด้วย **GORM Scopes** (แยก pagination/sort/filter เป็น scope ใช้ซ้ำได้)
- ค่าจาก user เข้าผ่าน **parameterized** เสมอ (`Where("status = ?", v)`) — sort field มาจาก whitelist map ไม่ต่อ string ตรง

```go
type ListEmployeeQuery struct {
    Page     int    `form:"page,default=1"    binding:"min=1"`
    PageSize int    `form:"page_size,default=20" binding:"min=1,max=100"`
    Sort     string `form:"sort"`
    Status   string `form:"status" binding:"omitempty,oneof=active inactive"`
    Q        string `form:"q"`
}
```

---

## 6. Frontend — วิธีใช้

- ส่ง param ผ่านชั้น `services/` (axios) เหมือนเดิม
- **ใส่ param ลงใน react-query key** เพื่อ cache แยกตามเงื่อนไข:
  ```js
  useQuery({ queryKey: ['employees', { page, pageSize, sort, status, q }], queryFn: ... })
  ```
- ตาราง (TanStack Table) ส่ง state sort/pagination กลับมาเป็น param เหล่านี้

---

## 7. Checklist ที่ qa-tester ตรวจ

- 🔴 sort/filter รับ field อิสระโดยไม่ whitelist (เสี่ยง injection)
- 🟠 ไม่มีเพดาน `page_size` (ขอ 100000 ได้) / `meta.total` ไม่ตรงหลังกรอง
- 🟡 param ผิดแล้วเงียบแทนที่จะ 400 / react-query key ไม่รวม param (cache ผิดหน้า)
