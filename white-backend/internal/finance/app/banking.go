package app

import (
	"context"
	"errors"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Banking reconciles uploaded bank and credit card statements against the
// ledger. A journal line settles one statement row only, which keeps a card
// charge from being recorded twice.
type Banking struct {
	Deps
	ledgers *Ledgers
}

func NewBanking(d Deps) *Banking { return &Banking{Deps: d, ledgers: NewLedgers(d)} }

const (
	maxStatementBytes = 5 << 20
	// Candidates sit within this many days of the statement date.
	matchWindowDays = 7
	// The person verifying an exception may look further than the system does.
	pickWindowDays = 45
)

// ReconAccount is an account that can be reconciled, with its row counts.
type ReconAccount struct {
	domain.AccountBalance
	IsCreditCard bool
	Counts       map[string]int
}

// Accounts lists cash, bank and credit card accounts.
func (b *Banking) Accounts(ctx context.Context) ([]ReconAccount, error) {
	active := true
	accounts, err := b.Store.ListAccounts(ctx, domain.AccountFilter{IsActive: &active})
	if err != nil {
		return nil, err
	}
	balances, err := b.Store.AccountBalances(ctx, domain.BalanceWindow{})
	if err != nil {
		return nil, err
	}
	balance := map[string]domain.AccountBalance{}
	for _, x := range balances {
		balance[x.ID] = x
	}
	out := []ReconAccount{}
	for _, a := range accounts {
		if !a.IsPostable || (!a.IsCashBank && !a.IsCreditCard) {
			continue
		}
		counts, err := b.Store.StatementLineCounts(ctx, a.ID)
		if err != nil {
			return nil, err
		}
		row := balance[a.ID]
		row.ID, row.Code, row.Name = a.ID, a.Code, a.Name
		out = append(out, ReconAccount{AccountBalance: row, IsCreditCard: a.IsCreditCard, Counts: counts})
	}
	return out, nil
}

func (b *Banking) reconAccount(ctx context.Context, id string) (*domain.Account, error) {
	acc, err := b.Store.GetAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	if acc == nil || !acc.IsPostable || (!acc.IsCashBank && !acc.IsCreditCard) {
		return nil, apperr.Field("account_id", "Pilih rekening kas, bank, atau kartu kredit")
	}
	return acc, nil
}

// StatementImport counts what an upload did.
type StatementImport struct{ Inserted, Skipped, Matched, Suggested, Unmatched int }

// Import stores a statement file, skips rows uploaded before and runs matching.
func (b *Banking) Import(ctx context.Context, userID, accountID, filename string, data []byte) (*StatementImport, error) {
	if _, err := b.reconAccount(ctx, accountID); err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > maxStatementBytes {
		return nil, apperr.Field("data", "Ukuran file mutasi maksimal 5 MB.")
	}
	rows, err := domain.ParseStatementCSV(data)
	if err != nil {
		return nil, err
	}
	res := &StatementImport{}
	err = b.Store.WithTx(ctx, func(tx domain.Store) error {
		statementID := ids.New()
		if err := tx.InsertBankStatement(ctx, statementID, accountID, domain.CleanFilename(filename), userID); err != nil {
			return err
		}
		for _, r := range rows {
			ok, err := tx.InsertStatementLine(ctx, domain.StatementLine{ID: ids.New(), StatementID: statementID, AccountID: accountID,
				LineDate: r.Date, Description: r.Description, SenderName: r.Sender, Amount: r.Amount, DedupeKey: r.DedupeKey})
			if err != nil {
				return err
			}
			if ok {
				res.Inserted++
			} else {
				res.Skipped++
			}
		}
		return tx.SetBankStatementCount(ctx, statementID, res.Inserted)
	})
	if err != nil {
		return nil, err
	}
	if res.Matched, res.Suggested, res.Unmatched, err = b.Rematch(ctx, accountID); err != nil {
		return nil, err
	}
	return res, nil
}

// Rematch scores every open row of an account. New journals since the last
// upload can settle rows that were unmatched before.
func (b *Banking) Rematch(ctx context.Context, accountID string) (matched, suggested, unmatched int, err error) {
	acc, err := b.reconAccount(ctx, accountID)
	if err != nil {
		return 0, 0, 0, err
	}
	lines, err := b.Store.PendingStatementLines(ctx, accountID)
	if err != nil {
		return 0, 0, 0, err
	}
	taken := map[string]bool{}
	for _, l := range lines {
		cands, err := b.candidates(ctx, *acc, l, matchWindowDays)
		if err != nil {
			return 0, 0, 0, err
		}
		free := cands[:0]
		for _, c := range cands {
			if !taken[c.Ref] {
				free = append(free, c)
			}
		}
		status, top := domain.Decide(free)
		l.Status, l.SuggestedKind, l.SuggestedRef, l.SuggestedLabel, l.SuggestedReason, l.SuggestedScore = status, "", "", "", "", 0
		if top != nil {
			l.SuggestedKind, l.SuggestedRef, l.SuggestedLabel, l.SuggestedReason, l.SuggestedScore = top.Kind, top.Ref, top.Label, top.Reason, top.Score
		}
		switch status {
		case domain.LineMatched:
			taken[top.Ref] = true
			line, err := b.Store.GetJournalLineForMatch(ctx, top.Ref)
			if err != nil {
				return 0, 0, 0, err
			}
			l.MatchedLineID, l.MatchedEntryID = line.LineID, line.EntryID
			matched++
		case domain.LineSuggested:
			suggested++
		default:
			unmatched++
		}
		if err := b.Store.UpdateStatementLine(ctx, l); err != nil {
			return 0, 0, 0, err
		}
	}
	return matched, suggested, unmatched, nil
}

// candidates finds journal lines and open invoices of the same amount, scored.
func (b *Banking) candidates(ctx context.Context, acc domain.Account, l domain.StatementLine, window int) ([]domain.MatchCandidate, error) {
	day, err := time.Parse(time.DateOnly, l.LineDate)
	if err != nil {
		return nil, err
	}
	from, to := day.AddDate(0, 0, -window).Format(time.DateOnly), day.AddDate(0, 0, window).Format(time.DateOnly)
	journal, err := b.Store.OpenJournalLines(ctx, l.AccountID, l.Side(), l.Abs(), from, to)
	if err != nil {
		return nil, err
	}
	out := make([]domain.MatchCandidate, 0, len(journal))
	for _, j := range journal {
		out = append(out, domain.ScoreCandidate(l, domain.MatchCandidate{Kind: domain.SuggestJournal, Ref: j.LineID, Label: j.EntryNo + " · " + j.Description,
			Date: j.EntryDate, Text: j.Description + " " + j.Memo, Amount: j.Amount}))
	}
	// A receipt or payment against an invoice goes through a cash or bank account.
	if !acc.IsCashBank {
		return out, nil
	}
	kind := domain.KindAP
	if l.Amount > 0 {
		kind = domain.KindAR
	}
	invoices, err := b.ledgers.OpenInvoices(ctx, kind)
	if err != nil {
		return nil, err
	}
	for _, inv := range invoices {
		if inv.Outstanding() == l.Abs() {
			out = append(out, domain.ScoreCandidate(l, domain.MatchCandidate{Kind: domain.SuggestInvoice, Ref: inv.ID, Label: inv.InvoiceNo + " · " + inv.PartyName,
				Text: inv.PartyName + " " + inv.InvoiceNo + " " + inv.Reference, Amount: inv.Outstanding()}))
		}
	}
	return out, nil
}

// Candidates is what a person picks from when verifying a row by hand.
func (b *Banking) Candidates(ctx context.Context, lineID string) ([]domain.MatchCandidate, error) {
	l, err := b.line(ctx, lineID)
	if err != nil {
		return nil, err
	}
	acc, err := b.reconAccount(ctx, l.AccountID)
	if err != nil {
		return nil, err
	}
	out, err := b.candidates(ctx, *acc, *l, pickWindowDays)
	if err != nil {
		return nil, err
	}
	domain.Decide(out) // Sorts by score.
	return out, nil
}

func (b *Banking) line(ctx context.Context, id string) (*domain.StatementLine, error) {
	l, err := b.Store.GetStatementLine(ctx, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, domain.NotFound("Baris mutasi tidak ditemukan")
	}
	return l, nil
}

func (b *Banking) openLine(ctx context.Context, id string) (*domain.StatementLine, error) {
	l, err := b.line(ctx, id)
	if err != nil {
		return nil, err
	}
	if l.Status == domain.LineMatched || l.Status == domain.LineIgnored {
		return nil, domain.Conflict("Baris mutasi ini sudah diselesaikan. Batalkan dulu untuk mengubahnya.")
	}
	return l, nil
}

func (b *Banking) settle(ctx context.Context, s domain.Store, l *domain.StatementLine, userID, lineID, entryID, note string) error {
	now := b.now()
	l.Status, l.MatchedLineID, l.MatchedEntryID, l.Note, l.VerifiedBy, l.VerifiedAt = domain.LineMatched, lineID, entryID, note, userID, &now
	if err := s.UpdateStatementLine(ctx, *l); err != nil {
		var unique *domain.ErrUnique
		if errors.As(err, &unique) {
			return domain.Conflict("Jurnal ini sudah dipakai oleh baris mutasi lain.")
		}
		return err
	}
	return nil
}

// MatchJournal settles a row with an existing journal line.
func (b *Banking) MatchJournal(ctx context.Context, userID, lineID, journalLineID string) (*domain.StatementLine, error) {
	l, err := b.openLine(ctx, lineID)
	if err != nil {
		return nil, err
	}
	j, err := b.Store.GetJournalLineForMatch(ctx, journalLineID)
	if err != nil {
		return nil, err
	}
	if j == nil || j.Status != domain.StatusPosted {
		return nil, domain.NotFound("Jurnal tidak ditemukan atau belum diposting")
	}
	if j.AccountID != l.AccountID || j.Side != l.Side() || j.Amount != l.Abs() {
		return nil, domain.Bad("Akun, arah, atau nominal jurnal tidak sama dengan mutasi")
	}
	if err := b.settle(ctx, b.Store, l, userID, j.LineID, j.EntryID, ""); err != nil {
		return nil, err
	}
	return b.Store.GetStatementLine(ctx, lineID)
}

// MatchInvoice settles a row by recording the invoice's receipt or payment.
func (b *Banking) MatchInvoice(ctx context.Context, userID, lineID, invoiceID string) (*domain.StatementLine, error) {
	l, err := b.openLine(ctx, lineID)
	if err != nil {
		return nil, err
	}
	inv, err := b.Store.GetInvoice(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, domain.NotFound("Invoice tidak ditemukan")
	}
	if (l.Amount > 0) != (inv.Kind == domain.KindAR) {
		return nil, domain.Bad("Uang masuk dicocokkan ke piutang, uang keluar ke hutang")
	}
	res, err := b.ledgers.CreatePayment(ctx, inv.Kind, userID, PaymentInput{PaymentDate: l.LineDate, CashAccountID: l.AccountID, Reference: l.Description,
		Notes: "Dari mutasi bank", Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: l.Abs()}}})
	if err != nil {
		return nil, err
	}
	journalLine, entryID := "", res.Payment.JournalEntryID
	if entryID != "" {
		if journalLine, err = b.Store.JournalLineOnAccount(ctx, entryID, l.AccountID, l.Side(), l.Abs()); err != nil {
			return nil, err
		}
	}
	note := res.Payment.PaymentNo
	if entryID == "" {
		note += ": " + res.Note
	}
	if journalLine == "" {
		entryID = ""
	}
	if err := b.settle(ctx, b.Store, l, userID, journalLine, entryID, note); err != nil {
		return nil, err
	}
	return b.Store.GetStatementLine(ctx, lineID)
}

