// Package crawler is the internal site crawler of the SEO audit: BFS over
// one origin, robots aware, sitemap seeded, with on-page issue checks.
package crawler

type Severity string

const (
	SeverityError   Severity = "ERROR"
	SeverityWarning Severity = "WARNING"
	SeverityNotice  Severity = "NOTICE"
)

type issueDef struct {
	Severity Severity
	Message  string
}

// IssueDefs catalogs the issue codes with their Bahasa Indonesia messages.
var IssueDefs = map[string]issueDef{
	"MISSING_TITLE":             {SeverityError, "Halaman tidak memiliki tag <title>."},
	"TITLE_TOO_LONG":            {SeverityWarning, "Title lebih dari 60 karakter dan berpotensi terpotong di hasil pencarian."},
	"TITLE_TOO_SHORT":           {SeverityWarning, "Title kurang dari 30 karakter."},
	"MISSING_META_DESCRIPTION":  {SeverityWarning, "Meta description tidak ditemukan."},
	"META_DESCRIPTION_TOO_LONG": {SeverityNotice, "Meta description lebih dari 160 karakter."},
	"MISSING_H1":                {SeverityError, "Halaman tidak memiliki heading H1."},
	"MULTIPLE_H1":               {SeverityWarning, "Halaman memiliki lebih dari satu H1."},
	"IMG_MISSING_ALT":           {SeverityWarning, "Gambar tanpa atribut alt."},
	"BROKEN_INTERNAL_LINK":      {SeverityError, "Tautan internal mengarah ke halaman 404."},
	"MISSING_CANONICAL":         {SeverityNotice, "Tag canonical tidak ditemukan."},
	"NOINDEX":                   {SeverityWarning, "Halaman diberi robots noindex."},
	"THIN_CONTENT":              {SeverityWarning, "Konten kurang dari 300 kata."},
	"SLOW_RESPONSE":             {SeverityWarning, "Waktu respons server lebih dari 1,5 detik."},
	"NO_STRUCTURED_DATA":        {SeverityNotice, "Tidak ada structured data (JSON-LD)."},
	"REDIRECT_CHAIN":            {SeverityWarning, "Rantai redirect lebih dari satu lompatan."},
	"MIXED_CONTENT":             {SeverityError, "Aset dimuat melalui HTTP pada halaman HTTPS."},
}

type Issue struct {
	Code     string
	Severity Severity
	Message  string
	// Page where the issue was found.
	URL string
}

type Page struct {
	URL              string
	StatusCode       int
	Title            *string
	MetaDescription  *string
	H1Count          int
	WordCount        int
	Canonical        *string
	Indexable        bool
	ImagesTotal      int
	ImagesMissingAlt int
	InternalLinks    int
	ExternalLinks    int
	LoadMs           *int
	HasJsonLd        bool
	HasHreflang      bool
}

type Result struct {
	Pages        []Page
	Issues       []Issue
	PagesCrawled int
	DurationMs   int64
}

const (
	UserAgent = "MixistAnalyticsBot/1.0 (+https://mixist.id/bot)"
	UAToken   = "mixistanalyticsbot"
)
