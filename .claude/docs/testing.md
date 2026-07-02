# Testing Strategy

มาตรฐานการเขียน test ของทั้งโปรเจกต์ ใช้เป็นสัญญากลางให้ **test-case-writer** (วางโครง RED),
**dev** (ทำให้เขียว), และ **qa-tester** (verify) ทำงานต่อกันได้โดยไม่ต้องเดา

> วางไว้ใน `.claude/docs/` เพราะเป็น "มาตรฐานคงที่" (โหลดเมื่อต้องใช้) —
> ส่วน **test case ต่อฟีเจอร์** ที่ agent ผลิต ไปที่ root `docs/tests/<slug>-testcases.md`

---

## 1. Test pyramid (สัดส่วนที่อยากได้)

```
        /\        E2E / integration ผ่าน HTTP (น้อย, ช้า, ของจริง)
       /  \
      /----\      Integration: repository เทียบ DB จริง (ปานกลาง)
     /------\
    /--------\    Unit: service + logic ด้วย mock (เยอะสุด, เร็ว)
```

- น้ำหนักหลักอยู่ที่ **unit test ของ service** (business logic) — เร็ว, deterministic
- **integration test ของ repository** ยืนยันว่า GORM query/constraint ทำงานกับ Postgres จริง
- E2E เท่าที่จำเป็น (happy path สำคัญ) อย่าเยอะจนช้า

---

## 2. Backend — Unit test (service) ด้วย mock

**หลักการพึ่งพา:** service ต้องพึ่ง **interface ของ repository** ไม่ใช่ struct จริง เพื่อ mock ได้

```go
// repository ประกาศ interface (ให้ service พึ่ง interface นี้)
type EmployeeRepository interface {
    Create(ctx context.Context, e *model.Employee) error
    FindByEmail(ctx context.Context, email string) (*model.Employee, error)
}
```

**เครื่องมือ mock — เลือกทางเดียวทั้งทีม:**

| วิธี | เมื่อไหร่ | หมายเหตุ |
| --- | --- | --- |
| **`testify/mock` เขียนเอง** (ค่าเริ่มต้น) | interface เล็ก, จำนวนน้อย | ไม่ต้องพึ่ง codegen เห็นทุกอย่างชัด |
| **`mockery` generate** | interface เยอะ/เปลี่ยนบ่อย | มี `//go:generate` + commit mock ที่ gen แล้ว |

> **ห้ามผสมสองวิธีมั่ว** — ถ้าเริ่ม mockery ให้ใช้ทั้งโปรเจกต์ ตกลงกันก่อน

ตัวอย่าง unit test (testify) — ตรงกับที่ test-case-writer วางไว้:
```go
func TestCreateEmployee_TC03_DuplicateEmail(t *testing.T) {
    // Arrange
    repo := new(mocks.EmployeeRepository)
    repo.On("FindByEmail", mock.Anything, "s@ex.com").
        Return(&model.Employee{ID: 1}, nil) // มี email นี้อยู่แล้ว
    svc := NewEmployeeService(repo)
    // Act
    _, err := svc.CreateEmployee(context.Background(), dto.CreateEmployeeRequest{Email: "s@ex.com"})
    // Assert (คาด CONFLICT ตาม api-response.md)
    require.ErrorIs(t, err, ErrEmailConflict)
}
```

---

## 3. Backend — Integration test (repository เทียบ DB จริง)

repository แตะ SQL/constraint จริง → **อย่า mock DB** ให้เทสกับ Postgres จริง เลือกทางเดียว:

| วิธี | ข้อดี | หมายเหตุ |
| --- | --- | --- |
| **`testcontainers-go`** (แนะนำ) | สปิน Postgres ใน container ต่อ run, สะอาด, ใกล้ prod | ต้องมี Docker บนเครื่อง/CI |
| DB ทดสอบแยก + `golang-migrate` | เร็วกว่า ไม่ต้อง Docker | ต้องดูแล DB test แยกเอง |

**ความสะอาดของ state ต่อ test:** ห่อแต่ละ test ด้วย **transaction แล้ว rollback** (หรือ truncate ก่อนเทส) เพื่อให้ test **independent** — ไม่พึ่งลำดับ/ข้อมูลค้างจาก test อื่น

```go
func TestEmployeeRepo_Create_Integration(t *testing.T) {
    if testing.Short() { t.Skip("ข้าม integration ใน -short") } // แยกจาก unit ได้
    db := testDB(t)          // เชื่อม container, migrate, คืน *gorm.DB
    tx := db.Begin(); defer tx.Rollback()
    repo := NewEmployeeRepository(tx)
    // ... act + assert กับ DB จริง
}
```
> ใช้ `go test -short ./...` รันเฉพาะ unit (เร็ว) ตอน dev; CI รันเต็ม

---

## 4. Frontend — Vitest + React Testing Library

| ชั้น | ทดสอบอะไร | วิธี |
| --- | --- | --- |
| **component** | render ถูก, event ทำงาน | render พร้อม props, `screen.findBy*`, `userEvent` |
| **hook (react-query)** | loading/error/data | ครอบด้วย `QueryClientProvider` wrapper แล้ว `renderHook` |
| **service (axios)** | ยิง/แกะ response ถูก | mock HTTP ที่ขอบด้วย **MSW** (mock service worker) |

- **mock ที่ขอบ HTTP ด้วย MSW** ดีกว่า mock `axios` ตรง ๆ (ทดสอบใกล้ของจริง, ไม่ผูก implementation)
- ทดสอบ **behavior ที่ผู้ใช้เห็น** (ข้อความ, ปุ่ม, บทบาท) ไม่ query ด้วย class/id ภายใน
- ทุก test สร้าง `QueryClient` ใหม่ (`retry: false`) กัน cache รั่วข้าม test

---

## 5. กฎร่วม (ทั้งสองฝั่ง)

- **AAA**: Arrange → Act → Assert — 1 test = 1 behavior
- **deterministic**: อะไรพึ่งเวลา/สุ่ม/uuid ให้ **inject/mock** (เช่น รับ `clock`/`idGen` เป็น dependency)
- **independent**: แต่ละ test ตั้ง state เองครบ ไม่พึ่ง test อื่น
- ผูกชื่อ test กับ TC id: `TestX_TC01_Happy`, `it('TC-02 edge: ...')`
- assert **ค่าจริงจาก AC** ไม่ใช่ `assert.True(t, true)`
- **ไม่ทดสอบ**: framework ภายใน, getter/setter ล้วน, สิ่งที่ไม่มี logic

---

## 6. Coverage (เป็นแนวทาง ไม่ใช่ตัวเลขบูชา)

| ชั้น | เป้าหมายคร่าว ๆ |
| --- | --- |
| service (business logic) | ≥ 80% |
| handler / repository | cover happy + error หลัก ๆ |
| util ที่มี logic | สูง (คุ้มค่า) |

> อย่าไล่ 100% จนเขียน test ไร้ค่า — เน้น **สาขาการตัดสินใจ (branch)** และ error path มากกว่าตัวเลขรวม

---

## 7. คำสั่งรัน (ตรงกับ verify commands ใน CLAUDE.md)

```bash
# backend
cd backend && go test -short ./...        # unit เร็ว (ตอน dev)
cd backend && go test ./...               # เต็ม รวม integration (CI)

# frontend
cd frontend && npm test                   # vitest run
```

ไลบรารีที่ใช้ → `standard-libraries.md` (testify, mockery, testcontainers-go / vitest, RTL, MSW)
