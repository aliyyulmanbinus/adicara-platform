package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerfinance/domain"
)

func scenarioLimit() *apierr.Error {
	return apierr.Conflict("max_scenarios_reached", fmt.Sprintf("at most %d scenarios per project", maxScenarios))
}

// enrich fills the computed fields of a scenario read from the repository:
// Included per item (the session takes part in the simulation), Allocated (the
// sum of the included items) and PresetAmount (the value in the preset the
// scenario came from, matched by item_key).
func enrich(sc *domain.Scenario, eventType pd.EventType) {
	included := func(scope string) bool {
		return len(sc.ActiveSubEvents) == 0 || scope == pd.AllScope || contains(sc.ActiveSubEvents, scope)
	}

	var presetAmount map[string]int64
	if sc.PresetKey != nil {
		if items, fixed, ok := domain.PresetItems(eventType, *sc.PresetKey, sc.TotalTarget); ok && (fixed != nil || sc.TotalTarget >= domain.MinPercentTotal) {
			presetAmount = make(map[string]int64, len(items))
			for _, it := range items {
				presetAmount[it.Key] = it.Amount
			}
		}
	}

	sc.Allocated = 0
	for i := range sc.Items {
		it := &sc.Items[i]
		it.Included = included(it.SubEventScope)
		it.PresetAmount = nil
		if it.ItemKey != nil {
			if v, ok := presetAmount[*it.ItemKey]; ok {
				it.PresetAmount = &v
			}
		}
		if it.Included {
			sc.Allocated += it.CostAmount
		}
	}
}

// ListScenarios returns the scenarios (at most three, by total_target) with
// their items and the presets of the project's event type.
func (s *Service) ListScenarios(ctx context.Context, p pd.Project) (domain.ScenarioList, error) {
	list, err := s.repo.ListScenarios(ctx, p.ID)
	if err != nil {
		return domain.ScenarioList{}, err
	}
	for i := range list {
		enrich(&list[i], p.EventType)
	}

	return domain.ScenarioList{Scenarios: list, Presets: domain.Presets(p.EventType)}, nil
}

func (s *Service) GetScenario(ctx context.Context, p pd.Project, id uuid.UUID) (domain.Scenario, error) {
	return s.scenario(ctx, p, id)
}

func (s *Service) scenario(ctx context.Context, p pd.Project, id uuid.UUID) (domain.Scenario, error) {
	sc, err := s.repo.ScenarioByID(ctx, p.ID, id)
	if err != nil {
		return domain.Scenario{}, mapErr(err, "scenario")
	}
	enrich(&sc, p.EventType)

	return sc, nil
}

// validateActive checks active_sub_events: every element is a non-system
// session of the project; duplicates are dropped; the result is never nil.
func (s *Service) validateActive(ctx context.Context, projectID uuid.UUID, active []string) ([]string, error) {
	_, sessions, err := s.scopeSet(ctx, projectID)
	if err != nil {
		return nil, err
	}
	known := map[string]bool{}
	for _, sess := range sessions {
		if !sess.IsSystem {
			known[sess.ScopeKey] = true
		}
	}

	out := []string{}
	seen := map[string]bool{}
	for _, raw := range active {
		k := strings.TrimSpace(raw)
		if !known[k] {
			return nil, apierr.BadRequest("invalid_scope", "active_sub_events: unknown session")
		}
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}

	return out, nil
}

// presetPlan resolves a preset of the event type into its items and the total
// of the scenario. A fixed preset (wedding) has its own total, which total
// may repeat; a percent preset is sized by total, which is then required and
// at least MinPercentTotal.
func presetPlan(eventType pd.EventType, key string, total *int64) (items []domain.PresetItem, resolved int64, err error) {
	var kind string
	for _, info := range domain.Presets(eventType) {
		if info.Key == key {
			kind = info.Kind
		}
	}
	if kind == "" {
		return nil, 0, invalidTemplate("preset_key")
	}

	if kind == domain.KindFixed {
		var fixed *int64
		items, fixed, _ = domain.PresetItems(eventType, key, 0)
		if total != nil && *total != *fixed {
			return nil, 0, invalidInput("total_target", "must match the preset total")
		}

		return items, *fixed, nil
	}

	if total == nil {
		return nil, 0, invalidInput("total_target", "required")
	}
	if err := validateTotal("total_target", *total); err != nil {
		return nil, 0, err
	}
	if *total < domain.MinPercentTotal {
		return nil, 0, invalidInput("total_target", fmt.Sprintf("must be at least %d", domain.MinPercentTotal))
	}
	items, _, _ = domain.PresetItems(eventType, key, *total)

	return items, *total, nil
}

