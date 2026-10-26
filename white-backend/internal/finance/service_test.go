package finance

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
)

// Undecodable or invalid payloads are acknowledged, never redelivered.
func TestHandlerDropsPermanentFailures(t *testing.T) {
	h := NewHandler(app.Deps{Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))}, nil)
	for _, msg := range []events.Message{
		{Subject: events.HRPayrollPaid, Data: []byte(`{not json`)},
		{Subject: events.HRPayrollPaid, Data: []byte(`{"runId":""}`)},
		{Subject: events.SalesInvoiceIssued, Data: []byte(`{"invoiceId":""}`)},
		{Subject: events.SalesPaymentRecorded, Data: []byte(`{"paymentId":"p"}`)},
		{Subject: "white.unknown", Data: []byte(`{}`)},
	} {
		if err := h.Handle(context.Background(), msg); err != nil {
			t.Errorf("%s %s: %v", msg.Subject, msg.Data, err)
		}
	}
}
