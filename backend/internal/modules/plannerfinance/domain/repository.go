package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// ErrNotFound, ErrInvalidScope and ErrConflict are the sentinels of the planner
// module (the shared TxManager already returns pd.ErrConflict for a failed
// COMMIT); they are reused so one mapping serves every planner module.
var (
	ErrNotFound     = pd.ErrNotFound
	ErrInvalidScope = pd.ErrInvalidScope
	ErrConflict     = pd.ErrConflict

	// ErrInvalidVendor: a line pointed at a vendor of another project (the composite foreign key).
	ErrInvalidVendor = errors.New("vendor does not belong to the project")
	// ErrVendorAlreadyLinked: the vendor is already the vendor of another line (planner_budget_items_vendor_idx).
	ErrVendorAlreadyLinked = errors.New("vendor is already linked to another budget line")
	// ErrKeyTaken: the (session, item_key) pair already exists (the partial unique indexes of lines and scenario items).
	ErrKeyTaken = errors.New("item_key already used in this session")
)

// SessionRef is the part of a session the checks and the export need.
type SessionRef struct {
	ScopeKey string
	Name     string
	IsSystem bool
}

// BudgetItem is the writable part of a planner_budget_items row. The payment
// columns, admin_doc_id and rollup_source are never written through it.
type BudgetItem struct {
	ID              uuid.UUID
	ProjectID       uuid.UUID
	SubEventScope   string
	Category        string
	ItemName        string
	ItemKey         *string
	Source          string
	VendorID        *uuid.UUID
	PricingMode     string
	QuantityX100    *int64 // quantity in hundredths (NUMERIC(12,2) as an integer)
	UnitLabel       *string
	UnitPrice       *int64
	InitialEstimate int64
	Notes           *string
	SortOrder       int
	Rollup          bool // read only: rollup_source is set
}

// AdjustRow pairs one scenario item with the budget line it lands on (03 BE 3.2).
type AdjustRow struct {
	Item     ScenarioItem
	BudgetID *uuid.UUID // nil = no line yet
	From     int64      // the line's initial_estimate
	Rollup   bool
	Unit     bool
}

// LineTotals are the aggregates of planner_budget_lines of one project.
type LineTotals struct {
	Estimate  int64
	Paid      int64
	Remaining int64 // Σ max(estimate − paid, 0)
	Overpaid  int64 // Σ max(paid − estimate, 0)
	Lines     int
	Status    LineStatus // lines with an estimate > 0
}

// TemplateStats describes the template lines replace_template would delete.
type TemplateStats struct {
	Deletable          int // template lines without a payment
	SkippedWithPayment int
	AtRisk             int // deletable lines that carry user work: an estimate, a vendor, a document or a shopping link
	Linked             int // deletable lines linked to a vendor, a document or the shopping list
}

// MonthCell is one cell of the monthly savings matrix.
type MonthCell struct {
	Month         string // "2026-02"
	ContributorID uuid.UUID
	Amount        int64
}

// SamplePayment is the realisation of one template line in the sample project.
type SamplePayment struct {
	Scope, Key                 string
	Estimate, DP, Term1, Term2 int64
}

