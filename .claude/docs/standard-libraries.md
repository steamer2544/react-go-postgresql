# Standard Libraries

รายชื่อไลบรารีมาตรฐานที่ทีมกำหนดให้ใช้ เป็นตัวที่ใช้กันสากล
**ห้ามใช้ตัวอื่นแทนโดยไม่ตกลงกับทีมก่อน** เพื่อลดความหลากหลายที่ดูแลยาก

---

## Frontend (React)

| หน้าที่ | แพ็กเกจ (npm) | หมายเหตุ |
| --- | --- | --- |
| Library หลัก | `react`, `react-dom` | function component + hooks |
| Build tool | `vite`, `@vitejs/plugin-react` | ไฟล์ component เป็น `.jsx` |
| Stylesheet | `bootstrap` (+ Custom CSS) | Bootstrap 5 เป็นฐาน |
| Table | `@tanstack/react-table` | headless table |
| Icon / Font | `@fortawesome/fontawesome-svg-core`, `@fortawesome/free-solid-svg-icons`, `@fortawesome/react-fontawesome` | Font Awesome |
| Dropdown | `react-select` | dropdown ค้นหา/multi-select |
| Datepicker | `react-datepicker` | เลือกวันที่ |
| Routing | `react-router-dom` | มาตรฐาน routing ของ React |
| HTTP client | `axios` | เรียก API, interceptor สะดวก |
| Data fetching / cache | `@tanstack/react-query` | จัดการ server state, cache, loading |
| Form + validation | `react-hook-form` + `zod` (หรือ `yup`) | ฟอร์มมีประสิทธิภาพ + schema validation |
| Date utility | `dayjs` (หรือ `date-fns`) | จัดการวันที่ เบากว่า moment |
| Notification | `react-toastify` | toast แจ้งเตือน |
| Lint / Format | `eslint`, `prettier` | คุมสไตล์โค้ด |
| **Testing** | `vitest`, `@testing-library/react`, `@testing-library/jest-dom`, `@testing-library/user-event`, `jsdom`, `msw` | unit/component test + mock HTTP ที่ขอบด้วย MSW (test-case-writer วางโครง, dev ทำให้ผ่าน — ดู `testing.md`) |

> รัน test frontend: `npm test` (ตั้ง `"test": "vitest run"` ใน `package.json`)
> `jsdom` เป็น environment ของ Vitest; `@testing-library/jest-dom` เพิ่ม matcher เช่น `toBeInTheDocument`

---

## Backend (Go)

| หน้าที่ | โมดูล | หมายเหตุ |
| --- | --- | --- |
| Web framework | `github.com/gin-gonic/gin` | framework หลัก |
| ORM | `gorm.io/gorm` | ORM หลัก |
| PostgreSQL driver | `gorm.io/driver/postgres` | driver สำหรับ GORM |
| Config / .env | `github.com/spf13/viper` หรือ `github.com/joho/godotenv` | อ่าน env/config |
| Validation | `github.com/go-playground/validator/v10` | มากับ Gin binding อยู่แล้ว |
| Auth (JWT) | `github.com/golang-jwt/jwt/v5` | ออก/ตรวจ token |
| **Password hashing** | `golang.org/x/crypto/bcrypt` | hash รหัสผ่าน (cost ≥ 10) — บังคับเมื่อมี auth |
| CORS | `github.com/gin-contrib/cors` | middleware CORS ของ Gin |
| Logger | `go.uber.org/zap` (หรือ `github.com/sirupsen/logrus`) | structured logging |
| UUID | `github.com/google/uuid` | สร้าง UUID |
| Migration | `github.com/golang-migrate/migrate/v4` | จัดการ schema (ดูนโยบาย migration ด้านล่าง) |
| Testing | `github.com/stretchr/testify` | assert/require/mock สำหรับ unit test |
| Mock generation (ถ้าใช้) | `github.com/vektra/mockery/v2` | gen mock จาก interface (เลือกทางเดียวกับ manual mock — ดู `testing.md`) |
| Integration test DB | `github.com/testcontainers/testcontainers-go` | สปิน Postgres จริงตอนเทส repository (ดู `testing.md`) |

---

## นโยบาย Migration (เลือกทางเดียวต่อ environment)

เพื่อกัน schema drift ให้ยึดกฎนี้ **ไม่ผสมสองวิธีใน environment เดียวกัน**:

| Environment | วิธี | หมายเหตุ |
| --- | --- | --- |
| dev (เครื่อง developer) | `GORM AutoMigrate` ได้ | เร็ว สะดวกตอน prototype |
| staging / production | `golang-migrate` เท่านั้น | ไฟล์ใน `migrations/` = **source of truth** ของ schema |

> ทุกการเปลี่ยน schema ที่จะขึ้น prod ต้องมีไฟล์ migration คู่ (`*.up.sql` / `*.down.sql`)
> ห้ามพึ่ง AutoMigrate บน prod เพราะควบคุมลำดับ/rollback ไม่ได้

---

## หมายเหตุเรื่องเวอร์ชัน

> เอกสารนี้ระบุ **ชื่อแพ็กเกจ** ที่เป็นมาตรฐาน ส่วนเวอร์ชันที่ล็อกจริงให้ยึดตาม
> `package.json` (frontend) และ `go.mod` (backend) เป็นแหล่งความจริง แนะนำ pin เวอร์ชัน
> และอัปเดตพร้อมกันทั้งทีมเป็นรอบ ๆ ไม่อัปเดตกลาง sprint
