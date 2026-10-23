package usecase

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/apierr"
	pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"
	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannervendor/domain"
)

func vendorLimitReached() *apierr.Error {
	return apierr.Conflict("vendor_limit_reached", "at most 200 vendors per project")
}

// VendorFilter is the query of GET /vendors. Scope: empty = every session, "all" = the general vendors only, any
// other scope_key = that session (the rule of menu 02).
type VendorFilter struct{ Scope, Status, Category string }

func (f VendorFilter) validate() error {
	if f.Scope != "" && !keyPattern.MatchString(f.Scope) {
		return invalidInput("scope", "must be a session scope_key")
	}
	if f.Status != "" && !domain.VendorStatus(f.Status).Valid() {
		return invalidInput("status", "must be one of rencana, negosiasi, fix, batal")
	}
	if f.Category != "" && !keyPattern.MatchString(f.Category) {
		return invalidInput("category", "must be a category key")
	}

	return nil
}

func (f VendorFilter) keep(v domain.Vendor) bool {
	return (f.Scope == "" || v.SubEventScope == f.Scope) &&
		(f.Status == "" || string(v.Status) == f.Status) &&
		(f.Category == "" || v.CategoryKey == f.Category)
}

// vendorData is the three reads the overview is built from.
type vendorData struct {
	vendors  []domain.VendorRow
	scenario []domain.ScenarioRow
	lines    []domain.LineRow
}

func (s *Service) readVendorData(ctx context.Context, p pd.Project) (vendorData, error) {
	var d vendorData
	var err error
	if d.vendors, err = s.repo.ListVendors(ctx, p.ID); err != nil {
		return d, err
	}
	if d.scenario, err = s.repo.ActiveScenarioRows(ctx, p.ID); err != nil {
		return d, err
	}
	d.lines, err = s.repo.Lines(ctx, p.ID)

	return d, err
}

// overview builds the whole-project view: KPI, categories, limit groups and every vendor.
func (s *Service) overview(ctx context.Context, p pd.Project) (domain.VendorOverview, error) {
	d, err := s.readVendorData(ctx, p)
	if err != nil {
		return domain.VendorOverview{}, err
	}
	cats, _ := s.specs(p)

	return domain.BuildOverview(cats, p.SkippedVendorCategories, d.vendors, d.scenario, d.lines), nil
}

func findVendor(ov domain.VendorOverview, id uuid.UUID) (domain.Vendor, bool) {
	for _, v := range ov.Vendors {
		if v.ID == id {
			return v, true
		}
	}

	return domain.Vendor{}, false
}

// ListVendors is GET /vendors (04 BE 3.1): the KPI, the categories and the limit groups always describe the whole
// project; only the vendor list is filtered.
func (s *Service) ListVendors(ctx context.Context, p pd.Project, f VendorFilter) (domain.VendorOverview, error) {
	if err := f.validate(); err != nil {
		return domain.VendorOverview{}, err
	}
	ov, err := s.overview(ctx, p)
	if err != nil {
		return domain.VendorOverview{}, err
	}
	kept := make([]domain.Vendor, 0, len(ov.Vendors))
	for _, v := range ov.Vendors {
		if f.keep(v) {
			kept = append(kept, v)
		}
	}
	ov.Vendors = kept

	return ov, nil
}

// GetVendor is GET /vendors/:id: the vendor with its limit and the line that points at it.
func (s *Service) GetVendor(ctx context.Context, p pd.Project, id uuid.UUID) (domain.Vendor, error) {
	ov, err := s.overview(ctx, p)
	if err != nil {
		return domain.Vendor{}, err
	}
	v, ok := findVendor(ov, id)
	if !ok {
		return domain.Vendor{}, apierr.NotFound("vendor")
	}

	return v, nil
}

