// Package events is the asynchronous seam between services: NATS JetStream
// on one stream ("WHITE", subjects white.>), JSON payloads, durable
// consumers with explicit acks. Without NATS_URL a no-op publisher is used so
// a single service still runs on its own.
package events

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

const (
	Stream        = "WHITE"
	SubjectPrefix = "white."

	// Lifecycle events published by identity.
	ClientCreated = "white.client.created"
	ClientDeleted = "white.client.deleted"
	UserDeleted   = "white.user.deleted"
	// Published by integrations when credentials change (cache bust).
	IntegrationUpdated = "white.integration.updated"
	ConnectionDeleted  = "white.connection.deleted"
	// Ticks published by the scheduler. Daily fan-out is per client and per
	// consuming service so each can scale and retry on its own.
	SyncDailyAnalytics = "white.sync.daily.analytics"
	SyncDailySeoSuite  = "white.sync.daily.seosuite"
	PublishTick        = "white.publish.tick"
	RetentionTick      = "white.retention.tick"

	// ERP ticks (TickPayload). Services schedule their own jobs off these.
	ERPTick15m    = "white.erp.tick.15m"
	ERPTickHourly = "white.erp.tick.hourly"
	ERPTickDaily  = "white.erp.tick.daily"

	// ERP domain events (payload types live with the publishing service's
	// proto or in the service package; consumers decode JSON).
	HRPayrollPaid            = "white.hr.payroll.paid"
	SalesInvoiceIssued       = "white.sales.invoice.issued"
	SalesPaymentRecorded     = "white.sales.payment.recorded"
	SalesDealWon             = "white.sales.deal.won"
	FinanceReimbursementPaid = "white.finance.reimbursement.paid"
	// Payload: recruitment domain.CandidateHiredPayload.
	RecruitmentCandidateHired = "white.recruitment.candidate.hired"

	// Published by publishing when a post reaches a final publish outcome
	// (PublishPostPayload).
	PublishingPostPublished = "white.publishing.post.published"
	PublishingPostFailed    = "white.publishing.post.failed"
)

type ClientCreatedPayload struct {
	ClientID   string   `json:"clientId"`
	Slug       string   `json:"slug"`
	WebsiteURL string   `json:"websiteUrl,omitempty"`
	Modules    []string `json:"modules"`
}

type ClientDeletedPayload struct {
	ClientID string `json:"clientId"`
	Slug     string `json:"slug"`
}

type UserDeletedPayload struct {
	UserID string `json:"userId"`
}

type IntegrationUpdatedPayload struct {
	Key string `json:"key"`
}

type ConnectionDeletedPayload struct {
	ClientID     string `json:"clientId"`
	ConnectionID string `json:"connectionId"`
}

type SyncDailyPayload struct {
	ClientID string    `json:"clientId"`
	Slug     string    `json:"slug"`
	RunAt    time.Time `json:"runAt"`
}

// PublishPostPayload reports a post's final publish outcome. ExternalID and
// Permalink come from the first published target; Error from the first
// failed one.
type PublishPostPayload struct {
	PostID     string    `json:"postId"`
	ClientID   string    `json:"clientId"`
	Status     string    `json:"status"`
	ExternalID string    `json:"externalId,omitempty"`
	Permalink  string    `json:"permalink,omitempty"`
	Error      string    `json:"error,omitempty"`
	At         time.Time `json:"at"`
}

type TickPayload struct {
	RunAt time.Time `json:"runAt"`
}

type Publisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}

type Message struct {
	Subject string
	Data    []byte
	ID      string
}

func (m Message) Decode(v any) error { return json.Unmarshal(m.Data, v) }

type Handler func(ctx context.Context, msg Message) error

// Bus is a connected JetStream publisher/subscriber.
type Bus struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	logger *slog.Logger
}

func Connect(ctx context.Context, url string, logger *slog.Logger) (*Bus, error) {
	nc, err := nats.Connect(url, nats.Name("white"), nats.MaxReconnects(-1), nats.ReconnectWait(2*time.Second))
	if err != nil {
		return nil, err
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, err
	}
	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      Stream,
		Subjects:  []string{SubjectPrefix + ">"},
		Retention: jetstream.LimitsPolicy,
		MaxAge:    7 * 24 * time.Hour,
		Storage:   jetstream.FileStorage,
	})
	if err != nil {
		nc.Close()
		return nil, err
	}
	return &Bus{nc: nc, js: js, logger: logger}, nil
}

func (b *Bus) Close() { b.nc.Drain() }

func (b *Bus) Publish(ctx context.Context, subject string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg := &nats.Msg{Subject: subject, Data: data, Header: nats.Header{}}
	msg.Header.Set(jetstream.MsgIDHeader, ids.New())
	_, err = b.js.PublishMsg(ctx, msg)
	return err
}

// Subscribe runs a durable pull consumer until ctx ends. A handler error
// NAKs the message with a delay; it is redelivered up to MaxDeliver times.
func (b *Bus) Subscribe(ctx context.Context, durable string, subjects []string, h Handler) error {
	cons, err := b.js.CreateOrUpdateConsumer(ctx, Stream, jetstream.ConsumerConfig{
		Durable:        durable,
		FilterSubjects: subjects,
		AckPolicy:      jetstream.AckExplicitPolicy,
		AckWait:        10 * time.Minute,
		MaxDeliver:     5,
		DeliverPolicy:  jetstream.DeliverNewPolicy,
	})
	if err != nil {
		return err
	}
	cc, err := cons.Consume(func(m jetstream.Msg) {
		msg := Message{Subject: m.Subject(), Data: m.Data(), ID: m.Headers().Get(jetstream.MsgIDHeader)}
		if err := h(ctx, msg); err != nil {
			b.logger.Warn("event handler failed", "subject", m.Subject(), "err", err)
			_ = m.NakWithDelay(30 * time.Second)
			return
		}
		_ = m.Ack()
	})
	if err != nil {
		return err
	}
	<-ctx.Done()
	cc.Stop()
	return nil
}

// Noop is used when NATS_URL is empty: publishes are logged and dropped.
type Noop struct{ Logger *slog.Logger }

func (n Noop) Publish(_ context.Context, subject string, payload any) error {
	if n.Logger != nil {
		n.Logger.Debug("event dropped (no NATS)", "subject", subject)
	}
	return nil
}

var ErrNoBus = errors.New("NATS tidak dikonfigurasi")
