package grpcapi

import (
	"context"

	financev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/finance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ─── BankReconService ────────────────────────

type bankReconServer struct {
	financev1.UnimplementedBankReconServiceServer
	svc Services
}

func statementLinePB(l *domain.StatementLine) *financev1.StatementLine {
	return &financev1.StatementLine{Id: l.ID, AccountId: l.AccountID, LineDate: l.LineDate, Description: l.Description, SenderName: l.SenderName,
		Amount: l.Amount.String(), Status: l.Status, SuggestedKind: l.SuggestedKind, SuggestedRef: l.SuggestedRef, SuggestedLabel: l.SuggestedLabel,
		SuggestedScore: int32(l.SuggestedScore), SuggestedReason: l.SuggestedReason, MatchedEntryId: l.MatchedEntryID, MatchedEntryNo: l.MatchedEntryNo,
		Note: l.Note, VerifiedBy: l.VerifiedBy, VerifiedAt: tsPtr(l.VerifiedAt)}
}

func importPB(r *app.StatementImport) *financev1.ImportStatementResponse {
	return &financev1.ImportStatementResponse{Inserted: int32(r.Inserted), Skipped: int32(r.Skipped), Matched: int32(r.Matched), Suggested: int32(r.Suggested), Unmatched: int32(r.Unmatched)}
}

func (s *bankReconServer) ListReconAccounts(ctx context.Context, _ *emptypb.Empty) (*financev1.ListReconAccountsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Banking.Accounts(ctx)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListReconAccountsResponse{}
	for _, a := range rows {
		out.Accounts = append(out.Accounts, &financev1.ReconAccount{Id: a.ID, Code: a.Code, CodeDisplay: domain.FormatAccountCodeDisplay(a.Code), Name: a.Name,
			IsCreditCard: a.IsCreditCard, BookBalance: a.Balance.String(), Unmatched: int32(a.Counts[domain.LineUnmatched]), Suggested: int32(a.Counts[domain.LineSuggested]),
			Matched: int32(a.Counts[domain.LineMatched]), Ignored: int32(a.Counts[domain.LineIgnored])})
	}
	return out, nil
}

func (s *bankReconServer) ListStatementLines(ctx context.Context, r *financev1.ListStatementLinesRequest) (*financev1.ListStatementLinesResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, total, err := s.svc.Banking.Lines(ctx, domain.StatementFilter{AccountID: r.AccountId, Status: r.Status, DateFrom: r.DateFrom, DateTo: r.DateTo, Page: int(r.Page), PageSize: int(r.PageSize)})
	if err != nil {
		return nil, err
	}
	out := &financev1.ListStatementLinesResponse{Total: int32(total)}
	for i := range rows {
		out.Lines = append(out.Lines, statementLinePB(&rows[i]))
	}
	return out, nil
}

func (s *bankReconServer) ImportStatement(ctx context.Context, r *financev1.ImportStatementRequest) (*financev1.ImportStatementResponse, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	res, err := s.svc.Banking.Import(ctx, p.UserID, r.AccountId, r.Filename, r.Data)
	if err != nil {
		return nil, err
	}
	return importPB(res), nil
}

func (s *bankReconServer) RematchStatements(ctx context.Context, r *financev1.RematchRequest) (*financev1.ImportStatementResponse, error) {
	if _, err := write(ctx); err != nil {
		return nil, err
	}
	matched, suggested, unmatched, err := s.svc.Banking.Rematch(ctx, r.AccountId)
	if err != nil {
		return nil, err
	}
	return importPB(&app.StatementImport{Matched: matched, Suggested: suggested, Unmatched: unmatched}), nil
}

func (s *bankReconServer) ListMatchCandidates(ctx context.Context, r *financev1.IdRequest) (*financev1.ListMatchCandidatesResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Banking.Candidates(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListMatchCandidatesResponse{}
	for _, c := range rows {
		out.Candidates = append(out.Candidates, &financev1.MatchCandidate{Kind: c.Kind, Ref: c.Ref, Label: c.Label, Date: c.Date, Amount: c.Amount.String(), Score: int32(c.Score), Reason: c.Reason})
	}
	return out, nil
}

func (s *bankReconServer) MatchStatementLine(ctx context.Context, r *financev1.MatchStatementLineRequest) (*financev1.StatementLine, error) {
	var (
		l   *domain.StatementLine
		err error
	)
	if r.Kind == domain.SuggestInvoice {
		p, perr := approve(ctx)
		if perr != nil {
			return nil, perr
		}
		l, err = s.svc.Banking.MatchInvoice(ctx, p.UserID, r.Id, r.Ref)
	} else {
		p, perr := write(ctx)
		if perr != nil {
			return nil, perr
		}
		l, err = s.svc.Banking.MatchJournal(ctx, p.UserID, r.Id, r.Ref)
	}
	if err != nil {
		return nil, err
	}
	return statementLinePB(l), nil
}

func (s *bankReconServer) RecordStatementLine(ctx context.Context, r *financev1.RecordStatementLineRequest) (*financev1.StatementLine, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	l, err := s.svc.Banking.Record(ctx, p.UserID, r.Id, r.CounterAccountId, r.Description)
	if err != nil {
		return nil, err
	}
	return statementLinePB(l), nil
}

func (s *bankReconServer) IgnoreStatementLine(ctx context.Context, r *financev1.IgnoreStatementLineRequest) (*financev1.StatementLine, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	l, err := s.svc.Banking.Ignore(ctx, p.UserID, r.Id, r.Note)
	if err != nil {
		return nil, err
	}
	return statementLinePB(l), nil
}

func (s *bankReconServer) ReopenStatementLine(ctx context.Context, r *financev1.IdRequest) (*financev1.StatementLine, error) {
	if _, err := write(ctx); err != nil {
		return nil, err
	}
	l, err := s.svc.Banking.Reopen(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return statementLinePB(l), nil
}

// ─── ScheduleService ─────────────────────────

type scheduleServer struct {
	financev1.UnimplementedScheduleServiceServer
	svc Services
}

func recurringPB(r *domain.RecurringJournal) *financev1.RecurringJournal {
	out := &financev1.RecurringJournal{Id: r.ID, Name: r.Name, Description: r.Description, Frequency: r.Frequency, NextRun: r.NextRun, EndDate: r.EndDate,
		IsActive: r.IsActive, LastEntryId: r.LastEntryID, LastError: r.LastError}
	var total domain.Money
	for _, l := range r.Lines {
		out.Lines = append(out.Lines, &financev1.JournalLineInput{AccountId: l.AccountID, EntrySide: sidePB(l.Side), Amount: l.Amount.String(), Memo: l.Memo})
		if l.Side == domain.Debit {
			total += l.Amount
		}
	}
	out.Total = total.String()
	return out
}

func assetPB(a *domain.FixedAsset) *financev1.FixedAsset {
	return &financev1.FixedAsset{Id: a.ID, Name: a.Name, AcquiredOn: a.AcquiredOn, Cost: a.Cost.String(), Salvage: a.Salvage.String(), LifeMonths: int32(a.LifeMonths),
		MonthlyAmount: a.DepreciationFor(1).String(), DepreciatedMonths: int32(a.DepreciatedMonths), Accumulated: a.Accumulated().String(), BookValue: (a.Cost - a.Accumulated()).String(),
		ExpenseAccountId: a.ExpenseAccountID, ExpenseAccountName: a.ExpenseAccountName, AccumAccountId: a.AccumAccountID, AccumAccountName: a.AccumAccountName,
		IsActive: a.IsActive, LastEntryId: a.LastEntryID, LastError: a.LastError}
}

func (s *scheduleServer) ListRecurringJournals(ctx context.Context, _ *emptypb.Empty) (*financev1.ListRecurringJournalsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Schedules.ListRecurring(ctx)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListRecurringJournalsResponse{}
	for i := range rows {
		out.Journals = append(out.Journals, recurringPB(&rows[i]))
	}
	return out, nil
}

// Schedules post journals without a further check, so setting one up takes
// the same permission as posting (finance.approve).
func (s *scheduleServer) CreateRecurringJournal(ctx context.Context, r *financev1.RecurringJournalInput) (*financev1.RecurringJournal, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	lines, err := linesFrom(r.Lines)
	if err != nil {
		return nil, err
	}
	out, err := s.svc.Schedules.CreateRecurring(ctx, p.UserID, app.RecurringInput{Name: r.Name, Description: r.Description, Frequency: r.Frequency, StartDate: r.StartDate, EndDate: r.EndDate, Lines: lines})
	if err != nil {
		return nil, err
	}
	return recurringPB(out), nil
}

func (s *scheduleServer) SetRecurringJournalActive(ctx context.Context, r *financev1.SetActiveRequest) (*financev1.RecurringJournal, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	out, err := s.svc.Schedules.SetRecurringActive(ctx, p.UserID, r.Id, r.Active)
	if err != nil {
		return nil, err
	}
	return recurringPB(out), nil
}

func (s *scheduleServer) DeleteRecurringJournal(ctx context.Context, r *financev1.IdRequest) (*emptypb.Empty, error) {
	if _, err := approve(ctx); err != nil {
		return nil, err
	}
	return empty, s.svc.Schedules.DeleteRecurring(ctx, r.Id)
}

func (s *scheduleServer) ListFixedAssets(ctx context.Context, _ *emptypb.Empty) (*financev1.ListFixedAssetsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Schedules.ListAssets(ctx)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListFixedAssetsResponse{}
	for i := range rows {
		out.Assets = append(out.Assets, assetPB(&rows[i]))
	}
	return out, nil
}

func (s *scheduleServer) CreateFixedAsset(ctx context.Context, r *financev1.FixedAssetInput) (*financev1.FixedAsset, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	cost, err := money("cost", r.Cost)
	if err != nil {
		return nil, err
	}
	salvage, err := money("salvage", orZero(r.Salvage))
	if err != nil {
		return nil, err
	}
	out, err := s.svc.Schedules.CreateAsset(ctx, p.UserID, app.AssetInput{Name: r.Name, AcquiredOn: r.AcquiredOn, Cost: cost, Salvage: salvage, LifeMonths: int(r.LifeMonths), ExpenseAccountID: r.ExpenseAccountId, AccumAccountID: r.AccumAccountId})
	if err != nil {
		return nil, err
	}
	return assetPB(out), nil
}

func orZero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}

func (s *scheduleServer) SetFixedAssetActive(ctx context.Context, r *financev1.SetActiveRequest) (*financev1.FixedAsset, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	out, err := s.svc.Schedules.SetAssetActive(ctx, p.UserID, r.Id, r.Active)
	if err != nil {
		return nil, err
	}
	return assetPB(out), nil
}

func (s *scheduleServer) DeleteFixedAsset(ctx context.Context, r *financev1.IdRequest) (*emptypb.Empty, error) {
	if _, err := approve(ctx); err != nil {
		return nil, err
	}
	return empty, s.svc.Schedules.DeleteAsset(ctx, r.Id)
}

func (s *scheduleServer) RunSchedules(ctx context.Context, _ *emptypb.Empty) (*financev1.RunSchedulesResponse, error) {
	if _, err := approve(ctx); err != nil {
		return nil, err
	}
	res, err := s.svc.Schedules.RunDue(ctx)
	if err != nil {
		return nil, err
	}
	return &financev1.RunSchedulesResponse{Posted: int32(res.Posted), Failed: int32(res.Failed)}, nil
}
