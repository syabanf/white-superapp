package app

import (
	"context"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/payroll"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
)

// ─── assets ──────────────────────────────────────────────────

// Assets tracks company equipment and who holds it. hr.read lists, hr.write
// changes; an employee sees what they hold through self service.
type Assets struct{ d Deps }

func NewAssets(d Deps) *Assets { return &Assets{d: d} }

var assetCategories = []string{"laptop", "phone", "vehicle", "furniture", "other"}

const msgAssetNotFound = "Aset tidak ditemukan."

func (s *Assets) List(ctx context.Context, a Actor, f domain.AssetFilter) ([]domain.Asset, error) {
	if !a.Can(authn.PermHRRead) {
		id, err := self(a)
		if err != nil {
			return nil, err
		}
		f = domain.AssetFilter{EmployeeID: id}
	}
	return s.d.Store.ListAssets(ctx, f)
}

func (s *Assets) Get(ctx context.Context, a Actor, id string) (*domain.Asset, []domain.AssetAssignment, error) {
	if err := require(a, authn.PermHRRead); err != nil {
		return nil, nil, err
	}
	asset, err := s.d.Store.GetAsset(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if asset == nil {
		return nil, nil, notFound(msgAssetNotFound)
	}
	history, err := s.d.Store.ListAssetAssignments(ctx, id)
	return asset, history, err
}

type AssetInput struct {
	ID, AssetCode, Name, Category, SerialNo, PurchaseDate, PurchaseValue, Notes string
	// Retired or Repair; empty keeps the current state.
	Status string
}

// Save creates or edits an asset. Lending and returning go through Assign
// and Return so the history stays complete.
func (s *Assets) Save(ctx context.Context, a Actor, in AssetInput) (*domain.Asset, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	in.AssetCode, in.Name = strings.TrimSpace(in.AssetCode), strings.TrimSpace(in.Name)
	if in.AssetCode == "" || in.Name == "" {
		return nil, invalid("name", "Kode dan nama aset wajib diisi.")
	}
	if !slices.Contains(assetCategories, in.Category) {
		return nil, invalid("category", "Kategori aset tidak valid.")
	}
	purchased, err := optDate("purchaseDate", in.PurchaseDate)
	if err != nil {
		return nil, err
	}
	value := 0.0
	if strings.TrimSpace(in.PurchaseValue) != "" {
		if value, err = amount("purchaseValue", in.PurchaseValue); err != nil {
			return nil, err
		}
	}
	asset := domain.Asset{ID: in.ID, Status: "available", CreatedBy: a.UserRef()}
	if in.ID != "" {
		cur, err := s.d.Store.GetAsset(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		if cur == nil {
			return nil, notFound(msgAssetNotFound)
		}
		asset = *cur
	} else {
		asset.ID = newID()
	}
	switch in.Status {
	case "":
	case "repair", "retired", "available":
		if asset.Status == "assigned" {
			return nil, conflict("Aset masih dipinjam. Catat pengembaliannya dulu.")
		}
		asset.Status = in.Status
	default:
		return nil, invalid("status", "Status aset tidak valid.")
	}
	asset.AssetCode, asset.Name, asset.Category, asset.SerialNo = in.AssetCode, in.Name, in.Category, strings.TrimSpace(in.SerialNo)
	asset.PurchaseDate, asset.PurchaseValue, asset.Notes = purchased, value, strings.TrimSpace(in.Notes)
	if err := s.d.Store.SaveAsset(ctx, asset); err != nil {
		return nil, uniqueErr(err, "Kode aset sudah dipakai.")
	}
	return s.d.Store.GetAsset(ctx, asset.ID)
}

// Assign lends an available asset to an active employee.
func (s *Assets) Assign(ctx context.Context, a Actor, assetID, employeeID, on, condition string) (*domain.Asset, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	day, err := reqDate("assignedOn", on)
	if err != nil {
		return nil, err
	}
	emp, err := s.d.Store.GetEmployee(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if emp == nil || !emp.IsActive {
		return nil, invalid("employeeId", "Pilih karyawan yang masih aktif.")
	}
	err = s.d.Store.WithTx(ctx, func(tx domain.Store) error {
		asset, err := tx.GetAsset(ctx, assetID)
		if err != nil {
			return err
		}
		if asset == nil {
			return notFound(msgAssetNotFound)
		}
		if asset.Status != "available" {
			return conflict("Hanya aset berstatus tersedia yang bisa dipinjamkan.")
		}
		asset.Status, asset.EmployeeID = "assigned", &emp.ID
		if err := tx.SaveAsset(ctx, *asset); err != nil {
			return err
		}
		return tx.InsertAssetAssignment(ctx, domain.AssetAssignment{ID: newID(), AssetID: assetID, EmployeeID: emp.ID, AssignedOn: day,
			ConditionOut: strings.TrimSpace(condition), CreatedBy: a.UserRef()})
	})
	if err != nil {
		return nil, err
	}
	asset, err := s.d.Store.GetAsset(ctx, assetID)
	if err == nil && asset != nil {
		notifyEmployee(ctx, s.d, emp.ID, NotifyInput{Type: "HR_ASSET_ASSIGNED", Title: "Anda menerima aset " + asset.Name, Body: asset.AssetCode, Href: "/erp/me"})
	}
	return asset, err
}

// Return closes the open assignment and makes the asset available again.
func (s *Assets) Return(ctx context.Context, a Actor, assetID, on, condition string) (*domain.Asset, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	day, err := reqDate("returnedOn", on)
	if err != nil {
		return nil, err
	}
	err = s.d.Store.WithTx(ctx, func(tx domain.Store) error {
		asset, err := tx.GetAsset(ctx, assetID)
		if err != nil {
			return err
		}
		if asset == nil {
			return notFound(msgAssetNotFound)
		}
		if asset.Status != "assigned" {
			return conflict("Aset ini tidak sedang dipinjam.")
		}
		if _, err := tx.CloseAssetAssignment(ctx, assetID, day, strings.TrimSpace(condition)); err != nil {
			return err
		}
		asset.Status, asset.EmployeeID = "available", nil
		return tx.SaveAsset(ctx, *asset)
	})
	if err != nil {
		return nil, err
	}
	return s.d.Store.GetAsset(ctx, assetID)
}

// ─── training ────────────────────────────────────────────────

// Trainings records courses, who joined and how they did.
type Trainings struct{ d Deps }

func NewTrainings(d Deps) *Trainings { return &Trainings{d: d} }

const msgTrainingNotFound = "Pelatihan tidak ditemukan."

var trainingStatuses = []string{"planned", "done", "cancelled"}

func (s *Trainings) List(ctx context.Context, a Actor, f domain.TrainingFilter) ([]domain.Training, error) {
	if !a.Can(authn.PermHRRead) {
		id, err := self(a)
		if err != nil {
			return nil, err
		}
		f = domain.TrainingFilter{EmployeeID: id, Year: f.Year}
	}
	return s.d.Store.ListTrainings(ctx, f)
}

func (s *Trainings) Get(ctx context.Context, a Actor, id string) (*domain.Training, []domain.TrainingParticipant, error) {
	t, err := s.d.Store.GetTraining(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if t == nil {
		return nil, nil, notFound(msgTrainingNotFound)
	}
	people, err := s.d.Store.ListTrainingParticipants(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if !a.Can(authn.PermHRRead) {
		// An employee sees a training they joined, and only their own row.
		mine := slices.IndexFunc(people, func(p domain.TrainingParticipant) bool { return p.EmployeeID == a.EmployeeID })
		if a.EmployeeID == "" || mine < 0 {
			return nil, nil, notFound(msgTrainingNotFound)
		}
		people = people[mine : mine+1]
	}
	t.ParticipantCount = len(people)
	return t, people, nil
}

type TrainingInput struct {
	ID, Title, Provider, StartDate, EndDate, Location, Cost, Status, Notes string
}

func (s *Trainings) Save(ctx context.Context, a Actor, in TrainingInput) (*domain.Training, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return nil, invalid("title", "Judul pelatihan wajib diisi.")
	}
	start, err := reqDate("startDate", in.StartDate)
	if err != nil {
		return nil, err
	}
	end, err := reqDate("endDate", in.EndDate)
	if err != nil {
		return nil, err
	}
	if end < start {
		return nil, invalid("endDate", "Tanggal selesai tidak boleh sebelum tanggal mulai.")
	}
	cost := 0.0
	if strings.TrimSpace(in.Cost) != "" {
		if cost, err = amount("cost", in.Cost); err != nil {
			return nil, err
		}
	}
	if in.Status == "" {
		in.Status = "planned"
	}
	if !slices.Contains(trainingStatuses, in.Status) {
		return nil, invalid("status", "Status pelatihan tidak valid.")
	}
	t := domain.Training{ID: in.ID, Title: in.Title, Provider: strings.TrimSpace(in.Provider), StartDate: start, EndDate: end, Location: strings.TrimSpace(in.Location),
		Cost: cost, Status: in.Status, Notes: strings.TrimSpace(in.Notes), CreatedBy: a.UserRef()}
	if t.ID == "" {
		t.ID = newID()
	} else if cur, err := s.d.Store.GetTraining(ctx, t.ID); err != nil {
		return nil, err
	} else if cur == nil {
		return nil, notFound(msgTrainingNotFound)
	}
	if err := s.d.Store.SaveTraining(ctx, t); err != nil {
		return nil, err
	}
	return s.d.Store.GetTraining(ctx, t.ID)
}

func (s *Trainings) Delete(ctx context.Context, a Actor, id string) error {
	if err := require(a, authn.PermHRWrite); err != nil {
		return err
	}
	ok, err := s.d.Store.DeleteTraining(ctx, id)
	if err == nil && !ok {
		return notFound(msgTrainingNotFound)
	}
	return err
}

// SetParticipants replaces the participant list; results already recorded
// for people who stay are kept. New participants are notified.
func (s *Trainings) SetParticipants(ctx context.Context, a Actor, id string, employeeIDs []string) ([]domain.TrainingParticipant, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	t, err := s.d.Store.GetTraining(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, notFound(msgTrainingNotFound)
	}
	before, err := s.d.Store.ListTrainingParticipants(ctx, id)
	if err != nil {
		return nil, err
	}
	keep := []string{}
	for _, e := range employeeIDs {
		if e = strings.TrimSpace(e); e != "" && !slices.Contains(keep, e) {
			keep = append(keep, e)
		}
	}
	err = s.d.Store.WithTx(ctx, func(tx domain.Store) error { return tx.SetTrainingParticipants(ctx, id, keep) })
	if err != nil {
		return nil, err
	}
	for _, e := range keep {
		if !slices.ContainsFunc(before, func(p domain.TrainingParticipant) bool { return p.EmployeeID == e }) {
			notifyEmployee(ctx, s.d, e, NotifyInput{Type: "HR_TRAINING", Title: "Anda terdaftar di pelatihan " + t.Title, Body: t.StartDate + " s.d. " + t.EndDate, Href: "/erp/me"})
		}
	}
	return s.d.Store.ListTrainingParticipants(ctx, id)
}

type ParticipantResult struct {
	EmployeeID, Status, Score, Certificate string
}

func (s *Trainings) RecordResult(ctx context.Context, a Actor, id string, in ParticipantResult) error {
	if err := require(a, authn.PermHRWrite); err != nil {
		return err
	}
	if !slices.Contains([]string{"registered", "attended", "absent"}, in.Status) {
		return invalid("status", "Status kehadiran tidak valid.")
	}
	var score *float64
	if strings.TrimSpace(in.Score) != "" {
		v, err := amount("score", in.Score)
		if err != nil || v > 100 {
			return invalid("score", "Nilai harus 0 sampai 100.")
		}
		score = &v
	}
	ok, err := s.d.Store.UpdateTrainingParticipant(ctx, domain.TrainingParticipant{TrainingID: id, EmployeeID: in.EmployeeID, Status: in.Status, Score: score, Certificate: strings.TrimSpace(in.Certificate)})
	if err == nil && !ok {
		return notFound("Peserta tidak ditemukan.")
	}
	return err
}

// ─── timesheet ───────────────────────────────────────────────

// Timesheet is where employees log hours per project. Entries move draft →
// submitted → approved or rejected; hr.approve decides. Approved hours
// spread each employee's payroll cost over projects (ProjectCosts).
type Timesheet struct{ d Deps }

func NewTimesheet(d Deps) *Timesheet { return &Timesheet{d: d} }

const msgEntryNotFound = "Entri timesheet tidak ditemukan."

// List shows anyone's entries with hr.read (empty employee = everyone),
// otherwise the caller's own.
func (s *Timesheet) List(ctx context.Context, a Actor, f domain.TimesheetFilter) ([]domain.TimesheetEntry, error) {
	var err error
	if !a.Can(authn.PermHRRead) {
		if f.EmployeeID, err = self(a); err != nil {
			return nil, err
		}
	}
	if f.From, err = reqDate("from", f.From); err != nil {
		return nil, err
	}
	if f.To, err = reqDate("to", f.To); err != nil {
		return nil, err
	}
	return s.d.Store.ListTimesheet(ctx, f)
}

type TimesheetInput struct {
	ID, WorkDate, ProjectID, ProjectName, Hours, Description string
}

// Save writes one of the caller's own entries; an edit sends it back to draft.
func (s *Timesheet) Save(ctx context.Context, a Actor, in TimesheetInput) (*domain.TimesheetEntry, error) {
	me, err := self(a)
	if err != nil {
		return nil, err
	}
	day, err := reqDate("workDate", in.WorkDate)
	if err != nil {
		return nil, err
	}
	if day > s.d.today() {
		return nil, invalid("workDate", "Timesheet tidak bisa diisi untuk hari yang belum lewat.")
	}
	hours, err := amount("hours", in.Hours)
	if err != nil || hours <= 0 || hours > 24 || math.Mod(hours*4, 1) != 0 {
		return nil, invalid("hours", "Jam kerja diisi kelipatan 0,25 jam, antara 0,25 dan 24.")
	}
	in.ProjectID, in.ProjectName = strings.TrimSpace(in.ProjectID), strings.TrimSpace(in.ProjectName)
	if in.ProjectID != "" && in.ProjectName == "" {
		return nil, invalid("projectId", "Nama proyek wajib ikut terkirim.")
	}
	if in.ProjectID == "" {
		in.ProjectName = ""
	}
	entry := domain.TimesheetEntry{ID: in.ID, EmployeeID: me, WorkDate: day, ProjectID: in.ProjectID, ProjectName: in.ProjectName, Hours: hours, Description: strings.TrimSpace(in.Description)}
	if entry.ID == "" {
		entry.ID = newID()
	} else {
		cur, err := s.d.Store.GetTimesheetEntry(ctx, entry.ID)
		if err != nil {
			return nil, err
		}
		if cur == nil || cur.EmployeeID != me {
			return nil, notFound(msgEntryNotFound)
		}
		if cur.Status == "submitted" || cur.Status == "approved" {
			return nil, conflict("Entri yang sudah diajukan atau disetujui tidak bisa diubah.")
		}
	}
	others, err := s.d.Store.SumTimesheetDay(ctx, me, day, entry.ID)
	if err != nil {
		return nil, err
	}
	if others+hours > 24 {
		return nil, invalid("hours", "Total jam pada hari itu melebihi 24 jam.")
	}
	if err := s.d.Store.SaveTimesheetEntry(ctx, entry); err != nil {
		return nil, err
	}
	return s.d.Store.GetTimesheetEntry(ctx, entry.ID)
}

func (s *Timesheet) Delete(ctx context.Context, a Actor, id string) error {
	me, err := self(a)
	if err != nil {
		return err
	}
	cur, err := s.d.Store.GetTimesheetEntry(ctx, id)
	if err != nil {
		return err
	}
	if cur == nil || cur.EmployeeID != me {
		return notFound(msgEntryNotFound)
	}
	ok, err := s.d.Store.DeleteTimesheetEntry(ctx, id)
	if err == nil && !ok {
		return conflict("Entri yang sudah diajukan atau disetujui tidak bisa dihapus.")
	}
	return err
}

// Submit sends the caller's draft and rejected entries in [from, to] for approval.
func (s *Timesheet) Submit(ctx context.Context, a Actor, from, to string) (int, error) {
	me, err := self(a)
	if err != nil {
		return 0, err
	}
	if from, err = reqDate("from", from); err != nil {
		return 0, err
	}
	if to, err = reqDate("to", to); err != nil {
		return 0, err
	}
	n, err := s.d.Store.SubmitTimesheet(ctx, me, from, to)
	if err == nil && n > 0 {
		emp, _ := s.d.Store.GetEmployee(ctx, me)
		name := ""
		if emp != nil {
			name = emp.FullName
		}
		notifyHR(ctx, s.d, NotifyInput{Type: "HR_TIMESHEET_SUBMITTED", Title: "Timesheet diajukan: " + name, Body: from + " s.d. " + to, Href: "/erp/hr/timesheet"})
	}
	return n, err
}

// Decide approves or rejects submitted entries; nobody decides their own.
func (s *Timesheet) Decide(ctx context.Context, a Actor, ids []string, approve bool, reason string) (int, error) {
	if err := require(a, authn.PermHRApprove); err != nil {
		return 0, err
	}
	reason = strings.TrimSpace(reason)
	if !approve && reason == "" {
		return 0, invalid("reason", "Alasan penolakan wajib diisi.")
	}
	status := "approved"
	if !approve {
		status = "rejected"
	}
	done := 0
	for _, id := range ids {
		e, err := s.d.Store.GetTimesheetEntry(ctx, id)
		if err != nil {
			return done, err
		}
		if e == nil {
			continue
		}
		if e.EmployeeID == a.EmployeeID {
			return done, forbidden("Timesheet sendiri harus disetujui orang lain.")
		}
		ok, err := s.d.Store.DecideTimesheetEntry(ctx, id, status, reason, a.UserRef())
		if err != nil {
			return done, err
		}
		if ok {
			done++
			if !approve {
				notifyEmployee(ctx, s.d, e.EmployeeID, NotifyInput{Type: "HR_TIMESHEET_REJECTED", Title: "Timesheet " + e.WorkDate + " ditolak", Body: reason, Href: "/erp/me"})
			}
		}
	}
	return done, nil
}

// ProjectCost is payroll cost assigned to a project for a month.
type ProjectCost struct {
	ProjectID, ProjectName string
	Hours, Cost            float64
}

// ProjectCosts spreads each employee's payroll cost for the month (gross
// plus employer BPJS, the amounts Finance expenses) over projects by their
// approved hours. Hours without a project keep their share unallocated.
// The month's payroll must be approved or paid.
func (s *Timesheet) ProjectCosts(ctx context.Context, month string) ([]ProjectCost, error) {
	period, err := time.Parse("2006-01", month)
	if err != nil {
		return nil, invalid("month", "Bulan tidak valid (YYYY-MM).")
	}
	from, to, _ := monthBounds(month)
	run, err := s.d.Store.GetPayrollRunByPeriod(ctx, int(period.Month()), period.Year())
	if err != nil {
		return nil, err
	}
	if run == nil || (run.Status != payroll.RunApproved && run.Status != payroll.RunPaid) {
		return nil, conflict("Payroll bulan " + month + " belum disetujui, jadi biayanya belum bisa dibagi ke proyek.")
	}
	slips, err := s.d.Store.ListPayslips(ctx, domain.PayslipFilter{RunID: run.ID})
	if err != nil {
		return nil, err
	}
	cost := map[string]float64{}
	for _, p := range slips {
		cost[p.EmployeeID] += p.GrossSalary + p.BpjsTkJhtEmployer + p.BpjsTkJpEmployer + p.BpjsTkJkkEmployer + p.BpjsTkJkmEmployer + p.BpjsKesEmployer
	}
	hours, err := s.d.Store.ApprovedProjectHours(ctx, from, to)
	if err != nil {
		return nil, err
	}
	return spreadCost(cost, hours), nil
}

// spreadCost gives each project its share of every employee's cost, rounded
// to whole rupiah.
func spreadCost(cost map[string]float64, hours []domain.ProjectHours) []ProjectCost {
	total := map[string]float64{}
	for _, h := range hours {
		total[h.EmployeeID] += h.Hours
	}
	byProject := map[string]*ProjectCost{}
	var order []string
	for _, h := range hours {
		if h.ProjectID == "" || total[h.EmployeeID] == 0 {
			continue
		}
		pc := byProject[h.ProjectID]
		if pc == nil {
			pc = &ProjectCost{ProjectID: h.ProjectID, ProjectName: h.ProjectName}
			byProject[h.ProjectID] = pc
			order = append(order, h.ProjectID)
		}
		pc.Hours += h.Hours
		pc.Cost += math.Round(cost[h.EmployeeID] * h.Hours / total[h.EmployeeID])
	}
	slices.Sort(order)
	out := make([]ProjectCost, 0, len(order))
	for _, id := range order {
		out = append(out, *byProject[id])
	}
	return out
}
