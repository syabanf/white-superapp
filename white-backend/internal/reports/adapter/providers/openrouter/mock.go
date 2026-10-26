package openrouter

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/syabanf/white-superapp/white-backend/internal/reports/domain"
)

// Mock fills templates with the payload numbers: same input, same output.
// Headings match the real adapter's contract (always Indonesian). The
// copy, em dashes included, is the old lib/providers/openrouter/mock.ts.
type Mock struct{}

func (Mock) Model() string { return MockModel }

func (Mock) Generate(_ context.Context, in domain.InsightInput) (string, error) {
	return GenerateMock(in), nil
}

var (
	fc  = domain.FormatCompact
	fn  = func(v float64) string { return domain.FormatNumber(v, 0) }
	fn1 = func(v float64) string { return domain.FormatNumber(v, 1) }
	fp  = func(v float64) string { return domain.FormatPercent(v, 2) }
	fp1 = func(v float64) string { return domain.FormatPercent(v, 1) }
)

func pick(lang, id, en string) string {
	if lang == "id" {
		return id
	}
	return en
}

func trend(pct *float64, lang string) string {
	switch {
	case pct == nil:
		return pick(lang, "stabil", "flat")
	case *pct > 1:
		return pick(lang, "naik ", "up ") + fp1(*pct)
	case *pct < -1:
		return pick(lang, "turun ", "down ") + fp1(math.Abs(*pct))
	}
	return pick(lang, "relatif stabil", "roughly flat")
}

