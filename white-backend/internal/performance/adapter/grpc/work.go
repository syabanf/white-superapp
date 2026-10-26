package grpcapi

import (
	"context"
	"fmt"

	pb "github.com/syabanf/white-superapp/white-backend/gen/go/white/performance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ─── LogbookService ──────────────────────────────────────────

type logbookServer struct {
	pb.UnimplementedLogbookServiceServer
	svc Services
}

func templatePB(t domain.LogbookTemplate) *pb.LogbookTemplate {
	out := &pb.LogbookTemplate{Id: t.ID, DepartmentId: t.DepartmentID, DepartmentName: t.DepartmentName, Name: t.Name, Description: t.Description, Frequency: t.Frequency,
		IsActive: t.IsActive, CreatedAt: ts(t.CreatedAt), Items: make([]*pb.LogbookTemplateItem, len(t.Items))}
	for i, it := range t.Items {
		out.Items[i] = &pb.LogbookTemplateItem{Id: it.ID, Title: it.Title, Description: it.Description, Weight: it.Weight,
			IsRequired: it.IsRequired, SortOrder: int32(it.SortOrder)}
	}
	return out
}

func entryPB(e domain.LogbookEntry) *pb.LogbookEntry {
	out := &pb.LogbookEntry{Id: e.ID, TemplateId: e.TemplateID, TemplateName: e.TemplateName, Frequency: e.Frequency, DepartmentId: e.DepartmentID,
		DepartmentName: e.DepartmentName, EmployeeId: e.EmployeeID, EmployeeName: e.EmployeeName, EntryDate: e.EntryDate, Title: e.Title, Status: e.Status, CompletionPercentage: e.Completion,
		KpiScore: e.KPIScore, Notes: e.Notes, SubmittedAt: tsPtr(e.SubmittedAt), ReviewedAt: tsPtr(e.ReviewedAt), ReviewNotes: e.ReviewNotes,
		Items: make([]*pb.LogbookEntryItem, len(e.Items))}
	for i, it := range e.Items {
		out.Items[i] = &pb.LogbookEntryItem{Id: it.ID, Title: it.Title, Description: it.Description, Weight: it.Weight, IsRequired: it.IsRequired,
			SortOrder: int32(it.SortOrder), IsChecked: it.IsChecked, CheckedAt: tsPtr(it.CheckedAt), Notes: it.Notes}
	}
	return out
}

func (s *logbookServer) GetLogbookMe(ctx context.Context, _ *emptypb.Empty) (*pb.LogbookMe, error) {
	m, err := s.svc.Logbook.Me(ctx)
	if err != nil {
		return nil, err
	}
	out := &pb.LogbookMe{Me: mePB(m.Actor), CanReview: m.CanReview, IsFullAccess: m.IsFullAccess}
	if m.Actor.Employee != nil {
		out.DepartmentName = m.Actor.Employee.DepartmentName
	}
	return out, nil
}

func (s *logbookServer) ListTemplates(ctx context.Context, req *pb.ListTemplatesRequest) (*pb.ListTemplatesResponse, error) {
	list, err := s.svc.Logbook.ListTemplates(ctx, req.DepartmentId, req.IncludeInactive)
	if err != nil {
		return nil, err
	}
	out := &pb.ListTemplatesResponse{Templates: make([]*pb.LogbookTemplate, len(list))}
	for i, t := range list {
		out.Templates[i] = templatePB(t)
	}
	return out, nil
}

func (s *logbookServer) CreateTemplate(ctx context.Context, req *pb.CreateTemplateRequest) (*pb.LogbookTemplate, error) {
	in := app.TemplateInput{DepartmentID: req.DepartmentId, Name: req.Name, Description: req.Description, Frequency: req.Frequency}
	for _, it := range req.Items {
		in.Items = append(in.Items, app.TemplateItemInput{Title: it.Title, Description: it.Description, Weight: it.Weight, IsRequired: it.IsRequired})
	}
	t, err := s.svc.Logbook.CreateTemplate(ctx, in)
	if err != nil {
		return nil, err
	}
	return templatePB(*t), nil
}

func (s *logbookServer) DeleteTemplate(ctx context.Context, req *pb.IdRequest) (*pb.DeleteTemplateResponse, error) {
	text, archived, err := s.svc.Logbook.DeleteTemplate(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteTemplateResponse{Message: text, Archived: archived}, nil
}

func (s *logbookServer) ListEntries(ctx context.Context, req *pb.ListEntriesRequest) (*pb.ListEntriesResponse, error) {
	list, total, err := s.svc.Logbook.ListEntries(ctx, app.EntryQuery{DepartmentID: req.DepartmentId, EmployeeID: req.EmployeeId, Status: req.Status,
		Date: req.Date, From: req.From, To: req.To, Mine: req.Mine, Page: int(req.Page), Limit: int(req.Limit)})
	if err != nil {
		return nil, err
	}
	out := &pb.ListEntriesResponse{Entries: make([]*pb.LogbookEntry, len(list)), Count: int32(total), Page: max(1, req.Page), Limit: req.Limit}
	for i, e := range list {
		out.Entries[i] = entryPB(e)
	}
	return out, nil
}

func (s *logbookServer) GetEntry(ctx context.Context, req *pb.IdRequest) (*pb.EntryResponse, error) {
	e, canReview, err := s.svc.Logbook.GetEntry(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.EntryResponse{Entry: entryPB(*e), CanReview: canReview}, nil
}

func (s *logbookServer) CreateEntry(ctx context.Context, req *pb.CreateEntryRequest) (*pb.LogbookEntry, error) {
	e, err := s.svc.Logbook.CreateEntry(ctx, req.TemplateId, req.EntryDate, req.Title, req.Notes)
	if err != nil {
		return nil, err
	}
	return entryPB(*e), nil
}

func (s *logbookServer) UpdateEntryItem(ctx context.Context, req *pb.UpdateEntryItemRequest) (*pb.LogbookEntry, error) {
	e, err := s.svc.Logbook.UpdateItem(ctx, req.ItemId, req.IsChecked, req.Notes)
	if err != nil {
		return nil, err
	}
	return entryPB(*e), nil
}

func (s *logbookServer) SubmitEntry(ctx context.Context, req *pb.SubmitEntryRequest) (*pb.LogbookEntry, error) {
	e, err := s.svc.Logbook.Submit(ctx, req.Id, req.Notes)
	if err != nil {
		return nil, err
	}
	return entryPB(*e), nil
}

func (s *logbookServer) ReviewEntry(ctx context.Context, req *pb.ReviewEntryRequest) (*pb.LogbookEntry, error) {
	e, err := s.svc.Logbook.Review(ctx, req.Id, req.Status == domain.LogbookRejected, req.ReviewNotes)
	if err != nil {
		return nil, err
	}
	return entryPB(*e), nil
}

func (s *logbookServer) DeleteEntry(ctx context.Context, req *pb.IdRequest) (*pb.MessageResponse, error) {
	if err := s.svc.Logbook.DeleteEntry(ctx, req.Id); err != nil {
		return nil, err
	}
	return msg("Logbook dihapus"), nil
}

func (s *logbookServer) LogbookSummary(ctx context.Context, req *pb.LogbookSummaryRequest) (*pb.LogbookSummaryResponse, error) {
	rows, names, err := s.svc.Logbook.Summary(ctx, req.DepartmentId, req.From, req.To)
	if err != nil {
		return nil, err
	}
	out := &pb.LogbookSummaryResponse{Departments: make([]*pb.LogbookSummaryResponse_Row, len(rows))}
	for i, r := range rows {
		out.Departments[i] = &pb.LogbookSummaryResponse_Row{DepartmentId: r.DepartmentID, DepartmentName: names[r.DepartmentID],
			TotalEntries: int32(r.TotalEntries), SubmittedEntries: int32(r.SubmittedEntries), ReviewedEntries: int32(r.ReviewedEntries),
			RejectedEntries: int32(r.RejectedEntries), AvgCompletion: r.AvgCompletion, AvgKpiScore: r.AvgKPIScore}
	}
	return out, nil
}

// ─── RecapService ────────────────────────────────────────────

type recapServer struct {
	pb.UnimplementedRecapServiceServer
	svc Services
}

func dailyPB(v app.DailyView) *pb.DailyRecap {
	r := v.DailyRecap
	out := &pb.DailyRecap{Id: r.ID, EmployeeId: r.EmployeeID, EmployeeName: v.EmployeeName, DepartmentId: r.DepartmentID,
		DepartmentName: v.DepartmentName, RecapDate: r.Date, Notes: r.Notes, GeneratedSummary: r.Summary, SummaryModel: r.SummaryModel,
		GeneratedAt: tsPtr(r.GeneratedAt), Saved: v.Saved, Tasks: []*pb.RecapTask{}, Logbook: []*pb.RecapLogbook{}}
	for _, t := range r.Items.Tasks {
		task := &pb.RecapTask{OccurrenceId: t.OccurrenceID, Title: t.Title, Status: t.Status, Progress: int32(t.Progress), Completed: t.Completed,
			Priority: t.Priority, Notes: t.Notes, Subtasks: make([]*pb.RecapSubtask, len(t.Subtasks))}
		for i, st := range t.Subtasks {
			task.Subtasks[i] = &pb.RecapSubtask{Title: st.Title, Checked: st.Checked}
		}
		out.Tasks = append(out.Tasks, task)
	}
	for _, l := range r.Items.Logbook {
		out.Logbook = append(out.Logbook, &pb.RecapLogbook{EntryId: l.EntryID, Title: l.Title, Status: l.Status, Completion: l.Completion, KpiScore: l.KPIScore})
	}
	return out
}

func dailiesPB(list []app.DailyView) *pb.ListDailyRecapsResponse {
	out := &pb.ListDailyRecapsResponse{Recaps: make([]*pb.DailyRecap, len(list))}
	for i, v := range list {
		out.Recaps[i] = dailyPB(v)
	}
	return out
}

func monthlyPB(v app.MonthlyView) *pb.MonthlyRecap {
	r, m := v.MonthlyRecap, v.Metrics
	metrics := &pb.MonthlyMetrics{TasksPlanned: int32(m.TasksPlanned), TasksDone: int32(m.TasksDone), TasksOnTime: int32(m.TasksOnTime),
		TasksOverdue: int32(m.TasksOverdue), CompletionRate: m.CompletionRate, OnTimeRate: m.OnTimeRate, KpiScore: m.KPIScore,
		LogbookEntries: int32(m.LogbookEntries), LogbookCompliance: m.LogbookCompliance, Members: int32(m.Members),
		ImprovementsOpen: int32(m.ImprovementsOpen)}
	if a := m.Attendance; a != nil {
		metrics.HasAttendance, metrics.WorkingDays, metrics.PresentDays, metrics.LateDays = true, int32(a.WorkingDays), int32(a.PresentDays), int32(a.LateDays)
		metrics.AbsentDays, metrics.LeaveDays, metrics.OvertimeHours = int32(a.AbsentDays), int32(a.LeaveDays), a.OvertimeHrs
	}
	return &pb.MonthlyRecap{Id: r.ID, Scope: r.Scope, SubjectId: r.SubjectID, SubjectName: v.SubjectName, DepartmentId: r.DepartmentID,
		DepartmentName: v.DepartmentName, PeriodYear: int32(r.Period.Year), PeriodMonth: int32(r.Period.Month), Metrics: metrics,
		Accomplishments: r.Accomplishments, Narrative: r.Narrative, NarrativeModel: r.NarrativeModel, GeneratedAt: ts(r.GeneratedAt)}
}

func (s *recapServer) GetMyDailyRecap(ctx context.Context, req *pb.DailyRecapRequest) (*pb.DailyRecap, error) {
	v, err := s.svc.Recaps.MyDaily(ctx, req.Date)
	if err != nil {
		return nil, err
	}
	return dailyPB(*v), nil
}

func (s *recapServer) UpdateMyDailyNotes(ctx context.Context, req *pb.UpdateDailyNotesRequest) (*pb.DailyRecap, error) {
	v, err := s.svc.Recaps.UpdateMyNotes(ctx, req.Date, req.Notes)
	if err != nil {
		return nil, err
	}
	return dailyPB(*v), nil
}

func (s *recapServer) RegenerateMyDailyRecap(ctx context.Context, req *pb.DailyRecapRequest) (*pb.DailyRecap, error) {
	v, err := s.svc.Recaps.RegenerateMyDaily(ctx, req.Date)
	if err != nil {
		return nil, err
	}
	return dailyPB(*v), nil
}

func (s *recapServer) ListTeamDailyRecaps(ctx context.Context, req *pb.TeamDailyRecapsRequest) (*pb.ListDailyRecapsResponse, error) {
	list, err := s.svc.Recaps.TeamDaily(ctx, req.Date, req.DepartmentId)
	if err != nil {
		return nil, err
	}
	return dailiesPB(list), nil
}

func (s *recapServer) ListDailyRecapHistory(ctx context.Context, req *pb.DailyRecapHistoryRequest) (*pb.ListDailyRecapsResponse, error) {
	list, err := s.svc.Recaps.History(ctx, req.EmployeeId, req.From, req.To, int(req.Limit))
	if err != nil {
		return nil, err
	}
	return dailiesPB(list), nil
}

func (s *recapServer) ListMonthlyRecaps(ctx context.Context, req *pb.ListMonthlyRecapsRequest) (*pb.ListMonthlyRecapsResponse, error) {
	q := app.MonthlyQuery{Scope: req.Scope, EmployeeID: req.EmployeeId, DepartmentID: req.DepartmentId}
	if p := period(req.PeriodYear, req.PeriodMonth); p.Valid() {
		q.Period = &p
	}
	list, err := s.svc.Recaps.ListMonthly(ctx, q)
	if err != nil {
		return nil, err
	}
	out := &pb.ListMonthlyRecapsResponse{Recaps: make([]*pb.MonthlyRecap, len(list))}
	for i, v := range list {
		out.Recaps[i] = monthlyPB(v)
	}
	return out, nil
}

func (s *recapServer) GetMonthlyRecap(ctx context.Context, req *pb.IdRequest) (*pb.MonthlyRecap, error) {
	v, err := s.svc.Recaps.GetMonthly(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return monthlyPB(*v), nil
}

func (s *recapServer) RegenerateMonthlyRecap(ctx context.Context, req *pb.RegenerateMonthlyRecapRequest) (*pb.RegenerateMonthlyRecapResponse, error) {
	v, filed, err := s.svc.Recaps.RegenerateMonthly(ctx, req.Scope, req.SubjectId, period(req.PeriodYear, req.PeriodMonth))
	if err != nil {
		return nil, err
	}
	return &pb.RegenerateMonthlyRecapResponse{Recap: monthlyPB(*v), SuggestionsFiled: int32(filed)}, nil
}

func (s *recapServer) ExportMonthlyRecap(ctx context.Context, req *pb.IdRequest) (*pb.FileContent, error) {
	name, data, err := s.svc.Recaps.ExportMonthly(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.FileContent{FileName: name, ContentType: "application/pdf", Data: data}, nil
}

// ─── ImprovementService ──────────────────────────────────────

type improvementServer struct {
	pb.UnimplementedImprovementServiceServer
	svc Services
}

func improvementPB(v app.ImprovementView, today string) *pb.Improvement {
	i := v.Improvement
	return &pb.Improvement{Id: i.ID, Scope: i.Scope, EmployeeId: i.EmployeeID, EmployeeName: v.EmployeeName, DepartmentId: i.DepartmentID,
		DepartmentName: v.DepartmentName, Source: i.Source, SourceRef: i.SourceRef, Title: i.Title, Description: i.Description,
		OwnerEmployeeId: i.OwnerEmployeeID, OwnerName: v.OwnerName, DueDate: i.DueDate, Status: i.Status, LinkedTaskId: i.LinkedTaskID,
		CompletedAt: tsPtr(i.CompletedAt), CreatedAt: ts(i.CreatedAt), UpdatedAt: ts(i.UpdatedAt),
		Overdue: domain.ImprovementActive(i.Status) && i.DueDate != nil && *i.DueDate < today}
}

func retroPB(v app.RetroView) *pb.Retro {
	r := v.RetroSession
	out := &pb.Retro{Id: r.ID, DepartmentId: r.DepartmentID, DepartmentName: v.DepartmentName, PeriodYear: int32(r.Period.Year), PeriodMonth: int32(r.Period.Month),
		WentWell: r.WentWell, ToImprove: r.ToImprove, Status: r.Status, ClosedAt: tsPtr(r.ClosedAt), UpdatedAt: ts(r.UpdatedAt),
		Actions: make([]*pb.RetroAction, len(r.Actions))}
	for i, a := range r.Actions {
		out.Actions[i] = &pb.RetroAction{Description: a.Description, OwnerEmployeeId: a.OwnerEmployeeID, DueDate: a.DueDate, ImprovementId: a.ImprovementID}
	}
	return out
}

func (s *improvementServer) today() string { return s.svc.Improvements.Today() }

func (s *improvementServer) ListImprovements(ctx context.Context, req *pb.ListImprovementsRequest) (*pb.ListImprovementsResponse, error) {
	list, err := s.svc.Improvements.List(ctx, app.ImprovementQuery{Scope: req.Scope, EmployeeID: req.EmployeeId, DepartmentID: req.DepartmentId,
		Status: req.Status, Source: req.Source, Mine: req.Mine})
	if err != nil {
		return nil, err
	}
	today := s.today()
	out := &pb.ListImprovementsResponse{Improvements: make([]*pb.Improvement, len(list))}
	for i, v := range list {
		out.Improvements[i] = improvementPB(v, today)
	}
	return out, nil
}

func (s *improvementServer) GetImprovement(ctx context.Context, req *pb.IdRequest) (*pb.Improvement, error) {
	v, err := s.svc.Improvements.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return improvementPB(*v, s.today()), nil
}

func improvementInput(req *pb.ImprovementInput) app.ImprovementInput {
	return app.ImprovementInput{Scope: req.Scope, EmployeeID: req.EmployeeId, DepartmentID: req.DepartmentId, Title: req.Title,
		Description: req.Description, OwnerEmployeeID: req.OwnerEmployeeId, DueDate: req.DueDate, Status: req.Status, LinkedTaskID: req.LinkedTaskId}
}

func (s *improvementServer) CreateImprovement(ctx context.Context, req *pb.ImprovementInput) (*pb.Improvement, error) {
	v, err := s.svc.Improvements.Create(ctx, improvementInput(req))
	if err != nil {
		return nil, err
	}
	return improvementPB(*v, s.today()), nil
}

func (s *improvementServer) UpdateImprovement(ctx context.Context, req *pb.ImprovementInput) (*pb.Improvement, error) {
	v, err := s.svc.Improvements.Update(ctx, req.Id, improvementInput(req))
	if err != nil {
		return nil, err
	}
	return improvementPB(*v, s.today()), nil
}

func (s *improvementServer) DeleteImprovement(ctx context.Context, req *pb.IdRequest) (*pb.MessageResponse, error) {
	if err := s.svc.Improvements.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	return msg("Item perbaikan dihapus"), nil
}

func (s *improvementServer) AcceptSuggestion(ctx context.Context, req *pb.DecideSuggestionRequest) (*pb.Improvement, error) {
	v, err := s.svc.Improvements.Decide(ctx, req.Id, true, req.OwnerEmployeeId, req.DueDate)
	if err != nil {
		return nil, err
	}
	return improvementPB(*v, s.today()), nil
}

func (s *improvementServer) DeclineSuggestion(ctx context.Context, req *pb.DecideSuggestionRequest) (*pb.Improvement, error) {
	v, err := s.svc.Improvements.Decide(ctx, req.Id, false, nil, nil)
	if err != nil {
		return nil, err
	}
	return improvementPB(*v, s.today()), nil
}

func (s *improvementServer) ListRetros(ctx context.Context, req *pb.ListRetrosRequest) (*pb.ListRetrosResponse, error) {
	list, err := s.svc.Improvements.ListRetros(ctx, req.DepartmentId)
	if err != nil {
		return nil, err
	}
	out := &pb.ListRetrosResponse{Retros: make([]*pb.Retro, len(list))}
	for i, r := range list {
		out.Retros[i] = retroPB(r)
	}
	return out, nil
}

func (s *improvementServer) GetRetro(ctx context.Context, req *pb.IdRequest) (*pb.Retro, error) {
	r, err := s.svc.Improvements.GetRetro(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return retroPB(*r), nil
}

func (s *improvementServer) SaveRetro(ctx context.Context, req *pb.SaveRetroRequest) (*pb.Retro, error) {
	in := app.RetroInput{DepartmentID: req.DepartmentId, Period: period(req.PeriodYear, req.PeriodMonth), WentWell: req.WentWell, ToImprove: req.ToImprove}
	for _, a := range req.Actions {
		in.Actions = append(in.Actions, domain.RetroAction{Description: a.Description, OwnerEmployeeID: a.OwnerEmployeeId, DueDate: a.DueDate})
	}
	r, err := s.svc.Improvements.SaveRetro(ctx, in)
	if err != nil {
		return nil, err
	}
	return retroPB(*r), nil
}

func (s *improvementServer) CloseRetro(ctx context.Context, req *pb.IdRequest) (*pb.CloseRetroResponse, error) {
	r, filed, err := s.svc.Improvements.CloseRetro(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.CloseRetroResponse{Retro: retroPB(*r), ImprovementsFiled: int32(filed),
		Message: fmt.Sprintf("Retrospektif ditutup, %d aksi menjadi item perbaikan", filed)}, nil
}
