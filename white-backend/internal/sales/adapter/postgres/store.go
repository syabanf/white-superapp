// Package postgres implements the sales ports on top of sqlc queries.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

type Store struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
	inTx bool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, q: sqlc.New(pool)}
}

// WithTx runs fn in a transaction; nested calls reuse the outer one.
func (s *Store) WithTx(ctx context.Context, fn func(domain.Store) error) error {
	if s.inTx {
		return fn(s)
	}
	return wrap(pg.Tx(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(&Store{pool: s.pool, q: s.q.WithTx(tx), inTx: true})
	}))
}

// wrap turns unique violations into domain errors.
func wrap(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return &domain.ErrUnique{Constraint: pgErr.ConstraintName}
	}
	return err
}

// one maps no rows to a nil result.
func one[R any, T any](row R, err error, conv func(R) T) (*T, error) {
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v := conv(row)
	return &v, nil
}

func many[R any, T any](rows []R, err error, conv func(R) T) ([]T, error) {
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(rows))
	for _, r := range rows {
		out = append(out, conv(r))
	}
	return out, nil
}

func nz(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func jsonOr(raw []byte, fallback string) []byte {
	if len(raw) == 0 {
		return []byte(fallback)
	}
	return raw
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("null")
	}
	return b
}

func limitOr(n, def int) int32 {
	if n <= 0 {
		return int32(def)
	}
	return int32(min(n, 1000))
}

func (s *Store) NextNumber(ctx context.Context, prefix string) (int, error) {
	n, err := s.q.NextNumber(ctx, prefix)
	return int(n), err
}

// ─── pipelines ───────────────────────────────────────────────

func toStage(r sqlc.SalesStage) domain.Stage {
	return domain.Stage{ID: r.ID, PipelineID: r.PipelineID, Code: r.Code, Name: r.Name, SortOrder: int(r.SortOrder), Probability: int(r.Probability), IsWon: r.IsWon, IsLost: r.IsLost, StuckThresholdDays: r.StuckThresholdDays, IsActive: r.IsActive}
}

func (s *Store) ListPipelines(ctx context.Context) ([]domain.Pipeline, error) {
	ps, err := s.q.ListPipelines(ctx)
	if err != nil {
		return nil, err
	}
	stages, err := s.q.ListStages(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Pipeline, 0, len(ps))
	for _, p := range ps {
		pl := domain.Pipeline{ID: p.ID, Code: p.Code, Name: p.Name, IsDefault: p.IsDefault, SortOrder: int(p.SortOrder)}
		for _, st := range stages {
			if st.PipelineID == p.ID {
				pl.Stages = append(pl.Stages, toStage(st))
			}
		}
		out = append(out, pl)
	}
	return out, nil
}

func (s *Store) SavePipeline(ctx context.Context, p domain.Pipeline) error {
	if err := s.q.UpsertPipeline(ctx, sqlc.UpsertPipelineParams{ID: p.ID, Code: p.Code, Name: p.Name, IsDefault: p.IsDefault, SortOrder: int32(p.SortOrder)}); err != nil {
		return wrap(err)
	}
	if p.IsDefault {
		return s.q.ClearDefaultPipeline(ctx, p.ID)
	}
	return nil
}

func (s *Store) GetStage(ctx context.Context, id string) (*domain.Stage, error) {
	r, err := s.q.GetStage(ctx, id)
	return one(r, err, toStage)
}

func (s *Store) SaveStage(ctx context.Context, st domain.Stage) error {
	return wrap(s.q.UpsertStage(ctx, sqlc.UpsertStageParams{ID: st.ID, PipelineID: st.PipelineID, Code: st.Code, Name: st.Name, SortOrder: int32(st.SortOrder), Probability: int32(st.Probability), IsWon: st.IsWon, IsLost: st.IsLost, StuckThresholdDays: st.StuckThresholdDays, IsActive: st.IsActive}))
}

func (s *Store) ListLostReasons(ctx context.Context) ([]domain.LostReason, error) {
	rows, err := s.q.ListLostReasons(ctx)
	return many(rows, err, func(r sqlc.SalesLostReason) domain.LostReason {
		return domain.LostReason{ID: r.ID, Code: r.Code, Name: r.Name, SortOrder: int(r.SortOrder), IsActive: r.IsActive}
	})
}

