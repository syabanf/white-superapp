package app

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
)

// auditExportLimit caps one CSV. An auditor narrows by date for more.
const auditExportLimit = 50_000

func (l *Ledgers) ListAuditTrail(ctx context.Context, f domain.AuditFilter) ([]domain.AuditEntry, int, error) {
	f.Search = trim(f.Search)
	return l.Store.ListAuditEntries(ctx, f)
}

// ExportAuditTrail writes the filtered trail as CSV, without the JSON bodies.
func (l *Ledgers) ExportAuditTrail(ctx context.Context, f domain.AuditFilter) ([]byte, error) {
	f.Limit, f.Offset = auditExportLimit, 0
	rows, _, err := l.ListAuditTrail(ctx, f)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"Waktu", "Pengguna", "Aksi", "Dokumen", "ID dokumen", "Hasil", "Keterangan"})
	for _, e := range rows {
		_ = w.Write([]string{e.At, e.ActorName, e.Action, e.DocNo, e.DocID, e.Result, e.Message})
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

// AuditSystem records a change no person asked for: an event or a schedule. A failed
// write only reaches the log, since the change itself has already happened.
func (d Deps) AuditSystem(ctx context.Context, action, docID string, payload any, message string) {
	body, _ := json.Marshal(payload)
	if err := d.Store.InsertAuditEntry(ctx, domain.AuditEntry{ActorName: "Sistem", Action: action, DocID: docID, Result: domain.AuditOK, Message: message, Request: string(body)}); err != nil && d.Logger != nil {
		d.Logger.Error("audit trail", "action", action, "err", err)
	}
}
