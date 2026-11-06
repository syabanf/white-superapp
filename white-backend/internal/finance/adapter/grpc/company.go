package grpcapi

import (
	"context"

	financev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/finance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/protobuf/types/known/emptypb"
)

type companyServer struct {
	financev1.UnimplementedCompanyServiceServer
	svc Services
}

func companyPB(c domain.Company) *financev1.CompanyProfile {
	return &financev1.CompanyProfile{LegalName: c.LegalName, Address: c.Address, Email: c.Email, Phone: c.Phone, Npwp: c.NPWP,
		BankName: c.BankName, BankAccountNo: c.BankAccountNo, BankAccountName: c.BankAccountName, BankLine: c.BankLine()}
}

// GetCompanyProfile: internal callers (Sales) or any signed-in user.
func (s *companyServer) GetCompanyProfile(ctx context.Context, _ *emptypb.Empty) (*financev1.CompanyProfile, error) {
	if grpcx.FromEdge(ctx) {
		if _, err := authn.Require(ctx); err != nil {
			return nil, err
		}
	}
	return companyPB(s.svc.Company.Get(ctx)), nil
}

func (s *companyServer) SaveCompanyProfile(ctx context.Context, r *financev1.CompanyProfile) (*financev1.CompanyProfile, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Company.Save(ctx, p.UserID, domain.Company{LegalName: r.LegalName, Address: r.Address, Email: r.Email, Phone: r.Phone, NPWP: r.Npwp,
		BankName: r.BankName, BankAccountNo: r.BankAccountNo, BankAccountName: r.BankAccountName})
	if err != nil {
		return nil, err
	}
	return companyPB(c), nil
}
