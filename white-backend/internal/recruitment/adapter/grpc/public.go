package grpcapi

import (
	"context"

	pb "github.com/syabanf/white-superapp/white-backend/gen/go/white/recruitment/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
)

// publicServer serves the candidate portals: no principal, the slug or the
// link token authorizes.
type publicServer struct {
	pb.UnimplementedPublicRecruitmentServiceServer
	svc Services
}

func publicJobPB(o *domain.JobOpening) *pb.PublicJob {
	return &pb.PublicJob{Slug: o.Slug, Title: o.Title, Department: o.Department, Location: o.Location, EmploymentType: o.EmploymentType,
		WorkMode: o.WorkMode, Headcount: int32(o.Headcount), Description: o.Description, Requirements: o.Requirements, Benefits: o.Benefits,
		PublishedAt: tsPtr(o.PublishedAt), ClosingDate: o.ClosingDate}
}

func (s *publicServer) ListPublicJobs(ctx context.Context, req *pb.ListPublicJobsRequest) (*pb.ListPublicJobsResponse, error) {
	list, err := s.svc.Portal.PublicJobs(ctx, req.GetSearch(), req.GetDepartment())
	if err != nil {
		return nil, err
	}
	out := &pb.ListPublicJobsResponse{Jobs: make([]*pb.PublicJob, len(list))}
	for i := range list {
		out.Jobs[i] = publicJobPB(&list[i])
	}
	return out, nil
}

func (s *publicServer) GetPublicJob(ctx context.Context, req *pb.PublicJobRequest) (*pb.PublicJob, error) {
	o, err := s.svc.Portal.PublicJob(ctx, req.GetSlug())
	if err != nil {
		return nil, err
	}
	return publicJobPB(o), nil
}

func (s *publicServer) Apply(ctx context.Context, req *pb.ApplyRequest) (*pb.ApplyResponse, error) {
	name, data := upload(req.GetCv())
	id, err := s.svc.Portal.Apply(ctx, req.GetSlug(), domain.Application{FullName: req.GetFullName(), Email: req.GetEmail(),
		Phone: req.GetPhone(), Domicile: req.GetDomicile(), Source: req.GetSource(), LastExperience: req.GetLastExperience(),
		LastEducation: req.GetLastEducation(), Availability: req.GetAvailability(), Notes: req.GetNotes(), ExpectedSalary: req.ExpectedSalary},
		name, data, grpcx.RemoteIP(ctx))
	if err != nil {
		return nil, err
	}
	return &pb.ApplyResponse{CandidateId: id, Message: domain.MsgApplied}, nil
}

// ─── psikotes ────────────────────────────────────────────────

func portalTestPB(t *domain.SessionTest) *pb.PortalTest {
	return &pb.PortalTest{Id: t.ID, Status: t.Status, SortOrder: int32(t.SortOrder), StartedAt: tsPtr(t.StartedAt),
		CompletedAt: tsPtr(t.CompletedAt), HasAttachment: t.AttachmentKey != nil,
		Instrument: &pb.PortalInstrument{Code: t.InstrumentCode, Name: t.InstrumentName, Kind: t.InstrumentKind,
			DurationSeconds: int32(t.DurationSeconds), Instructions: t.Instructions}}
}

func psikotesPortalPB(x *domain.PsikotesSession) *pb.PsikotesPortal {
	out := &pb.PsikotesPortal{Status: x.Status, CandidateName: x.CandidateName, PositionTitle: x.PositionTitle, WebcamConsent: x.WebcamConsent,
		ExpiresAt: tsPtr(x.ExpiresAt), StartedAt: tsPtr(x.StartedAt), CompletedAt: tsPtr(x.CompletedAt), Tests: []*pb.PortalTest{}}
	for i := range x.Tests {
		out.Tests = append(out.Tests, portalTestPB(&x.Tests[i]))
	}
	return out
}

