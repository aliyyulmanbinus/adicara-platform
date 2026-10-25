package domain

import (
	"sort"
	"strings"
)

// Cell is one row of `GROUP BY guest_side, category` (05 BE 2.8); Side nil = not decided yet.
type Cell struct {
	Side        *string
	Category    string
	Invitations int
	Pax         int
}

// Aggregates is everything AssembleOverview needs: the summary row and the side x category cells.
type Aggregates struct {
	Total, TotalPax                                       int
	Sent                                                  int // sent + delivered
	Attending, Declined, Maybe, Unconfirmed, AttendingPax int
	Cells                                                 []Cell
}

// categoryOrder gives the fixed order: registry categories in registry order, then "lainnya", then free-text
// categories alphabetically (case-insensitive).
func (s GuestSpec) categoryOrder(used map[string]bool) []string {
	var keys []string
	seen := map[string]bool{}
	for _, o := range s.Categories {
		keys = append(keys, o.Key)
		seen[o.Key] = true
	}
	if used[FallbackCategory] && !seen[FallbackCategory] {
		keys = append(keys, FallbackCategory)
		seen[FallbackCategory] = true
	}
	var custom []string
	for k := range used {
		if !seen[k] {
			custom = append(custom, k)
		}
	}
	sort.Slice(custom, func(i, j int) bool {
		a, b := strings.ToLower(custom[i]), strings.ToLower(custom[j])
		if a != b {
			return a < b
		}
		return custom[i] < custom[j]
	})
	return append(keys, custom...)
}

// AssembleOverview builds the GuestOverview from the SQL result: it adds the registry entries that are zero, the
// labels, the order and the "not decided" side. A type without sides never produces `sides` or `side_categories`.
func (s GuestSpec) AssembleOverview(a Aggregates) GuestOverview {
	o := GuestOverview{
		Total: a.Total, TotalPax: a.TotalPax,
		Sides: []GuestSide{}, Categories: []CategoryCount{}, SideCategories: []SideCategoryCount{},
		Invitation: InvitationCounts{Sent: a.Sent, Unsent: a.Total - a.Sent},
		RSVP:       RSVPSummary{Attending: a.Attending, Declined: a.Declined, Maybe: a.Maybe, Unconfirmed: a.Unconfirmed, AttendingPax: a.AttendingPax},
	}

	used := map[string]bool{}
	byCat := map[string]*CategoryCount{}
	for _, c := range a.Cells {
		used[c.Category] = true
		cc := byCat[c.Category]
		if cc == nil {
			cc = &CategoryCount{Key: c.Category}
			byCat[c.Category] = cc
		}
		cc.Invitations += c.Invitations
		cc.Pax += c.Pax
	}
	order := s.categoryOrder(used)
	rank := map[string]int{}
	for i, k := range order {
		rank[k] = i
		cc := byCat[k]
		if cc == nil {
			cc = &CategoryCount{Key: k}
		}
		cc.Label = s.CategoryLabel(k)
		cc.Custom = !s.registryCategory(k)
		o.Categories = append(o.Categories, *cc)
	}

	if len(s.Sides) == 0 {
		return o
	}

	// Sides: every registry side (also the ones at 0), then an unknown side if it holds data, then "not decided".
	type tot struct{ inv, pax int }
	known := map[string]*tot{}
	var none tot
	for _, c := range a.Cells {
		if c.Side == nil {
			none.inv += c.Invitations
			none.pax += c.Pax
			continue
		}
		t := known[*c.Side]
		if t == nil {
			t = &tot{}
			known[*c.Side] = t
		}
		t.inv += c.Invitations
		t.pax += c.Pax
	}
	sideRank := map[string]int{}
	for i, sd := range s.Sides {
		k := sd.Key
		sideRank[k] = i
		t := known[k]
		if t == nil {
			t = &tot{}
		}
		o.Sides = append(o.Sides, GuestSide{Key: &k, Label: sd.Label, Invitations: t.inv, Pax: t.pax})
		delete(known, k)
	}
	var extra []string
	for k := range known {
		extra = append(extra, k)
	}
	sort.Strings(extra)
	for i, k := range extra {
		k := k
		sideRank[k] = len(s.Sides) + i
		o.Sides = append(o.Sides, GuestSide{Key: &k, Label: k, Invitations: known[k].inv, Pax: known[k].pax})
	}
	if none.inv > 0 {
		o.Sides = append(o.Sides, GuestSide{Label: UndecidedSideLabel, Invitations: none.inv, Pax: none.pax})
	}

	// Side x category cells: filled ones only, by side and then category (the same order as above).
	sideIdx := func(p *string) int {
		if p == nil {
			return 1 << 30
		}
		return sideRank[*p]
	}
	for _, c := range a.Cells {
		if c.Invitations > 0 {
			o.SideCategories = append(o.SideCategories, SideCategoryCount{Side: c.Side, Category: c.Category, Invitations: c.Invitations, Pax: c.Pax})
		}
	}
	sort.Slice(o.SideCategories, func(i, j int) bool {
		x, y := o.SideCategories[i], o.SideCategories[j]
		if sideIdx(x.Side) != sideIdx(y.Side) {
			return sideIdx(x.Side) < sideIdx(y.Side)
		}
		return rank[x.Category] < rank[y.Category]
	})
	return o
}

// ReceptionFilters builds the filter chips of the reception tab (05 BE 2.5) from the counts of the expected guests:
// the three status counts and the categories, in the same order as the overview (the registry ones also at 0, then the
// used "lainnya" and free-text ones). Pax is not filled.
func (s GuestSpec) ReceptionFilters(all, checkedIn, pending int, byCategory map[string]int) ReceptionFilters {
	used := map[string]bool{}
	for k, n := range byCategory {
		if n > 0 {
			used[k] = true
		}
	}
	f := ReceptionFilters{All: all, CheckedIn: checkedIn, Pending: pending, Categories: []CategoryCount{}}
	for _, k := range s.categoryOrder(used) {
		f.Categories = append(f.Categories, CategoryCount{Key: k, Label: s.CategoryLabel(k), Custom: !s.registryCategory(k), Invitations: byCategory[k]})
	}

	return f
}
