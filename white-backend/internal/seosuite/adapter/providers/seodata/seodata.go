// Package seodata is the SEO data source behind the suite: the live
// DataForSEO client, the Apify Google Search Scraper (SERP positions only)
// and the deterministic Indonesian mock. Pick chooses per call from the
// credentials the integrations service holds, so saving keys in the setup
// wizard switches modes without a restart.
package seodata

import (
	"context"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
)

type SeoDataProvider interface {
	// KeywordIdeas expands a seed; limit 0 means the provider default.
	KeywordIdeas(ctx context.Context, seed string, locationCode int32, languageCode string, mode domain.ResearchMode, limit int) ([]domain.KeywordIdea, error)
	// KeywordMetrics returns one idea per input keyword, in input order.
	KeywordMetrics(ctx context.Context, keywords []string, locationCode int32, languageCode string) ([]domain.KeywordIdea, error)
	SerpRanks(ctx context.Context, items []domain.SerpRankRequest, locationCode int32, languageCode, targetDomain string) ([]domain.RankResult, error)
	BacklinkSummary(ctx context.Context, dom string) (domain.BacklinkSummary, error)
	Backlinks(ctx context.Context, dom string, limit int, mode domain.BacklinkMode) ([]domain.BacklinkItem, error)
	DomainOverview(ctx context.Context, dom string, locationCode int32, languageCode string) (domain.DomainOverview, error)
	DomainKeywords(ctx context.Context, dom string, locationCode int32, languageCode string, limit int) ([]domain.DomainKeyword, error)
	Competitors(ctx context.Context, dom string, locationCode int32, languageCode string) ([]domain.CompetitorSuggestion, error)
}

// Mode names the source that answers: it is also what the research cache
// stores in `provider` and what the UI shows in its demo banner.
type Mode string

const (
	ModeDataForSEO Mode = "dataforseo"
	ModeApify      Mode = "apify"
	ModeMock       Mode = "mock"
)

// Credentials are the decrypted halves of the DATAFORSEO and APIFY
// integrations as GetProviderCredentials returns them.
type Credentials struct {
	DataForSEOLogin    string
	DataForSEOPassword string
	ApifyToken         string
	// ApifyActors is the optional JSON override map (the APIFY public id).
	ApifyActors string
}

func (c Credentials) Mode() Mode {
	switch {
	case c.DataForSEOLogin != "" && c.DataForSEOPassword != "":
		return ModeDataForSEO
	case c.ApifyToken != "":
		return ModeApify
	}
	return ModeMock
}

type Options struct {
	// USDToIDR converts DataForSEO prices (default 16000).
	USDToIDR float64
	Clock    clock.Clock
	// Base URLs, overridable for tests.
	DataForSEOBase string
	ApifyBase      string
}

func (o Options) withDefaults() Options {
	if o.USDToIDR == 0 {
		o.USDToIDR = 16000
	}
	if o.Clock == nil {
		o.Clock = clock.Real{}
	}
	if o.DataForSEOBase == "" {
		o.DataForSEOBase = DataForSEOBase
	}
	if o.ApifyBase == "" {
		o.ApifyBase = ApifyBase
	}
	return o
}

// Picker builds the provider for a set of credentials.
type Picker struct {
	opts Options
	mock *Mock
}

func NewPicker(opts Options) *Picker {
	opts = opts.withDefaults()
	return &Picker{opts: opts, mock: NewMock(opts.Clock)}
}

// Pick: DataForSEO when both halves exist, otherwise the mock with live SERP
// positions from Apify when a token exists, otherwise the full mock.
func (p *Picker) Pick(c Credentials) (SeoDataProvider, Mode) {
	switch c.Mode() {
	case ModeDataForSEO:
		return NewDataForSEO(c.DataForSEOLogin, c.DataForSEOPassword, p.opts), ModeDataForSEO
	case ModeApify:
		return &apifyMix{Mock: p.mock, serp: NewApifySerp(c.ApifyToken, c.ApifyActors, p.opts.ApifyBase)}, ModeApify
	}
	return p.mock, ModeMock
}

// apifyMix is the mock everywhere except SerpRanks.
type apifyMix struct {
	*Mock
	serp *ApifySerp
}

func (m *apifyMix) SerpRanks(ctx context.Context, items []domain.SerpRankRequest, locationCode int32, languageCode, targetDomain string) ([]domain.RankResult, error) {
	return m.serp.SerpRanks(ctx, items, locationCode, languageCode, targetDomain)
}

// Source resolves the provider for the current credentials on every call.
type Source interface {
	Provider(ctx context.Context) (SeoDataProvider, Mode, error)
}

type CredentialsFunc func(ctx context.Context) (Credentials, error)

type source struct {
	creds  CredentialsFunc
	picker *Picker
}

func NewSource(creds CredentialsFunc, picker *Picker) Source {
	return &source{creds: creds, picker: picker}
}

func (s *source) Provider(ctx context.Context) (SeoDataProvider, Mode, error) {
	c, err := s.creds(ctx)
	if err != nil {
		return nil, "", err
	}
	p, mode := s.picker.Pick(c)
	return p, mode, nil
}

// Static always answers with one provider (tests, local runs without integrations).
func Static(p SeoDataProvider, mode Mode) Source {
	return staticSource{p: p, mode: mode}
}

type staticSource struct {
	p    SeoDataProvider
	mode Mode
}

func (s staticSource) Provider(context.Context) (SeoDataProvider, Mode, error) {
	return s.p, s.mode, nil
}
