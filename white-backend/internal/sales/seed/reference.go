// Package seed loads the sales reference data (pipeline, catalog, templates,
// chatbot settings) and the demo data for Mixist. Both are idempotent: rows
// have fixed ids and inserts skip existing ones, so admins' edits survive.
package seed

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
)

// Reference inserts the data the service needs to work at all. It runs on
// every serve start and is cheap when everything exists.
func Reference(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	exec := func(sql string, args ...any) {
		if err == nil {
			_, err = tx.Exec(ctx, sql, args...)
		}
	}
	exec(`INSERT INTO sales.pipelines (id, code, name, is_default, sort_order) VALUES ('pl-agency', 'agency', 'Pipeline Agensi', true, 1) ON CONFLICT DO NOTHING`)
	for _, s := range stages {
		exec(`INSERT INTO sales.stages (id, pipeline_id, code, name, sort_order, probability, is_won, is_lost, stuck_threshold_days) VALUES ($1, 'pl-agency', $2, $3, $4, $5, $6, $7, $8) ON CONFLICT DO NOTHING`,
			s.id, s.code, s.name, s.order, s.prob, s.won, s.lost, s.stuck)
	}
	for i, r := range lostReasons {
		exec(`INSERT INTO sales.lost_reasons (id, code, name, sort_order) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`, "lr-"+r[0], r[0], r[1], i+1)
	}
	for i, c := range catalog {
		exec(`INSERT INTO sales.service_catalog (id, code, name, service_line, unit, default_price, description, is_retainer, is_default_for_line, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) ON CONFLICT DO NOTHING`, "cat-"+c.code, c.code, c.name, c.line, c.unit, c.price, c.desc, c.retainer, c.def, i+1)
		for _, t := range c.tiers {
			exec(`INSERT INTO sales.catalog_tiers (catalog_id, tier, price, description) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`, "cat-"+c.code, t[0], t[1], t[2])
		}
	}
	for _, d := range [][2]any{{3, "0"}, {6, "5"}, {12, "10"}} {
		exec(`INSERT INTO sales.retainer_discounts (min_months, discount_percent) VALUES ($1, $2) ON CONFLICT DO NOTHING`, d[0], d[1])
	}
	exec(`INSERT INTO sales.approval_rules (id, name, level, min_discount_percent, approver_role) VALUES
		('ar-1', 'Diskon di atas 10% perlu persetujuan manajer sales', 1, 10, 'sales.approve'),
		('ar-2', 'Diskon di atas 20% perlu persetujuan direktur', 2, 20, 'ADMIN') ON CONFLICT DO NOTHING`)
	for i, c := range caseStudies {
		exec(`INSERT INTO sales.case_studies (id, title, client_name, industry, service_lines, challenge, solution, results, sort_order) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT DO NOTHING`,
			c.id, c.title, c.client, c.industry, c.lines, c.challenge, c.solution, c.results, i+1)
	}
	for _, t := range contractTemplates {
		exec(`INSERT INTO sales.contract_templates (id, name, type, body) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`, t.id, t.name, t.kind, t.body)
	}
	items, _ := json.Marshal(domain.DefaultOnboardingItems)
	exec(`INSERT INTO sales.onboarding_templates (id, name, is_default, items) VALUES ('ob-default', 'Onboarding klien agensi', true, $1) ON CONFLICT DO NOTHING`, items)
	exec(`INSERT INTO sales.chatbot_settings (id, enabled, bot_name, persona, services_offered, pricing_hints, business_hours_start, business_hours_end, handoff_rules, greeting, min_confidence)
		VALUES ('default', true, $1, $2, $3, $4, 9, 18, $5, $6, 0.55) ON CONFLICT DO NOTHING`, chatbot.name, chatbot.persona, chatbot.services, chatbot.pricing, chatbot.handoff, chatbot.greeting)
	fields, _ := json.Marshal(contactForm)
	exec(`INSERT INTO sales.forms (id, slug, name, description, fields, default_source, lead_temperature) VALUES ('form-konsultasi', 'konsultasi-gratis', 'Konsultasi Gratis', 'Ceritakan kebutuhan digital marketing Anda, tim Mixist akan menghubungi dalam 1 hari kerja.', $1, 'website', 'hangat') ON CONFLICT DO NOTHING`, fields)
	for _, t := range replyTemplates {
		exec(`INSERT INTO sales.reply_templates (id, title, body) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, t[0], t[1], t[2])
	}
	exec(`INSERT INTO sales.scoring_rules (id, name, kind, field, operator, value, event_type, max_count, points) VALUES
		('sr-hot', 'Lead panas', 'field', 'temperature', 'eq', '"panas"', NULL, 1, 30),
		('sr-referral', 'Dari referral', 'field', 'source', 'eq', '"referral"', NULL, 1, 20),
		('sr-email', 'Punya email PIC', 'field', 'pic_email', 'not_empty', NULL, NULL, 1, 5),
		('sr-meeting', 'Meeting selesai', 'event', NULL, NULL, '"meeting"', 'task_done', 3, 15),
		('sr-chat', 'Aktif chat', 'event', NULL, NULL, NULL, 'chat_inbound', 5, 2),
		('sr-quote', 'Quotation terkirim', 'event', NULL, NULL, NULL, 'quotation_sent', 2, 25)
		ON CONFLICT DO NOTHING`)
	exec(`INSERT INTO sales.workflow_rules (id, name, object, trigger_type, conditions, actions) VALUES
		('wf-new-lead', 'Lead baru: telepon dalam 1 hari', 'lead', 'created', '[]', '[{"type":"create_task","title":"Hubungi {{lead.pic_name}} ({{lead.org_name}})","activity_type":"telepon","due_in_days":1,"assign_to":"owner"}]'),
		('wf-proposal', 'Masuk Proposal: siapkan quotation', 'deal', 'stage_changed', '[]', '[{"type":"create_task","title":"Kirim quotation untuk {{deal.title}}","activity_type":"tugas","due_in_days":2,"priority":"high"},{"type":"notify_in_app","title":"Deal masuk tahap Proposal","message":"{{deal.title}} siap dibuatkan quotation."}]')
		ON CONFLICT DO NOTHING`)
	exec(`UPDATE sales.workflow_rules SET trigger_config = '{"to_stage":"proposal"}' WHERE id = 'wf-proposal' AND trigger_config = '{}'`)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type stage struct {
	id, code, name string
	order, prob    int
	won, lost      bool
	stuck          *int
}

func days(n int) *int { return &n }

var stages = []stage{
	{"st-prospek", "prospek-baru", "Prospek baru", 1, 10, false, false, days(7)},
	{"st-discovery", "discovery-call", "Discovery call", 2, 25, false, false, days(7)},
	{"st-proposal", "proposal", "Proposal", 3, 50, false, false, days(10)},
	{"st-negosiasi", "negosiasi", "Negosiasi", 4, 75, false, false, days(14)},
	{"st-menang", "menang", "Menang", 5, 100, true, false, nil},
	{"st-kalah", "kalah", "Kalah", 6, 0, false, true, nil},
}

var lostReasons = [][2]string{
	{"harga", "Harga tidak masuk anggaran"},
	{"kompetitor", "Memilih agensi lain"},
	{"in-house", "Dikerjakan tim internal"},
	{"timing", "Waktu belum tepat"},
	{"tidak-respon", "Tidak ada respons"},
	{"lainnya", "Lainnya"},
}

type catalogItem struct {
	code, name, line, unit, price, desc string
	retainer, def                       bool
	tiers                               [][3]string
}

var catalog = []catalogItem{
	{"smm-retainer", "Social Media Management", domain.LineSocialMedia, "bulan", "8500000", "Strategi konten bulanan, desain feed, copywriting, penjadwalan, dan community management untuk Instagram dan TikTok.", true, true,
		[][3]string{{"basic", "5500000", "12 konten feed per bulan, 1 platform"}, {"standard", "8500000", "16 konten feed + 8 story per bulan, 2 platform"}, {"premium", "14000000", "24 konten feed, 12 story, 4 Reels per bulan, 3 platform"}}},
	{"seo-retainer", "SEO Bulanan", domain.LineSEO, "bulan", "7500000", "Audit teknis, riset kata kunci, optimasi on-page, 4 artikel SEO, dan laporan peringkat bulanan.", true, true,
		[][3]string{{"basic", "4500000", "10 kata kunci, 2 artikel per bulan"}, {"standard", "7500000", "25 kata kunci, 4 artikel per bulan"}, {"premium", "12500000", "50 kata kunci, 8 artikel, link building"}}},
	{"ads-management", "Ads Management (Meta dan Google)", domain.LineAds, "bulan", "6000000", "Setup dan optimasi kampanye, uji materi iklan, pelacakan konversi, laporan mingguan. Belum termasuk ad spend.", true, true,
		[][3]string{{"basic", "4000000", "Ad spend sampai Rp30 juta per bulan, 1 platform"}, {"standard", "6000000", "Ad spend sampai Rp75 juta per bulan, 2 platform"}, {"premium", "9500000", "Ad spend sampai Rp200 juta per bulan, 3 platform"}}},
	{"content-shoot", "Produksi Foto dan Video", domain.LineContent, "paket", "12000000", "Konsep kreatif, 1 hari produksi, 20 foto edit, dan 4 video pendek siap tayang.", false, true,
		[][3]string{{"basic", "7500000", "Setengah hari, 10 foto, 2 video"}, {"standard", "12000000", "1 hari, 20 foto, 4 video"}, {"premium", "22000000", "2 hari, 40 foto, 8 video, talent"}}},
	{"website-company", "Website Company Profile", domain.LineWebsite, "proyek", "25000000", "Desain UI responsif sampai 8 halaman, CMS, SEO dasar, integrasi GA4, dan pelatihan admin.", false, true,
		[][3]string{{"basic", "15000000", "5 halaman, template premium"}, {"standard", "25000000", "8 halaman, desain kustom"}, {"premium", "45000000", "15 halaman, multibahasa, blog"}}},
	{"website-maintenance", "Pemeliharaan Website", domain.LineWebsite, "bulan", "1500000", "Update keamanan, backup, dan perubahan konten ringan.", true, false, nil},
	{"branding-identity", "Brand Identity", domain.LineBranding, "proyek", "18000000", "Riset brand, logo, palet warna, tipografi, dan brand guideline.", false, true,
		[][3]string{{"basic", "10000000", "Logo dan palet warna"}, {"standard", "18000000", "Identitas lengkap dan guideline 20 halaman"}, {"premium", "30000000", "Strategi brand, identitas, dan aplikasi media"}}},
}

type caseStudy struct {
	id, title, client, industry  string
	lines                        []string
	challenge, solution, results string
}

var caseStudies = []caseStudy{
	{"cs-kopi", "Menaikkan penjualan online kedai kopi", "Kopi Senja Nusantara", "F&B", []string{domain.LineSocialMedia, domain.LineAds},
		"Penjualan online stagnan dan engagement Instagram di bawah 1%.", "Pilar konten baru, Reels mingguan, dan kampanye Meta Ads berbasis lokasi.", "Engagement naik ke 4,8% dan pesanan online naik 62% dalam 3 bulan."},
	{"cs-klinik", "Klinik kecantikan halaman satu Google", "Klinik Aurora Dermatologi", "Kesehatan", []string{domain.LineSEO, domain.LineWebsite},
		"Website lambat dan tidak muncul untuk kata kunci layanan utama.", "Redesign website, perbaikan Core Web Vitals, dan 40 artikel SEO.", "18 kata kunci di halaman satu dan trafik organik naik 3 kali lipat."},
	{"cs-properti", "Leads properti dengan biaya efisien", "Griya Asri Residence", "Properti", []string{domain.LineAds, domain.LineContent},
		"Biaya per lead Google Ads terlalu tinggi.", "Video tur unit, landing page baru, dan struktur kampanye ulang.", "Biaya per lead turun 41% dan 320 leads berkualitas per bulan."},
}

type contractTemplate struct{ id, name, kind, body string }

var contractTemplates = []contractTemplate{
	{"ct-retainer", "Perjanjian Jasa Retainer", "retainer", `# PERJANJIAN JASA DIGITAL MARKETING (RETAINER)

Nomor: {{contract_number}}

Perjanjian ini dibuat pada {{today}} oleh dan antara:

1. **{{agency_name}}**, berkedudukan di Jakarta Selatan, selanjutnya disebut **Pihak Pertama**.
2. **{{client_name}}**, beralamat di {{client_address}}, diwakili oleh {{pic_name}} selaku {{pic_title}}, selanjutnya disebut **Pihak Kedua**.

## Pasal 1. Ruang Lingkup Pekerjaan

Pihak Pertama memberikan jasa berikut sesuai penawaran {{quote_number}}:

{{services_table}}

## Pasal 2. Jangka Waktu

Perjanjian berlaku sejak {{start_date}} sampai {{end_date}} ({{duration_months}} bulan) dan dapat diperpanjang dengan kesepakatan tertulis.

## Pasal 3. Nilai dan Cara Pembayaran

Nilai perjanjian sebesar **{{fees}}** (termasuk PPN {{ppn}}), dibayar sebagai berikut:

{{payment_terms}}

Pembayaran dilakukan paling lambat 14 hari kalender sejak invoice diterima.

## Pasal 4. Kewajiban Para Pihak

- Pihak Pertama menjalankan pekerjaan secara profesional dan melaporkan hasil setiap bulan.
- Pihak Kedua memberikan akses akun, aset brand, dan persetujuan materi paling lambat 3 hari kerja.
- Biaya iklan (ad spend) dibayar langsung oleh Pihak Kedua ke platform terkait.

## Pasal 5. Kerahasiaan

Para pihak wajib menjaga kerahasiaan seluruh informasi yang diterima selama dan setelah berakhirnya perjanjian ini.

## Pasal 6. Pengakhiran

Salah satu pihak dapat mengakhiri perjanjian dengan pemberitahuan tertulis 30 hari sebelumnya. Pekerjaan yang telah berjalan tetap ditagihkan.

## Pasal 7. Ketentuan Lain

{{terms}}

Perselisihan diselesaikan secara musyawarah, dan bila tidak tercapai, melalui Pengadilan Negeri Jakarta Selatan.
`},
	{"ct-project", "Perjanjian Jasa Proyek", "project", `# PERJANJIAN PEKERJAAN PROYEK

Nomor: {{contract_number}}

Pada {{today}}, **{{agency_name}}** (**Pihak Pertama**) dan **{{client_name}}** yang diwakili {{pic_name}} (**Pihak Kedua**) sepakat mengikatkan diri dalam perjanjian pekerjaan proyek {{deal_title}}.

## Pasal 1. Lingkup Pekerjaan

{{services_list}}

Rincian biaya mengacu pada penawaran {{quote_number}}:

{{services_table}}

## Pasal 2. Jadwal

Pekerjaan dimulai {{start_date}} dan ditargetkan selesai {{end_date}}. Keterlambatan karena materi atau persetujuan dari Pihak Kedua menggeser jadwal secara proporsional.

## Pasal 3. Biaya dan Termin

Total biaya proyek **{{fees}}**, dengan termin:

{{payment_terms}}

## Pasal 4. Revisi dan Serah Terima

- Revisi mayor maksimal 2 kali per deliverable, revisi minor tidak dibatasi selama masa proyek.
- Serah terima dinyatakan selesai setelah berita acara ditandatangani atau 7 hari kerja tanpa keberatan tertulis.

## Pasal 5. Hak Kekayaan Intelektual

Hak atas hasil akhir beralih kepada Pihak Kedua setelah seluruh pembayaran lunas. Pihak Pertama boleh menampilkan hasil pekerjaan sebagai portofolio.

## Pasal 6. Ketentuan Lain

{{terms}}
`},
	{"ct-nda", "Perjanjian Kerahasiaan (NDA)", "nda", `# PERJANJIAN KERAHASIAAN

Nomor: {{contract_number}}

Perjanjian ini dibuat pada {{today}} antara **{{agency_name}}** dan **{{client_name}}** yang diwakili {{pic_name}}, terkait pembahasan kerja sama {{deal_title}}.

## Pasal 1. Informasi Rahasia

Informasi Rahasia mencakup data bisnis, strategi pemasaran, data pelanggan, akses akun, dan informasi lain yang ditandai atau patut diduga rahasia.

## Pasal 2. Kewajiban

- Menggunakan Informasi Rahasia hanya untuk tujuan kerja sama.
- Tidak mengungkapkan kepada pihak ketiga tanpa persetujuan tertulis.
- Mengembalikan atau memusnahkan Informasi Rahasia bila diminta.

## Pasal 3. Jangka Waktu

Kewajiban kerahasiaan berlaku sejak {{start_date}} sampai {{end_date}} dan tetap berlaku 2 tahun setelahnya.

## Pasal 4. Hukum yang Berlaku

Perjanjian ini tunduk pada hukum Republik Indonesia.
`},
}

var chatbot = struct{ name, persona, services, pricing, handoff, greeting string }{
	name:    "Wira",
	persona: "Asisten Mixist yang ramah, sigap, dan to the point. Memakai sapaan Kak, tidak berlebihan, selalu menawarkan langkah berikutnya.",
	services: `- Social media management (Instagram, TikTok, LinkedIn)
- SEO dan konten artikel
- Ads management (Meta Ads, Google Ads, TikTok Ads)
- Produksi foto dan video
- Pembuatan website company profile dan landing page
- Branding dan identitas visual`,
	pricing: `- Social media management mulai Rp5.500.000 per bulan
- SEO mulai Rp4.500.000 per bulan
- Ads management mulai Rp4.000.000 per bulan (belum termasuk ad spend)
- Produksi foto dan video mulai Rp7.500.000 per paket
- Website company profile mulai Rp15.000.000
- Brand identity mulai Rp10.000.000`,
	handoff: `- Pengguna minta bicara dengan manusia, sales, atau minta ditelepon
- Komplain atau ketidakpuasan
- Pertanyaan kontrak, pembayaran, atau harga khusus di luar panduan
- Kamu tidak yakin dengan jawabannya`,
	greeting: "Halo Kak! Saya Wira dari Mixist. Lagi cari bantuan untuk social media, SEO, iklan, konten, website, atau branding?",
}

var contactForm = []domain.FormField{
	{Key: "pic_name", Label: "Nama lengkap", Type: "text", Required: true},
	{Key: "org_name", Label: "Nama perusahaan atau brand", Type: "text", Required: true},
	{Key: "pic_phone", Label: "Nomor WhatsApp", Type: "phone"},
	{Key: "pic_email", Label: "Email", Type: "email"},
	{Key: "layanan", Label: "Layanan yang dibutuhkan", Type: "select", Options: []string{"Social media", "SEO", "Iklan digital", "Konten", "Website", "Branding"}},
	{Key: "budget", Label: "Perkiraan budget per bulan", Type: "select", Options: []string{"< Rp5 juta", "Rp5-15 juta", "Rp15-50 juta", "> Rp50 juta"}},
	{Key: "notes", Label: "Ceritakan kebutuhan Anda", Type: "textarea"},
}

var replyTemplates = [][3]string{
	{"rt-salam", "Salam pembuka", "Halo Kak, terima kasih sudah menghubungi Mixist. Boleh diceritakan kebutuhan digital marketing-nya?"},
	{"rt-jadwal", "Ajak discovery call", "Bagaimana kalau kita jadwalkan discovery call 30 menit minggu ini? Silakan pilih waktu yang nyaman."},
	{"rt-penawaran", "Penawaran dikirim", "Penawaran resmi sudah kami kirim ke email Kakak. Kalau ada pertanyaan, kami siap bantu."},
}
