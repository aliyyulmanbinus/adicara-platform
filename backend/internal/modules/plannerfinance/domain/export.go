package domain

import (
	"time"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// ExportData is everything the spreadsheet of GET /budget/export is built from:
// every line of the project in sort_order, the session matrix, the savings
// overview of the whole project and the session names.
type ExportData struct {
	Project  pd.Project
	Today    time.Time // calendar date in the project's zone
	Sessions []SessionRef
	Lines    []BudgetLine
	Matrix   BudgetMatrix
	Savings  SavingsOverview
}