// ─── leads ───────────────────────────────────────────────────

func toLead(r sqlc.SalesLead) domain.Lead {
	l := domain.Lead{ID: r.ID, OrgName: r.OrgName, OrgType: r.OrgType, PicName: r.PicName, PicTitle: r.PicTitle, PicPhone: r.PicPhone, PicEmail: r.PicEmail, City: r.City, Source: r.Source, Temperature: r.Temperature, Status: r.Status, Notes: r.Notes, OwnerUserID: r.OwnerUserID, AccountID: r.AccountID, ContactID: r.ContactID, Custom: jsonOr(r.Custom, "{}"), Score: int(r.Score), ScoreUpdatedAt: r.ScoreUpdatedAt,
		Attribution: domain.Attribution{UtmSource: r.UtmSource, UtmMedium: r.UtmMedium, UtmCampaign: r.UtmCampaign, UtmTerm: r.UtmTerm, UtmContent: r.UtmContent, LandingPage: r.LandingPage, Referrer: r.Referrer},
		Audit:       domain.Audit{CreatedBy: r.CreatedBy, UpdatedBy: r.UpdatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}}
	_ = json.Unmarshal(r.ScoreBreakdown, &l.ScoreBreakdown)
	return l
}

func (s *Store) GetLead(ctx context.Context, id string) (*domain.Lead, error) {
	r, err := s.q.GetLead(ctx, id)
	return one(r, err, toLead)
}

func (s *Store) ListLeads(ctx context.Context, f domain.LeadFilter) ([]domain.Lead, int, error) {
	rows, err := s.q.ListLeads(ctx, sqlc.ListLeadsParams{Status: nz(f.Status), Temperature: nz(f.Temperature), Source: nz(f.Source), Owner: nz(f.OwnerUserID), Q: nz(f.Query), FromAt: f.From, ToAt: f.To, ScoreMin: f.ScoreMin, ScoreMax: f.ScoreMax, Off: int32(f.Offset), Lim: limitOr(f.Limit, 50)})
	leads, err := many(rows, err, toLead)
	if err != nil {
		return nil, 0, err
	}
	n, err := s.q.CountLeads(ctx, sqlc.CountLeadsParams{Status: nz(f.Status), Temperature: nz(f.Temperature), Source: nz(f.Source), Owner: nz(f.OwnerUserID), Q: nz(f.Query), FromAt: f.From, ToAt: f.To, ScoreMin: f.ScoreMin, ScoreMax: f.ScoreMax})
	return leads, int(n), err
}

func (s *Store) SaveLead(ctx context.Context, l *domain.Lead) error {
	a := l.Attribution
	return wrap(s.q.UpsertLead(ctx, sqlc.UpsertLeadParams{ID: l.ID, OrgName: l.OrgName, OrgType: l.OrgType, PicName: l.PicName, PicTitle: l.PicTitle, PicPhone: l.PicPhone, PicEmail: l.PicEmail, City: l.City, Source: l.Source, Temperature: l.Temperature, Status: l.Status, Notes: l.Notes, OwnerUserID: l.OwnerUserID, AccountID: l.AccountID, ContactID: l.ContactID, Custom: jsonOr(l.Custom, "{}"),
		UtmSource: a.UtmSource, UtmMedium: a.UtmMedium, UtmCampaign: a.UtmCampaign, UtmTerm: a.UtmTerm, UtmContent: a.UtmContent, LandingPage: a.LandingPage, Referrer: a.Referrer,
		CreatedBy: l.CreatedBy, UpdatedBy: l.UpdatedBy, CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt}))
}

func (s *Store) DeleteLead(ctx context.Context, id string, at time.Time) error {
	return s.q.DeleteLead(ctx, sqlc.DeleteLeadParams{ID: id, DeletedAt: &at})
}

