# Frontend Folder Structure (React + Vite)

โครงสร้างแบบ **feature-oriented** ผสม type-based ยืดหยุ่น ขยายง่ายเมื่อโปรเจกต์โต

```
frontend/
├── public/                     # static ที่ไม่ผ่าน build (favicon ฯลฯ)
├── src/
│   ├── assets/                 # รูป, ฟอนต์, ไฟล์ static ที่ import ในโค้ด
│   │   ├── images/
│   │   └── styles/             # custom CSS ระดับ global
│   │
│   ├── components/             # UI component ที่ใช้ซ้ำทั่วแอป (PascalCase)
│   │   ├── common/             # Button, Modal, Loading, Pagination ...
│   │   └── layout/             # Header, Sidebar, Footer, MainLayout
│   │
│   ├── features/               # แยกตามฟีเจอร์/โดเมน
│   │   └── employee/
│   │       ├── components/     # component เฉพาะฟีเจอร์นี้ (EmployeeList.jsx)
│   │       ├── hooks/          # useEmployee.js (ครอบ react-query)
│   │       ├── services/       # employeeService.js (ยิง axios ล้วน)
│   │       └── pages/          # EmployeeListPage.jsx
│   │
│   ├── pages/                  # หน้า route-level ที่ไม่ผูกฟีเจอร์เดียว (HomePage.jsx)
│   ├── hooks/                  # custom hook ที่ใช้ร่วม (useAuth.js, useDebounce.js)
│   ├── services/               # ตั้งค่า axios instance + service ที่ใช้ร่วม (apiClient.js)
│   ├── contexts/               # React Context (AuthContext.jsx)
│   ├── utils/                  # ฟังก์ชันช่วย (formatDate.js, validators.js)
│   ├── constants/              # ค่าคงที่ (apiEndpoints.js, appConfig.js)
│   ├── routes/                 # กำหนด routing (AppRoutes.jsx)
│   │
│   ├── App.jsx                 # root component
│   └── main.jsx                # entry point (mount React)
│
├── .env                        # ตัวแปร environment (VITE_API_URL ฯลฯ)
├── .env.example                # ตัวอย่าง env สำหรับ commit
├── .eslintrc.cjs               # config ESLint
├── .prettierrc                 # config Prettier
├── index.html                  # HTML template ของ Vite
├── package.json
├── vite.config.js
└── vitest.config.js            # (หรือรวมใน vite.config.js) — config test
```

## หลักการวางไฟล์

- **component ที่ใช้เฉพาะฟีเจอร์** → อยู่ใต้ `features/<feature>/components/`
- **component ที่ใช้ทั่วแอป** → อยู่ใน `components/common/`
- **ไฟล์ component** ตั้งชื่อ PascalCase + `.jsx` (`EmployeeList.jsx`)
- **ไฟล์ที่ไม่ใช่ component** (hook, service, util) ตั้งชื่อ camelCase (`useAuth.js`)
- ค่า config/endpoint อยู่ใน `constants/` หรือ `.env` (ขึ้นต้น `VITE_` ถึงจะเข้าถึงได้ใน Vite)

## การไหลของการเรียก API (3 ชั้น)

แยกให้ชัด อย่าปนกัน:

```
component  ── เรียก ──>  hook (react-query)  ── เรียก ──>  service (axios)  ── HTTP ──>  backend
```

- **service** (`employeeService.js`): ฟังก์ชันยิง axios ล้วน คืน data ดิบ ไม่รู้จัก react-query
- **hook** (`useEmployee.js`): ครอบ service ด้วย `useQuery`/`useMutation` จัดการ cache/loading/error
- **component**: เรียก hook เท่านั้น ไม่ import axios/service ตรง
- axios instance กลางจาก `services/apiClient.js` ตั้ง interceptor แกะ response ตาม `.claude/docs/api-response.md`

## ตัวอย่าง import path (แนะนำตั้ง alias `@` = `src`)
```js
import { Button } from '@/components/common/Button';
import { useAuth } from '@/hooks/useAuth';
import employeeService from '@/features/employee/services/employeeService';
```

## ไฟล์ test
- วางคู่ไฟล์ที่ทดสอบ (`EmployeeList.test.jsx`) หรือรวมใน `__tests__/` ต่อฟีเจอร์
- ใช้ Vitest + React Testing Library (ดู `standard-libraries.md`)
