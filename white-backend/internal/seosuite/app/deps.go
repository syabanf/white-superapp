// Package app holds the SEO suite use cases: keywords and ranks, research,
// backlinks, competitor domains, the daily job and the report numbers.
package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// User-facing copy, identical to features/seo-suite/strings.ts.
const (
	msgUnauthorizedClient = "Anda tidak punya akses ke klien ini."
	msgViewer             = "Hanya peran ADMIN/MANAGER yang dapat mengubah."
	msgNoProperty         = "Properti SEO tidak ditemukan."
	msgThrottled          = "Pembaruan terakhir kurang dari 10 menit lalu. Coba lagi beberapa menit."
	msgKeywordsInvalid    = "Masukkan minimal satu kata kunci (maks. 100 per tambah)."
	msgDomainInvalid      = "Format domain tidak valid."
	msgDomainExists       = "Domain sudah ada."
	msgDomainMax          = "Maksimal 5 kompetitor domain."
	msgOthers             = "Lainnya"
	msgRankDropTitle      = "Peringkat turun"
	msgBacklinkLostTitle  = "Backlink hilang"
)

// Access is what identity answers for one user and client.
type Access struct {
	Role       string
	CanManage  bool
	ClientSlug string
}

type AccessChecker interface {
	Check(ctx context.Context, userID, clientID string) (*Access, error)
}

// Property is the slice of analytics' SEO property the suite needs.
type Property struct {
	ID           string
	ClientID     string
	SiteURL      string
	LocationCode int32
	LanguageCode string
	CreatedAt    time.Time
}

func (p Property) Host() string { return domain.SiteHost(p.SiteURL) }

type PropertySource interface {
	ListProperties(ctx context.Context, clientID string) ([]Property, error)
}

// Notification goes to the client's whole audience (admins plus members).
type Notification struct {
	ClientID string
	Type     string
	Title    string
	Body     string
	Href     string
}

type Notifier interface {
	Notify(ctx context.Context, n Notification) error
}

// errText is what the old errorMessage returned: the user-facing message of
// an application error, otherwise the raw error text.
func errText(err error) string {
	var e *apperr.Error
	if errors.As(err, &e) {
		return e.Message
	}
	return err.Error()
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func ptr[T any](v T) *T { return &v }

// countMsg is the "{n} kata kunci" / "{n} domain" job message.
func countMsg(n int32, unit string) string { return fmt.Sprintf("%d %s", n, unit) }
