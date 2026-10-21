// Package domain holds the entities, requests, templates and repository
// interface of planner menu 03 (finance: scenarios, savings, budget lines).
// Money is int64 whole rupiah and calendar dates (DATE) travel as
// "YYYY-MM-DD". The structs below are the JSON of 03 BE chapter 5 verbatim:
// snake_case keys, optional keys are pointers with omitempty, lists are never
// null, no percentage is ever sent, and there are no binding tags (validation
// lives in the usecase so a blank string reaches it and gets a field-specific
// error).
package domain

import (
	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// Apply modes of POST /scenarios/apply and /scenarios/apply-preview.
const (
	ApplySmartAdjust    = "smart_adjust"
	ApplyCleanOverwrite = "clean_overwrite"
)

// Template modes of POST /budget/apply-template.
const (
	ModeAppend          = "append"
	ModeReplaceTemplate = "replace_template"
)

// ConfirmReset is the text clean_overwrite must be confirmed with.
const ConfirmReset = "RESET"

// Budget line sources (planner_budget_items.source).
const (
	SourceTemplate = "template"
	SourceScenario = "scenario"
	SourceVendor   = "vendor"
	SourceAdminDoc = "admin_doc"
	SourceManual   = "manual"
)

const (
	PricingLump = "lump"
	PricingUnit = "unit"
)

// Payment stages. StageFull is not a column: it fills the first empty stage with the remainder.
const (
	StageDP    = "dp"
	StageTerm1 = "term_1"
	StageTerm2 = "term_2"
	StageFull  = "full"
)

// Payment statuses, computed by the view planner_budget_lines.
const (
	StatusUnpaid  = "unpaid"
	StatusDPPaid  = "dp_paid"
	StatusPartial = "partial"
	StatusSettled = "settled"
)

const RollupShopping = "shopping"

// Preset kinds.
const (
	KindFixed   = "fixed"
	KindPercent = "percent"
)

// Skip reasons of ApplyReport.
const (
	SkipRollup      = "rollup"
	SkipUnitPricing = "unit_pricing"
)

// ---------- Tab 1: scenarios ----------

type ScenarioItem struct {
	ID            uuid.UUID `json:"id"`
	SubEventScope string    `json:"sub_event_scope"`
	ItemKey       *string   `json:"item_key,omitempty"`
	Category      string    `json:"category"`
	ItemName      string    `json:"item_name"`
	CostAmount    int64     `json:"cost_amount"`
	Included      bool      `json:"included"`                // computed: the session takes part in the simulation (active_sub_events empty, 'all' or listed)
	PresetAmount  *int64    `json:"preset_amount,omitempty"` // the value in the scenario's preset (same item_key); the basis of "back to the preset value"
}

type Scenario struct {
	ID              uuid.UUID      `json:"id"`
	Name            string         `json:"name"`
	TotalTarget     int64          `json:"total_target"`
	IsActive        bool           `json:"is_active"`
	ActiveSubEvents []string       `json:"active_sub_events"` // scope_key; empty = every non-system session
	PresetKey       *string        `json:"preset_key,omitempty"`
	Allocated       int64          `json:"allocated"` // Σ cost_amount of the included items
	Items           []ScenarioItem `json:"items"`
}

// PresetInfo: presets live in code. total_target is sent for kind "fixed" (wedding) only; "percent" uses the user's total.
type PresetInfo struct {
	Key         string `json:"key"`
	Kind        string `json:"kind"` // fixed | percent
	TotalTarget *int64 `json:"total_target,omitempty"`
}

type ScenarioList struct {
	Scenarios []Scenario   `json:"scenarios"`
	Presets   []PresetInfo `json:"presets"`
}

type ScenarioResult struct {
	Scenario Scenario    `json:"scenario"`
	Warnings []pd.Notice `json:"warnings"` // scope_fallback
}

type CreateScenarioRequest struct {
	Name            string   `json:"name"`
	TotalTarget     *int64   `json:"total_target"`
	PresetKey       *string  `json:"preset_key"`
	ActiveSubEvents []string `json:"active_sub_events"`
}

type UpdateScenarioRequest struct { // PUT: replaces the whole scenario header
	Name            string   `json:"name"`
	TotalTarget     int64    `json:"total_target"`
	ActiveSubEvents []string `json:"active_sub_events"`
}

type CloneScenarioRequest struct {
	NewName string `json:"new_name"`
}

type LoadPresetRequest struct {
	PresetKey   string `json:"preset_key"`
	TotalTarget *int64 `json:"total_target"`
	Confirm     bool   `json:"confirm"` // required when the scenario already has items
}

type ScenarioItemRequest struct { // POST
	SubEventScope string  `json:"sub_event_scope"`
	ItemKey       *string `json:"item_key"`
	Category      string  `json:"category"`
	ItemName      string  `json:"item_name"`
	CostAmount    int64   `json:"cost_amount"`
}

type UpdateScenarioItemRequest struct { // PUT: item_key does not change (sending it is 400 invalid_input: item_key)
	SubEventScope string `json:"sub_event_scope"`
	Category      string `json:"category"`
	ItemName      string `json:"item_name"`
	CostAmount    int64  `json:"cost_amount"`
}

type CompareRequest struct {
	A uuid.UUID `json:"a"`
	B uuid.UUID `json:"b"`
}

type CompareRow struct {
	SubEventScope string  `json:"sub_event_scope"`
	ItemKey       *string `json:"item_key,omitempty"`
	ItemName      string  `json:"item_name"`
	AAmount       int64   `json:"a_amount"`
	BAmount       int64   `json:"b_amount"`
	Delta         int64   `json:"delta"` // b − a
}

type CompareSide struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	TotalTarget int64     `json:"total_target"`
}

