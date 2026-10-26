package seed

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/app"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
)

// Apps are the use cases the demo goes through, so totals, numbers,
// approvals and events follow the real rules.
type Apps struct {
	Store      domain.Store
	Leads      *app.Leads
	Pipeline   *app.Pipeline
	Quotes     *app.Quotes
	Billing    *app.Billing
	Contracts  *app.Contracts
	Chat       *app.Chat
	Decks      *app.Decks
	Onboarding *app.Onboarding
}

type demoLead struct {
	org, orgType, pic, title, phone, email, city, source, temp, status, notes string
	lines                                                                     []string
	stage                                                                     string // empty: no deal
	value                                                                     string
}

var demoLeads = []demoLead{
	{"Kopi Senja Nusantara", "umkm", "Rina Prameswari", "Owner", "081234500001", "rina@kopisenja.id", "Jakarta", "instagram", "panas", "qualified", "Ingin menaikkan penjualan online 3 cabang. Instagram sepi interaksi. Budget sekitar 10 juta per bulan.", []string{domain.LineSocialMedia, domain.LineAds}, "menang", "45000000"},
	{"Klinik Aurora Dermatologi", "corporate", "dr. Sinta Wibowo", "Direktur", "081234500002", "sinta@auroraderma.co.id", "Bandung", "referral", "panas", "qualified", "Website lambat dan tidak muncul di Google untuk layanan laser. Butuh SEO dan redesign.", []string{domain.LineSEO, domain.LineWebsite}, "negosiasi", "70000000"},
	{"Griya Asri Residence", "corporate", "Bambang Hartono", "Marketing Manager", "081234500003", "bambang@griyaasri.com", "Tangerang", "google", "hangat", "qualified", "Biaya per lead Google Ads tinggi. Perlu video tur unit dan optimasi iklan.", []string{domain.LineAds, domain.LineContent}, "proposal", "55000000"},
	{"Batik Kencana", "brand", "Ayu Lestari", "Brand Manager", "081234500004", "ayu@batikkencana.id", "Solo", "pameran", "hangat", "dihubungi", "Rebranding untuk pasar anak muda dan konten TikTok.", []string{domain.LineBranding, domain.LineSocialMedia}, "proposal", "38000000"},
	{"Startup Gizi Kita", "startup", "Fajar Nugroho", "CEO", "081234500005", "fajar@gizikita.app", "Jakarta", "linkedin", "hangat", "qualified", "Launching aplikasi bulan depan, butuh landing page dan iklan install.", []string{domain.LineWebsite, domain.LineAds}, "discovery-call", "30000000"},
	{"Hotel Samudra Bali", "corporate", "Made Suarta", "Sales & Marketing Director", "081234500006", "made@samudrabali.com", "Denpasar", "website", "panas", "qualified", "Okupansi musim sepi turun. Ingin kampanye direct booking.", []string{domain.LineAds, domain.LineContent, domain.LineSocialMedia}, "negosiasi", "90000000"},
	{"Toko Bangunan Sinar Jaya", "umkm", "Hendra Gunawan", "Pemilik", "081234500007", "", "Bekasi", "wa", "dingin", "baru", "Tanya harga website toko online.", []string{domain.LineWebsite}, "prospek-baru", "15000000"},
	{"Universitas Cendekia", "pendidikan", "Dr. Ratna Dewi", "Kepala Humas", "081234500008", "humas@cendekia.ac.id", "Yogyakarta", "referral", "hangat", "qualified", "Penerimaan mahasiswa baru, butuh kampanye iklan dan konten.", []string{domain.LineAds, domain.LineContent}, "kalah", "60000000"},
	{"Fashion Lokal Arunika", "brand", "Nadia Putri", "Founder", "081234500009", "nadia@arunika.co", "Jakarta", "instagram", "hangat", "dihubungi", "Ingin kolaborasi influencer dan pengelolaan Instagram.", []string{domain.LineSocialMedia}, "discovery-call", "25000000"},
	{"PT Logistik Prima", "corporate", "Andi Saputra", "Head of Marketing", "081234500010", "andi@logistikprima.co.id", "Surabaya", "google", "hangat", "qualified", "Perlu SEO B2B dan konten LinkedIn.", []string{domain.LineSEO, domain.LineSocialMedia}, "prospek-baru", "40000000"},
	{"Rumah Makan Padang Saiyo", "umkm", "Yusuf Rahman", "Pemilik", "081234500011", "", "Padang", "wa", "dingin", "baru", "Baru tanya-tanya soal foto menu.", nil, "", ""},
	{"Dinas Pariwisata Kota", "pemerintah", "Ir. Wahyu Setiadi", "Kabid Promosi", "081234500012", "promosi@pariwisata.go.id", "Malang", "pameran", "hangat", "dihubungi", "Promosi event festival tahunan.", nil, "", ""},
	{"Skincare Glowup", "brand", "Melati Sari", "Marketing Lead", "081234500013", "melati@glowup.id", "Jakarta", "chatbot", "panas", "baru", "Butuh ads management untuk peluncuran produk serum.", nil, "", ""},
	{"Bengkel Motor Jaya Abadi", "umkm", "Tono Sumarno", "Pemilik", "081234500014", "", "Depok", "form", "dingin", "baru", "Ingin muncul di Google Maps.", nil, "", ""},
	{"Koperasi Tani Makmur", "lainnya", "Siti Aminah", "Ketua", "081234500015", "", "Garut", "referral", "dingin", "tidak-cocok", "Budget belum tersedia tahun ini.", nil, "", ""},
	{"EduKids Learning Center", "pendidikan", "Lina Marlina", "Direktur", "081234500016", "lina@edukids.id", "Bogor", "website", "hangat", "baru", "Ingin website baru dan SEO lokal.", nil, "", ""},
	{"Gym Kuat Bersama", "umkm", "Rizky Pratama", "Owner", "081234500017", "rizky@kuatbersama.id", "Jakarta", "instagram", "hangat", "dihubungi", "Promosi membership tahun baru.", nil, "", ""},
	{"Apotek Sehat Selalu", "corporate", "Dewi Anggraini", "Manajer Pemasaran", "081234500018", "dewi@sehatselalu.co.id", "Semarang", "google", "dingin", "baru", "Riset dulu soal SEO.", nil, "", ""},
	{"Travel Umrah Barokah", "corporate", "H. Ahmad Fauzi", "Direktur", "081234500019", "ahmad@barokahtravel.id", "Jakarta", "wa", "hangat", "qualified", "Leads jamaah via iklan Meta.", nil, "", ""},
	{"Studio Yoga Prana", "umkm", "Kadek Ayu", "Founder", "081234500020", "kadek@yogaprana.id", "Ubud", "linkedin", "dingin", "baru", "Butuh branding ulang.", nil, "", ""},
}