// vendorRow validates a POST or PUT body and returns the row to store (without id and order). The checks that need
// the database (the session exists, the limit of vendors) run in the transaction.
func (s *Service) vendorRow(p pd.Project, req domain.VendorRequest) (domain.VendorRow, error) {
	var row domain.VendorRow
	var err error
	cats, _ := s.specs(p)

	if row.SubEventScope, err = normalizeScope(req.SubEventScope); err != nil {
		return row, err
	}
	if row.CategoryKey, row.CategoryLabel, err = resolveCategory(cats, req.CategoryKey, req.CategoryLabel); err != nil {
		return row, err
	}
	if row.VendorName, err = cleanText("vendor_name", req.VendorName, maxVendorNameRunes, true); err != nil {
		return row, err
	}
	if row.PICName, err = optionalText("pic_name", req.PICName, maxPICRunes); err != nil {
		return row, err
	}
	if row.PhoneNumber, row.FormattedWANumber, err = cleanPhone("phone_number", req.PhoneNumber); err != nil {
		return row, err
	}
	if row.PortfolioURL, err = cleanURL("portfolio_url", req.PortfolioURL); err != nil {
		return row, err
	}
	if err = validateMoney("budget_limit", req.BudgetLimit); err != nil {
		return row, err
	}
	if err = validateMoney("agreed_price", req.AgreedPrice); err != nil {
		return row, err
	}
	row.BudgetLimit, row.AgreedPrice = req.BudgetLimit, req.AgreedPrice
	if row.Status, err = vendorStatus(req.Status, req.AgreedPrice); err != nil {
		return row, err
	}
	row.Notes, err = optionalNote("notes", req.Notes, maxNoteRunes)

	return row, err
}

// vendorStatus defaults to rencana; fix needs a deal.
func vendorStatus(status domain.VendorStatus, agreed int64) (domain.VendorStatus, error) {
	if status == "" {
		status = domain.VendorPlanned
	}
	if !status.Valid() {
		return "", invalidInput("status", "must be one of rencana, negosiasi, fix, batal")
	}
	if status == domain.VendorFixed && agreed <= 0 {
		return "", invalidInput("agreed_price", "must be greater than 0 when status is fix")
	}

	return status, nil
}

// resolveCategory: a registry key, or a custom label whose key is derived by CategoryKeyFromLabel. A label that derives a
// registry key merges with that category and takes its label.
func resolveCategory(cats []domain.CategorySpec, rawKey, rawLabel *string) (key, label string, err error) {
	if rawKey != nil {
		key = strings.TrimSpace(*rawKey)
	}
	if rawLabel != nil {
		label = strings.TrimSpace(*rawLabel)
	}
	find := func(k string) (domain.CategorySpec, bool) {
		for _, c := range cats {
			if c.Key == k {
				return c, true
			}
		}

		return domain.CategorySpec{}, false
	}

	if key != "" {
		c, ok := find(key)
		if !ok {
			return "", "", invalidInput("category_key", "must be a registry category key")
		}

		return c.Key, c.Label, nil
	}
	if label == "" {
		return "", "", invalidInput("category_label", "required when category_key is empty")
	}
	if label, err = cleanText("category_label", label, maxCategoryLabelRunes, true); err != nil {
		return "", "", err
	}
	key = domain.CategoryKeyFromLabel(label)
	if c, ok := find(key); ok {
		return c.Key, c.Label, nil
	}

	return key, label, nil
}

// CreateVendor is POST /vendors: a candidate (status rencana unless the body says otherwise).
func (s *Service) CreateVendor(ctx context.Context, p pd.Project, req domain.VendorRequest) (domain.VendorResult, error) {
	row, err := s.vendorRow(p, req)
	if err != nil {
		return domain.VendorResult{}, err
	}
	row.ID = uuid.New()

	res := domain.VendorResult{Warnings: []pd.Notice{}}
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		n, err := s.repo.CountVendors(ctx, p.ID)
		if err != nil {
			return err
		}
		if n >= maxVendors {
			return vendorLimitReached()
		}
		if err := s.checkScope(ctx, p.ID, row.SubEventScope); err != nil {
			return err
		}
		if err := s.repo.InsertVendor(ctx, p.ID, &row); err != nil {
			return err
		}

		res.Vendor, err = s.vendorDTO(ctx, p, row.ID)

		return err
	})
	if err != nil {
		return domain.VendorResult{}, mapErr(err, "vendor")
	}

	return res, nil
}

func (s *Service) vendorDTO(ctx context.Context, p pd.Project, id uuid.UUID) (domain.Vendor, error) {
	ov, err := s.overview(ctx, p)
	if err != nil {
		return domain.Vendor{}, err
	}
	v, ok := findVendor(ov, id)
	if !ok {
		return domain.Vendor{}, domain.ErrNotFound
	}

	return v, nil
}

