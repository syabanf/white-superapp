package domain

import (
	"bytes"
	"encoding/base64"
	"errors"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Client onboarding: the checklist a won deal starts with, the client
// brief collected on the public portal, and asset uploads.

// Onboarding item kinds that other events complete automatically.
const (
	ItemManual       = "manual"
	ItemBrief        = "brief"
	ItemAssets       = "assets"
	ItemContract     = "contract"
	ItemInvoicePaid  = "invoice_paid"
	ItemCreateClient = "create_client"
)

// Default checklist keys.
const (
	KeyKickoff      = "kickoff_meeting"
	KeyBrandAssets  = "collect_brand_assets"
	KeySocialAccess = "social_access"
	KeySearchAccess = "gsc_ga4_access"
	KeyAdAccess     = "ad_account_access"
	KeySignContract = "sign_contract"
	KeyFirstInvoice = "first_invoice_paid"
	KeyCreateClient = "create_marketing_client"
)

// OnboardingTemplateItem is one checklist entry of a template.
type OnboardingTemplateItem struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	DueDays     int    `json:"due_days"`
	Kind        string `json:"kind"`
}

// DefaultOnboardingItems is the agency checklist.
var DefaultOnboardingItems = []OnboardingTemplateItem{
	{KeyKickoff, "Kickoff meeting", "Jadwalkan kickoff dengan PIC klien: perkenalan tim, tujuan, KPI, dan jalur komunikasi.", 3, ItemManual},
	{KeyBrandAssets, "Kumpulkan aset brand", "Logo, panduan brand, foto produk, dan materi lain lewat portal onboarding.", 5, ItemAssets},
	{KeySocialAccess, "Akses akun media sosial", "Minta akses admin atau partner untuk Instagram, Facebook, TikTok, LinkedIn.", 5, ItemManual},
	{KeySearchAccess, "Akses Google Search Console dan GA4", "Tambahkan email tim WHITE sebagai pengguna Search Console dan GA4.", 7, ItemManual},
	{KeyAdAccess, "Akses akun iklan", "Akses Meta Business Manager, Google Ads, atau TikTok Ads sesuai layanan.", 7, ItemManual},
	{KeySignContract, "Tanda tangan kontrak", "Kontrak dikirim lewat tautan tanda tangan digital dan sudah ditandatangani klien.", 5, ItemContract},
	{KeyFirstInvoice, "Invoice pertama lunas", "Pembayaran termin pertama sudah diterima.", 14, ItemInvoicePaid},
	{KeyCreateClient, "Buat klien di platform marketing", "Daftarkan klien di platform analytics WHITE agar tim bisa mulai bekerja.", 7, ItemCreateClient},
}

// ItemKind returns the kind of a checklist key from the default template,
// so templates edited by admins keep their automation by key.
func ItemKind(key string, templateKind string) string {
	if templateKind != "" {
		return templateKind
	}
	for _, it := range DefaultOnboardingItems {
		if it.Key == key {
			return it.Kind
		}
	}
	return ItemManual
}

// Onboarding statuses and item statuses.
const (
	OnboardingActive    = "aktif"
	OnboardingDone      = "selesai"
	OnboardingCancelled = "batal"

	ItemTodo    = "todo"
	ItemDone    = "done"
	ItemSkipped = "skipped"
)

// OnboardingProgress is the share of done or skipped items, in percent.
func OnboardingProgress(statuses []string) int {
	if len(statuses) == 0 {
		return 0
	}
	done := 0
	for _, s := range statuses {
		if s == ItemDone || s == ItemSkipped {
			done++
		}
	}
	return done * 100 / len(statuses)
}

// BriefFields are the fields of the client brief form, in display order.
var BriefFields = []FormField{
	{Key: "brand_name", Label: "Nama brand", Type: "text", Required: true},
	{Key: "brand_description", Label: "Tentang brand", Type: "textarea", Required: true},
	{Key: "goals", Label: "Tujuan kerja sama (mis. awareness, leads, penjualan)", Type: "textarea", Required: true},
	{Key: "target_audience", Label: "Target audiens", Type: "textarea", Required: true},
	{Key: "competitors", Label: "Kompetitor utama", Type: "textarea"},
	{Key: "brand_colors", Label: "Warna dan gaya visual", Type: "text"},
	{Key: "instagram", Label: "Akun Instagram", Type: "text"},
	{Key: "tiktok", Label: "Akun TikTok", Type: "text"},
	{Key: "facebook", Label: "Halaman Facebook", Type: "text"},
	{Key: "linkedin", Label: "Halaman LinkedIn", Type: "text"},
	{Key: "youtube", Label: "Kanal YouTube", Type: "text"},
	{Key: "website", Label: "Website", Type: "text"},
	{Key: "notes", Label: "Catatan tambahan", Type: "textarea"},
}

