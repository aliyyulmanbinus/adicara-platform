package domain

import "github.com/google/uuid"

// Dashboard widget payloads. A provider of another planner module returns one
// of these from SummaryProvider.Summarize; the dashboard embeds it unchanged.

type UrgentTask struct {
	ID       uuid.UUID `json:"id"`
	Title    string    `json:"title"`
	DueDate  string    `json:"due_date"`
	DaysLeft int       `json:"days_left"` // negative = already overdue
}

type TimelineSummary struct {
	Total      int          `json:"total"`
	Completed  int          `json:"completed"`
	InProgress int          `json:"in_progress"`
	Overdue    int          `json:"overdue"`
	Urgent     []UrgentTask `json:"urgent"` // at most 3, by due date
}

type LineStatus struct {
	Settled int `json:"settled"`
	Partial int `json:"partial"`
	Unpaid  int `json:"unpaid"`
}

type FinanceSummary struct {
	TargetBudget     int64      `json:"target_budget"`
	TotalEstimate    int64      `json:"total_estimate"`
	TotalPaid        int64      `json:"total_paid"`
	Remaining        int64      `json:"remaining"` // Σ max(estimate − paid, 0)
	Overpaid         int64      `json:"overpaid"`  // Σ max(paid − estimate, 0)
	SavingsTarget    int64      `json:"savings_target"`
	SavingsCollected int64      `json:"savings_collected"`
	FreeCash         int64      `json:"free_cash"`    // collected − paid; negative = short
	CashDeficit      int64      `json:"cash_deficit"` // max(remaining − free_cash, 0)
	LineStatus       LineStatus `json:"line_status"`
}

type VendorSummary struct {
	CategoriesTotal int `json:"categories_total"`
	CategoriesFixed int `json:"categories_fixed"`
	Skipped         int `json:"skipped"`
	FixedVendors    int `json:"fixed_vendors"`
	Pipeline        int `json:"pipeline"`
}

type GuestSide struct {
	Key         *string `json:"key"` // nil = not decided yet
	Label       string  `json:"label"`
	Invitations int     `json:"invitations"`
	Pax         int     `json:"pax"`
}

type RSVPSummary struct {
	Attending    int `json:"attending"`
	Declined     int `json:"declined"`
	Maybe        int `json:"maybe"`
	Unconfirmed  int `json:"unconfirmed"`
	AttendingPax int `json:"attending_pax"`
}

type GuestSummary struct {
	Invitations int         `json:"invitations"`
	TotalPax    int         `json:"total_pax"`
	Sides       []GuestSide `json:"sides"`
	Sent        int         `json:"sent"`
	RSVP        RSVPSummary `json:"rsvp"`
}

type AdminDocSummary struct {
	Label     string   `json:"label"`
	Completed int      `json:"completed"`
	Total     int      `json:"total"`
	Pending   []string `json:"pending"` // at most 5 names
}

type CurrentAgenda struct {
	Activity string `json:"activity"`
	Session  string `json:"session"`
}

type NextSession struct {
	Name      string  `json:"name"`
	Date      string  `json:"date"`
	StartTime *string `json:"start_time"`
}

type RundownSummary struct {
	Sessions      int            `json:"sessions"`
	Agendas       int            `json:"agendas"`
	Completed     int            `json:"completed"`
	InProgress    int            `json:"in_progress"`
	Songs         int            `json:"songs"`
	CurrentAgenda *CurrentAgenda `json:"current_agenda"`
	NextSession   *NextSession   `json:"next_session"`
}
