package usecase

import (
	"context"

	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/domain"
)

// ExportContacts gathers the contact sheet (04 BE 2.7): the fixed vendors in order, with no price, limit or note, because
// the sheet is meant for the organiser or the committee.
func (s *Service) ExportContacts(ctx context.Context, p pd.Project) (domain.VendorExport, error) {
	today, err := s.today(p)
	if err != nil {
		return domain.VendorExport{}, err
	}
	_, sessions, err := s.scopeSet(ctx, p.ID)
	if err != nil {
		return domain.VendorExport{}, err
	}
	ov, err := s.overview(ctx, p)
	if err != nil {
		return domain.VendorExport{}, err
	}

	out := domain.VendorExport{Project: p, Today: today, Sessions: sessions, Vendors: []domain.Vendor{}}
	for _, v := range ov.Vendors {
		if v.Status == domain.VendorFixed {
			out.Vendors = append(out.Vendors, v)
		}
	}

	return out, nil
}

// ExportShopping gathers the shopping sheet: every item of the project in order.
func (s *Service) ExportShopping(ctx context.Context, p pd.Project) (domain.ShoppingExport, error) {
	today, err := s.today(p)
	if err != nil {
		return domain.ShoppingExport{}, err
	}
	_, sessions, err := s.scopeSet(ctx, p.ID)
	if err != nil {
		return domain.ShoppingExport{}, err
	}
	items, err := s.repo.ListShopping(ctx, p.ID)
	if err != nil {
		return domain.ShoppingExport{}, err
	}

	return domain.ShoppingExport{Project: p, Today: today, Sessions: sessions, Items: items, KPI: domain.ShoppingKPIOf(items)}, nil
}
