package app

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/salesevents"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Pipeline covers pipelines, stages, deals, deal members and activities
// (salesfunnel pipelines.go, deals.go, activities.go).
type Pipeline struct {
	*Deps
	auto       *Automation
	onboarding *Onboarding
}

func NewPipeline(d *Deps, auto *Automation, onboarding *Onboarding) *Pipeline {
	return &Pipeline{Deps: d, auto: auto, onboarding: onboarding}
}

func (p *Pipeline) ListPipelines(ctx context.Context) ([]domain.Pipeline, error) {
	return p.Store.ListPipelines(ctx)
}

func (p *Pipeline) SavePipeline(ctx context.Context, pl domain.Pipeline) (*domain.Pipeline, error) {
	errs := map[string]string{}
	if domain.JSTrim(pl.Name) == "" {
		errs["name"] = "Nama pipeline wajib diisi"
	}
	if !domain.FormSlugRe.MatchString(pl.Code) {
		errs["code"] = "Kode hanya huruf kecil, angka, dan tanda hubung"
	}
	if err := fieldErrs(errs); err != nil {
		return nil, err
	}
	if pl.ID == "" {
		pl.ID = ids.New()
	}
	if err := p.Store.SavePipeline(ctx, pl); err != nil {
		var u *domain.ErrUnique
		if asUnique(err, &u) {
			return nil, conflict("Kode pipeline sudah dipakai")
		}
		return nil, err
	}
	return &pl, nil
}

func (p *Pipeline) SaveStage(ctx context.Context, st domain.Stage) (*domain.Stage, error) {
	errs := map[string]string{}
	if domain.JSTrim(st.Name) == "" {
		errs["name"] = "Nama tahap wajib diisi"
	}
	if !domain.FormSlugRe.MatchString(st.Code) {
		errs["code"] = "Kode hanya huruf kecil, angka, dan tanda hubung"
	}
	if st.Probability < 0 || st.Probability > 100 {
		errs["probability"] = "Probabilitas harus 0-100"
	}
	if st.IsWon && st.IsLost {
		errs["isWon"] = "Tahap tidak bisa menang dan kalah sekaligus"
	}
	if st.PipelineID == "" {
		errs["pipelineId"] = "Pipeline wajib dipilih"
	}
	if err := fieldErrs(errs); err != nil {
		return nil, err
	}
	if st.ID == "" {
		st.ID = ids.New()
	}
	if err := p.Store.SaveStage(ctx, st); err != nil {
		var u *domain.ErrUnique
		if asUnique(err, &u) {
			return nil, conflict("Kode tahap sudah dipakai di pipeline ini")
		}
		return nil, err
	}
	return &st, nil
}

func (p *Pipeline) ListLostReasons(ctx context.Context) ([]domain.LostReason, error) {
	return p.Store.ListLostReasons(ctx)
}

// defaultStage is the first open stage of the given (or default) pipeline.
func (p *Pipeline) defaultStage(ctx context.Context, pipelineID string) (*domain.Stage, error) {
	pls, err := p.Store.ListPipelines(ctx)
	if err != nil {
		return nil, err
	}
	for _, pl := range pls {
		if (pipelineID == "" && pl.IsDefault) || pl.ID == pipelineID {
			for _, st := range pl.Stages {
				if st.IsActive && !st.IsWon && !st.IsLost {
					s := st
					return &s, nil
				}
			}
		}
	}
	return nil, invalid("Pipeline belum punya tahap aktif")
}

// DealInput is the create/update payload.
type DealInput struct {
	ID, LeadID, AccountID, Title, PipelineID, StageID string
	ServiceLines                                      []string
	ValueEstimate                                     string
	OwnerUserID, ExpectedCloseDate, Notes             string
	ForecastCategory                                  string
	Custom                                            json.RawMessage
}

