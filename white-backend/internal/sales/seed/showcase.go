package seed

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/app"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
)

const showcaseMarker = "show-rt-closing"

// Showcase fills what the history leaves empty so every sales page has
// data: contacts, custom fields, deal members, quotations in every status
// with discount approvals and revisions, contracts in every status (one
// unsigned with a live signing link), pitch decks, onboardings with a
// brief and assets, conversation statuses, notes and AI insights, reply
// templates and case studies. It runs once (marker row written last) and
// never creates invoices or payments through the use cases, so no billing
// events leave the seed.
func Showcase(ctx context.Context, pool *pgxpool.Pool, a Apps, owners []string) error {
	var done bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM sales.reply_templates WHERE id = $1)`, showcaseMarker).Scan(&done); err != nil || done {
		return err
	}
	if len(owners) == 0 {
		owners = []string{"demo-sales-1", "demo-sales-2"}
	}
	admin := app.Actor{UserID: owners[0], Admin: true, Approver: true}
	manager := app.Actor{UserID: owners[len(owners)-1], Approver: true}
	steps := []struct {
		name string
		run  func() error
	}{
		{"custom fields", func() error { return showcaseFields(ctx, a) }},
		{"rows", func() error { return showcaseRows(ctx, pool, owners) }},
		{"quotations", func() error { return showcaseQuotes(ctx, pool, a, admin, manager) }},
		{"contracts", func() error { return showcaseContracts(ctx, pool, a, admin) }},
		{"onboardings", func() error { return showcaseOnboardings(ctx, pool, a, admin) }},
		{"decks", func() error { return showcaseDecks(ctx, pool, a, admin) }},
		{"insights", func() error { return showcaseInsights(ctx, pool, a) }},
	}
	for _, s := range steps {
		if err := s.run(); err != nil {
			return fmt.Errorf("%s: %w", s.name, err)
		}
	}
	_, err := pool.Exec(ctx, `INSERT INTO sales.reply_templates (id, title, body) VALUES ($1, 'Closing penawaran', 'Halo {nama}, penawaran untuk {perusahaan} berlaku sampai akhir minggu ini. Jika sudah cocok, kami siapkan kontraknya hari ini juga.') ON CONFLICT DO NOTHING`, showcaseMarker)
	return err
}

func showcaseFields(ctx context.Context, a Apps) error {
	fields := []domain.CustomField{
		{ID: "show-cf-budget", Object: "lead", Key: "budget_bulanan", Label: "Budget per bulan", FieldType: "picklist", Options: []string{"< Rp5 juta", "Rp5-15 juta", "Rp15-50 juta", "> Rp50 juta"}, ShowInList: true, SortOrder: 1, IsActive: true},
		{ID: "show-cf-keputusan", Object: "lead", Key: "pengambil_keputusan", Label: "Pengambil keputusan", FieldType: "text", SortOrder: 2, IsActive: true},
		{ID: "show-cf-durasi", Object: "deal", Key: "durasi_kontrak", Label: "Durasi kontrak (bulan)", FieldType: "number", SortOrder: 1, IsActive: true},
		{ID: "show-cf-kompetitor", Object: "deal", Key: "kompetitor", Label: "Agensi pesaing", FieldType: "text", SortOrder: 2, IsActive: true},
		{ID: "show-cf-cabang", Object: "account", Key: "jumlah_cabang", Label: "Jumlah cabang", FieldType: "number", SortOrder: 1, IsActive: true},
		{ID: "show-cf-ultah", Object: "contact", Key: "ulang_tahun", Label: "Ulang tahun", FieldType: "date", SortOrder: 1, IsActive: true},
	}
	for _, f := range fields {
		if _, err := a.Leads.SaveCustomField(ctx, f); err != nil {
			return err
		}
	}
	return nil
}

// showcaseRows writes plain rows in one transaction.
func showcaseRows(ctx context.Context, pool *pgxpool.Pool, owners []string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	stmts := []struct {
		sql  string
		args []any
	}{
		// Two contacts per history account; the first is primary and becomes the lead's contact.
		{`INSERT INTO sales.contacts (id, account_id, name, title, phone, email, is_primary, custom, created_at, updated_at)
		  SELECT a.id || '-c' || n, a.id,
		         (ARRAY['Rina','Bayu','Sinta','Dimas','Laras','Fajar','Nadia','Hendra'])[1 + (abs(hashtext(a.id)) + n) % 8] || ' ' || (ARRAY['Wijaya','Saputra','Lestari','Pratama','Kusuma','Halim'])[1 + (abs(hashtext(a.id)) + n * 3) % 6],
		         CASE n WHEN 1 THEN 'Owner' ELSE 'Marketing Lead' END,
		         '0813' || lpad(((abs(hashtext(a.id)) + n) % 100000000)::text, 8, '0'),
		         CASE n WHEN 1 THEN 'owner' ELSE 'marketing' END || '@' || lower(regexp_replace(a.name, '[^a-zA-Z0-9]', '', 'g')) || '.id',
		         n = 1, CASE n WHEN 1 THEN jsonb_build_object('ulang_tahun', '1988-0' || (1 + abs(hashtext(a.id)) % 9) || '-1' || (abs(hashtext(a.id)) % 9)) ELSE '{}'::jsonb END,
		         a.created_at, a.created_at
		  FROM sales.accounts a CROSS JOIN generate_series(1, 2) AS n
		  WHERE a.id LIKE 'hist-acc-%' ON CONFLICT DO NOTHING`, nil},
		{`UPDATE sales.leads l SET contact_id = l.account_id || '-c1' WHERE l.id LIKE 'hist-lead-%' AND l.account_id IS NOT NULL AND l.contact_id IS NULL`, nil},
		{`UPDATE sales.accounts SET custom = jsonb_build_object('jumlah_cabang', 1 + abs(hashtext(id)) % 6) WHERE id LIKE 'hist-acc-%' AND custom = '{}'`, nil},
		{`UPDATE sales.leads SET custom = jsonb_build_object('budget_bulanan', (ARRAY['< Rp5 juta','Rp5-15 juta','Rp15-50 juta','> Rp50 juta'])[1 + abs(hashtext(id)) % 4], 'pengambil_keputusan', pic_name)
		  WHERE id LIKE 'hist-lead-%' AND custom = '{}'`, nil},
		{`UPDATE sales.deals SET custom = jsonb_build_object('durasi_kontrak', (ARRAY[3,6,6,12])[1 + abs(hashtext(id)) % 4], 'kompetitor', (ARRAY['Agensi Lokal A','Freelancer','Tim internal','Agensi Jakarta B'])[1 + abs(hashtext(id)) % 4])
		  WHERE id LIKE 'hist-deal-%' AND custom = '{}'`, nil},
		// Owner plus a supporting member with a commission split on most deals.
		{`INSERT INTO sales.deal_members (deal_id, user_id, role, split_percent, created_at)
		  SELECT id, owner_user_id, 'owner', CASE WHEN abs(hashtext(id)) % 3 = 0 THEN 100 ELSE 70 END, created_at FROM sales.deals WHERE owner_user_id IS NOT NULL AND deleted_at IS NULL ON CONFLICT DO NOTHING`, nil},
		{`INSERT INTO sales.deal_members (deal_id, user_id, role, split_percent, created_at)
		  SELECT d.id, o.uid, (ARRAY['support','pre_sales','account_manager'])[1 + abs(hashtext(d.id)) % 3], 30, d.created_at
		  FROM sales.deals d CROSS JOIN unnest($1::text[]) AS o(uid)
		  WHERE d.owner_user_id IS NOT NULL AND o.uid <> d.owner_user_id AND abs(hashtext(d.id)) % 3 <> 0 AND d.deleted_at IS NULL ON CONFLICT DO NOTHING`, []any{owners}},
		// Inbox: recent conversations in every working status, some flagged.
		{`UPDATE sales.lead_conversations SET status = (ARRAY['open','in_progress','waiting_customer'])[1 + abs(hashtext(id)) % 3],
		         resolved_at = NULL, unread_count = abs(hashtext(id)) % 3, priority = (ARRAY['normal','high','urgent','normal'])[1 + abs(hashtext(id)) % 4]
		  WHERE id LIKE 'hist-conv-%' AND created_at > now() - interval '21 days'`, nil},
		{`UPDATE sales.lead_conversations SET is_complaint = true, priority = 'high' WHERE id IN (SELECT id FROM sales.lead_conversations WHERE id LIKE 'hist-conv-%' ORDER BY created_at DESC OFFSET 2 LIMIT 1)`, nil},
		{`UPDATE sales.lead_conversations c SET lead_id = l.id, display_name = l.pic_name
		  FROM (SELECT id, pic_name, row_number() OVER (ORDER BY created_at DESC) AS rn FROM sales.leads WHERE id LIKE 'hist-lead-%' AND source IN ('chatbot', 'wa')) l,
		       (SELECT id, row_number() OVER (ORDER BY created_at DESC) AS rn FROM sales.lead_conversations WHERE id LIKE 'hist-conv-%') cc
		  WHERE c.id = cc.id AND cc.rn = l.rn AND c.lead_id IS NULL`, nil},
		{`INSERT INTO sales.conversation_notes (id, conversation_id, author_user_id, body, created_at)
		  SELECT c.id || '-n', c.id, coalesce(c.assigned_user_id, $1), (ARRAY['Minta dikirimi portofolio F&B sebelum meeting.','Budget masih dibahas dengan owner, follow up Kamis.','Klien komplain respons lambat, prioritaskan.','Sudah dikirim price list paket social media.'])[1 + abs(hashtext(c.id)) % 4], c.created_at + interval '2 hours'
		  FROM sales.lead_conversations c WHERE c.id LIKE 'hist-conv-%' AND c.created_at > now() - interval '45 days' ON CONFLICT DO NOTHING`, []any{owners[0]}},
		{`INSERT INTO sales.reply_templates (id, title, body) VALUES
		  ('show-rt-portfolio', 'Kirim portofolio', 'Berikut portofolio kami untuk industri {industri}: https://mixist.id/portofolio. Ada yang ingin dibahas lebih lanjut?'),
		  ('show-rt-jadwal', 'Ajak meeting', 'Boleh kami jadwalkan video call 30 menit minggu ini untuk membahas kebutuhan {perusahaan}? Silakan pilih hari yang cocok.')
		  ON CONFLICT DO NOTHING`, nil},
		{`INSERT INTO sales.case_studies (id, title, client_name, industry, service_lines, challenge, solution, results, sort_order) VALUES
		  ('show-cs-hotel', 'Direct booking naik 2x dalam 4 bulan', 'Hotel Bukit Indah', 'Hospitality', ARRAY['ads_management','content_production'], 'Ketergantungan pada OTA dengan komisi 18%.', 'Kampanye Meta dan Google untuk direct booking, video tur kamar mingguan.', 'Direct booking naik 112%, biaya per booking turun 35%.', 10),
		  ('show-cs-klinik', 'Klinik gigi penuh di jam sepi', 'Klinik Gigi Senyum Ceria', 'Kesehatan', ARRAY['seo','social_media_management'], 'Slot pagi dan siang kosong, ranking Google rendah.', 'SEO lokal 3 cabang, konten edukasi dan promo jam sepi di Instagram.', 'Reservasi online naik 68%, top 3 Google Maps untuk 12 kata kunci.', 11)
		  ON CONFLICT DO NOTHING`, nil},
		// Invoices in every status: a few submitted terms and one cancelled invoice (no events: direct rows).
		{`UPDATE sales.invoices SET status = 'diajukan', sent_at = NULL WHERE id IN (
		    SELECT i.id FROM sales.invoices i LEFT JOIN sales.payments p ON p.invoice_id = i.id
		    WHERE i.id LIKE 'hist-inv-%' AND p.id IS NULL AND i.created_at > now() - interval '20 days' ORDER BY i.created_at DESC LIMIT 3)`, nil},
		{`UPDATE sales.invoices SET status = 'batal', cancelled_at = created_at + interval '2 days', cancelled_by = $1, cancel_reason = 'Nilai termin direvisi, diganti invoice baru.' WHERE id IN (
		    SELECT i.id FROM sales.invoices i LEFT JOIN sales.payments p ON p.invoice_id = i.id
		    WHERE i.id LIKE 'hist-inv-%' AND i.status = 'terkirim' AND p.id IS NULL ORDER BY i.created_at LIMIT 1)`, []any{owners[0]}},
	}
	for _, s := range stmts {
		if _, err := tx.Exec(ctx, s.sql, s.args...); err != nil {
			return fmt.Errorf("%.60s: %w", s.sql, err)
		}
	}
	return tx.Commit(ctx)
}

// ids runs a query returning one text column.
func ids(ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) ([]string, error) {
	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func showcaseQuotes(ctx context.Context, pool *pgxpool.Pool, a Apps, admin, manager app.Actor) error {
	deals, err := ids(ctx, pool, `SELECT d.id FROM sales.deals d WHERE d.id LIKE 'hist-deal-%' AND d.stage_id IN ('st-proposal', 'st-negosiasi', 'st-discovery') ORDER BY d.created_at DESC LIMIT 5`)
	if err != nil {
		return err
	}
	if len(deals) < 5 {
		return fmt.Errorf("need 5 open deals, have %d", len(deals))
	}
	quote := func(dealID string, discount float64, price string) (*domain.Quotation, error) {
		return a.Quotes.CreateQuotation(ctx, admin, app.QuotationInput{DealID: dealID, UsePpn: true, DiscountPercent: discount, Notes: "Harga berlaku 14 hari.",
			Items: []app.QuotationItemInput{
				{ItemType: "bebas", Description: "Social Media Management", Scope: "12 konten feed, 20 story, laporan bulanan", Unit: "bulan", Qty: 6, UnitPrice: price},
				{ItemType: "bebas", Description: "Ads Management", Scope: "Meta dan Google Ads, optimasi mingguan", Unit: "bulan", Qty: 6, UnitPrice: "4500000"},
			},
			Terms: []app.QuotationTermInput{{Label: "DP 50%", Percent: 50}, {Label: "Pelunasan 50%", Percent: 50}}})
	}
	// Pending approval (15% needs level 1).
	if _, err := quote(deals[0], 15, "7500000"); err != nil {
		return err
	}
	// Approved by the manager, then sent.
	qb, err := quote(deals[1], 12, "6500000")
	if err != nil {
		return err
	}
	if qb.ApprovalRequestID != nil {
		if _, err := a.Quotes.Decide(ctx, manager, *qb.ApprovalRequestID, "approve", "Oke, klien retainer 6 bulan."); err != nil {
			return err
		}
	}
	if _, err := a.Quotes.SetStatus(ctx, admin, qb.ID, domain.QuoteSent); err != nil {
		return err
	}
	// Rejected at level 1 (25% would need two levels).
	qc, err := quote(deals[2], 25, "6000000")
	if err != nil {
		return err
	}
	if qc.ApprovalRequestID != nil {
		if _, err := a.Quotes.Decide(ctx, manager, *qc.ApprovalRequestID, "reject", "Diskon terlalu besar, maksimal 15%."); err != nil {
			return err
		}
	}
	// A plain draft.
	if _, err := quote(deals[3], 0, "5500000"); err != nil {
		return err
	}
	// Sent, revised to version 2 (version 1 superseded), version 2 sent.
	qe, err := quote(deals[4], 5, "8000000")
	if err != nil {
		return err
	}
	if _, err := a.Quotes.SetStatus(ctx, admin, qe.ID, domain.QuoteSent); err != nil {
		return err
	}
	v2, err := a.Quotes.Revise(ctx, admin, qe.ID)
	if err != nil {
		return err
	}
	_, err = a.Quotes.SetStatus(ctx, admin, v2.ID, domain.QuoteSent)
	return err
}

func showcaseContracts(ctx context.Context, pool *pgxpool.Pool, a Apps, admin app.Actor) error {
	quotes, err := ids(ctx, pool, `SELECT q.id FROM sales.quotations q JOIN sales.deals d ON d.id = q.deal_id
		WHERE q.id LIKE 'hist-qt-%' AND q.status = 'diterima' AND NOT EXISTS (SELECT 1 FROM sales.contracts c WHERE c.deal_id = d.id)
		ORDER BY d.closed_at DESC LIMIT 6`)
	if err != nil {
		return err
	}
	if len(quotes) < 6 {
		return fmt.Errorf("need 6 accepted quotations, have %d", len(quotes))
	}
	sig, err := signatureDataURL()
	if err != nil {
		return err
	}
	// target status per contract: draft, sent (live signing link), signed, active, ended, plus an NDA sent.
	plan := []struct{ template, status string }{
		{"ct-retainer", domain.ContractDraft}, {"ct-retainer", domain.ContractSent}, {"ct-project", domain.ContractSigned},
		{"ct-retainer", domain.ContractActive}, {"ct-retainer", domain.ContractEnded}, {"ct-nda", domain.ContractSent},
	}
	for i, p := range plan {
		ct, err := a.Contracts.Generate(ctx, admin, app.GenerateInput{QuotationID: quotes[i], TemplateID: p.template})
		if err != nil {
			return fmt.Errorf("generate %s: %w", p.template, err)
		}
		if p.status == domain.ContractDraft {
			continue
		}
		if ct, err = a.Contracts.Send(ctx, admin, ct.ID); err != nil {
			return err
		}
		if p.status == domain.ContractSent {
			continue
		}
		if _, err := a.Contracts.Sign(ctx, app.SignInput{Token: ct.SignToken, SignerName: "Direktur " + ct.ClientName, SignaturePNG: sig, Agree: true, IP: "203.0.113.20", UserAgent: "seed"}); err != nil {
			return err
		}
		if p.status == domain.ContractSigned {
			continue
		}
		if _, err := a.Contracts.SetStatus(ctx, admin, ct.ID, domain.ContractActive, ""); err != nil {
			return err
		}
		if p.status == domain.ContractEnded {
			if _, err := a.Contracts.SetStatus(ctx, admin, ct.ID, domain.ContractEnded, "Masa kontrak selesai, klien lanjut paket baru."); err != nil {
				return err
			}
		}
	}
	return nil
}

func showcaseOnboardings(ctx context.Context, pool *pgxpool.Pool, a Apps, admin app.Actor) error {
	deals, err := ids(ctx, pool, `SELECT d.id FROM sales.deals d WHERE d.id LIKE 'hist-deal-%' AND d.stage_id = 'st-menang'
		AND NOT EXISTS (SELECT 1 FROM sales.onboardings o WHERE o.deal_id = d.id) ORDER BY d.closed_at DESC LIMIT 3`)
	if err != nil {
		return err
	}
	for i, id := range deals {
		ob, err := a.Onboarding.Start(ctx, admin, id)
		if err != nil {
			return err
		}
		if i == 2 {
			continue // freshly started, nothing done yet
		}
		if ob, err = a.Onboarding.SubmitBrief(ctx, ob.PortalToken, "203.0.113.30", map[string]string{
			"brand_name": ob.ClientName, "brand_description": "Brand lokal yang tumbuh lewat penjualan online dan 3 cabang offline.",
			"goals": "Naikkan leads WhatsApp 2x dan awareness di Instagram.", "target_audience": "Perempuan 22-35 tahun, Jabodetabek, kelas menengah.",
			"competitors": "Dua brand lokal sejenis di Jakarta.", "brand_colors": "Hijau tua dan krem, gaya minimalis", "instagram": "@" + slug(ob.ClientName),
			"website": "https://" + slug(ob.ClientName) + ".id",
		}); err != nil {
			return err
		}
		for _, name := range []string{"logo.png", "brand-guideline.png"} {
			if ob, err = a.Onboarding.UploadAsset(ctx, ob.PortalToken, "203.0.113.30", name, swatchPNG(i)); err != nil {
				return err
			}
		}
		if i == 0 { // the first one is finished: every remaining item done
			for _, it := range ob.Items {
				if it.Status == domain.ItemTodo {
					if _, err := a.Onboarding.UpdateItem(ctx, admin, app.ItemUpdate{ID: it.ID, Status: domain.ItemDone}); err != nil {
						return err
					}
				}
			}
		} else if len(ob.Items) > 3 {
			if _, err := a.Onboarding.UpdateItem(ctx, admin, app.ItemUpdate{ID: ob.Items[len(ob.Items)-1].ID, Status: domain.ItemDone}); err != nil {
				return err
			}
		}
	}
	return nil
}

func showcaseDecks(ctx context.Context, pool *pgxpool.Pool, a Apps, admin app.Actor) error {
	deals, err := ids(ctx, pool, `SELECT id FROM sales.deals WHERE id LIKE 'hist-deal-%' AND stage_id IN ('st-proposal', 'st-negosiasi') ORDER BY created_at DESC LIMIT 2`)
	if err != nil {
		return err
	}
	leads, err := ids(ctx, pool, `SELECT id FROM sales.leads WHERE id LIKE 'hist-lead-%' AND status = 'dihubungi' ORDER BY created_at DESC LIMIT 1`)
	if err != nil {
		return err
	}
	for i, id := range deals {
		if _, err := a.Decks.Generate(ctx, admin, id, "", domain.DeckOptions{Tone: "profesional", IncludePricing: true, IncludeCaseStudies: true}); err != nil {
			return err
		}
		if i == 0 { // second version of the same deck
			if _, err := a.Decks.Generate(ctx, admin, id, "", domain.DeckOptions{Tone: "premium", IncludePricing: true, IncludeCaseStudies: true, ExtraNotes: "Tekankan hasil klien hospitality."}); err != nil {
				return err
			}
		}
	}
	for _, id := range leads {
		if _, err := a.Decks.Generate(ctx, admin, "", id, domain.DeckOptions{Tone: "santai", IncludeCaseStudies: true}); err != nil {
			return err
		}
	}
	return nil
}

func showcaseInsights(ctx context.Context, pool *pgxpool.Pool, a Apps) error {
	convs, err := ids(ctx, pool, `SELECT id FROM sales.lead_conversations WHERE id LIKE 'hist-conv-%' ORDER BY created_at DESC LIMIT 4`)
	if err != nil {
		return err
	}
	for _, id := range convs {
		if _, _, err := a.Chat.Analyze(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func slug(s string) string {
	var b bytes.Buffer
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + 32)
		}
	}
	return b.String()
}

// swatchPNG is a small solid image standing in for a client asset.
func swatchPNG(i int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	c := []color.NRGBA{{0x1b, 0x4d, 0xe4, 0xff}, {0x1b, 0xaf, 0x7a, 0xff}, {0xeb, 0x68, 0x34, 0xff}}[i%3]
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