// UpdateVendor is PUT /vendors/:id: it replaces the vendor entirely. Leaving fix releases the budget link; the estimate of
// the line is never changed here (only lock-deal does that).
func (s *Service) UpdateVendor(ctx context.Context, p pd.Project, id uuid.UUID, req domain.VendorRequest) (domain.VendorResult, error) {
	row, err := s.vendorRow(p, req)
	if err != nil {
		return domain.VendorResult{}, err
	}
	row.ID = id

	res := domain.VendorResult{Warnings: []pd.Notice{}}
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		res.Warnings = []pd.Notice{}
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		old, err := s.repo.VendorByID(ctx, p.ID, id)
		if err != nil {
			return err
		}
		if err := s.checkScope(ctx, p.ID, row.SubEventScope); err != nil {
			return err
		}
		row.SortOrder = old.SortOrder
		if err := s.repo.UpdateVendor(ctx, p.ID, &row); err != nil {
			return err
		}
		if res.Warnings, err = s.releaseOnLeavingFix(ctx, p, old, row.Status, res.Warnings); err != nil {
			return err
		}
		if res.Vendor, err = s.vendorDTO(ctx, p, id); err != nil {
			return err
		}
		if old.AgreedPrice != row.AgreedPrice && row.AgreedPrice > 0 && res.Vendor.Payment != nil && res.Vendor.Payment.Estimate != row.AgreedPrice {
			res.Warnings = append(res.Warnings, notice("estimate_differs", "the estimate of the linked budget line differs from the agreed price; the line was not changed", 1))
		}

		return nil
	})
	if err != nil {
		return domain.VendorResult{}, mapErr(err, "vendor")
	}

	return res, nil
}

// SetVendorStatus is PATCH /vendors/:id/status, the one-click pipeline move.
func (s *Service) SetVendorStatus(ctx context.Context, p pd.Project, id uuid.UUID, req domain.VendorStatusRequest) (domain.VendorResult, error) {
	if !req.Status.Valid() {
		return domain.VendorResult{}, invalidInput("status", "must be one of rencana, negosiasi, fix, batal")
	}

	res := domain.VendorResult{Warnings: []pd.Notice{}}
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		res.Warnings = []pd.Notice{}
		if err := s.repo.LockProject(ctx, p.ID); err != nil {
			return err
		}
		old, err := s.repo.VendorByID(ctx, p.ID, id)
		if err != nil {
			return err
		}
		if _, err := vendorStatus(req.Status, old.AgreedPrice); err != nil {
			return err
		}
		if err := s.repo.SetVendorStatus(ctx, p.ID, id, req.Status); err != nil {
			return err
		}
		if res.Warnings, err = s.releaseOnLeavingFix(ctx, p, old, req.Status, res.Warnings); err != nil {
			return err
		}
		res.Vendor, err = s.vendorDTO(ctx, p, id)

		return err
	})
	if err != nil {
		return domain.VendorResult{}, mapErr(err, "vendor")
	}

	return res, nil
}

// releaseOnLeavingFix unlinks the budget line of a vendor that is no longer fix (04 BE 2.2, step 4); the line stays as a
// plain line. It adds budget_link_released when it released something.
func (s *Service) releaseOnLeavingFix(ctx context.Context, p pd.Project, old domain.VendorRow, next domain.VendorStatus, warnings []pd.Notice) ([]pd.Notice, error) {
	if old.Status != domain.VendorFixed || next == domain.VendorFixed {
		return warnings, nil
	}
	n, err := s.repo.ReleaseVendorLines(ctx, p.ID, old.ID)
	if err != nil || n == 0 {
		return warnings, err
	}

	return append(warnings, notice("budget_link_released", "the vendor is no longer fix, so its budget line is no longer linked to it", n)), nil
}

// DeleteVendor is DELETE /vendors/:id. Tasks, budget lines, shopping items and agendas that point at the vendor keep
// existing with an empty vendor (the foreign keys SET NULL).
func (s *Service) DeleteVendor(ctx context.Context, p pd.Project, id uuid.UUID) error {
	return mapErr(s.repo.DeleteVendor(ctx, p.ID, id), "vendor")
}