func (p *Pipeline) applyDeal(ctx context.Context, in DealInput, d *domain.Deal) error {
	errs := map[string]string{}
	if domain.JSTrim(in.Title) == "" {
		errs["title"] = "Judul deal wajib diisi"
	}
	lines := []string{}
	for _, l := range in.ServiceLines {
		if !domain.Contains(domain.ServiceLines, l) {
			errs["service_lines"] = "Layanan tidak dikenal: " + l
		} else if !domain.Contains(lines, l) {
			lines = append(lines, l)
		}
	}
	if len(lines) == 0 {
		errs["service_lines"] = "Pilih minimal 1 layanan"
	}
	var estimate *domain.Money
	if in.ValueEstimate != "" {
		v, err := domain.ParseMoney(in.ValueEstimate)
		if err != nil || v < 0 {
			errs["value_estimate"] = "Nilai estimasi tidak valid"
		} else {
			estimate = &v
		}
	}
	closeDate, err := domain.ParseDate(in.ExpectedCloseDate)
	if err != nil {
		errs["expected_close_date"] = "Tanggal tidak valid"
	}
	if in.ForecastCategory != "" && !domain.Contains(domain.ForecastCategories, in.ForecastCategory) {
		errs["forecast_category"] = "Kategori forecast tidak valid"
	}
	if err := fieldErrs(errs); err != nil {
		return err
	}
	if in.LeadID != "" {
		lead, err := p.Store.GetLead(ctx, in.LeadID)
		if err != nil {
			return err
		}
		if lead == nil {
			return notFound(msgLeadNotFound)
		}
		if in.AccountID == "" && lead.AccountID != nil {
			in.AccountID = *lead.AccountID
		}
	}
	if in.AccountID != "" {
		if a, err := p.Store.GetAccount(ctx, in.AccountID); err != nil {
			return err
		} else if a == nil {
			return notFound(msgAccountNotFound)
		}
	}
	custom := in.Custom
	if len(custom) == 0 || string(custom) == "null" {
		custom = json.RawMessage("{}")
	}
	d.LeadID, d.AccountID, d.Title, d.ServiceLines = domain.TrimmedOrNil(in.LeadID), domain.TrimmedOrNil(in.AccountID), domain.JSTrim(in.Title), lines
	d.ValueEstimate, d.ExpectedCloseDate, d.Notes, d.Custom = estimate, closeDate, domain.TrimmedOrNil(in.Notes), custom
	if in.OwnerUserID != "" {
		d.OwnerUserID = &in.OwnerUserID
	}
	return nil
}

func (p *Pipeline) CreateDeal(ctx context.Context, actor Actor, in DealInput) (*domain.Deal, error) {
	now := p.now()
	d := &domain.Deal{ID: ids.New(), EnteredStageAt: now, Audit: domain.Audit{CreatedBy: actor.ref(), UpdatedBy: actor.ref(), CreatedAt: now, UpdatedAt: now}}
	if err := p.applyDeal(ctx, in, d); err != nil {
		return nil, err
	}
	var stage *domain.Stage
	var err error
	if in.StageID != "" {
		if stage, err = p.Store.GetStage(ctx, in.StageID); err != nil {
			return nil, err
		}
		if stage == nil {
			return nil, notFound(msgStageNotFound)
		}
		if stage.IsWon || stage.IsLost {
			return nil, invalid("Deal baru tidak bisa langsung di tahap menang atau kalah")
		}
	} else if stage, err = p.defaultStage(ctx, in.PipelineID); err != nil {
		return nil, err
	}
	d.PipelineID, d.StageID = stage.PipelineID, stage.ID
	d.ForecastCategory = domain.CategoryFromStage(stage.Flags())
	if in.ForecastCategory != "" {
		d.ForecastCategory = in.ForecastCategory
	}
	if d.OwnerUserID == nil {
		d.OwnerUserID = actor.ref()
	}
	err = p.Store.WithTx(ctx, func(st domain.Store) error {
		if err := st.SaveDeal(ctx, d); err != nil {
			return err
		}
		return st.AddStageHistory(ctx, domain.StageHistory{ID: ids.New(), DealID: d.ID, StageID: stage.ID, EnteredAt: now, MovedBy: actor.ref()})
	})
	if err != nil {
		return nil, err
	}
	p.auto.Emit(ctx, Event{Type: "deal.created", SubjectType: "deal", SubjectID: d.ID, Actor: actor.ref()})
	p.auto.Signal(ctx, deref(d.LeadID), SignalDealCreated, "")
	return p.Store.GetDeal(ctx, d.ID)
}

