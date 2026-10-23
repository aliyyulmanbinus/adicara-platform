package usecase

import (
	"errors"
	"strings"
	"testing"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/domain"
)

func wantAPIError(t *testing.T, err error, code, message string) {
	t.Helper()
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != code || ae.Message != message {
		t.Errorf("error = %v, want %s: %s", err, code, message)
	}
}

func ptr[T any](v T) *T { return &v }

func TestResolveCategory(t *testing.T) {
	cats := []domain.CategorySpec{{Key: "venue", Label: "Venue"}, {Key: "mua", Label: "MUA"}}

	for _, c := range []struct {
		key, label *string
		wantKey    string
		wantLabel  string
	}{
		{ptr("mua"), nil, "mua", "MUA"},
		{ptr(" mua "), ptr("Salon"), "mua", "MUA"}, // a registry key wins over a label
		{nil, ptr("Venue"), "venue", "Venue"},      // a label that derives a registry key merges with it
		{ptr(""), ptr(" venue  "), "venue", "Venue"},
		{nil, ptr("Dekorasi & Tenda"), "dekorasi_tenda", "Dekorasi & Tenda"},
		{nil, ptr("🎉"), "kategori", "🎉"},
	} {
		key, label, err := resolveCategory(cats, c.key, c.label)
		if err != nil || key != c.wantKey || label != c.wantLabel {
			t.Errorf("resolveCategory(%v, %v) = %q, %q, %v; want %q, %q", c.key, c.label, key, label, err, c.wantKey, c.wantLabel)
		}
	}

	_, _, err := resolveCategory(cats, ptr("kuliner"), nil)
	wantAPIError(t, err, "invalid_input", "category_key: must be a registry category key")
	_, _, err = resolveCategory(cats, nil, nil)
	wantAPIError(t, err, "invalid_input", "category_label: required when category_key is empty")
	_, _, err = resolveCategory(cats, nil, ptr(strings.Repeat("a", 61)))
	wantAPIError(t, err, "invalid_input", "category_label: must be at most 60 characters")
}

func TestVendorStatusRules(t *testing.T) {
	for _, c := range []struct {
		in     domain.VendorStatus
		agreed int64
		want   domain.VendorStatus
		ok     bool
	}{
		{"", 0, domain.VendorPlanned, true}, {"fix", 5, domain.VendorFixed, true}, {"fix", 0, "", false}, {"batal", 0, domain.VendorCancelled, true},
		{"negosiasi", 9, domain.VendorNegotiating, true}, {"lunas", 1, "", false},
	} {
		got, err := vendorStatus(c.in, c.agreed)
		if (err == nil) != c.ok || got != c.want {
			t.Errorf("vendorStatus(%q, %d) = %q, %v", c.in, c.agreed, got, err)
		}
	}
}

func TestShoppingWriteDefaultsAndLimits(t *testing.T) {
	w, err := shoppingWrite(shoppingInput{category: " Dekorasi ", name: " Balon ", estimated: 1000})
	if err != nil || w.SubEventScope != pd.AllScope || w.Category != "Dekorasi" || w.ItemName != "Balon" || w.QuantityX100 != 100 || w.Status != domain.ShoppingNotStarted || w.Purpose != nil || w.Unit != nil {
		t.Errorf("defaults: %+v, %v", w, err)
	}

	for _, c := range []struct {
		q       float64
		price   int64
		wantErr string
	}{
		{1_000_000, 10_000_000, ""}, {1_000_000, 10_000_001, "estimated_price: the total (quantity × price) must be at most 10000000000000"},
		{0.01, 10_000_000_000_000, ""}, {0.5, 20_000_000_000_001, "estimated_price: must be between 0 and 10000000000000"},
	} {
		_, err := shoppingWrite(shoppingInput{category: "a", name: "b", quantity: &c.q, estimated: c.price})
		if (c.wantErr == "") != (err == nil) || (err != nil && !strings.HasSuffix(err.Error(), c.wantErr)) {
			t.Errorf("quantity %v × %d: %v, want %q", c.q, c.price, err, c.wantErr)
		}
	}

	for _, q := range []float64{0, -1, 0.001, 1.234, 1_000_001} {
		_, err := shoppingWrite(shoppingInput{category: "a", name: "b", quantity: &q})
		wantAPIError(t, err, "invalid_input", "quantity: must be between 0.01 and 1000000")
	}
	_, err = shoppingWrite(shoppingInput{category: "a", name: "b", purpose: ptr("gift")})
	wantAPIError(t, err, "invalid_input", "purpose: must be one of souvenir")
	if w, err := shoppingWrite(shoppingInput{category: "a", name: "b", purpose: ptr("")}); err != nil || w.Purpose != nil {
		t.Errorf("an empty purpose is none: %+v, %v", w, err)
	}
}

func TestCleanPhone(t *testing.T) {
	phone, wa, err := cleanPhone("phone_number", ptr(" 0812-9876-5432 "))
	if err != nil || *phone != "0812-9876-5432" || *wa != "6281298765432" {
		t.Errorf("= %v %v %v", phone, wa, err)
	}
	for _, blank := range []*string{nil, ptr(""), ptr("   ")} {
		if phone, wa, err := cleanPhone("phone_number", blank); err != nil || phone != nil || wa != nil {
			t.Errorf("blank %v = %v %v %v", blank, phone, wa, err)
		}
	}
	_, _, err = cleanPhone("phone_number", ptr("+65 9123 4567"))
	wantAPIError(t, err, "invalid_phone", "phone_number: must be an Indonesian phone number (08…, +62…)")
	_, _, err = cleanPhone("phone_number", ptr(strings.Repeat("1", 41)))
	wantAPIError(t, err, "invalid_input", "phone_number: must be at most 40 characters")
}

func TestCleanURLAndText(t *testing.T) {
	for _, ok := range []string{"http://a.id", "https://example.com/x?y=1"} {
		if u, err := cleanURL("portfolio_url", &ok); err != nil || *u != ok {
			t.Errorf("%q: %v %v", ok, u, err)
		}
	}
	for _, bad := range []string{"example.com", "ftp://x.y", "javascript:alert(1)", "https://", "https://a b.com/\x00"} {
		bad := bad
		if _, err := cleanURL("portfolio_url", &bad); err == nil {
			t.Errorf("%q must be refused", bad)
		}
	}
	if got := truncRunes(strings.Repeat("é", 200), 150); len([]rune(got)) != 150 {
		t.Errorf("truncRunes cut to %d runes", len([]rune(got)))
	}
	if got := truncRunes("pendek", 150); got != "pendek" {
		t.Errorf("truncRunes(short) = %q", got)
	}
	if n, err := cleanNote("notes", "a\r\nb\n", 10); err != nil || n != "a\nb" {
		t.Errorf("cleanNote = %q, %v", n, err)
	}
}

func TestVendorFilter(t *testing.T) {
	if err := (VendorFilter{Scope: "resepsi", Status: "fix", Category: "venue"}).validate(); err != nil {
		t.Error(err)
	}
	for _, f := range []VendorFilter{{Scope: "A B"}, {Status: "x"}, {Category: "Has Space"}} {
		if err := f.validate(); err == nil {
			t.Errorf("%+v must be refused", f)
		}
	}
	v := domain.Vendor{SubEventScope: "resepsi", Status: domain.VendorFixed, CategoryKey: "venue"}
	if !(VendorFilter{}).keep(v) || !(VendorFilter{Scope: "resepsi", Status: "fix"}).keep(v) || (VendorFilter{Scope: "all"}).keep(v) || (VendorFilter{Category: "mua"}).keep(v) {
		t.Error("keep")
	}
}