// itemsFromPreset turns preset items into scenario items. A session the project
// no longer has sends the item to the general session; the second return value
// counts those (warning scope_fallback).
func itemsFromPreset(items []domain.PresetItem, sessions map[string]bool) ([]domain.ScenarioItem, int) {
	out := make([]domain.ScenarioItem, len(items))
	fallback := 0
	for i, it := range items {
		scope := it.Scope
		if !sessions[scope] {
			scope = pd.AllScope
			fallback++
		}
		key := it.Key
		out[i] = domain.ScenarioItem{ID: uuid.New(), SubEventScope: scope, ItemKey: &key, Category: it.Category, ItemName: it.Name, CostAmount: it.Amount}
	}

	return out, fallback
}

func fallbackWarnings(n int) []pd.Notice {
	if n == 0 {
		return []pd.Notice{}
	}

	return []pd.Notice{notice("scope_fallback", "items of sessions the project no longer has were placed in the general session", n)}
}

// CreateScenario makes a scenario, empty or copied from a preset, under the
// project lock so two simultaneous requests cannot pass the limit of three.
func (s *Service) CreateScenario(ctx context.Context, p pd.Project, req domain.CreateScenarioRequest) (domain.ScenarioResult, error) {
	name, err := cleanText("name", req.Name, maxNameRunes, true)
	if err != nil {
		return domain.ScenarioResult{}, err
	}

	var presetKey *string
	if req.PresetKey != nil && strings.TrimSpace(*req.PresetKey) != "" {
		k := strings.TrimSpace(*req.PresetKey)
		presetKey = &k
	}

	var presetItems []domain.PresetItem
	var total int64
	if presetKey != nil {
		if presetItems, total, err = presetPlan(p.EventType, *presetKey, req.TotalTarget); err != nil {
			return domain.ScenarioResult{}, err
		}
	} else {
		if req.TotalTarget == nil {
			return domain.ScenarioResult{}, invalidInput("total_target", "required")
		}
		total = *req.TotalTarget
		if err := validateTotal("total_target", total); err != nil {
			return domain.ScenarioResult{}, err
		}
	}

	var res domain.ScenarioResult
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		sc, fallback, err := s.insertScenario(ctx, p, name, presetKey, total, presetItems, req.ActiveSubEvents)
		if err != nil {
			return err
		}
		res = domain.ScenarioResult{Scenario: sc, Warnings: fallbackWarnings(fallback)}

		return nil
	})
	if err != nil {
		return domain.ScenarioResult{}, mapErr(err, "scenario")
	}

	return res, nil
}

// insertScenario is the part of CreateScenario that runs under the project lock;
// the sample overlay calls it directly.
func (s *Service) insertScenario(ctx context.Context, p pd.Project, name string, presetKey *string, total int64, preset []domain.PresetItem, active []string) (domain.Scenario, int, error) {
	n, err := s.repo.CountScenarios(ctx, p.ID)
	if err != nil {
		return domain.Scenario{}, 0, err
	}
	if n >= maxScenarios {
		return domain.Scenario{}, 0, scenarioLimit()
	}
	activeSessions, err := s.validateActive(ctx, p.ID, active)
	if err != nil {
		return domain.Scenario{}, 0, err
	}
	set, _, err := s.scopeSet(ctx, p.ID)
	if err != nil {
		return domain.Scenario{}, 0, err
	}

	items, fallback := itemsFromPreset(preset, set)
	sc := domain.Scenario{ID: uuid.New(), Name: name, TotalTarget: total, ActiveSubEvents: activeSessions, PresetKey: presetKey, Items: items}
	if err := s.repo.InsertScenario(ctx, p.ID, &sc); err != nil {
		return domain.Scenario{}, 0, err
	}
	enrich(&sc, p.EventType)

	return sc, fallback, nil
}

