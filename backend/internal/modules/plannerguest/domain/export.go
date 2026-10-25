package domain

import (
	"time"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// GuestExport is what GET /guests/export is built from: every guest in order (05 BE 2.6).
type GuestExport struct {
	Project  pd.Project
	Today    time.Time // calendar date in the project's zone
	Spec     GuestSpec
	Sessions []SessionRef
	Guests   []Guest
}

// AttendanceExport is the final attendance sheet: the expected guests and the ones who came, with the entry time in
// the project's zone.
type AttendanceExport struct {
	Project  pd.Project
	Today    time.Time
	Location *time.Location
	Spec     GuestSpec
	Sessions []SessionRef
	Guests   []Guest
	KPI      ReceptionKPI
}

// TemplateInput is everything the import template of one project needs; it comes from the registry and the session
// master.
type TemplateInput struct {
	GuestWord string   // Labels.Guest: "Tamu" or "Peserta"
	Sides     []Option // empty = no Pihak column
	Cats      []Option
	Sessions  []string // session names; "Umum / Semua Sesi" is always there
}

// TemplateExport adds the project and the date for the file name.
type TemplateExport struct {
	Project pd.Project
	Today   time.Time
	Input   TemplateInput
}
