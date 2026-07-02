# Naming Conventions

กฎการตั้งชื่อ บังคับใช้ทั้งโปรเจกต์ ห้ามข้าม

---

## Frontend (React)

| ประเภท | รูปแบบ | ตัวอย่าง |
| --- | --- | --- |
| Variable | camelCase | `userName` |
| Function | camelCase | `getEmployee()` |
| State | camelCase | `loading` |
| State Setter | `set` + PascalCase | `setLoading` |
| Component | PascalCase | `EmployeeList` |
| File Component | PascalCase + `.jsx` | `EmployeeList.jsx` |
| Custom Hook | `use` + PascalCase | `useAuth()` |
| Constant (ทั่วไป) | camelCase | `apiUrl` |
| Constant (ระดับแอป) | UPPER_SNAKE_CASE | `MAX_RETRY_COUNT` |
| CSS Class | kebab-case | `employee-card` |

เพิ่มเติม:
- ไฟล์ที่ **ไม่ใช่** component (hook, util, service) ตั้งชื่อ camelCase: `useAuth.js`, `formatDate.js`, `employeeService.js`
- โฟลเดอร์ทั่วไปใช้ kebab-case หรือ camelCase ให้สม่ำเสมอทั้งโปรเจกต์
- Boolean state/ตัวแปร ควรขึ้นต้นด้วย `is`, `has`, `should`: `isLoading`, `hasError`
- Event handler ขึ้นต้นด้วย `handle`: `handleSubmit`, `handleChange`

---

## Backend (Go)

Go กำหนด "การมองเห็น" (visibility) ด้วยตัวพิมพ์ตัวแรก จึงต้องแยกให้ชัด:

| ประเภท | รูปแบบ | ตัวอย่าง |
| --- | --- | --- |
| ตัวแปร local / unexported | camelCase | `userName`, `employeeID` |
| Identifier ที่ export (public) | PascalCase (บังคับโดยภาษา) | `GetEmployee`, `UserService` |
| Struct field ที่ต้อง serialize | PascalCase + json tag snake_case | `FirstName \`json:"first_name"\`` |
| Constant | MixedCaps (ไม่ใช้ UPPER_SNAKE) | `MaxRetryCount`, `defaultTimeout` |
| Package | ตัวเล็กคำเดียว ไม่มี `_` | `handler`, `repository` |
| Interface | PascalCase มักลงท้าย `-er` | `Repository`, `Reader` |
| ไฟล์ | snake_case | `employee_handler.go` |

> **หมายเหตุสำคัญ:** โจทย์กำหนดตัวแปร backend เป็น camelCase ซึ่งใช้ได้กับตัวแปร
> local/unexported เท่านั้น ส่วนอะไรที่ต้อง export ให้ package อื่นเห็น (handler, service,
> struct ที่ GORM/Gin ใช้) **จำเป็นต้องเป็น PascalCase** ตามกฎภาษา Go มิฉะนั้นจะ compile
> ไม่ผ่านหรือ bind ข้อมูลไม่ได้

Acronym ให้คงรูปตัวพิมพ์เดียวกันทั้งก้อน: `userID` ✅ ไม่ใช่ `userId`, `HTTPServer` ✅ ไม่ใช่ `HttpServer`
