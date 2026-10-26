package grpcapi

import (
	"context"

	pb "github.com/syabanf/white-superapp/white-backend/gen/go/white/recruitment/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/app"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ─── JobOpeningService ───────────────────────────────────────

type jobServer struct {
	pb.UnimplementedJobOpeningServiceServer
	svc Services
}

func jobInput(in *pb.JobOpeningInput) domain.JobOpeningInput {
	return domain.JobOpeningInput{Title: in.GetTitle(), Slug: in.GetSlug(), DepartmentID: in.DepartmentId, Department: in.GetDepartment(),
		Location: in.GetLocation(), EmploymentType: in.GetEmploymentType(), WorkMode: in.GetWorkMode(), Headcount: int(in.GetHeadcount()),
		Description: in.GetDescription(), Requirements: in.GetRequirements(), Benefits: in.GetBenefits(), SalaryMin: in.GetSalaryMin(),
		SalaryMax: in.GetSalaryMax(), HiringManagerUserID: in.HiringManagerUserId, Status: in.GetStatus(), ClosingDate: in.ClosingDate}
}

func (s *jobServer) ListJobOpenings(ctx context.Context, req *pb.ListJobOpeningsRequest) (*pb.ListJobOpeningsResponse, error) {
	list, err := s.svc.Jobs.List(ctx, req.GetStatus(), req.GetSearch())
	if err != nil {
		return nil, err
	}
	out := &pb.ListJobOpeningsResponse{JobOpenings: make([]*pb.JobOpening, len(list))}
	for i := range list {
		out.JobOpenings[i] = jobPB(&list[i])
	}
	return out, nil
}

func (s *jobServer) GetJobOpening(ctx context.Context, req *pb.JobOpeningRequest) (*pb.JobOpening, error) {
	o, err := s.svc.Jobs.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return jobPB(o), nil
}

func (s *jobServer) CreateJobOpening(ctx context.Context, req *pb.JobOpeningInput) (*pb.JobOpening, error) {
	o, err := s.svc.Jobs.Create(ctx, jobInput(req))
	if err != nil {
		return nil, err
	}
	return jobPB(o), nil
}

func (s *jobServer) UpdateJobOpening(ctx context.Context, req *pb.UpdateJobOpeningRequest) (*pb.JobOpening, error) {
	o, err := s.svc.Jobs.Update(ctx, req.GetId(), jobInput(req.GetInput()))
	if err != nil {
		return nil, err
	}
	return jobPB(o), nil
}

func (s *jobServer) DeleteJobOpening(ctx context.Context, req *pb.JobOpeningRequest) (*emptypb.Empty, error) {
	if err := s.svc.Jobs.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// ─── CandidateService ────────────────────────────────────────

type candidateServer struct {
	pb.UnimplementedCandidateServiceServer
	svc Services
}

func (s *candidateServer) ListCandidates(ctx context.Context, req *pb.ListCandidatesRequest) (*pb.ListCandidatesResponse, error) {
	p, err := s.svc.Candidates.List(ctx, app.ListQuery{Status: req.GetStatus(), JobOpeningID: req.GetJobOpeningId(), Search: req.GetSearch(),
		DateFrom: req.GetDateFrom(), DateTo: req.GetDateTo(), Sort: req.GetSort(), Page: int(req.GetPage()), Limit: int(req.GetLimit()),
		All: req.GetAll()})
	if err != nil {
		return nil, err
	}
	return pagePB(p), nil
}

func (s *candidateServer) GetCandidate(ctx context.Context, req *pb.CandidateRequest) (*pb.Candidate, error) {
	c, err := s.svc.Candidates.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return candidatePB(c), nil
}

func (s *candidateServer) CreateCandidate(ctx context.Context, req *pb.CandidateInput) (*pb.Candidate, error) {
	c, err := s.svc.Candidates.Create(ctx, domain.CandidateFields{FullName: req.GetFullName(), Email: req.GetEmail(), Phone: req.GetPhone(),
		Domicile: req.GetDomicile(), Source: req.GetSource(), JobOpeningID: req.JobOpeningId, Status: req.GetStatus(), Notes: req.GetNotes(),
		LastExperience: req.GetLastExperience(), LastEducation: req.GetLastEducation(), Availability: req.GetAvailability(),
		ExpectedSalary: req.ExpectedSalary})
	if err != nil {
		return nil, err
	}
	return candidatePB(c), nil
}

func (s *candidateServer) UpdateCandidate(ctx context.Context, req *pb.UpdateCandidateRequest) (*pb.Candidate, error) {
	in := req.GetInput()
	c, err := s.svc.Candidates.Update(ctx, req.GetId(), domain.CandidatePatch{FullName: in.FullName, Email: in.Email, Phone: in.Phone,
		Domicile: in.Domicile, Source: in.Source, JobOpeningID: in.JobOpeningId, Notes: in.Notes, LastExperience: in.LastExperience,
		LastEducation: in.LastEducation, Availability: in.Availability, ExpectedSalary: in.ExpectedSalary,
		ClearExpectedSalary: in.GetClearExpectedSalary()})
	if err != nil {
		return nil, err
	}
	return candidatePB(c), nil
}

func (s *candidateServer) DeleteCandidate(ctx context.Context, req *pb.CandidateRequest) (*pb.MessageResponse, error) {
	if err := s.svc.Candidates.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return msg("Kandidat dihapus"), nil
}

func (s *candidateServer) MoveCandidateStage(ctx context.Context, req *pb.MoveStageRequest) (*pb.MoveStageResponse, error) {
	status, prev, message, err := s.svc.Candidates.MoveStage(ctx, req.GetId(), req.GetStatus())
	if err != nil {
		return nil, err
	}
	return &pb.MoveStageResponse{Status: status, Previous: prev, Message: message}, nil
}

func (s *candidateServer) GetPipeline(ctx context.Context, req *pb.GetPipelineRequest) (*pb.Pipeline, error) {
	cols, err := s.svc.Candidates.Pipeline(ctx, req.GetJobOpeningId(), req.GetSearch())
	if err != nil {
		return nil, err
	}
	out := &pb.Pipeline{}
	for _, c := range cols {
		out.Columns = append(out.Columns, &pb.PipelineColumn{Status: c.Status, Label: c.Label, Count: int32(c.Count),
			Candidates: candidatesPB(c.Candidates)})
	}
	return out, nil
}

func (s *candidateServer) ListTalentPool(ctx context.Context, req *pb.ListTalentPoolRequest) (*pb.ListCandidatesResponse, error) {
	p, err := s.svc.Candidates.TalentPool(ctx, req.GetSearch(), req.GetIncludeArchived(), int(req.GetPage()), int(req.GetLimit()))
	if err != nil {
		return nil, err
	}
	return pagePB(p), nil
}

func countsPB(list []domain.Count) []*pb.CountByKey {
	out := make([]*pb.CountByKey, len(list))
	for i, c := range list {
		out[i] = &pb.CountByKey{Key: c.Key, Label: c.Label, Count: int32(c.Count)}
	}
	return out
}

func (s *candidateServer) GetRecruitmentSummary(ctx context.Context, _ *emptypb.Empty) (*pb.RecruitmentSummary, error) {
	sum, err := s.svc.Candidates.Summary(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.RecruitmentSummary{OpenJobs: int32(sum.OpenJobs), ActiveCandidates: int32(sum.ActiveCandidates),
		AppliedLast_30Days: int32(sum.AppliedLast30), HiredLast_90Days: int32(sum.HiredLast90), InterviewsNext_7Days: int32(sum.InterviewsNext7),
		OpenOffers: int32(sum.OpenOffers), ByStatus: countsPB(sum.ByStatus), BySource: countsPB(sum.BySource), AvgDaysToHire: sum.AvgDaysToHire}, nil
}

func notePB(n *domain.Note) *pb.CandidateNote {
	return &pb.CandidateNote{Id: n.ID, CandidateId: n.CandidateID, Content: n.Content, CreatedBy: n.CreatedBy,
		CreatedByName: n.CreatedByName, CreatedAt: ts(n.CreatedAt)}
}

func (s *candidateServer) ListNotes(ctx context.Context, req *pb.CandidateRequest) (*pb.ListNotesResponse, error) {
	list, err := s.svc.Candidates.Notes(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	out := &pb.ListNotesResponse{Notes: make([]*pb.CandidateNote, len(list))}
	for i := range list {
		out.Notes[i] = notePB(&list[i])
	}
	return out, nil
}

func (s *candidateServer) AddNote(ctx context.Context, req *pb.AddNoteRequest) (*pb.CandidateNote, error) {
	n, err := s.svc.Candidates.AddNote(ctx, req.GetId(), req.GetContent())
	if err != nil {
		return nil, err
	}
	return notePB(n), nil
}

func (s *candidateServer) ListActivities(ctx context.Context, req *pb.CandidateRequest) (*pb.ListActivitiesResponse, error) {
	list, err := s.svc.Candidates.Activities(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	out := &pb.ListActivitiesResponse{Activities: make([]*pb.CandidateActivity, len(list))}
	for i, a := range list {
		out.Activities[i] = &pb.CandidateActivity{Id: a.ID, CandidateId: a.CandidateID, ActivityType: a.Type, Description: a.Description,
			CreatedBy: a.CreatedBy, CreatedByName: a.CreatedByName, CreatedAt: ts(a.CreatedAt)}
	}
	return out, nil
}

func screeningPB(x *domain.Screening) *pb.Screening {
	if x == nil {
		return nil
	}
	return &pb.Screening{CandidateId: x.CandidateID, Contacted: x.Contacted, Interested: x.Interested, AvailabilityNote: x.AvailabilityNote,
		ConfirmedSalary: x.ConfirmedSalary, WillingShift: x.WillingShift, WillingPlacement: x.WillingPlacement, Notes: x.Notes,
		Recommendation: x.Recommendation, UpdatedByName: x.UpdatedByName, UpdatedAt: ts(x.UpdatedAt)}
}

func (s *candidateServer) GetScreening(ctx context.Context, req *pb.CandidateRequest) (*pb.ScreeningResponse, error) {
	x, err := s.svc.Candidates.Screening(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &pb.ScreeningResponse{Screening: screeningPB(x)}, nil
}

func (s *candidateServer) SaveScreening(ctx context.Context, req *pb.SaveScreeningRequest) (*pb.Screening, error) {
	x, err := s.svc.Candidates.SaveScreening(ctx, domain.Screening{CandidateID: req.GetId(), Contacted: req.GetContacted(),
		Interested: req.Interested, AvailabilityNote: req.GetAvailabilityNote(), ConfirmedSalary: req.ConfirmedSalary,
		WillingShift: req.WillingShift, WillingPlacement: req.WillingPlacement, Notes: req.GetNotes(), Recommendation: req.GetRecommendation()})
	if err != nil {
		return nil, err
	}
	return screeningPB(x), nil
}

func (s *candidateServer) UploadCv(ctx context.Context, req *pb.UploadCvRequest) (*pb.Candidate, error) {
	name, data := upload(req.GetFile())
	c, err := s.svc.Candidates.UploadCV(ctx, req.GetId(), name, data)
	if err != nil {
		return nil, err
	}
	return candidatePB(c), nil
}

func (s *candidateServer) DeleteCv(ctx context.Context, req *pb.CandidateRequest) (*pb.Candidate, error) {
	c, err := s.svc.Candidates.DeleteCV(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return candidatePB(c), nil
}

func (s *candidateServer) DownloadCv(ctx context.Context, req *pb.CandidateRequest) (*pb.FileContent, error) {
	f, err := s.svc.Candidates.DownloadCV(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return fileOf(f), nil
}

func analysisPB(a *domain.CandidateAnalysis) *pb.AiAnalysis {
	if a == nil {
		return nil
	}
	return &pb.AiAnalysis{CandidateId: a.CandidateID, Summary: a.Summary, MatchScore: int32(a.MatchScore), MatchReason: a.MatchReason,
		Strengths: nonNil(a.Strengths), Gaps: nonNil(a.Gaps), JobContext: a.JobContext, Model: a.Model, UpdatedAt: ts(a.UpdatedAt)}
}

func (s *candidateServer) GetAiAnalysis(ctx context.Context, req *pb.CandidateRequest) (*pb.AiAnalysisResponse, error) {
	a, err := s.svc.Candidates.Analysis(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &pb.AiAnalysisResponse{Analysis: analysisPB(a)}, nil
}

func (s *candidateServer) AnalyzeCandidate(ctx context.Context, req *pb.CandidateRequest) (*pb.AiAnalysis, error) {
	a, err := s.svc.Candidates.Analyze(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return analysisPB(a), nil
}

func (s *candidateServer) DownloadPipelineReport(ctx context.Context, req *pb.CandidateRequest) (*pb.FileContent, error) {
	f, err := s.svc.Candidates.PipelineReport(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return fileOf(f), nil
}

// ─── OfferService ────────────────────────────────────────────

type offerServer struct {
	pb.UnimplementedOfferServiceServer
	svc Services
}

func (s *offerServer) GetCandidateOffers(ctx context.Context, req *pb.CandidatePanelRequest) (*pb.OfferPanel, error) {
	ref, offers, err := s.svc.Offers.Panel(ctx, req.GetCandidateId())
	if err != nil {
		return nil, err
	}
	out := &pb.OfferPanel{SalaryReference: &pb.SalaryReference{ExpectedSalary: ref.ExpectedSalary, InterviewExpectation: ref.InterviewExpectation,
		PositionTitle: ref.PositionTitle, SalaryMin: moneyStr(ref.SalaryMin), SalaryMax: moneyStr(ref.SalaryMax),
		ConfirmedSalary: ref.ConfirmedSalary}, Offers: make([]*pb.Offer, len(offers))}
	for i := range offers {
		out.Offers[i] = offerPB(&offers[i])
	}
	return out, nil
}

func (s *offerServer) CreateOffer(ctx context.Context, req *pb.CreateOfferRequest) (*pb.Offer, error) {
	o, err := s.svc.Offers.Create(ctx, req.GetCandidateId(), domain.OfferInput{BaseSalary: req.GetBaseSalary(), Benefits: req.GetBenefits(),
		StartDate: req.StartDate, Notes: req.GetNotes(), ExpiresDays: int(req.GetExpiresDays()), PositionTitle: req.GetPositionTitle()})
	if err != nil {
		return nil, err
	}
	return offerPB(o), nil
}

func (s *offerServer) RecordOfferResponse(ctx context.Context, req *pb.RecordOfferResponseRequest) (*pb.Offer, error) {
	o, err := s.svc.Offers.RecordResponse(ctx, req.GetId(), req.GetStatus(), req.GetNote())
	if err != nil {
		return nil, err
	}
	return offerPB(o), nil
}

func (s *offerServer) DownloadOfferLetter(ctx context.Context, req *pb.OfferRequest) (*pb.FileContent, error) {
	f, err := s.svc.Offers.DownloadLetter(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return fileOf(f), nil
}
