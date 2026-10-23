// Package domain holds the entities, the pure rules and the repository contract of
// planner menu 04: the vendor pipeline and the shopping list. Money is int64 whole
// rupiah; shopping quantities travel as hundredths (NUMERIC(12,2) as an integer) in
// the repository and as a float with two decimals in JSON.
package domain

import (
	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// Conventions (same as menus 01-03): json tags are snake_case; an optional key is a
// pointer + omitempty (left out when empty, the frontend's `?:`); lists are always
// arrays, never null; NO `binding` tags (the usecase validates and answers
// "<field>: <reason>"); money is int64 whole rupiah; no percentage is ever sent
// (the client divides, 00 contract chapter 6).

// ----- Tab 1: vendors -----

type VendorStatus string

const (
	VendorPlanned     VendorStatus = "rencana"
	VendorNegotiating VendorStatus = "negosiasi"
	VendorFixed       VendorStatus = "fix"
	VendorCancelled   VendorStatus = "batal"
)

// Valid reports whether s is one of the four stored statuses.
func (s VendorStatus) Valid() bool {
	switch s {
	case VendorPlanned, VendorNegotiating, VendorFixed, VendorCancelled:
		return true
	}

	return false
}

// Where the limit of a vendor comes from (04 BE 2.1).
const (
	LimitSourceManual   = "manual"   // planner_vendors.budget_limit > 0
	LimitSourceScenario = "scenario" // one line of the active scenario, not shared with another category
	LimitSourceGroup    = "group"    // a scenario line shared by several categories: compared in a LimitGroup
	LimitSourceNone     = "none"     // no limit (a category without a line, or the active scenario lacks it)
)

// LinePayment is the effective figures of one budget line (the view planner_budget_lines), read from the
// line that points at the vendor (planner_budget_items.vendor_id) or, for LimitGroup.Line, from the shared line.
type LinePayment struct {
	BudgetItemID  uuid.UUID `json:"budget_item_id"`
	ItemName      string    `json:"item_name"`
	Estimate      int64     `json:"estimate"` // effective (from the shopping list when the line rolls up)
	Paid          int64     `json:"paid"`
	Remaining     int64     `json:"remaining"`      // < 0 = overpaid
	PaymentStatus string    `json:"payment_status"` // unpaid | dp_paid | partial | settled
}

type Vendor struct {
	ID                uuid.UUID    `json:"id"`
	SubEventScope     string       `json:"sub_event_scope"`
	CategoryKey       string       `json:"category_key"`
	CategoryLabel     string       `json:"category_label"`
	VendorName        string       `json:"vendor_name"`
	PICName           *string      `json:"pic_name,omitempty"`
	PhoneNumber       *string      `json:"phone_number,omitempty"`
	FormattedWANumber *string      `json:"formatted_wa_number,omitempty"` // 62…; the wa.me link is built by the client
	PortfolioURL      *string      `json:"portfolio_url,omitempty"`
	BudgetLimit       int64        `json:"budget_limit"`          // effective limit; 0 when LimitSource is group or none
	LimitSource       string       `json:"limit_source"`          // manual | scenario | group | none
	LimitGroup        *string      `json:"limit_group,omitempty"` // item_key of the group when LimitSource is group
	AgreedPrice       int64        `json:"agreed_price"`          // 0 = no offer yet
	PriceDelta        *int64       `json:"price_delta,omitempty"` // BudgetLimit − AgreedPrice (positive = saving); absent when either is 0 or LimitSource is group/none
	Status            VendorStatus `json:"status"`
	Notes             *string      `json:"notes,omitempty"`
	SortOrder         int          `json:"sort_order"`
	Payment           *LinePayment `json:"payment,omitempty"` // the line that points at this vendor; absent when none does
}

// LimitGroup compares the limit of one scenario line that several vendor categories share
// (wedding: venue_dekor, rias_busana, hiburan) with what their vendors ask.
type LimitGroup struct {
	ItemKey       string       `json:"item_key"`
	SubEventScope string       `json:"sub_event_scope"` // session of the scenario line (may be 'all')
	Limit         int64        `json:"limit"`
	TotalAgreed   int64        `json:"total_agreed"`    // Σ AgreedPrice of the members that are not cancelled (offers and deals)
	Delta         *int64       `json:"delta,omitempty"` // Limit − TotalAgreed; absent while no member has an offer
	Members       int          `json:"members"`
	Priced        int          `json:"priced"` // members with AgreedPrice > 0
	VendorIDs     []uuid.UUID  `json:"vendor_ids"`
	Line          *LinePayment `json:"line,omitempty"` // the shared budget line, when there is one
}

type CategoryState struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	State   string `json:"state"`   // fix | pipeline | empty
	Vendors int    `json:"vendors"` // not cancelled
	Custom  bool   `json:"custom"`  // not in the registry of the event type
	Skipped bool   `json:"skipped"` // listed in planner_projects.skipped_vendor_categories
}