// Repository is the PostgreSQL side of the module. Every method that takes a
// projectID filters by it; ids that come from a URL are always looked up with
// both (WHERE id = $x AND project_id = $y).
type Repository interface {
	// LockProject takes SELECT … FOR UPDATE on the project row. Needs a transaction.
	LockProject(ctx context.Context, projectID uuid.UUID) error
	Sessions(ctx context.Context, projectID uuid.UUID) ([]SessionRef, error)
	VendorInProject(ctx context.Context, projectID, vendorID uuid.UUID) (bool, error)
	// SetProjectTargets writes planner_projects.target_budget and/or clears savings_target (the third SQL exception of the contract).
	SetProjectTargets(ctx context.Context, projectID uuid.UUID, targetBudget *int64, clearSavings bool) error

	// ---------- scenarios ----------
	// ListScenarios returns every scenario with its items (items in sort_order), ordered by total_target.
	ListScenarios(ctx context.Context, projectID uuid.UUID) ([]Scenario, error)
	ScenarioByID(ctx context.Context, projectID, id uuid.UUID) (Scenario, error)
	CountScenarios(ctx context.Context, projectID uuid.UUID) (int, error)
	// InsertScenario stores the scenario and its items (sort_order = position). is_active is stored as given.
	InsertScenario(ctx context.Context, projectID uuid.UUID, sc *Scenario) error
	UpdateScenario(ctx context.Context, projectID, id uuid.UUID, name string, total int64, active []string) error
	DeleteScenario(ctx context.Context, projectID, id uuid.UUID) error
	// SetScenarioPreset records the preset a scenario was loaded from together with its total.
	SetScenarioPreset(ctx context.Context, projectID, id uuid.UUID, presetKey *string, total int64) error
	// ReplaceScenarioItems deletes the items of the scenario and inserts items (sort_order = position).
	ReplaceScenarioItems(ctx context.Context, projectID, scenarioID uuid.UUID, items []ScenarioItem) error
	CountScenarioItems(ctx context.Context, projectID, scenarioID uuid.UUID) (int, error)
	// InsertScenarioItem appends the item (sort_order = max + 1); ErrKeyTaken on a duplicate (session, item_key).
	InsertScenarioItem(ctx context.Context, projectID, scenarioID uuid.UUID, it *ScenarioItem) error
	ScenarioItemByID(ctx context.Context, projectID, scenarioID, itemID uuid.UUID) (ScenarioItem, error)
	// UpdateScenarioItem writes session, item_key, category, name and cost.
	UpdateScenarioItem(ctx context.Context, projectID, scenarioID uuid.UUID, it *ScenarioItem) error
	// ScenarioKeyTaken reports whether another item of the scenario already has (scope, key).
	ScenarioKeyTaken(ctx context.Context, scenarioID uuid.UUID, scope, key string, except uuid.UUID) (bool, error)
	DeleteScenarioItem(ctx context.Context, projectID, scenarioID, itemID uuid.UUID) error
	// DeactivateScenarios and ActivateScenario are two statements on purpose (03 BE 3.2): one UPDATE trips the partial unique index.
	DeactivateScenarios(ctx context.Context, projectID uuid.UUID) error
	ActivateScenario(ctx context.Context, projectID, id uuid.UUID) error
	ActiveScenario(ctx context.Context, projectID uuid.UUID) (id uuid.UUID, name string, total int64, found bool, err error)
	// AdjustRows pairs the items of the scenario that take part (included) with the project's lines.
	AdjustRows(ctx context.Context, projectID, scenarioID uuid.UUID) ([]AdjustRow, error)

	// ---------- budget lines ----------
	// LockBudgetItem takes FOR UPDATE on the base row (the view cannot be locked).
	LockBudgetItem(ctx context.Context, projectID, id uuid.UUID) error
	ListLines(ctx context.Context, projectID uuid.UUID, scope *string) ([]BudgetLine, error)
	LineByID(ctx context.Context, projectID, id uuid.UUID) (BudgetLine, error)
	BudgetItemByID(ctx context.Context, projectID, id uuid.UUID) (BudgetItem, error)
	CountLines(ctx context.Context, projectID uuid.UUID) (int, error)
	NextLineSortOrder(ctx context.Context, projectID uuid.UUID) (int, error)
	// InsertBudgetItem returns ErrKeyTaken, ErrInvalidVendor or ErrVendorAlreadyLinked.
	InsertBudgetItem(ctx context.Context, it *BudgetItem) error
	// UpdateBudgetItem writes scope, category, name, item_key, vendor, pricing and notes; payments, source, document and rollup stay.
	UpdateBudgetItem(ctx context.Context, it *BudgetItem) error
	// BudgetKeyTaken reports whether another line already has (scope, key).
	BudgetKeyTaken(ctx context.Context, projectID uuid.UUID, scope, key string, except uuid.UUID) (bool, error)
	UpdateEstimates(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID, amounts []int64) error
	// SetPayment writes one stage (stage is dp, term_1 or term_2); amount 0 with nil date and receipt clears it.
	SetPayment(ctx context.Context, projectID, id uuid.UUID, stage string, amount int64, paidAt *time.Time, receiptURL *string) error
	DeleteBudgetItem(ctx context.Context, projectID, id uuid.UUID) error
	DeleteAllBudgetItems(ctx context.Context, projectID uuid.UUID) error
	ReorderBudgetItems(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (changed int, err error)
	// InsertTemplateLine inserts unless (scope, item_key) already exists in the project.
	InsertTemplateLine(ctx context.Context, it *BudgetItem) (inserted bool, err error)
	TemplateStats(ctx context.Context, projectID uuid.UUID, scope *string) (TemplateStats, error)
	DeleteTemplateLines(ctx context.Context, projectID uuid.UUID, scope *string) (int, error)

	// ---------- aggregates ----------
	LineTotals(ctx context.Context, projectID uuid.UUID) (LineTotals, error)
	SavingsCollected(ctx context.Context, projectID uuid.UUID) (int64, error)
	MatrixRows(ctx context.Context, projectID uuid.UUID) ([]MatrixRow, error)

	// ---------- contributors and savings ----------
	ListContributors(ctx context.Context, projectID uuid.UUID) ([]Contributor, error)
	ContributorByID(ctx context.Context, projectID, id uuid.UUID) (Contributor, error)
	CountContributors(ctx context.Context, projectID uuid.UUID) (int, error)
	InsertContributor(ctx context.Context, projectID uuid.UUID, c *Contributor) error
	UpdateContributor(ctx context.Context, projectID uuid.UUID, c *Contributor) error
	DeleteContributor(ctx context.Context, projectID, id uuid.UUID) error
	ListEntries(ctx context.Context, projectID uuid.UUID, contributor *uuid.UUID) ([]SavingsEntry, error)
	EntryByID(ctx context.Context, projectID, id uuid.UUID) (SavingsEntry, error)
	CountEntries(ctx context.Context, projectID uuid.UUID) (int, error)
	InsertEntry(ctx context.Context, projectID uuid.UUID, e *SavingsEntry) error
	UpdateEntry(ctx context.Context, projectID uuid.UUID, e *SavingsEntry) error
	DeleteEntry(ctx context.Context, projectID, id uuid.UUID) error
	// Milestones lists the dated, non-system sessions in order with their FIFO share of the savings.
	Milestones(ctx context.Context, projectID uuid.UUID) ([]Milestone, error)
	// UnscheduledEstimate is Σ estimate of the lines in undated and system sessions.
	UnscheduledEstimate(ctx context.Context, projectID uuid.UUID) (int64, error)
	MonthCells(ctx context.Context, projectID uuid.UUID) ([]MonthCell, error)

	// ---------- sample dataset ----------
	// ApplySamplePayment sets the estimate and the payments of the template line (scope, key); ErrNotFound when it does not exist.
	ApplySamplePayment(ctx context.Context, projectID uuid.UUID, p SamplePayment) error
}
