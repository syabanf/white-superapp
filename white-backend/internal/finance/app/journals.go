package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Journals is the double-entry core every other use case posts through
// (NüHabit journal.go and posting.go).
type Journals struct{ Deps }

func NewJournals(d Deps) *Journals { return &Journals{d} }

const msgNoOpenPeriod = "Tidak ada fiscal period OPEN untuk tanggal jurnal. Konfigurasi Fiscal Years terlebih dahulu."

// NewEntry is a journal to create.
type NewEntry struct {
	UserID             string
	EntryDate          string
	Description        string
	IsRecon            bool
	Lines              []domain.Line
	Post               bool
	EntryType          string
	SourceModule       string
	SourceEventCode    string
	SourceDocumentType string
	SourceDocumentID   string
}

// periodFor checks the lines' accounts and returns the OPEN period of date.
func periodFor(ctx context.Context, tx domain.Store, date string, lines []domain.Line) (*domain.FiscalPeriod, error) {
	period, err := tx.OpenPeriodFor(ctx, date)
	if err != nil {
		return nil, err
	}
	if period == nil {
		return nil, domain.Bad(msgNoOpenPeriod)
	}
	accountIDs := make([]string, len(lines))
	for i, l := range lines {
		accountIDs[i] = l.AccountID
	}
	if err := checkAccounts(ctx, tx, accountIDs, "Akun COA wajib diisi", "Akun jurnal harus postable (bukan header)"); err != nil {
		return nil, err
	}
	return period, nil
}

// createEntry validates, then in one transaction (a savepoint when s is
// already transactional) resolves the OPEN period, checks the accounts,
// numbers and inserts the entry with its lines.
func createEntry(ctx context.Context, s domain.Store, in NewEntry) (*domain.JournalEntry, error) {
	if err := domain.ValidateBalanced(in.Lines); err != nil {
		return nil, err
	}
	var out *domain.JournalEntry
	err := s.WithTx(ctx, func(tx domain.Store) error {
		period, err := periodFor(ctx, tx, in.EntryDate, in.Lines)
		if err != nil {
			return err
		}
		prefix := domain.EntryNoPrefix(in.EntryDate)
		n, err := tx.NextSequence(ctx, prefix)
		if err != nil {
			return err
		}
		status := domain.StatusDraft
		if in.Post {
			status = domain.StatusPosted
		}
		entryType := in.EntryType
		if entryType == "" {
			entryType = domain.EntryManual
		}
		e := domain.JournalEntry{
			ID: ids.New(), EntryNo: domain.SequenceNo(prefix, n), EntryDate: in.EntryDate, Description: in.Description,
			FiscalPeriodID: period.ID, Status: status, IsRecon: in.IsRecon, EntryType: entryType, SourceModule: in.SourceModule,
			SourceEventCode: in.SourceEventCode, SourceDocumentType: in.SourceDocumentType, SourceDocumentID: in.SourceDocumentID,
			CreatedBy: in.UserID,
		}
		if err := tx.InsertJournalEntry(ctx, e); err != nil {
			return err
		}
		if err := tx.ReplaceJournalLines(ctx, e.ID, in.Lines); err != nil {
			return err
		}
		out, err = tx.GetJournalEntry(ctx, e.ID)
		return err
	})
	return out, err
}

// ─── Manual journals ────────────────────────────────────────

func (j *Journals) List(ctx context.Context, f domain.JournalFilter) ([]domain.JournalEntry, int, error) {
	f.Search = trim(f.Search)
	return j.Store.ListJournalEntries(ctx, f)
}

func (j *Journals) Get(ctx context.Context, id string) (*domain.JournalEntry, error) {
	e, err := j.Store.GetJournalEntry(ctx, id)
	if err == nil && e == nil {
		err = domain.NotFound("Journal entry tidak ditemukan")
	}
	return e, err
}

// EntryInput is the manual journal form.
type EntryInput struct {
	EntryDate   string
	Description string
	IsRecon     bool
	Lines       []domain.Line
	Post        bool
}

func (in *EntryInput) normalize() error {
	day, err := parseDay("entry_date", in.EntryDate)
	if err != nil {
		return err
	}
	in.EntryDate, in.Description = day, trim(in.Description)
	return nil
}

