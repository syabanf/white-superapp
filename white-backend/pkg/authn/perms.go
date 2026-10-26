package authn

import (
	"context"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// Permission keys for the ERP workspace, `<module>.<action>`. ADMIN holds
// every permission; other users get the keys an admin assigns in identity.
// A key ending in ".*" grants every action of that module.
const (
	PermFinanceRead    = "finance.read"
	PermFinanceWrite   = "finance.write"
	PermFinanceApprove = "finance.approve"

	PermHRRead    = "hr.read"
	PermHRWrite   = "hr.write"
	PermHRApprove = "hr.approve"
	PermPayroll   = "hr.payroll"

	PermRecruitRead  = "recruitment.read"
	PermRecruitWrite = "recruitment.write"

	PermSalesRead    = "sales.read"
	PermSalesWrite   = "sales.write"
	PermSalesApprove = "sales.approve"

	// Every signed-in employee: own attendance, leave, reimbursement, tasks.
	PermSelfService = "self.service"
)

// PermissionCatalog is the list the admin UI offers, in display order.
var PermissionCatalog = []PermissionDef{
	{PermFinanceRead, "Keuangan: lihat"},
	{PermFinanceWrite, "Keuangan: input jurnal dan transaksi"},
	{PermFinanceApprove, "Keuangan: setujui dan posting"},
	{PermHRRead, "SDM: lihat"},
	{PermHRWrite, "SDM: kelola data karyawan"},
	{PermHRApprove, "SDM: setujui cuti, lembur, reimburse"},
	{PermPayroll, "SDM: payroll"},
	{PermRecruitRead, "Rekrutmen: lihat"},
	{PermRecruitWrite, "Rekrutmen: kelola kandidat"},
	{PermSalesRead, "Penjualan: lihat"},
	{PermSalesWrite, "Penjualan: kelola lead, deal, quotation"},
	{PermSalesApprove, "Penjualan: setujui diskon"},
	{PermSelfService, "Layanan mandiri karyawan"},
}

type PermissionDef struct {
	Key   string
	Label string
}

// IsKnownPermission accepts catalog keys and module wildcards.
func IsKnownPermission(key string) bool {
	for _, p := range PermissionCatalog {
		if p.Key == key {
			return true
		}
		if strings.HasSuffix(key, ".*") && strings.HasPrefix(p.Key, strings.TrimSuffix(key, "*")) {
			return true
		}
	}
	return false
}

// Can reports whether the principal holds perm.
func (p Principal) Can(perm string) bool {
	if p.IsAdmin() {
		return true
	}
	module, _, _ := strings.Cut(perm, ".")
	for _, have := range p.Perms {
		if have == perm || have == module+".*" {
			return true
		}
	}
	return false
}

// RequirePerm returns the caller when they hold perm, UNAUTHORIZED otherwise.
func RequirePerm(ctx context.Context, perm string) (Principal, error) {
	p, err := Require(ctx)
	if err != nil {
		return p, err
	}
	if !p.Can(perm) {
		return p, apperr.New(apperr.Unauthorized, "")
	}
	return p, nil
}
