/**
 * Quotation calculation engine — mirrors the backend integer-cents algorithm.
 *
 * All intermediate math uses integer "satang" (cents) to avoid floating-point
 * tie-case rounding errors (e.g. 10.50 * 0.07 = 0.735 must round UP to 0.74).
 */

/**
 * Round a baht amount to integer cents using round-half-up.
 * @param {number} amount - baht value (float)
 * @returns {number} integer cents
 */
export function roundHalfUpCents(amount) {
  return Math.round(amount * 100);
}

/**
 * Calculate line total for a single item.
 * Returns baht (not cents).
 *
 * Algorithm: round unitPrice to cents, multiply by integer qty, convert back to baht.
 *
 * @param {number} unitPrice - price per unit in baht
 * @param {number} qty - integer quantity
 * @returns {number} line total in baht
 */
export function calcLineTotal(unitPrice, qty) {
  return (roundHalfUpCents(unitPrice) * qty) / 100;
}

/**
 * Calculate quotation totals: subtotal, VAT, and grand total.
 *
 * VAT is computed with round-half-up on integer cents:
 *   vatCents = floor((baseCents * 7 + 50) / 100)
 *
 * @param {{unitPrice: number, qty: number}[]} items
 * @param {number} discountAmount - discount in baht
 * @returns {{subtotal: number, discountAmount: number, vatAmount: number|null, total: number|null, error: string|null}}
 */
export function calcTotals(items, discountAmount) {
  // 1. line item cents array
  const lineItemCentsArray = items.map((i) => roundHalfUpCents(i.unitPrice) * i.qty);

  // 2. subtotal in cents
  const subtotalCents = lineItemCentsArray.reduce((a, b) => a + b, 0);

  // 3. discount in cents
  const discountCents = roundHalfUpCents(discountAmount);

  // 4. validate discount
  if (discountCents < 0 || discountCents > subtotalCents) {
    return {
      subtotal: subtotalCents / 100,
      discountAmount,
      vatAmount: null,
      total: null,
      error: 'DISCOUNT_EXCEEDS_SUBTOTAL',
    };
  }

  // 5. compute VAT and total
  const baseCents = subtotalCents - discountCents;
  const vatCents = Math.floor((baseCents * 7 + 50) / 100);
  const totalCents = baseCents + vatCents;

  return {
    subtotal: subtotalCents / 100,
    discountAmount,
    vatAmount: vatCents / 100,
    total: totalCents / 100,
    error: null,
  };
}

/**
 * Check whether the sum of payment-term amounts exactly matches the quotation total.
 * Compares using integer cents (via roundHalfUpCents) to avoid floating-point
 * tie-case errors. Empty/undefined/null terms is always valid (optional feature).
 *
 * @param {{amount: number}[] | undefined | null} terms
 * @param {number} total - baht value
 * @returns {boolean}
 */
export function paymentTermsSumMatchesTotal(terms, total) {
  if (!terms || terms.length === 0) return true;
  const sumCents = terms.reduce((sum, t) => sum + roundHalfUpCents(Number(t.amount) || 0), 0);
  const totalCents = roundHalfUpCents(Number(total) || 0);
  return sumCents === totalCents;
}
