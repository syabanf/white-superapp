package grpcapi

import (
	"context"
	"testing"

	financev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/finance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"google.golang.org/protobuf/types/known/emptypb"
)

func as(perms ...string) context.Context {
	return authn.WithPrincipal(context.Background(), authn.Principal{UserID: "u1", Role: authn.RoleMember, Perms: perms})
}

// Permission checks run before any use case, so empty services suffice.
func TestPermissions(t *testing.T) {
	svc := Services{}
	acc, journals, reimb := &accountServer{svc: svc}, &journalServer{svc: svc}, &reimbursementServer{svc: svc}
	pay := &payableServer{ledger: ledger{svc: svc, kind: domain.KindAP}}
	cases := []struct {
		name string
		call func() error
	}{
		{"list accounts needs finance.read", func() error {
			_, err := acc.ListAccounts(as(authn.PermSelfService), &financev1.ListAccountsRequest{})
			return err
		}},
		{"create account needs finance.write", func() error {
			_, err := acc.CreateAccount(as(authn.PermFinanceRead), &financev1.AccountInput{})
			return err
		}},
		{"post journal needs finance.approve", func() error {
			_, err := journals.PostJournalEntry(as(authn.PermFinanceWrite), &financev1.IdRequest{})
			return err
		}},
		{"create posted journal needs finance.approve", func() error {
			_, err := journals.CreateJournalEntry(as(authn.PermFinanceWrite), &financev1.JournalEntryInput{Post: true})
			return err
		}},
		{"void payment needs finance.approve", func() error {
			_, err := pay.VoidApPayment(as(authn.PermFinanceWrite), &financev1.VoidPaymentRequest{})
			return err
		}},
		{"my claims need self.service", func() error {
			_, err := reimb.ListMyClaims(as(authn.PermFinanceRead), &financev1.ListMyClaimsRequest{})
			return err
		}},
		{"pay claims needs finance.approve", func() error {
			_, err := reimb.PayClaims(as(authn.PermFinanceRead), &financev1.PayClaimsRequest{})
			return err
		}},
		{"policy needs finance.approve", func() error {
			_, err := reimb.UpdatePolicy(as(authn.PermFinanceWrite), &financev1.PolicyInput{})
			return err
		}},
		{"summary needs a finance or self-service key", func() error {
			_, err := reimb.GetMonthlySummary(as(authn.PermHRRead), &financev1.MonthlySummaryRequest{})
			return err
		}},
		{"inbox needs a session", func() error { _, err := reimb.ListApprovalInbox(context.Background(), &emptypb.Empty{}); return err }},
	}
	for _, c := range cases {
		err := c.call()
		if !apperr.Is(err, apperr.Unauthorized) && !apperr.Is(err, apperr.Unauthenticated) {
			t.Errorf("%s: got %v", c.name, err)
		}
	}
}

func TestEnumAndMoneyConversions(t *testing.T) {
	if claimStatusPB(domain.ClaimDraft) != financev1.ClaimStatus_CLAIM_DRAFT || claimStatusFrom(financev1.ClaimStatus_APPROVED_MANAGER) != domain.ClaimApprovedManager {
		t.Fatal("claim status")
	}
	if categoryFrom(categoryPB(domain.CategoryOperasional)) != domain.CategoryOperasional || categoryFrom(financev1.ClaimCategory_CLAIM_CATEGORY_UNSPECIFIED) != "" {
		t.Fatal("category")
	}
	if sideFrom(sidePB(domain.Credit)) != domain.Credit || docStatusFrom(docStatusPB(domain.StatusVoid)) != domain.StatusVoid || entryTypeFrom(entryTypePB("AUTO")) != domain.EntryAuto {
		t.Fatal("journal enums")
	}
	if _, err := money("amount", "12,5"); !apperr.Is(err, apperr.Validation) || apperr.As(err).FieldErrors["amount"] == nil {
		t.Fatalf("money field error %v", err)
	}
}

func TestContraAccountNormalBalance(t *testing.T) {
	accDep := accountPB(&domain.Account{ID: "a", Code: "1201002", TypeCode: domain.TypeAsset, NormalBalance: domain.Debit, IsContra: true})
	cash := accountPB(&domain.Account{ID: "b", Code: "1101001", TypeCode: domain.TypeAsset, NormalBalance: domain.Debit})
	if accDep.NormalBalance != financev1.EntrySide_CREDIT || cash.NormalBalance != financev1.EntrySide_DEBIT {
		t.Fatalf("normal balance contra=%v cash=%v", accDep.NormalBalance, cash.NormalBalance)
	}
}
