package grpcapi

import (
	"context"

	pb "github.com/syabanf/white-superapp/white-backend/gen/go/white/performance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"google.golang.org/protobuf/types/known/emptypb"
)

type taskServer struct {
	pb.UnimplementedTaskServiceServer
	svc Services
}

func taskItemPB(v app.TaskView) *pb.TaskItem {
	it := v.TaskItem
	return &pb.TaskItem{OccurrenceId: it.OccurrenceID, TaskId: it.TaskID, Title: it.Title, Description: it.Description, DepartmentId: it.DepartmentID,
		DepartmentName: v.DepartmentName, AssigneeEmployeeId: it.AssigneeID, AssigneeName: v.AssigneeName, CreatedByName: it.CreatorName,
		OccurrenceDate: it.Date, Status: it.Status, ProgressPercent: i32Ptr(it.ProgressPercent), StartedAt: tsPtr(it.StartedAt),
		DoneAt: tsPtr(it.DoneAt), DoneByName: it.DoneByName, ReviewNotes: it.ReviewNotes, ReviewedByName: it.ReviewedByName,
		Recurrence: it.Recurrence, Priority: it.Priority, ReviewRequired: it.ReviewRequired, KpiIndicatorId: it.KPIIndicatorID,
		KpiIndicatorCode: it.KPIIndicatorCode, KpiIndicatorName: it.KPIIndicatorName, KpiContribution: it.KPIContribution,
		SubtaskCount: int32(it.SubtaskCount), CommentCount: int32(it.CommentCount), AttachmentCount: int32(it.AttachmentCount),
		LastNote: it.LastNote, LastNoteBy: it.LastNoteBy, LastNoteAt: tsPtr(it.LastNoteAt), Progress: int32(v.Progress), Overdue: v.Overdue,
		Mine: v.Mine, CanWork: v.CanWork, CanReview: v.CanReview, CanManage: v.CanManage, WeeklyDay: i32Ptr(it.WeeklyDay),
		MonthlyDay: i32Ptr(it.MonthlyDay), DueDate: it.DueDate}
}

func taskItemsPB(list []app.TaskView) []*pb.TaskItem {
	out := make([]*pb.TaskItem, len(list))
	for i, v := range list {
		out[i] = taskItemPB(v)
	}
	return out
}

func attachmentPB(a domain.TaskAttachment) *pb.TaskAttachment {
	return &pb.TaskAttachment{Id: a.ID, OccurrenceId: a.OccurrenceID, Name: a.Name, Mime: a.Mime, SizeBytes: a.SizeBytes,
		UploadedByName: a.UploadedByName, CreatedAt: ts(a.CreatedAt)}
}

func (s *taskServer) GetTaskMeta(ctx context.Context, _ *emptypb.Empty) (*pb.TaskMeta, error) {
	m, err := s.svc.Tasks.Meta(ctx)
	if err != nil {
		return nil, err
	}
	out := &pb.TaskMeta{Me: mePB(m.Actor), TeamScope: m.TeamScope, Today: m.Today, Timezone: "Asia/Jakarta",
		Departments: []*pb.Department{}, Members: []*pb.Person{}, Indicators: []*pb.TaskIndicator{}}
	for _, d := range m.Departments {
		out.Departments = append(out.Departments, departmentPB(d))
	}
	for _, e := range m.Members {
		out.Members = append(out.Members, personPB(e))
	}
	for _, ind := range m.Indicators {
		depts := ind.DepartmentIDs
		if depts == nil {
			depts = []string{}
		}
		out.Indicators = append(out.Indicators, &pb.TaskIndicator{Id: ind.ID, Code: ind.Code, Name: ind.Name, Unit: ind.Unit,
			Direction: ind.Direction, DepartmentIds: depts})
	}
	return out, nil
}

func (s *taskServer) ListTaskItems(ctx context.Context, req *pb.ListTaskItemsRequest) (*pb.ListTaskItemsResponse, error) {
	list, err := s.svc.Tasks.ListItems(ctx, app.ItemFilter{Scope: req.Scope, View: req.View, Month: req.Month, Q: req.Q,
		DepartmentID: req.DepartmentId, AssigneeID: req.AssigneeId, Status: req.Status, IndicatorID: req.IndicatorId})
	if err != nil {
		return nil, err
	}
	return &pb.ListTaskItemsResponse{Items: taskItemsPB(list)}, nil
}

