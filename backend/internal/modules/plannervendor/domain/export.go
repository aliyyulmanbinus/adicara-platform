package domain

import (
	"time"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// VendorExport is what the contact sheet of GET /vendors/export-contacts is built from: the fixed vendors in order
// (no prices, limits or notes: the sheet goes to the wedding organiser or the committee) and the session names.
type VendorExport struct {
	Project  pd.Project
	Today    time.Time // calendar date in the project's zone
	Sessions []SessionRef
	Vendors  []Vendor
}

// ShoppingExport is what the sheet of GET /shopping-items/export-shopping-list is built from: every item in order.
type ShoppingExport struct {
	Project  pd.Project
	Today    time.Time
	Sessions []SessionRef
	Items    []ShoppingItem
	KPI      ShoppingKPI
}
