package grpcapi

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc"

	financev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/finance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
)

type auditSpy struct {
	domain.AuditRepo
	got []domain.AuditEntry
}

func (a *auditSpy) InsertAuditEntry(_ context.Context, e domain.AuditEntry) error {
	a.got = append(a.got, e)
	return nil
}

func TestAuditInterceptorRecordsChangesOnly(t *testing.T) {
	spy := &auditSpy{}
	run := AuditInterceptor(spy, nil)
	ctx := authn.WithPrincipal(context.Background(), authn.Principal{UserID: "u1", Name: "Rani"})
	call := func(method string, req any, resp any, err error) {
		t.Helper()
		_, _ = run(ctx, req, &grpc.UnaryServerInfo{FullMethod: method}, func(context.Context, any) (any, error) { return resp, err })
	}

	call("/white.finance.v1.PeriodEndService/ListDeferrals", &financev1.ListDeferralsRequest{}, &financev1.ListDeferralsResponse{}, nil)
	call("/white.sales.v1.BillingService/RecordPayment", &financev1.WriteOffRequest{}, nil, nil)
	if len(spy.got) != 0 {
		t.Fatalf("reads and other services stay out: %+v", spy.got)
	}

	call("/white.finance.v1.PeriodEndService/RemitTax", &financev1.TaxRemittance{Period: "2026-10"}, &financev1.TaxRemittance{Id: "r1", RemitNo: "SP-1"}, nil)
	e := spy.got[0]
	if e.ActorName != "Rani" || e.Action != "RemitTax" || e.DocID != "r1" || e.DocNo != "SP-1" || e.Result != domain.AuditOK {
		t.Fatalf("%+v", e)
	}

	call("/white.finance.v1.PeriodEndService/WriteOffInvoice", &financev1.WriteOffRequest{InvoiceId: "i1"}, nil, domain.Conflict("Ditolak"))
	e = spy.got[1]
	if e.DocID != "i1" || e.Result != domain.AuditRefused || e.Message != "Ditolak" {
		t.Fatalf("%+v", e)
	}
}

func TestAuditBodyDropsFilesAndSecrets(t *testing.T) {
	body := auditBody(&financev1.TaxExportFile{FileName: "a.csv", Data: []byte("isi rahasia")})
	if strings.Contains(body, "rahasia") || !strings.Contains(body, "a.csv") {
		t.Fatalf("%s", body)
	}
	if off := auditBody(&financev1.ControlSettings{}); !strings.Contains(off, "false") {
		t.Fatalf("switching off must show: %s", off)
	}
}