func (s *taskServer) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.IdResponse, error) {
	id, err := s.svc.Tasks.Create(ctx, app.TaskInput{DepartmentID: req.DepartmentId, AssigneeID: req.AssigneeEmployeeId, Priority: req.Priority,
		KPIIndicatorID: req.KpiIndicatorId, KPIContribution: req.KpiContribution,
		Body: domain.TaskBody{Title: req.Title, Description: req.Description, Recurrence: req.Recurrence, DueDate: req.DueDate,
			WeeklyDay: int(req.WeeklyDay), MonthlyDay: int(req.MonthlyDay), Subtasks: req.Subtasks}})
	if err != nil {
		return nil, err
	}
	return &pb.IdResponse{Id: id, Message: "Task dibuat"}, nil
}

func (s *taskServer) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.MessageResponse, error) {
	p := app.TaskPatch{Title: req.Title, Description: req.Description, Priority: req.Priority}
	if req.AssigneeEmployeeId != nil || req.ClearAssignee {
		p.AssigneeSet, p.AssigneeID = true, req.AssigneeEmployeeId
		if req.ClearAssignee {
			p.AssigneeID = nil
		}
	}
	if req.KpiIndicatorId != nil || req.ClearKpiIndicator {
		p.KPISet, p.KPIIndicatorID = true, req.KpiIndicatorId
		if req.ClearKpiIndicator {
			p.KPIIndicatorID = nil
		}
	}
	if req.KpiContribution != nil || req.ClearKpiContribution {
		p.KPIContributionSet, p.KPIContribution = true, req.KpiContribution
		if req.ClearKpiContribution {
			p.KPIContribution = nil
		}
	}
	if err := s.svc.Tasks.Update(ctx, req.Id, p); err != nil {
		return nil, err
	}
	return msg("Task diperbarui"), nil
}

func (s *taskServer) DeactivateTask(ctx context.Context, req *pb.IdRequest) (*pb.MessageResponse, error) {
	if err := s.svc.Tasks.Deactivate(ctx, req.Id); err != nil {
		return nil, err
	}
	return msg("Task dinonaktifkan"), nil
}

