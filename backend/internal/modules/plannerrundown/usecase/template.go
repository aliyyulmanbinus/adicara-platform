package usecase

import (
	"context"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerrundown/domain"
)

// ApplyRundownTemplate is POST /rundowns/apply-template (06 BE 2.6).
//
//   - append (the default; also what the wizard runs) adds the agendas whose template_key is not there yet and skips the rest.
//   - replace_template resets, in place, the template agendas that have not started: the times, the name, the PIC, the
//     description, the highlights and the order go back to the template, while the id, the vendor, the session and the
//     songs that point at the agenda stay. Agendas in progress or completed are not touched and counted as kept; agendas
//     the user made are not touched. It needs confirm when it resets anything, because the user's changes to those agendas
//     are overwritten.
func (s *Service) ApplyRundownTemplate(ctx context.Context, p pd.Project, req domain.ApplyTemplateRequest) (domain.TemplateResult, error) {
	mode, err := templateMode(req.Mode, false)
	if err != nil {
		return domain.TemplateResult{}, err
	}

	var res domain.TemplateResult
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		var err error
		res, err = s.applyAgendaTemplate(ctx, p, mode, req.Confirm)

		return err
	})
	if err != nil {
		return domain.TemplateResult{}, mapErr(err, "agenda")
	}

	return res, nil
}

// ApplyPlaylistTemplate is POST /playlists/apply-template. The playlist only appends: a template song has no status of
// work, so nothing is "kept", and a reset would overwrite the format, the cue and the status the user filled in.
func (s *Service) ApplyPlaylistTemplate(ctx context.Context, p pd.Project, req domain.ApplyTemplateRequest) (domain.TemplateResult, error) {
	if _, err := templateMode(req.Mode, true); err != nil {
		return domain.TemplateResult{}, err
	}

	var res domain.TemplateResult
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		var err error
		res, err = s.applyPlaylistTemplate(ctx, p)

		return err
	})
	if err != nil {
		return domain.TemplateResult{}, mapErr(err, "song")
	}

	return res, nil
}

// templateMode reads the mode of a request: empty is append. The playlist refuses replace_template with its own message.
func templateMode(mode string, appendOnly bool) (string, error) {
	switch {
	case mode == "" || mode == domain.ModeAppend:
		return domain.ModeAppend, nil
	case mode == domain.ModeReplaceTemplate && appendOnly:
		return "", invalidInput("mode", "must be append")
	case mode == domain.ModeReplaceTemplate:
		return mode, nil
	}

	return "", invalidInput("mode", "must be one of append, replace_template")
}

// agendaPlan is what replace_template decides for the template agendas before it writes anything.
type agendaPlan struct {
	insert []domain.AgendaWrite
	reset  []resetAgenda
	res    domain.TemplateResult
}

type resetAgenda struct {
	id uuid.UUID
	w  domain.AgendaWrite
}

// templateWrite is the row of a template agenda in its session; position is its place among the template agendas of
// that session, which is the sort_order a reset gives back.
func templateWrite(t domain.TemplateAgenda, sessionID uuid.UUID, position int) domain.AgendaWrite {
	key := t.Key
	times := domain.Times{Start: t.Start}
	if t.End != "" {
		end := t.End
		times.End = &end
	}

	return domain.AgendaWrite{
		ID: uuid.New(), SessionID: sessionID, Times: times, ActivityName: t.Name, PICName: t.PIC,
		Description: t.Description, Highlights: t.Highlights, TemplateKey: &key, SortOrder: position,
	}
}

