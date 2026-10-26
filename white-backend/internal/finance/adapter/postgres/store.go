// Package postgres implements the finance ports on top of sqlc queries.
package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

var _ domain.Store = (*Store)(nil)

type Store struct {
	pool *pgxpool.Pool
	tx   pgx.Tx
	q    *sqlc.Queries
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, q: sqlc.New(pool)}
}

// WithTx opens a transaction, or a savepoint inside one.
func (s *Store) WithTx(ctx context.Context, fn func(tx domain.Store) error) error {
	if s.tx != nil {
		sub, err := s.tx.Begin(ctx)
		if err != nil {
			return err
		}
		if err := fn(&Store{pool: s.pool, tx: sub, q: s.q.WithTx(sub)}); err != nil {
			_ = sub.Rollback(ctx)
			return err
		}
		return sub.Commit(ctx)
	}
	return pg.Tx(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(&Store{pool: s.pool, tx: tx, q: s.q.WithTx(tx)})
	})
}

func wrap(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return &domain.ErrUnique{Constraint: pgErr.ConstraintName}
	}
	return err
}

// ─── conversions ────────────────────────────────────────────

func sp(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func day(s string) time.Time {
	t, _ := time.Parse(time.DateOnly, s)
	return t
}

func dayp(s string) *time.Time {
	if s == "" {
		return nil
	}
	t := day(s)
	return &t
}

func ds(t time.Time) string { return t.Format(time.DateOnly) }

func dsp(t *time.Time) string {
	if t == nil {
		return ""
	}
	return ds(*t)
}

func moneyp(m domain.Money) *domain.Money {
	if m == 0 {
		return nil
	}
	return &m
}

func money(m *domain.Money) domain.Money {
	if m == nil {
		return 0
	}
	return *m
}

// one returns nil, nil for no rows.
func one[R, T any](r R, err error, conv func(R) T) (*T, error) {
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v := conv(r)
	return &v, nil
}

func mapAll[R, T any](rows []R, conv func(R) T) []T {
	out := make([]T, len(rows))
	for i, r := range rows {
		out[i] = conv(r)
	}
	return out
}

// ─── Account types and accounts ─────────────────────────────

func (s *Store) ListAccountTypes(ctx context.Context, includeInactive bool) ([]domain.AccountType, error) {
	rows, err := s.q.ListAccountTypes(ctx, includeInactive)
	return mapAll(rows, func(r sqlc.ListAccountTypesRow) domain.AccountType {
		return domain.AccountType{ID: r.ID, Code: r.Code, Name: r.Name, NormalBalance: r.NormalBalance, SortOrder: int(r.SortOrder), IsActive: r.IsActive, AccountCount: int(r.AccountCount)}
	}), err
}

func (s *Store) GetAccountType(ctx context.Context, id string) (*domain.AccountType, error) {
	r, err := s.q.GetAccountType(ctx, id)
	return one(r, err, func(r sqlc.GetAccountTypeRow) domain.AccountType {
		return domain.AccountType{ID: r.ID, Code: r.Code, Name: r.Name, NormalBalance: r.NormalBalance, SortOrder: int(r.SortOrder), IsActive: r.IsActive, AccountCount: int(r.AccountCount)}
	})
}

func (s *Store) UpsertAccountType(ctx context.Context, t domain.AccountType, by string) error {
	return wrap(s.q.UpsertAccountType(ctx, sqlc.UpsertAccountTypeParams{ID: t.ID, Code: t.Code, Name: t.Name, NormalBalance: t.NormalBalance, SortOrder: int32(t.SortOrder), IsActive: t.IsActive, By: sp(by)}))
}

func (s *Store) DeleteAccountType(ctx context.Context, id string) error {
	return s.q.DeleteAccountType(ctx, id)
}

func toAccount(r sqlc.ListAccountsRow) domain.Account {
	return domain.Account{
		ID: r.ID, Code: r.Code, Name: r.Name, ParentID: str(r.ParentID), AccountTypeID: r.AccountTypeID, TypeCode: r.TypeCode,
		TypeName: r.TypeName, NormalBalance: r.NormalBalance, Level: int(r.Level), IsPostable: r.IsPostable, IsContra: r.IsContra,
		IsCashBank: r.IsCashBank, CashFlowCategory: str(r.CashFlowCategory), Description: r.Description, IsActive: r.IsActive,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func (s *Store) ListAccounts(ctx context.Context, f domain.AccountFilter) ([]domain.Account, error) {
	rows, err := s.q.ListAccounts(ctx, sqlc.ListAccountsParams{AccountTypeID: sp(f.AccountTypeID), IsPostable: f.IsPostable, IsCashBank: f.IsCashBank, IsActive: f.IsActive, CashFlowCategory: sp(f.CashFlowCategory), Search: sp(f.Search)})
	return mapAll(rows, toAccount), err
}

func (s *Store) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	r, err := s.q.GetAccount(ctx, id)
	return one(r, err, func(r sqlc.GetAccountRow) domain.Account { return toAccount(sqlc.ListAccountsRow(r)) })
}

func (s *Store) AccountsByCode(ctx context.Context) (map[string]domain.Account, error) {
	rows, err := s.ListAccounts(ctx, domain.AccountFilter{})
	if err != nil {
		return nil, err
	}
	out := make(map[string]domain.Account, len(rows))
	for _, a := range rows {
		out[a.Code] = a
	}
	return out, nil
}

func (s *Store) InsertAccount(ctx context.Context, a domain.Account, by string) error {
	return wrap(s.q.InsertAccount(ctx, sqlc.InsertAccountParams{ID: a.ID, Code: a.Code, Name: a.Name, ParentID: sp(a.ParentID), AccountTypeID: a.AccountTypeID, Level: int32(a.Level), IsContra: a.IsContra, IsCashBank: a.IsCashBank, CashFlowCategory: sp(a.CashFlowCategory), Description: a.Description, IsActive: a.IsActive, By: sp(by)}))
}

func (s *Store) UpdateAccount(ctx context.Context, a domain.Account, by string) error {
	return wrap(s.q.UpdateAccount(ctx, sqlc.UpdateAccountParams{ID: a.ID, Code: a.Code, Name: a.Name, ParentID: sp(a.ParentID), AccountTypeID: a.AccountTypeID, Level: int32(a.Level), IsContra: a.IsContra, IsCashBank: a.IsCashBank, CashFlowCategory: sp(a.CashFlowCategory), Description: a.Description, IsActive: a.IsActive, By: sp(by)}))
}

func (s *Store) SoftDeleteAccount(ctx context.Context, id, by string) error {
	return s.q.SoftDeleteAccount(ctx, sqlc.SoftDeleteAccountParams{ID: id, By: sp(by)})
}

func (s *Store) CountChildren(ctx context.Context, id string) (int, error) {
	n, err := s.q.CountChildren(ctx, &id)
	return int(n), err
}

func (s *Store) RecomputePostable(ctx context.Context, id string) error {
	if id == "" {
		return nil
	}
	return s.q.RecomputePostableChain(ctx, id)
}

func (s *Store) RecomputeAllPostable(ctx context.Context) error { return s.q.RecomputeAllPostable(ctx) }

func (s *Store) AccountChecks(ctx context.Context, idList []string) ([]domain.AccountCheck, error) {
	rows, err := s.q.AccountChecks(ctx, idList)
	return mapAll(rows, func(r sqlc.AccountChecksRow) domain.AccountCheck {
		return domain.AccountCheck{ID: r.ID, IsPostable: r.IsPostable, Deleted: r.Deleted}
	}), err
}

// ─── Mappings ───────────────────────────────────────────────

func (s *Store) withLines(ctx context.Context, maps []domain.JournalMapping) ([]domain.JournalMapping, error) {
	if len(maps) == 0 {
		return maps, nil
	}
	idx := map[string]int{}
	mids := make([]string, len(maps))
	for i, m := range maps {
		idx[m.ID] = i
		mids[i] = m.ID
	}
	rows, err := s.q.MappingLines(ctx, mids)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		m := &maps[idx[r.MappingID]]
		m.Lines = append(m.Lines, domain.MappingLineView{
			MappingLine: domain.MappingLine{ID: r.ID, Side: r.EntrySide, AccountID: str(r.AccountID), AmountSource: r.AmountSource, SortOrder: int(r.SortOrder), IsRequired: r.IsRequired, LineRole: r.LineRole},
			AccountCode: r.AccountCode, AccountName: r.AccountName,
		})
	}
	return maps, nil
}

func toMapping(r sqlc.ListMappingsRow) domain.JournalMapping {
	return domain.JournalMapping{ID: r.ID, EventCode: r.EventCode, Name: r.Name, Description: r.Description, Module: r.Module, IsActive: r.IsActive, UpdatedAt: r.UpdatedAt}
}

func (s *Store) ListMappings(ctx context.Context, f domain.MappingFilter) ([]domain.JournalMapping, error) {
	rows, err := s.q.ListMappings(ctx, sqlc.ListMappingsParams{Module: sp(f.Module), IsActive: f.IsActive, Search: sp(f.Search)})
	if err != nil {
		return nil, err
	}
	return s.withLines(ctx, mapAll(rows, toMapping))
}

func (s *Store) oneMapping(ctx context.Context, m *domain.JournalMapping, err error) (*domain.JournalMapping, error) {
	if err != nil || m == nil {
		return nil, err
	}
	out, err := s.withLines(ctx, []domain.JournalMapping{*m})
	if err != nil {
		return nil, err
	}
	return &out[0], nil
}

func (s *Store) GetMapping(ctx context.Context, id string) (*domain.JournalMapping, error) {
	r, err := s.q.GetMapping(ctx, id)
	m, err := one(r, err, func(r sqlc.GetMappingRow) domain.JournalMapping { return toMapping(sqlc.ListMappingsRow(r)) })
	return s.oneMapping(ctx, m, err)
}

func (s *Store) GetActiveMapping(ctx context.Context, eventCode string) (*domain.JournalMapping, error) {
	r, err := s.q.GetActiveMapping(ctx, eventCode)
	m, err := one(r, err, func(r sqlc.GetActiveMappingRow) domain.JournalMapping { return toMapping(sqlc.ListMappingsRow(r)) })
	return s.oneMapping(ctx, m, err)
}

func (s *Store) InsertMapping(ctx context.Context, m domain.JournalMapping, by string) error {
	return wrap(s.q.InsertMapping(ctx, sqlc.InsertMappingParams{ID: m.ID, EventCode: m.EventCode, Name: m.Name, Description: m.Description, Module: m.Module, IsActive: m.IsActive, By: sp(by)}))
}

func (s *Store) UpdateMapping(ctx context.Context, m domain.JournalMapping, by string) error {
	return wrap(s.q.UpdateMapping(ctx, sqlc.UpdateMappingParams{ID: m.ID, EventCode: m.EventCode, Name: m.Name, Description: m.Description, Module: m.Module, IsActive: m.IsActive, By: sp(by)}))
}

func (s *Store) ReplaceMappingLines(ctx context.Context, mappingID string, lines []domain.MappingLine) error {
	if err := s.q.DeleteMappingLines(ctx, mappingID); err != nil {
		return err
	}
	for _, l := range lines {
		if err := s.q.InsertMappingLine(ctx, sqlc.InsertMappingLineParams{ID: ids.New(), MappingID: mappingID, EntrySide: l.Side, LineRole: l.LineRole, AccountID: sp(l.AccountID), AmountSource: l.AmountSource, SortOrder: int32(l.SortOrder), IsRequired: l.IsRequired}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) SoftDeleteMapping(ctx context.Context, id, by string) error {
	return s.q.SoftDeleteMapping(ctx, sqlc.SoftDeleteMappingParams{ID: id, By: sp(by)})
}

// ─── Fiscal ─────────────────────────────────────────────────

func toYear(r sqlc.ListFiscalYearsRow) domain.FiscalYear {
	return domain.FiscalYear{ID: r.ID, Code: r.Code, Name: r.Name, StartDate: ds(r.StartDate), EndDate: ds(r.EndDate), IsActive: r.IsActive, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func toPeriod(r sqlc.ListPeriodsRow) domain.FiscalPeriod {
	return domain.FiscalPeriod{ID: r.ID, FiscalYearID: r.FiscalYearID, PeriodNo: int(r.PeriodNo), Name: r.Name, StartDate: ds(r.StartDate), EndDate: ds(r.EndDate), Status: r.Status, YearCode: r.YearCode, YearName: r.YearName, YearActive: r.YearActive, PostedCount: int(r.PostedCount), DraftCount: int(r.DraftCount)}
}

func (s *Store) yearsWithPeriods(ctx context.Context, years []domain.FiscalYear) ([]domain.FiscalYear, error) {
	if len(years) == 0 {
		return years, nil
	}
	idx := map[string]int{}
	yids := make([]string, len(years))
	for i, y := range years {
		idx[y.ID] = i
		yids[i] = y.ID
	}
	rows, err := s.q.ListPeriods(ctx, sqlc.ListPeriodsParams{FiscalYearIds: yids})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		y := &years[idx[r.FiscalYearID]]
		y.Periods = append(y.Periods, toPeriod(r))
	}
	return years, nil
}

func (s *Store) ListFiscalYears(ctx context.Context, search string, active *bool) ([]domain.FiscalYear, error) {
	rows, err := s.q.ListFiscalYears(ctx, sqlc.ListFiscalYearsParams{IsActive: active, Search: sp(search)})
	if err != nil {
		return nil, err
	}
	return s.yearsWithPeriods(ctx, mapAll(rows, toYear))
}

func (s *Store) GetFiscalYear(ctx context.Context, id string) (*domain.FiscalYear, error) {
	r, err := s.q.GetFiscalYear(ctx, id)
	y, err := one(r, err, func(r sqlc.GetFiscalYearRow) domain.FiscalYear { return toYear(sqlc.ListFiscalYearsRow(r)) })
	if err != nil || y == nil {
		return nil, err
	}
	out, err := s.yearsWithPeriods(ctx, []domain.FiscalYear{*y})
	if err != nil {
		return nil, err
	}
	return &out[0], nil
}

func (s *Store) InsertFiscalYear(ctx context.Context, y domain.FiscalYear, by string) error {
	return wrap(s.q.InsertFiscalYear(ctx, sqlc.InsertFiscalYearParams{ID: y.ID, Code: y.Code, Name: y.Name, StartDate: day(y.StartDate), EndDate: day(y.EndDate), IsActive: y.IsActive, By: sp(by)}))
}

func (s *Store) UpdateFiscalYear(ctx context.Context, y domain.FiscalYear, by string) error {
	return wrap(s.q.UpdateFiscalYear(ctx, sqlc.UpdateFiscalYearParams{ID: y.ID, Code: y.Code, Name: y.Name, StartDate: day(y.StartDate), EndDate: day(y.EndDate), IsActive: y.IsActive, By: sp(by)}))
}

func (s *Store) SoftDeleteFiscalYear(ctx context.Context, id, by string) error {
	if err := s.q.DeleteYearPeriods(ctx, id); err != nil {
		return err
	}
	return s.q.SoftDeleteFiscalYear(ctx, sqlc.SoftDeleteFiscalYearParams{ID: id, By: sp(by)})
}

func (s *Store) UpsertPeriods(ctx context.Context, yearID string, periods []domain.PeriodInput) error {
	keep := make([]int32, len(periods))
	for i, p := range periods {
		keep[i] = int32(p.PeriodNo)
	}
	if err := s.q.DeleteUnusedPeriods(ctx, sqlc.DeleteUnusedPeriodsParams{FiscalYearID: yearID, Keep: keep}); err != nil {
		return err
	}
	for _, p := range periods {
		if err := s.q.UpsertPeriod(ctx, sqlc.UpsertPeriodParams{ID: ids.New(), FiscalYearID: yearID, PeriodNo: int32(p.PeriodNo), Name: p.Name, StartDate: day(p.StartDate), EndDate: day(p.EndDate), Status: p.Status}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UsedPeriodNos(ctx context.Context, yearID string) ([]int, error) {
	rows, err := s.q.UsedPeriodNos(ctx, yearID)
	return mapAll(rows, func(n int32) int { return int(n) }), err
}

func (s *Store) CountYearEntries(ctx context.Context, yearID string) (int, error) {
	n, err := s.q.CountYearEntries(ctx, yearID)
	return int(n), err
}

func (s *Store) PriorYearWithOpenPeriods(ctx context.Context, before, excludeID string) (*domain.YearOpenCount, error) {
	r, err := s.q.PriorYearWithOpenPeriods(ctx, sqlc.PriorYearWithOpenPeriodsParams{Before: day(before), ExcludeID: excludeID})
	return one(r, err, func(r sqlc.PriorYearWithOpenPeriodsRow) domain.YearOpenCount {
		return domain.YearOpenCount{ID: r.ID, Code: r.Code, Name: r.Name, EndDate: ds(r.EndDate), OpenCount: int(r.OpenCount)}
	})
}

func (s *Store) PriorFiscalYear(ctx context.Context, before string) (*domain.YearOpenCount, error) {
	r, err := s.q.PriorFiscalYear(ctx, day(before))
	return one(r, err, func(r sqlc.PriorFiscalYearRow) domain.YearOpenCount {
		return domain.YearOpenCount{ID: r.ID, Code: r.Code, Name: r.Name, EndDate: ds(r.EndDate), OpenCount: int(r.OpenCount)}
	})
}

func (s *Store) CoveringFiscalYearStart(ctx context.Context, date string) (string, error) {
	t, err := s.q.CoveringFiscalYearStart(ctx, day(date))
	if pg.IsNoRows(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return ds(t), nil
}

func (s *Store) ListPeriods(ctx context.Context, f domain.PeriodFilter) ([]domain.FiscalPeriod, error) {
	active := true
	p := sqlc.ListPeriodsParams{ActiveOnly: &active, Status: sp(f.Status), Search: sp(f.Search)}
	if f.FiscalYearID != "" {
		p.FiscalYearIds = []string{f.FiscalYearID}
	}
	rows, err := s.q.ListPeriods(ctx, p)
	return mapAll(rows, toPeriod), err
}

func (s *Store) GetPeriod(ctx context.Context, id string) (*domain.FiscalPeriod, error) {
	r, err := s.q.GetPeriod(ctx, id)
	return one(r, err, func(r sqlc.GetPeriodRow) domain.FiscalPeriod { return toPeriod(sqlc.ListPeriodsRow(r)) })
}

func (s *Store) SetPeriodStatus(ctx context.Context, idList []string, status string) error {
	return s.q.SetPeriodStatus(ctx, sqlc.SetPeriodStatusParams{Status: status, Ids: idList})
}

func shortPeriod(r sqlc.OpenPeriodForRow) domain.FiscalPeriod {
	return domain.FiscalPeriod{ID: r.ID, FiscalYearID: r.FiscalYearID, PeriodNo: int(r.PeriodNo), Name: r.Name, StartDate: ds(r.StartDate), EndDate: ds(r.EndDate), Status: r.Status, YearCode: r.YearCode, YearName: r.YearName, YearActive: r.YearActive}
}

func (s *Store) OpenPeriodFor(ctx context.Context, date string) (*domain.FiscalPeriod, error) {
	r, err := s.q.OpenPeriodFor(ctx, day(date))
	return one(r, err, shortPeriod)
}

func (s *Store) CoveringPeriod(ctx context.Context, date string) (*domain.FiscalPeriod, error) {
	r, err := s.q.CoveringPeriod(ctx, day(date))
	return one(r, err, func(r sqlc.CoveringPeriodRow) domain.FiscalPeriod { return shortPeriod(sqlc.OpenPeriodForRow(r)) })
}

func (s *Store) PreviousOpenPeriods(ctx context.Context, yearID string, periodNo int) ([]domain.FiscalPeriod, error) {
	rows, err := s.q.PreviousOpenPeriods(ctx, sqlc.PreviousOpenPeriodsParams{FiscalYearID: yearID, PeriodNo: int32(periodNo)})
	return mapAll(rows, func(r sqlc.PreviousOpenPeriodsRow) domain.FiscalPeriod {
		return domain.FiscalPeriod{ID: r.ID, FiscalYearID: r.FiscalYearID, PeriodNo: int(r.PeriodNo), Name: r.Name, StartDate: ds(r.StartDate), EndDate: ds(r.EndDate), Status: r.Status}
	}), err
}

func (s *Store) DraftEntries(ctx context.Context, periodID string, limit int) ([]domain.JournalEntry, error) {
	rows, err := s.q.DraftEntries(ctx, sqlc.DraftEntriesParams{FiscalPeriodID: periodID, Limit: int32(limit)})
	return mapAll(rows, func(r sqlc.DraftEntriesRow) domain.JournalEntry {
		return domain.JournalEntry{ID: r.ID, EntryNo: r.EntryNo, EntryDate: ds(r.EntryDate), Description: r.Description}
	}), err
}

// ─── Journals ───────────────────────────────────────────────

func (s *Store) NextSequence(ctx context.Context, prefix string) (int, error) {
	n, err := s.q.NextSequence(ctx, prefix)
	return int(n), err
}

func (s *Store) InsertJournalEntry(ctx context.Context, e domain.JournalEntry) error {
	return wrap(s.q.InsertJournalEntry(ctx, sqlc.InsertJournalEntryParams{
		ID: e.ID, EntryNo: e.EntryNo, EntryDate: day(e.EntryDate), Description: e.Description, FiscalPeriodID: e.FiscalPeriodID,
		Status: e.Status, IsRecon: e.IsRecon, EntryType: e.EntryType, SourceModule: e.SourceModule, SourceEventCode: e.SourceEventCode,
		SourceDocumentType: e.SourceDocumentType, SourceDocumentID: sp(e.SourceDocumentID), SourceFiscalYearID: sp(e.SourceFiscalYearID), By: sp(e.CreatedBy),
	}))
}

func (s *Store) UpdateJournalHeader(ctx context.Context, e domain.JournalEntry, by string) error {
	return s.q.UpdateJournalHeader(ctx, sqlc.UpdateJournalHeaderParams{ID: e.ID, EntryDate: day(e.EntryDate), Description: e.Description, FiscalPeriodID: e.FiscalPeriodID, Status: e.Status, IsRecon: e.IsRecon, By: sp(by)})
}

func (s *Store) ReplaceJournalLines(ctx context.Context, entryID string, lines []domain.Line) error {
	if err := s.q.DeleteJournalLines(ctx, entryID); err != nil {
		return err
	}
	for i, l := range lines {
		sort := l.SortOrder
		if sort == 0 {
			sort = (i + 1) * 10
		}
		if err := s.q.InsertJournalLine(ctx, sqlc.InsertJournalLineParams{ID: ids.New(), EntryID: entryID, AccountID: l.AccountID, EntrySide: l.Side, Amount: l.Amount, Memo: l.Memo, SortOrder: int32(sort)}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) MarkJournalPosted(ctx context.Context, id, by string) error {
	return s.q.MarkJournalPosted(ctx, sqlc.MarkJournalPostedParams{ID: id, By: sp(by)})
}

func (s *Store) SoftDeleteJournalEntry(ctx context.Context, id, by string) error {
	return s.q.SoftDeleteJournalEntry(ctx, sqlc.SoftDeleteJournalEntryParams{ID: id, By: sp(by)})
}

func toEntry(r sqlc.GetJournalEntryRow) domain.JournalEntry {
	return domain.JournalEntry{
		ID: r.ID, EntryNo: r.EntryNo, EntryDate: ds(r.EntryDate), Description: r.Description, FiscalPeriodID: r.FiscalPeriodID,
		PeriodName: r.PeriodName, YearCode: r.YearCode, EntryType: r.EntryType, Status: r.Status, IsRecon: r.IsRecon,
		PostedAt: r.PostedAt, PostedBy: str(r.PostedBy), SourceModule: r.SourceModule, SourceEventCode: r.SourceEventCode,
		SourceDocumentType: r.SourceDocumentType, SourceDocumentID: str(r.SourceDocumentID), SourceFiscalYearID: str(r.SourceFiscalYearID),
		CreatedBy: str(r.CreatedBy), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func (s *Store) entriesWithLines(ctx context.Context, entries []domain.JournalEntry) ([]domain.JournalEntry, error) {
	if len(entries) == 0 {
		return entries, nil
	}
	idx := map[string]int{}
	eids := make([]string, len(entries))
	for i, e := range entries {
		idx[e.ID] = i
		eids[i] = e.ID
	}
	rows, err := s.q.JournalLines(ctx, eids)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		e := &entries[idx[r.EntryID]]
		e.Lines = append(e.Lines, domain.JournalLine{ID: r.ID, AccountID: r.AccountID, AccountCode: r.AccountCode, AccountName: r.AccountName, Side: r.EntrySide, Amount: r.Amount, Memo: r.Memo, SortOrder: int(r.SortOrder)})
	}
	return entries, nil
}

func (s *Store) GetJournalEntry(ctx context.Context, id string) (*domain.JournalEntry, error) {
	r, err := s.q.GetJournalEntry(ctx, id)
	e, err := one(r, err, toEntry)
	if err != nil || e == nil {
		return nil, err
	}
	out, err := s.entriesWithLines(ctx, []domain.JournalEntry{*e})
	if err != nil {
		return nil, err
	}
	return &out[0], nil
}

func (s *Store) ListJournalEntries(ctx context.Context, f domain.JournalFilter) ([]domain.JournalEntry, int, error) {
	rows, err := s.q.ListJournalEntries(ctx, sqlc.ListJournalEntriesParams{
		Status: sp(f.Status), EntryType: sp(f.EntryType), DateFrom: dayp(f.DateFrom), DateTo: dayp(f.DateTo), Search: sp(f.Search),
		SourceDocumentType: sp(f.SourceDocumentType), AccountID: sp(f.AccountID), Lim: int32(f.Limit), Off: int32(f.Offset),
	})
	if err != nil {
		return nil, 0, err
	}
	total := 0
	entries := make([]domain.JournalEntry, len(rows))
	for i, r := range rows {
		total = int(r.Total)
		entries[i] = toEntry(sqlc.GetJournalEntryRow{ID: r.ID, EntryNo: r.EntryNo, EntryDate: r.EntryDate, Description: r.Description, FiscalPeriodID: r.FiscalPeriodID, PeriodName: r.PeriodName, YearCode: r.YearCode, EntryType: r.EntryType, Status: r.Status, IsRecon: r.IsRecon, PostedAt: r.PostedAt, PostedBy: r.PostedBy, SourceModule: r.SourceModule, SourceEventCode: r.SourceEventCode, SourceDocumentType: r.SourceDocumentType, SourceDocumentID: r.SourceDocumentID, SourceFiscalYearID: r.SourceFiscalYearID, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
	}
	entries, err = s.entriesWithLines(ctx, entries)
	return entries, total, err
}

func (s *Store) entryByID(ctx context.Context, id string, err error) (*domain.JournalEntry, error) {
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.GetJournalEntry(ctx, id)
}

func (s *Store) FindEntryBySource(ctx context.Context, eventCode, documentID string) (*domain.JournalEntry, error) {
	id, err := s.q.FindEntryBySource(ctx, sqlc.FindEntryBySourceParams{SourceEventCode: eventCode, SourceDocumentID: &documentID})
	return s.entryByID(ctx, id, err)
}

func (s *Store) OpeningEntry(ctx context.Context, yearID string) (*domain.JournalEntry, error) {
	id, err := s.q.OpeningEntryID(ctx, &yearID)
	return s.entryByID(ctx, id, err)
}

func (s *Store) CountEntries(ctx context.Context, from, to string) (int, int, error) {
	r, err := s.q.CountEntries(ctx, sqlc.CountEntriesParams{DateFrom: day(from), DateTo: day(to)})
	return int(r.Posted), int(r.Drafts), err
}

func toLog(r sqlc.FinancePostingLog) domain.PostingLog {
	return domain.PostingLog{ID: r.ID, EventCode: r.EventCode, DocumentType: r.DocumentType, DocumentID: r.DocumentID, DocumentLabel: r.DocumentLabel, Status: r.Status, Reason: r.Reason, EntryID: str(r.EntryID), Request: r.Request, Attempts: int(r.Attempts), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (s *Store) UpsertPostingLog(ctx context.Context, l domain.PostingLog) (*domain.PostingLog, error) {
	r, err := s.q.UpsertPostingLog(ctx, sqlc.UpsertPostingLogParams{ID: ids.New(), EventCode: l.EventCode, DocumentType: l.DocumentType, DocumentID: l.DocumentID, DocumentLabel: l.DocumentLabel, Status: l.Status, Reason: l.Reason, EntryID: sp(l.EntryID), Request: l.Request})
	return one(r, err, toLog)
}

func (s *Store) GetPostingLog(ctx context.Context, id string) (*domain.PostingLog, error) {
	r, err := s.q.GetPostingLog(ctx, id)
	return one(r, err, toLog)
}

func (s *Store) ListPostingLogs(ctx context.Context, status, eventCode string, limit int) ([]domain.PostingLog, error) {
	rows, err := s.q.ListPostingLogs(ctx, sqlc.ListPostingLogsParams{Limit: int32(limit), Status: sp(status), EventCode: sp(eventCode)})
	return mapAll(rows, toLog), err
}

// ─── Balances ───────────────────────────────────────────────

func (s *Store) AccountBalances(ctx context.Context, w domain.BalanceWindow) ([]domain.AccountBalance, error) {
	rows, err := s.q.AccountBalances(ctx, sqlc.AccountBalancesParams{AsOf: dayp(w.AsOf), DateFrom: dayp(w.From), DateTo: dayp(w.To)})
	return mapAll(rows, func(r sqlc.AccountBalancesRow) domain.AccountBalance {
		normal := domain.AsNormal(r.NormalBalance)
		return domain.AccountBalance{ID: r.ID, Code: r.Code, Name: r.Name, TypeCode: r.TypeCode, NormalBalance: normal, IsContra: r.IsContra, IsCashBank: r.IsCashBank, CashFlowCategory: r.CashFlowCategory, Level: int(r.Level), Debit: r.Debit, Credit: r.Credit, Balance: domain.Balance(r.Debit, r.Credit, normal, r.IsContra), MovementCount: int(r.MovementCount)}
	}), err
}

func (s *Store) YearAccountTotals(ctx context.Context, yearID string) ([]domain.AccountBalance, error) {
	rows, err := s.q.YearAccountTotals(ctx, yearID)
	return mapAll(rows, func(r sqlc.YearAccountTotalsRow) domain.AccountBalance {
		return domain.AccountBalance{ID: r.ID, Code: r.Code, Name: r.Name, TypeCode: r.TypeCode, NormalBalance: domain.AsNormal(r.NormalBalance), IsContra: r.IsContra, Debit: r.Debit, Credit: r.Credit}
	}), err
}

func (s *Store) AccountTotalsBefore(ctx context.Context, accountID, date string) (domain.Money, domain.Money, error) {
	r, err := s.q.AccountTotalsBefore(ctx, sqlc.AccountTotalsBeforeParams{AccountID: accountID, Before: day(date)})
	return r.Debit, r.Credit, err
}

func (s *Store) LedgerLines(ctx context.Context, accountID, from, to string) ([]domain.LedgerLine, error) {
	rows, err := s.q.LedgerLines(ctx, sqlc.LedgerLinesParams{AccountID: accountID, DateFrom: dayp(from), DateTo: dayp(to)})
	return mapAll(rows, func(r sqlc.LedgerLinesRow) domain.LedgerLine {
		return domain.LedgerLine{LineID: r.ID, EntryID: r.EntryID, EntryNo: r.EntryNo, EntryDate: ds(r.EntryDate), EntryType: r.EntryType, Description: r.Description, Memo: r.Memo, Side: r.EntrySide, Amount: r.Amount}
	}), err
}

func (s *Store) MonthlyPnL(ctx context.Context, from, to string) ([]domain.MonthPnL, error) {
	rows, err := s.q.MonthlyPnL(ctx, sqlc.MonthlyPnLParams{DateFrom: day(from), DateTo: day(to)})
	return mapAll(rows, func(r sqlc.MonthlyPnLRow) domain.MonthPnL {
		return domain.MonthPnL{Month: r.Month, Revenue: r.Revenue, Expense: r.Expense}
	}), err
}

// ─── Invoices and payments ──────────────────────────────────

func toInvoice(r sqlc.GetInvoiceRow) domain.Invoice {
	return domain.Invoice{
		ID: r.ID, Kind: r.Kind, InvoiceNo: r.InvoiceNo, InvoiceDate: ds(r.InvoiceDate), DueDate: dsp(r.DueDate), PartyName: r.PartyName,
		Reference: r.Reference, AccountID: str(r.AccountID), AccountName: r.AccountName, Currency: r.Currency, Subtotal: r.Subtotal,
		TaxAmount: r.TaxAmount, TotalAmount: r.TotalAmount, Status: r.Status, Description: r.Description, SalesInvoiceID: str(r.SalesInvoiceID),
		ClientID: str(r.ClientID), DealID: str(r.DealID), Allocated: r.Allocated, JournalEntryID: str(r.JournalEntryID), PostedAt: r.PostedAt,
		CreatedBy: str(r.CreatedBy), CreatedAt: r.CreatedAt,
	}
}

func (s *Store) ListInvoices(ctx context.Context, f domain.InvoiceFilter) ([]domain.Invoice, int, error) {
	rows, err := s.q.ListInvoices(ctx, sqlc.ListInvoicesParams{Kind: f.Kind, Status: sp(f.Status), Search: sp(f.Search), Lim: int32(f.Limit), Off: int32(f.Offset)})
	if err != nil {
		return nil, 0, err
	}
	total := 0
	out := make([]domain.Invoice, len(rows))
	for i, r := range rows {
		total = int(r.Total)
		out[i] = toInvoice(sqlc.GetInvoiceRow{ID: r.ID, Kind: r.Kind, InvoiceNo: r.InvoiceNo, InvoiceDate: r.InvoiceDate, DueDate: r.DueDate, PartyName: r.PartyName, Reference: r.Reference, AccountID: r.AccountID, AccountName: r.AccountName, Currency: r.Currency, Subtotal: r.Subtotal, TaxAmount: r.TaxAmount, TotalAmount: r.TotalAmount, Status: r.Status, Description: r.Description, SalesInvoiceID: r.SalesInvoiceID, ClientID: r.ClientID, DealID: r.DealID, JournalEntryID: r.JournalEntryID, PostedAt: r.PostedAt, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, Allocated: r.Allocated})
	}
	return out, total, nil
}

func (s *Store) GetInvoice(ctx context.Context, id string) (*domain.Invoice, error) {
	r, err := s.q.GetInvoice(ctx, id)
	return one(r, err, toInvoice)
}

func (s *Store) GetInvoiceBySalesID(ctx context.Context, salesInvoiceID string) (*domain.Invoice, error) {
	id, err := s.q.GetInvoiceIDBySales(ctx, &salesInvoiceID)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.GetInvoice(ctx, id)
}

func (s *Store) InsertInvoice(ctx context.Context, i domain.Invoice) error {
	return wrap(s.q.InsertInvoice(ctx, sqlc.InsertInvoiceParams{
		ID: i.ID, Kind: i.Kind, InvoiceNo: i.InvoiceNo, InvoiceDate: day(i.InvoiceDate), DueDate: dayp(i.DueDate), PartyName: i.PartyName,
		Reference: i.Reference, AccountID: sp(i.AccountID), Currency: i.Currency, Subtotal: i.Subtotal, TaxAmount: i.TaxAmount,
		TotalAmount: i.TotalAmount, Status: i.Status, Description: i.Description, SalesInvoiceID: sp(i.SalesInvoiceID),
		ClientID: sp(i.ClientID), DealID: sp(i.DealID), By: sp(i.CreatedBy),
	}))
}

func (s *Store) UpdateInvoice(ctx context.Context, i domain.Invoice, by string) error {
	return s.q.UpdateInvoice(ctx, sqlc.UpdateInvoiceParams{ID: i.ID, InvoiceDate: day(i.InvoiceDate), DueDate: dayp(i.DueDate), PartyName: i.PartyName, Reference: i.Reference, AccountID: sp(i.AccountID), Subtotal: i.Subtotal, TaxAmount: i.TaxAmount, TotalAmount: i.TotalAmount, Description: i.Description, ClientID: sp(i.ClientID), By: sp(by)})
}

func (s *Store) SoftDeleteInvoice(ctx context.Context, id, by string) error {
	return s.q.SoftDeleteInvoice(ctx, sqlc.SoftDeleteInvoiceParams{ID: id, By: sp(by)})
}

func (s *Store) MarkInvoicePosted(ctx context.Context, id, by string, at time.Time) error {
	return s.q.MarkInvoicePosted(ctx, sqlc.MarkInvoicePostedParams{ID: id, PostedAt: &at, By: sp(by)})
}

func (s *Store) SetInvoiceJournal(ctx context.Context, id, entryID string) error {
	return s.q.SetInvoiceJournal(ctx, sqlc.SetInvoiceJournalParams{ID: id, JournalEntryID: sp(entryID)})
}

func toPayment(r sqlc.GetPaymentRow) domain.Payment {
	return domain.Payment{
		ID: r.ID, Kind: r.Kind, PaymentNo: r.PaymentNo, PaymentDate: ds(r.PaymentDate), Amount: r.Amount, Method: r.Method,
		CashAccountID: str(r.CashAccountID), CashAccountName: r.CashAccountName, Reference: r.ReferenceNumber, Notes: r.Notes,
		Status: r.Status, PartyName: r.PartyName, SalesPaymentID: str(r.SalesPaymentID), JournalEntryID: str(r.JournalEntryID),
		VoidReason: r.VoidReason, VoidedAt: r.VoidedAt, CreatedBy: str(r.CreatedBy), CreatedAt: r.CreatedAt,
	}
}

func (s *Store) withAllocations(ctx context.Context, ps []domain.Payment) ([]domain.Payment, error) {
	if len(ps) == 0 {
		return ps, nil
	}
	idx := map[string]int{}
	pids := make([]string, len(ps))
	for i, p := range ps {
		idx[p.ID] = i
		pids[i] = p.ID
	}
	rows, err := s.q.PaymentAllocations(ctx, pids)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		p := &ps[idx[r.PaymentID]]
		p.Allocations = append(p.Allocations, domain.Allocation{InvoiceID: r.InvoiceID, InvoiceNo: r.InvoiceNo, Amount: r.Amount})
	}
	return ps, nil
}

func (s *Store) ListPayments(ctx context.Context, f domain.PaymentFilter) ([]domain.Payment, int, error) {
	rows, err := s.q.ListPayments(ctx, sqlc.ListPaymentsParams{Kind: f.Kind, Search: sp(f.Search), Lim: int32(f.Limit), Off: int32(f.Offset)})
	if err != nil {
		return nil, 0, err
	}
	total := 0
	out := make([]domain.Payment, len(rows))
	for i, r := range rows {
		total = int(r.Total)
		out[i] = toPayment(sqlc.GetPaymentRow{ID: r.ID, Kind: r.Kind, PaymentNo: r.PaymentNo, PaymentDate: r.PaymentDate, Amount: r.Amount, Method: r.Method, CashAccountID: r.CashAccountID, CashAccountName: r.CashAccountName, ReferenceNumber: r.ReferenceNumber, Notes: r.Notes, Status: r.Status, PartyName: r.PartyName, SalesPaymentID: r.SalesPaymentID, JournalEntryID: r.JournalEntryID, VoidReason: r.VoidReason, VoidedAt: r.VoidedAt, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt})
	}
	out, err = s.withAllocations(ctx, out)
	return out, total, err
}

func (s *Store) GetPayment(ctx context.Context, id string) (*domain.Payment, error) {
	r, err := s.q.GetPayment(ctx, id)
	p, err := one(r, err, toPayment)
	if err != nil || p == nil {
		return nil, err
	}
	out, err := s.withAllocations(ctx, []domain.Payment{*p})
	if err != nil {
		return nil, err
	}
	return &out[0], nil
}

func (s *Store) GetPaymentBySalesID(ctx context.Context, salesPaymentID string) (*domain.Payment, error) {
	id, err := s.q.GetPaymentIDBySales(ctx, &salesPaymentID)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.GetPayment(ctx, id)
}

func (s *Store) InsertPayment(ctx context.Context, p domain.Payment) error {
	if err := s.q.InsertPayment(ctx, sqlc.InsertPaymentParams{
		ID: p.ID, Kind: p.Kind, PaymentNo: p.PaymentNo, PaymentDate: day(p.PaymentDate), Amount: p.Amount, Method: p.Method,
		CashAccountID: sp(p.CashAccountID), ReferenceNumber: p.Reference, Notes: p.Notes, PartyName: p.PartyName,
		SalesPaymentID: sp(p.SalesPaymentID), By: sp(p.CreatedBy),
	}); err != nil {
		return wrap(err)
	}
	for _, a := range p.Allocations {
		if err := s.q.InsertAllocation(ctx, sqlc.InsertAllocationParams{ID: ids.New(), PaymentID: p.ID, InvoiceID: a.InvoiceID, Amount: a.Amount}); err != nil {
			return wrap(err)
		}
	}
	return nil
}

func (s *Store) VoidPayment(ctx context.Context, id, reason, by string) error {
	return s.q.VoidPayment(ctx, sqlc.VoidPaymentParams{ID: id, VoidReason: reason, By: sp(by)})
}

func (s *Store) SetPaymentJournal(ctx context.Context, id, entryID string) error {
	return s.q.SetPaymentJournal(ctx, sqlc.SetPaymentJournalParams{ID: id, JournalEntryID: sp(entryID)})
}

func (s *Store) SubsidiaryParties(ctx context.Context, kind string) ([]domain.SubsidiaryParty, error) {
	rows, err := s.q.SubsidiaryParties(ctx, kind)
	return mapAll(rows, func(r sqlc.SubsidiaryPartiesRow) domain.SubsidiaryParty {
		return domain.SubsidiaryParty{PartyName: r.PartyName, Outstanding: r.Outstanding, InvoiceCount: int(r.InvoiceCount)}
	}), err
}

func (s *Store) SubsidiaryMoves(ctx context.Context, kind, party, from, to string) ([]domain.SubsidiaryMove, error) {
	invs, err := s.q.SubsidiaryInvoices(ctx, sqlc.SubsidiaryInvoicesParams{Kind: kind, PartyName: party, DateFrom: dayp(from), DateTo: dayp(to)})
	if err != nil {
		return nil, err
	}
	allocs, err := s.q.SubsidiaryAllocations(ctx, sqlc.SubsidiaryAllocationsParams{Kind: kind, PartyName: party, DateFrom: dayp(from), DateTo: dayp(to)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.SubsidiaryMove, 0, len(invs)+len(allocs))
	for _, i := range invs {
		out = append(out, domain.SubsidiaryMove{ID: i.ID, DocNo: i.InvoiceNo, Date: ds(i.InvoiceDate), Amount: i.TotalAmount, Description: i.Description, IsInvoice: true})
	}
	for _, a := range allocs {
		out = append(out, domain.SubsidiaryMove{ID: a.ID, DocNo: a.PaymentNo, Date: ds(a.PaymentDate), Amount: a.Amount, Description: a.InvoiceNo})
	}
	return out, nil
}

func (s *Store) SubsidiaryOpening(ctx context.Context, kind, party, before string) (domain.Money, error) {
	return s.q.SubsidiaryOpening(ctx, sqlc.SubsidiaryOpeningParams{Kind: kind, PartyName: party, Before: day(before)})
}

// ─── Reimbursement ──────────────────────────────────────────

func toPolicy(r sqlc.ListPoliciesRow) domain.Policy {
	return domain.Policy{Category: r.Category, Label: r.Label, MonthlyLimit: money(r.MonthlyLimit), PerClaimLimit: money(r.PerClaimLimit), ExpenseAccountID: str(r.ExpenseAccountID), AutoApprove: r.AutoApprove, IsActive: r.IsActive}
}

func (s *Store) ListPolicies(ctx context.Context) ([]domain.Policy, error) {
	rows, err := s.q.ListPolicies(ctx)
	return mapAll(rows, toPolicy), err
}

func (s *Store) GetPolicy(ctx context.Context, category string) (*domain.Policy, error) {
	r, err := s.q.GetPolicy(ctx, category)
	return one(r, err, func(r sqlc.GetPolicyRow) domain.Policy { return toPolicy(sqlc.ListPoliciesRow(r)) })
}

func (s *Store) UpsertPolicy(ctx context.Context, p domain.Policy, by string) error {
	return s.q.UpsertPolicy(ctx, sqlc.UpsertPolicyParams{Category: p.Category, Label: p.Label, MonthlyLimit: moneyp(p.MonthlyLimit), PerClaimLimit: moneyp(p.PerClaimLimit), ExpenseAccountID: sp(p.ExpenseAccountID), AutoApprove: p.AutoApprove, IsActive: p.IsActive, By: sp(by)})
}

func (s *Store) InsertClaim(ctx context.Context, c domain.Claim) error {
	return wrap(s.q.InsertClaim(ctx, sqlc.InsertClaimParams{
		ID: c.ID, ClaimNo: c.ClaimNo, EmployeeID: c.EmployeeID, EmployeeUserID: sp(c.EmployeeUserID), EmployeeName: c.EmployeeName,
		DepartmentName: c.DepartmentName, ManagerEmployeeID: sp(c.ManagerEmployeeID), ManagerUserID: sp(c.ManagerUserID),
		Category: c.Category, TransactionDate: day(c.TransactionDate), Amount: c.Amount, Description: c.Description,
		ClientID: sp(c.ClientID), Project: c.Project, Status: string(c.Status), By: sp(c.CreatedBy),
	}))
}

func (s *Store) UpdateClaim(ctx context.Context, c domain.Claim) error {
	return s.q.UpdateClaim(ctx, sqlc.UpdateClaimParams{
		ID: c.ID, EmployeeName: c.EmployeeName, DepartmentName: c.DepartmentName, ManagerEmployeeID: sp(c.ManagerEmployeeID),
		ManagerUserID: sp(c.ManagerUserID), Category: c.Category, TransactionDate: day(c.TransactionDate), Amount: c.Amount,
		Description: c.Description, ClientID: sp(c.ClientID), Project: c.Project, Status: string(c.Status), OverLimit: c.OverLimit,
		LimitNote: c.LimitNote, SubmittedAt: c.SubmittedAt, ManagerApprovedAt: c.ManagerApprovedAt, ManagerApprovedBy: sp(c.ManagerApprovedBy),
		FinanceApprovedAt: c.FinanceApprovedAt, FinanceApprovedBy: sp(c.FinanceApprovedBy), RejectedAt: c.RejectedAt,
		RejectedBy: sp(c.RejectedBy), RejectedStage: c.RejectedStage, RejectReason: c.RejectReason, CancelledAt: c.CancelledAt,
		PaidAt: c.PaidAt, PaidBy: sp(c.PaidBy), PaymentBatchID: sp(c.PaymentBatchID), By: sp(c.UpdatedBy),
	})
}

func toClaim(r sqlc.FinanceReimbursementClaim) domain.Claim {
	return domain.Claim{
		ID: r.ID, ClaimNo: r.ClaimNo, EmployeeID: r.EmployeeID, EmployeeUserID: str(r.EmployeeUserID), EmployeeName: r.EmployeeName,
		DepartmentName: r.DepartmentName, ManagerEmployeeID: str(r.ManagerEmployeeID), ManagerUserID: str(r.ManagerUserID),
		Category: r.Category, TransactionDate: ds(r.TransactionDate), Amount: r.Amount, Description: r.Description,
		ClientID: str(r.ClientID), Project: r.Project, Status: domain.ClaimStatus(r.Status), OverLimit: r.OverLimit, LimitNote: r.LimitNote,
		SubmittedAt: r.SubmittedAt, ManagerApprovedAt: r.ManagerApprovedAt, ManagerApprovedBy: str(r.ManagerApprovedBy),
		FinanceApprovedAt: r.FinanceApprovedAt, FinanceApprovedBy: str(r.FinanceApprovedBy), RejectedAt: r.RejectedAt,
		RejectedBy: str(r.RejectedBy), RejectedStage: r.RejectedStage, RejectReason: r.RejectReason, CancelledAt: r.CancelledAt,
		PaidAt: r.PaidAt, PaidBy: str(r.PaidBy), PaymentBatchID: str(r.PaymentBatchID), LastRemindedAt: r.LastRemindedAt,
		CreatedBy: str(r.CreatedBy), CreatedAt: r.CreatedAt, UpdatedBy: str(r.UpdatedBy), UpdatedAt: r.UpdatedAt,
	}
}

func toReceipt(r sqlc.FinanceReimbursementReceipt) domain.Receipt {
	return domain.Receipt{ID: r.ID, ClaimID: r.ClaimID, StorageKey: r.StorageKey, Filename: r.Filename, ContentType: r.ContentType, SizeBytes: r.SizeBytes, CreatedBy: str(r.CreatedBy), CreatedAt: r.CreatedAt}
}

func (s *Store) withReceipts(ctx context.Context, cs []domain.Claim) ([]domain.Claim, error) {
	if len(cs) == 0 {
		return cs, nil
	}
	idx := map[string]int{}
	cids := make([]string, len(cs))
	for i, c := range cs {
		idx[c.ID] = i
		cids[i] = c.ID
	}
	rows, err := s.q.ClaimReceipts(ctx, cids)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		c := &cs[idx[r.ClaimID]]
		c.Receipts = append(c.Receipts, toReceipt(r))
	}
	return cs, nil
}

func (s *Store) GetClaim(ctx context.Context, id string) (*domain.Claim, error) {
	r, err := s.q.GetClaim(ctx, id)
	c, err := one(r, err, toClaim)
	if err != nil || c == nil {
		return nil, err
	}
	out, err := s.withReceipts(ctx, []domain.Claim{*c})
	if err != nil {
		return nil, err
	}
	events, err := s.q.ClaimEvents(ctx, id)
	if err != nil {
		return nil, err
	}
	out[0].History = mapAll(events, func(e sqlc.FinanceReimbursementEvent) domain.ClaimEvent {
		return domain.ClaimEvent{ID: e.ID, ClaimID: e.ClaimID, Action: e.Action, ActorID: str(e.ActorID), ActorName: e.ActorName, Note: e.Note, CreatedAt: e.CreatedAt}
	})
	return &out[0], nil
}

func (s *Store) ListClaims(ctx context.Context, f domain.ClaimFilter) ([]domain.Claim, int, domain.Money, error) {
	var statuses []string
	for _, st := range f.Statuses {
		statuses = append(statuses, string(st))
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 500
	}
	rows, err := s.q.ListClaims(ctx, sqlc.ListClaimsParams{
		Statuses: statuses, Category: sp(f.Category), EmployeeID: sp(f.EmployeeID), EmployeeUserID: sp(f.EmployeeUserID),
		ManagerUserID: sp(f.ManagerUserID), MonthFrom: dayp(f.MonthFrom), MonthTo: dayp(f.MonthTo), SubmittedBefore: f.SubmittedBefore,
		RemindedBefore: f.RemindedBefore, Search: sp(f.Search), Lim: int32(limit), Off: int32(f.Offset),
	})
	if err != nil {
		return nil, 0, 0, err
	}
	var total int
	var sum domain.Money
	out := make([]domain.Claim, len(rows))
	for i, r := range rows {
		total, sum = int(r.Total), r.TotalAmount
		out[i] = toClaim(sqlc.FinanceReimbursementClaim{ID: r.ID, ClaimNo: r.ClaimNo, EmployeeID: r.EmployeeID, EmployeeUserID: r.EmployeeUserID, EmployeeName: r.EmployeeName, DepartmentName: r.DepartmentName, ManagerEmployeeID: r.ManagerEmployeeID, ManagerUserID: r.ManagerUserID, Category: r.Category, TransactionDate: r.TransactionDate, Amount: r.Amount, Description: r.Description, ClientID: r.ClientID, Project: r.Project, Status: r.Status, OverLimit: r.OverLimit, LimitNote: r.LimitNote, SubmittedAt: r.SubmittedAt, ManagerApprovedAt: r.ManagerApprovedAt, ManagerApprovedBy: r.ManagerApprovedBy, FinanceApprovedAt: r.FinanceApprovedAt, FinanceApprovedBy: r.FinanceApprovedBy, RejectedAt: r.RejectedAt, RejectedBy: r.RejectedBy, RejectedStage: r.RejectedStage, RejectReason: r.RejectReason, CancelledAt: r.CancelledAt, PaidAt: r.PaidAt, PaidBy: r.PaidBy, PaymentBatchID: r.PaymentBatchID, LastRemindedAt: r.LastRemindedAt, CreatedAt: r.CreatedAt, CreatedBy: r.CreatedBy, UpdatedAt: r.UpdatedAt, UpdatedBy: r.UpdatedBy})
	}
	out, err = s.withReceipts(ctx, out)
	return out, total, sum, err
}

func (s *Store) MonthToDate(ctx context.Context, employeeID, category, from, to, excludeID string) (domain.Money, error) {
	return s.q.MonthToDate(ctx, sqlc.MonthToDateParams{EmployeeID: employeeID, Category: category, ID: excludeID, DateFrom: day(from), DateTo: day(to)})
}

func (s *Store) InsertReceipt(ctx context.Context, r domain.Receipt) error {
	return s.q.InsertReceipt(ctx, sqlc.InsertReceiptParams{ID: r.ID, ClaimID: r.ClaimID, StorageKey: r.StorageKey, Filename: r.Filename, ContentType: r.ContentType, SizeBytes: r.SizeBytes, CreatedBy: sp(r.CreatedBy)})
}

func (s *Store) DeleteReceipt(ctx context.Context, id string) error {
	return s.q.DeleteReceipt(ctx, id)
}

func (s *Store) GetReceipt(ctx context.Context, id string) (*domain.Receipt, error) {
	r, err := s.q.GetReceipt(ctx, id)
	return one(r, err, toReceipt)
}

func (s *Store) InsertClaimEvent(ctx context.Context, e domain.ClaimEvent) error {
	return s.q.InsertClaimEvent(ctx, sqlc.InsertClaimEventParams{ID: ids.New(), ClaimID: e.ClaimID, Action: e.Action, ActorID: sp(e.ActorID), ActorName: e.ActorName, Note: e.Note})
}

func (s *Store) MarkReminded(ctx context.Context, idList []string, at time.Time) error {
	return s.q.MarkReminded(ctx, sqlc.MarkRemindedParams{At: &at, Ids: idList})
}

func (s *Store) InsertBatch(ctx context.Context, b domain.ReimbursementBatch) error {
	return wrap(s.q.InsertBatch(ctx, sqlc.InsertBatchParams{ID: b.ID, BatchNo: b.BatchNo, PaidAt: day(b.PaidAt), CashAccountID: sp(b.CashAccountID), TotalAmount: b.TotalAmount, ClaimCount: int32(b.ClaimCount), Notes: b.Notes, CreatedBy: sp(b.CreatedBy)}))
}

func (s *Store) UpdateBatchJournal(ctx context.Context, id, entryID, status, reason string) error {
	return s.q.UpdateBatchJournal(ctx, sqlc.UpdateBatchJournalParams{ID: id, JournalEntryID: sp(entryID), JournalStatus: status, JournalReason: reason})
}

func (s *Store) ListBatches(ctx context.Context, limit int) ([]domain.ReimbursementBatch, error) {
	rows, err := s.q.ListBatches(ctx, int32(limit))
	return mapAll(rows, func(r sqlc.ListBatchesRow) domain.ReimbursementBatch {
		return domain.ReimbursementBatch{ID: r.ID, BatchNo: r.BatchNo, PaidAt: ds(r.PaidAt), CashAccountID: str(r.CashAccountID), CashAccountName: r.CashAccountName, TotalAmount: r.TotalAmount, ClaimCount: int(r.ClaimCount), JournalEntryID: str(r.JournalEntryID), JournalStatus: r.JournalStatus, JournalReason: r.JournalReason, Notes: r.Notes, CreatedBy: str(r.CreatedBy), CreatedAt: r.CreatedAt}
	}), err
}

func (s *Store) MonthlySummary(ctx context.Context, from, to, employeeID string) ([]domain.SummaryRow, error) {
	rows, err := s.q.MonthlySummary(ctx, sqlc.MonthlySummaryParams{DateFrom: day(from), DateTo: day(to), EmployeeID: sp(employeeID)})
	return mapAll(rows, func(r sqlc.MonthlySummaryRow) domain.SummaryRow {
		return domain.SummaryRow{EmployeeID: r.EmployeeID, EmployeeName: r.EmployeeName, DepartmentName: r.DepartmentName, Category: r.Category, ClaimCount: int(r.ClaimCount), SubmittedAmount: r.SubmittedAmount, ApprovedAmount: r.ApprovedAmount, PaidAmount: r.PaidAmount}
	}), err
}

func (s *Store) ForgetUser(ctx context.Context, userID string) ([]string, error) {
	keys, err := s.q.DraftClaimKeysOfUser(ctx, &userID)
	if err != nil {
		return nil, err
	}
	if err := s.q.DeleteDraftClaimsOfUser(ctx, &userID); err != nil {
		return nil, err
	}
	return keys, s.q.ClearManagerUser(ctx, &userID)
}

func (s *Store) CountPendingClaims(ctx context.Context) (int, error) {
	n, err := s.q.CountPendingClaims(ctx)
	return int(n), err
}

// ─── Insights ───────────────────────────────────────────────

func (s *Store) InsertInsight(ctx context.Context, i domain.FinancialInsight) error {
	return s.q.InsertInsight(ctx, sqlc.InsertInsightParams{ID: i.ID, DateFrom: day(i.DateFrom), DateTo: day(i.DateTo), Language: i.Language, Model: i.Model, Content: i.Content, Metrics: i.Metrics, CreatedBy: sp(i.CreatedBy)})
}

func (s *Store) ListInsights(ctx context.Context, limit int) ([]domain.FinancialInsight, error) {
	rows, err := s.q.ListInsights(ctx, int32(limit))
	return mapAll(rows, func(r sqlc.FinanceFinancialInsight) domain.FinancialInsight {
		return domain.FinancialInsight{ID: r.ID, DateFrom: ds(r.DateFrom), DateTo: ds(r.DateTo), Language: r.Language, Model: r.Model, Content: r.Content, Metrics: r.Metrics, CreatedBy: str(r.CreatedBy), CreatedAt: r.CreatedAt}
	}), err
}

// ─── Dashboard analytics ────────────────────────────────────

func (s *Store) MonthlyMovements(ctx context.Context, from, to string) ([]domain.MonthMovement, error) {
	rows, err := s.q.MonthlyAccountMovements(ctx, sqlc.MonthlyAccountMovementsParams{DateFrom: day(from), DateTo: day(to)})
	return mapAll(rows, func(r sqlc.MonthlyAccountMovementsRow) domain.MonthMovement {
		normal := domain.AsNormal(r.NormalBalance)
		return domain.MonthMovement{Month: r.Month, ParentID: r.ParentID, ParentName: r.ParentName, GrandparentID: r.GrandparentID,
			AccountBalance: domain.AccountBalance{ID: r.ID, Code: r.Code, Name: r.Name, TypeCode: r.TypeCode, NormalBalance: normal, IsContra: r.IsContra,
				IsCashBank: r.IsCashBank, CashFlowCategory: r.CashFlowCategory, Debit: r.Debit, Credit: r.Credit, Balance: domain.Balance(r.Debit, r.Credit, normal, r.IsContra)}}
	}), err
}

func (s *Store) AccountParents(ctx context.Context) ([]domain.AccountParent, error) {
	rows, err := s.q.AccountParents(ctx)
	return mapAll(rows, func(r sqlc.AccountParentsRow) domain.AccountParent {
		return domain.AccountParent{ID: r.ID, Code: r.Code, Name: r.Name, ParentID: r.ParentID, ParentName: r.ParentName, GrandparentID: r.GrandparentID}
	}), err
}

func (s *Store) InvoiceFacts(ctx context.Context, asOf string) ([]domain.InvoiceFact, error) {
	rows, err := s.q.AnalyticsInvoices(ctx, day(asOf))
	return mapAll(rows, func(r sqlc.AnalyticsInvoicesRow) domain.InvoiceFact {
		return domain.InvoiceFact{ID: r.ID, Kind: r.Kind, InvoiceNo: r.InvoiceNo, InvoiceDate: ds(r.InvoiceDate), DueDate: dsp(r.DueDate), PartyName: r.PartyName, Subtotal: r.Subtotal, Total: r.TotalAmount}
	}), err
}

func (s *Store) AllocationFacts(ctx context.Context, asOf string) ([]domain.AllocationFact, error) {
	rows, err := s.q.AnalyticsAllocations(ctx, day(asOf))
	return mapAll(rows, func(r sqlc.AnalyticsAllocationsRow) domain.AllocationFact {
		return domain.AllocationFact{InvoiceID: r.InvoiceID, PaymentDate: ds(r.PaymentDate), Amount: r.Amount}
	}), err
}

func (s *Store) ClaimFacts(ctx context.Context, from, to, department string) ([]domain.ClaimFact, error) {
	rows, err := s.q.AnalyticsClaims(ctx, sqlc.AnalyticsClaimsParams{Department: sp(department), DateFrom: day(from), DateTo: day(to)})
	return mapAll(rows, func(r sqlc.AnalyticsClaimsRow) domain.ClaimFact {
		return domain.ClaimFact{ID: r.ID, ClaimNo: r.ClaimNo, EmployeeName: r.EmployeeName, DepartmentName: r.DepartmentName, Category: r.Category,
			TransactionDate: ds(r.TransactionDate), Status: r.Status, Amount: r.Amount, SubmittedAt: r.SubmittedAt, ManagerApprovedAt: r.ManagerApprovedAt,
			FinanceApprovedAt: r.FinanceApprovedAt, PaidAt: r.PaidAt, RejectedAt: r.RejectedAt}
	}), err
}

func (s *Store) ClaimDepartments(ctx context.Context) ([]string, error) {
	return s.q.ClaimDepartments(ctx)
}