// UpdateScenario replaces name, total_target and active_sub_events. Changing
// the total does not scale the items.
func (s *Service) UpdateScenario(ctx context.Context, p pd.Project, id uuid.UUID, req domain.UpdateScenarioRequest) (domain.Scenario, error) {
	name, err := cleanText("name", req.Name, maxNameRunes, true)
	if err != nil {
		return domain.Scenario{}, err
	}
	if err := validateTotal("total_target", req.TotalTarget); err != nil {
		return domain.Scenario{}, err
	}
	active, err := s.validateActive(ctx, p.ID, req.ActiveSubEvents)
	if err != nil {
		return domain.Scenario{}, err
	}
	if err := s.repo.UpdateScenario(ctx, p.ID, id, name, req.TotalTarget, active); err != nil {
		return domain.Scenario{}, mapErr(err, "scenario")
	}

	return s.scenario(ctx, p, id)
}

// DeleteScenario removes a scenario and its items, also the active one; no
// other scenario is activated in its place (03 BE 3.1).
func (s *Service) DeleteScenario(ctx context.Context, p pd.Project, id uuid.UUID) error {
	return mapErr(s.repo.DeleteScenario(ctx, p.ID, id), "scenario")
}

// CloneScenario copies a scenario (items, active sessions, preset and total)
// as an inactive one; it counts toward the limit of three.
func (s *Service) CloneScenario(ctx context.Context, p pd.Project, id uuid.UUID, req domain.CloneScenarioRequest) (domain.ScenarioResult, error) {
	name, err := cleanText("new_name", req.NewName, maxNameRunes, true)
	if err != nil {
		return domain.ScenarioResult{}, err
	}

	var res domain.ScenarioResult
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		n, err := s.repo.CountScenarios(ctx, p.ID)
		if err != nil {
			return err
		}
		if n >= maxScenarios {
			return scenarioLimit()
		}
		src, err := s.repo.ScenarioByID(ctx, p.ID, id)
		if err != nil {
			return err
		}

		items := make([]domain.ScenarioItem, len(src.Items))
		for i, it := range src.Items {
			it.ID = uuid.New()
			items[i] = it
		}
		cp := domain.Scenario{ID: uuid.New(), Name: name, TotalTarget: src.TotalTarget, ActiveSubEvents: src.ActiveSubEvents, PresetKey: src.PresetKey, Items: items}
		if err := s.repo.InsertScenario(ctx, p.ID, &cp); err != nil {
			return err
		}
		enrich(&cp, p.EventType)
		res = domain.ScenarioResult{Scenario: cp, Warnings: []pd.Notice{}}

		return nil
	})
	if err != nil {
		return domain.ScenarioResult{}, mapErr(err, "scenario")
	}

	return res, nil
}

