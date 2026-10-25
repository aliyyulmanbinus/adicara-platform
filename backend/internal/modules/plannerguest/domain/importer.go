package domain

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/text/encoding/charmap"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
)

// Import limits (05 BE 3.4).
const (
	MaxImportRows      = 2000
	MaxImportBytes     = 5 << 20
	MaxGuestsInProject = 5000
)

// FileError becomes 400 invalid_file; Reason appears as "file: <Reason>".
type FileError struct{ Reason string }

func (e *FileError) Error() string { return e.Reason }

// Record is one row of the file: its line number (the header is 1) and its cells as text.
type Record struct {
	Row   int
	Cells []string
}

// DecodeCSV reads a CSV made by Excel as it is: UTF-8 with or without BOM, or Windows-1252, with a comma, a
// semicolon (Excel with Indonesian settings) or a tab as the separator. Blank rows are skipped; the row number is the
// line the record starts on.
func DecodeCSV(data []byte) ([]Record, error) {
	if bytes.IndexByte(data, 0) >= 0 {
		return nil, &FileError{"not a text file"}
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(data) {
		dec, err := charmap.Windows1252.NewDecoder().Bytes(data)
		if err != nil {
			return nil, &FileError{"not valid UTF-8 or Windows-1252"}
		}
		data = dec
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = detectDelimiter(data)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	var out []Record
	for {
		cells, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, &FileError{"malformed CSV: " + err.Error()}
		}
		line, _ := r.FieldPos(0)
		out = append(out, Record{Row: line, Cells: cells})
	}
	return out, nil
}

// detectDelimiter counts the separators outside quotes on the first non-empty line; a tie -> comma.
func detectDelimiter(data []byte) rune {
	count := map[rune]int{}
	inQuote := false
	for _, b := range data {
		if b == '"' {
			inQuote = !inQuote
		}
		if !inQuote {
			switch b {
			case ',', ';', '\t':
				count[rune(b)]++
			case '\n':
				if count[',']+count[';']+count['\t'] > 0 {
					goto done
				}
			}
		}
	}
done:
	best := ','
	for _, d := range []rune{';', '\t'} {
		if count[d] > count[best] {
			best = d
		}
	}
	return best
}

// ---------- columns ----------

type Column int

const (
	ColName Column = iota
	ColSide
	ColCategory
	ColSession
	ColTable
	ColPax
	ColPhone
	ColNotes
)

// Template headers: Nama Tamu | Pihak | Kategori | Sesi | No Meja | Pax | No WhatsApp | Catatan.
// The seminar uses "Nama Peserta"; both are recognised for every type.
var columnAliases = map[Column][]string{
	ColName:     {"nama tamu", "nama peserta", "nama"},
	ColSide:     {"pihak", "sisi"},
	ColCategory: {"kategori"},
	ColSession:  {"sesi"},
	ColTable:    {"no meja", "nomor meja", "meja"},
	ColPax:      {"pax", "jumlah pax", "jumlah"},
	ColPhone:    {"no whatsapp", "no wa", "whatsapp", "no hp", "no telepon", "telepon"},
	ColNotes:    {"catatan", "keterangan"},
}

func normHeader(s string) string {
	return strings.ToLower(squash(strings.TrimSpace(strings.TrimRight(strings.TrimSpace(s), "*"))))
}

// MapColumns finds the columns by header (case-insensitive, whitespace folded, * dropped). Only the name column is
// required; a missing column counts as empty; an unknown column is ignored; a repeated header: the first one wins.
func MapColumns(header []string) (map[Column]int, error) {
	m := map[Column]int{}
	for i, h := range header {
		n := normHeader(h)
		for col, aliases := range columnAliases {
			for _, a := range aliases {
				if n == a {
					if _, dup := m[col]; !dup {
						m[col] = i
					}
				}
			}
		}
	}
	if _, ok := m[ColName]; !ok {
		return nil, &FileError{"missing column: Nama Tamu"}
	}
	return m, nil
}

// ---------- rows ----------

type SessionRef struct{ ScopeKey, Name string }

type ExistingGuest struct {
	ID   uuid.UUID
	Name string
	WA   string // formatted_wa_number, "" when empty
}

type ImportOptions struct{ SkipInvalid, SkipDuplicates bool }

type ImportConfig struct {
	Spec     GuestSpec
	Sessions []SessionRef // including the system session all ("Umum / Semua Sesi")
	Existing []ExistingGuest
	Options  ImportOptions
}

// PlannedGuest is a valid row ready to be written; the usecase makes the token when it writes.
type PlannedGuest struct {
	Row       int
	Name      string
	Side      *string
	Category  string
	Scope     string
	Table     *string
	Pax       int
	Phone     *string
	WA        *string
	Notes     *string
	Duplicate bool
}

type ImportPlan struct {
	Result  ImportResult
	Insert  []PlannedGuest
	Blocked bool // there are invalid rows and SkipInvalid=false: nothing may be written
}

const (
	reasonPhone = "must be an Indonesian phone number (08…, +62…)"
)

func atMost(n int) string { return "must be at most " + strconv.Itoa(n) + " characters" }

func hasControl(s string, allowNewline bool) bool {
	for _, r := range s {
		if unicode.IsControl(r) && !(allowNewline && (r == '\n' || r == '\t' || r == '\r')) {
			return true
		}
	}
	return false
}

func cell(rec Record, cols map[Column]int, c Column) string {
	if i, ok := cols[c]; ok && i < len(rec.Cells) {
		return rec.Cells[i]
	}
	return ""
}

func blank(rec Record) bool {
	for _, c := range rec.Cells {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func ptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// BuildImportPlan checks the whole file (recs[0] = the header) and builds the preview result and the rows to write.
// It writes nothing: dry_run and the real import use the same function.
func BuildImportPlan(cfg ImportConfig, recs []Record) (ImportPlan, error) {
	plan := ImportPlan{Result: ImportResult{Errors: []ImportRowError{}, Duplicates: []ImportDuplicate{}, Preview: []ImportPreviewRow{}}}
	if len(recs) == 0 {
		return plan, &FileError{"empty file"}
	}
	cols, err := MapColumns(recs[0].Cells)
	if err != nil {
		return plan, err
	}
	var rows []Record
	for _, r := range recs[1:] {
		if !blank(r) {
			rows = append(rows, r)
		}
	}
	if len(rows) > MaxImportRows {
		return plan, &FileError{"too many rows (max " + strconv.Itoa(MaxImportRows) + ")"}
	}
	plan.Result.TotalRows = len(rows)

	scopeBy := map[string]string{}
	for _, s := range cfg.Sessions {
		scopeBy[strings.ToLower(squash(s.Name))] = s.ScopeKey
		scopeBy[strings.ToLower(s.ScopeKey)] = s.ScopeKey
	}
	scopeBy["umum"], scopeBy["semua"] = "all", "all"

	existing := map[string]uuid.UUID{}
	for _, g := range cfg.Existing {
		k := dedupeKey(g.Name, g.WA)
		if _, ok := existing[k]; !ok {
			existing[k] = g.ID
		}
	}
	firstRow := map[string]int{}

	for _, rec := range rows {
		var errs []ImportRowError
		bad := func(field, reason string) {
			errs = append(errs, ImportRowError{Row: rec.Row, Field: field, Reason: reason})
		}
		g := PlannedGuest{Row: rec.Row, Category: FallbackCategory, Scope: "all", Pax: 1}

		g.Name = squash(cell(rec, cols, ColName))
		switch {
		case g.Name == "":
			bad("name", "required")
		case utf8.RuneCountInString(g.Name) > 255:
			bad("name", atMost(255))
		case hasControl(g.Name, false):
			bad("name", "must not contain control characters")
		}

		if v := cell(rec, cols, ColSide); squash(v) != "" {
			if len(cfg.Spec.Sides) == 0 {
				bad("guest_side", "must be empty for an event type without sides")
			} else if k, ok := cfg.Spec.ResolveSide(v); ok {
				g.Side = ptr(k)
			} else {
				bad("guest_side", "must be a registry side key")
			}
		}

		g.Category = cfg.Spec.ResolveCategory(cell(rec, cols, ColCategory))
		if utf8.RuneCountInString(g.Category) > 60 {
			bad("category", atMost(60))
		} else if hasControl(g.Category, false) {
			bad("category", "must not contain control characters")
		}

		if v := strings.ToLower(squash(cell(rec, cols, ColSession))); v != "" {
			if k, ok := scopeBy[v]; ok {
				g.Scope = k
			} else {
				bad("sub_event_scope", "must be a session of this project")
			}
		}

		if v := squash(cell(rec, cols, ColTable)); v != "" {
			if utf8.RuneCountInString(v) > 30 {
				bad("table_number", atMost(30))
			} else if hasControl(v, false) {
				bad("table_number", "must not contain control characters")
			}
			g.Table = ptr(v)
		}

		if v := strings.TrimSpace(cell(rec, cols, ColPax)); v != "" {
			n, err := strconv.Atoi(v)
			switch {
			case err != nil:
				bad("pax_count", "must be an integer")
			case n < 1 || n > 20:
				bad("pax_count", "must be between 1 and 20")
			default:
				g.Pax = n
			}
		}

		if v := strings.TrimSpace(cell(rec, cols, ColPhone)); v != "" {
			if utf8.RuneCountInString(v) > 40 {
				bad("phone_number", atMost(40))
			} else if wa, err := pd.NormalizeWA(v); err != nil {
				bad("phone_number", reasonPhone)
			} else {
				g.Phone = ptr(v)
				g.WA = ptr(wa)
			}
		}

		if v := strings.TrimSpace(cell(rec, cols, ColNotes)); v != "" {
			if utf8.RuneCountInString(v) > 2000 {
				bad("notes", atMost(2000))
			} else if hasControl(v, true) {
				bad("notes", "must not contain control characters")
			}
			g.Notes = ptr(v)
		}

		if len(errs) > 0 {
			plan.Result.InvalidRows++
			plan.Result.Errors = append(plan.Result.Errors, errs...)
			continue
		}

		wa := ""
		if g.WA != nil {
			wa = *g.WA
		}
		k := dedupeKey(g.Name, wa)
		if id, ok := existing[k]; ok {
			g.Duplicate = true
			plan.Result.Duplicates = append(plan.Result.Duplicates, ImportDuplicate{Row: rec.Row, Name: g.Name, OfGuest: &id})
		} else if first, ok := firstRow[k]; ok {
			g.Duplicate = true
			f := first
			plan.Result.Duplicates = append(plan.Result.Duplicates, ImportDuplicate{Row: rec.Row, Name: g.Name, OfRow: &f})
		} else {
			firstRow[k] = rec.Row
		}

		plan.Result.ValidRows++
		if len(plan.Result.Preview) < 5 {
			plan.Result.Preview = append(plan.Result.Preview, ImportPreviewRow{Row: g.Row, Name: g.Name, GuestSide: g.Side, Category: g.Category, SubEventScope: g.Scope, TableNumber: g.Table, PaxCount: g.Pax, FormattedWANumber: g.WA})
		}
		if g.Duplicate && cfg.Options.SkipDuplicates {
			plan.Result.Skipped++
			continue
		}
		plan.Insert = append(plan.Insert, g)
	}

	if plan.Result.InvalidRows > 0 {
		if cfg.Options.SkipInvalid {
			plan.Result.Skipped += plan.Result.InvalidRows
		} else {
			plan.Blocked = true
			plan.Insert = nil
		}
	}
	return plan, nil
}

// dedupeKey: the name without case and double spaces, plus the normalised WhatsApp number (empty when there is none).
// Two guests with the same name and no number are duplicates; with different numbers they are not.
func dedupeKey(name, wa string) string { return strings.ToLower(squash(name)) + "|" + wa }