func (p *Pipeline) UpdateDeal(ctx context.Context, actor Actor, in DealInput) (*domain.Deal, error) {
	d, err := p.Store.GetDeal(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, notFound(msgDealNotFound)
	}
	if err := p.applyDeal(ctx, in, d); err != nil {
		return nil, err
	}
	if !d.IsWon && !d.IsLost {
		d.ForecastCategory = domain.CategoryFromStage(domain.StageFlags{Probability: d.Probability})
		if in.ForecastCategory != "" {
			d.ForecastCategory = in.ForecastCategory
		}
	}
	d.UpdatedBy, d.UpdatedAt = actor.ref(), p.now()
	if err := p.Store.SaveDeal(ctx, d); err != nil {
		return nil, err
	}
	p.auto.Emit(ctx, Event{Type: "deal.updated", SubjectType: "deal", SubjectID: d.ID, Actor: actor.ref()})
	return p.Store.GetDeal(ctx, d.ID)
}

func (p *Pipeline) DeleteDeal(ctx context.Context, id string) error {
	d, err := p.Store.GetDeal(ctx, id)
	if err != nil {
		return err
	}
	if d == nil {
		return notFound(msgDealNotFound)
	}
	return p.Store.DeleteDeal(ctx, id, p.now())
}

// MoveDealStage is planStageMove plus the side effects: stage history,
// workflow events, and on a won stage the deal.won event and onboarding.
func (p *Pipeline) MoveDealStage(ctx context.Context, actor Actor, dealID, stageID, valueFinal, lostReasonID string) (*domain.Deal, error) {
	d, err := p.Store.GetDeal(ctx, dealID)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, notFound(msgDealNotFound)
	}
	stage, err := p.Store.GetStage(ctx, stageID)
	if err != nil {
		return nil, err
	}
	if stage == nil || stage.PipelineID != d.PipelineID {
		return nil, notFound(msgStageNotFound)
	}
	var final *domain.Money
	if valueFinal != "" {
		v, err := domain.ParseMoney(valueFinal)
		if err != nil || v < 0 {
			return nil, invalid("Nilai final tidak valid")
		}
		final = &v
	}
	if lostReasonID != "" {
		reasons, err := p.Store.ListLostReasons(ctx)
		if err != nil {
			return nil, err
		}
		found := false
		for _, r := range reasons {
			found = found || r.ID == lostReasonID
		}
		if !found {
			return nil, invalid("Alasan kalah tidak dikenal")
		}
	}
	now := p.now()
	override := ""
	if domain.Contains(domain.ForecastCategories, d.ForecastCategory) && d.StageID == stageID {
		override = d.ForecastCategory
	}
	mv, err := domain.PlanStageMove(stage.Flags(), d.ValueFinal, final, domain.TrimmedOrNil(lostReasonID), override, now)
	if err != nil {
		return nil, invalid(err.Error())
	}
	fromStage, fromName := d.StageID, d.StageName
	changed := d.StageID != stage.ID
	d.StageID, d.ForecastCategory, d.ClosedAt, d.LostReasonID, d.ValueFinal = stage.ID, mv.ForecastCategory, mv.ClosedAt, mv.LostReasonID, mv.ValueFinal
	if changed {
		d.EnteredStageAt = now
	}
	d.UpdatedBy, d.UpdatedAt = actor.ref(), now
	err = p.Store.WithTx(ctx, func(st domain.Store) error {
		if err := st.SaveDeal(ctx, d); err != nil {
			return err
		}
		if !changed {
			return nil
		}
		return st.AddStageHistory(ctx, domain.StageHistory{ID: ids.New(), DealID: d.ID, StageID: stage.ID, EnteredAt: now, MovedBy: actor.ref()})
	})
	if err != nil {
		return nil, err
	}
	if changed {
		p.auto.Emit(ctx, Event{Type: "deal.stage_changed", SubjectType: "deal", SubjectID: d.ID, Actor: actor.ref(),
			Changes: map[string]domain.Change{"stage_id": {From: fromStage, To: stage.ID}, "stage_code": {From: "", To: stage.Code}},
			Payload: map[string]any{"from_stage_id": fromStage, "from_stage_name": fromName, "to_stage_id": stage.ID, "to_stage_name": stage.Name}})
	}
	if stage.IsWon && changed {
		if err := p.won(ctx, actor, dealID); err != nil {
			p.warn("deal won side effects failed", "deal", dealID, "err", err)
		}
	}
	return p.Store.GetDeal(ctx, d.ID)
}

