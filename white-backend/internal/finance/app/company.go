package app

import (
	"context"
	"regexp"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// company is the saved profile, or the COMPANY_* env defaults until someone
// saves one. A read failure falls back too: a letterhead never blocks a document.
func (d Deps) company(ctx context.Context) domain.Company {
	c, err := d.Store.CompanyProfile(ctx)
	if err != nil && d.Logger != nil {
		d.Logger.Warn("company profile read failed", "err", err)
	}
	if c == nil {
		return d.Company
	}
	return *c
}

// CompanyProfile is the master data behind every letterhead and the bank
// account clients see on invoices and the portal.
type CompanyProfile struct{ Deps }

func NewCompanyProfile(d Deps) *CompanyProfile { return &CompanyProfile{d} }

func (p *CompanyProfile) Get(ctx context.Context) domain.Company { return p.company(ctx) }

var (
	digitsRe = regexp.MustCompile(`^[0-9 .-]+$`)
	emailRe  = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

func (p *CompanyProfile) Save(ctx context.Context, userID string, c domain.Company) (domain.Company, error) {
	c = domain.Company{LegalName: trim(c.LegalName), Address: trim(c.Address), Email: trim(c.Email), Phone: trim(c.Phone), NPWP: trim(c.NPWP),
		BankName: trim(c.BankName), BankAccountNo: trim(c.BankAccountNo), BankAccountName: trim(c.BankAccountName)}
	check := validate.New().Required("legal_name", c.LegalName, "Nama perusahaan wajib diisi")
	if c.BankAccountNo != "" {
		if !digitsRe.MatchString(c.BankAccountNo) {
			check.Add("bank_account_no", "Nomor rekening hanya berisi angka")
		}
		if c.BankName == "" {
			check.Add("bank_name", "Nama bank wajib diisi bila nomor rekening diisi")
		}
	}
	if c.Email != "" && !emailRe.MatchString(c.Email) {
		check.Add("email", "Email tidak valid")
	}
	if err := check.Err(); err != nil {
		return domain.Company{}, err
	}
	if err := p.Store.SaveCompanyProfile(ctx, c, userID); err != nil {
		return domain.Company{}, err
	}
	return c, nil
}
