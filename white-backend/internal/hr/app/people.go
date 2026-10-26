package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// People manages employee records and master data (hris employees,
// departments, positions, employment statuses, documents, org chart).
type People struct{ d Deps }

func NewPeople(d Deps) *People { return &People{d: d} }

// EmployeeInput is the create/update body.
type EmployeeInput struct {
	EmployeeNo, FullName, Email, Phone               string
	DepartmentID, PositionID, ManagerID              string
	EmploymentStatus                                 string
	JoinDate, EndDate, ProbationEndDate              string
	NIK, NPWP, BpjsTkNo, BpjsKesNo                   string
	BirthPlace, BirthDate, Gender, MaritalStatus     string
	Religion, Address                                string
	BankName, BankAccountNo, BankAccountName         string
	EmergencyName, EmergencyPhone, EmergencyRelation string
	UserID                                           string
}

var (
	genders         = []string{"", "male", "female"}
	maritalStatuses = []string{"", "single", "married", "divorced", "widowed"}
	documentTypes   = []string{"ktp", "npwp", "kk", "ijazah", "cv", "contract", "certificate", "other"}
	documentMIME    = []string{"application/pdf", "image/jpeg", "image/png", "image/webp"}
)

const maxDocumentBytes = 10 << 20

// ─── Directory (EmployeeService) ─────────────────────────────

func (p *People) List(ctx context.Context, f domain.EmployeeFilter) ([]domain.Employee, int, error) {
	return p.d.Store.ListEmployees(ctx, f)
}

func (p *People) Get(ctx context.Context, id string) (*domain.Employee, error) {
	e, err := p.d.Store.GetEmployee(ctx, id)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, notFound(msgEmployeeNotFound)
	}
	return e, nil
}

// ByUser is the employee linked to an identity user (NOT_FOUND when none).
func (p *People) ByUser(ctx context.Context, userID string) (*domain.Employee, error) {
	id, err := p.d.Store.EmployeeIDByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, notFound(msgNotLinked)
	}
	return p.Get(ctx, id)
}

func (p *People) Departments(ctx context.Context, includeInactive bool) ([]domain.Department, error) {
	return p.d.Store.ListDepartments(ctx, includeInactive)
}

// AttendanceSummary is the month's attendance totals per employee for the
// KPI collectors (working days from the shift pattern, else weekdays).
type AttendanceSummary struct {
	EmployeeID                                     string
	WorkingDays, PresentDays, LateDays, AbsentDays int
	LeaveDays, LateMinutes                         int
	OvertimeHours                                  float64
}

func (p *People) AttendanceSummary(ctx context.Context, employeeIDs []string, month string) ([]AttendanceSummary, error) {
	start, end, ok := monthBounds(month)
	if !ok {
		return nil, invalid("month", "Bulan tidak valid (YYYY-MM).")
	}
	totals, err := p.d.Store.AttendanceTotals(ctx, employeeIDs, start, end)
	if err != nil {
		return nil, err
	}
	schedules, err := p.d.Store.ListScheduleRows(ctx, employeeIDs)
	if err != nil {
		return nil, err
	}
	leaves, _, err := p.d.Store.ListLeaves(ctx, domain.LeaveFilter{EmployeeIDs: employeeIDs, Status: hris.LeaveApproved, From: start, To: end})
	if err != nil {
		return nil, err
	}
	out := make([]AttendanceSummary, 0, len(employeeIDs))
	for _, id := range employeeIDs {
		t := totals[id]
		s := AttendanceSummary{EmployeeID: id, PresentDays: t.PresentDays, LateDays: t.LateDays, AbsentDays: t.AbsentDays,
			LateMinutes: t.LateMinutes, OvertimeHours: t.OvertimeHours}
		s.WorkingDays = workingDays(plainRows(scheduleOf(schedules, id)), start, end)
		for _, l := range leaves {
			if l.EmployeeID == id {
				s.LeaveDays += len(hris.EachDateISO(max(l.StartDate, start), min(l.EndDate, end)))
			}
		}
		out = append(out, s)
	}
	return out, nil
}