type CompareTotals struct {
	A     int64 `json:"a"`
	B     int64 `json:"b"`
	Delta int64 `json:"delta"`
}

type ScenarioComparison struct {
	A      CompareSide   `json:"a"`
	B      CompareSide   `json:"b"`
	Rows   []CompareRow  `json:"rows"`
	Totals CompareTotals `json:"totals"`
}

type ApplyScenarioRequest struct {
	ScenarioID        uuid.UUID `json:"scenario_id"`
	Mode              string    `json:"mode"` // smart_adjust | clean_overwrite
	SyncTargetBudget  bool      `json:"sync_target_budget"`
	SyncTargetSavings bool      `json:"sync_target_savings"` // savings_target := NULL (follows the budget again)
	Confirm           *string   `json:"confirm"`             // "RESET" is required for clean_overwrite
}

type AppliedUpdate struct {
	SubEventScope string  `json:"sub_event_scope"`
	ItemKey       *string `json:"item_key,omitempty"`
	ItemName      string  `json:"item_name"`
	From          int64   `json:"from"`
	To            int64   `json:"to"`
}

type AppliedCreate struct {
	SubEventScope string  `json:"sub_event_scope"`
	ItemKey       *string `json:"item_key,omitempty"`
	ItemName      string  `json:"item_name"`
	Amount        int64   `json:"amount"`
}

type AppliedSkip struct {
	SubEventScope string  `json:"sub_event_scope"`
	ItemKey       *string `json:"item_key,omitempty"`
	ItemName      string  `json:"item_name"`
	Reason        string  `json:"reason"` // rollup | unit_pricing
}

type AppliedDeleted struct {
	Lines     int   `json:"lines"`
	PaidTotal int64 `json:"paid_total"`
}

// ApplyReport is the same shape for apply-preview (nothing written) and apply (what was done).
type ApplyReport struct {
	Mode            string          `json:"mode"`
	Updated         []AppliedUpdate `json:"updated"`
	Created         []AppliedCreate `json:"created"`
	Skipped         []AppliedSkip   `json:"skipped"`
	Deleted         *AppliedDeleted `json:"deleted,omitempty"` // clean_overwrite only
	UntouchedCount  int             `json:"untouched_count"`
	EstimateAfter   int64           `json:"estimate_after"`
	NewTargetBudget *int64          `json:"new_target_budget,omitempty"` // present when sync_target_budget
}

// ---------- Tab 2: cash and savings ----------

type Contributor struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	RoleLabel  string    `json:"role_label"`
	Total      int64     `json:"total"`
	EntryCount int       `json:"entry_count"`
}

type ContributorRequest struct { // POST and PUT
	Name      string  `json:"name"`
	RoleLabel *string `json:"role_label"` // default "other"
}