func (s *Store) FindLeadByContact(ctx context.Context, phone, email string) (*domain.Lead, error) {
	if phone == "" && email == "" {
		return nil, nil
	}
	r, err := s.q.FindLeadByContact(ctx, sqlc.FindLeadByContactParams{Phone: nz(phone), Email: nz(email)})
	return one(r, err, toLead)
}

func (s *Store) LeadDedupKeys(ctx context.Context) (map[string]bool, error) {
	rows, err := s.q.LeadKeys(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		out[domain.LeadDedupKey(r.Phone, r.OrgName)] = true
	}
	return out, nil
}

func (s *Store) UpdateLeadScore(ctx context.Context, id string, score int, breakdown []domain.ScoreItem, at time.Time) error {
	return s.q.UpdateLeadScore(ctx, sqlc.UpdateLeadScoreParams{ID: id, Score: int32(score), ScoreBreakdown: mustJSON(breakdown), ScoreUpdatedAt: &at})
}

func (s *Store) ListLeadIDs(ctx context.Context) ([]string, error) { return s.q.ListLeadIDs(ctx) }

func (s *Store) StaleLeadIDs(ctx context.Context, before time.Time) ([]string, error) {
	return s.q.StaleLeadIDs(ctx, before)
}

// ─── accounts and contacts ──────────────────────────────────

func toAccount(r sqlc.SalesAccount) domain.Account {
	return domain.Account{ID: r.ID, Name: r.Name, AccountType: r.AccountType, Industry: r.Industry, Address: r.Address, City: r.City, Phone: r.Phone, Email: r.Email, Website: r.Website, Npwp: r.Npwp, Notes: r.Notes, OwnerUserID: r.OwnerUserID, Custom: jsonOr(r.Custom, "{}"),
		Audit: domain.Audit{CreatedBy: r.CreatedBy, UpdatedBy: r.UpdatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}}
}

func (s *Store) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	r, err := s.q.GetAccount(ctx, id)
	return one(r, err, toAccount)
}

func (s *Store) ListAccounts(ctx context.Context, query string, limit int) ([]domain.Account, error) {
	rows, err := s.q.ListAccounts(ctx, sqlc.ListAccountsParams{Q: nz(query), Lim: limitOr(limit, 200)})
	return many(rows, err, toAccount)
}

func (s *Store) SaveAccount(ctx context.Context, a *domain.Account) error {
	return wrap(s.q.UpsertAccount(ctx, sqlc.UpsertAccountParams{ID: a.ID, Name: a.Name, AccountType: a.AccountType, Industry: a.Industry, Address: a.Address, City: a.City, Phone: a.Phone, Email: a.Email, Website: a.Website, Npwp: a.Npwp, Notes: a.Notes, OwnerUserID: a.OwnerUserID, Custom: jsonOr(a.Custom, "{}"), CreatedBy: a.CreatedBy, UpdatedBy: a.UpdatedBy, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt}))
}

func (s *Store) DeleteAccount(ctx context.Context, id string, at time.Time) error {
	return s.q.DeleteAccount(ctx, sqlc.DeleteAccountParams{ID: id, DeletedAt: &at})
}

func toContact(r sqlc.SalesContact) domain.Contact {
	return domain.Contact{ID: r.ID, AccountID: r.AccountID, Name: r.Name, Title: r.Title, Phone: r.Phone, Email: r.Email, IsPrimary: r.IsPrimary, Notes: r.Notes, Custom: jsonOr(r.Custom, "{}"),
		Audit: domain.Audit{CreatedBy: r.CreatedBy, UpdatedBy: r.UpdatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}}
}

func (s *Store) GetContact(ctx context.Context, id string) (*domain.Contact, error) {
	r, err := s.q.GetContact(ctx, id)
	return one(r, err, toContact)
}

func (s *Store) ListContacts(ctx context.Context, accountID, query string, limit int) ([]domain.Contact, error) {
	rows, err := s.q.ListContacts(ctx, sqlc.ListContactsParams{Account: nz(accountID), Q: nz(query), Lim: limitOr(limit, 200)})
	return many(rows, err, toContact)
}