// ─── Profile CRUD ────────────────────────────────────────────

func (p *People) Create(ctx context.Context, a Actor, in EmployeeInput) (*domain.Employee, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	e := domain.Employee{ID: newID(), IsActive: true, CreatedBy: a.UserRef(), UpdatedBy: a.UserRef()}
	if err := p.apply(ctx, &e, in); err != nil {
		return nil, err
	}
	if e.EmployeeNo == "" {
		no, err := p.nextEmployeeNo(ctx, e.JoinDate)
		if err != nil {
			return nil, err
		}
		e.EmployeeNo = no
	}
	if uid := strings.TrimSpace(in.UserID); uid != "" {
		if err := p.checkUser(ctx, uid); err != nil {
			return nil, err
		}
		e.UserID = &uid
	}
	if err := p.d.Store.InsertEmployee(ctx, e); err != nil {
		return nil, employeeUnique(err)
	}
	return p.Get(ctx, e.ID)
}

func (p *People) Update(ctx context.Context, a Actor, id string, in EmployeeInput) (*domain.Employee, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	cur, err := p.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	next := *cur
	next.UpdatedBy = a.UserRef()
	if err := p.apply(ctx, &next, in); err != nil {
		return nil, err
	}
	if next.EmployeeNo == "" {
		next.EmployeeNo = cur.EmployeeNo
	}
	if next.ManagerID != nil {
		if err := p.checkNoCycle(ctx, id, *next.ManagerID); err != nil {
			return nil, err
		}
	}
	change := hris.EmploymentHistoryChange(
		hris.TrackedState{EmploymentStatus: &cur.EmploymentStatus, DepartmentID: cur.DepartmentID, JobTitleID: cur.PositionID},
		hris.TrackedUpdate{EmploymentStatus: &next.EmploymentStatus, SentStatus: true, DepartmentID: next.DepartmentID,
			SentDepartment: true, JobTitleID: next.PositionID, SentJobTitle: true})
	err = p.d.Store.WithTx(ctx, func(tx domain.Store) error {
		ok, err := tx.UpdateEmployee(ctx, next)
		if err != nil {
			return employeeUnique(err)
		}
		if !ok {
			return notFound(msgEmployeeNotFound)
		}
		if change == nil {
			return nil
		}
		return tx.InsertHistory(ctx, domain.HistoryEntry{ID: newID(), EmployeeID: id, ChangeType: "update", Notes: change.Notes,
			PrevStatus: change.PrevEmploymentStatus, NewStatus: change.NewEmploymentStatus,
			PrevDepartmentID: change.PrevDepartmentID, NewDepartmentID: change.NewDepartmentID,
			PrevPositionID: change.PrevJobTitleID, NewPositionID: change.NewJobTitleID, CreatedBy: a.UserRef()})
	})
	if err != nil {
		return nil, err
	}
	return p.Get(ctx, id)
}

func (p *People) Delete(ctx context.Context, a Actor, id string) error {
	if err := require(a, authn.PermHRWrite); err != nil {
		return err
	}
	ok, err := p.d.Store.DeleteEmployee(ctx, id, a.UserRef())
	if err == nil && !ok {
		return notFound(msgEmployeeNotFound)
	}
	return err
}

// LinkUser links an identity user (checked with identity GetUser); an
// empty userID unlinks.
func (p *People) LinkUser(ctx context.Context, a Actor, id, userID string) (*domain.Employee, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	var ref *string
	if userID = strings.TrimSpace(userID); userID != "" {
		if err := p.checkUser(ctx, userID); err != nil {
			return nil, err
		}
		ref = &userID
	}
	ok, err := p.d.Store.SetEmployeeUser(ctx, id, ref, a.UserRef())
	if err != nil {
		return nil, uniqueErr(err, "Akun ini sudah terhubung ke karyawan lain.")
	}
	if !ok {
		return nil, notFound(msgEmployeeNotFound)
	}
	return p.Get(ctx, id)
}

