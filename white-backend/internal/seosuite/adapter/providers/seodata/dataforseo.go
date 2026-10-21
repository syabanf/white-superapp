package seodata

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/httpx"
)

const (
	DataForSEOBase     = "https://api.dataforseo.com"
	dataForSEOTimeout  = 60 * time.Second
	providerDataForSEO = "dataforseo"
)

// DataForSEO is the live REST v3 client (Basic auth, every call a POST with
// one task, only tasks[0].result is read).
type DataForSEO struct {
	http     *httpx.Client
	base     string
	auth     string
	usdToIDR float64
	clk      clock.Clock
}

func NewDataForSEO(login, password string, opts Options) *DataForSEO {
	opts = opts.withDefaults()
	return &DataForSEO{
		http:     httpx.New(dataForSEOTimeout),
		base:     strings.TrimRight(opts.DataForSEOBase, "/"),
		auth:     "Basic " + base64.StdEncoding.EncodeToString([]byte(login+":"+password)),
		usdToIDR: opts.USDToIDR,
		clk:      opts.Clock,
	}
}

type dfsEnvelope struct {
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
	Tasks         []struct {
		StatusCode    int               `json:"status_code"`
		StatusMessage string            `json:"status_message"`
		Result        []json.RawMessage `json:"result"`
	} `json:"tasks"`
}

func dfsCodeForStatus(status int) apperr.Code {
	switch status {
	case 401:
		return apperr.TokenExpired
	case 402, 403:
		return apperr.Permission
	case 429:
		return apperr.RateLimit
	case 404:
		return apperr.NotFound
	}
	return apperr.Unknown
}

func dfsMessage(code apperr.Code, api string) string {
	switch code {
	case apperr.TokenExpired:
		return fmt.Sprintf("Kredensial DataForSEO ditolak saat memanggil %s. Periksa DATAFORSEO_LOGIN/PASSWORD.", api)
	case apperr.Permission:
		return fmt.Sprintf("Akun DataForSEO tidak punya akses/saldo untuk %s.", api)
	case apperr.RateLimit:
		return fmt.Sprintf("Batas permintaan DataForSEO tercapai (%s). Coba lagi beberapa menit.", api)
	case apperr.NotFound:
		return fmt.Sprintf("Endpoint DataForSEO tidak ditemukan (%s).", api)
	case apperr.Network:
		return fmt.Sprintf("Gangguan jaringan saat memanggil DataForSEO (%s).", api)
	}
	return fmt.Sprintf("DataForSEO mengembalikan kesalahan saat memanggil %s.", api)
}

func dfsError(code apperr.Code, path, detail string, cause error) error {
	msg := dfsMessage(code, path)
	if detail != "" {
		msg = fmt.Sprintf("%s (%s)", msg, detail)
	}
	return apperr.FromProvider(providerDataForSEO, code, msg, cause)
}

// post sends one task array and returns tasks[0].result.
func (d *DataForSEO) post(ctx context.Context, path string, payload []any) ([]json.RawMessage, error) {
	res, err := d.http.PostJSON(ctx, d.base+path, payload, map[string]string{"Authorization": d.auth})
	if err != nil {
		return nil, dfsError(apperr.Network, path, "", err)
	}
	if res.Status < 200 || res.Status >= 300 {
		return nil, dfsError(dfsCodeForStatus(res.Status), path, "", nil)
	}
	var env dfsEnvelope
	if err := res.JSON(&env); err != nil {
		return nil, dfsError(apperr.Unknown, path, "", err)
	}
	if env.StatusCode != 20000 {
		code := apperr.Unknown
		switch env.StatusCode {
		case 40200, 40201:
			code = apperr.Permission
		case 40202:
			code = apperr.RateLimit
		}
		return nil, dfsError(code, path, env.StatusMessage, nil)
	}
	if len(env.Tasks) == 0 {
		return []json.RawMessage{}, nil
	}
	task := env.Tasks[0]
	if task.StatusCode != 20000 {
		code := apperr.Unknown
		switch {
		case task.StatusCode == 40402:
			code = apperr.NotFound
		case task.StatusCode >= 40200 && task.StatusCode < 40300:
			code = apperr.Permission
		}
		return nil, dfsError(code, path, task.StatusMessage, nil)
	}
	return task.Result, nil
}

