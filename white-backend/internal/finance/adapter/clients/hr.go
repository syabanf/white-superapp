package clients

import (
	"context"
	"time"

	hrv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/hr/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HR reads hr's EmployeeService. GetEmployeeByUser is internal; GetEmployee
// runs with the caller's forwarded token.
type HR struct {
	client    hrv1.EmployeeServiceClient
	employees *ttlCache[*app.Employee]
}

func NewHR(conn grpc.ClientConnInterface) *HR {
	return &HR{client: hrv1.NewEmployeeServiceClient(conn), employees: newCache[*app.Employee](60 * time.Second)}
}

func toEmployee(e *hrv1.Employee) *app.Employee {
	return &app.Employee{ID: e.Id, UserID: e.GetUserId(), Name: e.FullName, DepartmentName: e.DepartmentName, ManagerID: e.GetManagerId()}
}

func (c *HR) EmployeeByUser(ctx context.Context, userID string) (*app.Employee, error) {
	key := "user:" + userID
	if e, ok := c.employees.get(key); ok {
		return e, nil
	}
	res, err := c.client.GetEmployeeByUser(ctx, &hrv1.GetEmployeeByUserRequest{UserId: userID})
	if status.Code(err) == codes.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	e := toEmployee(res)
	c.employees.put(key, e)
	return e, nil
}

func (c *HR) Employee(ctx context.Context, id string) (*app.Employee, error) {
	if e, ok := c.employees.get(id); ok {
		return e, nil
	}
	res, err := c.client.GetEmployee(ctx, &hrv1.GetEmployeeRequest{EmployeeId: id})
	if status.Code(err) == codes.NotFound {
		return nil, nil
	}
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	e := toEmployee(res)
	c.employees.put(id, e)
	return e, nil
}
