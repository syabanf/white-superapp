// Package grpcapi exposes the performance use cases over gRPC. Handlers
// convert proto to Go and back; every rule and permission check lives in app.
package grpcapi

import (
	"encoding/json"
	"time"

	pb "github.com/syabanf/white-superapp/white-backend/gen/go/white/performance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Services are the use cases the handlers call.
type Services struct {
	KPI          *app.KPI
	Reviews      *app.Reviews
	Feedback     *app.Feedback
	Tasks        *app.Tasks
	Logbook      *app.Logbook
	Recaps       *app.Recaps
	Improvements *app.Improvements
	Dashboard    *app.Dashboard
}

func Register(s *grpc.Server, svc Services) {
	pb.RegisterKpiServiceServer(s, &kpiServer{svc: svc})
	pb.RegisterReviewServiceServer(s, &reviewServer{svc: svc})
	pb.RegisterFeedbackServiceServer(s, &feedbackServer{svc: svc})
	pb.RegisterTaskServiceServer(s, &taskServer{svc: svc})
	pb.RegisterLogbookServiceServer(s, &logbookServer{svc: svc})
	pb.RegisterRecapServiceServer(s, &recapServer{svc: svc})
	pb.RegisterImprovementServiceServer(s, &improvementServer{svc: svc})
}

// ─── conversion helpers ──────────────────────────────────────

func ts(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func tsPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func i32Ptr(n *int) *int32 {
	if n == nil {
		return nil
	}
	v := int32(*n)
	return &v
}

func intPtr(n *int32) *int {
	if n == nil {
		return nil
	}
	v := int(*n)
	return &v
}

func jsonString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func msg(text string) *pb.MessageResponse { return &pb.MessageResponse{Message: text} }

func period(year, month int32) domain.Period {
	return domain.Period{Year: int(year), Month: int(month)}
}

func departmentPB(d domain.Department) *pb.Department {
	return &pb.Department{Id: d.ID, Code: d.Code, Name: d.Name, HeadEmployeeId: d.HeadEmployeeID}
}

func personPB(e domain.Employee) *pb.Person {
	return &pb.Person{Id: e.ID, Name: e.Name, DepartmentId: e.DepartmentID, DepartmentName: e.DepartmentName, PositionName: e.PositionName}
}

func mePB(a *app.Actor) *pb.Me {
	m := &pb.Me{UserId: a.UserID, Name: a.Name, HrRead: a.HRRead, HrWrite: a.HRWrite, HrApprove: a.HRApprove,
		HeadOf: []string{}, DirectReports: []string{}}
	if a.Employee != nil {
		m.EmployeeId, m.DepartmentId = &a.Employee.ID, a.Employee.DepartmentID
	}
	for id := range a.HeadOf {
		m.HeadOf = append(m.HeadOf, id)
	}
	for id := range a.Reports {
		m.DirectReports = append(m.DirectReports, id)
	}
	return m
}