func orZero(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// sliceUTF16 is String.prototype.slice(0, n).
func sliceUTF16(s string, n int) string {
	u := utf16.Encode([]rune(s))
	if len(u) <= n {
		return s
	}
	return string(utf16.Decode(u[:n]))
}

type lines struct {
	highlight, fix, rec, extra []string
}

func socialLines(s *domain.InsightSocial, lang string) lines {
	var best *domain.InsightTopPost
	if len(s.TopPosts) > 0 {
		best = &s.TopPosts[0]
	}
	var l lines
	if lang == "id" {
		l.highlight = []string{
			fmt.Sprintf("**Followers** kini %s (%s dengan tambahan %s followers).", fc(float64(s.Followers)), trend(s.FollowerGrowthPct, lang), fn(s.FollowerGrowth)),
			fmt.Sprintf("**Engagement rate** rata-rata %s dari %s postingan; jangkauan %s di %s.", fp(s.EngagementRate), fn(float64(s.Posts)), trend(s.ReachDeltaPct, lang), fc(float64(s.Reach))),
		}
		if best != nil {
			l.highlight = append(l.highlight, fmt.Sprintf("Konten terbaik: \"%s\" (%s, %s) dengan %s interaksi.", best.Caption, best.Platform, best.Type, fc(float64(best.Engagements))))
		}
		if orZero(s.EngagementRateDeltaPct) < 0 {
			l.fix = append(l.fix, fmt.Sprintf("Engagement rate melemah %s — kualitas interaksi perlu jadi fokus, bukan sekadar frekuensi posting.", fp1(math.Abs(orZero(s.EngagementRateDeltaPct)))))
		} else {
			l.fix = append(l.fix, "Pertahankan momentum engagement; ruang tumbuh masih ada di format video pendek.")
		}
		if orZero(s.ReachDeltaPct) < 0 {
			l.fix = append(l.fix, "Jangkauan menurun — distribusi konten terlalu bergantung pada followers existing.")
		} else {
			l.fix = append(l.fix, "Perluas jangkauan non-follower lewat kolaborasi dan konten shareable.")
		}
		quote := ""
		if best != nil {
			quote = fmt.Sprintf(" (\"%s…\")", sliceUTF16(best.Caption, 40))
		}
		l.rec = []string{"Replikasi pola konten terbaik" + quote + " minimal 2x per minggu.", "Naikkan rasio konten video/Reels untuk mendongkrak jangkauan organik."}
		return l
	}
	l.highlight = []string{
		fmt.Sprintf("**Followers** now at %s (%s, +%s net).", fc(float64(s.Followers)), trend(s.FollowerGrowthPct, lang), fn(s.FollowerGrowth)),
		fmt.Sprintf("Average **engagement rate** %s across %s posts; reach %s at %s.", fp(s.EngagementRate), fn(float64(s.Posts)), trend(s.ReachDeltaPct, lang), fc(float64(s.Reach))),
	}
	if best != nil {
		l.highlight = append(l.highlight, fmt.Sprintf("Best content: \"%s\" (%s, %s) with %s engagements.", best.Caption, best.Platform, best.Type, fc(float64(best.Engagements))))
	}
	if orZero(s.EngagementRateDeltaPct) < 0 {
		l.fix = append(l.fix, fmt.Sprintf("Engagement rate softened %s — prioritise interaction quality over posting volume.", fp1(math.Abs(orZero(s.EngagementRateDeltaPct)))))
	} else {
		l.fix = append(l.fix, "Keep the engagement momentum; short-form video still has headroom.")
	}
	if orZero(s.ReachDeltaPct) < 0 {
		l.fix = append(l.fix, "Reach declined — distribution relies too heavily on the existing follower base.")
	} else {
		l.fix = append(l.fix, "Expand non-follower reach through collaborations and shareable formats.")
	}
	quote := ""
	if best != nil {
		quote = fmt.Sprintf(" (\"%s…\")", sliceUTF16(best.Caption, 40))
	}
	l.rec = []string{"Replicate the top-performing content pattern" + quote + " at least twice a week.", "Increase the share of video/Reels to lift organic reach."}
	return l
}

func seoLines(s *domain.InsightSeo, lang string) lines {
	var top *domain.InsightTopQuery
	if len(s.TopQueries) > 0 {
		top = &s.TopQueries[0]
	}
	var opp []domain.InsightTopQuery
	for _, q := range s.TopQueries {
		if q.Position > 3 {
			opp = append(opp, q)
		}
	}
	opp = opp[:min(5, len(opp))]
	health := ""
	var l lines
	if lang == "id" {
		if s.Health != nil {
			health = fmt.Sprintf("; skor kesehatan situs %s/100", fn(float64(*s.Health)))
		}
		l.highlight = []string{
			fmt.Sprintf("**Klik organik** %s (%s) dari %s impresi; CTR %s.", fc(float64(s.Clicks)), trend(s.ClicksDeltaPct, lang), fc(float64(s.Impressions)), fp(s.Ctr)),
			fmt.Sprintf("Posisi rata-rata %s%s.", fn1(s.Position), health),
		}
		if top != nil {
			l.highlight = append(l.highlight, fmt.Sprintf("Kata kunci terkuat: \"%s\" — %s klik di posisi %s.", top.Query, fn(float64(top.Clicks)), fn1(top.Position)))
		}
		if orZero(s.ClicksDeltaPct) < 0 {
			l.fix = append(l.fix, "Klik organik menurun — audit konten yang kehilangan peringkat dan segarkan title/meta.")
		} else {
			l.fix = append(l.fix, "Jaga tren klik dengan memperbarui konten pilar secara berkala.")
		}
		if s.PositionDeltaAbs > 0 {
			l.fix = append(l.fix, fmt.Sprintf("Posisi rata-rata melemah %s poin — periksa halaman yang tergeser kompetitor.", fn1(math.Abs(s.PositionDeltaAbs))))
		} else {
			l.fix = append(l.fix, "Perkuat internal linking ke halaman komersial utama.")
		}
		if top != nil {
			l.rec = append(l.rec, fmt.Sprintf("Optimalkan halaman untuk \"%s\" agar bertahan di posisi teratas.", top.Query))
		}
		l.rec = append(l.rec, "Perbaiki title & meta description pada halaman ber-CTR rendah di halaman 1.")
		for _, q := range opp {
			l.extra = append(l.extra, fmt.Sprintf("**\"%s\"** — posisi %s, %s impresi: satu dorongan konten lagi menuju top 3.", q.Query, fn1(q.Position), fc(float64(q.Impressions))))
		}
		return l
	}
	if s.Health != nil {
		health = fmt.Sprintf("; site health score %s/100", fn(float64(*s.Health)))
	}
	l.highlight = []string{
		fmt.Sprintf("**Organic clicks** %s (%s) from %s impressions; CTR %s.", fc(float64(s.Clicks)), trend(s.ClicksDeltaPct, lang), fc(float64(s.Impressions)), fp(s.Ctr)),
		fmt.Sprintf("Average position %s%s.", fn1(s.Position), health),
	}
	if top != nil {
		l.highlight = append(l.highlight, fmt.Sprintf("Strongest query: \"%s\" — %s clicks at position %s.", top.Query, fn(float64(top.Clicks)), fn1(top.Position)))
	}
	if orZero(s.ClicksDeltaPct) < 0 {
		l.fix = append(l.fix, "Organic clicks declined — audit pages losing rank and refresh titles/meta.")
	} else {
		l.fix = append(l.fix, "Protect the click trend by refreshing pillar content regularly.")
	}
	if s.PositionDeltaAbs > 0 {
		l.fix = append(l.fix, fmt.Sprintf("Average position slipped %s points — review pages displaced by competitors.", fn1(math.Abs(s.PositionDeltaAbs))))
	} else {
		l.fix = append(l.fix, "Strengthen internal links into key commercial pages.")
	}
	if top != nil {
		l.rec = append(l.rec, fmt.Sprintf("Defend the top position for \"%s\" with content upkeep.", top.Query))
	}
	l.rec = append(l.rec, "Rewrite titles & meta descriptions on page-1 URLs with below-expected CTR.")
	for _, q := range opp {
		l.extra = append(l.extra, fmt.Sprintf("**\"%s\"** — position %s, %s impressions: one content push away from top 3.", q.Query, fn1(q.Position), fc(float64(q.Impressions))))
	}
	return l
}

func adsLines(a *domain.InsightAds, lang string) lines {
	cur := a.Currency
	if cur == "" {
		cur = "IDR"
	}
	money := func(v float64) string { return domain.FormatCurrency(v, cur, false) }
	moneyC := func(v float64) string { return domain.FormatCurrency(v, cur, true) }
	// Sorted by CPR ascending; a zero CPR sorts last like `cpr || Infinity`.
	idx := make([]int, len(a.Campaigns))
	for i := range idx {
		idx[i] = i
	}
	key := func(i int) float64 {
		if a.Campaigns[i].Cpr == 0 {
			return math.Inf(1)
		}
		return a.Campaigns[i].Cpr
	}
	for i := 1; i < len(idx); i++ { // stable insertion sort, like Array.prototype.sort
		for j := i; j > 0 && key(idx[j]) < key(idx[j-1]); j-- {
			idx[j], idx[j-1] = idx[j-1], idx[j]
		}
	}
	efficient := func(i int) bool { return a.Campaigns[i].Results > 0 && a.Campaigns[i].Cpr > 0 }
	best, worst := -1, -1
	for _, i := range idx {
		if efficient(i) {
			best = i
			break
		}
	}
	for k := len(idx) - 1; k >= 0; k-- {
		if i := idx[k]; efficient(i) && i != best {
			worst = i
			break
		}
	}
	total := 0.0
	for _, c := range a.Campaigns {
		total += c.Spend
	}
	if total == 0 {
		total = a.Spend
	}
	roas := ""
	if a.Roas != nil {
		roas = ", ROAS " + domain.FormatNumber(*a.Roas, 2) + "x"
	}
	budgetCamps := a.Campaigns[:min(5, len(a.Campaigns))]
	budget := func(ofTotal, results string) []string {
		var out []string
		for _, c := range budgetCamps {
			share := 0.0
			if total > 0 {
				share = c.Spend / total * 100
			}
			cpr := ""
			if c.Results > 0 && c.Cpr > 0 {
				cpr = ", CPR " + money(c.Cpr)
			}
			out = append(out, fmt.Sprintf("**%s** — %s (%s %s), %s %s%s.", c.Name, moneyC(c.Spend), fp1(share), ofTotal, fn(float64(c.Results)), results, cpr))
		}
		return out
	}
	var l lines
	if lang == "id" {
		l.highlight = []string{
			fmt.Sprintf("**Belanja iklan** %s (%s) menghasilkan %s hasil (%s).", moneyC(a.Spend), trend(a.SpendDeltaPct, lang), fn(float64(a.Results)), trend(a.ResultsDeltaPct, lang)),
			fmt.Sprintf("Biaya per hasil %s (%s); CTR %s, CPM %s%s.", money(a.Cpr), trend(a.CprDeltaPct, lang), fp(a.Ctr), money(a.Cpm), roas),
		}
		if best >= 0 {
			b := a.Campaigns[best]
			l.highlight = append(l.highlight, fmt.Sprintf("Kampanye paling efisien: **%s** — CPR %s untuk %s hasil.", b.Name, money(b.Cpr), fn(float64(b.Results))))
		}
		if orZero(a.CprDeltaPct) > 0 {
			l.fix = append(l.fix, fmt.Sprintf("Biaya per hasil naik %s — segarkan materi iklan sebelum fatigue menggerus efisiensi.", fp1(orZero(a.CprDeltaPct))))
		} else {
			l.fix = append(l.fix, "Efisiensi biaya terjaga; uji kreatif baru agar tidak stagnan.")
		}
		if worst >= 0 {
			w := a.Campaigns[worst]
			l.fix = append(l.fix, fmt.Sprintf("Kampanye **%s** paling boros (CPR %s) — evaluasi targeting dan penawarannya.", w.Name, money(w.Cpr)))
		}
		if best >= 0 {
			l.rec = append(l.rec, fmt.Sprintf("Geser anggaran bertahap ke kampanye \"%s\" yang terbukti paling efisien.", a.Campaigns[best].Name))
		}
		l.rec = append(l.rec, "Siapkan 2–3 variasi kreatif baru untuk rotasi bulan depan.")
		l.extra = budget("dari total", "hasil")
		return l
	}
	l.highlight = []string{
		fmt.Sprintf("**Ad spend** %s (%s) produced %s results (%s).", moneyC(a.Spend), trend(a.SpendDeltaPct, lang), fn(float64(a.Results)), trend(a.ResultsDeltaPct, lang)),
		fmt.Sprintf("Cost per result %s (%s); CTR %s, CPM %s%s.", money(a.Cpr), trend(a.CprDeltaPct, lang), fp(a.Ctr), money(a.Cpm), roas),
	}
	if best >= 0 {
		b := a.Campaigns[best]
		l.highlight = append(l.highlight, fmt.Sprintf("Most efficient campaign: **%s** — CPR %s for %s results.", b.Name, money(b.Cpr), fn(float64(b.Results))))
	}
	if orZero(a.CprDeltaPct) > 0 {
		l.fix = append(l.fix, fmt.Sprintf("Cost per result rose %s — refresh creatives before fatigue erodes efficiency.", fp1(orZero(a.CprDeltaPct))))
	} else {
		l.fix = append(l.fix, "Cost efficiency holding; keep testing new creatives to avoid stagnation.")
	}
	if worst >= 0 {
		w := a.Campaigns[worst]
		l.fix = append(l.fix, fmt.Sprintf("Campaign **%s** is the least efficient (CPR %s) — review targeting and offer.", w.Name, money(w.Cpr)))
	}
	if best >= 0 {
		l.rec = append(l.rec, fmt.Sprintf("Gradually shift budget toward \"%s\", the proven efficiency leader.", a.Campaigns[best].Name))
	}
	l.rec = append(l.rec, "Prepare 2–3 new creative variations for next month's rotation.")
	l.extra = budget("of total", "results")
	return l
}

func summarize(data domain.InsightPayload, clientName string, days int, lang string) string {
	var bits []string
	if s := data.Social; s != nil {
		bits = append(bits, fmt.Sprintf("%s followers (%s)", fc(float64(s.Followers)), trend(s.FollowerGrowthPct, lang)))
	}
	if s := data.Seo; s != nil {
		bits = append(bits, fmt.Sprintf("%s %s (%s)", fc(float64(s.Clicks)), pick(lang, "klik organik", "organic clicks"), trend(s.ClicksDeltaPct, lang)))
	}
	if a := data.Ads; a != nil {
		var inverted *float64
		if a.CprDeltaPct != nil {
			v := -*a.CprDeltaPct
			inverted = &v
		}
		spend := domain.FormatCurrency(a.Spend, a.Currency, true)
		bits = append(bits, pick(lang,
			fmt.Sprintf("belanja iklan %s dengan CPR %s", spend, trend(inverted, lang)),
			fmt.Sprintf("%s ad spend with CPR %s", spend, trend(inverted, lang))))
	}
	joined := strings.Join(bits, ", ")
	if lang == "id" {
		if joined == "" {
			joined = "aktivitas pemasaran yang terbatas"
		}
		return fmt.Sprintf("Dalam %d hari terakhir, %s mencatat %s. Analisa di bawah merangkum pendorong utama performa periode ini beserta prioritas eksekusi 30 hari ke depan.", days, clientName, joined)
	}
	if joined == "" {
		joined = "limited marketing activity"
	}
	return fmt.Sprintf("Over the last %d days, %s recorded %s. The analysis below summarises the main performance drivers this period and the execution priorities for the next 30 days.", days, clientName, joined)
}

// GenerateMock is generateMockInsight.
func GenerateMock(in domain.InsightInput) string {
	lang := domain.Lang(in.Language)
	var social, seo, ads lines
	hasSeo, hasAds := in.Data.Seo != nil, in.Data.Ads != nil
	if in.Data.Social != nil {
		social = socialLines(in.Data.Social, lang)
	}
	if hasSeo {
		seo = seoLines(in.Data.Seo, lang)
	}
	if hasAds {
		ads = adsLines(in.Data.Ads, lang)
	}
	highlights := concat(social.highlight, seo.highlight, ads.highlight)
	fixes := concat(social.fix, seo.fix, ads.fix)
	recs := concat(ads.rec, seo.rec, social.rec, []string{
		pick(lang, "Tinjau KPI mingguan bersama tim dan sepakati satu eksperimen baru per kanal.", "Review KPIs weekly with the team and agree one new experiment per channel."),
		pick(lang, "Dokumentasikan pembelajaran periode ini sebagai basis kalender konten & media plan bulan depan.", "Document this period's learnings as the basis for next month's content calendar & media plan."),
		pick(lang, "Selaraskan pesan organik dan iklan agar funnel dari jangkauan ke konversi konsisten.", "Align organic and paid messaging so the funnel from reach to conversion stays consistent."),
	})
	if len(recs) > 5 {
		recs = recs[:5]
	}
	for len(recs) < 5 {
		recs = append(recs, pick(lang, "Pertahankan ritme evaluasi dua mingguan.", "Keep a fortnightly evaluation cadence."))
	}

	var out []string
	bullets := func(items []string) {
		for _, it := range items {
			out = append(out, "- "+it)
		}
	}
	out = append(out, "## Ringkasan eksekutif", summarize(in.Data, in.ClientName, in.Range.Days, lang), "", "## Sorotan performa")
	bullets(highlights)
	out = append(out, "", "## Area perbaikan")
	bullets(fixes)
	out = append(out, "")
	if hasSeo && in.Module == domain.ModuleSeo {
		out = append(out, "## Peluang kata kunci")
		if len(seo.extra) > 0 {
			bullets(seo.extra)
		} else {
			out = append(out, pick(lang, "- Belum ada peluang menonjol pada periode ini.", "- No standout opportunities this period."))
		}
		out = append(out, "")
	}
	if hasAds && in.Module == domain.ModuleAds {
		out = append(out, "## Alokasi anggaran")
		if len(ads.extra) > 0 {
			bullets(ads.extra)
		} else {
			out = append(out, pick(lang, "- Belum ada kampanye berjalan pada periode ini.", "- No campaigns ran this period."))
		}
		out = append(out, "")
	}
	out = append(out, "## Rekomendasi 30 hari")
	for i, r := range recs {
		out = append(out, strconv.Itoa(i+1)+". "+r)
	}
	return strings.Join(out, "\n")
}

func concat(parts ...[]string) []string {
	var out []string
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}