// LoadPreset replaces the items of a scenario with those of a preset and
// records the preset and its total on the scenario. A scenario that already has
// items needs confirm.
func (s *Service) LoadPreset(ctx context.Context, p pd.Project, id uuid.UUID, req domain.LoadPresetRequest) (domain.ScenarioResult, error) {
	key := strings.TrimSpace(req.PresetKey)
	if key == "" {
		return domain.ScenarioResult{}, invalidInput("preset_key", "required")
	}

	var res domain.ScenarioResult
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		sc, err := s.repo.ScenarioByID(ctx, p.ID, id)
		if err != nil {
			return err
		}

		total := req.TotalTarget
		if total == nil && p.EventType != pd.EventWedding {
			total = &sc.TotalTarget // a percent preset keeps the scenario's own total unless a new one is sent
		}
		preset, resolved, err := presetPlan(p.EventType, key, total)
		if err != nil {
			return err
		}
		if len(sc.Items) > 0 && !req.Confirm {
			return confirmRequired()
		}

		set, _, err := s.scopeSet(ctx, p.ID)
		if err != nil {
			return err
		}
		items, fallback := itemsFromPreset(preset, set)
		if err := s.repo.ReplaceScenarioItems(ctx, p.ID, id, items); err != nil {
			return err
		}
		if err := s.repo.SetScenarioPreset(ctx, p.ID, id, &key, resolved); err != nil {
			return err
		}

		sc.PresetKey, sc.TotalTarget, sc.Items = &key, resolved, items
		enrich(&sc, p.EventType)
		res = domain.ScenarioResult{Scenario: sc, Warnings: fallbackWarnings(fallback)}

		return nil
	})
	if err != nil {
		return domain.ScenarioResult{}, mapErr(err, "scenario")
	}

	return res, nil
}

// ---------- scenario items ----------

func (s *Service) validateScenarioItem(ctx context.Context, p pd.Project, scope, category, name string, cost int64) (domain.ScenarioItem, error) {
	sc, err := normalizeScope(scope)
	if err != nil {
		return domain.ScenarioItem{}, err
	}
	cat, err := cleanText("category", category, maxCategoryRunes, true)
	if err != nil {
		return domain.ScenarioItem{}, err
	}
	nm, err := cleanText("item_name", name, maxItemNameRunes, true)
	if err != nil {
		return domain.ScenarioItem{}, err
	}
	if err := validateMoney("cost_amount", cost); err != nil {
		return domain.ScenarioItem{}, err
	}
	if err := s.checkScope(ctx, p.ID, sc); err != nil {
		return domain.ScenarioItem{}, err
	}

	return domain.ScenarioItem{SubEventScope: sc, Category: cat, ItemName: nm, CostAmount: cost}, nil
}

// CreateScenarioItem adds an item to a scenario (at most 100), under the project lock.
func (s *Service) CreateScenarioItem(ctx context.Context, p pd.Project, scenarioID uuid.UUID, req domain.ScenarioItemRequest) (domain.ScenarioItem, error) {
	key, err := cleanKey("item_key", req.ItemKey)
	if err != nil {
		return domain.ScenarioItem{}, err
	}

	var created domain.ScenarioItem
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		it, err := s.validateScenarioItem(ctx, p, req.SubEventScope, req.Category, req.ItemName, req.CostAmount)
		if err != nil {
			return err
		}
		it.ID, it.ItemKey = uuid.New(), key

		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		sc, err := s.repo.ScenarioByID(ctx, p.ID, scenarioID)
		if err != nil {
			return err
		}
		n, err := s.repo.CountScenarioItems(ctx, p.ID, scenarioID)
		if err != nil {
			return err
		}
		if n >= maxScenarioItems {
			return apierr.Conflict("scenario_item_limit_reached", fmt.Sprintf("at most %d items per scenario", maxScenarioItems))
		}
		if err := s.repo.InsertScenarioItem(ctx, p.ID, scenarioID, &it); err != nil {
			return err
		}

		sc.Items = []domain.ScenarioItem{it}
		enrich(&sc, p.EventType)
		created = sc.Items[0]

		return nil
	})
	if err != nil {
		return domain.ScenarioItem{}, mapErr(err, "scenario")
	}

	return created, nil
}