type VendorKPI struct {
	CategoriesTotal int   `json:"categories_total"` // registry ∪ custom categories in use, without the skipped ones
	CategoriesFixed int   `json:"categories_fixed"` // of those, with at least one fixed vendor
	Skipped         int   `json:"skipped"`
	FixedVendors    int   `json:"fixed_vendors"`
	Pipeline        int   `json:"pipeline"` // planned + negotiating
	FixedTotal      int64 `json:"fixed_total"`
	PipelineOffers  int64 `json:"pipeline_offers"` // Σ AgreedPrice (> 0) of planned and negotiating vendors
	GroupsTotal     int   `json:"groups_total"`
	GroupsOver      int   `json:"groups_over"` // groups with Delta < 0
}

type VendorOverview struct {
	KPI         VendorKPI       `json:"kpi"`          // always the whole project
	Categories  []CategoryState `json:"categories"`   // idem
	LimitGroups []LimitGroup    `json:"limit_groups"` // idem
	Vendors     []Vendor        `json:"vendors"`      // filtered by ?scope=&status=&category=
}

// VendorRequest is the body of POST and PUT (PUT replaces the vendor entirely). CategoryKey is a registry
// key, or empty for a custom category (the key is then derived from CategoryLabel by CategoryKeyFromLabel).
type VendorRequest struct {
	SubEventScope string       `json:"sub_event_scope"`
	CategoryKey   *string      `json:"category_key"`
	CategoryLabel *string      `json:"category_label"`
	VendorName    string       `json:"vendor_name"`
	PICName       *string      `json:"pic_name"`
	PhoneNumber   *string      `json:"phone_number"`
	PortfolioURL  *string      `json:"portfolio_url"`
	BudgetLimit   int64        `json:"budget_limit"` // 0 = follow the active scenario
	AgreedPrice   int64        `json:"agreed_price"`
	Status        VendorStatus `json:"status"`
	Notes         *string      `json:"notes"`
}

type VendorStatusRequest struct {
	Status VendorStatus `json:"status"`
}

// VendorResult answers POST, PUT and PATCH …/status. Warnings: estimate_differs, budget_link_released.
type VendorResult struct {
	Vendor   Vendor      `json:"vendor"`
	Warnings []pd.Notice `json:"warnings"`
}

type LockDealRequest struct {
	AgreedPrice       int64 `json:"agreed_price"`
	SyncToBudgetItems bool  `json:"sync_to_budget_items"`
}

// SharedItem is a budget line that several categories share; it is not linked and its estimate is not changed (04 BE 2.2).
type SharedItem struct {
	ItemKey      string     `json:"item_key"`
	ItemName     string     `json:"item_name"`                // name of the line, or of the scenario line when there is no budget line yet
	BudgetItemID *uuid.UUID `json:"budget_item_id,omitempty"` // absent when there is no budget line yet
	Plan         int64      `json:"plan"`                     // effective estimate of the line, or the scenario amount when there is no line
	TotalAgreed  int64      `json:"total_agreed"`
}

// LockDealResult: Warnings: shared_budget_item, unit_pricing.
type LockDealResult struct {
	Vendor       Vendor      `json:"vendor"`
	BudgetItemID *uuid.UUID  `json:"budget_item_id,omitempty"` // the line that was linked or created
	Created      bool        `json:"created"`                  // a new line was made
	SharedItem   *SharedItem `json:"shared_item,omitempty"`
	Warnings     []pd.Notice `json:"warnings"`
}

// ----- Tab 2: shopping list -----

const (
	ShoppingNotStarted = "belum"
	ShoppingInProgress = "proses"
	ShoppingDone       = "selesai"

	PurposeSouvenir = "souvenir"
)

type ShoppingItem struct {
	ID               uuid.UUID  `json:"id"`
	SubEventScope    string     `json:"sub_event_scope"`
	Category         string     `json:"category"`
	ItemName         string     `json:"item_name"`
	Quantity         float64    `json:"quantity"` // two decimals
	Unit             *string    `json:"unit,omitempty"`
	EstimatedPrice   int64      `json:"estimated_price"` // per unit
	ActualPrice      int64      `json:"actual_price"`    // per unit; 0 = not bought yet
	EstimatedTotal   int64      `json:"estimated_total"` // round(Quantity × EstimatedPrice), computed by the database
	ActualTotal      int64      `json:"actual_total"`
	PriceDelta       *int64     `json:"price_delta,omitempty"` // EstimatedTotal − ActualTotal (positive = saving); absent while ActualPrice is 0
	PurchaseLocation *string    `json:"purchase_location,omitempty"`
	VendorID         *uuid.UUID `json:"vendor_id,omitempty"`
	VendorName       *string    `json:"vendor_name,omitempty"`
	BudgetItemID     *uuid.UUID `json:"budget_item_id,omitempty"` // the rollup line; absent when the item is not rolled up
	Purpose          *string    `json:"purpose,omitempty"`        // souvenir
	Status           string     `json:"status"`                   // belum | proses | selesai
	SortOrder        int        `json:"sort_order"`
	TemplateKey      *string    `json:"template_key,omitempty"`
	Notes            *string    `json:"notes,omitempty"`
}