func (p *People) checkUser(ctx context.Context, userID string) error {
	u, err := p.d.Identity.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		return invalid("userId", "Akun pengguna tidak ditemukan.")
	}
	return nil
}

func (p *People) History(ctx context.Context, a Actor, id string) ([]domain.HistoryEntry, error) {
	if err := require(a, authn.PermHRRead); err != nil {
		return nil, err
	}
	return p.d.Store.ListHistory(ctx, id)
}

// apply validates in and copies it onto e.
func (p *People) apply(ctx context.Context, e *domain.Employee, in EmployeeInput) error {
	v := validate.New()
	v.Required("fullName", in.FullName, "Nama lengkap wajib diisi.")
	if in.Email != "" {
		v.Email("email", in.Email, "Format email tidak valid.")
	}
	v.OneOf("gender", in.Gender, genders, "Jenis kelamin tidak valid.")
	v.OneOf("maritalStatus", in.MaritalStatus, maritalStatuses, "Status pernikahan tidak valid.")
	join, err := reqDate("joinDate", in.JoinDate)
	if err != nil {
		v.Add("joinDate", "Tanggal bergabung wajib diisi (YYYY-MM-DD).")
	}
	for _, d := range []struct {
		field, raw string
		dst        **string
	}{{"endDate", in.EndDate, &e.EndDate}, {"probationEndDate", in.ProbationEndDate, &e.ProbationEndDate}, {"birthDate", in.BirthDate, &e.BirthDate}} {
		v2, err := optDate(d.field, d.raw)
		if err != nil {
			v.Add(d.field, msgInvalidDate)
		}
		*d.dst = v2
	}
	if err := v.Err(); err != nil {
		return err
	}
	status := strings.TrimSpace(in.EmploymentStatus)
	if status == "" {
		status = "probation"
	}
	statuses, err := p.d.Store.ListEmploymentStatuses(ctx)
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(statuses, func(s domain.EmploymentStatusOption) bool { return s.Code == status }) {
		return invalid("employmentStatus", "Status kepegawaian tidak dikenal.")
	}
	if e.DepartmentID = strPtr(in.DepartmentID); e.DepartmentID != nil {
		d, err := p.d.Store.GetDepartment(ctx, *e.DepartmentID)
		if err != nil {
			return err
		}
		if d == nil {
			return invalid("departmentId", "Departemen tidak ditemukan.")
		}
	}
	if e.PositionID = strPtr(in.PositionID); e.PositionID != nil {
		pos, err := p.d.Store.GetPosition(ctx, *e.PositionID)
		if err != nil {
			return err
		}
		if pos == nil {
			return invalid("positionId", "Jabatan tidak ditemukan.")
		}
	}
	if e.ManagerID = strPtr(in.ManagerID); e.ManagerID != nil {
		if *e.ManagerID == e.ID {
			return invalid("managerId", "Atasan tidak boleh diri sendiri.")
		}
		m, err := p.d.Store.GetEmployee(ctx, *e.ManagerID)
		if err != nil {
			return err
		}
		if m == nil {
			return invalid("managerId", "Atasan tidak ditemukan.")
		}
	}
	e.EmployeeNo = strings.TrimSpace(in.EmployeeNo)
	e.FullName = strings.TrimSpace(in.FullName)
	e.Email = strings.ToLower(strings.TrimSpace(in.Email))
	e.Phone = strings.TrimSpace(in.Phone)
	e.EmploymentStatus = status
	e.IsActive = status != "resigned" && status != "terminated"
	e.JoinDate = join
	e.NIK, e.NPWP = strings.TrimSpace(in.NIK), strings.TrimSpace(in.NPWP)
	e.BpjsTkNo, e.BpjsKesNo = strings.TrimSpace(in.BpjsTkNo), strings.TrimSpace(in.BpjsKesNo)
	e.BirthPlace, e.Gender, e.MaritalStatus = strings.TrimSpace(in.BirthPlace), in.Gender, in.MaritalStatus
	e.Religion, e.Address = strings.TrimSpace(in.Religion), strings.TrimSpace(in.Address)
	e.BankName, e.BankAccountNo, e.BankAccountName = strings.TrimSpace(in.BankName), strings.TrimSpace(in.BankAccountNo), strings.TrimSpace(in.BankAccountName)
	e.EmergencyName, e.EmergencyPhone, e.EmergencyRelation = strings.TrimSpace(in.EmergencyName), strings.TrimSpace(in.EmergencyPhone), strings.TrimSpace(in.EmergencyRelation)
	return nil
}

