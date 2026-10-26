package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Accounts covers account types, the chart of accounts with its XLSX
// import, and journal mappings (NüHabit service_setup.go, coa_import.go).
type Accounts struct{ Deps }

func NewAccounts(d Deps) *Accounts { return &Accounts{d} }

func uniqueAs(err error, msg string) error {
	var u *domain.ErrUnique
	if errors.As(err, &u) {
		return domain.Bad(msg)
	}
	return err
}

// ─── Account types ──────────────────────────────────────────

func (a *Accounts) ListAccountTypes(ctx context.Context, includeInactive bool) ([]domain.AccountType, error) {
	return a.Store.ListAccountTypes(ctx, includeInactive)
}

func validateAccountType(t *domain.AccountType) error {
	t.Code = strings.ToUpper(trim(t.Code))
	t.Name = trim(t.Name)
	v := validate.New().
		Required("code", t.Code, "Kode wajib diisi").
		Required("name", t.Name, "Nama wajib diisi").
		OneOf("normal_balance", t.NormalBalance, []string{domain.Debit, domain.Credit}, "Normal balance harus DEBIT atau CREDIT")
	return v.Err()
}

func (a *Accounts) SaveAccountType(ctx context.Context, userID string, t domain.AccountType) (*domain.AccountType, error) {
	if err := validateAccountType(&t); err != nil {
		return nil, err
	}
	if t.ID == "" {
		t.ID = ids.New()
	} else if existing, err := a.Store.GetAccountType(ctx, t.ID); err != nil {
		return nil, err
	} else if existing == nil {
		return nil, domain.NotFound("Account type tidak ditemukan")
	}
	if err := a.Store.UpsertAccountType(ctx, t, userID); err != nil {
		return nil, uniqueAs(err, "Kode account type sudah digunakan")
	}
	return a.Store.GetAccountType(ctx, t.ID)
}

func (a *Accounts) DeleteAccountType(ctx context.Context, id string) error {
	t, err := a.Store.GetAccountType(ctx, id)
	if err != nil {
		return err
	}
	if t == nil {
		return domain.NotFound("Account type tidak ditemukan")
	}
	if t.AccountCount > 0 {
		return domain.Bad(fmt.Sprintf("Tidak dapat dihapus: masih dipakai %d akun COA", t.AccountCount))
	}
	return a.Store.DeleteAccountType(ctx, id)
}

// ─── Chart of accounts ──────────────────────────────────────

func (a *Accounts) ListAccounts(ctx context.Context, f domain.AccountFilter) ([]domain.Account, error) {
	f.Search = trim(f.Search)
	return a.Store.ListAccounts(ctx, f)
}

func (a *Accounts) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	acc, err := a.Store.GetAccount(ctx, id)
	if err == nil && acc == nil {
		err = domain.NotFound("Akun tidak ditemukan")
	}
	return acc, err
}

func (a *Accounts) validateAccount(ctx context.Context, in *domain.Account) error {
	code, level, err := domain.ResolveAccountCode(in.Code)
	if err != nil {
		return err
	}
	in.Code, in.Level, in.Name = code, level, trim(in.Name)
	in.CashFlowCategory = strings.ToUpper(trim(in.CashFlowCategory))
	v := validate.New().Required("name", in.Name, "Nama akun wajib diisi").Required("account_type_id", in.AccountTypeID, "Account type wajib dipilih")
	if in.CashFlowCategory != "" && !slices.Contains(domain.CashFlowCategories, in.CashFlowCategory) {
		v.Add("cash_flow_category", "Cash flow category tidak valid")
	}
	if err := v.Err(); err != nil {
		return err
	}
	if t, err := a.Store.GetAccountType(ctx, in.AccountTypeID); err != nil {
		return err
	} else if t == nil {
		return apperr.Field("account_type_id", "Account type tidak ditemukan")
	}
	if in.ParentID == "" {
		return nil
	}
	if in.ParentID == in.ID {
		return domain.Bad("Parent tidak boleh akun itu sendiri")
	}
	parent, err := a.Store.GetAccount(ctx, in.ParentID)
	if err != nil {
		return err
	}
	if parent == nil {
		return domain.Bad("Parent akun tidak ditemukan")
	}
	return nil
}

func (a *Accounts) CreateAccount(ctx context.Context, userID string, in domain.Account) (*domain.Account, error) {
	if err := a.validateAccount(ctx, &in); err != nil {
		return nil, err
	}
	in.ID = ids.New()
	err := a.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertAccount(ctx, in, userID); err != nil {
			return uniqueAs(err, "Kode akun sudah digunakan")
		}
		if err := tx.RecomputePostable(ctx, in.ParentID); err != nil {
			return err
		}
		return tx.RecomputePostable(ctx, in.ID)
	})
	if err != nil {
		return nil, err
	}
	return a.Store.GetAccount(ctx, in.ID)
}

