package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Contract required from dev (internal/service/quotation_calc.go):
//
//	func roundHalfUpCents(amount float64) int64
//	func calcLineTotalCents(unitPrice float64, qty int) int64
//	func calcTotals(lineItemCents []int64, discountAmount float64) (subtotalCents, baseCents, vatCents, totalCents int64, err error)

func TestCalcLineTotalCents_TC01_Happy(t *testing.T) {
	// Arrange — AC1: two line items
	// Act + Assert (two separate assertions per spec)
	require.Equal(t, int64(200000), calcLineTotalCents(1000.00, 2))
	require.Equal(t, int64(75150), calcLineTotalCents(250.50, 3))
}

func TestCalcTotals_TC02_HappyAC1(t *testing.T) {
	// Arrange — AC1: lineItemCents=[200000,75150], discount=151.50
	// Act
	subtotalCents, baseCents, vatCents, totalCents, err := calcTotals([]int64{200000, 75150}, 151.50)
	// Assert
	require.NoError(t, err)
	require.Equal(t, int64(275150), subtotalCents)
	require.Equal(t, int64(260000), baseCents)
	require.Equal(t, int64(18200), vatCents)
	require.Equal(t, int64(278200), totalCents)
}

func TestCalcTotals_TC03_RoundHalfUpTieAC2(t *testing.T) {
	// Arrange — AC2: tie case base=10.50 => vat=0.74 (round half-up, not 0.73)
	// lineItemCents=[1050] => subtotalCents=1050, discount=0 => baseCents=1050
	// vatCents = (1050*7+50)/100 = 7350+50 / 100 = 7400/100 = 74
	// Act
	subtotalCents, baseCents, vatCents, totalCents, err := calcTotals([]int64{1050}, 0)
	// Assert
	require.NoError(t, err)
	require.Equal(t, int64(1050), subtotalCents)
	require.Equal(t, int64(1050), baseCents)
	require.Equal(t, int64(74), vatCents) // NOT 73 — round-half-up tie
	require.Equal(t, int64(1124), totalCents)
}

func TestCalcTotals_TC04_DiscountExceedsSubtotalAC3(t *testing.T) {
	// Arrange — AC3: discount_amount=3000.00 > subtotal=2751.50
	// discountCents = 300000 > subtotalCents=275150
	// Act
	_, _, _, _, err := calcTotals([]int64{275150}, 3000.00)
	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, ErrValidation)
}

func TestCalcTotals_TC05_DiscountEqualsSubtotalBoundaryAC4(t *testing.T) {
	// Arrange — AC4: discount == subtotal exactly
	// discountCents = 275150 == subtotalCents=275150 => base=0
	// Act
	subtotalCents, baseCents, vatCents, totalCents, err := calcTotals([]int64{275150}, 2751.50)
	// Assert
	require.NoError(t, err)
	require.Equal(t, int64(275150), subtotalCents)
	require.Equal(t, int64(0), baseCents)
	require.Equal(t, int64(0), vatCents)
	require.Equal(t, int64(0), totalCents)
}

func TestCalcTotals_TC06_NegativeDiscountInvalid(t *testing.T) {
	// Arrange — Decision #8: discount must be >= 0
	// discount=-0.01 => discountCents=-1 < 0
	// Act
	_, _, _, _, err := calcTotals([]int64{275150}, -0.01)
	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, ErrValidation)
}

func TestRoundHalfUpCents_TC07_BasicConversion(t *testing.T) {
	// Arrange — AC1 support: convert THB to cents
	// Act + Assert
	require.Equal(t, int64(25050), roundHalfUpCents(250.50))
	require.Equal(t, int64(100000), roundHalfUpCents(1000.00))
}

func TestRoundHalfUpCents_TC08_RoundsNotTruncates(t *testing.T) {
	// Arrange — Decision #1: must round half-up, not truncate
	// 0.006 THB = 0.6 cents => rounds up to 1 (not truncated to 0)
	// Act
	result := roundHalfUpCents(0.006)
	// Assert
	require.Equal(t, int64(1), result)
}
