package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
)

// ─── Department tasks ────────────────────────────────────────

func (s *Store) ActiveTaskSchedules(ctx context.Context, departmentIDs []string) ([]domain.TaskScheduleRow, error) {
	rows, err := s.q.ActiveTaskSchedules(ctx, filterIDs(departmentIDs))
	return each(rows, err, func(r sqlc.ActiveTaskSchedulesRow) domain.TaskScheduleRow {
		created := day(r.CreatedDate)
		return domain.TaskScheduleRow{TaskID: r.ID, TaskSchedule: domain.TaskSchedule{Recurrence: r.Recurrence, WeeklyDay: intp(r.WeeklyDay),
			MonthlyDay: intp(r.MonthlyDay), DueDate: dayPtr(r.DueDate), CreatedDate: &created}}
	})
}

func (s *Store) ActiveTaskDepartments(ctx context.Context) ([]string, error) {
	return s.q.ActiveTaskDepartments(ctx)
}

func (s *Store) InsertOccurrences(ctx context.Context, taskIDs, dates []string) error {
	if len(taskIDs) == 0 {
		return nil
	}
	days := make([]time.Time, len(dates))
	for i, d := range dates {
		days[i] = toDay(d)
	}
	return s.q.InsertOccurrences(ctx, sqlc.InsertOccurrencesParams{TaskIds: taskIDs, Dates: days})
}

func (s *Store) InsertTask(ctx context.Context, t domain.Task) error {
	return s.q.InsertTask(ctx, sqlc.InsertTaskParams{ID: t.ID, DepartmentID: t.DepartmentID, AssigneeEmployeeID: t.AssigneeEmployeeID,
		Title: t.Title, Description: t.Description, Recurrence: t.Recurrence, WeeklyDay: i32p(t.WeeklyDay), MonthlyDay: i32p(t.MonthlyDay),
		DueDate: toDayPtr(t.DueDate), Priority: t.Priority, KpiIndicatorID: t.KPIIndicatorID, KpiContribution: t.KPIContribution,
		Timezone: t.Timezone, ReviewRequired: t.ReviewRequired, CreatedByEmployeeID: t.CreatedByEmployeeID, CreatedByUserID: t.CreatedByUserID,
		CreatedByName: t.CreatedByName, CreatedAt: t.CreatedAt})
}

func (s *Store) InsertSubtask(ctx context.Context, taskID string, st domain.Subtask, order int) error {
	return s.q.InsertSubtask(ctx, sqlc.InsertSubtaskParams{ID: newID(), TaskID: taskID, Title: st.Title, Weight: st.Weight, SortOrder: int32(order)})
}