func (a *Accounts) UpdateAccount(ctx context.Context, userID string, in domain.Account) (*domain.Account, error) {
	existing, err := a.GetAccount(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if err := a.validateAccount(ctx, &in); err != nil {
		return nil, err
	}
	err = a.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.UpdateAccount(ctx, in, userID); err != nil {
			return uniqueAs(err, "Kode akun sudah digunakan")
		}
		for _, id := range []string{existing.ParentID, in.ParentID, in.ID} {
			if err := tx.RecomputePostable(ctx, id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return a.Store.GetAccount(ctx, in.ID)
}

// DeleteAccount soft-deletes; refused while live children exist.
func (a *Accounts) DeleteAccount(ctx context.Context, userID, id string) error {
	existing, err := a.GetAccount(ctx, id)
	if err != nil {
		return err
	}
	n, err := a.Store.CountChildren(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return domain.Bad(fmt.Sprintf("Tidak dapat dihapus: masih ada %d akun anak", n))
	}
	return a.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.SoftDeleteAccount(ctx, id, userID); err != nil {
			return err
		}
		return tx.RecomputePostable(ctx, existing.ParentID)
	})
}

// ImportResult is the preview or commit outcome of a COA import.
type ImportResult struct {
	Mode    string
	Issues  []domain.CoaIssue
	Preview []domain.CoaPreviewRow
	Summary domain.CoaImportSummary
	Message string
}

// ImportAccounts previews the workbook against the chart and, in "commit"
// mode with no errors, writes it in one transaction (parents first).
func (a *Accounts) ImportAccounts(ctx context.Context, userID string, file []byte, mode string) (*ImportResult, error) {
	if len(file) == 0 {
		return nil, apperr.Field("file", "File Excel wajib diunggah")
	}
	if mode != "commit" {
		mode = "preview"
	}
	matrix, err := a.ReadSheet(file)
	if err != nil {
		return nil, apperr.Field("file", "File Excel tidak bisa dibaca")
	}
	parsed, issues := domain.ParseCoaMatrix(matrix)
	if len(parsed) == 0 {
		return &ImportResult{Mode: mode, Issues: issues, Summary: domain.CoaImportSummary{Error: len(issues)}, Message: "Tidak ada baris valid untuk diimpor"}, nil
	}
	types, err := a.Store.ListAccountTypes(ctx, false)
	if err != nil {
		return nil, err
	}
	typeIDs := map[string]string{}
	for _, t := range types {
		typeIDs[strings.ToUpper(t.Code)] = t.ID
	}
	accounts, err := a.Store.AccountsByCode(ctx)
	if err != nil {
		return nil, err
	}
	existing := map[string]domain.ExistingAccount{}
	for code, acc := range accounts {
		existing[code] = domain.ExistingAccount{ID: acc.ID, Name: acc.Name}
	}
	preview := domain.BuildCoaImportPreview(parsed, typeIDs, existing)
	summary := domain.SummarizeCoaImport(preview, issues)
	res := &ImportResult{Mode: mode, Issues: issues, Preview: preview, Summary: summary}
	if mode != "commit" {
		return res, nil
	}
	if summary.Error > 0 {
		return nil, domain.Bad("Masih ada error, perbaiki file sebelum commit")
	}
	ordered := slices.Clone(parsed)
	slices.SortStableFunc(ordered, func(x, y domain.CoaRow) int {
		if x.Level != y.Level {
			return x.Level - y.Level
		}
		return strings.Compare(x.Code, y.Code)
	})
	err = a.Store.WithTx(ctx, func(tx domain.Store) error {
		idByCode := map[string]string{}
		for code, acc := range accounts {
			idByCode[code] = acc.ID
		}
		for _, row := range ordered {
			acc := domain.Account{
				Code: row.Code, Name: row.Name, ParentID: idByCode[row.ParentCode], AccountTypeID: typeIDs[row.AccountTypeCode],
				Level: row.Level, IsContra: row.IsContra, IsCashBank: row.IsCashBank, CashFlowCategory: row.CashFlowCategory,
				Description: row.Description, IsActive: true,
			}
			if ex, ok := accounts[row.Code]; ok {
				acc.ID = ex.ID
				if err := tx.UpdateAccount(ctx, acc, userID); err != nil {
					return err
				}
				continue
			}
			acc.ID = ids.New()
			if err := tx.InsertAccount(ctx, acc, userID); err != nil {
				return err
			}
			idByCode[row.Code] = acc.ID
		}
		return tx.RecomputeAllPostable(ctx)
	})
	if err != nil {
		return nil, err
	}
	res.Message = fmt.Sprintf("Import selesai: %d baru, %d update", summary.Create, summary.Update)
	return res, nil
}

// ─── Journal mappings ───────────────────────────────────────

func (a *Accounts) ListMappings(ctx context.Context, f domain.MappingFilter) ([]domain.JournalMapping, error) {
	f.Search = trim(f.Search)
	return a.Store.ListMappings(ctx, f)
}

func (a *Accounts) GetMapping(ctx context.Context, id string) (*domain.JournalMapping, error) {
	m, err := a.Store.GetMapping(ctx, id)
	if err == nil && m == nil {
		err = domain.NotFound("Journal mapping tidak ditemukan")
	}
	return m, err
}

const mappingHeaderMsg = "Akun mapping harus postable (bukan header)"

func (a *Accounts) validateMapping(ctx context.Context, tx domain.Store, m *domain.JournalMapping, lines []domain.MappingLine) error {
	m.EventCode = strings.ToUpper(trim(m.EventCode))
	m.Name = trim(m.Name)
	m.Module = strings.ToUpper(trim(m.Module))
	v := validate.New().
		Required("event_code", m.EventCode, "Event code wajib diisi").
		Required("name", m.Name, "Nama wajib diisi").
		OneOf("module", m.Module, domain.JournalModules, "Module tidak valid")
	if len(lines) == 0 {
		v.Add("lines", "Minimal satu baris mapping")
	}
	var accountIDs []string
	for i, l := range lines {
		key := fmt.Sprintf("lines.%d", i)
		if l.Side != domain.Debit && l.Side != domain.Credit {
			v.Add(key, "entry_side harus DEBIT atau CREDIT")
		}
		if trim(l.LineRole) == "" {
			v.Add(key, "Line role wajib diisi")
		}
		if !slices.Contains(domain.JournalAmountSources, l.AmountSource) {
			v.Add(key, "Amount source tidak valid")
		}
		accountIDs = append(accountIDs, l.AccountID)
	}
	if err := v.Err(); err != nil {
		return err
	}
	return checkAccounts(ctx, tx, accountIDs, "", mappingHeaderMsg)
}

// checkAccounts validates the distinct non-empty account ids. emptyMsg is
// returned when none is left ("" allows that, for mappings).
func checkAccounts(ctx context.Context, s domain.Store, accountIDs []string, emptyMsg, headerMsg string) error {
	unique := domain.UniqueIDs(accountIDs)
	if len(unique) == 0 {
		if emptyMsg == "" {
			return nil
		}
		return domain.Bad(emptyMsg)
	}
	rows, err := s.AccountChecks(ctx, unique)
	if err != nil {
		return err
	}
	return domain.CheckPostableAccounts(len(unique), rows, headerMsg)
}

func normalizeMappingLines(lines []domain.MappingLine) []domain.MappingLine {
	out := make([]domain.MappingLine, len(lines))
	for i, l := range lines {
		l.LineRole = strings.ToUpper(trim(l.LineRole))
		if l.SortOrder == 0 {
			l.SortOrder = (i + 1) * 10
		}
		out[i] = l
	}
	return out
}

func (a *Accounts) CreateMapping(ctx context.Context, userID string, m domain.JournalMapping, lines []domain.MappingLine) (*domain.JournalMapping, error) {
	lines = normalizeMappingLines(lines)
	m.ID = ids.New()
	err := a.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := a.validateMapping(ctx, tx, &m, lines); err != nil {
			return err
		}
		if err := tx.InsertMapping(ctx, m, userID); err != nil {
			return uniqueAs(err, "Mapping untuk event code ini sudah ada")
		}
		return tx.ReplaceMappingLines(ctx, m.ID, lines)
	})
	if err != nil {
		return nil, err
	}
	return a.Store.GetMapping(ctx, m.ID)
}

func (a *Accounts) UpdateMapping(ctx context.Context, userID string, m domain.JournalMapping, lines []domain.MappingLine) (*domain.JournalMapping, error) {
	if _, err := a.GetMapping(ctx, m.ID); err != nil {
		return nil, err
	}
	lines = normalizeMappingLines(lines)
	err := a.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := a.validateMapping(ctx, tx, &m, lines); err != nil {
			return err
		}
		if err := tx.UpdateMapping(ctx, m, userID); err != nil {
			return uniqueAs(err, "Mapping untuk event code ini sudah ada")
		}
		return tx.ReplaceMappingLines(ctx, m.ID, lines)
	})
	if err != nil {
		return nil, err
	}
	return a.Store.GetMapping(ctx, m.ID)
}

func (a *Accounts) DeleteMapping(ctx context.Context, userID, id string) error {
	if _, err := a.GetMapping(ctx, id); err != nil {
		return err
	}
	return a.Store.SoftDeleteMapping(ctx, id, userID)
}
