package app

import (
	"context"
	"errors"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// JobOpenings manages the openings and the public career listing.
type JobOpenings struct{ core }

func NewJobOpenings(d Deps) *JobOpenings { return &JobOpenings{core{d}} }

func (j *JobOpenings) List(ctx context.Context, status, search string) ([]domain.JobOpening, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	return j.Store.ListJobOpenings(ctx, status, search)
}

func (j *JobOpenings) Get(ctx context.Context, id string) (*domain.JobOpening, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	o, err := j.Store.GetJobOpening(ctx, id)
	if err == nil && o == nil {
		err = notFound(MsgJobNotFound)
	}
	return o, err
}

// departmentName fills the department label from hr when only the id was
// sent; a missing hr keeps the NüHabit default.
func (j *JobOpenings) departmentName(ctx context.Context, in *domain.JobOpeningInput) error {
	if in.DepartmentID == nil || *in.DepartmentID == "" {
		return nil
	}
	depts, err := j.Directory.Departments(ctx)
	if err != nil {
		j.log().Warn("hr departments unavailable", "err", err)
		return nil
	}
	for _, d := range depts {
		if d.ID == *in.DepartmentID {
			if strings.TrimSpace(in.Department) == "" {
				in.Department = d.Name
			}
			return nil
		}
	}
	return apperr.Field("department_id", "Departemen tidak ditemukan")
}

func slugTaken(err error) error {
	var u *domain.ErrUnique
	if errors.As(err, &u) {
		return apperr.Field("slug", "Slug lowongan sudah dipakai")
	}
	return err
}

// Create is SaveJobOpening without an id.
func (j *JobOpenings) Create(ctx context.Context, in domain.JobOpeningInput) (*domain.JobOpening, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	if err := j.departmentName(ctx, &in); err != nil {
		return nil, err
	}
	o, err := domain.NormalizeJobOpening(in, nil, j.now())
	if err != nil {
		return nil, err
	}
	o.ID, o.CreatedBy = newID(), a.id()
	if err := j.Store.InsertJobOpening(ctx, o); err != nil {
		return nil, slugTaken(err)
	}
	return j.Store.GetJobOpening(ctx, o.ID)
}

// Update replaces the opening; the first publication time is kept.
func (j *JobOpenings) Update(ctx context.Context, id string, in domain.JobOpeningInput) (*domain.JobOpening, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	prev, err := j.Store.GetJobOpening(ctx, id)
	if err != nil {
		return nil, err
	}
	if prev == nil {
		return nil, notFound(MsgJobNotFound)
	}
	if err := j.departmentName(ctx, &in); err != nil {
		return nil, err
	}
	o, err := domain.NormalizeJobOpening(in, prev, j.now())
	if err != nil {
		return nil, err
	}
	o.ID, o.UpdatedBy = id, a.id()
	if _, err := j.Store.UpdateJobOpening(ctx, o); err != nil {
		return nil, slugTaken(err)
	}
	return j.Store.GetJobOpening(ctx, id)
}

func (j *JobOpenings) Delete(ctx context.Context, id string) error {
	a, err := staff(ctx, true)
	if err != nil {
		return err
	}
	ok, err := j.Store.DeleteJobOpening(ctx, id, a.UserID)
	if err == nil && !ok {
		err = notFound(MsgJobNotFound)
	}
	return err
}
