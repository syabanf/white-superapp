package domain

import "time"

// Fallback link lifetimes when expires_at is NULL (a link never lives forever).
const (
	SessionLinkLifetime = 14 * 24 * time.Hour // psikotes and interview sessions
	OfferLinkLifetime   = 30 * 24 * time.Hour
)

// IsLinkExpired is isLinkExpired: expires_at decides, else issued_at plus
// maxLifetime (issued_at NULL counts as the epoch). Millisecond precision,
// like Date.getTime().
func IsLinkExpired(expiresAt, issuedAt *time.Time, maxLifetime time.Duration, now time.Time) bool {
	var expires int64
	switch {
	case expiresAt != nil:
		expires = expiresAt.UnixMilli()
	case issuedAt != nil:
		expires = issuedAt.UnixMilli() + maxLifetime.Milliseconds()
	default:
		expires = maxLifetime.Milliseconds()
	}
	return expires < now.UnixMilli()
}

// TooManyRequests is TOO_MANY_REQUESTS of route-helpers.ts.
const TooManyRequests = "Terlalu banyak permintaan, coba lagi sebentar lagi"

// Rate limit defaults of lib/rate-limit.ts: a fixed window that resets one
// minute after its first hit, with tighter buckets for heavy writes.
const (
	DefaultRateLimit = 100
	RateWindow       = time.Minute
	UploadRateLimit  = 6
	AnswerRateLimit  = 6
	ProctorRateLimit = 12
)

// Portal applications per network (portal/submit): 5 per 10 minutes.
const (
	ApplyRateLimit  = 5
	ApplyRateWindow = 10 * time.Minute
)

// PortalKind names a candidate portal and its token rules.
type PortalKind struct {
	Name      string // psikotes | interview | offer
	NotFound  string
	Lifetime  time.Duration
	Expirable []string
}

var (
	PsikotesPortal = PortalKind{Name: "psikotes", NotFound: "Link tes tidak berlaku", Lifetime: SessionLinkLifetime,
		Expirable: []string{"draft", "sent", "in_progress"}}
	InterviewPortal = PortalKind{Name: "interview", NotFound: "Link interview tidak berlaku", Lifetime: SessionLinkLifetime,
		Expirable: []string{"sent", "in_progress"}}
	OfferPortal = PortalKind{Name: "offer", NotFound: "Link penawaran tidak berlaku", Lifetime: OfferLinkLifetime,
		Expirable: []string{"sent", "negotiating"}}
)

// ShouldExpire reports a link that lapsed while still open.
func (k PortalKind) ShouldExpire(status string, expiresAt, issuedAt *time.Time, now time.Time) bool {
	for _, s := range k.Expirable {
		if s == status {
			return IsLinkExpired(expiresAt, issuedAt, k.Lifetime, now)
		}
	}
	return false
}