// checkNoCycle refuses a manager whose own chain reaches the employee.
func (p *People) checkNoCycle(ctx context.Context, employeeID, managerID string) error {
	seen := map[string]bool{employeeID: true}
	for cur := managerID; cur != ""; {
		if seen[cur] {
			return invalid("managerId", "Atasan membentuk struktur melingkar.")
		}
		seen[cur] = true
		m, err := p.d.Store.GetEmployee(ctx, cur)
		if err != nil || m == nil {
			return err
		}
		cur = deref(m.ManagerID)
	}
	return nil
}

func (p *People) nextEmployeeNo(ctx context.Context, joinDate string) (string, error) {
	year := p.d.now().Year()
	if t, ok := hris.ParseDate(joinDate); ok {
		year = t.Year()
	}
	taken, err := p.d.Store.ListEmployeeNos(ctx, fmt.Sprintf("EMP-%d-", year))
	if err != nil {
		return "", err
	}
	no := hris.NextAutoNip(year, taken)
	if no == "" {
		return "", conflict("Nomor karyawan tahun ini habis.")
	}
	return no, nil
}

func employeeUnique(err error) error {
	var u *domain.ErrUnique
	if !errors.As(err, &u) {
		return err
	}
	if msg := hris.EmployeeUniqueMessage(u.Constraint); msg != "" {
		return conflict(msg)
	}
	switch {
	case strings.Contains(u.Constraint, "employee_no"):
		return conflict("Nomor karyawan sudah dipakai.")
	case strings.Contains(u.Constraint, "user_id"):
		return conflict("Akun ini sudah terhubung ke karyawan lain.")
	}
	return conflict("")
}

// ─── Org chart ───────────────────────────────────────────────

type OrgNode struct {
	Employee domain.Employee
	Reports  []*OrgNode
}

func (p *People) OrgChart(ctx context.Context, a Actor, rootID string) ([]*OrgNode, int, error) {
	if err := require(a, authn.PermHRRead, authn.PermSelfService); err != nil {
		return nil, 0, err
	}
	all, _, err := p.d.Store.ListEmployees(ctx, domain.EmployeeFilter{})
	if err != nil {
		return nil, 0, err
	}
	nodes := make(map[string]*OrgNode, len(all))
	for _, e := range all {
		nodes[e.ID] = &OrgNode{Employee: e}
	}
	var roots []*OrgNode
	for _, e := range all {
		n := nodes[e.ID]
		if parent, ok := nodes[deref(e.ManagerID)]; ok && e.ManagerID != nil {
			parent.Reports = append(parent.Reports, n)
			continue
		}
		roots = append(roots, n)
	}
	if rootID != "" {
		n, ok := nodes[rootID]
		if !ok {
			return nil, 0, notFound(msgEmployeeNotFound)
		}
		roots = []*OrgNode{n}
	}
	return roots, len(all), nil
}

// ─── Documents ───────────────────────────────────────────────

func (p *People) Documents(ctx context.Context, a Actor, employeeID string) ([]domain.Document, error) {
	if err := p.canSeeEmployee(a, employeeID); err != nil {
		return nil, err
	}
	return p.d.Store.ListDocuments(ctx, employeeID)
}

