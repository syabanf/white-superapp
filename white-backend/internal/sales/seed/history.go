package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
)

// History loads 18 months of CRM history for the dashboard: leads, deals
// moving through the stages with won and lost outcomes, quotations, payment
// terms, invoices and payments, activities, chat conversations, form
// submissions and monthly targets. It writes rows directly (backdated, no
// events, so finance never sees these invoices) with fixed "hist-" ids. It
// does nothing once the history exists, so running it again changes nothing.
func History(ctx context.Context, pool *pgxpool.Pool, owners []string, now time.Time) error {
	var done bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM sales.leads WHERE id = 'hist-lead-0001')`).Scan(&done); err != nil || done {
		return err
	}
	if len(owners) == 0 {
		owners = []string{"demo-sales-1", "demo-sales-2"}
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	g := &historyGen{tx: tx, ctx: ctx, rnd: rand.New(rand.NewSource(20261026)), owners: owners, now: now.In(domain.Jakarta)}
	if err := g.run(); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type historyGen struct {
	tx     pgx.Tx
	ctx    context.Context
	rnd    *rand.Rand
	owners []string
	now    time.Time
	err    error
	n      map[string]int
}

func (g *historyGen) exec(sql string, args ...any) {
	if g.err == nil {
		if _, err := g.tx.Exec(g.ctx, sql, args...); err != nil {
			g.err = fmt.Errorf("%s: %w", strings.Fields(sql)[2], err)
		}
	}
}

// id returns the next fixed id of a kind ("hist-lead-0001").
func (g *historyGen) id(kind string) string {
	g.n[kind]++
	return fmt.Sprintf("hist-%s-%04d", kind, g.n[kind])
}

func (g *historyGen) pick(list []string) string { return list[g.rnd.Intn(len(list))] }

// weighted picks a key by integer weight; keys keep their order.
func (g *historyGen) weighted(pairs ...any) string {
	total := 0
	for i := 1; i < len(pairs); i += 2 {
		total += pairs[i].(int)
	}
	r := g.rnd.Intn(total)
	for i := 0; i < len(pairs); i += 2 {
		if r -= pairs[i+1].(int); r < 0 {
			return pairs[i].(string)
		}
	}
	return pairs[0].(string)
}

func (g *historyGen) between(lo, hi int) int { return lo + g.rnd.Intn(hi-lo+1) }

// later adds a random span of days (and working hours) to t.
func (g *historyGen) later(t time.Time, lo, hi int) time.Time {
	d := t.AddDate(0, 0, g.between(lo, hi))
	return time.Date(d.Year(), d.Month(), d.Day(), g.between(9, 17), g.rnd.Intn(60), 0, 0, domain.Jakarta)
}

type histAccount struct{ id, name, industry, kind, city string }

var histIndustries = []struct {
	industry, kind string
	names          []string
}{
	{"F&B", "umkm", []string{"Kopi Rasa Nusantara", "Bakmi Naga Mas", "Sate Pak Darmo", "Roti Bakar Delapan", "Dapur Nenek Ina", "Es Teh Segar Jaya", "Martabak Bulan Purnama", "Ayam Geprek Mantap"}},
	{"Kesehatan", "corporate", []string{"Klinik Pratama Sehat", "RSIA Bunda Kasih", "Apotek Medika Farma", "Klinik Gigi Senyum Ceria", "Lab Diagnostika Prima"}},
	{"Properti", "corporate", []string{"Bumi Asri Estate", "Taman Sari Residence", "Puri Indah Townhouse", "Graha Mandiri Property", "Serpong Hills Living"}},
	{"Pendidikan", "pendidikan", []string{"Sekolah Tunas Bangsa", "Bimbel Cerdas Ceria", "Kursus Bahasa Global", "Politeknik Nusantara", "TK Pelangi Kecil"}},
	{"Fashion & Kecantikan", "brand", []string{"Hijab Alila", "Sepatu Kaki Lokal", "Batik Sekar Jagad", "Kosmetik Ayu Natural", "Barbershop Gentleman", "Parfum Wangi Nusa"}},
	{"Hospitality", "corporate", []string{"Hotel Bukit Indah", "Villa Seminyak Retreat", "Resort Danau Asri", "Guesthouse Kota Lama"}},
	{"Teknologi", "startup", []string{"KasirKu App", "Logistik Cepat ID", "Dana Mitra Fintech", "Absensi Pro SaaS", "Belanja Desa"}},
	{"Retail", "umkm", []string{"Toko Elektronik Sinar", "Minimarket Sumber Rejeki", "Toko Mainan Ceria", "Grosir Sembako Berkah"}},
	{"Otomotif", "umkm", []string{"Bengkel Mobil Prima", "Dealer Motor Maju Jaya", "Cuci Mobil Kilap"}},
	{"Jasa Profesional", "corporate", []string{"Notaris Wijaya & Rekan", "Konsultan Pajak Amanah", "Firma Hukum Satria", "Akuntan Publik Cermat"}},
	{"Pemerintah", "pemerintah", []string{"Dinas Koperasi Kota", "BUMDes Makmur Sejahtera"}},
}

var (
	histCities    = []string{"Jakarta", "Bandung", "Surabaya", "Tangerang", "Bekasi", "Yogyakarta", "Semarang", "Denpasar", "Medan", "Makassar", "Malang", "Bogor"}
	histFirst     = []string{"Andi", "Budi", "Citra", "Dewi", "Eko", "Fitri", "Gilang", "Hana", "Indra", "Joko", "Kartika", "Lukman", "Maya", "Nanda", "Oki", "Putri", "Rangga", "Sari", "Taufik", "Wulan", "Yoga", "Zahra"}
	histLast      = []string{"Saputra", "Wijaya", "Lestari", "Pratama", "Hidayat", "Kusuma", "Santoso", "Rahmawati", "Nugroho", "Permata", "Siregar", "Halim"}
	histTitles    = []string{"Owner", "Marketing Manager", "Direktur", "Brand Manager", "Head of Growth", "Founder", "Kepala Humas"}
	histLineValue = map[string][2]int{domain.LineSocialMedia: {12, 30}, domain.LineSEO: {15, 40}, domain.LineAds: {15, 50}, domain.LineContent: {10, 35}, domain.LineWebsite: {20, 80}, domain.LineBranding: {25, 70}}
	// Share of leads per source that become a deal.
	histConvert = map[string]float64{"referral": .65, "linkedin": .5, "google": .45, "website": .45, "instagram": .4, "pameran": .4, "form": .4, "wa": .35, "chatbot": .35, "lainnya": .25}
)

// stage flow of the agency pipeline: code, id, days in stage, drop-out chance.
var histFlow = []struct {
	id       string
	lo, hi   int
	lostRate float64
}{
	{"st-prospek", 2, 9, .08},
	{"st-discovery", 3, 12, .12},
	{"st-proposal", 4, 16, .15},
	{"st-negosiasi", 3, 18, .15},
}

func (g *historyGen) run() error {
	g.n = map[string]int{}
	start := time.Date(g.now.Year(), g.now.Month(), 1, 0, 0, 0, 0, domain.Jakarta).AddDate(0, -17, 0)
	accounts := g.accounts()
	g.forms()
	accIdx := 0
	for m := 0; m < 18; m++ {
		month := start.AddDate(0, m, 0)
		leads := 11 + m/2 + g.rnd.Intn(6)
		for i := 0; i < leads; i++ {
			created := g.later(month, 0, month.AddDate(0, 1, -1).Day()-1)
			if created.After(g.now) {
				continue
			}
			acc := accounts[accIdx%len(accounts)]
			accIdx++
			g.lead(acc, created)
		}
		g.conversations(month)
		g.targets(month)
	}
	return g.err
}

func (g *historyGen) accounts() []histAccount {
	var out []histAccount
	for _, ind := range histIndustries {
		for _, name := range ind.names {
			a := histAccount{id: g.id("acc"), name: name, industry: ind.industry, kind: ind.kind, city: g.pick(histCities)}
			g.exec(`INSERT INTO sales.accounts (id, name, account_type, industry, city, owner_user_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $7) ON CONFLICT DO NOTHING`,
				a.id, a.name, a.kind, a.industry, a.city, g.owners[len(out)%len(g.owners)], g.now.AddDate(-2, 0, 0))
			out = append(out, a)
		}
	}
	g.rnd.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

var histForms = [][3]string{{"form-audit-seo", "audit-seo-gratis", "Audit SEO Gratis"}, {"form-penawaran-ads", "penawaran-ads", "Penawaran Iklan Digital"}}

func (g *historyGen) forms() {
	fields, _ := json.Marshal(contactForm)
	for _, f := range histForms {
		g.exec(`INSERT INTO sales.forms (id, slug, name, description, fields, default_source, lead_temperature, created_at) VALUES ($1, $2, $3, '', $4, 'form', 'hangat', $5) ON CONFLICT DO NOTHING`,
			f[0], f[1], f[2], fields, g.now.AddDate(-2, 0, 0))
	}
}

func (g *historyGen) lead(acc histAccount, created time.Time) {
	owner := g.owners[g.rnd.Intn(len(g.owners))]
	if len(g.owners) > 1 && g.rnd.Float64() < .15 {
		owner = g.owners[0]
	}
	source := g.weighted("instagram", 18, "google", 16, "referral", 12, "website", 12, "wa", 12, "linkedin", 8, "chatbot", 8, "form", 8, "pameran", 4, "lainnya", 2)
	pic := g.pick(histFirst) + " " + g.pick(histLast)
	score := int(math.Min(100, math.Max(0, g.rnd.NormFloat64()*18+map[bool]float64{true: 62, false: 42}[source == "referral" || source == "linkedin"])))
	temp := "dingin"
	if score >= 65 {
		temp = "panas"
	} else if score >= 40 {
		temp = "hangat"
	}
	convert := g.rnd.Float64() < histConvert[source]
	status := g.weighted("baru", 3, "dihubungi", 4, "tidak-cocok", 2)
	if convert {
		status = "qualified"
	}
	if created.After(g.now.AddDate(0, 0, -10)) && !convert {
		status = "baru"
	}
	id := g.id("lead")
	email := strings.ToLower(strings.Fields(pic)[0]) + "@" + strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(acc.name, " ", ""), "&", "")) + ".id"
	var accountID any
	if convert {
		accountID = acc.id
	}
	g.exec(`INSERT INTO sales.leads (id, org_name, org_type, pic_name, pic_title, pic_phone, pic_email, city, source, temperature, status, notes, owner_user_id, account_id, score, score_updated_at, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $13, $16, $16) ON CONFLICT DO NOTHING`,
		id, acc.name, acc.kind, pic, g.pick(histTitles), fmt.Sprintf("0812%08d", 10000000+g.rnd.Intn(89999999)), email, acc.city, source, temp, status,
		"Kebutuhan digital marketing untuk "+acc.industry+".", owner, accountID, score, created)
	if source == "form" || (source == "website" && g.rnd.Float64() < .5) {
		form := g.pick([]string{"form-konsultasi", histForms[0][0], histForms[1][0]})
		g.exec(`INSERT INTO sales.form_submissions (id, form_id, lead_id, payload, status, created_at) VALUES ($1, $2, $3, $4, 'accepted', $5) ON CONFLICT DO NOTHING`,
			g.id("fsub"), form, id, fmt.Sprintf(`{"pic_name": %q, "org_name": %q}`, pic, acc.name), created)
		if g.rnd.Float64() < .25 {
			g.exec(`INSERT INTO sales.form_submissions (id, form_id, payload, status, reason, created_at) VALUES ($1, $2, '{}', 'spam', 'honeypot', $3) ON CONFLICT DO NOTHING`,
				g.id("fsub"), form, created.Add(3*time.Hour))
		}
	}
	g.activity(id, "", owner, created.Add(2*time.Hour), "telepon", "Panggilan perkenalan "+acc.name)
	if convert {
		g.deal(id, acc, owner, g.later(created, 1, 6))
	}
}

func (g *historyGen) activity(leadID, dealID, owner string, at time.Time, kind, title string) {
	status, done := "done", any(at)
	if at.After(g.now) {
		status, done = "open", nil
	}
	subjectType, subjectID, lead, deal := "lead", leadID, any(leadID), any(nil)
	if dealID != "" {
		subjectType, subjectID, lead, deal = "deal", dealID, nil, dealID
	}
	g.exec(`INSERT INTO sales.activities (id, lead_id, deal_id, subject_type, subject_id, activity_type, title, status, due_at, done_at, owner_user_id, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11, $12, $12) ON CONFLICT DO NOTHING`,
		g.id("act"), lead, deal, subjectType, subjectID, kind, title, status, at, done, owner, minT(at, g.now))
}

func (g *historyGen) deal(leadID string, acc histAccount, owner string, created time.Time) {
	if created.After(g.now) {
		return
	}
	n := g.between(1, 3)
	lines := append([]string(nil), domain.ServiceLines...)
	g.rnd.Shuffle(len(lines), func(i, j int) { lines[i], lines[j] = lines[j], lines[i] })
	lines = lines[:n]
	value := 0
	for _, l := range lines {
		r := histLineValue[l]
		value += g.between(r[0], r[1])
	}
	estimate := float64(value) * 1_000_000
	labels := make([]string, n)
	for i, l := range lines {
		labels[i] = domain.ServiceLineLabel(l)
	}
	id := g.id("deal")
	// Walk the stages until the deal is won, lost, stalls or reaches today.
	at, stage, outcome := created, 0, ""
	var history []struct {
		stage string
		at    time.Time
	}
	stall := created.After(g.now.AddDate(0, -5, 0)) && g.rnd.Float64() < .35
	stallAt := g.between(1, 3)
	var quoteSent, closedAt time.Time
	for {
		history = append(history, struct {
			stage string
			at    time.Time
		}{histFlow[stage].id, at})
		if histFlow[stage].id == "st-proposal" && quoteSent.IsZero() {
			quoteSent = at.Add(26 * time.Hour)
		}
		next := g.later(at, histFlow[stage].lo, histFlow[stage].hi)
		if next.After(g.now) || (stall && stage >= stallAt) {
			break
		}
		if g.rnd.Float64() < histFlow[stage].lostRate {
			outcome, closedAt = "lost", next
			break
		}
		if stage == len(histFlow)-1 {
			outcome, closedAt = "won", next
			break
		}
		at, stage = next, stage+1
	}
	stageID, category, final, lostReason := histFlow[stage].id, []string{"pipeline", "pipeline", "best_case", "commit"}[stage], any(nil), any(nil)
	entered := history[len(history)-1].at
	var closed any
	switch outcome {
	case "won":
		stageID, category, closed, entered = "st-menang", "closed_won", closedAt, closedAt
		final = fmt.Sprintf("%.2f", estimate*(0.85+g.rnd.Float64()*0.15))
	case "lost":
		stageID, category, closed, entered = "st-kalah", "closed_lost", closedAt, closedAt
		lostReason = g.weighted("lr-harga", 30, "lr-kompetitor", 25, "lr-timing", 18, "lr-tidak-respon", 14, "lr-in-house", 9, "lr-lainnya", 4)
	}
	g.exec(`INSERT INTO sales.deals (id, lead_id, account_id, title, service_lines, pipeline_id, stage_id, value_estimate, value_final, owner_user_id, lost_reason_id, expected_close_date, forecast_category, entered_stage_at, closed_at, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'pl-agency', $6, $7, $8, $9, $10, $11, $12, $13, $14, $9, $15, $13) ON CONFLICT DO NOTHING`,
		id, leadID, acc.id, strings.Join(labels, " + ")+" "+acc.name, lines, stageID, fmt.Sprintf("%.2f", estimate), final, owner, lostReason,
		created.AddDate(0, 0, g.between(30, 60)), category, entered, closed, created)
	for i, h := range history {
		g.exec(`INSERT INTO sales.deal_stage_history (id, deal_id, stage_id, entered_at, moved_by) VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`, fmt.Sprintf("%s-h%d", id, i), id, h.stage, h.at, owner)
	}
	if outcome != "" {
		g.exec(`INSERT INTO sales.deal_stage_history (id, deal_id, stage_id, entered_at, moved_by) VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`, id+"-hx", id, stageID, closedAt, owner)
	}
	last := g.now
	if outcome != "" {
		last = closedAt
	}
	for i, n := 0, g.between(2, 5); i < n; i++ {
		at := created.Add(time.Duration(g.rnd.Int63n(int64(last.Sub(created)) + 1)))
		kind := g.weighted("meeting", 3, "wa", 4, "telepon", 3, "email", 2, "catatan", 1)
		g.activity("", id, owner, at, kind, map[string]string{"meeting": "Meeting kebutuhan", "wa": "Follow up WhatsApp", "telepon": "Telepon follow up", "email": "Kirim materi presentasi", "catatan": "Catatan diskusi"}[kind])
	}
	if outcome == "" {
		g.activity("", id, owner, g.later(g.now, 1, 7), "tugas", "Follow up keputusan "+acc.name)
	}
	if !quoteSent.IsZero() && !quoteSent.After(g.now) {
		g.quotation(id, owner, estimate, lines, quoteSent, outcome, closedAt)
	}
}

func (g *historyGen) quotation(dealID, owner string, estimate float64, lines []string, sent time.Time, outcome string, closedAt time.Time) {
	disc := []float64{0, 0, 5, 5, 10, 10, 15, 20}[g.rnd.Intn(8)]
	subtotal := estimate
	discNominal := subtotal * disc / 100
	ppn := (subtotal - discNominal) * 0.11
	total := subtotal - discNominal + ppn
	status, decided := "terkirim", any(nil)
	switch outcome {
	case "won":
		status, decided = "diterima", closedAt.Add(-time.Duration(g.between(2, 48))*time.Hour)
	case "lost":
		status, decided = "ditolak", closedAt
	}
	id := g.id("qt")
	g.exec(`INSERT INTO sales.quotations (id, deal_id, quote_number, status, subtotal, discount_percent, discount_nominal, ppn_nominal, total, valid_until, sent_at, decided_at, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $14) ON CONFLICT DO NOTHING`,
		id, dealID, "QH-"+strings.TrimPrefix(id, "hist-qt-"), status, money(subtotal), disc, money(discNominal), money(ppn), money(total), sent.AddDate(0, 0, 14), sent, decided, owner, sent.Add(-3*time.Hour))
	for i, l := range lines {
		price := subtotal / float64(len(lines))
		g.exec(`INSERT INTO sales.quotation_items (id, quotation_id, description, unit, qty, unit_price, line_total, sort_order) VALUES ($1, $2, $3, 'paket', 1, $4, $4, $5) ON CONFLICT DO NOTHING`,
			fmt.Sprintf("%s-i%d", id, i), id, domain.ServiceLineLabel(l), money(price), i)
	}
	if outcome == "won" {
		g.billing(dealID, id, owner, total, closedAt)
	}
}

// billing splits the accepted quotation into terms, invoices the ones due by
// today and records most of their payments.
func (g *historyGen) billing(dealID, quoteID, owner string, total float64, won time.Time) {
	terms := [][2]any{{"DP 50%", 50.0}, {"Pelunasan 50%", 50.0}}
	if total > 45_000_000 {
		terms = [][2]any{{"Termin 1 (30%)", 30.0}, {"Termin 2 (40%)", 40.0}, {"Termin 3 (30%)", 30.0}}
	}
	for i, t := range terms {
		due := won.AddDate(0, 0, 30*i)
		due = time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, domain.Jakarta)
		termID := fmt.Sprintf("%s-t%d", quoteID, i)
		label, pct := t[0].(string), t[1].(float64)
		g.exec(`INSERT INTO sales.quotation_terms (id, quotation_id, label, percent, due_date, sort_order) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING`, termID, quoteID, label, pct, due, i)
		if due.After(g.now) {
			continue
		}
		amount := total * pct / 100
		invID := g.id("inv")
		sent := due.Add(10 * time.Hour)
		g.exec(`INSERT INTO sales.invoices (id, deal_id, quotation_id, term_id, invoice_number, label, amount, due_date, status, sent_at, issued_by, created_by, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'terkirim', $9, $10, $10, $9, $9) ON CONFLICT DO NOTHING`,
			invID, dealID, quoteID, termID, "INVH-"+strings.TrimPrefix(invID, "hist-inv-"), label, money(amount), due.AddDate(0, 0, 14), sent, owner)
		age := int(g.now.Sub(due).Hours() / 24)
		unpaid := .08
		if age < 45 {
			unpaid = .35
		}
		if g.rnd.Float64() < unpaid {
			continue
		}
		paid := due.AddDate(0, 0, []int{3, 7, 10, 14, 18, 25, 35, 50}[g.rnd.Intn(8)])
		if paid.After(g.now) {
			continue
		}
		g.exec(`INSERT INTO sales.payments (id, deal_id, invoice_id, amount, method, paid_on, note, created_by, created_at) VALUES ($1, $2, $3, $4, 'transfer', $5, $6, $7, $8) ON CONFLICT DO NOTHING`,
			invID+"-p", dealID, invID, money(amount), paid, "Transfer "+label, owner, paid.Add(11*time.Hour))
	}
}

func (g *historyGen) conversations(month time.Time) {
	for i, n := 0, g.between(7, 13); i < n; i++ {
		created := g.later(month, 0, month.AddDate(0, 1, -1).Day()-1)
		if created.After(g.now) {
			continue
		}
		channel := g.weighted("web", 6, "whatsapp", 4)
		id := g.id("conv")
		secs := int(math.Exp(g.rnd.NormFloat64()*1.1 + 6.2)) // median ~8 minutes
		responded := created.Add(time.Duration(secs) * time.Second)
		status, resolved := "resolved", any(responded.Add(time.Duration(g.between(1, 30))*time.Hour))
		if created.After(g.now.AddDate(0, 0, -5)) {
			status, resolved = g.pick([]string{"open", "in_progress", "waiting_customer"}), nil
		}
		name := g.pick(histFirst)
		g.exec(`INSERT INTO sales.lead_conversations (id, channel, external_id, display_name, status, assigned_user_id, bot_active, last_message_at, last_message_preview, first_response_at, first_response_seconds, sla_response_breached, resolved_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, false, $7, $8, $7, $9, $10, $11, $12, $7) ON CONFLICT DO NOTHING`,
			id, channel, id, name, status, g.owners[i%len(g.owners)], responded, "Baik, tim kami siapkan penawarannya.", secs, secs > 900, resolved, created)
		g.exec(`INSERT INTO sales.lead_messages (id, conversation_id, direction, sender, body, status, created_at) VALUES ($1, $2, 'in', 'customer', $3, 'received', $4) ON CONFLICT DO NOTHING`,
			id+"-m1", id, "Halo, saya mau tanya paket "+g.pick([]string{"social media", "SEO", "iklan", "website"}), created)
		g.exec(`INSERT INTO sales.lead_messages (id, conversation_id, direction, sender, body, status, created_at) VALUES ($1, $2, 'out', 'agent', $3, 'sent', $4) ON CONFLICT DO NOTHING`,
			id+"-m2", id, "Baik, tim kami siapkan penawarannya.", responded)
	}
}

// targets: a growing company target split 55/45 between the first two owners.
func (g *historyGen) targets(month time.Time) {
	ym := month.Format("2006-01")
	idx := float64(month.Year()*12 + int(month.Month()) - (g.now.Year()*12 + int(g.now.Month())) + 17)
	company := math.Round((170+idx*6)/5) * 5 * 1_000_000
	g.exec(`INSERT INTO sales.targets (id, period_month, target_value, target_deals) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`, "hist-tg-"+ym, ym, money(company), 4+int(idx)/4)
	shares := []float64{.55, .45}
	for i, o := range g.owners {
		if i >= len(shares) {
			break
		}
		g.exec(`INSERT INTO sales.targets (id, user_id, period_month, target_value) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`, fmt.Sprintf("hist-tg-%s-%d", ym, i), o, ym, money(company*shares[i]))
	}
}

func money(v float64) string { return fmt.Sprintf("%.2f", v) }

func minT(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
