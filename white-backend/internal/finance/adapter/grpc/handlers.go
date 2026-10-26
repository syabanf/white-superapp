// Package grpcapi exposes the finance use cases over gRPC. Handlers check
// permissions (finance.read / finance.write / finance.approve, self.service
// for reimbursement self-service), convert proto <-> app values and call
// the use cases.
package grpcapi

import (
	"context"

	financev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/finance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Services struct {
	Accounts       *app.Accounts
	Fiscal         *app.Fiscal
	Journals       *app.Journals
	CashBank       *app.CashBank
	Ledgers        *app.Ledgers
	Reports        *app.Reports
	Reimbursements *app.Reimbursements
	Analysis       *app.Analysis
}

func Register(s *grpc.Server, svc Services) {
	financev1.RegisterAccountServiceServer(s, &accountServer{svc: svc})
	financev1.RegisterFiscalServiceServer(s, &fiscalServer{svc: svc})
	financev1.RegisterJournalServiceServer(s, &journalServer{svc: svc})
	financev1.RegisterCashBankServiceServer(s, &cashBankServer{svc: svc})
	financev1.RegisterPayableServiceServer(s, &payableServer{ledger: ledger{svc: svc, kind: domain.KindAP}})
	financev1.RegisterReceivableServiceServer(s, &receivableServer{ledger: ledger{svc: svc, kind: domain.KindAR}})
	financev1.RegisterReportServiceServer(s, &reportServer{svc: svc})
	financev1.RegisterReimbursementServiceServer(s, &reimbursementServer{svc: svc})
	financev1.RegisterAnalysisServiceServer(s, &analysisServer{svc: svc})
}

var empty = &emptypb.Empty{}

func read(ctx context.Context) (authn.Principal, error) {
	return authn.RequirePerm(ctx, authn.PermFinanceRead)
}

func write(ctx context.Context) (authn.Principal, error) {
	return authn.RequirePerm(ctx, authn.PermFinanceWrite)
}

func approve(ctx context.Context) (authn.Principal, error) {
	return authn.RequirePerm(ctx, authn.PermFinanceApprove)
}

// writeOrApprove needs finance.approve when the draft is saved as POSTED.
func writeOrApprove(ctx context.Context, post bool) (authn.Principal, error) {
	if post {
		return approve(ctx)
	}
	return write(ctx)
}

// ─── AccountService ─────────────────────────────────────────

type accountServer struct {
	financev1.UnimplementedAccountServiceServer
	svc Services
}

func (s *accountServer) ListAccountTypes(ctx context.Context, req *financev1.ListAccountTypesRequest) (*financev1.ListAccountTypesResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Accounts.ListAccountTypes(ctx, req.IncludeInactive)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListAccountTypesResponse{}
	for i := range rows {
		out.AccountTypes = append(out.AccountTypes, accountTypePB(&rows[i]))
	}
	return out, nil
}

func (s *accountServer) saveType(ctx context.Context, id string, in *financev1.AccountTypeInput) (*financev1.AccountType, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	t, err := s.svc.Accounts.SaveAccountType(ctx, p.UserID, domain.AccountType{ID: id, Code: in.GetCode(), Name: in.GetName(),
		NormalBalance: sideFrom(in.GetNormalBalance()), SortOrder: int(in.GetSortOrder()), IsActive: in.GetIsActive()})
	if err != nil {
		return nil, err
	}
	return accountTypePB(t), nil
}

func (s *accountServer) CreateAccountType(ctx context.Context, req *financev1.AccountTypeInput) (*financev1.AccountType, error) {
	return s.saveType(ctx, "", req)
}

func (s *accountServer) UpdateAccountType(ctx context.Context, req *financev1.UpdateAccountTypeRequest) (*financev1.AccountType, error) {
	return s.saveType(ctx, req.Id, req.Input)
}

func (s *accountServer) DeleteAccountType(ctx context.Context, req *financev1.IdRequest) (*emptypb.Empty, error) {
	if _, err := write(ctx); err != nil {
		return nil, err
	}
	return empty, s.svc.Accounts.DeleteAccountType(ctx, req.Id)
}

func (s *accountServer) ListAccounts(ctx context.Context, req *financev1.ListAccountsRequest) (*financev1.ListAccountsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Accounts.ListAccounts(ctx, domain.AccountFilter{AccountTypeID: req.AccountTypeId, IsPostable: req.IsPostable, IsCashBank: req.IsCashBank,
		IsActive: req.IsActive, CashFlowCategory: req.CashFlowCategory, Search: req.Search})
	if err != nil {
		return nil, err
	}
	out := &financev1.ListAccountsResponse{}
	for i := range rows {
		out.Accounts = append(out.Accounts, accountPB(&rows[i]))
	}
	return out, nil
}

func (s *accountServer) GetAccount(ctx context.Context, req *financev1.IdRequest) (*financev1.Account, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	a, err := s.svc.Accounts.GetAccount(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return accountPB(a), nil
}

func (s *accountServer) CreateAccount(ctx context.Context, req *financev1.AccountInput) (*financev1.Account, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	a, err := s.svc.Accounts.CreateAccount(ctx, p.UserID, accountFrom("", req))
	if err != nil {
		return nil, err
	}
	return accountPB(a), nil
}

func (s *accountServer) UpdateAccount(ctx context.Context, req *financev1.UpdateAccountRequest) (*financev1.Account, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	a, err := s.svc.Accounts.UpdateAccount(ctx, p.UserID, accountFrom(req.Id, req.Input))
	if err != nil {
		return nil, err
	}
	return accountPB(a), nil
}

func (s *accountServer) DeleteAccount(ctx context.Context, req *financev1.IdRequest) (*emptypb.Empty, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	return empty, s.svc.Accounts.DeleteAccount(ctx, p.UserID, req.Id)
}

func (s *accountServer) ImportAccounts(ctx context.Context, req *financev1.ImportAccountsRequest) (*financev1.ImportAccountsResponse, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	res, err := s.svc.Accounts.ImportAccounts(ctx, p.UserID, req.File, req.Mode)
	if err != nil {
		return nil, err
	}
	out := &financev1.ImportAccountsResponse{Mode: res.Mode, CreateCount: int32(res.Summary.Create), UpdateCount: int32(res.Summary.Update),
		SkipCount: int32(res.Summary.Skip), ErrorCount: int32(res.Summary.Error), Message: res.Message}
	for _, i := range res.Issues {
		out.Issues = append(out.Issues, &financev1.ImportIssue{Row: int32(i.Row), Code: i.Code, Message: i.Message})
	}
	for _, r := range res.Preview {
		out.Preview = append(out.Preview, &financev1.ImportPreviewRow{SourceRow: int32(r.SourceRow), Code: r.Code, Name: r.Name, ParentCode: r.ParentCode,
			AccountTypeCode: r.AccountTypeCode, Action: r.Action, Message: r.Message})
	}
	return out, nil
}

func (s *accountServer) ListPostableAccounts(ctx context.Context, _ *emptypb.Empty) (*financev1.ListPostableAccountsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	yes := true
	rows, err := s.svc.Accounts.ListAccounts(ctx, domain.AccountFilter{IsPostable: &yes, IsActive: &yes})
	if err != nil {
		return nil, err
	}
	out := &financev1.ListPostableAccountsResponse{}
	for _, a := range rows {
		out.Accounts = append(out.Accounts, postablePB(a))
	}
	return out, nil
}

func (s *accountServer) ListJournalEvents(ctx context.Context, _ *emptypb.Empty) (*financev1.ListJournalEventsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	out := &financev1.ListJournalEventsResponse{Modules: domain.JournalModules, AmountSources: domain.JournalAmountSources}
	for _, e := range domain.JournalEvents {
		out.Events = append(out.Events, &financev1.JournalEvent{EventCode: e.Code, Name: e.Name, Description: e.Description, Module: e.Module})
	}
	return out, nil
}

func (s *accountServer) ListJournalMappings(ctx context.Context, req *financev1.ListJournalMappingsRequest) (*financev1.ListJournalMappingsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Accounts.ListMappings(ctx, domain.MappingFilter{Search: req.Search, Module: req.Module, IsActive: req.IsActive})
	if err != nil {
		return nil, err
	}
	out := &financev1.ListJournalMappingsResponse{}
	for i := range rows {
		out.Mappings = append(out.Mappings, mappingPB(&rows[i]))
	}
	return out, nil
}

func (s *accountServer) GetJournalMapping(ctx context.Context, req *financev1.IdRequest) (*financev1.JournalMapping, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	m, err := s.svc.Accounts.GetMapping(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return mappingPB(m), nil
}

func (s *accountServer) CreateJournalMapping(ctx context.Context, req *financev1.JournalMappingInput) (*financev1.JournalMapping, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	m, lines := mappingFrom("", req)
	out, err := s.svc.Accounts.CreateMapping(ctx, p.UserID, m, lines)
	if err != nil {
		return nil, err
	}
	return mappingPB(out), nil
}

func (s *accountServer) UpdateJournalMapping(ctx context.Context, req *financev1.UpdateJournalMappingRequest) (*financev1.JournalMapping, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	m, lines := mappingFrom(req.Id, req.Input)
	out, err := s.svc.Accounts.UpdateMapping(ctx, p.UserID, m, lines)
	if err != nil {
		return nil, err
	}
	return mappingPB(out), nil
}

func (s *accountServer) DeleteJournalMapping(ctx context.Context, req *financev1.IdRequest) (*emptypb.Empty, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	return empty, s.svc.Accounts.DeleteMapping(ctx, p.UserID, req.Id)
}

// ─── FiscalService ──────────────────────────────────────────

type fiscalServer struct {
	financev1.UnimplementedFiscalServiceServer
	svc Services
}

func (s *fiscalServer) ListFiscalYears(ctx context.Context, req *financev1.ListFiscalYearsRequest) (*financev1.ListFiscalYearsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Fiscal.ListYears(ctx, req.Search, req.IsActive)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListFiscalYearsResponse{}
	for i := range rows {
		out.FiscalYears = append(out.FiscalYears, yearPB(&rows[i]))
	}
	return out, nil
}

func (s *fiscalServer) GetFiscalYear(ctx context.Context, req *financev1.IdRequest) (*financev1.FiscalYear, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	y, err := s.svc.Fiscal.GetYear(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return yearPB(y), nil
}

func (s *fiscalServer) CreateFiscalYear(ctx context.Context, req *financev1.FiscalYearInput) (*financev1.FiscalYear, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	y, err := s.svc.Fiscal.CreateYear(ctx, p.UserID, yearFrom(req))
	if err != nil {
		return nil, err
	}
	return yearPB(y), nil
}

func (s *fiscalServer) UpdateFiscalYear(ctx context.Context, req *financev1.UpdateFiscalYearRequest) (*financev1.FiscalYear, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	y, err := s.svc.Fiscal.UpdateYear(ctx, p.UserID, req.Id, yearFrom(req.Input))
	if err != nil {
		return nil, err
	}
	return yearPB(y), nil
}

func (s *fiscalServer) DeleteFiscalYear(ctx context.Context, req *financev1.IdRequest) (*emptypb.Empty, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	return empty, s.svc.Fiscal.DeleteYear(ctx, p.UserID, req.Id)
}

func (s *fiscalServer) ListFiscalPeriods(ctx context.Context, req *financev1.ListFiscalPeriodsRequest) (*financev1.ListFiscalPeriodsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Fiscal.ListPeriods(ctx, domain.PeriodFilter{FiscalYearID: req.FiscalYearId, Status: periodStatusFrom(req.Status), Search: req.Search})
	if err != nil {
		return nil, err
	}
	return &financev1.ListFiscalPeriodsResponse{Periods: periodsPB(rows)}, nil
}

func (s *fiscalServer) OpenFiscalPeriod(ctx context.Context, req *financev1.OpenFiscalPeriodRequest) (*financev1.FiscalPeriod, error) {
	if _, err := approve(ctx); err != nil {
		return nil, err
	}
	p, err := s.svc.Fiscal.OpenPeriod(ctx, req.Id, req.ClosePrevious)
	if err != nil {
		return nil, err
	}
	return periodPB(p), nil
}

func (s *fiscalServer) GetPeriodClosePreview(ctx context.Context, req *financev1.IdRequest) (*financev1.PeriodClosePreview, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	pv, err := s.svc.Fiscal.ClosePreview(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	out := &financev1.PeriodClosePreview{Period: periodPB(&pv.Period), PostedCount: int32(pv.Period.PostedCount), DraftCount: int32(pv.Period.DraftCount),
		CanClose: pv.CanClose, Blockers: pv.Blockers}
	for _, e := range pv.DraftEntries {
		out.DraftEntries = append(out.DraftEntries, &financev1.DraftEntryRef{Id: e.ID, EntryNo: e.EntryNo, EntryDate: e.EntryDate, Description: e.Description})
	}
	return out, nil
}

func (s *fiscalServer) CloseFiscalPeriod(ctx context.Context, req *financev1.IdRequest) (*financev1.FiscalPeriod, error) {
	if _, err := approve(ctx); err != nil {
		return nil, err
	}
	p, err := s.svc.Fiscal.ClosePeriod(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return periodPB(p), nil
}

func (s *fiscalServer) GetFiscalCoverage(ctx context.Context, req *financev1.GetFiscalCoverageRequest) (*financev1.FiscalCoverage, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	c, err := s.svc.Fiscal.Coverage(ctx, req.Date)
	if err != nil {
		return nil, err
	}
	out := &financev1.FiscalCoverage{Date: c.Date, Ready: c.Ready}
	if c.Period != nil {
		out.Period = periodPB(c.Period)
	}
	if c.Suggestion != nil {
		out.Suggestion = &financev1.FiscalOpenSuggestion{Period: periodPB(&c.Suggestion.Period), CanOpen: c.Suggestion.CanOpen,
			ClosePrevious: periodsPB(c.Suggestion.ClosePrevious), Message: c.Suggestion.Message}
	}
	return out, nil
}

func (s *fiscalServer) GetBeginningBalance(ctx context.Context, req *financev1.IdRequest) (*financev1.BeginningBalance, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	b, err := s.svc.Fiscal.BeginningBalance(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	out := &financev1.BeginningBalance{FiscalYearId: b.Year.ID, FiscalYearCode: b.Year.Code, FiscalYearName: b.Year.Name, StartDate: b.Year.StartDate,
		TotalDebit: b.TotalDebit.String(), TotalCredit: b.TotalCredit.String(), CanEdit: b.CanEdit, IsFirstYear: b.Prior == nil, Message: b.Message}
	if b.Period != nil {
		out.PeriodId, out.PeriodName = b.Period.ID, b.Period.Name
	}
	if b.Prior != nil {
		out.PriorFiscalYear = &financev1.FiscalYear{Id: b.Prior.ID, Code: b.Prior.Code, Name: b.Prior.Name, EndDate: b.Prior.EndDate}
		out.PriorFullyClosed = b.Prior.OpenCount == 0
	}
	if b.RetainedEarnings != nil {
		out.RetainedEarningsAccount = postablePB(*b.RetainedEarnings)
	}
	if b.Existing != nil {
		out.ExistingEntryId, out.ExistingStatus = b.Existing.ID, docStatusPB(b.Existing.Status)
	}
	for _, l := range b.Lines {
		out.Lines = append(out.Lines, &financev1.BeginningBalanceLine{AccountId: l.Account.ID, AccountCode: domain.FormatAccountCodeDisplay(l.Account.Code),
			AccountName: l.Account.Name, AccountTypeCode: l.Account.TypeCode, NormalBalance: sidePB(l.Account.NormalBalance), IsContra: l.Account.IsContra,
			SuggestedAmount: l.SuggestedAmount.String(), Amount: l.Amount.String(), EntrySide: sidePB(l.Side), Source: l.Source})
	}
	return out, nil
}

func (s *fiscalServer) SaveBeginningBalance(ctx context.Context, req *financev1.SaveBeginningBalanceRequest) (*financev1.SaveBeginningBalanceResponse, error) {
	p, err := writeOrApprove(ctx, req.Post)
	if err != nil {
		return nil, err
	}
	var lines []domain.Line
	for _, l := range req.Lines {
		amount, err := money("lines", l.Amount)
		if err != nil {
			return nil, err
		}
		lines = append(lines, domain.Line{AccountID: l.AccountId, Side: sideFrom(l.EntrySide), Amount: amount, Memo: l.Memo})
	}
	e, err := s.svc.Fiscal.SaveBeginningBalance(ctx, p.UserID, req.Id, lines, req.Post)
	if err != nil {
		return nil, err
	}
	return &financev1.SaveBeginningBalanceResponse{EntryId: e.ID, Status: docStatusPB(e.Status)}, nil
}

// ─── JournalService ─────────────────────────────────────────

type journalServer struct {
	financev1.UnimplementedJournalServiceServer
	svc Services
}

func (s *journalServer) ListJournalEntries(ctx context.Context, req *financev1.ListJournalEntriesRequest) (*financev1.ListJournalEntriesResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	limit, offset := app.Page(int(req.Page), int(req.PageSize))
	rows, total, err := s.svc.Journals.List(ctx, domain.JournalFilter{Search: req.Search, Status: docStatusFrom(req.Status), EntryType: entryTypeFrom(req.EntryType),
		DateFrom: req.DateFrom, DateTo: req.DateTo, AccountID: req.AccountId, Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	out := &financev1.ListJournalEntriesResponse{Total: int32(total)}
	for i := range rows {
		out.Entries = append(out.Entries, entryPB(&rows[i]))
	}
	return out, nil
}

func (s *journalServer) GetJournalEntry(ctx context.Context, req *financev1.IdRequest) (*financev1.JournalEntry, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	e, err := s.svc.Journals.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return entryPB(e), nil
}

func (s *journalServer) CreateJournalEntry(ctx context.Context, req *financev1.JournalEntryInput) (*financev1.JournalEntry, error) {
	p, err := writeOrApprove(ctx, req.Post)
	if err != nil {
		return nil, err
	}
	in, err := entryInputFrom(req)
	if err != nil {
		return nil, err
	}
	e, err := s.svc.Journals.Create(ctx, p.UserID, in)
	if err != nil {
		return nil, err
	}
	return entryPB(e), nil
}

func (s *journalServer) UpdateJournalEntry(ctx context.Context, req *financev1.UpdateJournalEntryRequest) (*financev1.JournalEntry, error) {
	p, err := writeOrApprove(ctx, req.GetInput().GetPost())
	if err != nil {
		return nil, err
	}
	in, err := entryInputFrom(req.Input)
	if err != nil {
		return nil, err
	}
	e, err := s.svc.Journals.Update(ctx, p.UserID, req.Id, in)
	if err != nil {
		return nil, err
	}
	return entryPB(e), nil
}

func (s *journalServer) DeleteJournalEntry(ctx context.Context, req *financev1.IdRequest) (*emptypb.Empty, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	return empty, s.svc.Journals.Delete(ctx, p.UserID, req.Id)
}

func (s *journalServer) PostJournalEntry(ctx context.Context, req *financev1.IdRequest) (*financev1.JournalEntry, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	e, err := s.svc.Journals.Post(ctx, p.UserID, req.Id)
	if err != nil {
		return nil, err
	}
	return entryPB(e), nil
}

func (s *journalServer) ListPostingLogs(ctx context.Context, req *financev1.ListPostingLogsRequest) (*financev1.ListPostingLogsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Journals.ListPostingLogs(ctx, req.Status, req.EventCode, int(req.Limit))
	if err != nil {
		return nil, err
	}
	out := &financev1.ListPostingLogsResponse{}
	for i := range rows {
		out.Logs = append(out.Logs, postingLogPB(&rows[i]))
	}
	return out, nil
}

func (s *journalServer) RetryPosting(ctx context.Context, req *financev1.IdRequest) (*financev1.PostingLog, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	l, err := s.svc.Journals.RetryPosting(ctx, p.UserID, req.Id)
	if err != nil {
		return nil, err
	}
	return postingLogPB(l), nil
}

// ─── CashBankService ────────────────────────────────────────

type cashBankServer struct {
	financev1.UnimplementedCashBankServiceServer
	svc Services
}

func (s *cashBankServer) ListCashBankAccounts(ctx context.Context, _ *emptypb.Empty) (*financev1.ListCashBankAccountsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.CashBank.Accounts(ctx)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListCashBankAccountsResponse{}
	var total domain.Money
	for _, b := range rows {
		total += b.Balance
		out.Accounts = append(out.Accounts, &financev1.CashBankAccount{Id: b.ID, Code: b.Code, CodeDisplay: domain.FormatAccountCodeDisplay(b.Code), Name: b.Name,
			NormalBalance: sidePB(b.NormalBalance), Balance: b.Balance.String(), MovementCount: int32(b.MovementCount)})
	}
	out.TotalBalance = total.String()
	return out, nil
}

func (s *cashBankServer) GetCashBankLedger(ctx context.Context, req *financev1.LedgerRequest) (*financev1.Ledger, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	l, err := s.svc.CashBank.Ledger(ctx, req.AccountId, req.DateFrom, req.DateTo)
	if err != nil {
		return nil, err
	}
	return ledgerPB(l), nil
}

func movementKind(k financev1.CashMovementKind) string {
	if k == financev1.CashMovementKind_CASH_IN {
		return app.KindCashIn
	}
	return app.KindCashOut
}

func movementFilter(req *financev1.ListCashMovementsRequest) domain.JournalFilter {
	limit, offset := app.Page(int(req.Page), int(req.PageSize))
	return domain.JournalFilter{Search: req.Search, DateFrom: req.DateFrom, DateTo: req.DateTo, Limit: limit, Offset: offset}
}

func (s *cashBankServer) ListCashMovements(ctx context.Context, req *financev1.ListCashMovementsRequest) (*financev1.ListCashMovementsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, total, err := s.svc.CashBank.ListMovements(ctx, movementKind(req.Kind), movementFilter(req))
	if err != nil {
		return nil, err
	}
	out := &financev1.ListCashMovementsResponse{Total: int32(total)}
	for i := range rows {
		out.Movements = append(out.Movements, movementPB(&rows[i]))
	}
	return out, nil
}

func (s *cashBankServer) CreateCashMovement(ctx context.Context, req *financev1.CashMovementInput) (*financev1.CashMovement, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	amount, err := money("amount", req.Amount)
	if err != nil {
		return nil, err
	}
	m, err := s.svc.CashBank.CreateMovement(ctx, p.UserID, app.MovementInput{Kind: movementKind(req.Kind), EntryDate: req.EntryDate, Amount: amount,
		CashAccountID: req.CashAccountId, OffsetAccountID: req.OffsetAccountId, Description: req.Description, Memo: req.Memo})
	if err != nil {
		return nil, err
	}
	return movementPB(m), nil
}

func (s *cashBankServer) ListCashTransfers(ctx context.Context, req *financev1.ListCashMovementsRequest) (*financev1.ListCashTransfersResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, total, err := s.svc.CashBank.ListMovements(ctx, app.KindCashTransfer, movementFilter(req))
	if err != nil {
		return nil, err
	}
	out := &financev1.ListCashTransfersResponse{Total: int32(total)}
	for i := range rows {
		out.Transfers = append(out.Transfers, transferPB(&rows[i]))
	}
	return out, nil
}

func (s *cashBankServer) CreateCashTransfer(ctx context.Context, req *financev1.CashTransferInput) (*financev1.CashTransfer, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	amount, err := money("amount", req.Amount)
	if err != nil {
		return nil, err
	}
	m, err := s.svc.CashBank.CreateTransfer(ctx, p.UserID, app.MovementInput{EntryDate: req.EntryDate, Amount: amount, CashAccountID: req.FromAccountId,
		OffsetAccountID: req.ToAccountId, Description: req.Description, Memo: req.Memo})
	if err != nil {
		return nil, err
	}
	return transferPB(m), nil
}

// ─── PayableService and ReceivableService ───────────────────

// ledger holds the handlers both services share, bound to one kind.
type ledger struct {
	svc  Services
	kind string
}

func (l ledger) invoice(i *domain.Invoice) *financev1.Invoice {
	return invoicePB(i, l.svc.Ledgers.PaymentStatus(*i))
}

func (l ledger) invoices(rows []domain.Invoice, total int) *financev1.ListInvoicesResponse {
	out := &financev1.ListInvoicesResponse{Total: int32(total)}
	for i := range rows {
		out.Invoices = append(out.Invoices, l.invoice(&rows[i]))
	}
	return out
}

func (l ledger) listInvoices(ctx context.Context, req *financev1.ListInvoicesRequest) (*financev1.ListInvoicesResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	limit, offset := app.Page(int(req.Page), int(req.PageSize))
	rows, total, err := l.svc.Ledgers.ListInvoices(ctx, domain.InvoiceFilter{Kind: l.kind, Status: docStatusFrom(req.Status), Search: req.Search,
		From: req.DateFrom, To: req.DateTo, PartyName: req.PartyName, Limit: limit, Offset: offset}, req.PaymentStatus)
	if err != nil {
		return nil, err
	}
	return l.invoices(rows, total), nil
}

func (l ledger) getInvoice(ctx context.Context, req *financev1.IdRequest) (*financev1.Invoice, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	inv, err := l.svc.Ledgers.GetInvoice(ctx, l.kind, req.Id)
	if err != nil {
		return nil, err
	}
	return l.invoice(inv), nil
}

func (l ledger) createInvoice(ctx context.Context, req *financev1.InvoiceInput) (*financev1.Invoice, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	in, err := invoiceInputFrom(req)
	if err != nil {
		return nil, err
	}
	inv, err := l.svc.Ledgers.CreateInvoice(ctx, l.kind, p.UserID, in)
	if err != nil {
		return nil, err
	}
	return l.invoice(inv), nil
}

func (l ledger) updateInvoice(ctx context.Context, req *financev1.UpdateInvoiceRequest) (*financev1.Invoice, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	in, err := invoiceInputFrom(req.Input)
	if err != nil {
		return nil, err
	}
	inv, err := l.svc.Ledgers.UpdateInvoice(ctx, l.kind, p.UserID, req.Id, in)
	if err != nil {
		return nil, err
	}
	return l.invoice(inv), nil
}

func (l ledger) deleteInvoice(ctx context.Context, req *financev1.IdRequest) (*emptypb.Empty, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	return empty, l.svc.Ledgers.DeleteInvoice(ctx, l.kind, p.UserID, req.Id)
}

func (l ledger) postInvoice(ctx context.Context, req *financev1.IdRequest) (*financev1.InvoiceResult, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	inv, note, err := l.svc.Ledgers.PostInvoice(ctx, l.kind, p.UserID, req.Id)
	if err != nil {
		return nil, err
	}
	return &financev1.InvoiceResult{Invoice: l.invoice(inv), Note: note}, nil
}

func (l ledger) open(ctx context.Context) (*financev1.ListInvoicesResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := l.svc.Ledgers.OpenInvoices(ctx, l.kind)
	if err != nil {
		return nil, err
	}
	return l.invoices(rows, len(rows)), nil
}

func (l ledger) aging(ctx context.Context, req *financev1.AgingRequest) (*financev1.Aging, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	a, err := l.svc.Ledgers.Aging(ctx, l.kind, req.AsOf)
	if err != nil {
		return nil, err
	}
	out := &financev1.Aging{AsOf: a.AsOf, TotalOutstanding: a.Total.String(), Invoices: l.invoices(a.Invoices, 0).Invoices}
	for _, b := range a.Buckets {
		out.Buckets = append(out.Buckets, &financev1.AgingBucket{Bucket: b.Bucket, InvoiceCount: int32(b.Count), Amount: b.Amount.String()})
	}
	return out, nil
}

func (l ledger) listPayments(ctx context.Context, req *financev1.ListPaymentsRequest) (*financev1.ListPaymentsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	limit, offset := app.Page(int(req.Page), int(req.PageSize))
	rows, total, err := l.svc.Ledgers.ListPayments(ctx, domain.PaymentFilter{Kind: l.kind, Search: req.Search, Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	out := &financev1.ListPaymentsResponse{Total: int32(total)}
	for i := range rows {
		out.Payments = append(out.Payments, paymentPB(&rows[i]))
	}
	return out, nil
}

func (l ledger) paymentResult(r *app.PaymentResult) *financev1.PaymentResult {
	out := &financev1.PaymentResult{Payment: paymentPB(&r.Payment), Note: r.Note}
	for i := range r.Invoices {
		out.Invoices = append(out.Invoices, l.invoice(&r.Invoices[i]))
	}
	return out
}

func (l ledger) createPayment(ctx context.Context, req *financev1.PaymentInput) (*financev1.PaymentResult, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	in, err := paymentInputFrom(req)
	if err != nil {
		return nil, err
	}
	r, err := l.svc.Ledgers.CreatePayment(ctx, l.kind, p.UserID, in)
	if err != nil {
		return nil, err
	}
	return l.paymentResult(r), nil
}

func (l ledger) voidPayment(ctx context.Context, req *financev1.VoidPaymentRequest) (*financev1.PaymentResult, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	r, err := l.svc.Ledgers.VoidPayment(ctx, l.kind, p.UserID, req.Id, req.Reason)
	if err != nil {
		return nil, err
	}
	return l.paymentResult(r), nil
}

type payableServer struct {
	financev1.UnimplementedPayableServiceServer
	ledger
}

var _ financev1.PayableServiceServer = (*payableServer)(nil)

func (s *payableServer) ListApInvoices(ctx context.Context, req *financev1.ListInvoicesRequest) (*financev1.ListInvoicesResponse, error) {
	return s.listInvoices(ctx, req)
}
func (s *payableServer) GetApInvoice(ctx context.Context, req *financev1.IdRequest) (*financev1.Invoice, error) {
	return s.getInvoice(ctx, req)
}
func (s *payableServer) CreateApInvoice(ctx context.Context, req *financev1.InvoiceInput) (*financev1.Invoice, error) {
	return s.createInvoice(ctx, req)
}
func (s *payableServer) UpdateApInvoice(ctx context.Context, req *financev1.UpdateInvoiceRequest) (*financev1.Invoice, error) {
	return s.updateInvoice(ctx, req)
}
func (s *payableServer) DeleteApInvoice(ctx context.Context, req *financev1.IdRequest) (*emptypb.Empty, error) {
	return s.deleteInvoice(ctx, req)
}
func (s *payableServer) PostApInvoice(ctx context.Context, req *financev1.IdRequest) (*financev1.InvoiceResult, error) {
	return s.postInvoice(ctx, req)
}
func (s *payableServer) ListApPayable(ctx context.Context, _ *emptypb.Empty) (*financev1.ListInvoicesResponse, error) {
	return s.open(ctx)
}
func (s *payableServer) GetApAging(ctx context.Context, req *financev1.AgingRequest) (*financev1.Aging, error) {
	return s.aging(ctx, req)
}
func (s *payableServer) ListApPayments(ctx context.Context, req *financev1.ListPaymentsRequest) (*financev1.ListPaymentsResponse, error) {
	return s.listPayments(ctx, req)
}
func (s *payableServer) CreateApPayment(ctx context.Context, req *financev1.PaymentInput) (*financev1.PaymentResult, error) {
	return s.createPayment(ctx, req)
}
func (s *payableServer) VoidApPayment(ctx context.Context, req *financev1.VoidPaymentRequest) (*financev1.PaymentResult, error) {
	return s.voidPayment(ctx, req)
}

type receivableServer struct {
	financev1.UnimplementedReceivableServiceServer
	ledger
}

var _ financev1.ReceivableServiceServer = (*receivableServer)(nil)

func (s *receivableServer) ListArInvoices(ctx context.Context, req *financev1.ListInvoicesRequest) (*financev1.ListInvoicesResponse, error) {
	return s.listInvoices(ctx, req)
}
func (s *receivableServer) GetArInvoice(ctx context.Context, req *financev1.IdRequest) (*financev1.Invoice, error) {
	return s.getInvoice(ctx, req)
}
func (s *receivableServer) GetArInvoiceBySalesInvoice(ctx context.Context, req *financev1.GetArInvoiceBySalesInvoiceRequest) (*financev1.Invoice, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	inv, err := s.svc.Ledgers.InvoiceBySales(ctx, req.SalesInvoiceId)
	if err != nil {
		return nil, err
	}
	return s.invoice(inv), nil
}
func (s *receivableServer) CreateArInvoice(ctx context.Context, req *financev1.InvoiceInput) (*financev1.Invoice, error) {
	return s.createInvoice(ctx, req)
}
func (s *receivableServer) UpdateArInvoice(ctx context.Context, req *financev1.UpdateInvoiceRequest) (*financev1.Invoice, error) {
	return s.updateInvoice(ctx, req)
}
func (s *receivableServer) DeleteArInvoice(ctx context.Context, req *financev1.IdRequest) (*emptypb.Empty, error) {
	return s.deleteInvoice(ctx, req)
}
func (s *receivableServer) PostArInvoice(ctx context.Context, req *financev1.IdRequest) (*financev1.InvoiceResult, error) {
	return s.postInvoice(ctx, req)
}
func (s *receivableServer) ListArReceivable(ctx context.Context, _ *emptypb.Empty) (*financev1.ListInvoicesResponse, error) {
	return s.open(ctx)
}
func (s *receivableServer) GetArAging(ctx context.Context, req *financev1.AgingRequest) (*financev1.Aging, error) {
	return s.aging(ctx, req)
}
func (s *receivableServer) ListArReceipts(ctx context.Context, req *financev1.ListPaymentsRequest) (*financev1.ListPaymentsResponse, error) {
	return s.listPayments(ctx, req)
}
func (s *receivableServer) CreateArReceipt(ctx context.Context, req *financev1.PaymentInput) (*financev1.PaymentResult, error) {
	return s.createPayment(ctx, req)
}
func (s *receivableServer) VoidArReceipt(ctx context.Context, req *financev1.VoidPaymentRequest) (*financev1.PaymentResult, error) {
	return s.voidPayment(ctx, req)
}

// ─── ReportService ──────────────────────────────────────────

type reportServer struct {
	financev1.UnimplementedReportServiceServer
	svc Services
}

func (s *reportServer) GetTrialBalance(ctx context.Context, req *financev1.AsOfRequest) (*financev1.TrialBalance, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	tb, err := s.svc.Reports.TrialBalance(ctx, req.AsOf)
	if err != nil {
		return nil, err
	}
	return &financev1.TrialBalance{AsOf: tb.AsOf, Rows: balancesPB(tb.Rows), TotalDebit: tb.TotalDebit.String(), TotalCredit: tb.TotalCredit.String()}, nil
}

func balanceSheetPB(bs *app.BalanceSheet) *financev1.BalanceSheet {
	return &financev1.BalanceSheet{AsOf: bs.AsOf, Assets: balancesPB(bs.Assets), Liabilities: balancesPB(bs.Liabilities), Equity: balancesPB(bs.Equity),
		CurrentYearEarnings: bs.CurrentYearEarnings.String(), TotalAssets: bs.TotalAssets.String(), TotalLiabilities: bs.TotalLiabilities.String(),
		TotalEquity: bs.TotalEquity.String(), TotalLiabilitiesAndEquity: bs.TotalLiabilitiesAndEquity.String()}
}

func (s *reportServer) GetBalanceSheet(ctx context.Context, req *financev1.AsOfRequest) (*financev1.BalanceSheet, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	bs, err := s.svc.Reports.BalanceSheet(ctx, req.AsOf)
	if err != nil {
		return nil, err
	}
	return balanceSheetPB(bs), nil
}

func (s *reportServer) GetIncomeStatement(ctx context.Context, req *financev1.PeriodRequest) (*financev1.IncomeStatement, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	is, err := s.svc.Reports.IncomeStatement(ctx, req.DateFrom, req.DateTo)
	if err != nil {
		return nil, err
	}
	return &financev1.IncomeStatement{DateFrom: is.From, DateTo: is.To, Revenue: balancesPB(is.Revenue), Cogs: balancesPB(is.COGS), Expenses: balancesPB(is.Expenses),
		OtherIncome: balancesPB(is.OtherIncome), OtherExpenses: balancesPB(is.OtherExp), TotalRevenue: is.TotalRevenue.String(), TotalCogs: is.TotalCOGS.String(),
		GrossProfit: is.GrossProfit.String(), TotalExpenses: is.TotalExpenses.String(), OperatingIncome: is.OperatingIncome.String(),
		TotalOtherIncome: is.TotalOtherIncome.String(), TotalOtherExpenses: is.TotalOtherExpenses.String(), NetIncome: is.NetIncome.String()}, nil
}

func sectionPB(c app.CashFlowSection) *financev1.CashFlowSection {
	return &financev1.CashFlowSection{Category: c.Category, Rows: balancesPB(c.Rows), Total: c.Total.String()}
}

func (s *reportServer) GetCashFlow(ctx context.Context, req *financev1.PeriodRequest) (*financev1.CashFlow, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	cf, err := s.svc.Reports.CashFlow(ctx, req.DateFrom, req.DateTo)
	if err != nil {
		return nil, err
	}
	return &financev1.CashFlow{DateFrom: cf.From, DateTo: cf.To, Operating: sectionPB(cf.Operating), Investing: sectionPB(cf.Investing), Financing: sectionPB(cf.Financing),
		NonCash: sectionPB(cf.NonCash), NetCashChange: cf.NetChange.String(), CashOpening: cf.Opening.String(), CashClosing: cf.Closing.String()}, nil
}

func (s *reportServer) ListGeneralLedgerAccounts(ctx context.Context, req *financev1.AsOfRequest) (*financev1.ListGeneralLedgerAccountsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Reports.GeneralLedgerAccounts(ctx, req.AsOf)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListGeneralLedgerAccountsResponse{}
	for _, b := range rows {
		out.Accounts = append(out.Accounts, &financev1.GeneralLedgerAccount{Id: b.ID, Code: b.Code, CodeDisplay: domain.FormatAccountCodeDisplay(b.Code), Name: b.Name,
			AccountTypeCode: b.TypeCode, Balance: b.Balance.String()})
	}
	return out, nil
}

func (s *reportServer) GetGeneralLedger(ctx context.Context, req *financev1.LedgerRequest) (*financev1.Ledger, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	l, err := s.svc.Reports.GeneralLedger(ctx, req.AccountId, req.DateFrom, req.DateTo)
	if err != nil {
		return nil, err
	}
	return ledgerPB(l), nil
}

func (s *reportServer) ListSubsidiaryParties(ctx context.Context, req *financev1.ListSubsidiaryPartiesRequest) (*financev1.ListSubsidiaryPartiesResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Ledgers.SubsidiaryParties(ctx, kindFrom(req.Kind))
	if err != nil {
		return nil, err
	}
	out := &financev1.ListSubsidiaryPartiesResponse{}
	for _, r := range rows {
		out.Parties = append(out.Parties, &financev1.SubsidiaryParty{PartyName: r.PartyName, Outstanding: r.Outstanding.String(), InvoiceCount: int32(r.InvoiceCount)})
	}
	return out, nil
}

func (s *reportServer) GetSubsidiaryLedger(ctx context.Context, req *financev1.SubsidiaryLedgerRequest) (*financev1.SubsidiaryLedger, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	sl, err := s.svc.Ledgers.SubsidiaryLedger(ctx, kindFrom(req.Kind), req.PartyName, req.DateFrom, req.DateTo)
	if err != nil {
		return nil, err
	}
	out := &financev1.SubsidiaryLedger{Kind: kindPB(sl.Kind), PartyName: sl.Party, DateFrom: sl.From, DateTo: sl.To, OpeningBalance: sl.Opening.String(),
		TotalDebit: sl.TotalDebit.String(), TotalCredit: sl.TotalCredit.String(), ClosingBalance: sl.Closing.String()}
	docType, verb := "payment", "Bayar "
	if sl.Kind == domain.KindAR {
		docType, verb = "receipt", "Terima "
	}
	for _, l := range sl.Lines {
		line := &financev1.SubsidiaryLine{LineId: l.ID, EntryDate: l.Date, DocType: docType, DocNo: l.DocNo, Description: l.Description,
			Debit: l.Debit.String(), Credit: l.Credit.String(), SourceId: l.ID, RunningBalance: l.Running.String()}
		if l.IsInvoice {
			line.DocType = "invoice"
		} else {
			line.Description = verb + l.Description
		}
		out.Lines = append(out.Lines, line)
	}
	return out, nil
}

func nameValues(rows []app.NameValue) []*financev1.NameValue {
	out := make([]*financev1.NameValue, len(rows))
	for i, r := range rows {
		out[i] = &financev1.NameValue{Name: r.Name, Value: r.Value.String()}
	}
	return out
}

func (s *reportServer) GetDashboard(ctx context.Context, req *financev1.DashboardRequest) (*financev1.Dashboard, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	d, err := s.svc.Reports.Dashboard(ctx, req.AsOf, req.DateFrom, req.DateTo)
	if err != nil {
		return nil, err
	}
	out := &financev1.Dashboard{AsOf: d.AsOf, DateFrom: d.From, DateTo: d.To,
		Kpis: &financev1.DashboardKpis{
			TotalAssets: d.BalanceSheet.TotalAssets.String(), TotalLiabilities: d.BalanceSheet.TotalLiabilities.String(), TotalEquity: d.BalanceSheet.TotalEquity.String(),
			NetIncome: d.Income.NetIncome.String(), CashBalance: d.Cash.Closing.String(), CashOpening: d.Cash.Opening.String(), CashChange: d.Cash.NetChange.String(),
			TrialBalanceDiff: d.TrialBalanceDiff.String(), PostedEntries: int32(d.PostedEntries), DraftEntries: int32(d.Drafts),
			ReceivableOutstanding: d.Receivable.String(), PayableOutstanding: d.Payable.String(), PendingReimbursements: int32(d.PendingClaims),
			CurrentYearEarnings: d.BalanceSheet.CurrentYearEarnings.String(),
		},
		BalanceComposition: nameValues(d.BalanceComposition), PnlBreakdown: nameValues(d.PnlBreakdown), CashFlowBreakdown: nameValues(d.CashFlowBreakdown),
		TopExpenseAccounts: nameValues(d.TopExpenses)}
	for _, m := range d.MonthlyTrend {
		out.MonthlyTrend = append(out.MonthlyTrend, &financev1.MonthTrend{Month: m.Month, Revenue: m.Revenue.String(), Expense: m.Expense.String(), Net: m.Net.String()})
	}
	return out, nil
}

// ─── AnalysisService ────────────────────────────────────────

type analysisServer struct {
	financev1.UnimplementedAnalysisServiceServer
	svc Services
}

func (s *analysisServer) GetFinancialAnalysis(ctx context.Context, req *financev1.FinancialAnalysisRequest) (*financev1.FinancialAnalysis, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	fa, err := s.svc.Analysis.Analyze(ctx, req.From, req.To, req.Compare, req.CompareMode)
	if err != nil {
		return nil, err
	}
	out := &financev1.FinancialAnalysis{Current: dateRange(fa.From, fa.To),
		Profitability: metricsPB(fa.Profitability), Liquidity: metricsPB(fa.Liquidity), Leverage: metricsPB(fa.Leverage),
		Efficiency: metricsPB(fa.Efficiency), Cash: metricsPB(fa.Cash), ExpenseBreakdown: expensesPB(fa.Expenses, fa.Compare),
		TopMovers: expensesPB(fa.TopMovers, fa.Compare), Revenue: fa.Current.Revenue.String(), NetIncome: fa.Current.NetIncome().String(),
		CashBalance: fa.Current.Cash.String()}
	if fa.Compare {
		out.Previous = dateRange(fa.PrevFrom, fa.PrevTo)
	}
	return out, nil
}

func (s *analysisServer) GenerateFinancialInsight(ctx context.Context, req *financev1.GenerateFinancialInsightRequest) (*financev1.FinancialInsight, error) {
	p, err := read(ctx)
	if err != nil {
		return nil, err
	}
	ins, err := s.svc.Analysis.GenerateInsight(ctx, p.UserID, req.From, req.To, req.Language)
	if err != nil {
		return nil, err
	}
	return insightPB(ins), nil
}

func (s *analysisServer) ListFinancialInsights(ctx context.Context, req *financev1.ListFinancialInsightsRequest) (*financev1.ListFinancialInsightsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Analysis.ListInsights(ctx, int(req.Limit))
	if err != nil {
		return nil, err
	}
	out := &financev1.ListFinancialInsightsResponse{}
	for i := range rows {
		out.Insights = append(out.Insights, insightPB(&rows[i]))
	}
	return out, nil
}