// CleanBrief validates the brief: required fields present, known keys
// only, text trimmed and capped.
func CleanBrief(in map[string]string) (map[string]string, map[string]string) {
	out := map[string]string{}
	errs := map[string]string{}
	for _, f := range BriefFields {
		v := JSTrim(in[f.Key])
		limit := 200
		if f.Type == "textarea" {
			limit = 3000
		}
		v = SliceUTF16(v, limit)
		if v == "" {
			if f.Required {
				errs[f.Key] = f.Label + " wajib diisi"
			}
			continue
		}
		out[f.Key] = v
	}
	return out, errs
}

// Upload limits and the allowlist of the onboarding portal.
const MaxAssetBytes = 15 << 20

var assetTypes = map[string]string{
	"image/png": "png", "image/jpeg": "jpg", "image/webp": "webp", "image/gif": "gif",
	"application/pdf": "pdf", "application/zip": "zip", "image/svg+xml": "svg",
}

var (
	ErrAssetType  = errors.New("Jenis file tidak didukung. Gunakan PNG, JPG, WEBP, GIF, SVG, PDF, atau ZIP.")
	ErrAssetSize  = errors.New("Ukuran file maksimal 15 MB.")
	ErrAssetEmpty = errors.New("File kosong.")
)

// SniffAsset checks the magic bytes and returns the content type and
// extension. The browser's MIME type is never trusted.
func SniffAsset(data []byte, fileName string) (string, string, error) {
	if len(data) == 0 {
		return "", "", ErrAssetEmpty
	}
	if len(data) > MaxAssetBytes {
		return "", "", ErrAssetSize
	}
	var ct string
	switch {
	case bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}):
		ct = "image/png"
	case bytes.HasPrefix(data, []byte{0xff, 0xd8, 0xff}):
		ct = "image/jpeg"
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		ct = "image/webp"
	case bytes.HasPrefix(data, []byte("GIF87a")), bytes.HasPrefix(data, []byte("GIF89a")):
		ct = "image/gif"
	case bytes.HasPrefix(data, []byte("%PDF-")):
		ct = "application/pdf"
	case bytes.HasPrefix(data, []byte{'P', 'K', 0x03, 0x04}):
		ct = "application/zip"
	case strings.EqualFold(filepath.Ext(fileName), ".svg") && isSVG(data):
		ct = "image/svg+xml"
	default:
		return "", "", ErrAssetType
	}
	return ct, assetTypes[ct], nil
}

var svgScriptRe = regexp.MustCompile(`(?i)<script|on[a-z]+\s*=|javascript:`)

// isSVG accepts an SVG document without scripts or event handlers.
func isSVG(data []byte) bool {
	head := strings.ToLower(string(data[:min(len(data), 1024)]))
	return strings.Contains(head, "<svg") && http.DetectContentType(data) != "application/octet-stream" && !svgScriptRe.Match(data)
}

// SafeFileName keeps a display name without path parts.
func SafeFileName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == '/' || r == '"' {
			return -1
		}
		return r
	}, name)
	if name == "." || name == "" {
		return "file"
	}
	return SliceUTF16(name, 120)
}

var ErrSignature = errors.New("Tanda tangan tidak valid. Gambar ulang tanda tangan Anda.")

// MaxSignatureBytes caps the decoded signature PNG.
const MaxSignatureBytes = 512 << 10

// DecodeSignature reads a "data:image/png;base64,..." URL and checks the
// PNG magic bytes.
func DecodeSignature(dataURL string) ([]byte, error) {
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(dataURL, prefix) {
		return nil, ErrSignature
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(dataURL, prefix))
	if err != nil || len(raw) < 64 || len(raw) > MaxSignatureBytes || !bytes.HasPrefix(raw, []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}) {
		return nil, ErrSignature
	}
	return raw, nil
}

// DueDate is today (WIB) plus n days, as a calendar date.
func DueDate(now time.Time, days int) time.Time {
	n := now.In(Jakarta).AddDate(0, 0, days)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}
