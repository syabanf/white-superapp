package grpcapi

import (
	"time"

	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
	financev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/finance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ts(t time.Time) *timestamppb.Timestamp { return timestamppb.New(t) }

func tsPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

// money parses a proto amount; field names the input for the form.
func money(field, s string) (domain.Money, error) {
	m, err := domain.ParseMoney(s)
	if err != nil {
		return 0, apperr.Field(field, "Nominal tidak valid.")
	}
	return m, nil
}

// ─── enums ──────────────────────────────────────────────────

func sidePB(s string) financev1.EntrySide {
	if s == domain.Credit {
		return financev1.EntrySide_CREDIT
	}
	return financev1.EntrySide_DEBIT
}

func sideFrom(s financev1.EntrySide) string {
	switch s {
	case financev1.EntrySide_DEBIT:
		return domain.Debit
	case financev1.EntrySide_CREDIT:
		return domain.Credit
	}
	return ""
}

func docStatusPB(s string) financev1.DocumentStatus {
	switch s {
	case domain.StatusDraft:
		return financev1.DocumentStatus_DRAFT
	case domain.StatusPosted:
		return financev1.DocumentStatus_POSTED
	case domain.StatusVoid:
		return financev1.DocumentStatus_VOID
	}
	return financev1.DocumentStatus_DOCUMENT_STATUS_UNSPECIFIED
}

func docStatusFrom(s financev1.DocumentStatus) string {
	switch s {
	case financev1.DocumentStatus_DRAFT:
		return domain.StatusDraft
	case financev1.DocumentStatus_POSTED:
		return domain.StatusPosted
	case financev1.DocumentStatus_VOID:
		return domain.StatusVoid
	}
	return ""
}

func entryTypePB(s string) financev1.EntryType {
	switch domain.NormalizeEntryType(s) {
	case domain.EntryOpening:
		return financev1.EntryType_OPENING
	case domain.EntryAuto:
		return financev1.EntryType_AUTO
	}
	return financev1.EntryType_MANUAL
}

func entryTypeFrom(t financev1.EntryType) string {
	switch t {
	case financev1.EntryType_MANUAL:
		return domain.EntryManual
	case financev1.EntryType_OPENING:
		return domain.EntryOpening
	case financev1.EntryType_AUTO:
		return domain.EntryAuto
	}
	return ""
}

func periodStatusPB(s string) financev1.PeriodStatus {
	if s == domain.PeriodOpen {
		return financev1.PeriodStatus_OPEN
	}
	return financev1.PeriodStatus_CLOSED
}

func periodStatusFrom(s financev1.PeriodStatus) string {
	switch s {
	case financev1.PeriodStatus_OPEN:
		return domain.PeriodOpen
	case financev1.PeriodStatus_CLOSED:
		return domain.PeriodClosed
	}
	return ""
}

func kindPB(k string) financev1.LedgerKind {
	if k == domain.KindAR {
		return financev1.LedgerKind_AR
	}
	return financev1.LedgerKind_AP
}

func kindFrom(k financev1.LedgerKind) string {
	if k == financev1.LedgerKind_AR {
		return domain.KindAR
	}
	return domain.KindAP
}

func categoryPB(c string) financev1.ClaimCategory {
	return financev1.ClaimCategory(financev1.ClaimCategory_value[c])
}

func categoryFrom(c financev1.ClaimCategory) string {
	if c == financev1.ClaimCategory_CLAIM_CATEGORY_UNSPECIFIED {
		return ""
	}
	return c.String()
}

func claimStatusPB(s domain.ClaimStatus) financev1.ClaimStatus {
	if s == domain.ClaimDraft {
		return financev1.ClaimStatus_CLAIM_DRAFT
	}
	return financev1.ClaimStatus(financev1.ClaimStatus_value[string(s)])
}

func claimStatusFrom(s financev1.ClaimStatus) domain.ClaimStatus {
	switch s {
	case financev1.ClaimStatus_CLAIM_STATUS_UNSPECIFIED:
		return ""
	case financev1.ClaimStatus_CLAIM_DRAFT:
		return domain.ClaimDraft
	}
	return domain.ClaimStatus(s.String())
}

// ─── accounts ───────────────────────────────────────────────

func accountTypePB(t *domain.AccountType) *financev1.AccountType {
	return &financev1.AccountType{Id: t.ID, Code: t.Code, Name: t.Name, NormalBalance: sidePB(t.NormalBalance), SortOrder: int32(t.SortOrder), IsActive: t.IsActive, AccountCount: int32(t.AccountCount)}
}

func accountPB(a *domain.Account) *financev1.Account {
	out := &financev1.Account{
		Id: a.ID, Code: a.Code, CodeDisplay: domain.FormatAccountCodeDisplay(a.Code), Name: a.Name, AccountTypeId: a.AccountTypeID,
		AccountTypeCode: a.TypeCode, AccountTypeName: a.TypeName, NormalBalance: sidePB(domain.EffectiveNormal(a.NormalBalance, a.IsContra)), Level: int32(a.Level),
		IsPostable: a.IsPostable, IsContra: a.IsContra, IsCashBank: a.IsCashBank, CashFlowCategory: a.CashFlowCategory,
		Description: a.Description, IsActive: a.IsActive, CreatedAt: ts(a.CreatedAt), UpdatedAt: ts(a.UpdatedAt),
	}
	if a.ParentID != "" {
		out.ParentId = &a.ParentID
	}
	return out
}

func postablePB(a domain.Account) *financev1.PostableAccount {
	if a.ID == "" {
		return nil
	}
	return &financev1.PostableAccount{Id: a.ID, Code: a.Code, CodeDisplay: domain.FormatAccountCodeDisplay(a.Code), Name: a.Name, IsCashBank: a.IsCashBank, AccountTypeCode: a.TypeCode}
}

func accountFrom(id string, in *financev1.AccountInput) domain.Account {
	return domain.Account{ID: id, Code: in.GetCode(), Name: in.GetName(), ParentID: in.GetParentId(), AccountTypeID: in.GetAccountTypeId(),
		IsContra: in.GetIsContra(), IsCashBank: in.GetIsCashBank(), CashFlowCategory: in.GetCashFlowCategory(), Description: in.GetDescription(), IsActive: in.GetIsActive()}
}

func mappingPB(m *domain.JournalMapping) *financev1.JournalMapping {
	out := &financev1.JournalMapping{Id: m.ID, EventCode: m.EventCode, Name: m.Name, Description: m.Description, Module: m.Module, IsActive: m.IsActive,
		IsReady: domain.HasRequiredAccounts(m.Template()), UpdatedAt: ts(m.UpdatedAt)}
	for _, l := range m.Lines {
		line := &financev1.JournalMappingLine{Id: l.ID, EntrySide: sidePB(l.Side), LineRole: l.LineRole, AccountCode: l.AccountCode, AccountName: l.AccountName,
			AmountSource: l.AmountSource, SortOrder: int32(l.SortOrder), IsRequired: l.IsRequired}
		if l.AccountID != "" {
			line.AccountId = &l.AccountID
		}
		out.Lines = append(out.Lines, line)
	}
	return out
}

func mappingFrom(id string, in *financev1.JournalMappingInput) (domain.JournalMapping, []domain.MappingLine) {
	m := domain.JournalMapping{ID: id, EventCode: in.GetEventCode(), Name: in.GetName(), Description: in.GetDescription(), Module: in.GetModule(), IsActive: in.GetIsActive()}
	var lines []domain.MappingLine
	for _, l := range in.GetLines() {
		required := true
		if l.IsRequired != nil {
			required = *l.IsRequired
		}
		lines = append(lines, domain.MappingLine{Side: sideFrom(l.EntrySide), LineRole: l.LineRole, AccountID: l.GetAccountId(), AmountSource: l.AmountSource, SortOrder: int(l.GetSortOrder()), IsRequired: required})
	}
	return m, lines
}

// ─── fiscal ─────────────────────────────────────────────────

func periodPB(p *domain.FiscalPeriod) *financev1.FiscalPeriod {
	return &financev1.FiscalPeriod{Id: p.ID, FiscalYearId: p.FiscalYearID, PeriodNo: int32(p.PeriodNo), Name: p.Name, StartDate: p.StartDate, EndDate: p.EndDate,
		Status: periodStatusPB(p.Status), FiscalYearCode: p.YearCode, FiscalYearName: p.YearName, FiscalYearIsActive: p.YearActive,
		PostedCount: int32(p.PostedCount), DraftCount: int32(p.DraftCount)}
}

func periodsPB(ps []domain.FiscalPeriod) []*financev1.FiscalPeriod {
	out := make([]*financev1.FiscalPeriod, len(ps))
	for i := range ps {
		out[i] = periodPB(&ps[i])
	}
	return out
}

func yearPB(y *domain.FiscalYear) *financev1.FiscalYear {
	return &financev1.FiscalYear{Id: y.ID, Code: y.Code, Name: y.Name, StartDate: y.StartDate, EndDate: y.EndDate, IsActive: y.IsActive,
		Periods: periodsPB(y.Periods), OpenPeriodsCount: int32(y.OpenCount()), CreatedAt: ts(y.CreatedAt), UpdatedAt: ts(y.UpdatedAt)}
}

func yearFrom(in *financev1.FiscalYearInput) app.YearInput {
	out := app.YearInput{Code: in.GetCode(), Name: in.GetName(), StartDate: in.GetStartDate(), EndDate: in.GetEndDate(), IsActive: in.GetIsActive()}
	for _, p := range in.GetPeriods() {
		out.Periods = append(out.Periods, domain.PeriodInput{PeriodNo: int(p.PeriodNo), Name: p.Name, StartDate: p.StartDate, EndDate: p.EndDate, Status: periodStatusFrom(p.Status)})
	}
	return out
}

// ─── journals ───────────────────────────────────────────────

func entryPB(e *domain.JournalEntry) *financev1.JournalEntry {
	debit, credit := e.Totals()
	out := &financev1.JournalEntry{
		Id: e.ID, EntryNo: e.EntryNo, EntryDate: e.EntryDate, Description: e.Description, FiscalPeriodId: e.FiscalPeriodID,
		FiscalPeriodName: e.PeriodName, FiscalYearCode: e.YearCode, EntryType: entryTypePB(e.EntryType), Status: docStatusPB(e.Status),
		IsRecon: e.IsRecon, PostedAt: tsPtr(e.PostedAt), PostedBy: e.PostedBy, SourceModule: e.SourceModule, SourceEventCode: e.SourceEventCode,
		SourceDocumentType: e.SourceDocumentType, SourceDocumentId: e.SourceDocumentID, TotalDebit: debit.String(), TotalCredit: credit.String(),
		CanEdit: e.CanEdit(), CreatedAt: ts(e.CreatedAt), UpdatedAt: ts(e.UpdatedAt),
	}
	for _, l := range e.Lines {
		out.Lines = append(out.Lines, &financev1.JournalLine{Id: l.ID, AccountId: l.AccountID, AccountCode: domain.FormatAccountCodeDisplay(l.AccountCode),
			AccountName: l.AccountName, EntrySide: sidePB(l.Side), Amount: l.Amount.String(), Memo: l.Memo, SortOrder: int32(l.SortOrder)})
	}
	return out
}

func linesFrom(in []*financev1.JournalLineInput) ([]domain.Line, error) {
	out := make([]domain.Line, 0, len(in))
	for i, l := range in {
		amount, err := money("lines", l.Amount)
		if err != nil {
			return nil, err
		}
		out = append(out, domain.Line{AccountID: l.AccountId, Side: sideFrom(l.EntrySide), Amount: amount, Memo: l.Memo, SortOrder: (i + 1) * 10})
	}
	return out, nil
}

func entryInputFrom(in *financev1.JournalEntryInput) (app.EntryInput, error) {
	lines, err := linesFrom(in.GetLines())
	return app.EntryInput{EntryDate: in.GetEntryDate(), Description: in.GetDescription(), IsRecon: in.GetIsRecon(), Lines: lines, Post: in.GetPost()}, err
}

func postingLogPB(l *domain.PostingLog) *financev1.PostingLog {
	return &financev1.PostingLog{Id: l.ID, EventCode: l.EventCode, DocumentType: l.DocumentType, DocumentId: l.DocumentID, DocumentLabel: l.DocumentLabel,
		Status: l.Status, Reason: l.Reason, EntryId: l.EntryID, Attempts: int32(l.Attempts), CreatedAt: ts(l.CreatedAt), UpdatedAt: ts(l.UpdatedAt)}
}

// ─── balances and ledgers ───────────────────────────────────

func balancePB(b domain.AccountBalance) *financev1.AccountBalance {
	return &financev1.AccountBalance{Id: b.ID, Code: b.Code, CodeDisplay: domain.FormatAccountCodeDisplay(b.Code), Name: b.Name, AccountTypeCode: b.TypeCode,
		NormalBalance: sidePB(b.NormalBalance), IsContra: b.IsContra, IsCashBank: b.IsCashBank, CashFlowCategory: b.CashFlowCategory, Level: int32(b.Level),
		Debit: b.Debit.String(), Credit: b.Credit.String(), Balance: b.Balance.String()}
}

func balancesPB(rows []domain.AccountBalance) []*financev1.AccountBalance {
	out := make([]*financev1.AccountBalance, len(rows))
	for i, r := range rows {
		out[i] = balancePB(r)
	}
	return out
}

func ledgerPB(l *app.Ledger) *financev1.Ledger {
	out := &financev1.Ledger{Account: postablePB(l.Account), NormalBalance: sidePB(l.Account.NormalBalance), DateFrom: l.DateFrom, DateTo: l.DateTo,
		OpeningBalance: l.Opening.String(), ClosingBalance: l.Closing.String(), TotalDebit: l.TotalDebit.String(), TotalCredit: l.TotalCredit.String()}
	for _, r := range l.Lines {
		out.Lines = append(out.Lines, &financev1.LedgerLine{LineId: r.LineID, EntryId: r.EntryID, EntryNo: r.EntryNo, EntryDate: r.EntryDate, EntryType: entryTypePB(r.EntryType),
			Description: r.Description, Memo: r.Memo, EntrySide: sidePB(r.Side), Debit: r.Debit.String(), Credit: r.Credit.String(), RunningBalance: r.Running.String()})
	}
	return out
}

func movementPB(m *app.Movement) *financev1.CashMovement {
	kind := financev1.CashMovementKind_CASH_OUT
	if m.Kind == app.KindCashIn {
		kind = financev1.CashMovementKind_CASH_IN
	}
	return &financev1.CashMovement{Id: m.Entry.ID, EntryNo: m.Entry.EntryNo, EntryDate: m.Entry.EntryDate, Description: m.Entry.Description, Kind: kind,
		Amount: m.Amount.String(), CashAccount: postablePB(m.Cash), OffsetAccount: postablePB(m.Offset), Status: docStatusPB(m.Entry.Status), CreatedAt: ts(m.Entry.CreatedAt)}
}

func transferPB(m *app.Movement) *financev1.CashTransfer {
	return &financev1.CashTransfer{Id: m.Entry.ID, EntryNo: m.Entry.EntryNo, EntryDate: m.Entry.EntryDate, Description: m.Entry.Description, Amount: m.Amount.String(),
		FromAccount: postablePB(m.Cash), ToAccount: postablePB(m.Offset), Status: docStatusPB(m.Entry.Status), CreatedAt: ts(m.Entry.CreatedAt)}
}

// ─── invoices and payments ──────────────────────────────────

func invoicePB(i *domain.Invoice, paymentStatus string) *financev1.Invoice {
	return &financev1.Invoice{
		Id: i.ID, Kind: kindPB(i.Kind), InvoiceNo: i.InvoiceNo, InvoiceDate: i.InvoiceDate, DueDate: i.DueDate, PartyName: i.PartyName, Reference: i.Reference,
		AccountId: i.AccountID, AccountName: i.AccountName, Currency: i.Currency, Subtotal: i.Subtotal.String(), TaxAmount: i.TaxAmount.String(),
		TotalAmount: i.TotalAmount.String(), Status: docStatusPB(i.Status), Description: i.Description, SalesInvoiceId: i.SalesInvoiceID, ClientId: i.ClientID,
		DealId: i.DealID, AllocatedAmount: i.Allocated.String(), OutstandingAmount: i.Outstanding().String(), PaymentStatus: paymentStatus,
		JournalEntryId: i.JournalEntryID, PostedAt: tsPtr(i.PostedAt), CreatedAt: ts(i.CreatedAt),
	}
}

func invoiceInputFrom(in *financev1.InvoiceInput) (app.InvoiceInput, error) {
	sub, err := money("subtotal", in.GetSubtotal())
	if err != nil {
		return app.InvoiceInput{}, err
	}
	tax, err := money("tax_amount", in.GetTaxAmount())
	return app.InvoiceInput{InvoiceDate: in.GetInvoiceDate(), DueDate: in.GetDueDate(), PartyName: in.GetPartyName(), Reference: in.GetReference(),
		AccountID: in.GetAccountId(), Description: in.GetDescription(), ClientID: in.GetClientId(), Subtotal: sub, TaxAmount: tax}, err
}

func paymentPB(p *domain.Payment) *financev1.Payment {
	out := &financev1.Payment{
		Id: p.ID, Kind: kindPB(p.Kind), PaymentNo: p.PaymentNo, PaymentDate: p.PaymentDate, Amount: p.Amount.String(), Method: p.Method,
		CashAccountId: p.CashAccountID, CashAccountName: p.CashAccountName, ReferenceNumber: p.Reference, Notes: p.Notes, Status: docStatusPB(p.Status),
		PartyName: p.PartyName, VoidReason: p.VoidReason, VoidedAt: tsPtr(p.VoidedAt), JournalEntryId: p.JournalEntryID, CreatedAt: ts(p.CreatedAt),
	}
	for _, a := range p.Allocations {
		out.Allocations = append(out.Allocations, &financev1.Allocation{InvoiceId: a.InvoiceID, InvoiceNo: a.InvoiceNo, Amount: a.Amount.String()})
	}
	return out
}

func paymentInputFrom(in *financev1.PaymentInput) (app.PaymentInput, error) {
	out := app.PaymentInput{PaymentDate: in.GetPaymentDate(), Method: in.GetMethod(), CashAccountID: in.GetCashAccountId(), Reference: in.GetReferenceNumber(), Notes: in.GetNotes()}
	for _, a := range in.GetAllocations() {
		amount, err := money("allocations", a.Amount)
		if err != nil {
			return out, err
		}
		out.Allocations = append(out.Allocations, domain.Allocation{InvoiceID: a.InvoiceId, Amount: amount})
	}
	return out, nil
}

// ─── reimbursement ──────────────────────────────────────────

func claimFrom(in *financev1.ClaimInput) (app.ClaimInput, error) {
	amount, err := money("amount", in.GetAmount())
	return app.ClaimInput{Category: categoryFrom(in.GetCategory()), TransactionDate: in.GetTransactionDate(), Description: in.GetDescription(),
		ClientID: in.GetClientId(), Project: in.GetProject(), Amount: amount}, err
}

func batchPB(b *domain.ReimbursementBatch) *financev1.ReimbursementBatch {
	return &financev1.ReimbursementBatch{Id: b.ID, BatchNo: b.BatchNo, PaidAt: b.PaidAt, CashAccountId: b.CashAccountID, CashAccountName: b.CashAccountName,
		TotalAmount: b.TotalAmount.String(), ClaimCount: int32(b.ClaimCount), JournalEntryId: b.JournalEntryID, JournalStatus: b.JournalStatus,
		JournalReason: b.JournalReason, Notes: b.Notes, CreatedAt: ts(b.CreatedAt)}
}

func optionalMoney(m domain.Money) string {
	if m == 0 {
		return ""
	}
	return m.String()
}

// ─── analysis ───────────────────────────────────────────────

func metricsPB(ms []app.Metric) []*financev1.Metric {
	out := make([]*financev1.Metric, len(ms))
	for i, m := range ms {
		pb := &financev1.Metric{Key: m.Key, Label: m.Label, Value: m.Current.Value, Unavailable: !m.Current.OK, LowerIsBetter: m.LowerIsBetter, Unit: m.Unit}
		if m.Previous != nil && m.Previous.OK && m.Current.OK {
			pb.Delta = metrics.ComputeDelta(m.Current.Value, m.Previous.Value).Proto()
		}
		out[i] = pb
	}
	return out
}

func expensesPB(items []app.ExpenseItem, compare bool) []*financev1.ExpenseItem {
	out := make([]*financev1.ExpenseItem, len(items))
	for i, e := range items {
		pb := &financev1.ExpenseItem{AccountId: e.AccountID, AccountCode: domain.FormatAccountCodeDisplay(e.Code), AccountName: e.Name,
			Amount: e.Amount.String(), PreviousAmount: e.Previous.String(), SharePct: e.SharePct}
		if compare {
			pb.Delta = metrics.ComputeDelta(e.Amount.Float(), e.Previous.Float()).Proto()
		}
		out[i] = pb
	}
	return out
}

func insightPB(i *domain.FinancialInsight) *financev1.FinancialInsight {
	return &financev1.FinancialInsight{Id: i.ID, DateFrom: i.DateFrom, DateTo: i.DateTo, Language: i.Language, Model: i.Model, Content: i.Content,
		CreatedAt: ts(i.CreatedAt), CreatedBy: i.CreatedBy}
}

func dateRange(from, to string) *commonv1.DateRange { return &commonv1.DateRange{From: from, To: to} }
