package domain

import (
	"math"
	"math/bits"
)

// MaxMoney bounds every money value (contract chapter 6): safe for a JavaScript Number and enough for a bigint.
const MaxMoney int64 = 10_000_000_000_000

// MaxQuantityX100 is 1.000.000,00: the largest quantity of an item, in hundredths. MinQuantityX100 is 0,01.
const (
	MinQuantityX100 int64 = 1
	MaxQuantityX100 int64 = 100_000_000
)

// QuantityX100 converts a quantity with at most two decimals to hundredths; ok is false for NaN, a negative number,
// a number beyond 2.000.000 or one with a third decimal.
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

// LineTotal is round(quantity × price) without floats, with the quantity in hundredths (NUMERIC(12,2) → q100). The
// rounding is that of the generated column `round(quantity * price)::bigint` (half up for positive values). ok is false
// when the result exceeds MaxMoney: the usecase refuses before writing, because the generated column answers
// `bigint out of range` (a 500).
func LineTotal(q100, price int64) (total int64, ok bool) {
	if q100 < 0 || price < 0 {
		return 0, false
	}
	hi, lo := bits.Mul64(uint64(q100), uint64(price))
	if hi != 0 || lo > ^uint64(0)-50 {
		return 0, false
	}
	t := (lo + 50) / 100
	if t > uint64(MaxMoney) {
		return 0, false
	}

	return int64(t), true
}

// ShoppingKPIOf adds up the shopping list (04 BE 2.4): the estimate of every item, the realisation of the finished ones.
func ShoppingKPIOf(items []ShoppingItem) ShoppingKPI {
	var k ShoppingKPI
	var stockX100 int64
	var hasSouvenir bool
	for _, it := range items {
		k.Total++
		k.EstimatedTotal += it.EstimatedTotal
		switch it.Status {
		case ShoppingDone:
			k.Done++
			k.ActualTotal += it.ActualTotal
		case ShoppingInProgress:
			k.InProgress++
		default:
			k.NotStarted++
		}
		if it.Purpose != nil && *it.Purpose == PurposeSouvenir {
			hasSouvenir = true
			if it.Status == ShoppingDone {
				stockX100 += int64(math.Round(it.Quantity * 100)) // hundredths: no float sum
			}
		}
	}
	if hasSouvenir {
		s := stockX100 / 100
		k.SouvenirStock = &s
	}

	return k
}
