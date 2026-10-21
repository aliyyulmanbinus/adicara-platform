package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/domain"
)

func (s *Service) spec(p pd.Project) pd.EventTypeSpec {
	spec, _ := s.registry.Get(p.EventType) // the CHECK on event_type guarantees it exists

	return spec
}

func hasSubject(spec pd.EventTypeSpec, key string) bool {
	for _, o := range spec.AdminSubjects {
		if o.Key == key {
			return true
		}
	}

	return false
}

// defaultSubject is what a document gets when the request names none: "bersama"
// (both sides) for the wedding, "umum" for the other types.
func defaultSubject(spec pd.EventTypeSpec) string {
	for _, key := range []string{"bersama", "umum"} {
		if hasSubject(spec, key) {
			return key
		}
	}
	if len(spec.AdminSubjects) > 0 {
		return spec.AdminSubjects[0].Key
	}

	return "bersama"
}

func checkSubject(spec pd.EventTypeSpec, key string) error {
	if !hasSubject(spec, key) {
		return invalidInput("subject_role", "must be one of the event type subjects")
	}

	return nil
}

// ListDocs returns the filtered documents in sort_order; the summary counts every document.
func (s *Service) ListDocs(ctx context.Context, p pd.Project, subject, status string) (domain.AdminDocList, error) {
	f := domain.DocFilter{}
	if subject != "" {
		if err := checkSubject(s.spec(p), subject); err != nil {
			return domain.AdminDocList{}, invalidInput("subject", "must be one of the event type subjects")
		}
		f.Subject = subject
	}
	if status != "" {
		if !domain.DocStatus(status).Valid() {
			return domain.AdminDocList{}, invalidInput("status", "must be one of belum, selesai")
		}
		f.Status = status
	}

	return s.docList(ctx, p.ID, f)
}

func (s *Service) docList(ctx context.Context, projectID uuid.UUID, f domain.DocFilter) (domain.AdminDocList, error) {
	docs, err := s.repo.ListDocs(ctx, projectID, f)
	if err != nil {
		return domain.AdminDocList{}, err
	}
	sum, err := s.repo.SummarizeDocs(ctx, projectID)
	if err != nil {
		return domain.AdminDocList{}, err
	}

	return domain.AdminDocList{Docs: docs, Summary: sum}, nil
}

func (s *Service) CreateDoc(ctx context.Context, p pd.Project, req domain.CreateAdminDocRequest) (domain.AdminDoc, error) {
	spec := s.spec(p)
	name, err := cleanText("document_name", req.DocumentName, maxTitleRunes, true)
	if err != nil {
		return domain.AdminDoc{}, err
	}
	subject := strings.TrimSpace(req.SubjectRole)
	if subject == "" {
		subject = defaultSubject(spec)
	}
	if err := checkSubject(spec, subject); err != nil {
		return domain.AdminDoc{}, err
	}
	qty, err := cleanText("quantity_detail", req.QuantityDetail, maxQuantityRunes, false)
	if err != nil {
		return domain.AdminDoc{}, err
	}
	if err := validateMoney("nominal_cost", req.NominalCost); err != nil {
		return domain.AdminDoc{}, err
	}
	notes, err := cleanNote("notes", req.Notes, maxNoteRunes)
	if err != nil {
		return domain.AdminDoc{}, err
	}

	var id uuid.UUID
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		n, err := s.repo.CountDocs(ctx, p.ID)
		if err != nil {
			return err
		}
		if n >= maxDocs {
			return apierr.Conflict("admin_doc_limit_reached", fmt.Sprintf("at most %d documents per project", maxDocs))
		}
		order, err := s.repo.NextDocSortOrder(ctx, p.ID)
		if err != nil {
			return err
		}
		d := domain.AdminDoc{
			ID: uuid.New(), ProjectID: p.ID, DocumentName: name, SubjectRole: subject, QuantityDetail: qty,
			NominalCost: req.NominalCost, Status: domain.DocBelum, SortOrder: order, Notes: notes, Origin: domain.OriginCustom,
		}
		id = d.ID

		return s.repo.InsertDoc(ctx, &d)
	})
	if err != nil {
		return domain.AdminDoc{}, mapErr(err, "document")
	}

	d, err := s.repo.DocByID(ctx, p.ID, id)

	return d, mapErr(err, "document")
}

// UpdateDoc applies a PATCH under the project lock (read, merge, write).
func (s *Service) UpdateDoc(ctx context.Context, p pd.Project, id uuid.UUID, req domain.UpdateAdminDocRequest) (domain.AdminDoc, error) {
	spec := s.spec(p)

	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		d, err := s.repo.DocByID(ctx, p.ID, id)
		if err != nil {
			return err
		}

		if req.DocumentName != nil {
			if d.DocumentName, err = cleanText("document_name", *req.DocumentName, maxTitleRunes, true); err != nil {
				return err
			}
		}
		if req.SubjectRole != nil {
			subject := strings.TrimSpace(*req.SubjectRole)
			if err := checkSubject(spec, subject); err != nil {
				return err
			}
			d.SubjectRole = subject
		}
		if req.QuantityDetail != nil {
			if d.QuantityDetail, err = cleanText("quantity_detail", *req.QuantityDetail, maxQuantityRunes, false); err != nil {
				return err
			}
		}
		if req.NominalCost != nil {
			if err := validateMoney("nominal_cost", *req.NominalCost); err != nil {
				return err
			}
			d.NominalCost = *req.NominalCost
		}
		if req.Notes != nil {
			if d.Notes, err = cleanNote("notes", *req.Notes, maxNoteRunes); err != nil {
				return err
			}
		}

		return s.repo.UpdateDoc(ctx, &d)
	})
	if err != nil {
		return domain.AdminDoc{}, mapErr(err, "document")
	}

	d, err := s.repo.DocByID(ctx, p.ID, id)

	return d, mapErr(err, "document")
}

