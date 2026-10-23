package domain

import (
	"slices"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// ScopeAll is the session every project has for what belongs to no particular session.
const ScopeAll = pd.AllScope

// BuildOverview and its helpers are pure (no database): they shape VendorOverview from the reads of the
// repository (04 BE 2.1 limits, 2.8 categories and KPI). The tests run them against the sample project and
// against all nine event types.

// CategorySpec is one vendor category of the event type registry. BudgetItemKeys empty = no line (manual limit only).
type CategorySpec struct {
	Key, Label     string
	BudgetItemKeys []string
}

// VendorRow is the part of planner_vendors that the overview and the writers use.
type VendorRow struct {
	ID                                        uuid.UUID
	SubEventScope, CategoryKey, CategoryLabel string
	VendorName                                string
	PICName, PhoneNumber, FormattedWANumber   *string
	PortfolioURL, Notes                       *string
	BudgetLimit, AgreedPrice                  int64
	Status                                    VendorStatus
	SortOrder                                 int
}

// ScenarioRow is a line of the ACTIVE scenario that takes part in the simulation (same session rule as Smart Adjust).
type ScenarioRow struct {
	SubEventScope, ItemKey, ItemName string
	Cost                             int64
}

// LineRow is a row of the view planner_budget_lines (the keyed lines and the lines that point at a vendor).
type LineRow struct {
	ID                        uuid.UUID
	SubEventScope, ItemKey    string
	ItemName                  string
	VendorID                  *uuid.UUID
	Estimate, Paid, Remaining int64
	PaymentStatus             string
}

func (l LineRow) payment() *LinePayment {
	return &LinePayment{BudgetItemID: l.ID, ItemName: l.ItemName, Estimate: l.Estimate, Paid: l.Paid, Remaining: l.Remaining, PaymentStatus: l.PaymentStatus}
}

// SharedKeys are the item_keys that two or more categories of the registry point at. Static (it does not depend on
// the vendors that exist), so adding a second vendor never changes how the first one is compared.
func SharedKeys(cats []CategorySpec) map[string]bool {
	n := map[string]int{}
	for _, c := range cats {
		if len(c.BudgetItemKeys) > 0 {
			n[c.BudgetItemKeys[0]]++
		}
	}
	out := map[string]bool{}
	for k, c := range n {
		if c >= 2 {
			out[k] = true
		}
	}

	return out
}

// ScenarioKey addresses a line by session and item_key.
type ScenarioKey struct{ Scope, Key string }

// ResolveKey is the first item_key of the category that the active scenario holds, looked up in the session of the
// vendor first and then in 'all': every scenario line and template line of a non-wedding event type is in 'all'.
func ResolveKey(c CategorySpec, vendorScope string, scenario map[ScenarioKey]int64) (ScenarioKey, bool) {
	for _, k := range c.BudgetItemKeys {
		for _, scope := range []string{vendorScope, ScopeAll} {
			if _, ok := scenario[ScenarioKey{scope, k}]; ok {
				return ScenarioKey{scope, k}, true
			}
		}
	}

	return ScenarioKey{}, false
}

// ScenarioCosts indexes the active scenario by session and key; the amounts of equal keys add up.
func ScenarioCosts(rows []ScenarioRow) map[ScenarioKey]int64 {
	cost := map[ScenarioKey]int64{}
	for _, r := range rows {
		cost[ScenarioKey{r.SubEventScope, r.ItemKey}] += r.Cost
	}

	return cost
}

// BuildOverview applies 04 BE 2.1 (limits) and 2.8 (categories, KPI). skipped = planner_projects.skipped_vendor_categories.
// vendors must come in display order (sort_order, then creation); the slice is not changed.
func BuildOverview(cats []CategorySpec, skipped []string, vendors []VendorRow, scenario []ScenarioRow, lines []LineRow) VendorOverview {
	vendors = slices.Clone(vendors)
	out := VendorOverview{Vendors: []Vendor{}, LimitGroups: []LimitGroup{}, Categories: []CategoryState{}}

	buildLimits(&out, cats, vendors, ScenarioCosts(scenario), lines)
	buildCategories(&out, cats, skipped, vendors)

	return out
}

// buildLimits fills Vendors, LimitGroups and the group counters of the KPI.
func buildLimits(out *VendorOverview, cats []CategorySpec, vendors []VendorRow, cost map[ScenarioKey]int64, lines []LineRow) {
	byCat := map[string]CategorySpec{}
	for _, c := range cats {
		byCat[c.Key] = c
	}
	shared := SharedKeys(cats)
	payByVendor := map[uuid.UUID]LineRow{}
	lineByKey := map[ScenarioKey]LineRow{}
	for _, l := range lines {
		if l.VendorID != nil {
			payByVendor[*l.VendorID] = l
		}
		if _, taken := lineByKey[ScenarioKey{l.SubEventScope, l.ItemKey}]; l.ItemKey != "" && !taken {
			lineByKey[ScenarioKey{l.SubEventScope, l.ItemKey}] = l
		}
	}

	groupIdx := map[ScenarioKey]*LimitGroup{}
	var groupOrder []ScenarioKey
	for _, v := range vendors {
		dto := Vendor{
			ID: v.ID, SubEventScope: v.SubEventScope, CategoryKey: v.CategoryKey, CategoryLabel: v.CategoryLabel, VendorName: v.VendorName,
			PICName: v.PICName, PhoneNumber: v.PhoneNumber, FormattedWANumber: v.FormattedWANumber, PortfolioURL: v.PortfolioURL,
			AgreedPrice: v.AgreedPrice, Status: v.Status, Notes: v.Notes, SortOrder: v.SortOrder, LimitSource: LimitSourceNone,
		}
		if l, ok := payByVendor[v.ID]; ok {
			dto.Payment = l.payment()
		}
		var sk ScenarioKey
		var found bool
		if c, ok := byCat[v.CategoryKey]; ok {
			sk, found = ResolveKey(c, v.SubEventScope, cost)
		}
		switch {
		case v.BudgetLimit > 0:
			dto.BudgetLimit, dto.LimitSource = v.BudgetLimit, LimitSourceManual
		case found && shared[sk.Key]:
			g := sk.Key
			dto.LimitSource, dto.LimitGroup = LimitSourceGroup, &g
			if v.Status != VendorCancelled {
				joinGroup(groupIdx, &groupOrder, sk, cost[sk], lineByKey, v)
			}
		case found:
			dto.BudgetLimit, dto.LimitSource = cost[sk], LimitSourceScenario
		}
		if (dto.LimitSource == LimitSourceManual || dto.LimitSource == LimitSourceScenario) && dto.BudgetLimit > 0 && v.AgreedPrice > 0 {
			d := dto.BudgetLimit - v.AgreedPrice
			dto.PriceDelta = &d
		}
		out.Vendors = append(out.Vendors, dto)
	}

	for _, sk := range groupOrder {
		g := groupIdx[sk]
		if g.Priced > 0 {
			d := g.Limit - g.TotalAgreed
			g.Delta = &d
			if d < 0 {
				out.KPI.GroupsOver++
			}
		}
		out.LimitGroups = append(out.LimitGroups, *g)
	}
	out.KPI.GroupsTotal = len(out.LimitGroups)
}

// joinGroup adds v to the group of the shared line sk, making the group on first use.
func joinGroup(groups map[ScenarioKey]*LimitGroup, order *[]ScenarioKey, sk ScenarioKey, limit int64, lineByKey map[ScenarioKey]LineRow, v VendorRow) {
	grp, ok := groups[sk]
	if !ok {
		grp = &LimitGroup{ItemKey: sk.Key, SubEventScope: sk.Scope, Limit: limit, VendorIDs: []uuid.UUID{}}
		if l, ok := lineByKey[sk]; ok && l.VendorID == nil {
			grp.Line = l.payment()
		}
		groups[sk] = grp
		*order = append(*order, sk)
	}
	grp.Members++
	grp.VendorIDs = append(grp.VendorIDs, v.ID)
	if v.AgreedPrice > 0 {
		grp.Priced++
		grp.TotalAgreed += v.AgreedPrice
	}
}

// buildCategories fills Categories and the vendor counters of the KPI: the registry categories in registry order,
// then the custom categories in use in order of appearance.
func buildCategories(out *VendorOverview, cats []CategorySpec, skipped []string, vendors []VendorRow) {
	byCat := map[string]bool{}
	for _, c := range cats {
		byCat[c.Key] = true
	}
	skip := map[string]bool{}
	for _, k := range skipped {
		skip[k] = true
	}

	type tally struct{ fix, pipeline, active int }
	counts := map[string]*tally{}
	var customOrder []VendorRow
	seenCustom := map[string]bool{}
	for _, v := range vendors {
		if v.Status == VendorCancelled {
			continue
		}
		t := counts[v.CategoryKey]
		if t == nil {
			t = &tally{}
			counts[v.CategoryKey] = t
		}
		t.active++
		if v.Status == VendorFixed {
			t.fix++
			out.KPI.FixedVendors++
			out.KPI.FixedTotal += v.AgreedPrice
		} else {
			t.pipeline++
			out.KPI.Pipeline++
			if v.AgreedPrice > 0 {
				out.KPI.PipelineOffers += v.AgreedPrice
			}
		}
		if !byCat[v.CategoryKey] && !seenCustom[v.CategoryKey] {
			seenCustom[v.CategoryKey] = true
			customOrder = append(customOrder, v)
		}
	}

	add := func(key, label string, custom bool) {
		st := CategoryState{Key: key, Label: label, State: "empty", Custom: custom, Skipped: skip[key]}
		if t := counts[key]; t != nil {
			st.Vendors = t.active
			if t.fix > 0 {
				st.State = "fix"
			} else if t.pipeline > 0 {
				st.State = "pipeline"
			}
		}
		out.Categories = append(out.Categories, st)
		if st.Skipped {
			out.KPI.Skipped++

			return
		}
		out.KPI.CategoriesTotal++
		if st.State == "fix" {
			out.KPI.CategoriesFixed++
		}
	}
	for _, c := range cats {
		add(c.Key, c.Label, false)
	}
	for _, v := range customOrder {
		add(v.CategoryKey, v.CategoryLabel, true)
	}
}

// SharedItemOf describes the line that a shared category stands for when its vendor is locked (04 BE 2.2, step 2b):
// the budget line of the group (same session rule as the limit) when it exists and is not linked to a vendor, else the
// scenario line. ok is false when the category does not use a shared key. TotalAgreed is the sum of the offers and deals
// of the group, cancelled vendors left out.
func SharedItemOf(cats []CategorySpec, categoryKey, vendorScope string, scenario []ScenarioRow, lines []LineRow, vendors []VendorRow) (SharedItem, bool) {
	var spec CategorySpec
	for _, c := range cats {
		if c.Key == categoryKey {
			spec = c
		}
	}
	if len(spec.BudgetItemKeys) == 0 || !SharedKeys(cats)[spec.BudgetItemKeys[0]] {
		return SharedItem{}, false
	}
	key := spec.BudgetItemKeys[0]
	item := SharedItem{ItemKey: key, ItemName: spec.Label}
	fillSharedPlan(&item, vendorScope, scenario, lines)

	sharing := map[string]bool{}
	for _, c := range cats {
		if len(c.BudgetItemKeys) > 0 && c.BudgetItemKeys[0] == key {
			sharing[c.Key] = true
		}
	}
	for _, v := range vendors {
		if sharing[v.CategoryKey] && v.Status != VendorCancelled {
			item.TotalAgreed += v.AgreedPrice
		}
	}

	return item, true
}

// fillSharedPlan takes the name and the amount from the budget line of the group, or from the scenario line when there
// is no budget line yet; the session of the vendor goes before 'all'.
func fillSharedPlan(item *SharedItem, vendorScope string, scenario []ScenarioRow, lines []LineRow) {
	for _, scope := range []string{vendorScope, ScopeAll} {
		for _, l := range lines {
			if l.SubEventScope == scope && l.ItemKey == item.ItemKey && l.VendorID == nil {
				id := l.ID
				item.ItemName, item.BudgetItemID, item.Plan = l.ItemName, &id, l.Estimate

				return
			}
		}
		for _, r := range scenario {
			if r.SubEventScope == scope && r.ItemKey == item.ItemKey {
				item.ItemName, item.Plan = r.ItemName, r.Cost

				return
			}
		}
	}
}
