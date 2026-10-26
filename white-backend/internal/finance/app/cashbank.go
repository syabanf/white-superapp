package app

import (
	"context"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
)

// CashBank covers cash and bank balances, ledgers, cash in, cash out and
// transfers posted as MANUAL POSTED journals (NüHabit service_cashbank.go).
type CashBank struct{ Deps }

func NewCashBank(d Deps) *CashBank { return &CashBank{d} }

// Cash movement kinds (source_document_type of their journals).
const (
	KindCashIn       = "cash_in"
	KindCashOut      = "cash_out"
	KindCashTransfer = "cash_transfer"
)

// Accounts lists active cash/bank accounts with their posted balance.
func (c *CashBank) Accounts(ctx context.Context) ([]domain.AccountBalance, error) {
	all, err := c.Store.AccountBalances(ctx, domain.BalanceWindow{})
	if err != nil {
		return nil, err
	}
	out := []domain.AccountBalance{}
	for _, b := range all {
		if b.IsCashBank {
			out = append(out, b)
		}
	}
	return out, nil
}

// Ledger is a running-balance account ledger.
type Ledger struct {
	Account                 domain.Account
	DateFrom, DateTo        string
	Opening, Closing        domain.Money
	TotalDebit, TotalCredit domain.Money
	Lines                   []LedgerLine
}

type LedgerLine struct {
	domain.LedgerLine
	Debit, Credit, Running domain.Money
}

// buildLedger is the running-balance ledger of one account: posted lines,
// opening = movements before from.
func buildLedger(ctx context.Context, s domain.Store, acc *domain.Account, from, to string, contra bool) (*Ledger, error) {
	normal := domain.AsNormal(acc.NormalBalance)
	l := &Ledger{Account: *acc, DateFrom: from, DateTo: to}
	if from != "" {
		d, c, err := s.AccountTotalsBefore(ctx, acc.ID, from)
		if err != nil {
			return nil, err
		}
		l.Opening = domain.Balance(d, c, normal, contra)
	}
	rows, err := s.LedgerLines(ctx, acc.ID, from, to)
	if err != nil {
		return nil, err
	}
	running := l.Opening
	for _, r := range rows {
		line := LedgerLine{LedgerLine: r}
		if r.Side == domain.Debit {
			line.Debit = r.Amount
		} else {
			line.Credit = r.Amount
		}
		l.TotalDebit += line.Debit
		l.TotalCredit += line.Credit
		running += domain.SignedDelta(r.Side, r.Amount, normal, contra)
		line.Running = running
		l.Lines = append(l.Lines, line)
	}
	l.Closing = running
	return l, nil
}

func ledgerRange(from, to string) (string, string, error) {
	from, err := optionalDay("date_from", from)
	if err != nil {
		return "", "", err
	}
	to, err = optionalDay("date_to", to)
	return from, to, err
}

func (c *CashBank) Ledger(ctx context.Context, accountID, from, to string) (*Ledger, error) {
	from, to, err := ledgerRange(from, to)
	if err != nil {
		return nil, err
	}
	acc, err := c.Store.GetAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if acc == nil || !acc.IsCashBank {
		return nil, domain.NotFound("Akun kas/bank tidak ditemukan")
	}
	return buildLedger(ctx, c.Store, acc, from, to, false)
}

func (c *CashBank) cashAccount(ctx context.Context, id string) (*domain.Account, error) {
	acc, err := c.Store.GetAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	if acc == nil || !acc.IsPostable || !acc.IsCashBank {
		return nil, domain.Bad("Akun kas/bank tidak valid")
	}
	return acc, nil
}

// MovementInput is a cash in/out (Offset = the other account) or a transfer
// (Cash = from, Offset = to).
type MovementInput struct {
	Kind              string
	EntryDate         string
	Amount            domain.Money
	CashAccountID     string
	OffsetAccountID   string
	Description, Memo string
}

// Movement is a cash journal with its two accounts.
type Movement struct {
	Entry  domain.JournalEntry
	Kind   string
	Amount domain.Money
	Cash   domain.Account
	Offset domain.Account
}

func orDefault(s, def string) string {
	if s = trim(s); s != "" {
		return s
	}
	return def
}