func (p *People) UploadDocument(ctx context.Context, a Actor, employeeID, docType, name string, data []byte) (*domain.Document, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	if !slices.Contains(documentTypes, docType) {
		return nil, invalid("docType", "Jenis dokumen tidak valid.")
	}
	if _, err := p.Get(ctx, employeeID); err != nil {
		return nil, err
	}
	key, ct, err := p.d.Files.Put(ctx, "documents/"+employeeID, data, documentMIME, maxDocumentBytes)
	if err != nil {
		return nil, fileErr(err, "Ukuran dokumen maksimal 10 MB.")
	}
	if name = strings.TrimSpace(name); name == "" {
		name = docType
	}
	doc := domain.Document{ID: newID(), EmployeeID: employeeID, DocType: docType, Name: name, StorageKey: key,
		ContentType: ct, SizeBytes: int64(len(data)), CreatedBy: a.UserRef(), CreatedAt: p.d.now()}
	if err := p.d.Store.InsertDocument(ctx, doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func (p *People) DownloadDocument(ctx context.Context, a Actor, id string) (*domain.Document, []byte, error) {
	doc, err := p.d.Store.GetDocument(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if doc == nil {
		return nil, nil, notFound("Dokumen tidak ditemukan.")
	}
	if err := p.canSeeEmployee(a, doc.EmployeeID); err != nil {
		return nil, nil, err
	}
	data, _, err := p.d.Files.Get(ctx, doc.StorageKey)
	if err != nil {
		return nil, nil, err
	}
	return doc, data, nil
}

func (p *People) DeleteDocument(ctx context.Context, a Actor, id string) error {
	if err := require(a, authn.PermHRWrite); err != nil {
		return err
	}
	ok, err := p.d.Store.DeleteDocument(ctx, id)
	if err == nil && !ok {
		return notFound("Dokumen tidak ditemukan.")
	}
	return err
}

// canSeeEmployee: hr.read for anyone, self.service for oneself.
func (p *People) canSeeEmployee(a Actor, employeeID string) error {
	if a.Can(authn.PermHRRead) {
		return nil
	}
	if a.EmployeeID == employeeID && a.Can(authn.PermSelfService) {
		return nil
	}
	return apperr.New(apperr.Unauthorized, "")
}

func fileErr(err error, tooLarge string) error {
	switch {
	case errors.Is(err, ErrFileType):
		return invalid("data", "Jenis file tidak didukung.")
	case errors.Is(err, ErrFileLarge):
		return invalid("data", tooLarge)
	}
	return err
}

// ─── Departments, positions, statuses ────────────────────────

type DepartmentInput struct {
	Code, Name, ParentID, HeadEmployeeID string
	IsActive                             bool
}

func (p *People) SaveDepartment(ctx context.Context, a Actor, id string, in DepartmentInput) (*domain.Department, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	v := validate.New()
	v.Required("code", in.Code, "Kode departemen wajib diisi.")
	v.Required("name", in.Name, "Nama departemen wajib diisi.")
	if err := v.Err(); err != nil {
		return nil, err
	}
	d := domain.Department{ID: id, Code: strings.ToUpper(strings.TrimSpace(in.Code)), Name: strings.TrimSpace(in.Name),
		ParentID: strPtr(in.ParentID), HeadEmployeeID: strPtr(in.HeadEmployeeID), IsActive: in.IsActive}
	if d.ParentID != nil && *d.ParentID == id {
		return nil, invalid("parentId", "Induk departemen tidak boleh dirinya sendiri.")
	}
	if d.HeadEmployeeID != nil {
		if _, err := p.Get(ctx, *d.HeadEmployeeID); err != nil {
			return nil, invalid("headEmployeeId", "Kepala departemen tidak ditemukan.")
		}
	}
	var err error
	if id == "" {
		d.ID = newID()
		err = p.d.Store.InsertDepartment(ctx, d, a.UserRef())
	} else {
		var ok bool
		ok, err = p.d.Store.UpdateDepartment(ctx, d, a.UserRef())
		if err == nil && !ok {
			return nil, notFound("Departemen tidak ditemukan.")
		}
	}
	if err != nil {
		return nil, uniqueErr(err, "Kode departemen sudah dipakai.")
	}
	return p.d.Store.GetDepartment(ctx, d.ID)
}

func (p *People) DeleteDepartment(ctx context.Context, a Actor, id string) error {
	if err := require(a, authn.PermHRWrite); err != nil {
		return err
	}
	n, err := p.d.Store.CountDepartmentEmployees(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return conflict(fmt.Sprintf("Departemen masih punya %d karyawan aktif.", n))
	}
	ok, err := p.d.Store.DeleteDepartment(ctx, id, a.UserRef())
	if err == nil && !ok {
		return notFound("Departemen tidak ditemukan.")
	}
	return err
}

var positionLevels = []string{"staff", "senior", "lead", "manager", "director"}

type PositionInput struct {
	Title, DepartmentID, Level, SalaryMin, SalaryMax string
	IsActive                                         bool
}

func (p *People) Positions(ctx context.Context, a Actor, departmentID string, includeInactive bool) ([]domain.Position, error) {
	if err := require(a, authn.PermHRRead, authn.PermSelfService); err != nil {
		return nil, err
	}
	return p.d.Store.ListPositions(ctx, departmentID, includeInactive)
}

func (p *People) SavePosition(ctx context.Context, a Actor, id string, in PositionInput) (*domain.Position, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	level := in.Level
	if level == "" {
		level = "staff"
	}
	v := validate.New()
	v.Required("title", in.Title, "Nama jabatan wajib diisi.")
	v.OneOf("level", level, positionLevels, "Level jabatan tidak valid.")
	if err := v.Err(); err != nil {
		return nil, err
	}
	lo, err := amount("salaryMin", in.SalaryMin)
	if err != nil {
		return nil, err
	}
	hi, err := amount("salaryMax", in.SalaryMax)
	if err != nil {
		return nil, err
	}
	if hi > 0 && lo > hi {
		return nil, invalid("salaryMax", "Gaji maksimum harus lebih besar dari minimum.")
	}
	pos := domain.Position{ID: id, Title: strings.TrimSpace(in.Title), DepartmentID: strPtr(in.DepartmentID), Level: level,
		SalaryMin: lo, SalaryMax: hi, IsActive: in.IsActive}
	if id == "" {
		pos.ID = newID()
		err = p.d.Store.InsertPosition(ctx, pos, a.UserRef())
	} else {
		var ok bool
		ok, err = p.d.Store.UpdatePosition(ctx, pos, a.UserRef())
		if err == nil && !ok {
			return nil, notFound("Jabatan tidak ditemukan.")
		}
	}
	if err != nil {
		return nil, err
	}
	return p.d.Store.GetPosition(ctx, pos.ID)
}

func (p *People) DeletePosition(ctx context.Context, a Actor, id string) error {
	if err := require(a, authn.PermHRWrite); err != nil {
		return err
	}
	ok, err := p.d.Store.DeletePosition(ctx, id, a.UserRef())
	if err == nil && !ok {
		return notFound("Jabatan tidak ditemukan.")
	}
	return err
}

func (p *People) EmploymentStatuses(ctx context.Context, a Actor) ([]domain.EmploymentStatusOption, error) {
	if err := require(a, authn.PermHRRead, authn.PermSelfService); err != nil {
		return nil, err
	}
	return p.d.Store.ListEmploymentStatuses(ctx)
}

func (p *People) UpsertEmploymentStatus(ctx context.Context, a Actor, o domain.EmploymentStatusOption) (*domain.EmploymentStatusOption, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	o.Code = strings.ToLower(strings.TrimSpace(o.Code))
	v := validate.New()
	v.Required("code", o.Code, "Kode status wajib diisi.")
	v.Required("name", o.Name, "Nama status wajib diisi.")
	if err := v.Err(); err != nil {
		return nil, err
	}
	if err := p.d.Store.UpsertEmploymentStatus(ctx, o); err != nil {
		return nil, err
	}
	return &o, nil
}

func (p *People) DeleteEmploymentStatus(ctx context.Context, a Actor, code string) error {
	if err := require(a, authn.PermHRWrite); err != nil {
		return err
	}
	ok, err := p.d.Store.DeleteEmploymentStatus(ctx, code)
	if err == nil && !ok {
		return conflict("Status tidak ditemukan atau masih dipakai karyawan.")
	}
	return err
}
