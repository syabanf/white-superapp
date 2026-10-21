package seodata

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/providers/mockrand"
)

// Mock is the deterministic DataForSEO stand-in: every value is a hash of
// its inputs (same seeds as the Next.js mock), ranks are a stable random
// walk keyed by keyword, target and day.
type Mock struct {
	clk clock.Clock
}

func NewMock(clk clock.Clock) *Mock {
	if clk == nil {
		clk = clock.Real{}
	}
	return &Mock{clk: clk}
}

var (
	modifiers      = []string{"harga", "terdekat", "terbaik", "murah", "review", "2026", "promo", "online", "rekomendasi", "grosir"}
	questionPrefix = []string{"cara", "apa itu", "bagaimana", "kenapa", "berapa harga", "apakah", "dimana beli", "kapan"}
	cities         = []string{"jakarta", "bandung", "surabaya", "yogyakarta", "bali", "medan", "semarang", "makassar", "tangerang", "bekasi"}
	relatedSuffix  = []string{"premium", "lokal", "kekinian", "untuk pemula", "paket", "bisnis", "rumahan", "modern", "tradisional", "keluarga"}
	serpFeatures   = []string{"featured_snippet", "people_also_ask", "images", "video", "local_pack", "shopping", "sitelinks", "reviews", "top_stories", "knowledge_panel"}

	genericDomains  = []string{"tokopedia.com", "shopee.co.id", "kompas.com", "detik.com", "tribunnews.com", "wikipedia.org", "idntimes.com", "liputan6.com", "kumparan.com", "cnnindonesia.com", "medium.com", "blogspot.com", "wordpress.com", "traveloka.com", "rumah123.com", "olx.co.id", "bukalapak.com", "tiket.com", "gramedia.com", "sehatq.com"}
	backlinkDomains = []string{"kompas.com", "detik.com", "tribunnews.com", "medium.com", "blogspot.com", "wordpress.com", "liputan6.com", "kumparan.com", "idntimes.com", "tempo.co", "suara.com", "merdeka.com", "brilio.net", "hipwee.com", "pikiran-rakyat.com", "kaskus.co.id", "quora.com", "reddit.com", "linkedin.com", "pinterest.com", "issuu.com", "weebly.com", "wixsite.com", "zenn.dev", "seo-backlink-cheap.xyz", "linkfarm-id.info", "bestlinks-2024.top", "artikel-murah.site"}
	anchors         = []string{"{brand}", "{brand}", "kunjungi situs", "baca selengkapnya", "klik di sini", "{brand} resmi", "website", "sumber", "{brand}.id", "lihat di sini", "info lengkap"}
)

type industry struct {
	key         string
	match       *regexp.Regexp
	seed        string
	competitors []string
}

var industries = []industry{
	{key: "kopi", match: regexp.MustCompile(`kopi|coffee|cafe|kafe|ngopi|arabika|roast|brew`), seed: "kopi", competitors: []string{"kopikenangan.com", "janjijiwa.com", "fore.coffee", "tomoro.id", "kopisenja.id", "ngopidulu.com"}},
	{key: "properti", match: regexp.MustCompile(`propert|residence|rumah|land|estate|realty|adiharjo|cluster|home`), seed: "rumah dijual", competitors: []string{"rumah123.com", "summarecon.com", "sinarmasland.com", "ciputra.com", "lippokarawaci.co.id", "agungpodomoro.com"}},
	{key: "villa", match: regexp.MustCompile(`villa|bali|hotel|stay|escape|resort|travel|trip`), seed: "villa bali", competitors: []string{"airbnb.com", "booking.com", "balivillas.com", "elitehavens.com", "thebalibible.com", "villa-bali.com"}},
}

var genericIndustry = industry{key: "umum", seed: "jasa", competitors: []string{"kompas.com", "detik.com", "tokopedia.com", "shopee.co.id", "olx.co.id", "medium.com"}}