// postInto decodes every result item into T.
func postInto[T any](ctx context.Context, d *DataForSEO, path string, payload []any) ([]T, error) {
	raw, err := d.post(ctx, path, payload)
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(raw))
	for _, r := range raw {
		var v T
		if err := json.Unmarshal(r, &v); err != nil {
			return nil, dfsError(apperr.Unknown, path, "", err)
		}
		out = append(out, v)
	}
	return out, nil
}

// ─── normalisers ─────────────────────────────────────────────

type dfsMonthly struct {
	Year         int   `json:"year"`
	Month        int   `json:"month"`
	SearchVolume int32 `json:"search_volume"`
}

type dfsKeywordItem struct {
	Keyword     string `json:"keyword"`
	KeywordInfo *struct {
		SearchVolume    *int32       `json:"search_volume"`
		CPC             *float64     `json:"cpc"`
		Competition     *float64     `json:"competition"`
		MonthlySearches []dfsMonthly `json:"monthly_searches"`
	} `json:"keyword_info"`
	KeywordProperties *struct {
		KeywordDifficulty *float64 `json:"keyword_difficulty"`
	} `json:"keyword_properties"`
	SearchIntentInfo *struct {
		MainIntent *string `json:"main_intent"`
	} `json:"search_intent_info"`
	SerpInfo *struct {
		SerpItemTypes []string `json:"serp_item_types"`
	} `json:"serp_info"`
}

type dfsItems struct {
	Items []dfsKeywordItem `json:"items"`
}

func intentFrom(v *string) domain.Intent {
	if v == nil {
		return domain.IntentInformational
	}
	switch strings.ToLower(*v) {
	case "navigational":
		return domain.IntentNavigational
	case "commercial":
		return domain.IntentCommercial
	case "transactional":
		return domain.IntentTransactional
	}
	return domain.IntentInformational
}

func (d *DataForSEO) ideaFrom(item dfsKeywordItem) domain.KeywordIdea {
	idea := domain.KeywordIdea{Keyword: item.Keyword, Intent: domain.IntentInformational, SerpFeatures: []string{}}
	var monthly []dfsMonthly
	if item.KeywordInfo != nil {
		if item.KeywordInfo.SearchVolume != nil {
			idea.Volume = *item.KeywordInfo.SearchVolume
		}
		if item.KeywordInfo.CPC != nil {
			idea.CPC = math.Round(*item.KeywordInfo.CPC * d.usdToIDR)
		}
		if item.KeywordInfo.Competition != nil {
			idea.Competition = *item.KeywordInfo.Competition
		}
		monthly = append(monthly, item.KeywordInfo.MonthlySearches...)
	}
	if item.KeywordProperties != nil && item.KeywordProperties.KeywordDifficulty != nil {
		idea.Difficulty = int32(math.Round(*item.KeywordProperties.KeywordDifficulty))
	}
	if item.SearchIntentInfo != nil {
		idea.Intent = intentFrom(item.SearchIntentInfo.MainIntent)
	}
	if item.SerpInfo != nil && item.SerpInfo.SerpItemTypes != nil {
		idea.SerpFeatures = item.SerpInfo.SerpItemTypes
	}
	sort.SliceStable(monthly, func(i, j int) bool {
		if monthly[i].Year != monthly[j].Year {
			return monthly[i].Year < monthly[j].Year
		}
		return monthly[i].Month < monthly[j].Month
	})
	if len(monthly) > 12 {
		monthly = monthly[len(monthly)-12:]
	}
	trend := make([]int32, 0, 12)
	for _, m := range monthly {
		trend = append(trend, m.SearchVolume)
	}
	if len(trend) == 0 {
		for i := 0; i < 12; i++ {
			trend = append(trend, idea.Volume)
		}
	}
	for len(trend) < 12 {
		trend = append([]int32{trend[0]}, trend...)
	}
	idea.Trend = trend
	return idea
}

