package grpc

import (
	"context"
	"testing"

	salesv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/sales/v1"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Handlers reject callers before touching any use case, so nil services
// are enough here.
func TestPermissionGates(t *testing.T) {
	ctx := context.Background()
	member := authn.WithPrincipal(ctx, authn.Principal{UserID: "u", Role: authn.RoleMember, Perms: []string{authn.PermSalesRead}})
	leads := &leadServer{}
	if _, err := leads.ListLeads(ctx, &salesv1.ListLeadsRequest{}); !apperr.Is(err, apperr.Unauthenticated) {
		t.Fatalf("anonymous: %v", err)
	}
	if _, err := leads.CreateLead(member, &salesv1.LeadInput{}); !apperr.Is(err, apperr.Unauthorized) {
		t.Fatalf("read-only writes: %v", err)
	}
	if _, err := leads.SaveCustomField(member, &salesv1.CustomField{}); !apperr.Is(err, apperr.Unauthorized) {
		t.Fatalf("settings need ADMIN: %v", err)
	}
	if _, err := (&billingServer{}).IssueInvoice(member, &salesv1.IdRequest{}); !apperr.Is(err, apperr.Unauthorized) {
		t.Fatalf("issue needs finance.write: %v", err)
	}
	if _, err := (&chatServer{}).SaveChatbotSettings(member, &salesv1.ChatbotSettings{}); !apperr.Is(err, apperr.Unauthorized) {
		t.Fatalf("chatbot settings need ADMIN: %v", err)
	}
	if _, err := (&insightServer{}).SaveTargets(member, &salesv1.SaveTargetsRequest{}); !apperr.Is(err, apperr.Unauthorized) {
		t.Fatalf("targets need sales.approve: %v", err)
	}
	outsider := authn.WithPrincipal(ctx, authn.Principal{UserID: "u", Role: authn.RoleMember})
	if _, err := (&billingServer{}).GetInvoice(ctx, &salesv1.IdRequest{Id: "i"}); !apperr.Is(err, apperr.Unauthenticated) {
		t.Fatalf("get invoice anonymous: %v", err)
	}
	if _, err := (&billingServer{}).GetInvoice(outsider, &salesv1.IdRequest{Id: "i"}); !apperr.Is(err, apperr.Unauthorized) {
		t.Fatalf("get invoice needs sales.read or finance.read: %v", err)
	}
	if _, err := (&insightServer{}).GetDashboardSummary(outsider, &salesv1.DashboardRequest{}); !apperr.Is(err, apperr.Unauthorized) {
		t.Fatalf("dashboard needs sales.read: %v", err)
	}
	if _, err := (&automationServer{}).ListWorkflowRules(member, &emptypb.Empty{}); !apperr.Is(err, apperr.Unauthorized) {
		t.Fatalf("automation needs ADMIN: %v", err)
	}
}