// Demo loads leads, deals across every stage, quotations, an issued
// invoice with a payment, one signed contract and a chatbot conversation.
// It does nothing when leads already exist.
func Demo(ctx context.Context, a Apps, owners []string) error {
	if _, total, err := a.Store.ListLeads(ctx, domain.LeadFilter{Limit: 1}); err != nil || total > 0 {
		return err
	}
	if len(owners) == 0 {
		owners = []string{"demo-sales-1", "demo-sales-2", "demo-sales-3"}
	}
	pls, err := a.Store.ListPipelines(ctx)
	if err != nil {
		return err
	}
	stageID := map[string]string{}
	for _, p := range pls {
		for _, s := range p.Stages {
			stageID[s.Code] = s.ID
		}
	}
	admin := app.Actor{UserID: owners[0], Admin: true, Approver: true}
	for i, d := range demoLeads {
		owner := owners[i%len(owners)]
		actor := app.Actor{UserID: owner, Admin: true, Approver: true}
		lead, err := a.Leads.CreateLead(ctx, actor, app.LeadInput{OrgName: d.org, OrgType: d.orgType, PicName: d.pic, PicTitle: d.title, PicPhone: d.phone, PicEmail: d.email, City: d.city, Source: d.source, Temperature: d.temp, Status: d.status, Notes: d.notes, OwnerUserID: owner})
		if err != nil {
			return fmt.Errorf("lead %s: %w", d.org, err)
		}
		if d.stage == "" {
			continue
		}
		acc, err := a.Leads.CreateAccount(ctx, actor, app.AccountInput{Name: d.org, AccountType: d.orgType, City: d.city, Phone: d.phone, Email: d.email, Address: "Jl. Contoh No. " + fmt.Sprint(i+10) + ", " + d.city, OwnerUserID: owner})
		if err != nil {
			return err
		}
		deal, err := a.Pipeline.CreateDeal(ctx, actor, app.DealInput{LeadID: lead.ID, AccountID: acc.ID, Title: dealTitle(d), ServiceLines: d.lines, ValueEstimate: d.value, OwnerUserID: owner, Notes: d.notes})
		if err != nil {
			return fmt.Errorf("deal %s: %w", d.org, err)
		}
		if err := demoAdvance(ctx, a, admin, deal.ID, d, stageID); err != nil {
			return fmt.Errorf("deal %s: %w", d.org, err)
		}
	}
	return demoChat(ctx, a)
}

func dealTitle(d demoLead) string {
	labels := make([]string, len(d.lines))
	for i, l := range d.lines {
		labels[i] = domain.ServiceLineLabel(l)
	}
	return strings.Join(labels, " + ") + " " + d.org
}

