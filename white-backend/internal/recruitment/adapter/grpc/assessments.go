package grpcapi

import (
	"context"

	pb "github.com/syabanf/white-superapp/white-backend/gen/go/white/recruitment/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/app"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ─── PsikotesService ─────────────────────────────────────────

type psikotesServer struct {
	pb.UnimplementedPsikotesServiceServer
	svc Services
}

func instrumentPB(i *domain.Instrument) *pb.Instrument {
	out := &pb.Instrument{Id: i.ID, Code: i.Code, Name: i.Name, Kind: i.Kind, DurationSeconds: int32(i.DurationSeconds), Shuffle: i.Shuffle,
		Instructions: i.Instructions, IsActive: i.IsActive, SortOrder: int32(i.SortOrder), ActiveQuestions: int32(i.ActiveQuestions)}
	if i.QuestionCount != nil {
		n := int32(*i.QuestionCount)
		out.QuestionCount = &n
	}
	return out
}

func intPtr(n *int32) *int {
	if n == nil {
		return nil
	}
	v := int(*n)
	return &v
}

func (s *psikotesServer) ListInstruments(ctx context.Context, _ *emptypb.Empty) (*pb.ListInstrumentsResponse, error) {
	list, err := s.svc.Psikotes.Instruments(ctx)
	if err != nil {
		return nil, err
	}
	out := &pb.ListInstrumentsResponse{Instruments: make([]*pb.Instrument, len(list))}
	for i := range list {
		out.Instruments[i] = instrumentPB(&list[i])
	}
	return out, nil
}

func (s *psikotesServer) UpdateInstrument(ctx context.Context, req *pb.UpdateInstrumentRequest) (*pb.Instrument, error) {
	i, err := s.svc.Psikotes.UpdateInstrument(ctx, req.GetId(), domain.InstrumentPatch{Name: req.Name, IsActive: req.IsActive,
		DurationSeconds: intPtr(req.DurationSeconds), QuestionCount: intPtr(req.QuestionCount), ClearQuestionCount: req.GetClearQuestionCount(),
		Shuffle: req.Shuffle, Instructions: req.Instructions})
	if err != nil {
		return nil, err
	}
	return instrumentPB(i), nil
}

func questionsPB(list []domain.Question) []*pb.Question {
	out := make([]*pb.Question, len(list))
	for i, q := range list {
		out[i] = questionPB(q)
	}
	return out
}

func (s *psikotesServer) ListQuestions(ctx context.Context, req *pb.ListQuestionsRequest) (*pb.ListQuestionsResponse, error) {
	list, err := s.svc.Psikotes.Questions(ctx, req.GetInstrumentId())
	if err != nil {
		return nil, err
	}
	return &pb.ListQuestionsResponse{Questions: questionsPB(list)}, nil
}

func questionInput(in *pb.QuestionInput) domain.QuestionInput {
	out := domain.QuestionInput{Body: in.GetBody(), CorrectKey: in.GetCorrectKey(), SortOrder: int(in.GetSortOrder()), IsActive: true,
		StatementA: domain.PapiStatement{Text: in.GetStatementA().GetText(), Scale: in.GetStatementA().GetScale()},
		StatementB: domain.PapiStatement{Text: in.GetStatementB().GetText(), Scale: in.GetStatementB().GetScale()}}
	if in.IsActive != nil {
		out.IsActive = *in.IsActive
	}
	for _, o := range in.GetOptions() {
		out.Options = append(out.Options, domain.McqOption{Key: o.GetKey(), Text: o.GetText()})
	}
	return out
}

func (s *psikotesServer) CreateQuestion(ctx context.Context, req *pb.QuestionInput) (*pb.Question, error) {
	q, err := s.svc.Psikotes.CreateQuestion(ctx, req.GetInstrumentId(), questionInput(req))
	if err != nil {
		return nil, err
	}
	return questionPB(*q), nil
}

func (s *psikotesServer) UpdateQuestion(ctx context.Context, req *pb.UpdateQuestionRequest) (*pb.Question, error) {
	q, err := s.svc.Psikotes.UpdateQuestion(ctx, req.GetId(), questionInput(req.GetInput()))
	if err != nil {
		return nil, err
	}
	return questionPB(*q), nil
}

func (s *psikotesServer) DeleteQuestion(ctx context.Context, req *pb.QuestionRequest) (*emptypb.Empty, error) {
	if err := s.svc.Psikotes.DeleteQuestion(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func scoreDetailPB(d *domain.ScoreDetail) *pb.ScoreDetail {
	if d == nil {
		return nil
	}
	out := &pb.ScoreDetail{McqTotal: int32(d.Total), McqCorrect: int32(d.Correct), PapiAnswered: int32(d.Answered), PapiTotal: int32(d.PapiTotal),
		PapiScales: map[string]int32{}}
	for _, it := range d.PerQuestion {
		out.McqItems = append(out.McqItems, &pb.McqItem{QuestionId: it.ID, Given: it.Given, CorrectKey: it.CorrectKey, IsCorrect: it.IsCorrect})
	}
	for k, v := range d.Scales {
		out.PapiScales[k] = int32(v)
	}
	for _, x := range d.Dominant {
		out.PapiDominant = append(out.PapiDominant, &pb.PapiDominant{Code: x.Code, Label: x.Label, Count: int32(x.Count)})
	}
	return out
}

func sessionTestPB(t *domain.SessionTest) *pb.SessionTest {
	return &pb.SessionTest{Id: t.ID, SessionId: t.SessionID, InstrumentId: t.InstrumentID, InstrumentCode: t.InstrumentCode,
		InstrumentName: t.InstrumentName, InstrumentKind: t.InstrumentKind, Status: t.Status, Score: t.Score,
		ScoreDetail: scoreDetailPB(t.ScoreDetail), HasAttachment: t.AttachmentKey != nil, ReviewNotes: t.ReviewNotes,
		ReviewedByName: t.ReviewedByName, SortOrder: int32(t.SortOrder), StartedAt: tsPtr(t.StartedAt), CompletedAt: tsPtr(t.CompletedAt)}
}

func tallyPB(t domain.ProctorTally) *pb.ProctorTally {
	return &pb.ProctorTally{Flags: int32(t.Flags)}
}

func psikotesSessionPB(x *domain.PsikotesSession) *pb.PsikotesSession {
	out := &pb.PsikotesSession{Id: x.ID, CandidateId: x.CandidateID, Token: x.Token, Status: x.Status, WebcamConsent: x.WebcamConsent,
		InvitedAt: tsPtr(x.InvitedAt), ExpiresAt: tsPtr(x.ExpiresAt), StartedAt: tsPtr(x.StartedAt), CompletedAt: tsPtr(x.CompletedAt),
		CreatedByName: x.CreatedByName, Proctor: tallyPB(x.Proctor), Tests: []*pb.SessionTest{}}
	for i := range x.Tests {
		out.Tests = append(out.Tests, sessionTestPB(&x.Tests[i]))
	}
	return out
}

func summaryPB(x *domain.PsikotesSummary) *pb.PsikotesSummary {
	if x == nil {
		return nil
	}
	return &pb.PsikotesSummary{CandidateId: x.CandidateID, Recommendation: x.Recommendation, Notes: x.Notes, UpdatedByName: x.UpdatedByName,
		UpdatedAt: ts(x.UpdatedAt)}
}

func (s *psikotesServer) GetCandidatePsikotes(ctx context.Context, req *pb.CandidatePanelRequest) (*pb.PsikotesPanel, error) {
	summary, sessions, err := s.svc.Psikotes.Panel(ctx, req.GetCandidateId())
	if err != nil {
		return nil, err
	}
	out := &pb.PsikotesPanel{Summary: summaryPB(summary), Sessions: make([]*pb.PsikotesSession, len(sessions))}
	for i := range sessions {
		out.Sessions[i] = psikotesSessionPB(&sessions[i])
	}
	return out, nil
}

func (s *psikotesServer) InvitePsikotes(ctx context.Context, req *pb.InvitePsikotesRequest) (*pb.PsikotesSession, error) {
	x, err := s.svc.Psikotes.Invite(ctx, req.GetCandidateId(), req.GetInstrumentIds(), int(req.GetExpiresDays()))
	if err != nil {
		return nil, err
	}
	return psikotesSessionPB(x), nil
}

func (s *psikotesServer) SavePsikotesSummary(ctx context.Context, req *pb.SavePsikotesSummaryRequest) (*pb.PsikotesSummary, error) {
	x, err := s.svc.Psikotes.SaveSummary(ctx, domain.PsikotesSummary{CandidateID: req.GetCandidateId(), Recommendation: req.GetRecommendation(),
		Notes: req.GetNotes()})
	if err != nil {
		return nil, err
	}
	return summaryPB(x), nil
}

func (s *psikotesServer) GetTestAnswers(ctx context.Context, req *pb.SessionTestRequest) (*pb.TestAnswers, error) {
	kind, items, err := s.svc.Psikotes.Answers(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	out := &pb.TestAnswers{Kind: kind, Items: make([]*pb.AnswerItem, len(items))}
	for i, it := range items {
		q := questionPB(it.Question)
		out.Items[i] = &pb.AnswerItem{QuestionId: q.Id, Body: q.Body, Options: q.Options, StatementA: q.StatementA, StatementB: q.StatementB,
			Given: it.Given, CorrectKey: q.CorrectKey, IsCorrect: it.IsCorrect}
	}
	return out, nil
}

func (s *psikotesServer) ReviewTest(ctx context.Context, req *pb.ReviewTestRequest) (*pb.SessionTest, error) {
	t, err := s.svc.Psikotes.Review(ctx, req.GetId(), req.GetReviewNotes())
	if err != nil {
		return nil, err
	}
	return sessionTestPB(t), nil
}

func (s *psikotesServer) DownloadTestAttachment(ctx context.Context, req *pb.SessionTestRequest) (*pb.FileContent, error) {
	f, err := s.svc.Psikotes.Attachment(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return fileOf(f), nil
}

func (s *psikotesServer) ListProctorEvents(ctx context.Context, req *pb.ListProctorEventsRequest) (*pb.ListProctorEventsResponse, error) {
	list, err := s.svc.Psikotes.ProctorEvents(ctx, req.GetSessionType(), req.GetSessionId())
	if err != nil {
		return nil, err
	}
	out := &pb.ListProctorEventsResponse{Events: make([]*pb.ProctorEvent, len(list))}
	for i := range list {
		out.Events[i] = proctorEventPB(&list[i])
	}
	return out, nil
}

// ─── InterviewService ────────────────────────────────────────

type interviewServer struct {
	pb.UnimplementedInterviewServiceServer
	svc Services
}

func interviewSummaryPB(x *domain.InterviewSummary) *pb.InterviewSummary {
	if x == nil {
		return nil
	}
	return &pb.InterviewSummary{Summary: x.Summary, RelevanceScore: int32(x.Relevance.Score), RelevanceConclusion: x.Relevance.Conclusion,
		RelevanceReason: x.Relevance.Reason, Skills: nonNil(x.Skills), SalaryMentioned: x.Salary.Mentioned, SalaryValue: deref(x.Salary.Value),
		SalaryNote: x.Salary.Note, RedFlags: nonNil(x.RedFlags), FollowUps: nonNil(x.FollowUps), Limitations: x.Limitations}
}

func aiSessionPB(x *domain.AiSession) *pb.AiInterviewSession {
	out := &pb.AiInterviewSession{Id: x.ID, CandidateId: x.CandidateID, Token: x.Token, Status: x.Status, WebcamConsent: x.WebcamConsent,
		MaxQuestions: int32(domain.ClampMaxQuestions(x.MaxQuestions)), AiSummary: interviewSummaryPB(x.Summary), SummaryModel: x.SummaryModel,
		InvitedAt: tsPtr(x.InvitedAt), ExpiresAt: tsPtr(x.ExpiresAt), StartedAt: tsPtr(x.StartedAt), CompletedAt: tsPtr(x.CompletedAt),
		CreatedByName: x.CreatedByName, Proctor: tallyPB(x.Proctor), Turns: []*pb.AiInterviewTurn{}}
	for _, t := range x.Turns {
		out.Turns = append(out.Turns, &pb.AiInterviewTurn{Id: t.ID, TurnNo: int32(t.TurnNo), Topic: deref(t.Topic), Question: t.Question,
			AnswerTranscript: deref(t.Answer), AnswerMode: deref(t.AnswerMode), AskedAt: ts(t.AskedAt), AnsweredAt: tsPtr(t.AnsweredAt)})
	}
	return out
}

func (s *interviewServer) GetCandidateAiInterviews(ctx context.Context, req *pb.CandidatePanelRequest) (*pb.AiInterviewPanel, error) {
	list, err := s.svc.Interviews.AiPanel(ctx, req.GetCandidateId())
	if err != nil {
		return nil, err
	}
	out := &pb.AiInterviewPanel{Sessions: make([]*pb.AiInterviewSession, len(list))}
	for i := range list {
		out.Sessions[i] = aiSessionPB(&list[i])
	}
	return out, nil
}

func (s *interviewServer) InviteAiInterview(ctx context.Context, req *pb.InviteAiInterviewRequest) (*pb.AiInterviewSession, error) {
	x, err := s.svc.Interviews.InviteAi(ctx, req.GetCandidateId(), int(req.GetExpiresDays()), int(req.GetMaxQuestions()))
	if err != nil {
		return nil, err
	}
	return aiSessionPB(x), nil
}

func interviewPB(i *domain.Interview) *pb.Interview {
	out := &pb.Interview{Id: i.ID, CandidateId: i.CandidateID, CandidateName: i.CandidateName, JobOpeningTitle: i.JobOpeningTitle,
		InterviewerEmployeeId: i.InterviewerEmployeeID, InterviewerUserId: i.InterviewerUserID, InterviewerName: i.InterviewerName,
		InterviewDate: ts(i.InterviewDate), DurationMinutes: int32(i.DurationMinutes), Type: i.Type, Mode: i.Mode, MeetingLink: i.MeetingLink,
		Location: i.Location, Status: i.Status, OverallScore: i.OverallScore, Recommendation: i.Recommendation, Notes: i.Notes,
		CancelReason: i.CancelReason, CreatedAt: ts(i.CreatedAt), UpdatedAt: ts(i.UpdatedAt), Scorecard: []*pb.ScoreItem{}}
	for _, it := range i.Scorecard {
		out.Scorecard = append(out.Scorecard, &pb.ScoreItem{Criterion: it.Criterion, Score: int32(it.Score), Note: it.Note})
	}
	return out
}

func interviewInput(in *pb.InterviewInput) domain.InterviewInput {
	return domain.InterviewInput{CandidateID: in.GetCandidateId(), InterviewerEmployeeID: in.InterviewerEmployeeId,
		InterviewerName: in.GetInterviewerName(), InterviewDate: timeOf(in.GetInterviewDate()), DurationMinutes: int(in.GetDurationMinutes()),
		Type: in.GetType(), Mode: in.GetMode(), MeetingLink: in.GetMeetingLink(), Location: in.GetLocation(), Notes: in.GetNotes()}
}

func (s *interviewServer) ListInterviews(ctx context.Context, req *pb.ListInterviewsRequest) (*pb.ListInterviewsResponse, error) {
	list, err := s.svc.Interviews.List(ctx, app.ListFilter{CandidateID: req.GetCandidateId(), Status: req.GetStatus(),
		InterviewerEmployeeID: req.GetInterviewerEmployeeId(), DateFrom: req.GetDateFrom(), DateTo: req.GetDateTo()})
	if err != nil {
		return nil, err
	}
	out := &pb.ListInterviewsResponse{Interviews: make([]*pb.Interview, len(list))}
	for i := range list {
		out.Interviews[i] = interviewPB(&list[i])
	}
	return out, nil
}

func (s *interviewServer) ScheduleInterview(ctx context.Context, req *pb.InterviewInput) (*pb.Interview, error) {
	i, err := s.svc.Interviews.Schedule(ctx, interviewInput(req))
	if err != nil {
		return nil, err
	}
	return interviewPB(i), nil
}

func (s *interviewServer) UpdateInterview(ctx context.Context, req *pb.UpdateInterviewRequest) (*pb.Interview, error) {
	i, err := s.svc.Interviews.Update(ctx, req.GetId(), interviewInput(req.GetInput()))
	if err != nil {
		return nil, err
	}
	return interviewPB(i), nil
}

func (s *interviewServer) CancelInterview(ctx context.Context, req *pb.CancelInterviewRequest) (*pb.Interview, error) {
	i, err := s.svc.Interviews.Cancel(ctx, req.GetId(), req.GetReason())
	if err != nil {
		return nil, err
	}
	return interviewPB(i), nil
}

func (s *interviewServer) SubmitScorecard(ctx context.Context, req *pb.SubmitScorecardRequest) (*pb.Interview, error) {
	items := make([]domain.ScoreItem, len(req.GetScorecard()))
	for i, it := range req.GetScorecard() {
		items[i] = domain.ScoreItem{Criterion: it.GetCriterion(), Score: int(it.GetScore()), Note: it.GetNote()}
	}
	i, err := s.svc.Interviews.SubmitScorecard(ctx, req.GetId(), items, req.GetRecommendation(), req.GetNotes())
	if err != nil {
		return nil, err
	}
	return interviewPB(i), nil
}