type SavingsEntry struct {
	ID              uuid.UUID `json:"id"`
	ContributorID   uuid.UUID `json:"contributor_id"`
	ContributorName string    `json:"contributor_name"`
	RoleLabel       string    `json:"role_label"`
	EntryDate       string    `json:"entry_date"` // YYYY-MM-DD
	Amount          int64     `json:"amount"`
	Notes           *string   `json:"notes,omitempty"`
}

type SavingsEntryRequest struct { // POST and PUT
	ContributorID uuid.UUID `json:"contributor_id"`
	EntryDate     string    `json:"entry_date"`
	Amount        int64     `json:"amount"`
	Notes         *string   `json:"notes"`
}

type Milestone struct {
	SubEventScope string `json:"sub_event_scope"`
	Name          string `json:"name"`
	SessionDate   string `json:"session_date"`
	TargetAmount  int64  `json:"target_amount"`
	SecuredAmount int64  `json:"secured_amount"`
	IsSecured     bool   `json:"is_secured"`
}

type SavingsMonth struct {
	Month         string           `json:"month"`          // "2026-02"
	ByContributor map[string]int64 `json:"by_contributor"` // key = contributor id; every contributor is always present (0 when empty)
	Total         int64            `json:"total"`
}

type SavingsKPI struct {
	Target              int64 `json:"target"` // COALESCE(savings_target, target_budget)
	Collected           int64 `json:"collected"`
	Shortfall           int64 `json:"shortfall"`
	TargetFollowsBudget bool  `json:"target_follows_budget"` // savings_target IS NULL
}

type Recommendation struct {
	MonthsLeft         int    `json:"months_left"`
	RecommendedMonthly int64  `json:"recommended_monthly"`
	PerContributor     *int64 `json:"per_contributor,omitempty"` // omitted while the project has no contributor
	TargetDate         string `json:"target_date"`
	Basis              string `json:"basis"` // session | event_date
}

type SavingsMatrix struct {
	Months []string       `json:"months"`
	Rows   []SavingsMonth `json:"rows"`
}

type SavingsOverview struct {
	KPI                 SavingsKPI     `json:"kpi"`
	Contributors        []Contributor  `json:"contributors"`
	Milestones          []Milestone    `json:"milestones"`
	UnscheduledEstimate int64          `json:"unscheduled_estimate"`
	Recommendation      Recommendation `json:"recommendation"`
	Matrix              SavingsMatrix  `json:"matrix"`
	Entries             []SavingsEntry `json:"entries"` // filtered by ?contributor_id=; every other part is the whole project
}

// ---------- Tab 3: realisation ----------

// BudgetLine is one row of planner_budget_lines (effective figures). The table
// columns paid_total and remaining_balance are NOT sent: they are wrong for a rollup line.
type BudgetLine struct {
	ID              uuid.UUID  `json:"id"`
	SubEventScope   string     `json:"sub_event_scope"`
	Category        string     `json:"category"`
	ItemName        string     `json:"item_name"`
	ItemKey         *string    `json:"item_key,omitempty"`
	Source          string     `json:"source"` // template | scenario | vendor | admin_doc | manual
	VendorID        *uuid.UUID `json:"vendor_id,omitempty"`
	VendorName      *string    `json:"vendor_name,omitempty"`
	AdminDocID      *uuid.UUID `json:"admin_doc_id,omitempty"`
	RollupSource    *string    `json:"rollup_source,omitempty"` // "shopping": estimate and paid come from the shopping list
	PricingMode     string     `json:"pricing_mode"`            // lump | unit
	Quantity        *float64   `json:"quantity,omitempty"`
	UnitLabel       *string    `json:"unit_label,omitempty"`
	UnitPrice       *int64     `json:"unit_price,omitempty"`
	Estimate        int64      `json:"estimate"` // effective (from the shopping list for a rollup line)
	DPAmount        int64      `json:"dp_amount"`
	Term1Amount     int64      `json:"term_1_amount"`
	Term2Amount     int64      `json:"term_2_amount"`
	DPPaidAt        *string    `json:"dp_paid_at,omitempty"` // YYYY-MM-DD
	Term1PaidAt     *string    `json:"term_1_paid_at,omitempty"`
	Term2PaidAt     *string    `json:"term_2_paid_at,omitempty"`
	DPReceiptURL    *string    `json:"dp_receipt_url,omitempty"`
	Term1ReceiptURL *string    `json:"term_1_receipt_url,omitempty"`
	Term2ReceiptURL *string    `json:"term_2_receipt_url,omitempty"`
	Paid            int64      `json:"paid"`
	Remaining       int64      `json:"remaining"`      // < 0 = overpaid
	PaymentStatus   string     `json:"payment_status"` // unpaid | dp_paid | partial | settled
	SortOrder       int        `json:"sort_order"`
	Notes           *string    `json:"notes,omitempty"`
}