var wwwRe = regexp.MustCompile(`^www\.`)

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return raw
	}
	return wwwRe.ReplaceAllString(u.Hostname(), "")
}

func sameDomain(host, target string) bool {
	h := wwwRe.ReplaceAllString(host, "")
	return h == target || strings.HasSuffix(h, "."+target)
}

var questionRe = regexp.MustCompile(`(?i)^(cara|apa|bagaimana|kenapa|mengapa|berapa|apakah|dimana|di mana|kapan|siapa|how|what|why|where|when|who)\b`)

// ─── provider ────────────────────────────────────────────────

func (d *DataForSEO) KeywordIdeas(ctx context.Context, seed string, locationCode int32, languageCode string, mode domain.ResearchMode, limit int) ([]domain.KeywordIdea, error) {
	if limit <= 0 {
		limit = 100
	}
	common := map[string]any{"location_code": locationCode, "language_code": languageCode, "limit": limit, "include_serp_info": true}
	task := func(extra map[string]any) []any {
		m := map[string]any{}
		for k, v := range common {
			m[k] = v
		}
		for k, v := range extra {
			m[k] = v
		}
		return []any{m}
	}
	var items []dfsKeywordItem
	switch mode {
	case domain.ModeQuestions:
		res, err := postInto[dfsItems](ctx, d, "/v3/dataforseo_labs/google/keyword_suggestions/live", task(map[string]any{"keyword": seed, "filters": []any{"keyword_info.search_volume", ">", 0}}))
		if err != nil {
			return nil, err
		}
		for _, r := range res {
			items = append(items, r.Items...)
		}
	case domain.ModeRelated:
		type rel struct {
			Items []struct {
				KeywordData *dfsKeywordItem `json:"keyword_data"`
			} `json:"items"`
		}
		res, err := postInto[rel](ctx, d, "/v3/dataforseo_labs/google/related_keywords/live", task(map[string]any{"keyword": seed, "depth": 2}))
		if err != nil {
			return nil, err
		}
		for _, r := range res {
			for _, i := range r.Items {
				if i.KeywordData != nil {
					items = append(items, *i.KeywordData)
				}
			}
		}
	default:
		res, err := postInto[dfsItems](ctx, d, "/v3/dataforseo_labs/google/keyword_ideas/live", task(map[string]any{"keywords": []string{seed}, "include_clickstream_data": false}))
		if err != nil {
			return nil, err
		}
		for _, r := range res {
			items = append(items, r.Items...)
		}
	}
	ideas := make([]domain.KeywordIdea, 0, len(items))
	for _, it := range items {
		idea := d.ideaFrom(it)
		if idea.Keyword == "" {
			continue
		}
		if mode == domain.ModeQuestions && !questionRe.MatchString(idea.Keyword) {
			continue
		}
		ideas = append(ideas, idea)
	}
	sortByVolume(ideas)
	if len(ideas) > limit {
		ideas = ideas[:limit]
	}
	return ideas, nil
}

