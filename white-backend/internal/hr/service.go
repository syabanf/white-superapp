// Package hr wires the hr service: employees and master data, attendance,
// schedules, leave, overtime, contracts, announcements, on/offboarding,
// payroll, loans, employee self service and HR reports.
//
// Env: IDENTITY_ADDR (default localhost:9001), HR_UPLOAD_DIR (selfies and
// employee documents, default ./.uploads/hr), HR_COMPANY_NAME (payslip and
// recap header, default Mixist).
package hr

import (
	"context"
	"embed"
	"io/fs"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/clients"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/docs"
	grpcapi "github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/grpc"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const Schema = "hr"

func Migrations() fs.FS {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

func Spec() pkgapp.Spec {
	return pkgapp.Spec{
		Name:       "hr",
		Schema:     Schema,
		Migrations: Migrations(),
		Register:   register,
		Worker:     worker,
	}
}

// Wire builds the use cases over a pool; the returned closer drops the
// identity connection.
func Wire(d pkgapp.Deps) (*grpcapi.Services, *clients.Identity, func(), error) {
	conn, err := grpcx.Dial(config.Env("IDENTITY_ADDR", "localhost:9001"))
	if err != nil {
		return nil, nil, nil, err
	}
	identity := clients.NewIdentity(conn)
	deps := app.Deps{
		Store:    postgres.NewStore(d.Pool),
		Identity: identity,
		Files:    storage.NewLocal(config.Env("HR_UPLOAD_DIR", ".uploads/hr")),
		Events:   d.Events,
		Clock:    clock.Real{},
		Logger:   d.Logger,
	}
	renderer := docs.Renderer{CompanyName: config.Env("HR_COMPANY_NAME", "Mixist")}
	leaves := app.NewLeaves(deps)
	announcements := app.NewAnnouncements(deps)
	svc := &grpcapi.Services{
		Deps:          deps,
		People:        app.NewPeople(deps),
		Attendance:    app.NewAttendance(deps, renderer),
		Schedules:     app.NewSchedules(deps),
		Leaves:        leaves,
		Overtime:      app.NewOvertime(deps),
		Contracts:     app.NewContracts(deps),
		Announcements: announcements,
		Lifecycle:     app.NewLifecycle(deps),
		Payroll:       app.NewPayroll(deps, renderer),
		Loans:         app.NewLoans(deps),
		Self:          app.NewSelfService(deps, leaves, announcements),
		Reports:       app.NewReports(deps),
		Dashboard:     app.NewDashboard(deps, leaves),
	}
	return svc, identity, func() { _ = conn.Close() }, nil
}

func register(ctx context.Context, d pkgapp.Deps, s *grpc.Server) error {
	svc, _, closeFn, err := Wire(d)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		closeFn()
	}()
	grpcapi.Register(s, svc)
	return nil
}

// worker unlinks deleted users, turns hired candidates into probation
// employees and sends the daily HR reminders (contract expiry 30/14/7 days,
// probation end, birthdays) on the ERP daily tick.
func worker(ctx context.Context, d pkgapp.Deps) error {
	if d.Bus == nil {
		<-ctx.Done()
		return nil
	}
	svc, _, closeFn, err := Wire(d)
	if err != nil {
		return err
	}
	defer closeFn()
	maintenance := app.NewMaintenance(svc.Deps)
	subjects := []string{events.UserDeleted, events.ERPTickDaily, events.RecruitmentCandidateHired}
	return d.Bus.Subscribe(ctx, "hr-events", subjects, func(ctx context.Context, msg events.Message) error {
		switch msg.Subject {
		case events.UserDeleted:
			var p events.UserDeletedPayload
			if err := msg.Decode(&p); err != nil {
				return err
			}
			return maintenance.ForgetUser(ctx, p.UserID)
		case events.ERPTickDaily:
			var p events.TickPayload
			if err := msg.Decode(&p); err != nil {
				return err
			}
			rep, err := maintenance.Daily(ctx, p.RunAt)
			if err != nil {
				return err
			}
			d.Logger.Info("hr daily reminders", "contracts", rep.ContractReminders, "probations", rep.ProbationReminders, "birthdays", rep.Birthdays)
		case events.RecruitmentCandidateHired:
			var p app.CandidateHired
			if err := msg.Decode(&p); err != nil {
				d.Logger.Warn("malformed candidate hired event", "err", err)
				return nil
			}
			return svc.Lifecycle.OnCandidateHired(ctx, p)
		}
		return nil
	})
}