func (s *publicServer) GetPsikotesPortal(ctx context.Context, req *pb.PortalTokenRequest) (*pb.PsikotesPortal, error) {
	x, err := s.svc.Portal.PsikotesPortal(ctx, req.GetToken())
	if err != nil {
		return nil, err
	}
	return psikotesPortalPB(x), nil
}

func (s *publicServer) StartPsikotes(ctx context.Context, req *pb.StartPortalRequest) (*pb.PsikotesPortal, error) {
	x, err := s.svc.Portal.StartPsikotes(ctx, req.GetToken(), req.GetWebcamConsent())
	if err != nil {
		return nil, err
	}
	return psikotesPortalPB(x), nil
}

func (s *publicServer) StartPsikotesTest(ctx context.Context, req *pb.PortalTestRequest) (*pb.PortalTestRun, error) {
	run, err := s.svc.Portal.StartTest(ctx, req.GetToken(), req.GetTestId())
	if err != nil {
		return nil, err
	}
	saved := run.Saved
	if saved == nil {
		saved = map[string]string{}
	}
	return &pb.PortalTestRun{Test: portalTestPB(&run.Test), Questions: questionsPB(run.Questions), SavedAnswers: saved,
		EndsAt: tsPtr(run.Test.Deadline())}, nil
}

func (s *publicServer) SavePsikotesAnswers(ctx context.Context, req *pb.SaveAnswersRequest) (*pb.SaveAnswersResponse, error) {
	n, err := s.svc.Portal.SaveAnswers(ctx, req.GetToken(), req.GetTestId(), req.GetAnswers())
	if err != nil {
		return nil, err
	}
	return &pb.SaveAnswersResponse{Saved: int32(n)}, nil
}

func (s *publicServer) FinishPsikotesTest(ctx context.Context, req *pb.SaveAnswersRequest) (*pb.PortalActionResponse, error) {
	status, message, err := s.svc.Portal.FinishTest(ctx, req.GetToken(), req.GetTestId(), req.GetAnswers())
	if err != nil {
		return nil, err
	}
	return &pb.PortalActionResponse{Status: status, Message: message}, nil
}

func (s *publicServer) UploadPsikotesDrawing(ctx context.Context, req *pb.UploadDrawingRequest) (*pb.PortalActionResponse, error) {
	name, data := upload(req.GetFile())
	if err := s.svc.Portal.UploadDrawing(ctx, req.GetToken(), req.GetTestId(), name, data); err != nil {
		return nil, err
	}
	return &pb.PortalActionResponse{Status: "uploaded", Message: "Gambar terunggah"}, nil
}

func (s *publicServer) FinishPsikotes(ctx context.Context, req *pb.PortalTokenRequest) (*pb.PortalActionResponse, error) {
	status, message, err := s.svc.Portal.FinishPsikotes(ctx, req.GetToken())
	if err != nil {
		return nil, err
	}
	return &pb.PortalActionResponse{Status: status, Message: message}, nil
}

func (s *publicServer) RecordPsikotesProctorEvent(ctx context.Context, req *pb.ProctorEventRequest) (*pb.ProctorEvent, error) {
	e, err := s.svc.Portal.RecordProctorEvent(ctx, "psikotes", req.GetToken(), req.GetEventType(), req.GetMeta())
	if err != nil {
		return nil, err
	}
	return proctorEventPB(e), nil
}

// ─── AI interview ────────────────────────────────────────────

func portalTurnPB(t *domain.Turn) *pb.PortalTurn {
	if t == nil {
		return nil
	}
	return &pb.PortalTurn{Id: t.ID, TurnNo: int32(t.TurnNo), Question: t.Question, AnswerTranscript: deref(t.Answer), AskedAt: ts(t.AskedAt),
		AnsweredAt: tsPtr(t.AnsweredAt)}
}