func (s *Store) SaveContact(ctx context.Context, c *domain.Contact) error {
	return wrap(s.q.UpsertContact(ctx, sqlc.UpsertContactParams{ID: c.ID, AccountID: c.AccountID, Name: c.Name, Title: c.Title, Phone: c.Phone, Email: c.Email, IsPrimary: c.IsPrimary, Notes: c.Notes, Custom: jsonOr(c.Custom, "{}"), CreatedBy: c.CreatedBy, UpdatedBy: c.UpdatedBy, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}))
}

func (s *Store) DeleteContact(ctx context.Context, id string, at time.Time) error {
	return s.q.DeleteContact(ctx, sqlc.DeleteContactParams{ID: id, DeletedAt: &at})
}

func (s *Store) ListCustomFields(ctx context.Context, object string) ([]domain.CustomField, error) {
	rows, err := s.q.ListCustomFields(ctx, nz(object))
	return many(rows, err, func(r sqlc.SalesCustomField) domain.CustomField {
		f := domain.CustomField{ID: r.ID, Object: r.Object, Key: r.Key, Label: r.Label, FieldType: r.FieldType, IsRequired: r.IsRequired, ShowInList: r.ShowInList, SortOrder: int(r.SortOrder), IsActive: r.IsActive}
		_ = json.Unmarshal(r.Options, &f.Options)
		return f
	})
}

func (s *Store) SaveCustomField(ctx context.Context, f domain.CustomField) error {
	if f.Options == nil {
		f.Options = []string{}
	}
	return wrap(s.q.UpsertCustomField(ctx, sqlc.UpsertCustomFieldParams{ID: f.ID, Object: f.Object, Key: f.Key, Label: f.Label, FieldType: f.FieldType, Options: mustJSON(f.Options), IsRequired: f.IsRequired, ShowInList: f.ShowInList, SortOrder: int32(f.SortOrder), IsActive: f.IsActive}))
}

func (s *Store) DeleteCustomField(ctx context.Context, id string) error {
	return s.q.DeleteCustomField(ctx, id)
}

// ─── deals ───────────────────────────────────────────────────

func toDeal(d sqlc.SalesDeal) domain.Deal {
	return domain.Deal{ID: d.ID, LeadID: d.LeadID, AccountID: d.AccountID, Title: d.Title, ServiceLines: d.ServiceLines, PipelineID: d.PipelineID, StageID: d.StageID, ValueEstimate: domain.MoneyPtr(d.ValueEstimate), ValueFinal: domain.MoneyPtr(d.ValueFinal), OwnerUserID: d.OwnerUserID, LostReasonID: d.LostReasonID, ExpectedCloseDate: d.ExpectedCloseDate, Notes: d.Notes, ForecastCategory: d.ForecastCategory, EnteredStageAt: d.EnteredStageAt, ClosedAt: d.ClosedAt, Custom: jsonOr(d.Custom, "{}"),
		Audit: domain.Audit{CreatedBy: d.CreatedBy, UpdatedBy: d.UpdatedBy, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}}
}

func dealRow(r sqlc.GetDealRow) domain.Deal {
	d := toDeal(r.SalesDeal)
	d.StageCode, d.StageName, d.IsWon, d.IsLost, d.Probability = r.StageCode, r.StageName, r.IsWon, r.IsLost, int(r.Probability)
	d.OrgName, d.PicName, d.PicPhone, d.PicEmail, d.AccountName = r.OrgName, r.PicName, r.PicPhone, r.PicEmail, r.AccountName
	return d
}

func (s *Store) GetDeal(ctx context.Context, id string) (*domain.Deal, error) {
	r, err := s.q.GetDeal(ctx, id)
	return one(r, err, dealRow)
}