func (d *DataForSEO) KeywordMetrics(ctx context.Context, keywords []string, locationCode int32, languageCode string) ([]domain.KeywordIdea, error) {
	if len(keywords) == 0 {
		return []domain.KeywordIdea{}, nil
	}
	volumes, err := postInto[dfsItems](ctx, d, "/v3/dataforseo_labs/google/historical_search_volume/live", []any{map[string]any{"keywords": keywords, "location_code": locationCode, "language_code": languageCode, "include_serp_info": true}})
	if err != nil {
		// Fall back to Google Ads volumes (no intent or SERP info).
		type adsRow struct {
			Keyword          string       `json:"keyword"`
			SearchVolume     *int32       `json:"search_volume"`
			CPC              *float64     `json:"cpc"`
			CompetitionIndex *float64     `json:"competition_index"`
			MonthlySearches  []dfsMonthly `json:"monthly_searches"`
		}
		rows, err := postInto[adsRow](ctx, d, "/v3/keywords_data/google_ads/search_volume/live", []any{map[string]any{"keywords": keywords, "location_code": locationCode, "language_code": languageCode}})
		if err != nil {
			return nil, err
		}
		var fb dfsItems
		for _, r := range rows {
			comp := 0.0
			if r.CompetitionIndex != nil {
				comp = *r.CompetitionIndex / 100
			}
			item := dfsKeywordItem{Keyword: r.Keyword}
			item.KeywordInfo = &struct {
				SearchVolume    *int32       `json:"search_volume"`
				CPC             *float64     `json:"cpc"`
				Competition     *float64     `json:"competition"`
				MonthlySearches []dfsMonthly `json:"monthly_searches"`
			}{SearchVolume: r.SearchVolume, CPC: r.CPC, Competition: &comp, MonthlySearches: r.MonthlySearches}
			fb.Items = append(fb.Items, item)
		}
		volumes = []dfsItems{fb}
	}
	type kdItems struct {
		Items []struct {
			Keyword           string   `json:"keyword"`
			KeywordDifficulty *float64 `json:"keyword_difficulty"`
		} `json:"items"`
	}
	kd := map[string]int32{}
	if difficulty, err := postInto[kdItems](ctx, d, "/v3/dataforseo_labs/google/bulk_keyword_difficulty/live", []any{map[string]any{"keywords": keywords, "location_code": locationCode, "language_code": languageCode}}); err == nil {
		for _, r := range difficulty {
			for _, i := range r.Items {
				v := 0.0
				if i.KeywordDifficulty != nil {
					v = *i.KeywordDifficulty
				}
				kd[strings.ToLower(i.Keyword)] = int32(math.Round(v))
			}
		}
	}
	byKeyword := map[string]domain.KeywordIdea{}
	for _, r := range volumes {
		for _, i := range r.Items {
			idea := d.ideaFrom(i)
			key := strings.ToLower(idea.Keyword)
			if v, ok := kd[key]; ok {
				idea.Difficulty = v
			}
			byKeyword[key] = idea
		}
	}
	out := make([]domain.KeywordIdea, 0, len(keywords))
	for _, k := range keywords {
		key := strings.ToLower(k)
		if idea, ok := byKeyword[key]; ok {
			out = append(out, idea)
			continue
		}
		out = append(out, domain.KeywordIdea{Keyword: k, Difficulty: kd[key], Intent: domain.IntentInformational, Trend: make([]int32, 12), SerpFeatures: []string{}})
	}
	return out, nil
}

type dfsSerpResult struct {
	ItemTypes []string `json:"item_types"`
	Items     []struct {
		Type         string `json:"type"`
		RankAbsolute *int32 `json:"rank_absolute"`
		RankGroup    *int32 `json:"rank_group"`
		Domain       string `json:"domain"`
		URL          string `json:"url"`
	} `json:"items"`
}

func (d *DataForSEO) SerpRanks(ctx context.Context, items []domain.SerpRankRequest, locationCode int32, languageCode, targetDomain string) ([]domain.RankResult, error) {
	results := make([]domain.RankResult, len(items))
	errs := make([]error, len(items))
	// live/regular accepts one task per call; run in parallel batches of 5.
	for i := 0; i < len(items); i += 5 {
		end := min(i+5, len(items))
		var wg sync.WaitGroup
		for j := i; j < end; j++ {
			wg.Add(1)
			go func(j int) {
				defer wg.Done()
				src := items[j]
				task := []any{map[string]any{"keyword": src.Keyword, "location_code": locationCode, "language_code": languageCode, "device": strings.ToLower(string(src.Device)), "depth": 100}}
				res, err := postInto[dfsSerpResult](ctx, d, "/v3/serp/google/organic/live/regular", task)
				if err != nil {
					errs[j] = err
					return
				}
				results[j] = d.rankResult(src, res, targetDomain)
			}(j)
		}
		wg.Wait()
		for j := i; j < end; j++ {
			if errs[j] != nil {
				return nil, errs[j]
			}
		}
	}
	return results, nil
}

