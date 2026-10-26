// Package clients wraps the gRPC clients of hr (employee directory and
// attendance), identity (notifications) and integrations (OpenRouter
// credentials) behind the app ports. The directory is cached 60 s and the
// credentials 60 s (busted by white.integration.updated).
package clients

import (
	"context"
	"sync"
	"time"

	hrv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/hr/v1"
	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	integrationsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/integrations/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const cacheTTL = 60 * time.Second

// cached holds one value for cacheTTL.
type cached[T any] struct {
	mu  sync.Mutex
	val T
	at  time.Time
}

func (c *cached[T]) get(now time.Time, load func() (T, error)) (T, error) {
	c.mu.Lock()
	if !c.at.IsZero() && now.Sub(c.at) < cacheTTL {
		v := c.val
		c.mu.Unlock()
		return v, nil
	}
	c.mu.Unlock()
	v, err := load()
	if err != nil {
		return v, err
	}
	c.mu.Lock()
	c.val, c.at = v, now
	c.mu.Unlock()
	return v, nil
}

func (c *cached[T]) reset() {
	c.mu.Lock()
	c.at = time.Time{}
	c.mu.Unlock()
}

// ─── hr ──────────────────────────────────────────────────────

// internal drops the caller's forwarded token: the directory is cached and
// shared by every caller, so hr must answer it as a service-to-service
// call (no principal, no edge marker), never with one user's rights.
func internal(ctx context.Context) context.Context {
	return metadata.NewIncomingContext(ctx, metadata.MD{})
}

// HR implements app.Directory over hr.EmployeeService.
type HR struct {
	client hrv1.EmployeeServiceClient
	clk    clock.Clock

	active, all cached[[]domain.Employee]
	depts       cached[[]domain.Department]
}

func NewHR(conn grpc.ClientConnInterface, clk clock.Clock) *HR {
	return &HR{client: hrv1.NewEmployeeServiceClient(conn), clk: clk}
}

var _ app.Directory = (*HR)(nil)

func employeeFrom(e *hrv1.Employee) domain.Employee {
	active := e.GetStatus() != hrv1.EmploymentStatus_RESIGNED && e.GetStatus() != hrv1.EmploymentStatus_TERMINATED
	return domain.Employee{ID: e.GetId(), UserID: e.UserId, EmployeeNo: e.GetEmployeeNo(), Name: e.GetFullName(), Email: e.GetEmail(),
		DepartmentID: e.DepartmentId, DepartmentName: e.GetDepartmentName(), PositionName: e.GetPositionName(), ManagerID: e.ManagerId,
		Active: active}
}

func (h *HR) EmployeeByUser(ctx context.Context, userID string) (*domain.Employee, error) {
	e, err := h.client.GetEmployeeByUser(internal(ctx), &hrv1.GetEmployeeByUserRequest{UserId: userID})
	if err != nil {
		if ae := grpcx.FromStatus(err); ae.Code == apperr.NotFound {
			return nil, nil
		}
		return nil, grpcx.FromStatus(err)
	}
	out := employeeFrom(e)
	return &out, nil
}

func (h *HR) Employees(ctx context.Context, includeInactive bool) ([]domain.Employee, error) {
	c := &h.active
	if includeInactive {
		c = &h.all
	}
	return c.get(h.clk.Now(), func() ([]domain.Employee, error) {
		var out []domain.Employee
		token := ""
		for {
			res, err := h.client.ListEmployees(internal(ctx), &hrv1.ListEmployeesRequest{IncludeInactive: includeInactive, PageSize: 500, PageToken: token})
			if err != nil {
				return nil, grpcx.FromStatus(err)
			}
			for _, e := range res.GetEmployees() {
				out = append(out, employeeFrom(e))
			}
			if token = res.GetNextPageToken(); token == "" {
				return out, nil
			}
		}
	})
}

func (h *HR) Departments(ctx context.Context) ([]domain.Department, error) {
	return h.depts.get(h.clk.Now(), func() ([]domain.Department, error) {
		res, err := h.client.ListDepartments(internal(ctx), &hrv1.ListDepartmentsRequest{})
		if err != nil {
			return nil, grpcx.FromStatus(err)
		}
		out := make([]domain.Department, 0, len(res.GetDepartments()))
		for _, d := range res.GetDepartments() {
			out = append(out, domain.Department{ID: d.GetId(), Code: d.GetCode(), Name: d.GetName(), HeadEmployeeID: d.HeadEmployeeId, Active: d.GetIsActive()})
		}
		return out, nil
	})
}

func (h *HR) Attendance(ctx context.Context, employeeIDs []string, p domain.Period) (map[string]domain.Attendance, error) {
	res, err := h.client.GetAttendanceSummary(internal(ctx), &hrv1.GetAttendanceSummaryRequest{EmployeeIds: employeeIDs, Month: p.Key()})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	out := make(map[string]domain.Attendance, len(res.GetSummaries()))
	for _, s := range res.GetSummaries() {
		out[s.GetEmployeeId()] = domain.Attendance{EmployeeID: s.GetEmployeeId(), WorkingDays: int(s.GetWorkingDays()), PresentDays: int(s.GetPresentDays()),
			LateDays: int(s.GetLateDays()), AbsentDays: int(s.GetAbsentDays()), LeaveDays: int(s.GetLeaveDays()), LateMinutes: int(s.GetLateMinutes()),
			OvertimeHours: s.GetOvertimeHours()}
	}
	return out, nil
}

// Invalidate drops the directory cache.
func (h *HR) Invalidate() {
	h.active.reset()
	h.all.reset()
	h.depts.reset()
}

// ─── identity ────────────────────────────────────────────────

// Identity delivers notices through NotificationService.Notify.
type Identity struct {
	client identityv1.NotificationServiceClient
}

func NewIdentity(conn grpc.ClientConnInterface) *Identity {
	return &Identity{client: identityv1.NewNotificationServiceClient(conn)}
}

func (i *Identity) Notify(ctx context.Context, n app.Notice) error {
	_, err := i.client.Notify(ctx, &identityv1.NotifyRequest{UserIds: n.UserIDs, Type: n.Type, Title: n.Title, Body: n.Body, Href: n.Href})
	if err != nil {
		return grpcx.FromStatus(err)
	}
	return nil
}

// ─── integrations ────────────────────────────────────────────

// Integrations fetches the OPENROUTER credentials.
type Integrations struct {
	client integrationsv1.SettingsServiceClient
	clk    clock.Clock
	creds  cached[openrouter.Credentials]
}

func NewIntegrations(conn grpc.ClientConnInterface, clk clock.Clock) *Integrations {
	return &Integrations{client: integrationsv1.NewSettingsServiceClient(conn), clk: clk}
}

// Credentials implements openrouter.CredentialsFunc.
func (c *Integrations) Credentials(ctx context.Context) (openrouter.Credentials, error) {
	return c.creds.get(c.clk.Now(), func() (openrouter.Credentials, error) {
		res, err := c.client.GetProviderCredentials(ctx, &integrationsv1.GetProviderCredentialsRequest{Key: integrationsv1.IntegrationKey_OPENROUTER})
		if err != nil {
			return openrouter.Credentials{}, grpcx.FromStatus(err)
		}
		if !res.GetConfigured() {
			return openrouter.Credentials{}, nil
		}
		return openrouter.Credentials{APIKey: res.GetSecret(), Model: res.GetPublicId()}, nil
	})
}

// Invalidate drops the cache so the next call sees new keys.
func (c *Integrations) Invalidate() { c.creds.reset() }
