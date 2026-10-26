package app

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
)

// Contracts is PKWT/PKWTT management (hris contracts.go): numbered drafts,
// activate, end, terminate, convert, renew, and the expiring list.
type Contracts struct{ d Deps }

func NewContracts(d Deps) *Contracts { return &Contracts{d: d} }

type ContractInput struct {
	EmployeeID, ContractType, StartDate, EndDate, ProbationEndDate string
	PositionTitle, DepartmentName, WorkLocation, BaseSalary, Notes string
}

func (c *Contracts) List(ctx context.Context, a Actor, f domain.ContractFilter) ([]domain.Contract, error) {
	if err := require(a, authn.PermHRRead); err != nil {
		return nil, err
	}
	switch f.Status {
	case "":
		f.Status = "active"
	case "all":
		f.Status = ""
	}
	return c.d.Store.ListContracts(ctx, f)
}

func (c *Contracts) Get(ctx context.Context, a Actor, id string) (*domain.Contract, error) {
	ct, err := c.d.Store.GetContract(ctx, id)
	if err != nil {
		return nil, err
	}
	if ct == nil {
		return nil, notFound("Kontrak tidak ditemukan.")
	}
	if ct.EmployeeID != a.EmployeeID {
		if err := require(a, authn.PermHRRead); err != nil {
			return nil, err
		}
	}
	return ct, nil
}

// draft validates in and fills the defaults from the employee and the
// active salary (createDraftContract).
func (c *Contracts) draft(ctx context.Context, in ContractInput) (*domain.Contract, error) {
	if in.ContractType != "pkwt" && in.ContractType != "pkwtt" {
		return nil, invalid("contractType", "Jenis kontrak harus PKWT atau PKWTT.")
	}
	ct := domain.Contract{EmployeeID: in.EmployeeID, ContractType: in.ContractType, StartDate: strings.TrimSpace(in.StartDate),
		EndDate: strPtr(in.EndDate), ProbationEndDate: strPtr(in.ProbationEndDate), PositionTitle: strings.TrimSpace(in.PositionTitle),
		DepartmentName: strings.TrimSpace(in.DepartmentName), WorkLocation: strings.TrimSpace(in.WorkLocation), Notes: strings.TrimSpace(in.Notes)}
	if errs := hris.ValidateContractDates(hris.ContractDates{ContractType: ct.ContractType, StartDate: ct.StartDate,
		EndDate: ct.EndDate, ProbationEndDate: ct.ProbationEndDate}); len(errs) > 0 {
		return nil, invalid("startDate", strings.Join(errs, " "))
	}
	emp, err := c.d.Store.GetEmployee(ctx, in.EmployeeID)
	if err != nil {
		return nil, err
	}
	if emp == nil {
		return nil, notFound(msgEmployeeNotFound)
	}
	if ct.PositionTitle == "" {
		ct.PositionTitle = emp.PositionName
	}
	if ct.DepartmentName == "" {
		ct.DepartmentName = emp.DepartmentName
	}
	if strings.TrimSpace(in.BaseSalary) != "" {
		v, err := amount("baseSalary", in.BaseSalary)
		if err != nil {
			return nil, err
		}
		ct.BaseSalary = &v
	} else if sal, err := c.d.Store.ActiveSalary(ctx, in.EmployeeID); err != nil {
		return nil, err
	} else if sal != nil {
		ct.BaseSalary = &sal.BaseSalary
	}
	return &ct, nil
}

func (c *Contracts) Create(ctx context.Context, a Actor, in ContractInput) (*domain.Contract, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	ct, err := c.draft(ctx, in)
	if err != nil {
		return nil, err
	}
	ct.Sequence = 1
	if ct.ContractType == "pkwt" {
		chain, err := c.pkwtChain(ctx, ct.EmployeeID)
		if err != nil {
			return nil, err
		}
		if msg := hris.ValidatePkwtTotal(hris.PkwtChainTotalMonths(chain), hris.MonthsWorked(ct.StartDate, deref(ct.EndDate))); msg != "" {
			return nil, invalid("endDate", msg)
		}
		ct.Sequence = len(chain) + 1
	}
	ct.CreatedBy = a.UserRef()
	return c.insertNumbered(ctx, *ct)
}