// UpdateScenarioItem replaces the editable fields of an item. item_key never
// changes; moving the item to a session that already has the same key makes it
// a keyless item (the rule of 01 BE 4.6).
func (s *Service) UpdateScenarioItem(ctx context.Context, p pd.Project, scenarioID, itemID uuid.UUID, req domain.UpdateScenarioItemRequest) (domain.ScenarioItem, error) {
	it, err := s.validateScenarioItem(ctx, p, req.SubEventScope, req.Category, req.ItemName, req.CostAmount)
	if err != nil {
		return domain.ScenarioItem{}, err
	}

	var out domain.ScenarioItem
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		sc, err := s.repo.ScenarioByID(ctx, p.ID, scenarioID)
		if err != nil {
			return err
		}
		cur, err := s.repo.ScenarioItemByID(ctx, p.ID, scenarioID, itemID)
		if err != nil {
			return err
		}

		it.ID, it.ItemKey = cur.ID, cur.ItemKey
		if cur.ItemKey != nil && it.SubEventScope != cur.SubEventScope {
			taken, err := s.repo.ScenarioKeyTaken(ctx, scenarioID, it.SubEventScope, *cur.ItemKey, cur.ID)
			if err != nil {
				return err
			}
			if taken {
				it.ItemKey = nil
			}
		}
		if err := s.repo.UpdateScenarioItem(ctx, p.ID, scenarioID, &it); err != nil {
			return err
		}

		sc.Items = []domain.ScenarioItem{it}
		enrich(&sc, p.EventType)
		out = sc.Items[0]

		return nil
	})
	if err != nil {
		return domain.ScenarioItem{}, mapErr(err, "scenario item")
	}

	return out, nil
}

func (s *Service) DeleteScenarioItem(ctx context.Context, p pd.Project, scenarioID, itemID uuid.UUID) error {
	return mapErr(s.repo.DeleteScenarioItem(ctx, p.ID, scenarioID, itemID), "scenario item")
}

// ---------- compare ----------

func pairKey(it domain.ScenarioItem) string {
	if it.ItemKey != nil {
		return it.SubEventScope + "|k|" + *it.ItemKey
	}

	return it.SubEventScope + "|n|" + strings.ToLower(it.ItemName)
}

// CompareScenarios lays two scenarios side by side. Items are paired by
// (session, item_key), and keyless items by (session, lower(name)) like the
// Smart Adjust pairing; the rows follow the order of A, then the items only B
// has. delta is b − a.
func (s *Service) CompareScenarios(ctx context.Context, p pd.Project, req domain.CompareRequest) (domain.ScenarioComparison, error) {
	if req.A == uuid.Nil {
		return domain.ScenarioComparison{}, invalidInput("a", "required")
	}
	if req.B == uuid.Nil {
		return domain.ScenarioComparison{}, invalidInput("b", "required")
	}
	if req.A == req.B {
		return domain.ScenarioComparison{}, invalidInput("b", "must differ from a")
	}
	a, err := s.repo.ScenarioByID(ctx, p.ID, req.A)
	if err != nil {
		return domain.ScenarioComparison{}, mapErr(err, "scenario")
	}
	b, err := s.repo.ScenarioByID(ctx, p.ID, req.B)
	if err != nil {
		return domain.ScenarioComparison{}, mapErr(err, "scenario")
	}

	return compare(a, b), nil
}

func compare(a, b domain.Scenario) domain.ScenarioComparison {
	out := domain.ScenarioComparison{
		A:    domain.CompareSide{ID: a.ID, Name: a.Name, TotalTarget: a.TotalTarget},
		B:    domain.CompareSide{ID: b.ID, Name: b.Name, TotalTarget: b.TotalTarget},
		Rows: []domain.CompareRow{},
	}

	index := map[string]int{}
	add := func(it domain.ScenarioItem, isB bool) {
		k := pairKey(it)
		i, ok := index[k]
		if !ok {
			index[k] = len(out.Rows)
			i = len(out.Rows)
			out.Rows = append(out.Rows, domain.CompareRow{SubEventScope: it.SubEventScope, ItemKey: it.ItemKey, ItemName: it.ItemName})
		}
		if isB {
			out.Rows[i].BAmount += it.CostAmount
		} else {
			out.Rows[i].AAmount += it.CostAmount
		}
	}
	for _, it := range a.Items {
		add(it, false)
	}
	for _, it := range b.Items {
		add(it, true)
	}
	for i := range out.Rows {
		r := &out.Rows[i]
		r.Delta = r.BAmount - r.AAmount
		out.Totals.A += r.AAmount
		out.Totals.B += r.BAmount
	}
	out.Totals.Delta = out.Totals.B - out.Totals.A

	return out
}
