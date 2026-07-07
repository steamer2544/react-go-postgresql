package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Contract required from dev (internal/service/quotation_calc.go):
//
//	func validatePaymentTermsCents(termAmounts []float64, totalCents int64) error

func TestValidatePaymentTermsCents_TC_CALC_PT01_HappyAC1(t *testing.T) {
	// Arrange — AC1: 3 terms [891.67, 891.67, 891.66] sum cents = 89167+89167+89166 = 267500
	// Act
	err := validatePaymentTermsCents([]float64{891.67, 891.67, 891.66}, 267500)
	// Assert
	require.NoError(t, err)
}

func TestValidatePaymentTermsCents_TC_CALC_PT02_FloatPrecisionAC2(t *testing.T) {
	// Arrange — AC2: float-precision case total=100.00, terms [33.33, 33.33, 33.34]
	// cents: 3333+3333+3334 = 10000 == totalCents(10000)
	// Act
	err := validatePaymentTermsCents([]float64{33.33, 33.33, 33.34}, 10000)
	// Assert
	require.NoError(t, err)
}

func TestValidatePaymentTermsCents_TC_CALC_PT03_MismatchAC3(t *testing.T) {
	// Arrange — AC3: terms [1000, 1000, 1000] sum cents = 300000 != totalCents(267500)
	// Act
	err := validatePaymentTermsCents([]float64{1000, 1000, 1000}, 267500)
	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, ErrValidation)
}

func TestValidatePaymentTermsCents_TC_CALC_PT04_EmptyNilAC5(t *testing.T) {
	// Arrange — AC5: nil and empty slice should both pass (optional feature)
	// Act + Assert (nil input)
	errNil := validatePaymentTermsCents(nil, 267500)
	require.NoError(t, errNil)

	// Act + Assert (empty slice)
	errEmpty := validatePaymentTermsCents([]float64{}, 267500)
	require.NoError(t, errEmpty)
}

func TestValidatePaymentTermsCents_TC_CALC_PT05_OffByOneSatangAC3(t *testing.T) {
	// Arrange — AC3 defense: [891.68, 891.67, 891.66] sum cents = 267501 != 267500
	// Proves exact int64 compare, not tolerance.
	// Act
	err := validatePaymentTermsCents([]float64{891.68, 891.67, 891.66}, 267500)
	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, ErrValidation)
}