// Record posts a new journal for a row the books do not have yet, such as a
// bank fee or a card charge, against the account a person picks.
func (b *Banking) Record(ctx context.Context, userID, lineID, counterAccountID, description string) (*domain.StatementLine, error) {
	l, err := b.openLine(ctx, lineID)
	if err != nil {
		return nil, err
	}
	counterAccountID = trim(counterAccountID)
	if counterAccountID == "" {
		return nil, apperr.Field("counter_account_id", "Pilih akun lawan")
	}
	if counterAccountID == l.AccountID {
		return nil, apperr.Field("counter_account_id", "Akun lawan harus berbeda dari rekening")
	}
	if err := checkControlLines(ctx, b.Store, []domain.Line{{AccountID: counterAccountID}}); err != nil {
		return nil, err
	}
	if description = trim(description); description == "" {
		description = l.Description
	}
	side, other := domain.Debit, domain.Credit
	if l.Side() == domain.Credit {
		side, other = domain.Credit, domain.Debit
	}
	err = b.Store.WithTx(ctx, func(tx domain.Store) error {
		e, err := createEntry(ctx, tx, NewEntry{UserID: userID, EntryDate: l.LineDate, Description: description, Post: true, SourceModule: "finance",
			SourceDocumentType: "bank_statement_line", SourceDocumentID: l.ID, Lines: []domain.Line{
				{AccountID: l.AccountID, Side: side, Amount: l.Abs(), Memo: l.Description},
				{AccountID: counterAccountID, Side: other, Amount: l.Abs(), Memo: l.Description, SortOrder: 1},
			}})
		if err != nil {
			return err
		}
		line, err := tx.JournalLineOnAccount(ctx, e.ID, l.AccountID, side, l.Abs())
		if err != nil {
			return err
		}
		return b.settle(ctx, tx, l, userID, line, e.ID, "")
	})
	if err != nil {
		return nil, err
	}
	return b.Store.GetStatementLine(ctx, lineID)
}