var (
	infoPrefixRe    = regexp.MustCompile(`^(cara|apa itu|bagaimana|kenapa|apakah|kapan|manfaat|beda|resep|tips)`)
	infoWordRe      = regexp.MustCompile(`\b(cara|manfaat|resep|tips|arti)\b`)
	transWordRe     = regexp.MustCompile(`\b(beli|jual|harga|promo|murah|diskon|order|pesan|grosir|paket)\b`)
	commercialRe    = regexp.MustCompile(`\b(terbaik|review|rekomendasi|vs|terdekat|top|perbandingan)\b`)
	localRe         = regexp.MustCompile(`terdekat|jakarta|bandung|surabaya|bali`)
	spammyRe        = regexp.MustCompile(`xyz|info|top|site$`)
	nonAlnumRe      = regexp.MustCompile(`[^a-z0-9]+`)
	nonAlnumStripRe = regexp.MustCompile(`[^a-z0-9]`)
)

func industryFor(dom string) industry {
	d := strings.ToLower(dom)
	for _, i := range industries {
		if i.match.MatchString(d) {
			return i
		}
	}
	return genericIndustry
}

func brandFromDomain(dom string) string {
	return strings.SplitN(strings.TrimPrefix(dom, "www."), ".", 2)[0]
}

// jsRound mirrors Math.round (half up).
func jsRound(x float64) float64 { return math.Floor(x + 0.5) }

func unit(key string) float64 { return float64(mockrand.Seed(key)) / 4294967296 }

func slugOf(keyword string) string {
	return nonAlnumRe.ReplaceAllString(strings.ToLower(keyword), "-")
}

// volumeFor: log-normal volume, heavy tail, most keywords small.
func volumeFor(keyword string, base float64) int32 {
	r := mockrand.New("vol:" + keyword)
	u := 1 - r.Float64()
	v := r.Float64()
	z := math.Sqrt(-2*math.Log(u)) * math.Cos(2*math.Pi*v)
	raw := math.Exp(math.Log(base) + z*0.9)
	var rounded float64
	switch {
	case raw < 100:
		rounded = jsRound(raw/10) * 10
	case raw < 1000:
		rounded = jsRound(raw/50) * 50
	default:
		rounded = jsRound(raw/100) * 100
	}
	return int32(math.Max(10, rounded))
}

func intentFor(keyword string) domain.Intent {
	k := strings.ToLower(keyword)
	switch {
	case infoPrefixRe.MatchString(k) || infoWordRe.MatchString(k):
		return domain.IntentInformational
	case transWordRe.MatchString(k):
		return domain.IntentTransactional
	case commercialRe.MatchString(k):
		return domain.IntentCommercial
	case unit("intent:"+k) < 0.15:
		return domain.IntentNavigational
	}
	return domain.IntentInformational
}

