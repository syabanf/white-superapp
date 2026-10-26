// Package grpcapi exposes the recruitment use cases over gRPC. Handlers
// convert proto to Go and back; every rule and permission check lives in
// app. PublicRecruitmentService needs no principal.
package grpcapi

import (
	"time"

	pb "github.com/syabanf/white-superapp/white-backend/gen/go/white/recruitment/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/app"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Services are the use cases the handlers call.
type Services struct {
	Jobs       *app.JobOpenings
	Candidates *app.Candidates
	Psikotes   *app.Psikotes
	Interviews *app.Interviews
	Offers     *app.Offers
	Portal     *app.Portal
}

func Register(s *grpc.Server, svc Services) {
	pb.RegisterJobOpeningServiceServer(s, &jobServer{svc: svc})
	pb.RegisterCandidateServiceServer(s, &candidateServer{svc: svc})
	pb.RegisterPsikotesServiceServer(s, &psikotesServer{svc: svc})
	pb.RegisterInterviewServiceServer(s, &interviewServer{svc: svc})
	pb.RegisterOfferServiceServer(s, &offerServer{svc: svc})
	pb.RegisterPublicRecruitmentServiceServer(s, &publicServer{svc: svc})
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

func timeOf(t *timestamppb.Timestamp) time.Time {
	if t == nil {
		return time.Time{}
	}
	return t.AsTime()
}

func moneyStr(m *domain.Money) string {
	if m == nil {
		return ""
	}
	return m.String()
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func fileOf(f *app.File) *pb.FileContent {
	return &pb.FileContent{FileName: f.Name, ContentType: f.ContentType, Data: f.Data}
}

func upload(f *pb.FileUpload) (string, []byte) {
	if f == nil {
		return "", nil
	}
	return f.GetFileName(), f.GetData()
}

func msg(text string) *pb.MessageResponse { return &pb.MessageResponse{Message: text} }

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// ─── shared messages ─────────────────────────────────────────

func jobPB(o *domain.JobOpening) *pb.JobOpening {
	return &pb.JobOpening{Id: o.ID, Title: o.Title, Slug: o.Slug, DepartmentId: o.DepartmentID, Department: o.Department,
		Location: o.Location, EmploymentType: o.EmploymentType, WorkMode: o.WorkMode, Headcount: int32(o.Headcount),
		Description: o.Description, Requirements: o.Requirements, Benefits: o.Benefits, SalaryMin: moneyStr(o.SalaryMin),
		SalaryMax: moneyStr(o.SalaryMax), HiringManagerUserId: o.HiringManagerUserID, Status: o.Status, PublishedAt: tsPtr(o.PublishedAt),
		ClosingDate: o.ClosingDate, CandidateCount: int32(o.CandidateCount), HiredCount: int32(o.HiredCount),
		CreatedAt: ts(o.CreatedAt), UpdatedAt: ts(o.UpdatedAt)}
}

func candidatePB(c *domain.Candidate) *pb.Candidate {
	label, _ := domain.StatusLabel(c.Status)
	return &pb.Candidate{Id: c.ID, FullName: c.FullName, Email: c.Email, Phone: c.Phone, Domicile: c.Domicile, Source: c.Source,
		JobOpeningId: c.JobOpeningID, JobOpeningTitle: c.JobOpeningTitle, Status: c.Status, StatusLabel: label, Notes: c.Notes,
		LastExperience: c.LastExperience, LastEducation: c.LastEducation, Availability: c.Availability,
		ExpectedSalary: c.ExpectedSalary, HasCv: c.CVKey != nil, CvFileName: c.CVFileName, HiredAt: tsPtr(c.HiredAt),
		CreatedAt: ts(c.CreatedAt), UpdatedAt: ts(c.UpdatedAt)}
}

func candidatesPB(list []domain.Candidate) []*pb.Candidate {
	out := make([]*pb.Candidate, len(list))
	for i := range list {
		out[i] = candidatePB(&list[i])
	}
	return out
}

func pagePB(p *app.Page) *pb.ListCandidatesResponse {
	return &pb.ListCandidatesResponse{Candidates: candidatesPB(p.Candidates), Total: int32(p.Total), Page: int32(p.Page),
		Limit: int32(p.Limit), TotalPages: int32(p.TotalPages)}
}

func questionPB(q domain.Question) *pb.Question {
	out := &pb.Question{Id: q.ID, InstrumentId: q.InstrumentID, Body: q.Body, CorrectKey: q.CorrectKey, SortOrder: int32(q.SortOrder),
		IsActive: q.IsActive, Options: []*pb.McqOption{}}
	for _, o := range q.Options {
		out.Options = append(out.Options, &pb.McqOption{Key: o.Key, Text: o.Text})
	}
	if q.Pair != nil {
		out.StatementA = &pb.PapiStatement{Text: q.Pair.A.Text, Scale: q.Pair.A.Scale}
		out.StatementB = &pb.PapiStatement{Text: q.Pair.B.Text, Scale: q.Pair.B.Scale}
	}
	return out
}

func proctorEventPB(e *domain.ProctorEvent) *pb.ProctorEvent {
	meta := e.Meta
	if meta == nil {
		meta = map[string]string{}
	}
	return &pb.ProctorEvent{Id: e.ID, EventType: e.EventType, Meta: meta, CreatedAt: ts(e.CreatedAt)}
}

func offerPB(o *domain.Offer) *pb.Offer {
	return &pb.Offer{Id: o.ID, CandidateId: o.CandidateID, Version: int32(o.Version), Token: o.Token, Status: o.Status,
		PositionTitle: o.PositionTitle, BaseSalary: o.BaseSalary.String(), Benefits: nonNil(o.Benefits), StartDate: o.StartDate,
		Notes: o.Notes, ResponseNote: o.ResponseNote, RespondedAt: tsPtr(o.RespondedAt), ResponseSource: o.ResponseSource,
		SentAt: tsPtr(o.SentAt), ExpiresAt: tsPtr(o.ExpiresAt), CreatedByName: o.CreatedByName, CreatedAt: ts(o.CreatedAt)}
}
