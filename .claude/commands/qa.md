---
description: รัน qa-tester verify ฟีเจอร์ตาม slug แล้วเขียนรายงาน docs/reports/<slug>-qa.md (verify อย่างเดียว ไม่แก้โค้ด)
argument-hint: <slug>
disable-model-invocation: true
model: inherit
---

# /qa — verify อย่างเดียว

slug: **$ARGUMENTS**

delegate → subagent `qa-tester` เพื่อ verify ฟีเจอร์ `<slug>`:
- อ่าน `docs/tests/<slug>-testcases.md` + `docs/plans/<slug>.md`
- รัน test ทั้งสองฝั่ง + เช็คทีละ TC + AC + review ตาม checklist ในแต่ละ `.claude/docs/`
- **เขียนรายงานเป็นไฟล์** `docs/reports/<slug>-qa.md` แล้วตอบ **PASS/FAIL** กลับ
- **ไม่แก้ source** — ถ้า FAIL ให้ระบุปัญหาให้ชัด (path + บรรทัด) เพื่อส่งต่อให้ `/feature <slug>` หรือ dev แก้

> ใช้ตอนแก้ไฟล์ด้วยมือแล้วอยาก re-verify หรืออยากตรวจซ้ำเฉพาะ QA โดยไม่รันทั้งสายพาน