func (s *Store) ListDeals(ctx context.Context, f domain.DealFilter) ([]domain.Deal, int, error) {
	c := sqlc.CountDealsParams{Pipeline: nz(f.PipelineID), Stage: nz(f.StageID), Owner: nz(f.OwnerUserID), Lead: nz(f.LeadID), Account: nz(f.AccountID), Status: nz(f.Status), Q: nz(f.Query),
		Line: nz(f.ServiceLine), Industry: nz(f.Industry), Source: nz(f.Source), LostReason: nz(f.LostReason), FromAt: f.From, ToAt: f.To, ByClosed: f.ByClosed}
	rows, err := s.q.ListDeals(ctx, sqlc.ListDealsParams{Pipeline: c.Pipeline, Stage: c.Stage, Owner: c.Owner, Lead: c.Lead, Account: c.Account, Status: c.Status, Q: c.Q,
		Line: c.Line, Industry: c.Industry, Source: c.Source, LostReason: c.LostReason, FromAt: c.FromAt, ToAt: c.ToAt, ByClosed: c.ByClosed, Off: int32(f.Offset), Lim: limitOr(f.Limit, 200)})
	deals, err := many(rows, err, func(r sqlc.ListDealsRow) domain.Deal { return dealRow(sqlc.GetDealRow(r)) })
	if err != nil {
		return nil, 0, err
	}
	n, err := s.q.CountDeals(ctx, c)
	return deals, int(n), err
}

func (s *Store) SaveDeal(ctx context.Context, d *domain.Deal) error {
	lines := d.ServiceLines
	if lines == nil {
		lines = []string{}
	}
	return wrap(s.q.UpsertDeal(ctx, sqlc.UpsertDealParams{ID: d.ID, LeadID: d.LeadID, AccountID: d.AccountID, Title: d.Title, ServiceLines: lines, PipelineID: d.PipelineID, StageID: d.StageID, ValueEstimate: d.ValueEstimate.StringPtr(), ValueFinal: d.ValueFinal.StringPtr(), OwnerUserID: d.OwnerUserID, LostReasonID: d.LostReasonID, ExpectedCloseDate: d.ExpectedCloseDate, Notes: d.Notes, ForecastCategory: d.ForecastCategory, EnteredStageAt: d.EnteredStageAt, ClosedAt: d.ClosedAt, Custom: jsonOr(d.Custom, "{}"), CreatedBy: d.CreatedBy, UpdatedBy: d.UpdatedBy, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}))
}

func (s *Store) DeleteDeal(ctx context.Context, id string, at time.Time) error {
	return s.q.DeleteDeal(ctx, sqlc.DeleteDealParams{ID: id, DeletedAt: &at})
}

func (s *Store) ListDealMembers(ctx context.Context, dealID string) ([]domain.DealMember, error) {
	rows, err := s.q.ListDealMembers(ctx, dealID)
	return many(rows, err, func(r sqlc.SalesDealMember) domain.DealMember {
		return domain.DealMember{DealID: r.DealID, UserID: r.UserID, Role: r.Role, SplitPercent: domain.ParsePercent(r.SplitPercent)}
	})
}

func (s *Store) ReplaceDealMembers(ctx context.Context, dealID string, members []domain.DealMember) error {
	if err := s.q.DeleteDealMembers(ctx, dealID); err != nil {
		return err
	}
	for _, m := range members {
		if err := s.q.InsertDealMember(ctx, sqlc.InsertDealMemberParams{DealID: dealID, UserID: m.UserID, Role: m.Role, SplitPercent: domain.PercentString(m.SplitPercent)}); err != nil {
			return wrap(err)
		}
	}
	return nil
}

func (s *Store) AddStageHistory(ctx context.Context, h domain.StageHistory) error {
	return s.q.InsertStageHistory(ctx, sqlc.InsertStageHistoryParams{ID: h.ID, DealID: h.DealID, StageID: h.StageID, EnteredAt: h.EnteredAt, MovedBy: h.MovedBy})
}

func (s *Store) ListStageHistory(ctx context.Context, dealID string) ([]domain.StageHistory, error) {
	rows, err := s.q.ListStageHistory(ctx, dealID)
	return many(rows, err, func(r sqlc.ListStageHistoryRow) domain.StageHistory {
		return domain.StageHistory{ID: r.ID, DealID: r.DealID, StageID: r.StageID, StageName: r.StageName, EnteredAt: r.EnteredAt, MovedBy: r.MovedBy}
	})
}

