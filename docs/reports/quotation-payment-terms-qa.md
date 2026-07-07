# QA Report: Payment Terms (slug: quotation-payment-terms)

อ้างอิง: docs/plans/quotation-payment-terms.md · docs/tests/quotation-payment-terms-testcases.md
รอบที่: 1 | วันที่: 2026-07-08 | verify โดย: main agent/reviewer (self-verify + targeted review)

## ผลรวม: PASS

รันจริงยืนยัน (ไม่เชื่อคำรายงาน dev อย่างเดียว):

```
BACKEND : gofmt สะอาด · go vet ผ่าน · go build ผ่าน · go test -short → 101 PASS / 7 SKIP / 0 FAIL
FRONTEND: npm run lint 0 error · npm test → 34/34 PASS (13 files) · npm run build สำเร็จ
```

- ไม่มี regression: baseline เดิม (backend 87, frontend 26) ยังเขียวครบ + เพิ่ม payment-terms (backend +14, frontend +8)
- 7 SKIP = integration test ที่ต้องใช้ Docker (TC-REPO-PT-01..03 + TC-REPO ของ quotation-crud เดิม) — เครื่อง dev ไม่มี Docker, ต้องรันใน CI

## AC ครบ 8/8 (AC1–AC8)

| AC      | เรื่อง                                                                            | ผล                         |
| ------- | --------------------------------------------------------------------------------- | -------------------------- |
| AC1     | 3 งวดผลรวม = total (891.67+891.67+891.66=2675.00)                                 | ✅ TC-SVC/CALC-PT          |
| AC2     | float precision 33.33+33.33+33.34=100.00 ผ่านด้วย cents-compare                   | ✅                         |
| AC3     | ผลรวมงวด ≠ total (3000≠2675) → 400 VALIDATION_ERROR                               | ✅                         |
| AC4     | amount ≤ 0 → 400 (binding gt=0)                                                   | ✅ TC-HDL-PT               |
| AC5     | ไม่ส่ง payment_terms → ผ่านปกติ (optional)                                        | ✅                         |
| AC6     | non-draft แก้งวดไม่ได้ → 403 FORBIDDEN                                            | ✅ (reuse draft-only เดิม) |
| AC7/AC8 | frontend: warning + disable submit เมื่อ mismatch, disable inputs เมื่อ non-draft | ✅ TC-FE-FORM-PT           |

## Targeted code review (จุดเสี่ยงสูง — reviewer อ่าน diff จริง)

- **`validatePaymentTermsCents`** (quotation_calc.go): ใช้ **integer satang** (`roundHalfUpCents` ต่องวดแล้ว sum เทียบ int64) ✅ ไม่เทียบ float ตรง ๆ — empty→nil (optional), mismatch→ErrValidation
- **`buildPaymentTerms`** (quotation_service.go): `term_no`/`sort_order` = index+1 server-derived ไม่รับจาก client ✅ (Decision #2)
- **`repository.Update` full-replace**: delete+insert ทั้ง items และ payment_terms ใน transaction เดียว, `Omit("Items","PaymentTerms","CreatedAt")` ✅ atomic, mirror pattern items เป๊ะ
- draft-only + ownership check อยู่ก่อนแตะ payload — ครอบ payment terms อัตโนมัติ ✅

## Bug ที่ dev จับ+แก้เอง (บันทึกไว้)

- `QuotationFormPage.jsx` `totals`/`paymentTermsMismatch` `useMemo` เดิมพึ่ง `watch()` array **by reference** ซึ่ง react-hook-form คืน reference เดิมได้แม้เนื้อในเปลี่ยน → memo คืนค่าค้าง (latent bug ตั้งแต่ quotation-crud, ถูกบังเพราะ test เดิมพิมพ์ discount ต่อเสมอ) แก้ด้วย content-key (`JSON.stringify(watchedItems)`) ใน dep array

## Tech-debt (เลื่อน ไม่กระทบ AC)

- TC-REPO-PT-01..03 (preload order, full-replace, cascade) ต้องรันกับ Postgres จริงใน CI ที่มี Docker ก่อนเชื่อว่าเขียวจริง
- VISUAL_DESIGN_GUIDE styling เต็มรูป + component decomposition ยังค้างจาก quotation-crud (ยกยอดมา)