// Ignore closes a row with a reason, such as a transfer between own accounts
// already covered by another row.
func (b *Banking) Ignore(ctx context.Context, userID, lineID, note string) (*domain.StatementLine, error) {
	l, err := b.openLine(ctx, lineID)
	if err != nil {
		return nil, err
	}
	if note = trim(note); note == "" {
		return nil, apperr.Field("note", "Alasan wajib diisi")
	}
	now := b.now()
	l.Status, l.Note, l.VerifiedBy, l.VerifiedAt = domain.LineIgnored, note, userID, &now
	if err := b.Store.UpdateStatementLine(ctx, *l); err != nil {
		return nil, err
	}
	return b.Store.GetStatementLine(ctx, lineID)
}

// Reopen undoes a match or an ignore. A journal or payment made from the row
// stays, because it may already be in a report; the person reverses it there.
func (b *Banking) Reopen(ctx context.Context, lineID string) (*domain.StatementLine, error) {
	l, err := b.line(ctx, lineID)
	if err != nil {
		return nil, err
	}
	l.Status, l.MatchedLineID, l.MatchedEntryID, l.Note, l.VerifiedBy, l.VerifiedAt = domain.LineUnmatched, "", "", "", "", nil
	l.SuggestedKind, l.SuggestedRef, l.SuggestedLabel, l.SuggestedReason, l.SuggestedScore = "", "", "", "", 0
	if err := b.Store.UpdateStatementLine(ctx, *l); err != nil {
		return nil, err
	}
	return b.Store.GetStatementLine(ctx, lineID)
}

// Lines lists statement rows, newest first.
func (b *Banking) Lines(ctx context.Context, f domain.StatementFilter) ([]domain.StatementLine, int, error) {
	var err error
	if f.DateFrom, f.DateTo, err = ledgerRange(f.DateFrom, f.DateTo); err != nil {
		return nil, 0, err
	}
	limit, offset := Page(f.Page, f.PageSize)
	f.PageSize, f.Page = limit, offset/limit+1
	return b.Store.ListStatementLines(ctx, f)
}