func demoAdvance(ctx context.Context, a Apps, actor app.Actor, dealID string, d demoLead, stageID map[string]string) error {
	order := []string{"prospek-baru", "discovery-call", "proposal", "negosiasi", "menang"}
	target := d.stage
	if target == "kalah" {
		target = "proposal"
	}
	var quote *domain.Quotation
	for _, code := range order[1:] {
		if indexOf(order, code) > indexOf(order, target) {
			break
		}
		if code == "menang" {
			if quote != nil {
				if _, err := a.Quotes.SetStatus(ctx, actor, quote.ID, domain.QuoteAccepted); err != nil {
					return err
				}
			}
			if _, err := a.Pipeline.MoveDealStage(ctx, actor, dealID, stageID[code], d.value, ""); err != nil {
				return err
			}
			continue
		}
		if _, err := a.Pipeline.MoveDealStage(ctx, actor, dealID, stageID[code], "", ""); err != nil {
			return err
		}
		if code == "proposal" {
			res, err := a.Quotes.GenerateDraft(ctx, actor, dealID, app.DraftOptions{Tier: "standard", RetainerMonths: 6})
			if err != nil {
				return err
			}
			quote = res.Quotation
			if _, err := a.Quotes.SetStatus(ctx, actor, quote.ID, domain.QuoteSent); err != nil {
				return err
			}
		}
	}
	if d.stage == "kalah" {
		if quote != nil {
			if _, err := a.Quotes.SetStatus(ctx, actor, quote.ID, domain.QuoteRejected); err != nil {
				return err
			}
		}
		_, err := a.Pipeline.MoveDealStage(ctx, actor, dealID, stageID["kalah"], "", "lr-kompetitor")
		return err
	}
	if d.stage == "menang" && quote != nil {
		return demoWon(ctx, a, actor, dealID, quote.ID)
	}
	return nil
}

// demoWon bills the first term, records its payment and signs a contract.
func demoWon(ctx context.Context, a Apps, actor app.Actor, dealID, quoteID string) error {
	invs, err := a.Billing.SubmitFromTerms(ctx, actor, quoteID)
	if err != nil {
		return err
	}
	if len(invs) > 0 {
		first := invs[len(invs)-1]
		if _, err := a.Billing.IssueInvoice(ctx, actor, first.ID); err != nil {
			return err
		}
		if _, err := a.Billing.RecordPayment(ctx, actor, app.PaymentInput{DealID: dealID, InvoiceID: first.ID, Amount: first.Amount.String(), Method: "transfer", Note: "BCA transfer termin 1"}); err != nil {
			return err
		}
	}
	ct, err := a.Contracts.Generate(ctx, actor, app.GenerateInput{QuotationID: quoteID, TemplateID: "ct-retainer"})
	if err != nil {
		return err
	}
	if ct, err = a.Contracts.Send(ctx, actor, ct.ID); err != nil {
		return err
	}
	sig, err := signatureDataURL()
	if err != nil {
		return err
	}
	if _, err := a.Contracts.Sign(ctx, app.SignInput{Token: ct.SignToken, SignerName: "Rina Prameswari", SignaturePNG: sig, Agree: true, IP: "203.0.113.10", UserAgent: "seed"}); err != nil {
		return err
	}
	_, err = a.Contracts.SetStatus(ctx, actor, ct.ID, domain.ContractActive, "")
	return err
}

// signatureDataURL draws a small squiggle as a PNG data URL.
func signatureDataURL() (string, error) {
	img := image.NewNRGBA(image.Rect(0, 0, 240, 80))
	ink := color.NRGBA{R: 0x11, G: 0x22, B: 0x55, A: 0xff}
	for x := 10; x < 230; x++ {
		y := 40 + int(18*float64((x%60)-30)/30)*((x/60)%2*2-1)
		for dy := -1; dy <= 1; dy++ {
			img.Set(x, y+dy, ink)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func demoChat(ctx context.Context, a Apps) error {
	chat, err := a.Chat.StartPublic(ctx, "seed", app.StartInput{Name: "Melati", PageURL: "https://white.id/layanan/ads"})
	if err != nil {
		return err
	}
	for _, msg := range []string{
		"Halo, saya mau tanya soal iklan Meta untuk produk skincare",
		"Nama saya Melati dari brand Glowup, budget sekitar 15 juta, mulai bulan depan",
		"Email saya melati@glowup.id",
	} {
		if _, err := a.Chat.SendPublic(ctx, "seed", chat.Token, msg); err != nil {
			return err
		}
	}
	return nil
}

func indexOf(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return -1
}
