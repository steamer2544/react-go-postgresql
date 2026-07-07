// Package service provides business logic for the application.
package service

import (
	"math"
)

// roundHalfUpCents converts a monetary amount in major units (e.g. THB)
// to integer cents (satang), rounding half-up (ties away from zero).
func roundHalfUpCents(amount float64) int64 {
	return int64(math.Floor(amount*100 + 0.5))
}

// calcLineTotalCents returns qty * roundHalfUpCents(unitPrice).
// qty is an exact integer so this multiplication is 100% exact.
func calcLineTotalCents(unitPrice float64, qty int) int64 {
	return int64(qty) * roundHalfUpCents(unitPrice)
}

// calcTotals sums lineItemCents (already-computed line totals in cents),
// applies discountAmount (major units), and computes VAT 7% with
// half-up rounding using pure integer arithmetic:
//
//	subtotalCents = sum(lineItemCents)
//	discountCents = roundHalfUpCents(discountAmount)
//	baseCents     = subtotalCents - discountCents
//	vatCents      = (baseCents*7 + 50) / 100   // integer division, half-up
//	totalCents    = baseCents + vatCents
//
// Returns service.ErrValidation if discountCents < 0 or discountCents > subtotalCents.
// On error, the four returned cents values are zero.
func calcTotals(lineItemCents []int64, discountAmount float64) (subtotalCents, baseCents, vatCents, totalCents int64, err error) {
	subtotalCents = 0
	for _, c := range lineItemCents {
		subtotalCents += c
	}

	discountCents := roundHalfUpCents(discountAmount)
	if discountCents < 0 || discountCents > subtotalCents {
		return 0, 0, 0, 0, ErrValidation
	}

	baseCents = subtotalCents - discountCents
	vatCents = (baseCents*7 + 50) / 100
	totalCents = baseCents + vatCents
	return subtotalCents, baseCents, vatCents, totalCents, nil
}

// validatePaymentTermsCents checks that payment-term amounts sum exactly to
// totalCents (in satang). Empty/nil termAmounts is always valid (feature is
// optional — a quotation may have zero payment terms).
func validatePaymentTermsCents(termAmounts []float64, totalCents int64) error {
	if len(termAmounts) == 0 {
		return nil
	}
	var sumCents int64
	for _, a := range termAmounts {
		sumCents += roundHalfUpCents(a)
	}
	if sumCents != totalCents {
		return ErrValidation
	}
	return nil
}
