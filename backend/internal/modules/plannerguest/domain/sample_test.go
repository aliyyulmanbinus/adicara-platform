package domain

import "testing"

// The 20 guests of the sample project are the table of 05 BE 6: the document is read, and every column must agree.
func TestSampleGuestsMatchSpec(t *testing.T) {
	rows := sampleRows(t)
	if len(rows) != 20 || len(SampleGuests) != 20 {
		t.Fatalf("spec has %d rows, code %d; want 20", len(rows), len(SampleGuests))
	}
	var pax, sent int
	for i, r := range rows {
		g := SampleGuests[i]
		if g.Name != r.Name || g.Side != r.Side || g.Category != r.Category || g.Scope != r.Scope || g.Table != r.Table || g.Pax != r.Pax ||
			g.Phone != r.Phone || g.RSVP != r.RSVP || g.Sent != r.Sent {
			t.Errorf("row %d: code %+v, spec %+v", i+1, g, r)
		}
		if r.No != i+1 {
			t.Errorf("row %d is numbered %d in the spec", i+1, r.No)
		}
		pax += g.Pax
		if g.Sent {
			sent++
		}
	}
	if pax != 43 || sent != 14 {
		t.Errorf("%d pax and %d sent, want 43 and 14", pax, sent)
	}

	// Every side and category of the table is one of the wedding registry.
	spec := specsFromTemplateEvent(t)["wedding"]
	for _, g := range SampleGuests {
		if _, ok := spec.ResolveSide(g.Side); !ok || g.Side == "" {
			t.Errorf("%s: side %q", g.Name, g.Side)
		}
		if !spec.registryCategory(g.Category) {
			t.Errorf("%s: category %q is not in the registry", g.Name, g.Category)
		}
	}
}