type ShoppingKPI struct {
	EstimatedTotal int64  `json:"estimated_total"` // Σ EstimatedTotal of every item
	ActualTotal    int64  `json:"actual_total"`    // Σ ActualTotal of the finished items
	Done           int    `json:"done"`
	InProgress     int    `json:"in_progress"`
	NotStarted     int    `json:"not_started"`
	Total          int    `json:"total"`
	SouvenirStock  *int64 `json:"souvenir_stock,omitempty"` // floor(Σ Quantity) of the finished souvenir items; absent when no item is a souvenir
}

// RollupLine is a budget line whose figures are computed from the shopping list.
type RollupLine struct {
	BudgetItemID  uuid.UUID `json:"budget_item_id"`
	ItemName      string    `json:"item_name"`
	SubEventScope string    `json:"sub_event_scope"`
	Estimate      int64     `json:"estimate"`
	Paid          int64     `json:"paid"`
	LinkedItems   int       `json:"linked_items"`
}

// RollupTarget is the line behind the button "compute the line from this list" (its key comes from the registry, 04 BE 2.4).
type RollupTarget struct {
	BudgetItemID  *uuid.UUID `json:"budget_item_id,omitempty"` // absent when the line does not exist yet: it will be created
	ItemKey       string     `json:"item_key"`
	ItemName      string     `json:"item_name"`
	SubEventScope string     `json:"sub_event_scope"`
	Rollup        bool       `json:"rollup"`       // already computed from the shopping list
	HasPayments   bool       `json:"has_payments"` // stored payments > 0: switching it on needs confirm
	Estimate      int64      `json:"estimate"`
	Paid          int64      `json:"paid"`
}

type ShoppingOverview struct {
	KPI           ShoppingKPI    `json:"kpi"` // always the whole project
	Rollups       []RollupLine   `json:"rollups"`
	DefaultTarget RollupTarget   `json:"default_target"`
	Items         []ShoppingItem `json:"items"` // filtered by ?scope=&category=&status=
}

// ShoppingItemRequest is the body of POST. BudgetItemID exists only here (it must be a rollup line of the
// project); PUT never changes the link.
type ShoppingItemRequest struct {
	SubEventScope    string     `json:"sub_event_scope"`
	Category         string     `json:"category"`
	ItemName         string     `json:"item_name"`
	Quantity         *float64   `json:"quantity"` // empty = 1
	Unit             *string    `json:"unit"`
	EstimatedPrice   int64      `json:"estimated_price"`
	ActualPrice      int64      `json:"actual_price"`
	PurchaseLocation *string    `json:"purchase_location"`
	VendorID         *uuid.UUID `json:"vendor_id"`
	BudgetItemID     *uuid.UUID `json:"budget_item_id"`
	Purpose          *string    `json:"purpose"`
	Status           string     `json:"status"`
	Notes            *string    `json:"notes"`
}

// UpdateShoppingItemRequest is the body of PUT: it replaces the item, except the rollup link, the template key and the order.
type UpdateShoppingItemRequest struct {
	SubEventScope    string     `json:"sub_event_scope"`
	Category         string     `json:"category"`
	ItemName         string     `json:"item_name"`
	Quantity         *float64   `json:"quantity"`
	Unit             *string    `json:"unit"`
	EstimatedPrice   int64      `json:"estimated_price"`
	ActualPrice      int64      `json:"actual_price"`
	PurchaseLocation *string    `json:"purchase_location"`
	VendorID         *uuid.UUID `json:"vendor_id"`
	Purpose          *string    `json:"purpose"`
	Status           string     `json:"status"`
	Notes            *string    `json:"notes"`
}

type ShoppingStatusRequest struct {
	Status string `json:"status"`
}

const (
	ModeAppend          = "append"
	ModeReplaceTemplate = "replace_template"
)

type ShoppingTemplateRequest struct {
	Mode    string `json:"mode"` // append | replace_template
	Confirm bool   `json:"confirm"`
}

// ShoppingTemplateResult: Warnings: budget_link_lost.
type ShoppingTemplateResult struct {
	Created  int         `json:"created"`
	Skipped  int         `json:"skipped"`
	Deleted  int         `json:"deleted"`
	Warnings []pd.Notice `json:"warnings"`
}

type LinkToBudgetRequest struct {
	BudgetItemID *uuid.UUID  `json:"budget_item_id"` // empty = the default line of the registry (RollupTarget)
	ItemIDs      []uuid.UUID `json:"item_ids"`       // empty = every item that is not linked to another rollup line
	Confirm      bool        `json:"confirm"`
}

type LinkToBudgetResult struct {
	Line    RollupLine `json:"line"`
	Created bool       `json:"created"` // the line did not exist and was made
	Linked  int        `json:"linked"`  // items that were linked now
}

type UnlinkFromBudgetRequest struct {
	BudgetItemID uuid.UUID `json:"budget_item_id"`
}

type UnlinkFromBudgetResult struct {
	Unlinked int `json:"unlinked"`
}