func (d *DataForSEO) rankResult(src domain.SerpRankRequest, res []dfsSerpResult, targetDomain string) domain.RankResult {
	out := domain.RankResult{Keyword: src.Keyword, Device: src.Device, SerpFeatures: []string{}, Top10: []domain.TopResult{}}
	if len(res) == 0 {
		return out
	}
	first := res[0]
	for _, t := range first.ItemTypes {
		if t != "organic" {
			out.SerpFeatures = append(out.SerpFeatures, t)
		}
	}
	k := 0
	for _, x := range first.Items {
		if x.Type != "organic" {
			continue
		}
		host := x.Domain
		if host == "" {
			host = hostOf(x.URL)
		}
		if out.Position == nil && sameDomain(host, targetDomain) {
			pos := x.RankGroup
			if pos == nil {
				pos = x.RankAbsolute
			}
			if pos != nil {
				p := *pos
				out.Position = &p
				u := x.URL
				out.URL = &u
			}
		}
		if k < 10 {
			position := int32(k + 1)
			if x.RankGroup != nil {
				position = *x.RankGroup
			}
			out.Top10 = append(out.Top10, domain.TopResult{Domain: wwwRe.ReplaceAllString(host, ""), Position: position, URL: x.URL})
		}
		k++
	}
	return out
}

type dfsBacklinkSummary struct {
	Backlinks               *int32           `json:"backlinks"`
	ReferringDomains        *int32           `json:"referring_domains"`
	Rank                    *float64         `json:"rank"`
	ReferringLinksAttribute map[string]int32 `json:"referring_links_attributes"`
	BacklinksSpamScore      *float64         `json:"backlinks_spam_score"`
}

func (d *DataForSEO) BacklinkSummary(ctx context.Context, dom string) (domain.BacklinkSummary, error) {
	sums, err := postInto[dfsBacklinkSummary](ctx, d, "/v3/backlinks/summary/live", []any{map[string]any{"target": dom, "include_subdomains": true, "backlinks_status_type": "live"}})
	if err != nil {
		return domain.BacklinkSummary{}, err
	}
	type hist struct {
		Items []struct {
			NewBacklinks  int32 `json:"new_backlinks"`
			LostBacklinks int32 `json:"lost_backlinks"`
		} `json:"items"`
	}
	dateFrom := dates.Format(dates.AddDays(dates.Day(d.clk.Now()), -30))
	histories, err := postInto[hist](ctx, d, "/v3/backlinks/timeseries_new_lost_summary/live", []any{map[string]any{"target": dom, "date_from": dateFrom, "group_range": "month"}})
	if err != nil {
		histories = nil
	}
	var s domain.BacklinkSummary
	if len(sums) > 0 {
		sum := sums[0]
		if sum.Backlinks != nil {
			s.Backlinks = *sum.Backlinks
		}
		if sum.ReferringDomains != nil {
			s.ReferringDomains = *sum.ReferringDomains
		}
		s.Nofollow = sum.ReferringLinksAttribute["nofollow"]
		s.Dofollow = int32(math.Max(0, float64(s.Backlinks-s.Nofollow)))
		if sum.Rank != nil {
			s.DomainRank = int32(math.Round(*sum.Rank / 10))
		}
		if sum.BacklinksSpamScore != nil {
			s.ToxicShare = math.Min(1, *sum.BacklinksSpamScore/100)
		}
	}
	if len(histories) > 0 {
		for _, i := range histories[0].Items {
			s.NewLast30 += i.NewBacklinks
			s.LostLast30 += i.LostBacklinks
		}
	}
	return s, nil
}

