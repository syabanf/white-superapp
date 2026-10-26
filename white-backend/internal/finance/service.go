// Package finance wires the finance service: chart of accounts, fiscal
// calendar, journals with automatic posting, cash and bank, payables and
// receivables, reports, reimbursement claims and financial analysis.
//
// Env: GRPC_ADDR (default :9007 from cmd/finance), IDENTITY_ADDR,
// INTEGRATIONS_ADDR, HR_ADDR, FINANCE_UPLOAD_DIR (receipts, default
// ./.uploads/finance), APP_BASE_URL (OpenRouter referer).
package finance

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	_ "time/tzdata" // Asia/Jakarta must resolve inside slim containers

	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/clients"
	grpcapi "github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/grpc"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/xlsx"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const Schema = "finance"

func Migrations() fs.FS {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

func Spec() pkgapp.Spec {
	return pkgapp.Spec{
		Name:       "finance",
		Schema:     Schema,
		Migrations: Migrations(),
		Register:   register,
		Worker:     worker,
	}
}

type wiring struct {
	deps         app.Deps
	integrations *clients.Integrations
	closers      []func() error
}

func (w *wiring) close() {
	for _, c := range w.closers {
		_ = c()
	}
}

func wire(d pkgapp.Deps) (*wiring, error) {
	w := &wiring{}
	dial := func(env, fallback string) (*grpc.ClientConn, error) {
		conn, err := grpcx.Dial(config.Env(env, fallback))
		if err != nil {
			w.close()
			return nil, err
		}
		w.closers = append(w.closers, conn.Close)
		return conn, nil
	}
	identityConn, err := dial("IDENTITY_ADDR", "localhost:9001")
	if err != nil {
		return nil, err
	}
	integrationsConn, err := dial("INTEGRATIONS_ADDR", "localhost:9002")
	if err != nil {
		return nil, err
	}
	hrConn, err := dial("HR_ADDR", "localhost:9008")
	if err != nil {
		return nil, err
	}
	w.integrations = clients.NewIntegrations(integrationsConn)
	w.deps = app.Deps{
		Store:     postgres.NewStore(d.Pool),
		Identity:  clients.NewIdentity(identityConn),
		HR:        clients.NewHR(hrConn),
		Storage:   storage.NewLocal(config.Env("FINANCE_UPLOAD_DIR", "")),
		Insights:  openrouter.NewSource(w.integrations.Credentials, openrouter.Options{Referer: config.Env("APP_BASE_URL", "")}),
		Events:    d.Events,
		ReadSheet: xlsx.ReadCoaSheet,
		Clock:     clock.Real{},
		Logger:    d.Logger,
	}
	return w, nil
}

func register(ctx context.Context, d pkgapp.Deps, s *grpc.Server) error {
	w, err := wire(d)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		w.close()
	}()
	grpcapi.Register(s, grpcapi.Services{
		Accounts:       app.NewAccounts(w.deps),
		Fiscal:         app.NewFiscal(w.deps),
		Journals:       app.NewJournals(w.deps),
		CashBank:       app.NewCashBank(w.deps),
		Ledgers:        app.NewLedgers(w.deps),
		Reports:        app.NewReports(w.deps),
		Reimbursements: app.NewReimbursements(w.deps),
		Analysis:       app.NewAnalysis(w.deps),
	})
	return nil
}

// Subjects the worker consumes.
var subjects = []string{
	events.HRPayrollPaid, events.SalesInvoiceIssued, events.SalesPaymentRecorded,
	events.ERPTickDaily, events.UserDeleted, events.IntegrationUpdated,
}

// worker posts payroll and sales journals, reminds approvers daily and
// forgets deleted users.
func worker(ctx context.Context, d pkgapp.Deps) error {
	if d.Bus == nil {
		<-ctx.Done()
		return nil
	}
	w, err := wire(d)
	if err != nil {
		return err
	}
	defer w.close()
	h := NewHandler(w.deps, w.integrations.Invalidate)
	return d.Bus.Subscribe(ctx, "finance-events", subjects, h.Handle)
}

// Handler routes one event to its use case.
type Handler struct {
	deps       app.Deps
	autoPost   *app.AutoPost
	reimb      *app.Reimbursements
	invalidate func()
}

func NewHandler(deps app.Deps, invalidate func()) *Handler {
	return &Handler{deps: deps, autoPost: app.NewAutoPost(deps), reimb: app.NewReimbursements(deps), invalidate: invalidate}
}

// Handle acknowledges events that can never succeed (bad payload,
// validation failure) after logging them; other errors are redelivered.
func (h *Handler) Handle(ctx context.Context, msg events.Message) error {
	err := h.handle(ctx, msg)
	if err != nil && (errors.Is(err, app.ErrPermanent) || apperr.Is(err, apperr.Validation)) {
		h.deps.Logger.Warn("event dropped", "subject", msg.Subject, "err", err)
		return nil
	}
	return err
}

func decode(msg events.Message, v any) error {
	if err := msg.Decode(v); err != nil {
		return fmt.Errorf("%w: %v", app.ErrPermanent, err)
	}
	return nil
}

func (h *Handler) handle(ctx context.Context, msg events.Message) error {
	switch msg.Subject {
	case events.HRPayrollPaid:
		var p domain.PayrollPaid
		if err := decode(msg, &p); err != nil {
			return err
		}
		results, err := h.autoPost.PayrollPaid(ctx, p)
		if err == nil {
			h.deps.Logger.Info("payroll journals", "run", p.RunID, "result", domain.Summarize(results...))
		}
		return err
	case events.SalesInvoiceIssued:
		var p domain.SalesInvoiceIssued
		if err := decode(msg, &p); err != nil {
			return err
		}
		_, err := h.autoPost.SalesInvoiceIssued(ctx, p)
		return err
	case events.SalesPaymentRecorded:
		var p domain.SalesPaymentRecorded
		if err := decode(msg, &p); err != nil {
			return err
		}
		_, err := h.autoPost.SalesPaymentRecorded(ctx, p)
		return err
	case events.ERPTickDaily:
		var p events.TickPayload
		if err := decode(msg, &p); err != nil {
			return err
		}
		n, err := h.reimb.RemindApprovers(ctx, p.RunAt)
		if err == nil && n > 0 {
			h.deps.Logger.Info("reimbursement reminders", "claims", n)
		}
		return err
	case events.UserDeleted:
		var p events.UserDeletedPayload
		if err := decode(msg, &p); err != nil {
			return err
		}
		return h.reimb.ForgetUser(ctx, p.UserID)
	case events.IntegrationUpdated:
		if h.invalidate != nil {
			h.invalidate()
		}
	}
	return nil
}