func (j *Journals) Create(ctx context.Context, userID string, in EntryInput) (*domain.JournalEntry, error) {
	if err := in.normalize(); err != nil {
		return nil, err
	}
	return createEntry(ctx, j.Store, NewEntry{UserID: userID, EntryDate: in.EntryDate, Description: in.Description, IsRecon: in.IsRecon, Lines: in.Lines, Post: in.Post})
}

func (j *Journals) Update(ctx context.Context, userID, id string, in EntryInput) (*domain.JournalEntry, error) {
	existing, err := j.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := domain.AssertMutable(domain.NormalizeEntryType(existing.EntryType), existing.Status, existing.IsRecon); err != nil {
		return nil, err
	}
	if err := in.normalize(); err != nil {
		return nil, err
	}
	if err := domain.ValidateBalanced(in.Lines); err != nil {
		return nil, err
	}
	err = j.Store.WithTx(ctx, func(tx domain.Store) error {
		period, err := periodFor(ctx, tx, in.EntryDate, in.Lines)
		if err != nil {
			return err
		}
		e := *existing
		e.EntryDate, e.Description, e.FiscalPeriodID, e.IsRecon = in.EntryDate, in.Description, period.ID, in.IsRecon
		e.Status = domain.StatusDraft
		if in.Post {
			e.Status = domain.StatusPosted
		}
		if err := tx.UpdateJournalHeader(ctx, e, userID); err != nil {
			return err
		}
		return tx.ReplaceJournalLines(ctx, id, in.Lines)
	})
	if err != nil {
		return nil, err
	}
	return j.Get(ctx, id)
}

// Post moves DRAFT -> POSTED inside an OPEN period.
func (j *Journals) Post(ctx context.Context, userID, id string) (*domain.JournalEntry, error) {
	existing, err := j.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	debit, credit := existing.Totals()
	if err := domain.AssertPostable(existing.Status, len(existing.Lines), debit, credit); err != nil {
		return nil, err
	}
	if p, err := j.Store.OpenPeriodFor(ctx, existing.EntryDate); err != nil {
		return nil, err
	} else if p == nil {
		return nil, domain.Bad(msgNoOpenPeriod)
	}
	if err := j.Store.MarkJournalPosted(ctx, id, userID); err != nil {
		return nil, err
	}
	return j.Get(ctx, id)
}

func (j *Journals) Delete(ctx context.Context, userID, id string) error {
	existing, err := j.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := domain.AssertMutable(domain.NormalizeEntryType(existing.EntryType), existing.Status, existing.IsRecon); err != nil {
		return err
	}
	return j.Store.SoftDeleteJournalEntry(ctx, id, userID)
}

// ─── Posting through journal mappings ───────────────────────

// MappingPost is one operational event to journal. It is stored with the
// posting log so a DRAFT or SKIPPED result can be retried as is.
type MappingPost struct {
	UserID        string                    `json:"userId"`
	EventCode     string                    `json:"eventCode"`
	DocumentType  string                    `json:"documentType"`
	DocumentID    string                    `json:"documentId"`
	DocumentLabel string                    `json:"documentLabel"`
	EntryDate     string                    `json:"entryDate"`
	Description   string                    `json:"description"`
	Module        string                    `json:"module"`
	Amounts       domain.Amounts            `json:"amounts"`
	Overrides     map[string]string         `json:"overrides,omitempty"`
	Splits        map[string][]domain.Split `json:"splits,omitempty"`
}

// PostFromMapping posts an AUTO journal keyed by (event, document):
//   - an entry already exists for the source: that entry (already_exists)
//   - no active mapping: skipped
//   - mapping incomplete, zero amount, closed period or invalid account:
//     draft with the reason, nothing written
//   - ready: AUTO POSTED entry
//
// Every outcome except already_exists is recorded in posting_logs. It runs
// in a savepoint when s is transactional.
func PostFromMapping(ctx context.Context, s domain.Store, in MappingPost) (domain.PostResult, error) {
	existing, err := s.FindEntryBySource(ctx, in.EventCode, in.DocumentID)
	if err != nil {
		return domain.PostResult{}, err
	}
	if existing != nil {
		return existingResult(existing), nil
	}
	res, err := buildAndPost(ctx, s, in)
	if err != nil {
		return domain.PostResult{}, err
	}
	if res.Reason == domain.AlreadyExists {
		return res, nil
	}
	request, err := json.Marshal(in)
	if err != nil {
		return domain.PostResult{}, err
	}
	_, err = s.UpsertPostingLog(ctx, domain.PostingLog{
		EventCode: in.EventCode, DocumentType: in.DocumentType, DocumentID: in.DocumentID, DocumentLabel: in.DocumentLabel,
		Status: res.Status, Reason: res.Reason, EntryID: res.EntryID, Request: request,
	})
	return res, err
}

