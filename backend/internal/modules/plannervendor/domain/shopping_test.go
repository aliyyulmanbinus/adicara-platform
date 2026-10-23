package domain

import (
	"encoding/json"
	"os"
	"testing"
)

// The vectors were computed by PostgreSQL 16.2 (`round(q100 / 100.0 * price)::bigint`, null above 10¹³); the frontend
// (04 FE lineTotalX100) reads the same file.
const lineTotalVectors = "../../../../../docs/APP_SPEC/PLANNER_SPEC/04_VENDOR_DAN_BELANJA/fixtures/planner-line-totals.json"

func TestLineTotalMatchesPostgres(t *testing.T) {
	raw, err := os.ReadFile(lineTotalVectors)
	if err != nil {
		t.Skipf("vectors not available: %v", err)
	}
	var vectors []struct {
		Q100  int64  `json:"q100"`
		Price int64  `json:"price"`
		Total *int64 `json:"total"`
	}
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 300 {
		t.Fatalf("%d vectors, want 300", len(vectors))
	}
	for _, v := range vectors {
		got, ok := LineTotal(v.Q100, v.Price)
		if v.Total == nil {
			if ok {
				t.Errorf("%d × %d: want over the limit, got %d", v.Q100, v.Price, got)
			}

			continue
		}
		if !ok || got != *v.Total {
			t.Errorf("%d × %d: got %d, %v; want %d", v.Q100, v.Price, got, ok, *v.Total)
		}
	}
}

func TestLineTotalEdges(t *testing.T) {
	for _, c := range []struct {
		q, price int64
		want     int64
		ok       bool
	}{
		{100, 194_000, 194_000, true},
		{50, 1, 1, true}, // 0,5 × 1 rounds half up
		{1, 50, 1, true}, // 0,01 × 50
		{1, 49, 0, true}, // 0,49 rounds down
		{MaxQuantityX100, 100_000, 100_000_000_000, true},
		{MaxQuantityX100, 10_000_000, MaxMoney, true}, // exactly 10¹³
		{MaxQuantityX100, 10_000_001, 0, false},       // 10¹³ + 1
		{-1, 1, 0, false},
		{1, -1, 0, false},
		{1 << 62, 1 << 62, 0, false}, // the 128-bit product does not fit in 64 bits
	} {
		if got, ok := LineTotal(c.q, c.price); ok != c.ok || (ok && got != c.want) {
			t.Errorf("LineTotal(%d, %d) = %d, %v; want %d, %v", c.q, c.price, got, ok, c.want, c.ok)
		}
	}
}

func TestQuantityX100(t *testing.T) {
	for in, want := range map[float64]int64{1: 100, 0.01: 1, 50.5: 5050, 300: 30000, 1_000_000: 100_000_000, 0.29: 29, 1.15: 115} {
		if got, ok := QuantityX100(in); !ok || got != want {
			t.Errorf("QuantityX100(%v) = %d, %v; want %d", in, got, ok, want)
		}
	}
	for _, in := range []float64{-1, 0.005, 1.234, 3_000_000} {
		if _, ok := QuantityX100(in); ok {
			t.Errorf("QuantityX100(%v) must be refused", in)
		}
	}
}

func TestShoppingKPIOf(t *testing.T) {
	souvenir := PurposeSouvenir
	items := []ShoppingItem{
		{Status: ShoppingDone, EstimatedTotal: 100, ActualTotal: 90, Quantity: 300, Purpose: &souvenir},
		{Status: ShoppingDone, EstimatedTotal: 50, ActualTotal: 55, Quantity: 50.5, Purpose: &souvenir},
		{Status: ShoppingInProgress, EstimatedTotal: 30, ActualTotal: 28, Quantity: 20, Purpose: &souvenir},
		{Status: ShoppingNotStarted, EstimatedTotal: 20},
	}
	k := ShoppingKPIOf(items)
	if k.EstimatedTotal != 200 || k.ActualTotal != 145 || k.Done != 2 || k.InProgress != 1 || k.NotStarted != 1 || k.Total != 4 {
		t.Errorf("kpi %+v", k)
	}
	if k.SouvenirStock == nil || *k.SouvenirStock != 350 { // floor(300 + 50,5); the item still in progress is not stock
		t.Errorf("stock %v", k.SouvenirStock)
	}
	if k := ShoppingKPIOf(items[3:]); k.SouvenirStock != nil {
		t.Errorf("stock without souvenir items: %v", *k.SouvenirStock)
	}
	unfinished := items[2]
	if k := ShoppingKPIOf([]ShoppingItem{unfinished}); k.SouvenirStock == nil || *k.SouvenirStock != 0 {
		t.Errorf("souvenir items that are not finished give a stock of 0, got %v", k.SouvenirStock)
	}
}