func (d *DataForSEO) Backlinks(ctx context.Context, dom string, limit int, mode domain.BacklinkMode) ([]domain.BacklinkItem, error) {
	if limit <= 0 {
		limit = 200
	}
	type row struct {
		URLFrom           string   `json:"url_from"`
		DomainFrom        string   `json:"domain_from"`
		URLTo             string   `json:"url_to"`
		Anchor            *string  `json:"anchor"`
		Dofollow          *bool    `json:"dofollow"`
		DomainFromRank    *float64 `json:"domain_from_rank"`
		BacklinkSpamScore *int32   `json:"backlink_spam_score"`
		FirstSeen         *string  `json:"first_seen"`
		LastSeen          *string  `json:"last_seen"`
		IsLost            *bool    `json:"is_lost"`
	}
	type res struct {
		Items []row `json:"items"`
	}
	body := map[string]any{"target": dom, "mode": "as_is", "limit": limit, "order_by": []string{"rank,desc"}, "backlinks_status_type": "live"}
	switch mode {
	case domain.BacklinksLost:
		body["backlinks_status_type"] = "all"
		body["filters"] = []any{"is_lost", "=", true}
	case domain.BacklinksNew:
		body["filters"] = []any{"is_new", "=", true}
	}
	rs, err := postInto[res](ctx, d, "/v3/backlinks/backlinks/live", []any{body})
	if err != nil {
		return nil, err
	}
	out := []domain.BacklinkItem{}
	if len(rs) == 0 {
		return out, nil
	}
	day := func(s *string) string {
		if s == nil {
			return ""
		}
		if len(*s) > 10 {
			return (*s)[:10]
		}
		return *s
	}
	for _, r := range rs[0].Items {
		item := domain.BacklinkItem{SourceURL: r.URLFrom, SourceDomain: r.DomainFrom, TargetURL: r.URLTo, Dofollow: true, FirstSeen: day(r.FirstSeen), LastSeen: day(r.LastSeen)}
		if r.Anchor != nil {
			item.Anchor = *r.Anchor
		}
		if r.Dofollow != nil {
			item.Dofollow = *r.Dofollow
		}
		if r.DomainFromRank != nil {
			item.DomainRank = int32(math.Round(*r.DomainFromRank / 10))
		}
		if r.BacklinkSpamScore != nil {
			item.SpamScore = *r.BacklinkSpamScore
		}
		if r.IsLost != nil {
			item.IsLost = *r.IsLost
		}
		out = append(out, item)
	}
	return out, nil
}

func (d *DataForSEO) DomainOverview(ctx context.Context, dom string, locationCode int32, languageCode string) (domain.DomainOverview, error) {
	type organic struct {
		Count                *int32   `json:"count"`
		ETV                  *float64 `json:"etv"`
		EstimatedPaidTraffic *float64 `json:"estimated_paid_traffic_cost"`
		Pos1                 *int32   `json:"pos_1"`
		Pos23                *int32   `json:"pos_2_3"`
		Pos410               *int32   `json:"pos_4_10"`
	}
	type res struct {
		Items []struct {
			Metrics *struct {
				Organic *organic `json:"organic"`
			} `json:"metrics"`
		} `json:"items"`
	}
	var (
		wg    sync.WaitGroup
		rs    []res
		rsErr error
		links *domain.BacklinkSummary
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		rs, rsErr = postInto[res](ctx, d, "/v3/dataforseo_labs/google/domain_rank_overview/live", []any{map[string]any{"target": dom, "location_code": locationCode, "language_code": languageCode}})
	}()
	go func() {
		defer wg.Done()
		if s, err := d.BacklinkSummary(ctx, dom); err == nil {
			links = &s
		}
	}()
	wg.Wait()
	if rsErr != nil {
		return domain.DomainOverview{}, rsErr
	}
	o := organic{}
	if len(rs) > 0 && len(rs[0].Items) > 0 && rs[0].Items[0].Metrics != nil && rs[0].Items[0].Metrics.Organic != nil {
		o = *rs[0].Items[0].Metrics.Organic
	}
	val := func(p *int32) int32 {
		if p == nil {
			return 0
		}
		return *p
	}
	fval := func(p *float64) float64 {
		if p == nil {
			return 0
		}
		return *p
	}
	top3 := val(o.Pos1) + val(o.Pos23)
	out := domain.DomainOverview{
		Domain:          dom,
		OrganicKeywords: val(o.Count),
		OrganicTraffic:  int32(math.Round(fval(o.ETV))),
		OrganicCost:     math.Round(fval(o.EstimatedPaidTraffic) * d.usdToIDR),
		Top3:            top3,
		Top10:           top3 + val(o.Pos410),
		Top100:          val(o.Count),
	}
	if links != nil {
		out.Backlinks = links.Backlinks
		out.ReferringDomains = links.ReferringDomains
		out.DomainRank = links.DomainRank
	}
	return out, nil
}

