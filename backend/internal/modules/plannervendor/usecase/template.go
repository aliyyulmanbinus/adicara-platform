package usecase

import (
	"context"

	"github.com/google/uuid"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/domain"
)

// ApplyTemplate is POST /shopping-items/apply-template (04 BE 2.6). An event type has one template (the wedding's 37
// seserahan items); the other types only have categories, so they answer invalid_template.
//
//   - append (the default; also what the wizard runs) skips the items whose template_key is already there.
//   - replace_template deletes every template item and inserts the template anew. Items the user made are safe. It needs
//     confirm when a deleted item carries work of the user (a status, an actual price, a link or the souvenir mark), and
//     answers budget_link_lost when deleted items were rolled up into a budget line (the line recomputes without them).
func (s *Service) ApplyTemplate(ctx context.Context, p pd.Project, req domain.ShoppingTemplateRequest) (domain.ShoppingTemplateResult, error) {
	mode := req.Mode
	if mode == "" {
		mode = domain.ModeAppend
	}
	if mode != domain.ModeAppend && mode != domain.ModeReplaceTemplate {
		return domain.ShoppingTemplateResult{}, invalidInput("mode", "must be one of append, replace_template")
	}

	var res domain.ShoppingTemplateResult
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		var err error
		res, err = s.applyTemplate(ctx, p, mode, req.Confirm)

		return err
	})
	if err != nil {
		return domain.ShoppingTemplateResult{}, mapErr(err, "shopping item")
	}

	return res, nil
}

// applyTemplate runs in a transaction (the caller's, or the project-creation one for the wizard).
func (s *Service) applyTemplate(ctx context.Context, p pd.Project, mode string, confirm bool) (domain.ShoppingTemplateResult, error) {
	res := domain.ShoppingTemplateResult{Warnings: []pd.Notice{}}
	template := domain.TemplateFor(p.EventType)
	if len(template) == 0 {
		return res, invalidTemplate("shopping_list")
	}
	if err := s.repo.LockProject(ctx, p.ID); err != nil {
		return res, err
	}

	if mode == domain.ModeReplaceTemplate {
		impact, err := s.repo.TemplateImpact(ctx, p.ID)
		if err != nil {
			return res, err
		}
		if impact.CarryWork > 0 && !confirm {
			return res, confirmRequired()
		}
		if res.Deleted, err = s.repo.DeleteTemplateItems(ctx, p.ID); err != nil {
			return res, err
		}
		if impact.RollupLinks > 0 {
			res.Warnings = append(res.Warnings, notice("budget_link_lost", "shopping items that were rolled up into a budget line were replaced; the line recomputes without them", impact.RollupLinks))
		}
	}

	for _, it := range template {
		key := it.Key()
		inserted, err := s.repo.InsertTemplateItem(ctx, p.ID, &domain.ShoppingWrite{
			ID: uuid.New(), SubEventScope: pd.AllScope, Category: it.Category, ItemName: it.Name, QuantityX100: 100,
			EstimatedPrice: it.Price, Status: domain.ShoppingNotStarted, TemplateKey: &key,
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

	// Counted after the insert: all or nothing, the transaction is rolled back when the list would pass the limit.
	n, err := s.repo.CountShopping(ctx, p.ID)
	if err != nil {
		return res, err
	}
	if n > maxShoppingItems {
		return res, shoppingLimitReached()
	}

	return res, nil
}