func existingResult(e *domain.JournalEntry) domain.PostResult {
	status := domain.PostDraft
	if e.Status == domain.StatusPosted {
		status = domain.PostPosted
	}
	return domain.PostResult{Status: status, EntryID: e.ID, Reason: domain.AlreadyExists}
}

func buildAndPost(ctx context.Context, s domain.Store, in MappingPost) (domain.PostResult, error) {
	mapping, err := s.GetActiveMapping(ctx, in.EventCode)
	if err != nil {
		return domain.PostResult{}, err
	}
	if mapping == nil {
		return domain.PostResult{Status: domain.PostSkipped, Reason: "Tidak ada journal mapping aktif untuk " + in.EventCode}, nil
	}
	template := domain.WithAccountOverrides(mapping.Template(), in.Overrides)
	lines, reason := domain.BuildFromMapping(template, in.Amounts, in.Description, in.Splits)
	if reason != "" {
		return domain.PostResult{Status: domain.PostDraft, Reason: reason}, nil
	}
	entry, err := createEntry(ctx, s, NewEntry{
		UserID: in.UserID, EntryDate: in.EntryDate, Description: in.Description, Lines: lines, Post: true,
		EntryType: domain.EntryAuto, SourceModule: in.Module, SourceEventCode: in.EventCode,
		SourceDocumentType: in.DocumentType, SourceDocumentID: in.DocumentID,
	})
	if err == nil {
		return domain.PostResult{Status: domain.PostPosted, EntryID: entry.ID}, nil
	}
	var unique *domain.ErrUnique
	if errors.As(err, &unique) {
		again, ferr := s.FindEntryBySource(ctx, in.EventCode, in.DocumentID)
		if ferr != nil {
			return domain.PostResult{}, ferr
		}
		if again != nil {
			return existingResult(again), nil
		}
	}
	if e := apperr.As(err); e.Code == apperr.Validation || e.Code == apperr.NotFound {
		return domain.PostResult{Status: domain.PostDraft, Reason: fmt.Sprintf("Gagal posting jurnal %s: %s", in.EventCode, e.Message)}, nil
	}
	return domain.PostResult{}, err
}

func (j *Journals) ListPostingLogs(ctx context.Context, status, eventCode string, limit int) ([]domain.PostingLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	return j.Store.ListPostingLogs(ctx, status, eventCode, limit)
}

// RetryPosting replays a DRAFT or SKIPPED automatic posting and links the
// new journal to its source document.
func (j *Journals) RetryPosting(ctx context.Context, userID, id string) (*domain.PostingLog, error) {
	log, err := j.Store.GetPostingLog(ctx, id)
	if err != nil {
		return nil, err
	}
	if log == nil {
		return nil, domain.NotFound("Log posting tidak ditemukan")
	}
	if log.Status == domain.PostPosted {
		return log, nil
	}
	var in MappingPost
	if err := json.Unmarshal(log.Request, &in); err != nil {
		return nil, err
	}
	if in.UserID == "" {
		in.UserID = userID
	}
	res, err := PostFromMapping(ctx, j.Store, in)
	if err != nil {
		return nil, err
	}
	if res.Status == domain.PostPosted {
		if err := linkJournal(ctx, j.Store, in.DocumentType, in.DocumentID, res.EntryID, res); err != nil {
			return nil, err
		}
	}
	return j.Store.GetPostingLog(ctx, id)
}

// Source document types that carry their journal id.
const (
	DocInvoice    = "invoice"
	DocPayment    = "payment"
	DocPayrollRun = "payroll_run"
	DocReimbBatch = "reimbursement_batch"
)

// linkJournal stores the journal on the document that produced it.
func linkJournal(ctx context.Context, s domain.Store, docType, docID, entryID string, res domain.PostResult) error {
	switch docType {
	case DocInvoice:
		return s.SetInvoiceJournal(ctx, docID, entryID)
	case DocPayment:
		return s.SetPaymentJournal(ctx, docID, entryID)
	case DocReimbBatch:
		return s.UpdateBatchJournal(ctx, docID, entryID, res.Status, res.Reason)
	}
	return nil
}