func ideaFor(keyword string, seedBase float64) domain.KeywordIdea {
	volume := volumeFor(keyword, seedBase)
	r := mockrand.New("idea:" + keyword)
	kd := jsRound(math.Min(100, math.Max(2, 12*math.Log10(float64(volume))+(r.Float64()-0.5)*30)))
	intent := intentFor(keyword)
	cpcBase := 1500.0
	switch intent {
	case domain.IntentTransactional:
		cpcBase = 6000
	case domain.IntentCommercial:
		cpcBase = 4000
	case domain.IntentNavigational:
		cpcBase = 1200
	}
	cpc := jsRound(math.Min(15000, math.Max(500, cpcBase*(0.5+r.Float64()*1.4)+kd*40))/50) * 50
	compBase := 0.25
	if intent == domain.IntentTransactional {
		compBase = 0.6
	}
	competition := jsRound(math.Min(1, math.Max(0.02, compBase+(r.Float64()-0.5)*0.5))*100) / 100
	slope := (r.Float64() - 0.45) * 0.04
	phase := r.Float64() * math.Pi * 2
	trend := make([]int32, 12)
	for i := 0; i < 12; i++ {
		v := float64(volume) * (1 + slope*float64(i-11) + 0.12*math.Sin(phase+float64(i)/12*math.Pi*2) + (r.Float64()-0.5)*0.1)
		trend[i] = int32(math.Max(10, jsRound(v)))
	}
	trend[11] = volume
	features := []string{}
	for i, f := range serpFeatures {
		threshold := 0.2
		if i < 2 {
			threshold = 0.45
		}
		if r.Float64() < threshold {
			features = append(features, f)
		}
	}
	if intent == domain.IntentTransactional && !contains(features, "shopping") && r.Float64() < 0.6 {
		features = append(features, "shopping")
	}
	if localRe.MatchString(keyword) && !contains(features, "local_pack") {
		features = append(features, "local_pack")
	}
	return domain.KeywordIdea{Keyword: keyword, Volume: volume, Difficulty: int32(kd), CPC: cpc, Competition: competition, Intent: intent, Trend: trend, SerpFeatures: features}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// orderedSet keeps insertion order like a JS Set.
type orderedSet struct {
	seen map[string]bool
	list []string
}

func (s *orderedSet) add(v string) {
	if s.seen == nil {
		s.seen = map[string]bool{}
	}
	if !s.seen[v] {
		s.seen[v] = true
		s.list = append(s.list, v)
	}
}

func expandSeed(seed string, mode domain.ResearchMode, limit int) []string {
	s := domain.NormalizeKeyword(seed)
	r := mockrand.New(fmt.Sprintf("expand:%s:%s", mode, s))
	var out orderedSet
	switch mode {
	case domain.ModeIdeas:
		out.add(s)
		for _, m := range modifiers {
			out.add(s + " " + m)
		}
		for _, m := range []string{"harga", "review", "grosir", "rekomendasi"} {
			if r.Float64() < 0.7 {
				out.add(m + " " + s)
			}
		}
		for _, c := range cities {
			out.add(s + " " + c)
		}
		for _, x := range relatedSuffix {
			out.add(s + " " + x)
		}
		for _, m := range modifiers[:5] {
			for _, c := range cities[:4] {
				out.add(s + " " + m + " " + c)
			}
		}
	case domain.ModeQuestions:
		for _, q := range questionPrefix {
			out.add(q + " " + s)
		}
		for _, q := range questionPrefix[:5] {
			for _, x := range relatedSuffix[:5] {
				out.add(q + " " + s + " " + x)
			}
		}
		for _, v := range []string{s + " itu apa", s + " bagus tidak", s + " untuk apa", "kelebihan " + s, "kekurangan " + s, "manfaat " + s} {
			out.add(v)
		}
	default:
		head := strings.SplitN(s, " ", 2)[0]
		for _, x := range relatedSuffix {
			out.add(head + " " + x)
		}
		for _, m := range modifiers {
			out.add(head + " " + m)
		}
		for _, v := range []string{"jenis " + s, s + " vs", "alternatif " + s, s + " terpopuler", "supplier " + s, s + " kemasan"} {
			out.add(v)
		}
		for _, c := range cities[:6] {
			out.add(head + " " + c)
		}
	}
	arr := out.list
	for i := len(arr) - 1; i > 0; i-- {
		j := int(math.Floor(r.Float64() * float64(i+1)))
		arr[i], arr[j] = arr[j], arr[i]
	}
	if len(arr) > limit {
		arr = arr[:limit]
	}
	return arr
}

// RankKey makes desktop and mobile walk independently for the same target.
func RankKey(targetDomain string, device domain.Device) string {
	if device == domain.DeviceDesktop {
		return targetDomain + "#d"
	}
	return targetDomain
}

func isBrandKeyword(keyword, targetDomain string) bool {
	brand := strings.ToLower(brandFromDomain(strings.TrimSuffix(targetDomain, "#d")))
	return len(brand) >= 4 && strings.Contains(nonAlnumStripRe.ReplaceAllString(strings.ToLower(keyword), ""), brand)
}

func dayIndex(day time.Time) int64 { return day.Unix() / 86400 }

// MockPositionOn is stable per (keyword, target, day). Brand keywords rank
// 1-3; other keywords are climbers, sliders or drop-outs with sine drift and
// daily jitter. nil above 100.
func MockPositionOn(keyword, target string, day time.Time) *int32 {
	key := target + "|" + strings.ToLower(keyword)
	r := mockrand.New("rank:" + key)
	idx := float64(dayIndex(day))
	jitter := (unit("j:"+key+":"+dates.Format(day)) - 0.5) * 2.2
	pos := func(v float64) *int32 {
		p := int32(math.Max(1, jsRound(v)))
		return &p
	}
	if isBrandKeyword(keyword, target) {
		return pos(1.2 + r.Float64()*1.5 + 0.6*math.Sin(idx/11) + jitter*0.4)
	}
	arche := r.Float64()
	base := 2 + math.Pow(r.Float64(), 1.6)*60
	amp := 1 + r.Float64()*8
	period := 20 + r.Float64()*60
	phase := r.Float64() * math.Pi * 2
	slope := (r.Float64() - 0.5) * 0.08
	switch {
	case arche < 0.08:
		slope = 0.5
	case arche < 0.22:
		slope = -0.12
	case arche < 0.34:
		slope = 0.12
	}
	p := base + amp*math.Sin(phase+(idx/period)*math.Pi*2) + slope*math.Mod(idx, 365) + jitter
	if p > 100 {
		return nil
	}
	if arche >= 0.08 && arche < 0.12 && math.Sin(phase+idx/15) > 0.35 {
		return nil
	}
	return pos(p)
}

// MockTop10 fills the top 10 with industry competitors and generic domains,
// placing the target at its position when it ranks there.
func MockTop10(keyword, targetDomain string, position *int32, day time.Time) []domain.TopResult {
	ind := industryFor(targetDomain)
	pool := append(append([]string{}, ind.competitors...), genericDomains...)
	r := mockrand.New("top:" + keyword + ":" + dates.Format(day)[:7])
	picked := []string{}
	for len(picked) < 10 {
		d := pool[int(math.Floor(r.Float64()*float64(len(pool))))]
		if !contains(picked, d) && d != targetDomain {
			picked = append(picked, d)
		}
	}
	slug := slugOf(keyword)
	out := make([]domain.TopResult, 0, 10)
	k := 0
	for p := int32(1); p <= 10; p++ {
		if position != nil && *position == p {
			out = append(out, domain.TopResult{Domain: targetDomain, Position: p, URL: "https://" + targetDomain + "/" + slug})
			continue
		}
		d := picked[k]
		k++
		out = append(out, domain.TopResult{Domain: d, Position: p, URL: "https://" + d + "/" + slug})
	}
	return out
}

// MockBacklinkList has 70-120 links: a few spam domains (spam score >= 60),
// 14% lost, 68% dofollow.
func (m *Mock) MockBacklinkList(dom string) []domain.BacklinkItem {
	r := mockrand.New("bl:" + dom)
	brand := brandFromDomain(dom)
	today := dates.Day(m.clk.Now())
	n := 70 + int(math.Floor(r.Float64()*50))
	paths := []string{"/", "/blog/", "/promo/", "/tentang", "/produk/"}
	kinds := []string{"artikel", "berita", "post", "review", "p"}
	out := make([]domain.BacklinkItem, 0, n)
	for i := 0; i < n; i++ {
		src := backlinkDomains[int(math.Floor(r.Float64()*float64(len(backlinkDomains))))]
		spammy := spammyRe.MatchString(src)
		var spamScore, domainRank int32
		if spammy {
			spamScore = 60 + int32(math.Floor(r.Float64()*40))
		} else {
			spamScore = int32(math.Floor(r.Float64() * 35))
		}
		if spammy {
			domainRank = 2 + int32(math.Floor(r.Float64()*12))
		} else {
			domainRank = 20 + int32(math.Floor(r.Float64()*75))
		}
		firstAge := int(math.Floor(math.Pow(r.Float64(), 1.4) * 400))
		firstSeen := dates.AddDays(today, -firstAge)
		isLost := r.Float64() < 0.14 && firstAge > 20
		var lastSeen time.Time
		if isLost {
			lastSeen = dates.AddDays(firstSeen, int(math.Max(3, math.Floor(r.Float64()*float64(firstAge)))))
		} else {
			lastSeen = dates.AddDays(today, -int(math.Floor(r.Float64()*3)))
		}
		anchor := strings.ReplaceAll(anchors[int(math.Floor(r.Float64()*float64(len(anchors))))], "{brand}", brand)
		path := paths[int(math.Floor(r.Float64()*5))]
		kind := kinds[int(math.Floor(r.Float64()*5))]
		num := int(math.Floor(r.Float64()*90000 + 1000))
		dofollow := r.Float64() < 0.68
		out = append(out, domain.BacklinkItem{
			SourceURL:    fmt.Sprintf("https://%s/%s/%s-%d", src, kind, brand, num),
			SourceDomain: src,
			TargetURL:    "https://" + dom + path,
			Anchor:       anchor,
			Dofollow:     dofollow,
			DomainRank:   domainRank,
			SpamScore:    spamScore,
			FirstSeen:    dates.Format(firstSeen),
			LastSeen:     dates.Format(lastSeen),
			IsLost:       isLost,
		})
	}
	return out
}

func sortByVolume(ideas []domain.KeywordIdea) {
	sort.SliceStable(ideas, func(i, j int) bool { return ideas[i].Volume > ideas[j].Volume })
}

func (m *Mock) KeywordIdeas(_ context.Context, seed string, _ int32, _ string, mode domain.ResearchMode, limit int) ([]domain.KeywordIdea, error) {
	if limit <= 0 {
		limit = 60
	}
	base := 300 + unit("base:"+strings.ToLower(seed))*3000
	keys := expandSeed(seed, mode, limit)
	out := make([]domain.KeywordIdea, 0, len(keys))
	for _, k := range keys {
		out = append(out, ideaFor(k, base))
	}
	sortByVolume(out)
	return out, nil
}

func (m *Mock) KeywordMetrics(_ context.Context, keywords []string, _ int32, _ string) ([]domain.KeywordIdea, error) {
	out := make([]domain.KeywordIdea, 0, len(keywords))
	for _, k := range keywords {
		head := strings.SplitN(k, " ", 2)[0]
		out = append(out, ideaFor(strings.ToLower(strings.TrimSpace(k)), 200+unit("base:"+head)*1500))
	}
	return out, nil
}

func (m *Mock) SerpRanks(_ context.Context, items []domain.SerpRankRequest, _ int32, _ string, targetDomain string) ([]domain.RankResult, error) {
	day := dates.Day(m.clk.Now())
	out := make([]domain.RankResult, 0, len(items))
	for _, it := range items {
		position := MockPositionOn(it.Keyword, RankKey(targetDomain, it.Device), day)
		var url *string
		if position != nil {
			u := "https://" + targetDomain + "/" + slugOf(it.Keyword)
			url = &u
		}
		out = append(out, domain.RankResult{
			Keyword: it.Keyword, Device: it.Device, Position: position, URL: url,
			SerpFeatures: ideaFor(it.Keyword, 500).SerpFeatures,
			Top10:        MockTop10(it.Keyword, targetDomain, position, day),
		})
	}
	return out, nil
}

func (m *Mock) BacklinkSummary(_ context.Context, dom string) (domain.BacklinkSummary, error) {
	list := m.MockBacklinkList(dom)
	today := dates.Day(m.clk.Now())
	var live, dofollow, toxic, newLast30, lostLast30 int32
	domains := map[string]bool{}
	for _, b := range list {
		first, _ := dates.Parse(b.FirstSeen)
		last, _ := dates.Parse(b.LastSeen)
		if today.Sub(first) <= 30*24*time.Hour {
			newLast30++
		}
		if b.IsLost && today.Sub(last) <= 30*24*time.Hour {
			lostLast30++
		}
		if b.IsLost {
			continue
		}
		live++
		domains[b.SourceDomain] = true
		if b.Dofollow {
			dofollow++
		}
		if b.SpamScore >= 60 {
			toxic++
		}
	}
	r := mockrand.New("dr:" + dom)
	s := domain.BacklinkSummary{
		Backlinks:        live * (12 + int32(math.Floor(r.Float64()*30))),
		ReferringDomains: int32(len(domains)) * (3 + int32(math.Floor(r.Float64()*6))),
		Dofollow:         dofollow * 14,
		Nofollow:         (live - dofollow) * 14,
		DomainRank:       18 + int32(math.Floor(r.Float64()*40)),
		NewLast30:        newLast30,
		LostLast30:       lostLast30,
	}
	if live > 0 {
		s.ToxicShare = float64(toxic) / float64(live)
	}
	return s, nil
}

func (m *Mock) Backlinks(_ context.Context, dom string, limit int, mode domain.BacklinkMode) ([]domain.BacklinkItem, error) {
	if limit <= 0 {
		limit = 200
	}
	today := dates.Day(m.clk.Now())
	list := m.MockBacklinkList(dom)
	out := make([]domain.BacklinkItem, 0, len(list))
	for _, b := range list {
		switch mode {
		case domain.BacklinksNew:
			first, _ := dates.Parse(b.FirstSeen)
			if b.IsLost || today.Sub(first) > 30*24*time.Hour {
				continue
			}
		case domain.BacklinksLost:
			if !b.IsLost {
				continue
			}
		}
		out = append(out, b)
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *Mock) DomainOverview(ctx context.Context, dom string, _ int32, _ string) (domain.DomainOverview, error) {
	r := mockrand.New("dom:" + dom)
	scale := 0.4 + math.Pow(r.Float64(), 1.3)*6
	drift := 1 + (float64(dayIndex(dates.Day(m.clk.Now()))%90)/90)*(r.Float64()-0.4)*0.2
	organicKeywords := jsRound(800 * scale * drift)
	top100 := jsRound(organicKeywords * 0.92)
	top10 := jsRound(top100 * (0.08 + r.Float64()*0.1))
	top3 := jsRound(top10 * (0.25 + r.Float64()*0.2))
	organicTraffic := jsRound(organicKeywords * (4 + r.Float64()*10) * drift)
	summary, _ := m.BacklinkSummary(ctx, dom)
	return domain.DomainOverview{
		Domain:           dom,
		OrganicKeywords:  int32(organicKeywords),
		OrganicTraffic:   int32(organicTraffic),
		OrganicCost:      jsRound(organicTraffic * (900 + r.Float64()*2500)),
		Top3:             int32(top3),
		Top10:            int32(top10),
		Top100:           int32(top100),
		Backlinks:        summary.Backlinks,
		ReferringDomains: summary.ReferringDomains,
		DomainRank:       summary.DomainRank,
	}, nil
}

func (m *Mock) DomainKeywords(ctx context.Context, dom string, _ int32, _ string, limit int) ([]domain.DomainKeyword, error) {
	if limit <= 0 {
		limit = 100
	}
	ind := industryFor(dom)
	var universe []domain.KeywordIdea
	for _, part := range []struct {
		mode  domain.ResearchMode
		limit int
	}{{domain.ModeIdeas, 60}, {domain.ModeQuestions, 25}, {domain.ModeRelated, 25}} {
		ideas, _ := m.KeywordIdeas(ctx, ind.seed, 2360, "id", part.mode, part.limit)
		universe = append(universe, ideas...)
	}
	out := []domain.DomainKeyword{}
	for _, idea := range universe {
		if unit("dk:"+dom+":"+idea.Keyword) > 0.58 {
			continue
		}
		pos := int32(math.Max(1, jsRound(1+math.Pow(unit("dp:"+dom+":"+idea.Keyword), 1.5)*60)))
		out = append(out, domain.DomainKeyword{Keyword: idea.Keyword, Position: pos, Volume: idea.Volume, URL: "https://" + dom + "/" + nonAlnumRe.ReplaceAllString(idea.Keyword, "-")})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Volume > out[j].Volume })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *Mock) Competitors(_ context.Context, dom string, _ int32, _ string) ([]domain.CompetitorSuggestion, error) {
	ind := industryFor(dom)
	out := []domain.CompetitorSuggestion{}
	for _, d := range ind.competitors {
		if d == dom {
			continue
		}
		r := mockrand.New("comp:" + dom + ":" + d)
		out = append(out, domain.CompetitorSuggestion{Domain: d, Intersections: 40 + int32(math.Floor(r.Float64()*400)), OrganicTraffic: int32(jsRound(2000 + r.Float64()*90000))})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Intersections > out[j].Intersections })
	return out, nil
}