// won publishes white.sales.deal.won and starts client onboarding.
func (p *Pipeline) won(ctx context.Context, actor Actor, dealID string) error {
	d, err := p.Store.GetDeal(ctx, dealID)
	if err != nil || d == nil {
		return err
	}
	value := ""
	if d.ValueFinal != nil {
		value = d.ValueFinal.String()
	}
	wonAt := p.now()
	if d.ClosedAt != nil {
		wonAt = *d.ClosedAt
	}
	if err := p.publish(ctx, events.SalesDealWon, salesevents.DealWon{DealID: d.ID, Title: d.Title, LeadID: deref(d.LeadID), AccountID: deref(d.AccountID), ClientName: d.ClientName(),
		PicName: d.PicName, PicEmail: d.PicEmail, PicPhone: d.PicPhone, ValueFinal: value, ServiceLines: d.ServiceLines, OwnerUserID: deref(d.OwnerUserID), WonAt: wonAt}); err != nil {
		p.warn("publish deal won failed", "err", err)
	}
	_, err = p.onboarding.Start(ctx, actor, dealID)
	return err
}

func (p *Pipeline) ListDeals(ctx context.Context, f domain.DealFilter) ([]domain.Deal, int, error) {
	return p.Store.ListDeals(ctx, f)
}

// DealDetail is GET /deals/{id} with everything the deal page shows.
type DealDetail struct {
	Deal           domain.Deal
	Members        []domain.DealMember
	History        []domain.StageHistory
	Quotations     []domain.Quotation
	Invoices       []domain.Invoice
	Payments       []domain.Payment
	TermProgress   []domain.TermProgress
	ReferenceTotal domain.Money
	PaidTotal      domain.Money
	PaymentStatus  string
	Contracts      []domain.Contract
	Onboarding     *domain.Onboarding
}

func (p *Pipeline) GetDeal(ctx context.Context, id string) (*DealDetail, error) {
	d, err := p.Store.GetDeal(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, notFound(msgDealNotFound)
	}
	out := &DealDetail{Deal: *d}
	if out.Members, err = p.Store.ListDealMembers(ctx, id); err != nil {
		return nil, err
	}
	if out.History, err = p.Store.ListStageHistory(ctx, id); err != nil {
		return nil, err
	}
	if out.Quotations, err = p.Store.ListQuotations(ctx, domain.QuotationFilter{DealID: id, Limit: 100}); err != nil {
		return nil, err
	}
	if out.Invoices, err = p.Store.ListInvoices(ctx, domain.InvoiceFilter{DealID: id, Limit: 100}); err != nil {
		return nil, err
	}
	if out.Payments, err = p.Store.ListPayments(ctx, id); err != nil {
		return nil, err
	}
	if out.PaidTotal, err = p.Store.PaidTotal(ctx, id); err != nil {
		return nil, err
	}
	if out.Contracts, err = p.Store.ListContracts(ctx, domain.ContractFilter{DealID: id, Limit: 50}); err != nil {
		return nil, err
	}
	if out.Onboarding, err = p.Store.GetOnboardingByDeal(ctx, id); err != nil {
		return nil, err
	}
	latest, err := p.Store.LatestQuotation(ctx, id)
	if err != nil {
		return nil, err
	}
	out.ReferenceTotal = domain.ReferenceTotal(latest, d.ValueFinal, d.ValueEstimate)
	if latest != nil {
		terms := make([]domain.TermInput, len(latest.Terms))
		for i, t := range latest.Terms {
			terms[i] = domain.TermInput{ID: t.ID, Label: t.Label, DueDate: t.DueDate, Percent: t.Percent}
		}
		out.TermProgress = domain.TermProgressOf(terms, out.ReferenceTotal, out.PaidTotal)
	}
	out.PaymentStatus = domain.PaymentStatus(out.PaidTotal, out.ReferenceTotal)
	return out, nil
}

