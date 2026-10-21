package domain

import (
	"math"
	"math/big"
)

// MaxMoney is the largest amount of any money field (10^13, as in menus 01 and 02).
const MaxMoney int64 = 10_000_000_000_000

// Quantity of a unit-priced line is NUMERIC(12,2); it travels as hundredths so
// the arithmetic stays exact: 0,01 … 1.000.000,00.
const (
	MinQuantityX100 int64 = 1
	MaxQuantityX100 int64 = 100_000_000
)

// QuantityX100 converts a quantity with at most two decimals to hundredths.
func QuantityX100(q float64) (int64, bool) {
	if math.IsNaN(q) || math.IsInf(q, 0) || q < 0 || q > 2_000_000 {
		return 0, false
	}
	hundredths := q * 100
	r := math.Round(hundredths)
	if math.Abs(hundredths-r) > 1e-6 {
		return 0, false
	}

	return int64(r), true
}

// QuantityFloat is the quantity as the API shows it.
func QuantityFloat(x100 int64) float64 { return float64(x100) / 100 }

// UnitEstimate is round(quantity × unitPrice) with the quantity in hundredths,
// half up. It is computed with math/big because quantity × price reaches 10^21,
// far beyond int64, and PostgreSQL would answer "bigint out of range" (a 500)
// for a product written unchecked. ok is false when the result exceeds MaxMoney.
func UnitEstimate(quantityX100, unitPrice int64) (int64, bool) {
	p := new(big.Int).Mul(big.NewInt(quantityX100), big.NewInt(unitPrice))
	p.Add(p, big.NewInt(50))
	p.Quo(p, big.NewInt(100))
	if !p.IsInt64() || p.Int64() > MaxMoney {
		return 0, false
	}

	return p.Int64(), true
}

// CeilDiv is ⌈a ÷ b⌉ for a ≥ 0, b > 0.
func CeilDiv(a, b int64) int64 {
	if a <= 0 {
		return 0
	}

	return (a + b - 1) / b
}
