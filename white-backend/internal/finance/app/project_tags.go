package app

import (
	"context"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// ProjectTags manages tags and the profit report built from tagged journals.
type ProjectTags struct{ Deps }

func NewProjectTags(d Deps) *ProjectTags { return &ProjectTags{d} }

// checkTag refuses a tag that is missing or inactive; empty means no tag.
func checkTag(ctx context.Context, s domain.Store, id string) error {
	if id == "" {
		return nil
	}
	t, err := s.GetProjectTag(ctx, id)
	if err != nil {
		return err
	}
	if t == nil || !t.IsActive {
		return apperr.Field("tag_id", "Tag proyek tidak ditemukan atau nonaktif")
	}
	return nil
}

func (p *ProjectTags) List(ctx context.Context, activeOnly bool) ([]domain.ProjectTag, error) {
	return p.Store.ListProjectTags(ctx, activeOnly)
}

func (p *ProjectTags) Save(ctx context.Context, userID string, t domain.ProjectTag) (*domain.ProjectTag, error) {
	t.Name, t.ClientName, t.Notes, t.UpdatedBy = trim(t.Name), trim(t.ClientName), trim(t.Notes), userID
	if err := validate.New().Required("name", t.Name, "Nama proyek wajib diisi").Err(); err != nil {
		return nil, err
	}
	rows, err := p.Store.ListProjectTags(ctx, false)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.ID != t.ID && strings.EqualFold(r.Name, t.Name) && strings.EqualFold(r.ClientName, t.ClientName) {
			return nil, apperr.Field("name", "Proyek ini sudah ada untuk klien tersebut")
		}
	}
	if t.ID == "" {
		t.ID = ids.New()
	}
	if err := p.Store.SaveProjectTag(ctx, t); err != nil {
		return nil, err
	}
	return p.Store.GetProjectTag(ctx, t.ID)
}

// Delete removes a tag nothing refers to yet.
func (p *ProjectTags) Delete(ctx context.Context, id string) error {
	used, err := p.Store.ProjectTagInUse(ctx, id)
	if err != nil {
		return err
	}
	if used {
		return domain.Conflict("Tag sudah dipakai transaksi. Nonaktifkan saja.")
	}
	return p.Store.DeleteProjectTag(ctx, id)
}

// ProjectProfit is revenue and cost per tag. Together with the untagged row it
// adds up to the income statement of the same period.
func (p *ProjectTags) ProjectProfit(ctx context.Context, from, to, clientName string) ([]domain.ProjectProfitRow, string, string, error) {
	from, to, err := NewReports(p.Deps).period(from, to)
	if err != nil {
		return nil, "", "", err
	}
	rows, err := p.Store.ProjectProfit(ctx, from, to, trim(clientName))
	if err != nil {
		return nil, "", "", err
	}
	allocations, err := p.Store.ListProjectAllocations(ctx, from[:7], to[:7])
	if err != nil {
		return nil, "", "", err
	}
	return applyAllocations(rows, allocations, trim(clientName)), from, to, nil
}