// SetMembers replaces the deal team (deal members with revenue split).
func (p *Pipeline) SetMembers(ctx context.Context, dealID string, members []domain.DealMember) ([]domain.DealMember, error) {
	d, err := p.Store.GetDeal(ctx, dealID)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, notFound(msgDealNotFound)
	}
	seen := map[string]bool{}
	splits := make([]float64, 0, len(members))
	for i := range members {
		m := &members[i]
		m.DealID = dealID
		if m.Role == "" {
			m.Role = "support"
		}
		switch {
		case strings.TrimSpace(m.UserID) == "":
			return nil, invalid("Anggota tim wajib dipilih")
		case seen[m.UserID]:
			return nil, invalid("Anggota tim duplikat")
		case !domain.Contains(domain.DealMemberRoles, m.Role):
			return nil, invalid("Peran anggota tidak valid")
		}
		seen[m.UserID] = true
		splits = append(splits, m.SplitPercent)
	}
	if !domain.SplitTotalOK(splits) {
		return nil, invalid("Total pembagian komisi maksimal 100%")
	}
	if err := p.Store.WithTx(ctx, func(st domain.Store) error { return st.ReplaceDealMembers(ctx, dealID, members) }); err != nil {
		return nil, err
	}
	return p.Store.ListDealMembers(ctx, dealID)
}

// ─── activities ──────────────────────────────────────────────

// ActivityInput is the create/update payload of a task or activity log.
type ActivityInput struct {
	ID, LeadID, DealID, SubjectType, SubjectID string
	ActivityType, Title, Notes                 string
	Priority, Status                           string
	DueAt, ReminderAt                          *time.Time
	Recurrence                                 *domain.Recurrence
	OwnerUserID                                string
}

func (p *Pipeline) applyActivity(in ActivityInput, a *domain.Activity, creating bool) error {
	errs := map[string]string{}
	if domain.JSTrim(in.Title) == "" {
		errs["title"] = "Judul wajib diisi"
	}
	if !domain.Contains(domain.ActivityTypes, in.ActivityType) {
		errs["activity_type"] = "Jenis aktivitas tidak valid"
	}
	priority := in.Priority
	if priority == "" {
		priority = "normal"
	} else if !domain.Contains(domain.TaskPriorities, priority) {
		errs["priority"] = "Prioritas tidak valid"
	}
	status := in.Status
	if status == "" {
		status = a.Status
		if status == "" {
			status = "open"
		}
	} else if !domain.Contains(domain.TaskStatuses, status) {
		errs["status"] = "Status tidak valid"
	}
	if in.Recurrence != nil {
		raw, _ := json.Marshal(in.Recurrence)
		if _, ok := domain.ParseRecurrence(raw); !ok {
			errs["recurrence"] = "Pengulangan tidak valid"
		} else if in.DueAt == nil {
			errs["recurrence"] = "Tugas berulang wajib punya tenggat"
		}
	}
	if in.ReminderAt != nil && in.DueAt != nil && in.ReminderAt.After(*in.DueAt) {
		errs["reminder_at"] = "Pengingat harus sebelum tenggat"
	}
	if err := fieldErrs(errs); err != nil {
		return err
	}
	if creating {
		subjectType, subjectID, ok := domain.ResolveTaskSubject(in.SubjectType, in.SubjectID, in.DealID, in.LeadID)
		if !ok {
			return invalid("Pilih lead, deal, akun, atau kontak terkait")
		}
		if !domain.Contains(domain.TaskSubjectTypes, subjectType) {
			return invalid("Subjek tidak valid")
		}
		a.SubjectType, a.SubjectID = &subjectType, &subjectID
		a.LeadID, a.DealID = domain.TrimmedOrNil(in.LeadID), domain.TrimmedOrNil(in.DealID)
	}
	if a.ReminderAt == nil || in.ReminderAt == nil || !a.ReminderAt.Equal(*in.ReminderAt) {
		a.ReminderSentAt = nil
	}
	a.ActivityType, a.Title, a.Notes, a.Priority, a.Status = in.ActivityType, domain.SliceUTF16(domain.JSTrim(in.Title), 200), domain.TrimmedOrNil(in.Notes), priority, status
	a.DueAt, a.ReminderAt = in.DueAt, in.ReminderAt
	a.Recurrence = nil
	if in.Recurrence != nil {
		a.Recurrence, _ = json.Marshal(in.Recurrence)
	}
	if in.OwnerUserID != "" {
		a.OwnerUserID = &in.OwnerUserID
	}
	return nil
}