func (s *publicServer) GetInterviewPortal(ctx context.Context, req *pb.PortalTokenRequest) (*pb.InterviewPortal, error) {
	x, current, err := s.svc.Portal.InterviewPortal(ctx, req.GetToken())
	if err != nil {
		return nil, err
	}
	out := &pb.InterviewPortal{Status: x.Status, CandidateName: x.CandidateName, PositionTitle: x.PositionTitle, WebcamConsent: x.WebcamConsent,
		MaxQuestions: int32(domain.ClampMaxQuestions(x.MaxQuestions)), ExpiresAt: tsPtr(x.ExpiresAt), StartedAt: tsPtr(x.StartedAt),
		CompletedAt: tsPtr(x.CompletedAt), CurrentTurn: portalTurnPB(current), Turns: []*pb.PortalTurn{}}
	for i := range x.Turns {
		out.Turns = append(out.Turns, portalTurnPB(&x.Turns[i]))
	}
	return out, nil
}

func (s *publicServer) StartAiInterview(ctx context.Context, req *pb.StartPortalRequest) (*pb.InterviewTurnResponse, error) {
	t, err := s.svc.Portal.StartInterview(ctx, req.GetToken(), req.GetWebcamConsent())
	if err != nil {
		return nil, err
	}
	return &pb.InterviewTurnResponse{Turn: portalTurnPB(t)}, nil
}

func (s *publicServer) AnswerAiInterview(ctx context.Context, req *pb.AnswerInterviewRequest) (*pb.InterviewTurnResponse, error) {
	t, done, err := s.svc.Portal.AnswerInterview(ctx, req.GetToken(), req.GetTurnId(), req.GetAnswerText())
	if err != nil {
		return nil, err
	}
	return &pb.InterviewTurnResponse{Done: done, Turn: portalTurnPB(t)}, nil
}

func (s *publicServer) RecordInterviewProctorEvent(ctx context.Context, req *pb.ProctorEventRequest) (*pb.ProctorEvent, error) {
	e, err := s.svc.Portal.RecordProctorEvent(ctx, "interview", req.GetToken(), req.GetEventType(), req.GetMeta())
	if err != nil {
		return nil, err
	}
	return proctorEventPB(e), nil
}

// ─── offer ───────────────────────────────────────────────────

func (s *publicServer) offerPortalPB(o *domain.Offer) *pb.OfferPortal {
	return &pb.OfferPortal{Status: o.Status, Version: int32(o.Version), CandidateName: o.CandidateName, CompanyName: s.svc.Portal.Company(),
		PositionTitle: o.PositionTitle, BaseSalary: o.BaseSalary.String(), Benefits: nonNil(o.Benefits), StartDate: o.StartDate,
		ResponseNote: o.ResponseNote, RespondedAt: tsPtr(o.RespondedAt), SentAt: tsPtr(o.SentAt), ExpiresAt: tsPtr(o.ExpiresAt)}
}

func (s *publicServer) GetOfferPortal(ctx context.Context, req *pb.PortalTokenRequest) (*pb.OfferPortal, error) {
	o, err := s.svc.Portal.OfferPortal(ctx, req.GetToken())
	if err != nil {
		return nil, err
	}
	return s.offerPortalPB(o), nil
}

func (s *publicServer) RespondOffer(ctx context.Context, req *pb.RespondOfferRequest) (*pb.OfferPortal, error) {
	o, err := s.svc.Portal.RespondOffer(ctx, req.GetToken(), req.GetAction(), req.GetNote(), grpcx.RemoteIP(ctx))
	if err != nil {
		return nil, err
	}
	return s.offerPortalPB(o), nil
}

func (s *publicServer) DownloadPublicOfferLetter(ctx context.Context, req *pb.PortalTokenRequest) (*pb.FileContent, error) {
	f, err := s.svc.Portal.OfferLetter(ctx, req.GetToken())
	if err != nil {
		return nil, err
	}
	return fileOf(f), nil
}
