// Test cases: TC-FE-CALC-PT-01 .. TC-FE-CALC-PT-05
//
// Assumed contract (documented — not yet in the codebase):
//   - "@/features/quotation/utils/calcQuotation" exports:
//       roundHalfUpCents(amount: number): number          (changed from private -> export)
//       paymentTermsSumMatchesTotal(terms: {amount:number}[] | undefined | null, total: number): boolean
//     terms empty/undefined/null => true (optional feature)
//     compares integer cents exactly (no direct float comparison)

import { describe, expect, it } from 'vitest';
import {
  roundHalfUpCents,
  paymentTermsSumMatchesTotal,
} from '@/features/quotation/utils/calcQuotation';

describe('calcQuotation — payment terms', () => {
  // TC-FE-CALC-PT-01: happy — 3 terms [891.67, 891.67, 891.66] sum == 2675.00
  it('TC-FE-CALC-PT-01: paymentTermsSumMatchesTotal([891.67,891.67,891.66], 2675.00) === true', () => {
    const terms = [{ amount: 891.67 }, { amount: 891.67 }, { amount: 891.66 }];
    expect(paymentTermsSumMatchesTotal(terms, 2675.0)).toBe(true);
  });

  // TC-FE-CALC-PT-02: edge (float precision) — [33.33, 33.33, 33.34] sum == 100.00
  it('TC-FE-CALC-PT-02: paymentTermsSumMatchesTotal([33.33,33.33,33.34], 100.00) === true', () => {
    const terms = [{ amount: 33.33 }, { amount: 33.33 }, { amount: 33.34 }];
    expect(paymentTermsSumMatchesTotal(terms, 100.0)).toBe(true);
  });

  // TC-FE-CALC-PT-03: mismatch — [1000, 1000, 1000] sum 3000 ≠ 2675
  it('TC-FE-CALC-PT-03: paymentTermsSumMatchesTotal([1000,1000,1000], 2675.00) === false', () => {
    const terms = [{ amount: 1000 }, { amount: 1000 }, { amount: 1000 }];
    expect(paymentTermsSumMatchesTotal(terms, 2675.0)).toBe(false);
  });

  // TC-FE-CALC-PT-04: edge — empty/undefined/null => always true
  it('TC-FE-CALC-PT-04: paymentTermsSumMatchesTotal([], 2675.00) and undefined => true', () => {
    expect(paymentTermsSumMatchesTotal([], 2675.0)).toBe(true);
    expect(paymentTermsSumMatchesTotal(undefined, 100)).toBe(true);
  });

  // TC-FE-CALC-PT-05: roundHalfUpCents is exported and correct
  it('TC-FE-CALC-PT-05: roundHalfUpCents(1000.00) === 100000 and roundHalfUpCents(0.006) === 1', () => {
    expect(roundHalfUpCents(1000.0)).toBe(100000);
    expect(roundHalfUpCents(0.006)).toBe(1);
  });
});