// planAgendas sorts the template agendas of the project's type into the ones to insert, to reset and to leave alone.
func (s *Service) planAgendas(ctx context.Context, p pd.Project, mode string) (agendaPlan, error) {
	plan := agendaPlan{res: domain.TemplateResult{Warnings: []pd.Notice{}}}
	template := domain.AgendasFor(p.EventType)

	sessions, err := s.repo.Sessions(ctx, p.ID)
	if err != nil {
		return plan, err
	}
	byScope := make(map[string]domain.SessionRef, len(sessions))
	for _, sess := range sessions {
		if !sess.IsSystem {
			byScope[sess.ScopeKey] = sess
		}
	}

	var states map[string]domain.TemplateAgendaState
	if mode == domain.ModeReplaceTemplate {
		keys := make([]string, len(template))
		for i, t := range template {
			keys[i] = t.Key
		}
		if states, err = s.repo.TemplateAgendaStates(ctx, p.ID, keys); err != nil {
			return plan, err
		}
	}

	position := map[string]int{}
	missing := 0
	for _, t := range template {
		sess, ok := byScope[t.ScopeKey]
		if !ok {
			missing++
			continue
		}
		position[t.ScopeKey]++
		w := templateWrite(t, sess.ID, position[t.ScopeKey])

		st, exists := states[t.Key]
		switch {
		case !exists:
			plan.insert = append(plan.insert, w)
		case st.Status != domain.StatusPending:
			plan.res.Kept++
		case st.SessionID != sess.ID:
			plan.res.Skipped++ // the user moved it to another session: it is theirs now
		default:
			plan.reset = append(plan.reset, resetAgenda{id: st.ID, w: w})
		}
	}
	plan.res.Reset = len(plan.reset)
	if missing > 0 {
		plan.res.Warnings = append(plan.res.Warnings, notice("scope_missing", "agendas of sessions that no longer exist were skipped", missing))
	}

	return plan, nil
}

// applyAgendaTemplate runs in a transaction (the caller's, or the project-creation one for the wizard).
func (s *Service) applyAgendaTemplate(ctx context.Context, p pd.Project, mode string, confirm bool) (domain.TemplateResult, error) {
	if len(domain.AgendasFor(p.EventType)) == 0 {
		return domain.TemplateResult{}, invalidTemplate("rundown")
	}
	if err := s.repo.LockProject(ctx, p.ID); err != nil {
		return domain.TemplateResult{}, err
	}
	plan, err := s.planAgendas(ctx, p, mode)
	if err != nil {
		return domain.TemplateResult{}, err
	}
	if len(plan.reset) > 0 && !confirm {
		return domain.TemplateResult{}, confirmRequired()
	}

	for i := range plan.reset {
		if err := s.repo.ResetTemplateAgenda(ctx, p.ID, plan.reset[i].id, &plan.reset[i].w); err != nil {
			return domain.TemplateResult{}, err
		}
	}
	for i := range plan.insert {
		inserted, err := s.repo.InsertTemplateAgenda(ctx, p.ID, &plan.insert[i])
		if err != nil {
			return domain.TemplateResult{}, err
		}
		if inserted {
			plan.res.Created++
		} else {
			plan.res.Skipped++
		}
	}

	// Counted after the inserts: all or nothing, the transaction is rolled back when the rundown would pass the limit.
	n, err := s.repo.CountAgendas(ctx, p.ID)
	if err != nil {
		return domain.TemplateResult{}, err
	}
	if n > maxAgendas {
		return domain.TemplateResult{}, agendaLimitReached()
	}

	return plan.res, nil
}

// applyPlaylistTemplate appends the songs of the template whose key is not there yet.
func (s *Service) applyPlaylistTemplate(ctx context.Context, p pd.Project) (domain.TemplateResult, error) {
	res := domain.TemplateResult{Warnings: []pd.Notice{}}
	template := domain.PlaylistFor(p.EventType)
	if len(template) == 0 {
		return res, invalidTemplate("playlist")
	}
	if err := s.repo.LockProject(ctx, p.ID); err != nil {
		return res, err
	}

	for _, t := range template {
		key := t.Key
		inserted, err := s.repo.InsertTemplateSong(ctx, p.ID, &domain.SongWrite{
			ID: uuid.New(), SongTitle: t.Title, Artist: t.Artist, MomentTag: t.MomentTag,
			AudioFormat: domain.FormatDigitalPlayback, Status: domain.AudioNeedsReview, TemplateKey: &key,
		})
		if err != nil {
			return res, err
		}
		if inserted {
			res.Created++
		} else {
			res.Skipped++
		}
	}

	n, err := s.repo.CountPlaylists(ctx, p.ID)
	if err != nil {
		return res, err
	}
	if n > maxSongs {
		return res, playlistLimitReached()
	}

	return res, nil
}