// CreateMovement: cash in debits the cash/bank account, cash out credits
// it, against a postable offset account.
func (c *CashBank) CreateMovement(ctx context.Context, userID string, in MovementInput) (*Movement, error) {
	day, err := parseDay("entry_date", in.EntryDate)
	if err != nil {
		return nil, err
	}
	if in.Amount <= 0 {
		return nil, domain.Bad("Amount harus lebih dari 0")
	}
	cash, err := c.cashAccount(ctx, in.CashAccountID)
	if err != nil {
		return nil, err
	}
	if in.OffsetAccountID == in.CashAccountID {
		return nil, domain.Bad("Akun lawan tidak boleh sama dengan akun kas/bank")
	}
	offset, err := c.Store.GetAccount(ctx, in.OffsetAccountID)
	if err != nil {
		return nil, err
	}
	if offset == nil || !offset.IsPostable {
		return nil, domain.Bad("Akun lawan tidak valid / tidak postable")
	}
	isIn := in.Kind == KindCashIn
	label, cashSide, offsetSide, event := "Cash Out", domain.Credit, domain.Debit, "CASH_OUT"
	if isIn {
		label, cashSide, offsetSide, event = "Cash In", domain.Debit, domain.Credit, "CASH_IN"
	}
	desc := orDefault(in.Description, label+": "+cash.Code+" "+cash.Name+" ke "+offset.Code+" "+offset.Name)
	memo := orDefault(in.Memo, label)
	entry, err := createEntry(ctx, c.Store, NewEntry{
		UserID: userID, EntryDate: day, Description: desc, Post: true, SourceModule: "CASH_BANK", SourceEventCode: event, SourceDocumentType: in.Kind,
		Lines: []domain.Line{
			{AccountID: cash.ID, Side: cashSide, Amount: in.Amount, Memo: memo, SortOrder: 10},
			{AccountID: offset.ID, Side: offsetSide, Amount: in.Amount, Memo: memo, SortOrder: 20},
		},
	})
	if err != nil {
		return nil, err
	}
	return &Movement{Entry: *entry, Kind: in.Kind, Amount: in.Amount, Cash: *cash, Offset: *offset}, nil
}

// CreateTransfer debits the destination and credits the source.
func (c *CashBank) CreateTransfer(ctx context.Context, userID string, in MovementInput) (*Movement, error) {
	day, err := parseDay("entry_date", in.EntryDate)
	if err != nil {
		return nil, err
	}
	if in.Amount <= 0 {
		return nil, domain.Bad("Amount harus lebih dari 0")
	}
	if in.CashAccountID == in.OffsetAccountID {
		return nil, domain.Bad("Akun asal dan tujuan tidak boleh sama")
	}
	from, err := c.cashAccount(ctx, in.CashAccountID)
	if err != nil {
		return nil, err
	}
	to, err := c.cashAccount(ctx, in.OffsetAccountID)
	if err != nil {
		return nil, err
	}
	desc := orDefault(in.Description, "Transfer: "+from.Code+" "+from.Name+" ke "+to.Code+" "+to.Name)
	memo := orDefault(in.Memo, "Cash Transfer")
	entry, err := createEntry(ctx, c.Store, NewEntry{
		UserID: userID, EntryDate: day, Description: desc, Post: true, SourceModule: "CASH_BANK", SourceEventCode: "CASH_TRANSFER", SourceDocumentType: KindCashTransfer,
		Lines: []domain.Line{
			{AccountID: to.ID, Side: domain.Debit, Amount: in.Amount, Memo: memo, SortOrder: 10},
			{AccountID: from.ID, Side: domain.Credit, Amount: in.Amount, Memo: memo, SortOrder: 20},
		},
	})
	if err != nil {
		return nil, err
	}
	return &Movement{Entry: *entry, Kind: KindCashTransfer, Amount: in.Amount, Cash: *from, Offset: *to}, nil
}

// ListMovements returns cash in, cash out or transfer journals. For a
// transfer, Cash is the source (credit) and Offset the destination.
func (c *CashBank) ListMovements(ctx context.Context, kind string, f domain.JournalFilter) ([]Movement, int, error) {
	f.SourceDocumentType = kind
	f.Search = trim(f.Search)
	entries, total, err := c.Store.ListJournalEntries(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	accounts, err := c.Store.AccountsByCode(ctx)
	if err != nil {
		return nil, 0, err
	}
	byID := map[string]domain.Account{}
	for _, a := range accounts {
		byID[a.ID] = a
	}
	cashSide := domain.Credit
	if kind == KindCashIn {
		cashSide = domain.Debit
	}
	out := make([]Movement, 0, len(entries))
	for _, e := range entries {
		m := Movement{Entry: e, Kind: kind}
		for _, l := range e.Lines {
			if l.Side == cashSide {
				m.Cash, m.Amount = byID[l.AccountID], l.Amount
			} else {
				m.Offset = byID[l.AccountID]
			}
		}
		out = append(out, m)
	}
	return out, total, nil
}