func toTask(r sqlc.PerformanceDepartmentTask) domain.Task {
	return domain.Task{ID: r.ID, DepartmentID: r.DepartmentID, AssigneeEmployeeID: r.AssigneeEmployeeID, Title: r.Title, Description: r.Description,
		Recurrence: r.Recurrence, WeeklyDay: intp(r.WeeklyDay), MonthlyDay: intp(r.MonthlyDay), DueDate: dayPtr(r.DueDate), IsActive: r.IsActive,
		Priority: r.Priority, KPIIndicatorID: r.KpiIndicatorID, KPIContribution: r.KpiContribution, Timezone: r.Timezone,
		ReviewRequired: r.ReviewRequired, CreatedByEmployeeID: r.CreatedByEmployeeID, CreatedByUserID: r.CreatedByUserID,
		CreatedByName: r.CreatedByName, UpdatedBy: r.UpdatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (s *Store) GetTask(ctx context.Context, id string) (*domain.Task, error) {
	r, err := s.q.GetTask(ctx, id)
	return one(r, err, toTask)
}

func (s *Store) UpdateTask(ctx context.Context, t domain.Task) error {
	return s.q.UpdateTask(ctx, sqlc.UpdateTaskParams{ID: t.ID, Title: t.Title, Description: t.Description, Priority: t.Priority,
		AssigneeEmployeeID: t.AssigneeEmployeeID, KpiIndicatorID: t.KPIIndicatorID, KpiContribution: t.KPIContribution,
		IsActive: t.IsActive, UpdatedBy: t.UpdatedBy})
}

func (s *Store) QueryItems(ctx context.Context, q domain.ItemQuery) ([]domain.TaskItem, error) {
	rows, err := s.q.QueryItems(ctx, sqlc.QueryItemsParams{OccurrenceID: q.OccurrenceID, DepartmentIds: filterIDs(q.DepartmentIDs),
		MineOnly: q.MineOnly, MineEmployeeID: q.MineEmployeeID, MineDepartmentID: q.MineDepartmentID, MineUserID: q.MineUserID,
		SpanStart: q.SpanStart, SpanEnd: q.SpanEnd, OverdueBefore: q.OverdueBefore, AssigneeID: q.AssigneeID, Status: q.Status,
		IndicatorID: q.IndicatorID, KpiOnly: q.KPIOnly, Q: q.Q})
	return each(rows, err, func(r sqlc.QueryItemsRow) domain.TaskItem {
		it := domain.TaskItem{OccurrenceID: r.OccurrenceID, TaskID: r.TaskID, Title: r.Title, Description: r.Description,
			DepartmentID: r.DepartmentID, AssigneeID: r.AssigneeEmployeeID, CreatorUserID: r.CreatedByUserID, CreatorName: r.CreatedByName,
			Date: day(r.OccurrenceDate), Status: r.Status, ProgressPercent: intp(r.ProgressPercent), StartedAt: r.StartedAt, DoneAt: r.DoneAt,
			DoneByName: r.DoneByName, ReviewNotes: r.ReviewNotes, ReviewedByName: r.ReviewedByName, Recurrence: r.Recurrence,
			Priority: r.Priority, ReviewRequired: r.ReviewRequired, Timezone: r.Timezone, KPIIndicatorID: r.KpiIndicatorID,
			KPIIndicatorCode: r.KpiCode, KPIIndicatorName: r.KpiName, KPIContribution: r.KpiContribution, SubtaskCount: int(r.SubtaskCount),
			CommentCount: int(r.CommentCount), AttachmentCount: int(r.AttachmentCount), LastNote: r.LastNote,
			WeeklyDay: intp(r.WeeklyDay), MonthlyDay: intp(r.MonthlyDay), DueDate: dayPtr(r.DueDate)}
		if r.SubtaskCount > 0 {
			total, done := r.SubtaskTotal, r.SubtaskDone
			it.SubtaskTotal, it.SubtaskDone = &total, &done
		}
		if r.LastNote != nil {
			by, at := r.LastNoteBy, r.LastNoteAt
			it.LastNoteBy, it.LastNoteAt = &by, &at
		}
		return it
	})
}

func toOccurrence(r sqlc.GetOccurrenceRow) domain.Occurrence {
	return domain.Occurrence{ID: r.ID, TaskID: r.TaskID, Date: day(r.OccurrenceDate), Status: r.Status, ProgressPercent: intp(r.ProgressPercent),
		StartedAt: r.StartedAt, DoneAt: r.DoneAt, DoneBy: r.DoneBy, DoneByName: r.DoneByName, ReviewNotes: r.ReviewNotes,
		ReviewedBy: r.ReviewedBy, ReviewedByName: r.ReviewedByName, ReviewedAt: r.ReviewedAt, Title: r.Title, DepartmentID: r.DepartmentID,
		AssigneeID: r.AssigneeEmployeeID, ReviewRequired: r.ReviewRequired, CreatorUserID: r.CreatedByUserID, KPIIndicatorID: r.KpiIndicatorID}
}

func (s *Store) GetOccurrence(ctx context.Context, id string) (*domain.Occurrence, error) {
	r, err := s.q.GetOccurrence(ctx, id)
	return one(r, err, toOccurrence)
}

func (s *Store) SaveOccurrence(ctx context.Context, o domain.Occurrence) error {
	return s.q.SaveOccurrence(ctx, sqlc.SaveOccurrenceParams{ID: o.ID, Status: o.Status, ProgressPercent: i32p(o.ProgressPercent),
		StartedAt: o.StartedAt, DoneAt: o.DoneAt, DoneBy: o.DoneBy, DoneByName: o.DoneByName, ReviewNotes: o.ReviewNotes,
		ReviewedBy: o.ReviewedBy, ReviewedByName: o.ReviewedByName, ReviewedAt: o.ReviewedAt})
}

func (s *Store) CountSubtasks(ctx context.Context, taskID string) (int, error) {
	n, err := s.q.CountSubtasks(ctx, taskID)
	return int(n), err
}

func (s *Store) SubtaskExists(ctx context.Context, taskID, subtaskID string) (bool, error) {
	return s.q.SubtaskExists(ctx, sqlc.SubtaskExistsParams{ID: subtaskID, TaskID: taskID})
}

func (s *Store) ListSubtaskStates(ctx context.Context, occurrenceID, taskID string) ([]domain.SubtaskState, error) {
	rows, err := s.q.ListSubtaskStates(ctx, sqlc.ListSubtaskStatesParams{OccurrenceID: occurrenceID, TaskID: taskID})
	return each(rows, err, func(r sqlc.ListSubtaskStatesRow) domain.SubtaskState {
		return domain.SubtaskState{ID: r.ID, Title: r.Title, Weight: r.Weight, SortOrder: int(r.SortOrder), IsChecked: r.IsChecked,
			CheckedByName: r.CheckedByName, CheckedAt: r.CheckedAt}
	})
}

func (s *Store) UpsertOccurrenceItem(ctx context.Context, occurrenceID, subtaskID string, checked bool, at *time.Time, by *string) error {
	return s.q.UpsertOccurrenceItem(ctx, sqlc.UpsertOccurrenceItemParams{OccurrenceID: occurrenceID, SubtaskID: subtaskID, IsChecked: checked,
		CheckedAt: at, CheckedByName: by})
}

func (s *Store) SubtaskProgress(ctx context.Context, occurrenceID, taskID string) (float64, float64, error) {
	r, err := s.q.SubtaskProgress(ctx, sqlc.SubtaskProgressParams{OccurrenceID: occurrenceID, TaskID: taskID})
	return r.Total, r.Done, err
}

func (s *Store) InsertTaskUpdate(ctx context.Context, u domain.TaskUpdate) error {
	return s.q.InsertTaskUpdate(ctx, sqlc.InsertTaskUpdateParams{ID: u.ID, OccurrenceID: u.OccurrenceID, Kind: u.Kind, Note: u.Note,
		ProgressPercent: i32p(u.ProgressPercent), AuthorEmployeeID: u.AuthorEmployeeID, AuthorUserID: u.AuthorUserID,
		AuthorName: u.AuthorName, Detail: jsonOr(u.Detail, "{}"), CreatedAt: u.CreatedAt})
}

func (s *Store) ListTaskUpdates(ctx context.Context, occurrenceID string, limit int) ([]domain.TaskUpdate, error) {
	rows, err := s.q.ListTaskUpdates(ctx, sqlc.ListTaskUpdatesParams{OccurrenceID: occurrenceID, Limit: int32(limit)})
	return each(rows, err, func(r sqlc.PerformanceDepartmentTaskUpdate) domain.TaskUpdate {
		detail := map[string]any{}
		_ = json.Unmarshal(r.Detail, &detail)
		return domain.TaskUpdate{ID: r.ID, OccurrenceID: r.OccurrenceID, Kind: r.Kind, Note: r.Note, ProgressPercent: intp(r.ProgressPercent),
			AuthorEmployeeID: r.AuthorEmployeeID, AuthorUserID: r.AuthorUserID, AuthorName: r.AuthorName, Detail: detail, CreatedAt: r.CreatedAt}
	})
}

func toAttachment(r sqlc.PerformanceDepartmentTaskAttachment) domain.TaskAttachment {
	return domain.TaskAttachment{ID: r.ID, OccurrenceID: r.OccurrenceID, Name: r.Name, Mime: r.Mime, SizeBytes: r.SizeBytes,
		StoragePath: r.StoragePath, UploadedByUserID: r.UploadedByUserID, UploadedByName: r.UploadedByName, CreatedAt: r.CreatedAt}
}

func (s *Store) InsertAttachment(ctx context.Context, a domain.TaskAttachment) error {
	return s.q.InsertAttachment(ctx, sqlc.InsertAttachmentParams{ID: a.ID, OccurrenceID: a.OccurrenceID, Name: a.Name, Mime: a.Mime,
		SizeBytes: a.SizeBytes, StoragePath: a.StoragePath, UploadedByUserID: a.UploadedByUserID, UploadedByName: a.UploadedByName})
}

func (s *Store) ListAttachments(ctx context.Context, occurrenceID string) ([]domain.TaskAttachment, error) {
	rows, err := s.q.ListAttachments(ctx, occurrenceID)
	return each(rows, err, toAttachment)
}

func (s *Store) GetAttachment(ctx context.Context, id string) (*domain.TaskAttachment, error) {
	r, err := s.q.GetAttachment(ctx, id)
	return one(r, err, toAttachment)
}

func (s *Store) DeleteAttachment(ctx context.Context, id string) error {
	return s.q.DeleteAttachment(ctx, id)
}

func (s *Store) RecordContribution(ctx context.Context, occurrenceID, taskID string, p domain.Period) error {
	return s.q.RecordContribution(ctx, sqlc.RecordContributionParams{OccurrenceID: occurrenceID, TaskID: taskID,
		PeriodYear: int32(p.Year), PeriodMonth: int32(p.Month)})
}

func (s *Store) DropContribution(ctx context.Context, occurrenceID string) error {
	return s.q.DropContribution(ctx, occurrenceID)
}

func (s *Store) GetContribution(ctx context.Context, occurrenceID string) (*domain.TaskContribution, error) {
	r, err := s.q.GetContribution(ctx, occurrenceID)
	return one(r, err, func(r sqlc.PerformanceDepartmentTaskKpiContribution) domain.TaskContribution {
		return domain.TaskContribution{OccurrenceID: r.OccurrenceID, IndicatorID: r.IndicatorID, DepartmentID: r.DepartmentID,
			EmployeeID: r.EmployeeID, Period: domain.Period{Year: int(r.PeriodYear), Month: int(r.PeriodMonth)}, Amount: r.Amount, RecordedAt: r.RecordedAt}
	})
}

func (s *Store) ContributionTotals(ctx context.Context, departmentID string, span domain.DaySpan) (map[string]float64, error) {
	rows, err := s.q.ContributionTotals(ctx, sqlc.ContributionTotalsParams{DepartmentID: departmentID, StartDate: toDay(span.Start), EndDate: toDay(span.End)})
	if err != nil {
		return nil, err
	}
	out := map[string]float64{}
	for _, r := range rows {
		out[r.IndicatorID] = r.Total
	}
	return out, nil
}

func (s *Store) EmployeeContributions(ctx context.Context, employeeID string, span domain.DaySpan) ([]domain.ContributionSummary, error) {
	rows, err := s.q.EmployeeContributions(ctx, sqlc.EmployeeContributionsParams{EmployeeID: employeeID, StartDate: toDay(span.Start), EndDate: toDay(span.End)})
	return each(rows, err, func(r sqlc.EmployeeContributionsRow) domain.ContributionSummary {
		return domain.ContributionSummary{IndicatorID: r.IndicatorID, Code: r.Code, TasksTotal: int(r.TasksTotal), TasksDone: int(r.TasksDone),
			ContributionTotal: r.ContributionTotal}
	})
}

func (s *Store) DepartmentKPIIndicators(ctx context.Context, departmentID string) ([]domain.KPIIndicatorWeight, error) {
	rows, err := s.q.DepartmentKPIIndicators(ctx, departmentID)
	return each(rows, err, func(r sqlc.DepartmentKPIIndicatorsRow) domain.KPIIndicatorWeight {
		return domain.KPIIndicatorWeight{Weight: r.DepartmentWeight, Indicator: toIndicator(sqlc.PerformanceKpiIndicator{ID: r.ID, Code: r.Code,
			Name: r.Name, Description: r.Description, Unit: r.Unit, Direction: r.Direction, SourceKind: r.SourceKind,
			DefaultTarget: r.DefaultTarget, IsActive: r.IsActive, CreatedBy: r.CreatedBy, UpdatedBy: r.UpdatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})}
	})
}

func (s *Store) IndicatorDepartments(ctx context.Context) (map[string][]string, error) {
	rows, err := s.q.IndicatorDepartments(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string][]string{}
	for _, r := range rows {
		out[r.IndicatorID] = r.DepartmentIds
	}
	return out, nil
}

func (s *Store) ReminderLines(ctx context.Context, upTo string) ([]domain.ReminderLine, error) {
	rows, err := s.q.ReminderLines(ctx, toDay(upTo))
	return each(rows, err, func(r sqlc.ReminderLinesRow) domain.ReminderLine {
		return domain.ReminderLine{EmployeeID: r.EmployeeID, Title: r.Title, Date: day(r.OccurrenceDate), Priority: r.Priority}
	})
}

func (s *Store) RecapOccurrences(ctx context.Context, employeeID string, from, to time.Time) ([]domain.RecapOccurrence, error) {
	rows, err := s.q.RecapOccurrences(ctx, sqlc.RecapOccurrencesParams{EmployeeID: employeeID, FromAt: from, ToAt: to})
	return each(rows, err, func(r sqlc.RecapOccurrencesRow) domain.RecapOccurrence {
		o := domain.RecapOccurrence{OccurrenceID: r.OccurrenceID, TaskID: r.TaskID, Title: r.Title, Status: r.Status, Priority: r.Priority,
			ProgressPercent: intp(r.ProgressPercent), CompletedInDay: r.CompletedInDay}
		if r.SubtaskTotal > 0 {
			total, done := r.SubtaskTotal, r.SubtaskDone
			o.SubtaskTotal, o.SubtaskDone = &total, &done
		}
		return o
	})
}

func (s *Store) RecapNotes(ctx context.Context, employeeID string, from, to time.Time) ([]domain.RecapNote, error) {
	rows, err := s.q.RecapNotes(ctx, sqlc.RecapNotesParams{EmployeeID: employeeID, FromAt: from, ToAt: to})
	return each(rows, err, func(r sqlc.RecapNotesRow) domain.RecapNote {
		return domain.RecapNote{OccurrenceID: r.OccurrenceID, Note: r.Note}
	})
}

func (s *Store) MonthTasks(ctx context.Context, employeeIDs []string, span domain.DaySpan) (map[string][]domain.MonthTask, error) {
	rows, err := s.q.MonthTasks(ctx, sqlc.MonthTasksParams{EmployeeIds: append([]string{}, employeeIDs...), StartDate: toDay(span.Start), EndDate: toDay(span.End)})
	if err != nil {
		return nil, err
	}
	out := map[string][]domain.MonthTask{}
	for _, r := range rows {
		out[r.EmployeeID] = append(out[r.EmployeeID], domain.MonthTask{Title: r.Title, Date: day(r.OccurrenceDate), Status: r.Status,
			Priority: r.Priority, DoneAt: r.DoneAt, KPIIndicator: r.KpiCode, KPIContributed: r.Contributed})
	}
	return out, nil
}

// ─── Logbook ─────────────────────────────────────────────────

func toTemplate(r sqlc.PerformanceLogbookTemplate) domain.LogbookTemplate {
	return domain.LogbookTemplate{ID: r.ID, DepartmentID: r.DepartmentID, Name: r.Name, Description: r.Description, Frequency: r.Frequency,
		IsActive: r.IsActive, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Items: []domain.LogbookTemplateItem{}}
}

func (s *Store) withTemplateItems(ctx context.Context, templates []domain.LogbookTemplate) ([]domain.LogbookTemplate, error) {
	if len(templates) == 0 {
		return templates, nil
	}
	ids := make([]string, len(templates))
	index := map[string]int{}
	for i, t := range templates {
		ids[i] = t.ID
		index[t.ID] = i
	}
	items, err := s.q.ListTemplateItems(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		t := &templates[index[it.TemplateID]]
		t.Items = append(t.Items, domain.LogbookTemplateItem{ID: it.ID, TemplateID: it.TemplateID, Title: it.Title, Description: it.Description,
			Weight: it.Weight, IsRequired: it.IsRequired, SortOrder: int(it.SortOrder)})
	}
	return templates, nil
}

func (s *Store) ListTemplates(ctx context.Context, departmentID string, includeInactive bool) ([]domain.LogbookTemplate, error) {
	rows, err := s.q.ListTemplates(ctx, sqlc.ListTemplatesParams{DepartmentID: departmentID, IncludeInactive: includeInactive})
	out, err := each(rows, err, toTemplate)
	if err != nil {
		return nil, err
	}
	return s.withTemplateItems(ctx, out)
}

func (s *Store) GetTemplate(ctx context.Context, id string) (*domain.LogbookTemplate, error) {
	r, err := s.q.GetTemplate(ctx, id)
	t, err := one(r, err, toTemplate)
	if err != nil || t == nil {
		return t, err
	}
	withItems, err := s.withTemplateItems(ctx, []domain.LogbookTemplate{*t})
	if err != nil {
		return nil, err
	}
	return &withItems[0], nil
}

func (s *Store) InsertTemplate(ctx context.Context, t domain.LogbookTemplate) error {
	return s.q.InsertTemplate(ctx, sqlc.InsertTemplateParams{ID: t.ID, DepartmentID: t.DepartmentID, Name: t.Name, Description: t.Description,
		Frequency: t.Frequency, IsActive: t.IsActive, CreatedBy: t.CreatedBy})
}

func (s *Store) InsertTemplateItem(ctx context.Context, it domain.LogbookTemplateItem) error {
	return s.q.InsertTemplateItem(ctx, sqlc.InsertTemplateItemParams{ID: it.ID, TemplateID: it.TemplateID, Title: it.Title,
		Description: it.Description, Weight: it.Weight, IsRequired: it.IsRequired, SortOrder: int32(it.SortOrder)})
}

func (s *Store) CountTemplateEntries(ctx context.Context, templateID string) (int, error) {
	n, err := s.q.CountTemplateEntries(ctx, templateID)
	return int(n), err
}

func (s *Store) ArchiveTemplate(ctx context.Context, id string) error {
	return s.q.ArchiveTemplate(ctx, id)
}

func (s *Store) DeleteTemplate(ctx context.Context, id string) error {
	return s.q.DeleteTemplate(ctx, id)
}

func (s *Store) InsertEntry(ctx context.Context, e domain.LogbookEntry) error {
	return wrap(s.q.InsertEntry(ctx, sqlc.InsertEntryParams{ID: e.ID, TemplateID: e.TemplateID, DepartmentID: e.DepartmentID,
		EmployeeID: e.EmployeeID, EntryDate: toDay(e.EntryDate), Title: e.Title, Notes: e.Notes, CreatedBy: e.CreatedBy}))
}

func (s *Store) CopyTemplateItems(ctx context.Context, entryID, templateID string) error {
	return s.q.CopyTemplateItems(ctx, sqlc.CopyTemplateItemsParams{EntryID: entryID, TemplateID: templateID})
}

func toEntry(r sqlc.GetEntryRow) domain.LogbookEntry {
	return domain.LogbookEntry{ID: r.ID, TemplateID: r.TemplateID, TemplateName: r.TemplateName, Frequency: r.Frequency,
		DepartmentID: r.DepartmentID, EmployeeID: r.EmployeeID, EntryDate: day(r.EntryDate), Title: r.Title, Status: r.Status,
		Completion: r.CompletionPercentage, KPIScore: r.KpiScore, Notes: r.Notes, SubmittedBy: r.SubmittedBy, SubmittedAt: r.SubmittedAt,
		ReviewedBy: r.ReviewedBy, ReviewedAt: r.ReviewedAt, ReviewNotes: r.ReviewNotes, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt, Items: []domain.LogbookEntryItem{}}
}

func toEntryItem(r sqlc.PerformanceLogbookEntryItem) domain.LogbookEntryItem {
	return domain.LogbookEntryItem{ID: r.ID, EntryID: r.EntryID, TemplateItemID: r.TemplateItemID, Title: r.Title, Description: r.Description,
		Weight: r.Weight, IsRequired: r.IsRequired, SortOrder: int(r.SortOrder), IsChecked: r.IsChecked, CheckedBy: r.CheckedBy,
		CheckedAt: r.CheckedAt, Notes: r.Notes}
}

func (s *Store) withEntryItems(ctx context.Context, entries []domain.LogbookEntry) ([]domain.LogbookEntry, error) {
	if len(entries) == 0 {
		return entries, nil
	}
	ids := make([]string, len(entries))
	index := map[string]int{}
	for i, e := range entries {
		ids[i] = e.ID
		index[e.ID] = i
	}
	items, err := s.q.ListEntryItems(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		e := &entries[index[it.EntryID]]
		e.Items = append(e.Items, toEntryItem(it))
	}
	return entries, nil
}

func (s *Store) GetEntry(ctx context.Context, id string) (*domain.LogbookEntry, error) {
	r, err := s.q.GetEntry(ctx, id)
	e, err := one(r, err, toEntry)
	if err != nil || e == nil {
		return e, err
	}
	withItems, err := s.withEntryItems(ctx, []domain.LogbookEntry{*e})
	if err != nil {
		return nil, err
	}
	return &withItems[0], nil
}

func (s *Store) GetEntryItem(ctx context.Context, id string) (*domain.LogbookEntryItem, error) {
	r, err := s.q.GetEntryItem(ctx, id)
	return one(r, err, toEntryItem)
}

func (s *Store) ListEntries(ctx context.Context, f domain.EntryFilter) ([]domain.LogbookEntry, int, error) {
	total, err := s.q.CountEntries(ctx, sqlc.CountEntriesParams{DepartmentID: f.DepartmentID, EmployeeIds: filterIDs(f.EmployeeIDs),
		Status: f.Status, EntryDate: f.Date, FromDate: f.From, ToDate: f.To})
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.q.ListEntries(ctx, sqlc.ListEntriesParams{DepartmentID: f.DepartmentID, EmployeeIds: filterIDs(f.EmployeeIDs),
		Status: f.Status, EntryDate: f.Date, FromDate: f.From, ToDate: f.To, RowLimit: int32(f.Limit), RowOffset: int32(f.Offset)})
	entries, err := each(rows, err, func(r sqlc.ListEntriesRow) domain.LogbookEntry { return toEntry(sqlc.GetEntryRow(r)) })
	if err != nil {
		return nil, 0, err
	}
	entries, err = s.withEntryItems(ctx, entries)
	return entries, int(total), err
}

func (s *Store) UpdateEntryItem(ctx context.Context, it domain.LogbookEntryItem) error {
	return s.q.UpdateEntryItem(ctx, sqlc.UpdateEntryItemParams{ID: it.ID, IsChecked: it.IsChecked, CheckedBy: it.CheckedBy,
		CheckedAt: it.CheckedAt, Notes: it.Notes})
}

func (s *Store) UpdateEntry(ctx context.Context, e domain.LogbookEntry) error {
	return s.q.UpdateEntry(ctx, sqlc.UpdateEntryParams{ID: e.ID, Status: e.Status, Notes: e.Notes, SubmittedBy: e.SubmittedBy,
		SubmittedAt: e.SubmittedAt, ReviewedBy: e.ReviewedBy, ReviewedAt: e.ReviewedAt, ReviewNotes: e.ReviewNotes})
}

func (s *Store) SetEntryScore(ctx context.Context, id string, completion, kpiScore float64) error {
	return s.q.SetEntryScore(ctx, sqlc.SetEntryScoreParams{ID: id, CompletionPercentage: completion, KpiScore: kpiScore})
}

func (s *Store) DeleteEntry(ctx context.Context, id string) error { return s.q.DeleteEntry(ctx, id) }

func (s *Store) LogbookSummaryRows(ctx context.Context, departmentID, from, to string) ([]domain.LogbookSummaryEntry, error) {
	rows, err := s.q.LogbookSummaryRows(ctx, sqlc.LogbookSummaryRowsParams{DepartmentID: departmentID, FromDate: from, ToDate: to})
	return each(rows, err, func(r sqlc.LogbookSummaryRowsRow) domain.LogbookSummaryEntry {
		return domain.LogbookSummaryEntry{DepartmentID: r.DepartmentID, Status: r.Status, Completion: r.CompletionPercentage, KPIScore: r.KpiScore}
	})
}

func (s *Store) DailyTemplateDepartments(ctx context.Context) ([]string, error) {
	return s.q.DailyTemplateDepartments(ctx)
}

// ─── Recaps ──────────────────────────────────────────────────

func toDailyRecap(r sqlc.PerformanceDailyRecap) domain.DailyRecap {
	var items domain.DailyRecapItems
	_ = json.Unmarshal(r.Items, &items)
	return domain.DailyRecap{ID: r.ID, EmployeeID: r.EmployeeID, DepartmentID: r.DepartmentID, Date: day(r.RecapDate), Items: items,
		Notes: r.Notes, Summary: r.GeneratedSummary, SummaryModel: r.SummaryModel, GeneratedAt: r.GeneratedAt,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (s *Store) GetDailyRecap(ctx context.Context, employeeID, date string) (*domain.DailyRecap, error) {
	r, err := s.q.GetDailyRecap(ctx, sqlc.GetDailyRecapParams{EmployeeID: employeeID, RecapDate: toDay(date)})
	return one(r, err, toDailyRecap)
}

func (s *Store) UpsertDailyRecap(ctx context.Context, r domain.DailyRecap) error {
	return s.q.UpsertDailyRecap(ctx, sqlc.UpsertDailyRecapParams{ID: newID(), EmployeeID: r.EmployeeID, DepartmentID: r.DepartmentID,
		RecapDate: toDay(r.Date), Items: jsonOr(r.Items, "{}"), GeneratedSummary: r.Summary, SummaryModel: r.SummaryModel, GeneratedAt: r.GeneratedAt})
}

func (s *Store) SetDailyNotes(ctx context.Context, r domain.DailyRecap) error {
	return s.q.SetDailyNotes(ctx, sqlc.SetDailyNotesParams{ID: newID(), EmployeeID: r.EmployeeID, DepartmentID: r.DepartmentID,
		RecapDate: toDay(r.Date), Notes: r.Notes})
}

func (s *Store) ListDailyRecaps(ctx context.Context, f domain.DailyRecapFilter) ([]domain.DailyRecap, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.q.ListDailyRecaps(ctx, sqlc.ListDailyRecapsParams{RecapDate: f.Date, EmployeeIds: filterIDs(f.EmployeeIDs),
		FromDate: f.From, ToDate: f.To, RowLimit: int32(limit)})
	return each(rows, err, toDailyRecap)
}

func toMonthlyRecap(r sqlc.PerformanceMonthlyRecap) domain.MonthlyRecap {
	var m domain.MonthlyMetrics
	_ = json.Unmarshal(r.Metrics, &m)
	return domain.MonthlyRecap{ID: r.ID, Scope: r.Scope, SubjectID: r.SubjectID, DepartmentID: r.DepartmentID,
		Period: domain.Period{Year: int(r.PeriodYear), Month: int(r.PeriodMonth)}, Metrics: m, Accomplishments: r.Accomplishments,
		Narrative: r.Narrative, NarrativeModel: r.NarrativeModel, GeneratedAt: r.GeneratedAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (s *Store) UpsertMonthlyRecap(ctx context.Context, r domain.MonthlyRecap) (string, error) {
	id := r.ID
	if id == "" {
		id = newID()
	}
	return s.q.UpsertMonthlyRecap(ctx, sqlc.UpsertMonthlyRecapParams{ID: id, Scope: r.Scope, SubjectID: r.SubjectID, DepartmentID: r.DepartmentID,
		PeriodYear: int32(r.Period.Year), PeriodMonth: int32(r.Period.Month), Metrics: jsonOr(r.Metrics, "{}"),
		Accomplishments: append([]string{}, r.Accomplishments...), Narrative: r.Narrative, NarrativeModel: r.NarrativeModel})
}

func (s *Store) GetMonthlyRecap(ctx context.Context, id string) (*domain.MonthlyRecap, error) {
	r, err := s.q.GetMonthlyRecap(ctx, id)
	return one(r, err, toMonthlyRecap)
}

func (s *Store) ListMonthlyRecaps(ctx context.Context, f domain.MonthlyRecapFilter) ([]domain.MonthlyRecap, error) {
	p := sqlc.ListMonthlyRecapsParams{Scope: f.Scope, SubjectIds: filterIDs(f.SubjectIDs), DepartmentID: f.DepartmentID, RowLimit: int32(f.Limit)}
	if p.RowLimit <= 0 {
		p.RowLimit = 200
	}
	if f.Period != nil {
		y, m := int32(f.Period.Year), int32(f.Period.Month)
		p.PeriodYear, p.PeriodMonth = &y, &m
	}
	rows, err := s.q.ListMonthlyRecaps(ctx, p)
	return each(rows, err, toMonthlyRecap)
}

// ─── Improvements ────────────────────────────────────────────

func toImprovement(r sqlc.PerformanceImprovement) domain.Improvement {
	return domain.Improvement{ID: r.ID, Scope: r.Scope, EmployeeID: r.EmployeeID, DepartmentID: r.DepartmentID, Source: r.Source,
		SourceRef: r.SourceRef, Title: r.Title, Description: r.Description, OwnerEmployeeID: r.OwnerEmployeeID, DueDate: dayPtr(r.DueDate),
		Status: r.Status, LinkedTaskID: r.LinkedTaskID, CompletedAt: r.CompletedAt, CreatedBy: r.CreatedBy, UpdatedBy: r.UpdatedBy,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (s *Store) InsertImprovement(ctx context.Context, i domain.Improvement) (bool, error) {
	n, err := s.q.InsertImprovement(ctx, sqlc.InsertImprovementParams{ID: i.ID, Scope: i.Scope, EmployeeID: i.EmployeeID, DepartmentID: i.DepartmentID,
		Source: i.Source, SourceRef: i.SourceRef, Title: i.Title, Description: i.Description, OwnerEmployeeID: i.OwnerEmployeeID,
		DueDate: toDayPtr(i.DueDate), Status: i.Status, LinkedTaskID: i.LinkedTaskID, CreatedBy: i.CreatedBy})
	return n > 0, err
}

func (s *Store) GetImprovement(ctx context.Context, id string) (*domain.Improvement, error) {
	r, err := s.q.GetImprovement(ctx, id)
	return one(r, err, toImprovement)
}

func (s *Store) UpdateImprovement(ctx context.Context, i domain.Improvement) error {
	return s.q.UpdateImprovement(ctx, sqlc.UpdateImprovementParams{ID: i.ID, Title: i.Title, Description: i.Description,
		OwnerEmployeeID: i.OwnerEmployeeID, DueDate: toDayPtr(i.DueDate), Status: i.Status, LinkedTaskID: i.LinkedTaskID,
		CompletedAt: i.CompletedAt, UpdatedBy: i.UpdatedBy})
}

func (s *Store) SoftDeleteImprovement(ctx context.Context, id, by string) error {
	return s.q.SoftDeleteImprovement(ctx, sqlc.SoftDeleteImprovementParams{ID: id, UpdatedBy: &by})
}

func (s *Store) ListImprovements(ctx context.Context, f domain.ImprovementFilter) ([]domain.Improvement, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 300
	}
	rows, err := s.q.ListImprovements(ctx, sqlc.ListImprovementsParams{Scope: f.Scope, EmployeeIds: filterIDs(f.EmployeeIDs),
		DepartmentIds: filterIDs(f.DepartmentIDs), PersonID: f.PersonID, Statuses: filterIDs(f.Statuses), Source: f.Source, RowLimit: int32(limit)})
	return each(rows, err, toImprovement)
}

func (s *Store) OverdueImprovements(ctx context.Context, today string) ([]domain.Improvement, error) {
	rows, err := s.q.OverdueImprovements(ctx, toDay(today))
	return each(rows, err, toImprovement)
}

func toRetro(r sqlc.PerformanceRetroSession) domain.RetroSession {
	actions := []domain.RetroAction{}
	_ = json.Unmarshal(r.Actions, &actions)
	return domain.RetroSession{ID: r.ID, DepartmentID: r.DepartmentID, Period: domain.Period{Year: int(r.PeriodYear), Month: int(r.PeriodMonth)},
		WentWell: r.WentWell, ToImprove: r.ToImprove, Actions: actions, Status: r.Status, CreatedBy: r.CreatedBy, UpdatedBy: r.UpdatedBy,
		ClosedAt: r.ClosedAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (s *Store) ListRetros(ctx context.Context, departmentIDs []string) ([]domain.RetroSession, error) {
	rows, err := s.q.ListRetros(ctx, filterIDs(departmentIDs))
	return each(rows, err, toRetro)
}

func (s *Store) GetRetro(ctx context.Context, id string) (*domain.RetroSession, error) {
	r, err := s.q.GetRetro(ctx, id)
	return one(r, err, toRetro)
}

func (s *Store) GetRetroFor(ctx context.Context, departmentID string, p domain.Period) (*domain.RetroSession, error) {
	r, err := s.q.GetRetroFor(ctx, sqlc.GetRetroForParams{DepartmentID: departmentID, PeriodYear: int32(p.Year), PeriodMonth: int32(p.Month)})
	return one(r, err, toRetro)
}

func (s *Store) InsertRetro(ctx context.Context, r domain.RetroSession) error {
	return wrap(s.q.InsertRetro(ctx, sqlc.InsertRetroParams{ID: r.ID, DepartmentID: r.DepartmentID, PeriodYear: int32(r.Period.Year),
		PeriodMonth: int32(r.Period.Month), WentWell: append([]string{}, r.WentWell...), ToImprove: append([]string{}, r.ToImprove...),
		Actions: jsonOr(r.Actions, "[]"), Status: r.Status, CreatedBy: r.CreatedBy}))
}

func (s *Store) UpdateRetro(ctx context.Context, r domain.RetroSession) error {
	return s.q.UpdateRetro(ctx, sqlc.UpdateRetroParams{ID: r.ID, WentWell: append([]string{}, r.WentWell...), ToImprove: append([]string{}, r.ToImprove...),
		Actions: jsonOr(r.Actions, "[]"), Status: r.Status, ClosedAt: r.ClosedAt, UpdatedBy: r.UpdatedBy})
}