func (s *Service) SetDocStatus(ctx context.Context, p pd.Project, id uuid.UUID, status domain.DocStatus) (domain.AdminDoc, error) {
	if !status.Valid() {
		return domain.AdminDoc{}, invalidInput("status", "must be one of belum, selesai")
	}
	if err := s.repo.SetDocStatus(ctx, p.ID, id, status); err != nil {
		return domain.AdminDoc{}, mapErr(err, "document")
	}

	d, err := s.repo.DocByID(ctx, p.ID, id)

	return d, mapErr(err, "document")
}

// DeleteDoc removes the document; a budget line that points at it keeps its
// amount and loses the link (ON DELETE SET NULL).
func (s *Service) DeleteDoc(ctx context.Context, p pd.Project, id uuid.UUID) error {
	return mapErr(s.repo.DeleteDoc(ctx, p.ID, id), "document")
}

func (s *Service) ReorderDocs(ctx context.Context, p pd.Project, req domain.ReorderRequest) (domain.AdminDocList, error) {
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		n, err := s.repo.CountDocs(ctx, p.ID)
		if err != nil {
			return err
		}
		if err := checkIDs(req.IDs, n); err != nil {
			return err
		}
		changed, err := s.repo.ReorderDocs(ctx, p.ID, req.IDs)
		if err != nil {
			return err
		}
		if changed != n {
			return invalidInput("ids", "must list every id exactly once")
		}

		return nil
	})
	if err != nil {
		return domain.AdminDocList{}, mapErr(err, "document")
	}

	return s.docList(ctx, p.ID, domain.DocFilter{})
}

// ApplyDocTemplate is POST /admin-docs/apply-template.
func (s *Service) ApplyDocTemplate(ctx context.Context, p pd.Project, req domain.ApplyDocsRequest) (domain.GenerateResult, error) {
	mode, err := parseMode(req.Mode, false)
	if err != nil {
		return domain.GenerateResult{}, err
	}
	if len(domain.DocsFor(p.EventType)) == 0 {
		return domain.GenerateResult{}, invalidTemplate("admin_docs")
	}

	return s.applyDocs(ctx, p, mode, req.Confirm)
}

// applyDocs inserts the standard documents of the event type. With
// replace_template the template documents are deleted first; that needs
// confirm when one is done or a budget line points at it, and the response
// warns (budget_link_lost) that those lines keep their amounts but lose the link.
func (s *Service) applyDocs(ctx context.Context, p pd.Project, mode string, confirm bool) (domain.GenerateResult, error) {
	templates := domain.DocsFor(p.EventType)
	res := domain.GenerateResult{Warnings: []pd.Notice{}}
	linkedLost := 0

	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		existing, err := s.repo.CountDocs(ctx, p.ID)
		if err != nil {
			return err
		}

		var present map[string]bool
		switch mode {
		case domain.ModeAppend:
			if present, err = s.repo.DocTemplateKeys(ctx, p.ID); err != nil {
				return err
			}
		case domain.ModeReplaceTemplate:
			finished, linked, err := s.repo.DocsProgress(ctx, p.ID)
			if err != nil {
				return err
			}
			if finished+linked > 0 && !confirm {
				return confirmRequired()
			}
			linkedLost = linked
			if res.Deleted, err = s.repo.DeleteTemplateDocs(ctx, p.ID); err != nil {
				return err
			}
		}

		var todo []domain.DocTemplate
		for _, t := range templates {
			if present[t.Key] {
				res.Skipped++
			} else {
				todo = append(todo, t)
			}
		}
		if existing-res.Deleted+len(todo) > maxDocs {
			return apierr.Conflict("admin_doc_limit_reached", fmt.Sprintf("at most %d documents per project", maxDocs))
		}

		next, err := s.repo.NextDocSortOrder(ctx, p.ID)
		if err != nil {
			return err
		}
		for i, t := range todo {
			key := t.Key
			d := domain.AdminDoc{
				ID: uuid.New(), ProjectID: p.ID, DocumentName: t.Name, SubjectRole: t.Subject, QuantityDetail: t.Quantity,
				NominalCost: t.Cost, Status: domain.DocBelum, SortOrder: next + i, Notes: t.Note, TemplateKey: &key, Origin: t.Origin,
			}
			inserted, err := s.repo.InsertTemplateDoc(ctx, &d)
			if err != nil {
				return err
			}
			if inserted {
				res.Created++
			} else {
				res.Skipped++
			}
		}

		return nil
	})
	if err != nil {
		return domain.GenerateResult{}, mapErr(err, "document")
	}

	if linkedLost > 0 {
		res.Warnings = append(res.Warnings, notice("budget_link_lost", "budget lines that pointed at the replaced documents keep their amounts but lost the link", linkedLost))
	}

	return res, nil
}
