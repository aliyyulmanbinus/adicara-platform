package usecase

import (
	"context"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannertimeline/domain"
)

// SyncBudget copies the cost of a document into the budget (02 BE 4.5). It is
// idempotent and never overwrites a figure the user typed:
//
//   - the document is already linked to a budget line: nothing changes;
//   - a line with the matching (session, item_key) that no document points at
//     is linked and, when its estimate is 0, filled with the cost (a different
//     non-zero estimate stays and raises the warning estimate_differs);
//   - otherwise a new line is created under category "Legalitas".
//
// The document row is locked FOR UPDATE for the whole transaction: the budget
// table has no unique index on admin_doc_id, so without the lock a double click
// would create two lines.
func (s *Service) SyncBudget(ctx context.Context, p pd.Project, docID uuid.UUID, req domain.SyncBudgetRequest) (domain.SyncBudgetResponse, error) {
	var requested *string
	if req.SubEventScope != nil {
		scope, err := normalizeScope(*req.SubEventScope)
		if err != nil {
			return domain.SyncBudgetResponse{}, err
		}
		requested = &scope
	}

	res := domain.SyncBudgetResponse{Warnings: []pd.Notice{}}
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		res = domain.SyncBudgetResponse{Warnings: []pd.Notice{}}

		doc, err := s.repo.LockDoc(ctx, p.ID, docID)
		if err != nil {
			return err
		}
		if doc.NominalCost <= 0 {
			return invalidInput("nominal_cost", "must be greater than 0")
		}

		if linked, found, err := s.repo.LinkedBudgetItem(ctx, p.ID, docID); err != nil {
			return err
		} else if found {
			res.BudgetItemID = linked.ID

			return nil
		}

		scopes, _, err := s.scopeSet(ctx, p.ID)
		if err != nil {
			return err
		}
		scope, itemKey := pd.AllScope, ""
		if doc.TemplateKey != nil && *doc.TemplateKey == domain.KUADocKey {
			scope, itemKey = domain.KUABudgetScope, domain.KUABudgetItemKey
			if !scopes[scope] {
				scope = pd.AllScope // the mapped session was deleted
			}
		}
		if requested != nil {
			scope = *requested
			if !scopes[scope] {
				return errUnknownScope()
			}
		}

		if itemKey != "" {
			ref, found, err := s.repo.UnlinkedBudgetItem(ctx, p.ID, scope, itemKey)
			if err != nil {
				return err
			}
			if found {
				if err := s.repo.LinkBudgetItem(ctx, p.ID, ref.ID, docID, doc.NominalCost); err != nil {
					return err
				}
				res.BudgetItemID, res.Linked = ref.ID, true
				if ref.Estimate > 0 && ref.Estimate != doc.NominalCost {
					res.Warnings = append(res.Warnings, notice("estimate_differs", "the budget line keeps its own estimate, which differs from the document cost", 1))
				}

				return nil
			}
		}

		id, err := s.repo.InsertBudgetItem(ctx, p.ID, scope, doc.DocumentName, docID, doc.NominalCost)
		if err != nil {
			return err
		}
		res.BudgetItemID, res.Created = id, true

		return nil
	})
	if err != nil {
		return domain.SyncBudgetResponse{}, mapErr(err, "document")
	}

	return res, nil
}