func (p *Pipeline) CreateActivity(ctx context.Context, actor Actor, in ActivityInput) (*domain.Activity, error) {
	now := p.now()
	a := &domain.Activity{ID: ids.New(), Audit: domain.Audit{CreatedBy: actor.ref(), UpdatedBy: actor.ref(), CreatedAt: now, UpdatedAt: now}}
	if err := p.applyActivity(in, a, true); err != nil {
		return nil, err
	}
	if a.OwnerUserID == nil {
		a.OwnerUserID = actor.ref()
	}
	if a.Status == "done" {
		a.DoneAt = &now
	}
	if err := p.Store.SaveActivity(ctx, a); err != nil {
		return nil, err
	}
	p.auto.Emit(ctx, Event{Type: "task.created", SubjectType: "task", SubjectID: a.ID, Actor: actor.ref()})
	return a, nil
}

// UpdateActivity applies the edit; finishing a recurring task spawns the
// next occurrence (spawnNextTask).
func (p *Pipeline) UpdateActivity(ctx context.Context, actor Actor, in ActivityInput) (*domain.Activity, error) {
	a, err := p.Store.GetActivity(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, notFound(msgActivityNotFound)
	}
	wasOpen := domain.IsTaskOpen(a.Status)
	if err := p.applyActivity(in, a, false); err != nil {
		return nil, err
	}
	now := p.now()
	finished := wasOpen && a.Status == "done"
	switch {
	case finished:
		a.DoneAt = &now
	case a.Status != "done":
		a.DoneAt = nil
	}
	a.UpdatedBy, a.UpdatedAt = actor.ref(), now
	var next *domain.Activity
	if finished && a.DueAt != nil {
		if r, ok := domain.ParseRecurrence(a.Recurrence); ok {
			if due, rem, ok := domain.SpawnNextTask(*a.DueAt, a.ReminderAt, r); ok {
				n := *a
				n.ID, n.Status, n.DoneAt, n.DueAt, n.ReminderAt, n.ReminderSentAt = ids.New(), "open", nil, &due, rem, nil
				parent := a.ID
				if a.ParentTaskID != nil {
					parent = *a.ParentTaskID
				}
				n.ParentTaskID = &parent
				n.CreatedAt, n.UpdatedAt = now, now
				next = &n
			}
		}
	}
	err = p.Store.WithTx(ctx, func(st domain.Store) error {
		if err := st.SaveActivity(ctx, a); err != nil {
			return err
		}
		if next != nil {
			return st.SaveActivity(ctx, next)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	p.auto.Emit(ctx, Event{Type: "task.updated", SubjectType: "task", SubjectID: a.ID, Actor: actor.ref()})
	if finished {
		p.auto.Emit(ctx, Event{Type: "task.done", SubjectType: "task", SubjectID: a.ID, Actor: actor.ref()})
		lead := deref(a.LeadID)
		if lead == "" && a.DealID != nil {
			if d, err := p.Store.GetDeal(ctx, *a.DealID); err == nil && d != nil {
				lead = deref(d.LeadID)
			}
		}
		p.auto.Signal(ctx, lead, SignalTaskDone, a.ActivityType)
	}
	return a, nil
}

func (p *Pipeline) DeleteActivity(ctx context.Context, id string) error {
	a, err := p.Store.GetActivity(ctx, id)
	if err != nil {
		return err
	}
	if a == nil {
		return notFound(msgActivityNotFound)
	}
	return p.Store.DeleteActivity(ctx, id, p.now())
}

func (p *Pipeline) ListActivities(ctx context.Context, f domain.ActivityFilter) ([]domain.Activity, error) {
	return p.Store.ListActivities(ctx, f)
}

// SendDueReminders notifies owners of tasks whose reminder time passed
// (followup-reminder-watcher), at most once per task by claiming first.
func (p *Pipeline) SendDueReminders(ctx context.Context) (int, error) {
	due, err := p.Store.DueReminders(ctx, p.now(), 100)
	if err != nil {
		return 0, err
	}
	sent := 0
	for _, a := range due {
		ok, err := p.Store.ClaimReminder(ctx, a.ID, p.now())
		if err != nil {
			return sent, err
		}
		if !ok || a.OwnerUserID == nil {
			continue
		}
		body := "Tugas akan jatuh tempo."
		if a.DueAt != nil {
			body = "Tenggat " + a.DueAt.In(domain.Jakarta).Format("02/01/2006 15:04") + " WIB"
		}
		p.notify(ctx, Notification{UserIDs: []string{*a.OwnerUserID}, Type: NotifTaskReminder, Title: "Pengingat: " + a.Title, Body: body, Href: "/erp/sales/tasks?task=" + a.ID})
		sent++
	}
	return sent, nil
}
