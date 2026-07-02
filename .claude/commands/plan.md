---
description: วางแผนฟีเจอร์อย่างเดียว (planner) แล้วหยุดให้ human review ก่อนลงมือเขียน test/โค้ด
argument-hint: <คำอธิบายฟีเจอร์>
disable-model-invocation: true
model: inherit
---

# /plan — planner อย่างเดียว

โจทย์: **$ARGUMENTS**

delegate → subagent `planner` เพื่อแตกงาน + acceptance criteria แล้วเขียน `docs/plans/<slug>.md`
- ตั้ง `<slug>` แบบ kebab-case จากโจทย์ แล้ว**แจ้ง slug ที่ใช้**
- **หยุดหลังได้ plan** — ยังไม่เรียก test-case-writer / dev
- ตอบผู้ใช้ด้วย path ของ plan + สรุปสั้น ๆ + คำถามค้าง (ถ้ามี)

> พอใจแผนแล้วรันต่อทั้งสายด้วย `/feature <slug>` ได้เลย (จะ resume ข้ามเฟส planner ให้อัตโนมัติ)
