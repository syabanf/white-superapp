// Package clients wraps the gRPC clients of hr (employee directory),
// identity (notifications) and integrations (OpenRouter credentials) behind
// the app ports. The directory and the credentials are cached 60 s; the
// credentials are also busted by white.integration.updated.
package clients

import (
	"context"
	"sync"
	"time"

	hrv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/hr/v1"
	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	integrationsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/integrations/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/app"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
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
// shared by every caller (and portal calls have none), so hr answers it as
// a service-to-service call, never with one user's rights.
func internal(ctx context.Context) context.Context {
	return metadata.NewIncomingContext(ctx, metadata.MD{})
}

// HR implements app.Directory over hr.EmployeeService.
type HR struct {
	client hrv1.EmployeeServiceClient
	clk    clock.Clock
	people cached[[]domain.Employee]
	depts  cached[[]domain.Department]
}

func NewHR(conn grpc.ClientConnInterface, clk clock.Clock) *HR {
	return &HR{client: hrv1.NewEmployeeServiceClient(conn), clk: clk}
}

var _ app.Directory = (*HR)(nil)

func employeeFrom(e *hrv1.Employee) domain.Employee {
	active := e.GetStatus() != hrv1.EmploymentStatus_RESIGNED && e.GetStatus() != hrv1.EmploymentStatus_TERMINATED
	return domain.Employee{ID: e.GetId(), UserID: e.UserId, Name: e.GetFullName(), DepartmentID: e.DepartmentId,
		DepartmentName: e.GetDepartmentName(), Active: active}
}

func (h *HR) Employee(ctx context.Context, id string) (*domain.Employee, error) {
	e, err := h.client.GetEmployee(internal(ctx), &hrv1.GetEmployeeRequest{EmployeeId: id})
	if err != nil {
		if ae := grpcx.FromStatus(err); ae.Code == apperr.NotFound {
			return nil, nil
		}
		return nil, grpcx.FromStatus(err)
	}
	out := employeeFrom(e)
	return &out, nil
}

func (h *HR) Employees(ctx context.Context) ([]domain.Employee, error) {
	return h.people.get(h.clk.Now(), func() ([]domain.Employee, error) {
		var out []domain.Employee
		token := ""
		for {
			res, err := h.client.ListEmployees(internal(ctx), &hrv1.ListEmployeesRequest{PageSize: 500, PageToken: token})
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
			out = append(out, domain.Department{ID: d.GetId(), Code: d.GetCode(), Name: d.GetName(), Active: d.GetIsActive()})
		}
		return out, nil
	})
}

// Invalidate drops the directory cache.
func (h *HR) Invalidate() {
	h.people.reset()
	h.depts.reset()
}

// ─── identity ────────────────────────────────────────────────

// Identity delivers notices through NotificationService.Notify and lists
// permission holders (AccessService.ListPermissionHolders, cached).
type Identity struct {
	client  identityv1.NotificationServiceClient
	access  identityv1.AccessServiceClient
	clk     clock.Clock
	holders cached[[]string]
}

func NewIdentity(conn grpc.ClientConnInterface, clk clock.Clock) *Identity {
	return &Identity{client: identityv1.NewNotificationServiceClient(conn), access: identityv1.NewAccessServiceClient(conn), clk: clk}
}

var _ app.Audience = (*Identity)(nil)

// PermissionHolders caches one permission: recruitment only asks for
// recruitment.write.
func (i *Identity) PermissionHolders(ctx context.Context, perm string) ([]string, error) {
	return i.holders.get(i.clk.Now(), func() ([]string, error) {
		res, err := i.access.ListPermissionHolders(internal(ctx), &identityv1.ListPermissionHoldersRequest{Permission: perm})
		if err != nil {
			return nil, grpcx.FromStatus(err)
		}
		ids := make([]string, 0, len(res.GetUsers()))
		for _, u := range res.GetUsers() {
			ids = append(ids, u.GetId())
		}
		return ids, nil
	})
}

func (i *Identity) Notify(ctx context.Context, n app.Notice) error {
	_, err := i.client.Notify(internal(ctx), &identityv1.NotifyRequest{UserIds: n.UserIDs, Type: n.Type, Title: n.Title, Body: n.Body, Href: n.Href})
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
		res, err := c.client.GetProviderCredentials(internal(ctx),
			&integrationsv1.GetProviderCredentialsRequest{Key: integrationsv1.IntegrationKey_OPENROUTER})
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
