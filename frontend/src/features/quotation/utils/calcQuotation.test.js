// Test cases: TC-FE-CALC-01 .. TC-FE-CALC-05
//
// Assumed contract (documented — not yet in the codebase):
//   - "@/features/quotation/utils/calcQuotation" exports:
//       calcLineTotal(unitPrice: number, qty: number): number
//       calcTotals(items: {unitPrice, qty}[], discountAmount: number):
//         { subtotal, discountAmount, vatAmount, total, error }
//     error === 'DISCOUNT_EXCEEDS_SUBTOTAL' when discountAmount < 0 or > subtotal.
//     Internally mirrors the backend integer-cents algorithm (Math.round(x*100),
//     integer VAT formula) — NOT toFixed on a float product.

import { describe, expect, it } from 'vitest';
import { calcLineTotal, calcTotals } from '@/features/quotation/utils/calcQuotation';

describe('calcQuotation', () => {
  // TC-FE-CALC-01: happy — calcLineTotal two pairs
  it('TC-FE-CALC-01: calcLineTotal(1000, 2) ≈ 2000 and calcLineTotal(250.50, 3) ≈ 751.50', () => {
    expect(calcLineTotal(1000.0, 2)).toBeCloseTo(2000.0, 2);
    expect(calcLineTotal(250.5, 3)).toBeCloseTo(751.5, 2);
  });

  // TC-FE-CALC-02: happy — full totals with discount
  it('TC-FE-CALC-02: calcTotals with 2 items + discount 151.50 yields subtotal=2751.50, vat=182.00, total=2782.00', () => {
    const items = [
      { unitPrice: 1000.0, qty: 2 },
      { unitPrice: 250.5, qty: 3 },
    ];
    const result = calcTotals(items, 151.5);
    expect(result.error).toBeNull();
    expect(result.subtotal).toBeCloseTo(2751.5, 2);
    expect(result.discountAmount).toBeCloseTo(151.5, 2);
    expect(result.vatAmount).toBeCloseTo(182.0, 2);
    expect(result.total).toBeCloseTo(2782.0, 2);
  });

  // TC-FE-CALC-03: edge (tie) — VAT rounding must round up (0.74, not 0.73)
  it('TC-FE-CALC-03: tie-case subtotal=10.50 discount=0 => vatAmount=0.74 (not 0.73), total=11.24', () => {
    const result = calcTotals([{ unitPrice: 10.5, qty: 1 }], 0);
    expect(result.error).toBeNull();
    // 10.50 * 0.07 = 0.735 — tie must round UP to 0.74
    expect(result.vatAmount).toBeCloseTo(0.74, 2);
    expect(result.total).toBeCloseTo(11.24, 2);
  });

  // TC-FE-CALC-04: error — discount exceeds subtotal
  it('TC-FE-CALC-04: discount > subtotal returns error code, vatAmount=null, total=null', () => {
    const items = [
      { unitPrice: 1000.0, qty: 2 },
      { unitPrice: 250.5, qty: 3 },
    ];
    const result = calcTotals(items, 3000.0);
    expect(result.error).toBe('DISCOUNT_EXCEEDS_SUBTOTAL');
    expect(result.vatAmount).toBeNull();
    expect(result.total).toBeNull();
  });

  // TC-FE-CALC-05: boundary — discount == subtotal exactly
  it('TC-FE-CALC-05: discount === subtotal => vat=0, total=0, error=null', () => {
    const items = [
      { unitPrice: 1000.0, qty: 2 },
      { unitPrice: 250.5, qty: 3 },
    ];
    const result = calcTotals(items, 2751.5);
    expect(result.error).toBeNull();
    expect(result.vatAmount).toBeCloseTo(0, 2);
    expect(result.total).toBeCloseTo(0, 2);
  });
});