func (s *Store) ForecastDeals(ctx context.Context, from, to time.Time) ([]domain.ForecastDeal, error) {
	fromAt, toAt := from.Add(-7*time.Hour), to.Add(-7*time.Hour) // WIB month boundaries
	rows, err := s.q.ForecastDeals(ctx, sqlc.ForecastDealsParams{FromAt: &fromAt, ToAt: &toAt, FromDay: from, ToDay: to})
	return many(rows, err, func(r sqlc.ForecastDealsRow) domain.ForecastDeal {
		return domain.ForecastDeal{OwnerUserID: r.OwnerUserID, Value: domain.MustMoney(r.Value), Probability: int(r.Probability), Category: r.Category}
	})
}

func (s *Store) StaleDealIDs(ctx context.Context, before time.Time) ([]string, error) {
	return s.q.StaleDealIDs(ctx, before)
}

// ─── activities ──────────────────────────────────────────────

func toActivity(r sqlc.SalesActivity) domain.Activity {
	return domain.Activity{ID: r.ID, LeadID: r.LeadID, DealID: r.DealID, SubjectType: r.SubjectType, SubjectID: r.SubjectID, ActivityType: r.ActivityType, Title: r.Title, Notes: r.Notes, Priority: r.Priority, Status: r.Status, DueAt: r.DueAt, DoneAt: r.DoneAt, ReminderAt: r.ReminderAt, ReminderSentAt: r.ReminderSentAt, Recurrence: r.Recurrence, ParentTaskID: r.ParentTaskID, OwnerUserID: r.OwnerUserID,
		Audit: domain.Audit{CreatedBy: r.CreatedBy, UpdatedBy: r.UpdatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}}
}

func (s *Store) GetActivity(ctx context.Context, id string) (*domain.Activity, error) {
	r, err := s.q.GetActivity(ctx, id)
	return one(r, err, toActivity)
}

func (s *Store) ListActivities(ctx context.Context, f domain.ActivityFilter) ([]domain.Activity, error) {
	rows, err := s.q.ListActivities(ctx, sqlc.ListActivitiesParams{Lead: nz(f.LeadID), Deal: nz(f.DealID), Owner: nz(f.OwnerUserID), Status: nz(f.Status), OpenOnly: f.OpenOnly, DueBefore: f.DueBefore, Lim: limitOr(f.Limit, 200)})
	return many(rows, err, toActivity)
}

func (s *Store) SaveActivity(ctx context.Context, a *domain.Activity) error {
	return wrap(s.q.UpsertActivity(ctx, sqlc.UpsertActivityParams{ID: a.ID, LeadID: a.LeadID, DealID: a.DealID, SubjectType: a.SubjectType, SubjectID: a.SubjectID, ActivityType: a.ActivityType, Title: a.Title, Notes: a.Notes, Priority: a.Priority, Status: a.Status, DueAt: a.DueAt, DoneAt: a.DoneAt, ReminderAt: a.ReminderAt, ReminderSentAt: a.ReminderSentAt, Recurrence: a.Recurrence, ParentTaskID: a.ParentTaskID, OwnerUserID: a.OwnerUserID, CreatedBy: a.CreatedBy, UpdatedBy: a.UpdatedBy, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt}))
}

func (s *Store) DeleteActivity(ctx context.Context, id string, at time.Time) error {
	return s.q.DeleteActivity(ctx, sqlc.DeleteActivityParams{ID: id, DeletedAt: &at})
}

func (s *Store) DueReminders(ctx context.Context, now time.Time, limit int) ([]domain.Activity, error) {
	rows, err := s.q.DueReminders(ctx, sqlc.DueRemindersParams{ReminderAt: &now, Limit: limitOr(limit, 100)})
	return many(rows, err, toActivity)
}

func (s *Store) ClaimReminder(ctx context.Context, id string, at time.Time) (bool, error) {
	n, err := s.q.ClaimReminder(ctx, sqlc.ClaimReminderParams{ID: id, ReminderSentAt: &at})
	return n == 1, err
}

func (s *Store) TasksDueBetween(ctx context.Context, from, to time.Time) ([]string, error) {
	return s.q.TasksDueBetween(ctx, sqlc.TasksDueBetweenParams{DueAt: &from, DueAt_2: &to})
}