// insertNumbered gives the next number of the type this year, retrying
// once when a concurrent insert took it.
func (c *Contracts) insertNumbered(ctx context.Context, ct domain.Contract) (*domain.Contract, error) {
	ct.ID = newID()
	var err error
	for range 2 {
		n, cerr := c.d.Store.CountContractsOfType(ctx, ct.ContractType, hris.NowWIB(c.d.now()).Year())
		if cerr != nil {
			return nil, cerr
		}
		ct.ContractNumber = hris.BuildContractNumber(ct.ContractType, n+1, hris.NowWIB(c.d.now()))
		var u *domain.ErrUnique
		if err = c.d.Store.InsertContract(ctx, ct); !errors.As(err, &u) {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	return c.d.Store.GetContract(ctx, ct.ID)
}

func (c *Contracts) pkwtChain(ctx context.Context, employeeID string) ([]hris.ContractPeriod, error) {
	rows, err := c.d.Store.PkwtChain(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	out := make([]hris.ContractPeriod, len(rows))
	for i, r := range rows {
		out[i] = hris.ContractPeriod{StartDate: *r[0], EndDate: r[1]}
	}
	return out, nil
}

// UpdateDraft is editDraft: drafts only.
func (c *Contracts) UpdateDraft(ctx context.Context, a Actor, id string, in ContractInput) (*domain.Contract, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	cur, err := c.Get(ctx, a, id)
	if err != nil {
		return nil, err
	}
	if cur.Status != "draft" {
		return nil, conflict("Hanya kontrak berstatus draft yang bisa diedit.")
	}
	in.EmployeeID = cur.EmployeeID
	ct, err := c.draft(ctx, in)
	if err != nil {
		return nil, err
	}
	ct.ID = id
	ok, err := c.d.Store.UpdateDraftContract(ctx, *ct, a.UserRef())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, conflict("Kontrak sudah bukan draft, muat ulang halaman.")
	}
	return c.d.Store.GetContract(ctx, id)
}

func (c *Contracts) Delete(ctx context.Context, a Actor, id string) error {
	if err := require(a, authn.PermHRWrite); err != nil {
		return err
	}
	ok, err := c.d.Store.DeleteDraftContract(ctx, id)
	if err == nil && !ok {
		return conflict("Hanya draft kontrak yang bisa dihapus.")
	}
	return err
}

type ContractActionInput struct {
	Action, EndDate, Reason, SignedAt string
}

type ContractActionResult struct {
	Message      string
	Contract     *domain.Contract
	Compensation *float64
}

// Action runs activate | end | terminate | convert | renew.
func (c *Contracts) Action(ctx context.Context, a Actor, id string, in ContractActionInput) (*ContractActionResult, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	ct, err := c.Get(ctx, a, id)
	if err != nil {
		return nil, err
	}
	var res *ContractActionResult
	switch in.Action {
	case "activate":
		res, err = c.activate(ctx, a, *ct, in)
	case "end":
		res, err = c.end(ctx, a, *ct, in)
	case "terminate":
		res, err = c.terminate(ctx, a, *ct, in)
	case "convert":
		if ct.Status != "active" || ct.ContractType != "pkwt" {
			return nil, conflict("Hanya kontrak PKWT aktif yang bisa dikonversi ke PKWTT.")
		}
		if err := c.setStatus(ctx, domain.ContractStatusUpdate{ID: id, From: "active", To: "converted", Actor: a.UserRef()}); err != nil {
			return nil, err
		}
		res = &ContractActionResult{Message: "Kontrak " + ct.ContractNumber + " ditandai konversi. Buat kontrak PKWTT baru untuk karyawan ini."}
	case "renew":
		return c.renew(ctx, a, *ct, in)
	default:
		return nil, invalid("action", "Aksi kontrak tidak dikenal.")
	}
	if err != nil {
		return nil, err
	}
	res.Contract, err = c.d.Store.GetContract(ctx, id)
	return res, err
}

func (c *Contracts) setStatus(ctx context.Context, u domain.ContractStatusUpdate) error {
	ok, err := c.d.Store.SetContractStatus(ctx, u)
	if err != nil {
		return uniqueErr(err, "Karyawan masih punya kontrak aktif.")
	}
	if !ok {
		return conflict("Status kontrak sudah berubah, muat ulang halaman.")
	}
	return nil
}

// activate: one active contract per employee; the employee status, the
// history row and a salary version follow in the same transaction.
func (c *Contracts) activate(ctx context.Context, a Actor, ct domain.Contract, in ContractActionInput) (*ContractActionResult, error) {
	if ct.Status != "draft" {
		return nil, conflict("Hanya kontrak berstatus draft yang bisa diaktifkan.")
	}
	other, err := c.d.Store.ActiveContractNumber(ctx, ct.EmployeeID, ct.ID)
	if err != nil {
		return nil, err
	}
	if other != "" {
		return nil, conflict("Karyawan masih punya kontrak aktif (" + other + "), akhiri dulu sebelum mengaktifkan kontrak baru.")
	}
	signed, err := optDate("signedAt", in.SignedAt)
	if err != nil {
		return nil, err
	}
	emp, err := c.d.Store.GetEmployee(ctx, ct.EmployeeID)
	if err != nil || emp == nil {
		return nil, err
	}
	newStatus := hris.EmploymentStatusOnActivate(ct.ContractType, ct.ProbationEndDate, c.d.today())
	err = c.d.Store.WithTx(ctx, func(tx domain.Store) error {
		ok, err := tx.SetContractStatus(ctx, domain.ContractStatusUpdate{ID: ct.ID, From: "draft", To: "active", SignedAt: signed, Actor: a.UserRef()})
		if err != nil {
			return err
		}
		if !ok {
			return conflict("Status kontrak sudah berubah, muat ulang halaman.")
		}
		next := *emp
		next.EmploymentStatus, next.IsActive, next.UpdatedBy = newStatus, true, a.UserRef()
		next.EndDate = ct.EndDate
		if ct.ProbationEndDate != nil {
			next.ProbationEndDate = ct.ProbationEndDate
		}
		if _, err := tx.UpdateEmployee(ctx, next); err != nil {
			return err
		}
		if err := tx.InsertHistory(ctx, domain.HistoryEntry{ID: newID(), EmployeeID: ct.EmployeeID, ChangeType: "contract",
			Notes: "Kontrak " + ct.ContractNumber + " diaktifkan", PrevStatus: &emp.EmploymentStatus, NewStatus: &newStatus,
			CreatedBy: a.UserRef()}); err != nil {
			return err
		}
		return syncSalaryFromContract(ctx, tx, ct, a.UserRef())
	})
	if err != nil {
		return nil, uniqueErr(err, "Karyawan masih punya kontrak aktif.")
	}
	return &ContractActionResult{Message: "Kontrak " + ct.ContractNumber + " diaktifkan."}, nil
}

// syncSalaryFromContract starts a salary version with the contract's base
// salary (allowances and flags carried over) when it differs.
func syncSalaryFromContract(ctx context.Context, tx domain.Store, ct domain.Contract, actor *string) error {
	if ct.BaseSalary == nil || *ct.BaseSalary <= 0 {
		return nil
	}
	cur, err := tx.ActiveSalary(ctx, ct.EmployeeID)
	if err != nil {
		return err
	}
	next := domain.Salary{PTKPStatus: "TK/0", IsTaxable: true, BpjsTkEnrolled: true, BpjsKesEnrolled: true, TaperaEnrolled: true}
	if cur != nil {
		if cur.BaseSalary == *ct.BaseSalary {
			return nil
		}
		next = *cur
	}
	next.ID, next.EmployeeID, next.BaseSalary = newID(), ct.EmployeeID, *ct.BaseSalary
	next.EffectiveDate, next.Notes = ct.StartDate, "Dari kontrak "+ct.ContractNumber
	return tx.InsertSalary(ctx, next, hris.AddDaysISO(ct.StartDate, -1), actor)
}

func (c *Contracts) end(ctx context.Context, a Actor, ct domain.Contract, in ContractActionInput) (*ContractActionResult, error) {
	if ct.Status != "active" {
		return nil, conflict("Hanya kontrak aktif yang bisa diakhiri.")
	}
	actual := c.d.today()
	if d, err := optDate("endDate", in.EndDate); err != nil {
		return nil, err
	} else if d != nil {
		actual = *d
	} else if ct.EndDate != nil {
		actual = *ct.EndDate
	}
	var comp *float64
	if ct.ContractType == "pkwt" {
		v := hris.ComputeKompensasi(baseOf(ct), ct.StartDate, actual)
		comp = &v
	}
	if err := c.setStatus(ctx, domain.ContractStatusUpdate{ID: ct.ID, From: "active", To: "ended", EndDate: &actual, Compensation: comp, Actor: a.UserRef()}); err != nil {
		return nil, err
	}
	return &ContractActionResult{Message: "Kontrak " + ct.ContractNumber + " berakhir.", Compensation: comp}, nil
}

// terminate: an early-terminated active PKWT still earns pro-rata
// compensation.
func (c *Contracts) terminate(ctx context.Context, a Actor, ct domain.Contract, in ContractActionInput) (*ContractActionResult, error) {
	if ct.Status != "draft" && ct.Status != "active" {
		return nil, conflict("Kontrak sudah tidak berjalan.")
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		return nil, invalid("reason", "Alasan pemutusan kontrak wajib diisi.")
	}
	var comp *float64
	if ct.ContractType == "pkwt" && ct.Status == "active" {
		v := hris.ComputeKompensasi(baseOf(ct), ct.StartDate, c.d.today())
		comp = &v
	}
	if err := c.setStatus(ctx, domain.ContractStatusUpdate{ID: ct.ID, From: ct.Status, To: "terminated", Compensation: comp, TerminatedReason: &reason, Actor: a.UserRef()}); err != nil {
		return nil, err
	}
	return &ContractActionResult{Message: "Kontrak " + ct.ContractNumber + " diputus.", Compensation: comp}, nil
}

// renew drafts the next PKWT starting the day after the running one ends.
func (c *Contracts) renew(ctx context.Context, a Actor, ct domain.Contract, in ContractActionInput) (*ContractActionResult, error) {
	if ct.Status != "active" || ct.ContractType != "pkwt" {
		return nil, conflict("Hanya kontrak PKWT aktif yang bisa diperpanjang.")
	}
	end, err := optDate("endDate", in.EndDate)
	if err != nil {
		return nil, err
	}
	if end == nil {
		return nil, invalid("endDate", "Tanggal berakhir kontrak perpanjangan wajib diisi.")
	}
	if ct.EndDate == nil {
		return nil, apperr.New(apperr.Unknown, "")
	}
	start := hris.RenewalStartDate(*ct.EndDate)
	if *end <= start {
		return nil, invalid("endDate", "Tanggal berakhir perpanjangan harus setelah "+start+".")
	}
	chain, err := c.pkwtChain(ctx, ct.EmployeeID)
	if err != nil {
		return nil, err
	}
	if msg := hris.ValidatePkwtTotal(hris.PkwtChainTotalMonths(chain), hris.MonthsWorked(start, *end)); msg != "" {
		return nil, invalid("endDate", msg)
	}
	next := domain.Contract{EmployeeID: ct.EmployeeID, ContractType: "pkwt", StartDate: start, EndDate: end, ParentContractID: &ct.ID,
		Sequence: ct.Sequence + 1, PositionTitle: ct.PositionTitle, DepartmentName: ct.DepartmentName, WorkLocation: ct.WorkLocation,
		BaseSalary: ct.BaseSalary, CreatedBy: a.UserRef()}
	row, err := c.insertNumbered(ctx, next)
	if err != nil {
		return nil, err
	}
	return &ContractActionResult{Message: "Draft perpanjangan " + row.ContractNumber + " dibuat (mulai " + start +
		"). Aktifkan setelah kontrak berjalan berakhir.", Contract: row}, nil
}

func baseOf(ct domain.Contract) float64 {
	if ct.BaseSalary == nil {
		return 0
	}
	return *ct.BaseSalary
}

type Expiring struct {
	Days            int
	Contracts       []domain.Contract
	Probations      []domain.Employee
	WithoutContract []domain.Employee
}

// Expiring lists contracts ending within days (1..90, default 30)
// including overdue ones, probations ending, and active employees
// without a contract.
func (c *Contracts) Expiring(ctx context.Context, a Actor, days int) (*Expiring, error) {
	if err := require(a, authn.PermHRRead); err != nil {
		return nil, err
	}
	days = hris.ClampExpiringDays(strconv.Itoa(days), days != 0)
	today := c.d.today()
	limit := hris.AddDaysISO(today, days)
	active, err := c.d.Store.ListContracts(ctx, domain.ContractFilter{Status: "active"})
	if err != nil {
		return nil, err
	}
	out := &Expiring{Days: days}
	for _, ct := range active {
		if ct.EndDate != nil && *ct.EndDate <= limit {
			out.Contracts = append(out.Contracts, ct)
		}
	}
	if out.Probations, err = c.d.Store.ProbationsEnding(ctx, hris.AddDaysISO(today, -30), limit); err != nil {
		return nil, err
	}
	if out.WithoutContract, err = c.d.Store.EmployeesWithoutContract(ctx); err != nil {
		return nil, err
	}
	return out, nil
}
