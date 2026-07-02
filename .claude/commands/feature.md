---
description: รันสายพาน TDD เต็ม (planner → test-case-writer → dev → qa-tester) ของ 1 ฟีเจอร์ จนกว่า QA จะ PASS — resume ข้ามเฟสที่ทำเสร็จแล้วได้
argument-hint: <slug หรือ คำอธิบายฟีเจอร์>
disable-model-invocation: true
model: inherit
---

# /feature — orchestrate สายพาน TDD ของ 1 ฟีเจอร์

โจทย์ฟีเจอร์ / slug: **$ARGUMENTS**

คุณคือ **main agent** ทำหน้าที่ orchestrate subagent ตามลำดับ TDD ด้านล่าง
**ห้ามเขียน plan / test / โค้ด เอง** — ทุกเฟสต้อง **delegate ให้ subagent เจ้าของเฟส** (ผ่าน Task):
`planner` → `test-case-writer` → `dev` → `qa-tester` แล้วส่งต่อ output ระหว่างกัน

---

## 0. ตั้ง slug + ตรวจ artifact ที่มีอยู่ (เพื่อ resume)
- ถ้า `$ARGUMENTS` เป็น kebab-case คำเดียว → ถือเป็น `<slug>`
- ถ้าเป็นประโยคอธิบาย → ตั้ง `<slug>` แบบ kebab-case เอง (เช่น `employee-crud`) แล้ว**แจ้ง slug ที่ใช้**
- เช็คไฟล์ที่มีอยู่แล้ว **ข้ามเฟสที่ทำเสร็จแล้ว** (resume) เว้นแต่ผู้ใช้พิมพ์ `--fresh`:
  - `docs/plans/<slug>.md`            → มี = ข้ามเฟส planner
  - `docs/tests/<slug>-testcases.md`  → มี = ข้ามเฟส test-case-writer

## 1. planner  (ถ้ายังไม่มี plan)
delegate → `planner` : "วางแผนฟีเจอร์ <slug>: <โจทย์>"
รับ path ของ `docs/plans/<slug>.md` + สรุป

## 2. test-case-writer — RED  (ถ้ายังไม่มี test case)
delegate → `test-case-writer` : "อ่าน docs/plans/<slug>.md แล้วเขียน test case + failing test (RED) ของ <slug>"
ต้องได้ `docs/tests/<slug>-testcases.md` + ไฟล์ test + **ยืนยันผลรัน RED**
> ถ้ารายงานว่า "ผ่านตั้งแต่ยังไม่ implement" = RED ไม่จริง → สั่งแก้ให้ fail ด้วยเหตุผลที่ถูกก่อนไปต่อ

## 3. dev — GREEN → Refactor
delegate → `dev` : "implement <slug> ให้ test (RED) ผ่านครบ ตาม docs/plans/<slug>.md + docs/tests/<slug>-testcases.md"
รับสรุปไฟล์ที่แก้ + ผลรัน test

## 4. qa-tester — VERIFY  (วนจน PASS, สูงสุด 3 รอบ)
delegate → `qa-tester` : "verify <slug> เทียบ test case + AC แล้วเขียนรายงาน (รอบที่ N)"
qa-tester ต้องเขียน `docs/reports/<slug>-qa.md` และตอบ PASS/FAIL
- **PASS** → จบ ไปที่ "เอาต์พุตสุดท้าย"
- **FAIL** → ส่ง "สิ่งที่ต้องให้ dev แก้" จากรายงานกลับให้ `dev` แก้ แล้ววน qa-tester รอบถัดไป
- ทำซ้ำ dev↔qa **ไม่เกิน 3 รอบ** — ถ้าครบ 3 รอบยัง FAIL ให้ **หยุด** สรุปสิ่งที่ยังค้าง + ทางเลือก แล้ว**ถามผู้ใช้**
> อย่าวนไม่รู้จบ และอย่าแก้ test ให้ผ่านแบบผิดเจตนาเพื่อให้จบเร็ว

---

## เอาต์พุตสุดท้าย (ตอบผู้ใช้)
- slug ที่ใช้ + สถานะ (**PASS** / **FAIL หลัง 3 รอบ**)
- path: `docs/plans/<slug>.md` · `docs/tests/<slug>-testcases.md` · `docs/reports/<slug>-qa.md`
- สรุปผล: test ผ่าน X/Y | AC ครบ X/Y | ปัญหาที่ยังค้าง (ถ้ามี)