func (s *taskServer) GetOccurrence(ctx context.Context, req *pb.IdRequest) (*pb.TaskDetail, error) {
	d, err := s.svc.Tasks.GetOccurrence(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	out := &pb.TaskDetail{Item: taskItemPB(d.TaskView), Subtasks: make([]*pb.Subtask, len(d.Subtasks)), Updates: make([]*pb.TaskUpdate, len(d.Updates)),
		Attachments: make([]*pb.TaskAttachment, len(d.Attachments))}
	for i, st := range d.Subtasks {
		out.Subtasks[i] = &pb.Subtask{Id: st.ID, Title: st.Title, Weight: st.Weight, SortOrder: int32(st.SortOrder), IsChecked: st.IsChecked,
			CheckedByName: st.CheckedByName, CheckedAt: tsPtr(st.CheckedAt)}
	}
	for i, u := range d.Updates {
		out.Updates[i] = &pb.TaskUpdate{Id: u.ID, Kind: u.Kind, Note: u.Note, ProgressPercent: i32Ptr(u.ProgressPercent), AuthorName: u.AuthorName,
			Detail: jsonString(u.Detail), CreatedAt: ts(u.CreatedAt)}
	}
	for i, a := range d.Attachments {
		out.Attachments[i] = attachmentPB(a)
	}
	if c := d.Contribution; c != nil {
		out.Contribution = &pb.TaskContribution{IndicatorId: c.IndicatorID, PeriodYear: int32(c.Period.Year), PeriodMonth: int32(c.Period.Month),
			Amount: c.Amount, RecordedAt: ts(c.RecordedAt)}
	}
	return out, nil
}

func (s *taskServer) OccurrenceAction(ctx context.Context, req *pb.OccurrenceActionRequest) (*pb.OccurrenceActionResponse, error) {
	res, err := s.svc.Tasks.Act(ctx, req.Id, app.OccurrenceAction{Action: req.Action, Notes: req.Notes, Note: req.Note,
		ProgressPercent: intPtr(req.ProgressPercent), SubtaskID: req.SubtaskId, Checked: req.Checked})
	if err != nil {
		return nil, err
	}
	return &pb.OccurrenceActionResponse{Message: res.Message, Progress: res.Progress, Complete: res.Complete}, nil
}

func (s *taskServer) AddAttachment(ctx context.Context, req *pb.AddAttachmentRequest) (*pb.TaskAttachment, error) {
	a, err := s.svc.Tasks.AddAttachment(ctx, req.OccurrenceId, req.Name, req.Mime, req.Data)
	if err != nil {
		return nil, err
	}
	return attachmentPB(*a), nil
}

func (s *taskServer) GetAttachment(ctx context.Context, req *pb.AttachmentRequest) (*pb.FileContent, error) {
	a, data, err := s.svc.Tasks.GetAttachment(ctx, req.OccurrenceId, req.AttachmentId)
	if err != nil {
		return nil, err
	}
	return &pb.FileContent{FileName: a.Name, ContentType: a.Mime, Data: data}, nil
}

func (s *taskServer) DeleteAttachment(ctx context.Context, req *pb.AttachmentRequest) (*pb.MessageResponse, error) {
	if err := s.svc.Tasks.DeleteAttachment(ctx, req.OccurrenceId, req.AttachmentId); err != nil {
		return nil, err
	}
	return msg("Lampiran dihapus"), nil
}

func (s *taskServer) GetKpiTasks(ctx context.Context, req *pb.GetKpiTasksRequest) (*pb.GetKpiTasksResponse, error) {
	list, err := s.svc.Tasks.KPIView(ctx, req.DepartmentId, req.Month)
	if err != nil {
		return nil, err
	}
	out := &pb.GetKpiTasksResponse{DepartmentId: req.DepartmentId, Month: req.Month, Indicators: make([]*pb.IndicatorTasks, len(list))}
	for i, it := range list {
		out.Indicators[i] = &pb.IndicatorTasks{IndicatorId: it.ID, Code: it.Code, Name: it.Name, Unit: it.Unit, Direction: it.Direction,
			Weight: it.Weight, Total: int32(it.Total), Approved: int32(it.Approved), WaitingReview: int32(it.WaitingReview), Open: int32(it.Open),
			Overdue: int32(it.Overdue), Rate: it.Rate, ExpectedTotal: it.ExpectedTotal, ContributionTotal: it.ContributionTotal,
			Tasks: taskItemsPB(it.Tasks)}
	}
	return out, nil
}

func (s *taskServer) ListContributions(ctx context.Context, req *pb.ListContributionsRequest) (*pb.ListContributionsResponse, error) {
	list, err := s.svc.Tasks.Contributions(ctx, req.EmployeeId, req.Month)
	if err != nil {
		return nil, err
	}
	out := &pb.ListContributionsResponse{Items: make([]*pb.ListContributionsResponse_Item, len(list))}
	for i, c := range list {
		out.Items[i] = &pb.ListContributionsResponse_Item{IndicatorId: c.IndicatorID, Code: c.Code, TasksTotal: int32(c.TasksTotal),
			TasksDone: int32(c.TasksDone), ContributionTotal: c.ContributionTotal}
	}
	return out, nil
}

func monitorSummaryPB(s domain.MonitorSummary) *pb.MonitorSummary {
	return &pb.MonitorSummary{Scheduled: int32(s.Scheduled), Due: int32(s.Due), Approved: int32(s.Approved), WaitingReview: int32(s.WaitingReview),
		InProgress: int32(s.InProgress), NotStarted: int32(s.NotStarted), Rejected: int32(s.Rejected), Overdue: int32(s.Overdue),
		Completion: s.Completion, AvgProgress: i32Ptr(s.AvgProgress)}
}

func monitorRowsPB(rows []domain.MonitorRow) []*pb.MonitorRow {
	out := make([]*pb.MonitorRow, len(rows))
	for i, r := range rows {
		out[i] = &pb.MonitorRow{OccurrenceId: r.OccurrenceID, TaskId: r.TaskID, Title: r.Title, DepartmentId: r.DepartmentID,
			DepartmentName: r.DepartmentName, AssigneeId: r.AssigneeID, AssigneeName: r.AssigneeName, OccurrenceDate: r.Date,
			Status: r.Status, Progress: int32(r.Progress), LastNote: r.LastNote}
	}
	return out
}

func (s *taskServer) GetMonitor(ctx context.Context, req *pb.GetMonitorRequest) (*pb.MonitorResponse, error) {
	m, err := s.svc.Tasks.Monitor(ctx, req.DepartmentId, req.Month)
	if err != nil {
		return nil, err
	}
	out := &pb.MonitorResponse{Start: m.Span.Start, End: m.Span.End, Today: m.Today, Summary: monitorSummaryPB(m.Summary), Scope: m.Scope,
		Overdue: monitorRowsPB(m.Overdue), WaitingReview: monitorRowsPB(m.WaitingReview), Departments: make([]*pb.MonitorResponse_DepartmentRow, len(m.Departments))}
	for i, d := range m.Departments {
		row := &pb.MonitorResponse_DepartmentRow{DepartmentId: d.DepartmentID, DepartmentName: d.DepartmentName, Summary: monitorSummaryPB(d.MonitorSummary),
			Assignees: make([]*pb.MonitorResponse_Assignee, len(d.Assignees))}
		for j, a := range d.Assignees {
			row.Assignees[j] = &pb.MonitorResponse_Assignee{AssigneeId: a.AssigneeID, AssigneeName: a.AssigneeName, Summary: monitorSummaryPB(a.MonitorSummary)}
		}
		out.Departments[i] = row
	}
	return out, nil
}
