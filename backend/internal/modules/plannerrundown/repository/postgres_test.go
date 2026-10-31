package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/domain"
)

func TestMapErr(t *testing.T) {
	pg := func(code, constraint string) error { return &pgconn.PgError{Code: code, ConstraintName: constraint} }
	for name, tc := range map[string]struct {
		err  error
		want error
	}{
		"session":             {pg("23503", "planner_rundowns_session_id_project_id_fkey"), domain.ErrInvalidScope},
		"vendor":              {pg("23503", "planner_rundowns_vendor_id_project_id_fkey"), domain.ErrInvalidVendor},
		"agenda of a song":    {pg("23503", "planner_playlists_rundown_id_project_id_fkey"), domain.ErrInvalidAgenda},
		"another foreign key": {pg("23503", "planner_rundowns_project_id_fkey"), domain.ErrConflict},
		"unique":              {pg("23505", "planner_rundowns_template_key_idx"), domain.ErrConflict},
		"serialization":       {pg("40001", ""), domain.ErrConflict},
		"deadlock":            {pg("40P01", ""), domain.ErrConflict},
		"wrapped":             {errors.Join(errors.New("ctx"), pg("23503", "planner_rundowns_vendor_id_project_id_fkey")), domain.ErrInvalidVendor},
	} {
		if got := mapErr(tc.err); !errors.Is(got, tc.want) {
			t.Errorf("%s: %v, want %v", name, got, tc.want)
		}
	}
	other := pg("22001", "")
	if mapErr(other) != other {
		t.Error("an error nobody maps was rewritten")
	}
	plain := errors.New("plain")
	if mapErr(plain) != plain {
		t.Error("a plain error was rewritten")
	}
	if !errors.Is(notFound(pgx.ErrNoRows), domain.ErrNotFound) || notFound(plain) != plain {
		t.Error("notFound")
	}
	if err := affected(pgconn.NewCommandTag("UPDATE 0"), nil); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("no row affected: %v", err)
	}
	if err := affected(pgconn.NewCommandTag("UPDATE 1"), nil); err != nil {
		t.Errorf("one row affected: %v", err)
	}
	if err := affected(pgconn.CommandTag{}, pg("23505", "")); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("an error: %v", err)
	}
}

// timestamptz columns come back in the zone of the process; the API speaks UTC.
func TestUTC(t *testing.T) {
	if utc(nil) != nil {
		t.Error("nil must stay nil")
	}
	jkt := time.FixedZone("WIB", 7*3600)
	in := time.Date(2027, 3, 28, 9, 0, 0, 0, jkt)
	got := utc(&in)
	if got.Location() != time.UTC || !got.Equal(in) || got.Hour() != 2 {
		t.Errorf("utc = %v", got)
	}
	if in.Location() != jkt {
		t.Error("the argument was changed")
	}
}
