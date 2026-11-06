// Package mail sends transactional email through Resend. Without an API key
// it logs each message instead, so local runs and tests never send mail.
package mail

import (
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/httpx"
)

type Attachment struct {
	Filename string
	Content  []byte
}

type Message struct {
	To      []string
	Subject string
	// Text is the plain body; HTML is derived from it when empty.
	Text        string
	HTML        string
	ReplyTo     string
	Attachments []Attachment
}

type Sender interface {
	Send(ctx context.Context, m Message) error
}

const (
	resendURL   = "https://api.resend.com/emails"
	defaultFrom = "Mixist <noreply@mixist.id>"
)

// Credentials returns the Resend API key and From address. An empty key
// means email is not set up yet: messages are logged instead.
type Credentials func(ctx context.Context) (apiKey, from string, err error)

// Resend posts to the Resend API with credentials read on every send, so a
// key saved in the app takes effect without a restart.
type Resend struct {
	creds  Credentials
	http   *httpx.Client
	logger *slog.Logger
}

// Log records what would have been sent.
type Log struct{ Logger *slog.Logger }

func New(creds Credentials, logger *slog.Logger) *Resend {
	return &Resend{creds: creds, http: httpx.New(15 * time.Second), logger: logger}
}

func (r *Resend) Send(ctx context.Context, m Message) error {
	to := recipients(m.To)
	if len(to) == 0 {
		return nil
	}
	key, from, err := r.creds(ctx)
	if err != nil {
		return fmt.Errorf("resend credentials: %w", err)
	}
	if key == "" {
		return Log{Logger: r.logger}.Send(ctx, m)
	}
	if from == "" {
		from = defaultFrom
	}
	payload := map[string]any{"from": from, "to": to, "subject": m.Subject, "text": m.Text, "html": htmlBody(m)}
	if m.ReplyTo != "" {
		payload["reply_to"] = m.ReplyTo
	}
	if len(m.Attachments) > 0 {
		files := make([]map[string]string, 0, len(m.Attachments))
		for _, a := range m.Attachments {
			files = append(files, map[string]string{"filename": a.Filename, "content": base64.StdEncoding.EncodeToString(a.Content)})
		}
		payload["attachments"] = files
	}
	res, err := r.http.PostJSON(ctx, resendURL, payload, map[string]string{"Authorization": "Bearer " + key})
	if err != nil {
		return err
	}
	if res.Status >= 300 {
		return fmt.Errorf("resend: %s", httpx.Describe(res))
	}
	return nil
}

func (l Log) Send(_ context.Context, m Message) error {
	if l.Logger != nil && len(recipients(m.To)) > 0 {
		l.Logger.Info("mail (Resend belum diatur, tidak dikirim)", "to", strings.Join(recipients(m.To), ","), "subject", m.Subject, "attachments", len(m.Attachments))
	}
	return nil
}

// SendLater sends in the background so a slow mail API never holds a request
// open. Failures are logged; email is best effort next to in-app notices.
func SendLater(s Sender, logger *slog.Logger, m Message) {
	if s == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.Send(ctx, m); err != nil && logger != nil {
			logger.Warn("mail failed", "subject", m.Subject, "err", err)
		}
	}()
}

func recipients(to []string) []string {
	out := make([]string, 0, len(to))
	for _, t := range to {
		if t = strings.TrimSpace(t); strings.Contains(t, "@") {
			out = append(out, t)
		}
	}
	return out
}

func htmlBody(m Message) string {
	if m.HTML != "" {
		return m.HTML
	}
	paras := strings.Split(strings.TrimSpace(m.Text), "\n\n")
	var b strings.Builder
	b.WriteString(`<div style="font-family:Arial,sans-serif;font-size:14px;line-height:1.6;color:#111">`)
	for _, p := range paras {
		b.WriteString("<p>" + linkify(strings.ReplaceAll(html.EscapeString(p), "\n", "<br>")) + "</p>")
	}
	b.WriteString("</div>")
	return b.String()
}

var urlRe = regexp.MustCompile(`https?://[^\s<]+`)

// linkify turns bare http(s) URLs into anchors. Input is already escaped.
func linkify(s string) string {
	return urlRe.ReplaceAllString(s, `<a href="$0">$0</a>`)
}