type BudgetLineList struct {
	Items []BudgetLine `json:"items"`
}

// BudgetItemRequest: POST and PUT. PUT replaces the whole line: an omitted vendor_id RELEASES the vendor.
type BudgetItemRequest struct {
	SubEventScope   string     `json:"sub_event_scope"`
	Category        string     `json:"category"`
	ItemName        string     `json:"item_name"`
	VendorID        *uuid.UUID `json:"vendor_id"`
	PricingMode     *string    `json:"pricing_mode"`     // default lump
	InitialEstimate *int64     `json:"initial_estimate"` // ignored when pricing_mode is unit or the line is a rollup
	Quantity        *float64   `json:"quantity"`
	UnitLabel       *string    `json:"unit_label"`
	UnitPrice       *int64     `json:"unit_price"`
	Notes           *string    `json:"notes"`
}

type PaymentRequest struct {
	PaymentStage string  `json:"payment_stage"` // dp | term_1 | term_2 | full
	Amount       *int64  `json:"amount"`        // required and > 0 except for full (ignored)
	PaidAt       string  `json:"paid_at"`       // YYYY-MM-DD
	ReceiptURL   *string `json:"receipt_url"`   // http/https; omitted = the receipt of that stage is cleared
}

type ApplyTemplateRequest struct {
	Mode          string  `json:"mode"` // append | replace_template
	SubEventScope *string `json:"sub_event_scope"`
	Confirm       bool    `json:"confirm"`
}

type ApplyTemplateResult struct {
	Created  int         `json:"created"`
	Skipped  int         `json:"skipped"`
	Deleted  int         `json:"deleted"`
	Warnings []pd.Notice `json:"warnings"`
}

type CashflowSummary struct {
	TotalSavingsCollected int64 `json:"total_savings_collected"`
	TotalPaid             int64 `json:"total_paid"`
	AvailableCash         int64 `json:"available_cash"` // collected − paid; may be negative
	RemainingDebt         int64 `json:"remaining_debt"` // Σ max(estimate − paid, 0)
	Overpaid              int64 `json:"overpaid"`       // Σ max(paid − estimate, 0)
	CashDeficit           int64 `json:"cash_deficit"`   // max(remaining_debt − available_cash, 0)
}

type MatrixRow struct {
	SubEventScope string `json:"sub_event_scope"` // "" on the total row
	Estimate      int64  `json:"estimate"`
	DP            int64  `json:"dp"`
	Term1         int64  `json:"term_1"`
	Term2         int64  `json:"term_2"`
	RollupPaid    int64  `json:"rollup_paid"`
	Paid          int64  `json:"paid"` // dp + term_1 + term_2 + rollup_paid
	Remaining     int64  `json:"remaining"`
	Overpaid      int64  `json:"overpaid"`
	LineCount     int    `json:"line_count"`
}

// LineStatus is FinanceSummary.line_status of menu 01.
type LineStatus = pd.LineStatus

type MatrixKPI struct {
	TotalEstimate  int64      `json:"total_estimate"`
	TotalPaid      int64      `json:"total_paid"`
	TotalRemaining int64      `json:"total_remaining"`
	TotalOverpaid  int64      `json:"total_overpaid"`
	LineCount      int        `json:"line_count"`
	LineStatus     LineStatus `json:"line_status"` // lines with an estimate > 0 only
}

type ActiveScenario struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	TotalTarget int64     `json:"total_target"`
	Unallocated int64     `json:"unallocated"` // total_target − Σ estimate of every line; may be negative
}

type BudgetMatrix struct {
	KPI            MatrixKPI       `json:"kpi"`
	Rows           []MatrixRow     `json:"rows"` // one per session that has lines, in session order, Umum last
	Total          MatrixRow       `json:"total"`
	ActiveScenario *ActiveScenario `json:"active_scenario"` // null when no scenario is active
}