func (d *DataForSEO) DomainKeywords(ctx context.Context, dom string, locationCode int32, languageCode string, limit int) ([]domain.DomainKeyword, error) {
	if limit <= 0 {
		limit = 200
	}
	type res struct {
		Items []struct {
			KeywordData *struct {
				Keyword     string `json:"keyword"`
				KeywordInfo *struct {
					SearchVolume *int32 `json:"search_volume"`
				} `json:"keyword_info"`
			} `json:"keyword_data"`
			RankedSerpElement *struct {
				SerpItem *struct {
					RankGroup *int32 `json:"rank_group"`
					URL       string `json:"url"`
				} `json:"serp_item"`
			} `json:"ranked_serp_element"`
		} `json:"items"`
	}
	rs, err := postInto[res](ctx, d, "/v3/dataforseo_labs/google/ranked_keywords/live", []any{map[string]any{"target": dom, "location_code": locationCode, "language_code": languageCode, "limit": limit, "order_by": []string{"keyword_data.keyword_info.search_volume,desc"}}})
	if err != nil {
		return nil, err
	}
	out := []domain.DomainKeyword{}
	if len(rs) == 0 {
		return out, nil
	}
	for _, i := range rs[0].Items {
		if i.KeywordData == nil || i.KeywordData.Keyword == "" {
			continue
		}
		k := domain.DomainKeyword{Keyword: i.KeywordData.Keyword, Position: 100}
		if i.KeywordData.KeywordInfo != nil && i.KeywordData.KeywordInfo.SearchVolume != nil {
			k.Volume = *i.KeywordData.KeywordInfo.SearchVolume
		}
		if i.RankedSerpElement != nil && i.RankedSerpElement.SerpItem != nil {
			if i.RankedSerpElement.SerpItem.RankGroup != nil {
				k.Position = *i.RankedSerpElement.SerpItem.RankGroup
			}
			k.URL = i.RankedSerpElement.SerpItem.URL
		}
		out = append(out, k)
	}
	return out, nil
}

func (d *DataForSEO) Competitors(ctx context.Context, dom string, locationCode int32, languageCode string) ([]domain.CompetitorSuggestion, error) {
	type res struct {
		Items []struct {
			Domain        string `json:"domain"`
			Intersections *int32 `json:"intersections"`
			Metrics       *struct {
				Organic *struct {
					ETV *float64 `json:"etv"`
				} `json:"organic"`
			} `json:"metrics"`
		} `json:"items"`
	}
	rs, err := postInto[res](ctx, d, "/v3/dataforseo_labs/google/competitors_domain/live", []any{map[string]any{"target": dom, "location_code": locationCode, "language_code": languageCode, "limit": 10, "exclude_top_domains": true}})
	if err != nil {
		return nil, err
	}
	out := []domain.CompetitorSuggestion{}
	if len(rs) == 0 {
		return out, nil
	}
	for _, i := range rs[0].Items {
		if i.Domain == "" || i.Domain == dom {
			continue
		}
		c := domain.CompetitorSuggestion{Domain: i.Domain}
		if i.Intersections != nil {
			c.Intersections = *i.Intersections
		}
		if i.Metrics != nil && i.Metrics.Organic != nil && i.Metrics.Organic.ETV != nil {
			c.OrganicTraffic = int32(math.Round(*i.Metrics.Organic.ETV))
		}
		out = append(out, c)
	}
	return out, nil
}
