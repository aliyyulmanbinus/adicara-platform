package domain

import (
	"sort"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// Timeline presets (templates_available.timeline_presets of the registry).
const (
	PresetWeddingPDF44       = "wedding_pdf_44"
	PresetWeddingEssential22 = "wedding_essential_22"
	PresetGeneral            = "general"
)

// Horizons the offsets of the templates are written for (TEMPLATE_EVENT chapter 1).
const (
	WeddingHorizon = 180
	GeneralHorizon = 60
)

var weddingPhaseNames = []string{
	"Persiapan Awal & Lamaran", "Administrasi & Booking Vendor Inti", "Detail Acara & Perlengkapan", "Finalisasi & Menuju Hari-H",
}

var generalPhaseNames = []string{"Perencanaan", "Persiapan", "Pelaksanaan", "Pekan Terakhir"}

// The one document whose cost has a budget line waiting for it: the KUA fee is
// the line `biaya_kua` of the session `akad` (02 BE 4.5). Any other document gets
// a new line under the session "all".
const (
	KUADocKey        = "wedding.pendaftaran_kua"
	KUABudgetScope   = "akad"
	KUABudgetItemKey = "biaya_kua"
)

// Sample project statuses from Wedding_Planner.pdf: tasks 1-15 are "Selesai"
// and 16-44 "Proses" (p. 3); documents 1, 2, 3, 6, 7, 8, 10, 14, 15 are done (p. 11).
const (
	SampleTasksDoneUpTo       = 15
	SampleTasksInProgressUpTo = 44
)

var sampleDocsDone = map[int]bool{1: true, 2: true, 3: true, 6: true, 7: true, 8: true, 10: true, 14: true, 15: true}

// TaskTemplate is a task as written in a template: offsets are days before
// Hari-H on the horizon of the template. Scope is a scope_key of the session
// master; a project that deleted that session gets "all" instead.
type TaskTemplate struct {
	Key       string
	Title     string
	Scope     string
	Category  string
	Phase     int
	Start     int // H-Start
	End       int // H-End, End <= Start
	Essential bool
	Origin    Origin
}

// Schedule is a resolved template: its tasks in the order they are stored
// (sort_order = position), the horizon and the phase names.
type Schedule struct {
	Horizon    int
	PhaseNames []string
	Tasks      []TaskTemplate
}

// PhaseName returns the name of phase n (1-4).
func (s Schedule) PhaseName(n int) string {
	if n < 1 || n > len(s.PhaseNames) {
		return ""
	}

	return s.PhaseNames[n-1]
}

// BuildSchedule resolves a preset for an event type. ok is false when the
// preset does not belong to the type. An empty preset is the wedding's six
// Adicara tasks alone (include_adicara_tasks without a base preset).
func BuildSchedule(preset string, eventType pd.EventType, includeAdicara bool) (Schedule, bool) {
	wedding := eventType == pd.EventWedding

	switch {
	case wedding && (preset == PresetWeddingPDF44 || preset == PresetWeddingEssential22 || preset == ""):
		s := Schedule{Horizon: WeddingHorizon, PhaseNames: weddingPhaseNames}
		for _, t := range weddingTasks {
			if preset == "" || (preset == PresetWeddingEssential22 && !t.Essential) {
				continue
			}
			t.Origin = OriginPDF
			s.Tasks = append(s.Tasks, t)
		}
		if includeAdicara {
			for _, t := range weddingAdicaraTasks {
				t.Origin = OriginAdicara
				s.Tasks = append(s.Tasks, t)
			}
		}
		if len(s.Tasks) == 0 {
			return Schedule{}, false
		}

		return s, true

	case !wedding && preset == PresetGeneral:
		all := make([]TaskTemplate, 0, len(generalTasks)+len(typeTasks[eventType]))
		all = append(append(all, generalTasks...), typeTasks[eventType]...)
		// Chronological: later offsets first; a stable sort keeps the general
		// tasks ahead of a special task that starts the same day.
		sort.SliceStable(all, func(i, j int) bool { return all[i].Start > all[j].Start })
		for i := range all {
			all[i].Origin = OriginAdicara
		}

		return Schedule{Horizon: GeneralHorizon, PhaseNames: generalPhaseNames, Tasks: all}, true
	}

	return Schedule{}, false
}

// DocTemplate is a document of the standard checklist of an event type.
type DocTemplate struct {
	Key      string
	Name     string
	Subject  string
	Quantity string
	Cost     int64
	Note     string
	Origin   Origin
}

// DocsFor returns the standard documents of an event type in sort order: the
// 15 KUA documents of the wedding, a short permit list for the others, and
// nothing for "other" (the user adds their own).
func DocsFor(eventType pd.EventType) []DocTemplate {
	src, origin := typeDocs[eventType], OriginAdicara
	if eventType == pd.EventWedding {
		src, origin = weddingDocs, OriginPDF
	}

	out := make([]DocTemplate, len(src))
	for i, d := range src {
		d.Origin = origin
		out[i] = d
	}

	return out
}

// SampleDocsDone returns the template keys of the documents that the PDF marks as done.
func SampleDocsDone() []string {
	var keys []string
	for i, d := range weddingDocs {
		if sampleDocsDone[i+1] {
			keys = append(keys, d.Key)
		}
	}

	return keys
}
