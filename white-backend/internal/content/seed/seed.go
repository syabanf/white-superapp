// Package seed fills the content schema with demo data for WHITE's
// marketing clients: an approved editorial plan for last month, a draft
// plan for next month with items in every workflow state, caption
// revisions, design tasks with files, a brand asset library and one
// content report. Clients that already have last month's plan are skipped,
// so running it twice changes nothing.
package seed

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/syabanf/white-superapp/white-backend/internal/content/app"
	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/providers/mockrand"
)

type Client struct {
	ID   string
	Name string
}

// Seeder needs the store, the file store and an AI for the report summary.
type Seeder struct {
	Store domain.Store
	Files domain.Files
	AI    app.AI
	Now   time.Time
}

type Result struct {
	Clients, Plans, Items, Assets, Reports int
	// HistoryMonths counts months added by the history pass.
	HistoryMonths int
	Skipped       []string
}

// theme is one client's content angle.
type theme struct {
	// keywords in the client name that select this theme.
	keywords []string
	pillars  []string
	color    color.RGBA
	topics   []string
	tone     string
	goals    string
}

var themes = []theme{
	{
		keywords: []string{"kopi", "coffee", "cafe", "kafe", "resto", "food", "bakery"},
		pillars:  []string{"Menu andalan", "Promo", "Behind the bar", "Tips ngopi"},
		color:    color.RGBA{0x6b, 0x3e, 0x26, 0xff},
		topics:   []string{"kopi susu gula aren", "manual brew V60", "paket sarapan hemat", "barista kami", "biji kopi Gayo", "promo payday", "cold brew botolan", "cara simpan biji kopi", "menu musim hujan", "kopi decaf", "loyalty card", "live music Jumat"},
		tone:     "hangat, santai, akrab seperti teman ngopi",
		goals:    "Naikkan kunjungan outlet 15% dan perkenalkan menu musiman",
	},
	{
		keywords: []string{"batik", "fashion", "butik", "boutique", "wear", "apparel"},
		pillars:  []string{"Koleksi baru", "Styling", "Promo", "Cerita brand"},
		color:    color.RGBA{0x1f, 0x3a, 0x5f, 0xff},
		topics:   []string{"koleksi batik modern", "mix and match kantor", "flash sale 11.11", "proses jahit artisan", "outfit kondangan", "kain tenun NTT", "panduan ukuran", "lookbook akhir tahun", "bahan linen adem", "gratis ongkir", "testimoni pelanggan", "capsule wardrobe"},
		tone:     "elegan, percaya diri, tetap membumi",
		goals:    "Dorong penjualan online lewat Instagram Shop dan kenalkan koleksi baru",
	},
	{
		keywords: []string{"skin", "beauty", "kosmetik", "glow", "klinik", "clinic"},
		pillars:  []string{"Edukasi skincare", "Produk", "Testimoni", "Promo"},
		color:    color.RGBA{0xc0, 0x6c, 0x84, 0xff},
		topics:   []string{"urutan skincare pagi", "serum niacinamide", "mitos sunscreen", "kulit berminyak", "bundling hemat", "review pelanggan", "double cleansing", "bahan aktif aman bumil", "barrier kulit", "giveaway", "skincare untuk pria", "routine malam"},
		tone:     "ramah, informatif, berbasis fakta dermatologi",
		goals:    "Bangun kepercayaan lewat edukasi dan naikkan konversi bundling",
	},
	{
		keywords: []string{"property", "properti", "residence", "realty", "land", "estate", "rumah"},
		pillars:  []string{"Unit unggulan", "Edukasi KPR", "Progres pembangunan", "Lifestyle kawasan"},
		color:    color.RGBA{0x2e, 0x4a, 0x3a, 0xff},
		topics:   []string{"tipe 45 siap huni", "simulasi cicilan KPR", "progres tower B", "akses tol 5 menit", "promo DP 0%", "fasilitas clubhouse", "tips memilih rumah pertama", "virtual tour show unit", "legalitas SHM", "open house akhir pekan", "testimoni penghuni", "investasi properti 2027"},
		tone:     "meyakinkan, informatif, dekat dengan keluarga muda",
		goals:    "Kumpulkan leads site visit dan jelaskan skema KPR dengan sederhana",
	},
	{
		keywords: []string{"villa", "hotel", "resort", "travel", "bali", "escape", "trip", "tour"},
		pillars:  []string{"Pengalaman menginap", "Destinasi", "Promo musiman", "Cerita tamu"},
		color:    color.RGBA{0x0f, 0x6e, 0x6e, 0xff},
		topics:   []string{"private pool villa", "sunrise di Ubud", "paket honeymoon", "sarapan floating", "itinerary 3 hari", "early bird Desember", "spa tradisional Bali", "ulasan tamu dari Jakarta", "workation dengan wifi kencang", "pantai tersembunyi", "staycation keluarga", "upacara adat terdekat"},
		tone:     "hangat, inspiratif, menggoda untuk berlibur",
		goals:    "Naikkan direct booking lewat Instagram dan kurangi ketergantungan OTA",
	},
}

// themeFor picks the theme whose keyword appears in the client name, or
// rotates by position.
func themeFor(name string, i int) theme {
	lower := strings.ToLower(name)
	for _, t := range themes {
		for _, k := range t.keywords {
			if strings.Contains(lower, k) {
				return t
			}
		}
	}
	return themes[i%len(themes)]
}

var hooks = []string{
	"Pernah ngerasa begini juga?",
	"Simpan dulu, pasti kepake.",
	"Ini yang jarang orang tahu.",
	"Stop scroll, ada kabar baik.",
	"Tiga detik untuk berubah pikiran.",
}

var ctas = []string{
	"Ceritakan pengalamanmu di kolom komentar!",
	"Klik link di bio untuk pesan sekarang.",
	"Tag teman yang wajib tahu soal ini.",
	"Simpan postingan ini untuk nanti.",
}

// Run seeds up to three clients. users are identity user ids to assign
// (may be empty).
func (s *Seeder) Run(ctx context.Context, clients []Client, users []string) (*Result, error) {
	res := &Result{}
	if len(clients) > 3 {
		clients = clients[:3]
	}
	last, next := domain.MonthOf(s.Now).Prev(), domain.MonthOf(s.Now).Next()
	for i, c := range clients {
		existing, err := s.Store.GetPlanByMonth(ctx, c.ID, last)
		if err != nil {
			return res, err
		}
		cs := clientSeed{Seeder: s, client: c, theme: themeFor(c.Name, i), users: users, offset: i}
		if existing != nil {
			res.Skipped = append(res.Skipped, c.Name)
		} else {
			if err := cs.run(ctx, last, next, res); err != nil {
				return res, fmt.Errorf("%s: %w", c.Name, err)
			}
			res.Clients++
		}
		if err := cs.history(ctx, res); err != nil {
			return res, fmt.Errorf("%s: %w", c.Name, err)
		}
	}
	return res, nil
}

type clientSeed struct {
	*Seeder
	client Client
	theme  theme
	users  []string
	offset int
	images map[string][]byte
}

func (c *clientSeed) user(n int) *string {
	if len(c.users) == 0 {
		return nil
	}
	return &c.users[(n+c.offset)%len(c.users)]
}

func (c *clientSeed) copywriter() *string { return c.user(0) }
func (c *clientSeed) designer() *string   { return c.user(1) }
func (c *clientSeed) lead() *string       { return c.user(2) }

func at(day time.Time, hour int) *time.Time {
	t := time.Date(day.Year(), day.Month(), day.Day(), hour-7, 0, 0, 0, time.UTC) // WIB to UTC
	return &t
}

// image returns a small PNG in the brand colour (cached per variant).
func (c *clientSeed) image(variant int, w, h int) []byte {
	key := fmt.Sprintf("%d-%dx%d", variant, w, h)
	if b, ok := c.images[key]; ok {
		return b
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	base := c.theme.color
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			col := base
			if (y/(h/6)+variant)%2 == 0 {
				col = color.RGBA{uint8(min(255, int(base.R)+40)), uint8(min(255, int(base.G)+40)), uint8(min(255, int(base.B)+40)), 0xff}
			}
			if x > w/8 && x < w*7/8 && y > h*3/8 && y < h*5/8 {
				col = color.RGBA{0xfa, 0xf7, 0xf2, 0xff}
			}
			img.SetRGBA(x, y, col)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	if c.images == nil {
		c.images = map[string][]byte{}
	}
	c.images[key] = buf.Bytes()
	return buf.Bytes()
}

func (c *clientSeed) put(ext string, data []byte) (*domain.StoredFile, error) {
	return c.Files.Put(context.Background(), c.client.ID, ext, data)
}

// itemSpec describes one seeded item.
type itemSpec struct {
	dayOfMonth int
	platform   domain.Platform
	format     domain.Format
	status     domain.ItemStatus
	rounds     int32
	late       bool
	// slip is how many days late the work ran (late alone means 2).
	slip int
	// hour is the WIB publish hour (0 picks one by position).
	hour int
}

var lastMonthSpecs = []itemSpec{
	{2, domain.PlatformInstagram, domain.FormatCarousel, domain.ItemPublished, 1, false, 0, 0},
	{4, domain.PlatformInstagram, domain.FormatReel, domain.ItemPublished, 0, false, 0, 0},
	{7, domain.PlatformFacebook, domain.FormatFeed, domain.ItemPublished, 0, false, 0, 0},
	{9, domain.PlatformInstagram, domain.FormatStory, domain.ItemPublished, 0, false, 0, 0},
	{11, domain.PlatformInstagram, domain.FormatFeed, domain.ItemPublished, 2, true, 0, 0},
	{14, domain.PlatformTikTok, domain.FormatReel, domain.ItemPublished, 1, false, 0, 0},
	{16, domain.PlatformInstagram, domain.FormatCarousel, domain.ItemPublished, 0, false, 0, 0},
	{19, domain.PlatformFacebook, domain.FormatReel, domain.ItemPublished, 1, true, 0, 0},
	{22, domain.PlatformInstagram, domain.FormatReel, domain.ItemFailed, 0, false, 0, 0},
	{25, domain.PlatformInstagram, domain.FormatFeed, domain.ItemPublished, 0, false, 0, 0},
	{28, domain.PlatformInstagram, domain.FormatStory, domain.ItemApproved, 1, true, 0, 0},
}

var nextMonthSpecs = []itemSpec{
	{2, domain.PlatformInstagram, domain.FormatCarousel, domain.ItemScheduled, 1, false, 0, 0},
	{4, domain.PlatformInstagram, domain.FormatReel, domain.ItemApproved, 0, false, 0, 0},
	{6, domain.PlatformFacebook, domain.FormatFeed, domain.ItemReview, 0, false, 0, 0},
	{9, domain.PlatformInstagram, domain.FormatFeed, domain.ItemRevision, 1, false, 0, 0},
	{11, domain.PlatformInstagram, domain.FormatStory, domain.ItemDesign, 0, false, 0, 0},
	{13, domain.PlatformTikTok, domain.FormatReel, domain.ItemDesign, 0, false, 0, 0},
	{16, domain.PlatformInstagram, domain.FormatCarousel, domain.ItemCopywriting, 0, false, 0, 0},
	{18, domain.PlatformFacebook, domain.FormatReel, domain.ItemCopywriting, 0, false, 0, 0},
	{21, domain.PlatformInstagram, domain.FormatFeed, domain.ItemIdea, 0, false, 0, 0},
	{24, domain.PlatformInstagram, domain.FormatReel, domain.ItemIdea, 0, false, 0, 0},
	{27, domain.PlatformInstagram, domain.FormatStory, domain.ItemIdea, 0, false, 0, 0},
}

func (c *clientSeed) brief() domain.Brief {
	return domain.Brief{
		Goals: c.theme.goals, Pillars: c.theme.pillars, Tone: c.theme.tone,
		Notes:     "Hindari klaim berlebihan. Selalu cantumkan logo di pojok kanan bawah.",
		Frequency: []domain.Frequency{{Platform: domain.PlatformInstagram, PerWeek: 3}, {Platform: domain.PlatformFacebook, PerWeek: 1}, {Platform: domain.PlatformTikTok, PerWeek: 1}},
	}
}

func (c *clientSeed) run(ctx context.Context, last, next domain.Month, res *Result) error {
	lastPlan := domain.Plan{ID: ids.New(), ClientID: c.client.ID, Month: last, Title: "EP " + last.Label(), Status: domain.PlanApproved, Brief: c.brief(), CreatedBy: c.copywriter()}
	nextPlan := domain.Plan{ID: ids.New(), ClientID: c.client.ID, Month: next, Title: "EP " + next.Label(), Status: domain.PlanDraft, Brief: c.brief(), CreatedBy: c.copywriter()}
	var reportItems []domain.Item
	err := c.Store.WithTx(ctx, func(tx domain.Store) error {
		for _, p := range []*domain.Plan{&lastPlan, &nextPlan} {
			if err := tx.InsertPlan(ctx, *p); err != nil {
				return err
			}
			res.Plans++
		}
		submitted, approved := at(last.AddDate(0, 0, -6), 10), at(last.AddDate(0, 0, -4), 15)
		model := "mock"
		lastPlan.AIModel, lastPlan.GeneratedAt = &model, at(last.AddDate(0, 0, -8), 9)
		lastPlan.SubmittedBy, lastPlan.SubmittedAt, lastPlan.ApprovedBy, lastPlan.ApprovedAt = c.copywriter(), submitted, c.lead(), approved
		lastPlan.ReviewNote = strPtr("Sudah sesuai brief, lanjut produksi.")
		if err := tx.UpdatePlan(ctx, lastPlan); err != nil {
			return err
		}
		nextPlan.AIModel, nextPlan.GeneratedAt = &model, at(next.AddDate(0, 0, -12), 9)
		if err := tx.UpdatePlan(ctx, nextPlan); err != nil {
			return err
		}
		for _, comment := range []struct {
			plan, body string
			user       *string
		}{
			{lastPlan.ID, "Tolong tambah satu konten promo payday di minggu terakhir.", c.lead()},
			{lastPlan.ID, "Sudah ditambahkan di tanggal 25, mohon dicek.", c.copywriter()},
			{nextPlan.ID, "Draft awal dari AI sudah saya rapikan, pilar promo masih kurang.", c.copywriter()},
		} {
			if err := tx.InsertPlanComment(ctx, domain.PlanComment{ID: ids.New(), PlanID: comment.plan, UserID: comment.user, Body: comment.body}); err != nil {
				return err
			}
		}
		for n, spec := range lastMonthSpecs {
			it, err := c.item(ctx, tx, lastPlan, spec, n)
			if err != nil {
				return err
			}
			reportItems = append(reportItems, *it)
			res.Items++
		}
		for n, spec := range nextMonthSpecs {
			if _, err := c.item(ctx, tx, nextPlan, spec, n+len(lastMonthSpecs)); err != nil {
				return err
			}
			res.Items++
		}
		return nil
	})
	if err != nil {
		return err
	}
	n, err := c.assets(ctx)
	if err != nil {
		return err
	}
	res.Assets += n
	if err := c.report(ctx, last, reportItems); err != nil {
		return err
	}
	res.Reports++
	return nil
}

func strPtr(s string) *string { return &s }

// before keeps a seeded instant in the past: at most now minus back.
func (c *clientSeed) before(t time.Time, back time.Duration) time.Time {
	if limit := c.Now.Add(-back); t.After(limit) {
		return limit
	}
	return t
}

func (c *clientSeed) beforePtr(t *time.Time, back time.Duration) *time.Time {
	v := c.before(*t, back)
	return &v
}

// seedBriefTail ends every seeded visual brief; the repair pass uses it to
// recognise seeded items.
const seedBriefTail = "Warna brand, foto produk close-up, teks besar maksimal 6 kata."

var publishHours = []int{19, 12, 9, 20, 17}

// item writes one item with its caption history, review trail and design
// work, every row dated along the item's timeline.
func (c *clientSeed) item(ctx context.Context, tx domain.Store, plan domain.Plan, spec itemSpec, n int) (*domain.Item, error) {
	r := mockrand.New(fmt.Sprintf("%s|%d", c.client.ID, n))
	topic := c.theme.topics[n%len(c.theme.topics)]
	pillar := c.theme.pillars[n%len(c.theme.pillars)]
	planned := time.Date(plan.Month.Year(), plan.Month.Month(), spec.dayOfMonth, 0, 0, 0, 0, time.UTC)
	title := fmt.Sprintf("%s: %s", pillar, topic)
	hook := mockrand.Pick(r, hooks)
	brand := hashtag(c.client.Name)
	tags := []string{"#" + brand, "#" + hashtag(pillar), "#" + hashtag(topic)}
	caption := func(v int) string {
		base := fmt.Sprintf("%s\n\nKali ini %s bahas %s. %s", hook, c.client.Name, topic, mockrand.Pick(r, ctas))
		if v > 1 {
			base += fmt.Sprintf("\n\nBerlaku sampai %s.", planned.AddDate(0, 0, 7).Format("02/01"))
		}
		return base
	}
	hour := spec.hour
	if hour == 0 {
		hour = publishHours[n%len(publishHours)]
	}
	slip := spec.slip
	if spec.late && slip == 0 {
		slip = 2
	}
	copyDue, designDue, reviewDue := planned.AddDate(0, 0, -4), planned.AddDate(0, 0, -2), planned.AddDate(0, 0, -1)
	created := c.before(*at(planned.AddDate(0, 0, -12), 9), 72*time.Hour)
	it := domain.Item{
		ID: ids.New(), PlanID: plan.ID, ClientID: c.client.ID, PlannedDate: planned, PublishAt: at(planned, hour),
		Platform: spec.platform, Format: spec.format, Pillar: pillar, Title: title, Hook: hook, Caption: caption(1), Hashtags: tags,
		VisualBrief:  fmt.Sprintf("%s %s bertema %s. %s", domain.FormatLabel(spec.format), domain.PlatformLabel(spec.platform), topic, seedBriefTail),
		CopywriterID: c.copywriter(), DesignerID: c.designer(), CopyDue: &copyDue, DesignDue: &designDue, ReviewDue: &reviewDue,
		Status: domain.ItemIdea, CaptionVersion: 1, CreatedBy: c.copywriter(), CreatedAt: created,
	}
	if spec.format == domain.FormatStory {
		it.Caption = ""
	}
	if err := tx.InsertItem(ctx, it); err != nil {
		return nil, err
	}
	activity := func(typ string, user *string, from, to domain.ItemStatus, note string, when time.Time) error {
		f, t := string(from), string(to)
		a := domain.ItemActivity{ID: ids.New(), ItemID: it.ID, UserID: user, Type: typ, ToStatus: &t, Note: optional(note), CreatedAt: c.before(when, time.Hour)}
		if from != "" {
			a.FromStatus = &f
		}
		return tx.InsertActivity(ctx, a)
	}
	if err := tx.InsertRevision(ctx, domain.CaptionRevision{ID: ids.New(), ItemID: it.ID, Version: 1, Caption: it.Caption, Hook: hook, Hashtags: tags, Note: "Draft awal", CreatedBy: c.copywriter(), CreatedAt: created}); err != nil {
		return nil, err
	}
	if err := activity(domain.ActCreated, c.copywriter(), "", domain.ItemIdea, "", created); err != nil {
		return nil, err
	}

	order := []domain.ItemStatus{domain.ItemIdea, domain.ItemCopywriting, domain.ItemDesign, domain.ItemReview, domain.ItemApproved, domain.ItemScheduled, domain.ItemPublished}
	rank := map[domain.ItemStatus]int{}
	for i, s := range order {
		rank[s] = i
	}
	target := spec.status
	switch target {
	case domain.ItemRevision:
		target = domain.ItemReview
	case domain.ItemFailed:
		target = domain.ItemScheduled
	}
	prev, when := domain.ItemIdea, created
	for _, s := range order[1 : rank[target]+1] {
		switch s {
		case domain.ItemCopywriting:
			when = created.Add(2 * time.Hour)
		case domain.ItemDesign:
			it.CopyDoneAt = c.beforePtr(at(copyDue.AddDate(0, 0, slip-1), 16), 30*time.Hour)
			when = *it.CopyDoneAt
		case domain.ItemReview:
			// Designs take one to four days after copy, varying per item.
			it.DesignDoneAt = c.beforePtr(at(designDue.AddDate(0, 0, slip+r.IntN(4)-2), 10+r.IntN(8)), 6*time.Hour)
			when = *it.DesignDoneAt
		case domain.ItemApproved:
			it.ApprovedAt, it.ApprovedBy = c.beforePtr(at(reviewDue.AddDate(0, 0, slip/2+r.IntN(2)), 9+r.IntN(8)), 2*time.Hour), c.lead()
			if it.DesignDoneAt != nil && it.ApprovedAt.Before(*it.DesignDoneAt) {
				approved := c.before(it.DesignDoneAt.Add(4*time.Hour), 2*time.Hour)
				it.ApprovedAt = &approved
			}
			when = *it.ApprovedAt
		case domain.ItemScheduled:
			it.ScheduledBy = c.lead()
			when = it.ApprovedAt.Add(3 * time.Hour)
		}
		if err := activity(domain.ActMoved, c.copywriter(), prev, s, "", when); err != nil {
			return nil, err
		}
		prev = s
	}
	// Revision rounds after design: changes requested, new caption version, back to review.
	roundAt := func(round int32, h int) time.Time {
		base := created
		if it.DesignDoneAt != nil {
			base = *it.DesignDoneAt
		}
		return c.before(base.Add(time.Duration(int(round)*5+h)*time.Hour), time.Hour)
	}
	for round := int32(1); round <= spec.rounds; round++ {
		note := []string{"CTA kurang jelas, tolong pertegas promonya.", "Hook terlalu panjang untuk tiga detik pertama.", "Hashtag brand belum ada di caption."}[(int(round)+n)%3]
		if err := tx.InsertItemComment(ctx, domain.ItemComment{ID: ids.New(), ItemID: it.ID, UserID: c.lead(), Body: note, Kind: domain.DecisionChangesRequested, Round: round, CreatedAt: roundAt(round, 0)}); err != nil {
			return nil, err
		}
		if err := activity(domain.ActMoved, c.lead(), domain.ItemReview, domain.ItemRevision, note, roundAt(round, 0)); err != nil {
			return nil, err
		}
		it.RevisionRound = round
		if spec.status == domain.ItemRevision && round == spec.rounds {
			break
		}
		it.CaptionVersion++
		if spec.format != domain.FormatStory {
			it.Caption = caption(int(it.CaptionVersion))
		}
		if err := tx.InsertRevision(ctx, domain.CaptionRevision{ID: ids.New(), ItemID: it.ID, Version: it.CaptionVersion, Caption: it.Caption, Hook: hook, Hashtags: tags, Note: "Revisi: " + note, CreatedBy: c.copywriter(), CreatedAt: roundAt(round, 2)}); err != nil {
			return nil, err
		}
		if err := activity(domain.ActRevised, c.copywriter(), domain.ItemRevision, domain.ItemReview, "", roundAt(round, 2)); err != nil {
			return nil, err
		}
	}
	it.Status = spec.status
	last := when
	switch spec.status {
	case domain.ItemPublished:
		ext := fmt.Sprintf("seed_%s_%d", c.client.ID, n)
		it.PublishedAt, it.ExternalIDs = it.PublishAt, []string{ext}
		if slip >= 2 {
			it.PublishedAt = at(planned.AddDate(0, 0, 1), 19)
		}
		it.PublishedAt = c.beforePtr(it.PublishedAt, time.Hour)
		it.Permalink = strPtr("https://www.instagram.com/p/" + ext + "/")
		last = *it.PublishedAt
		if err := activity(domain.ActPublished, nil, domain.ItemScheduled, domain.ItemPublished, "", last); err != nil {
			return nil, err
		}
	case domain.ItemFailed:
		it.PublishError = strPtr("Token akses Instagram kedaluwarsa. Hubungkan ulang akun lalu jadwalkan lagi.")
		last = *it.PublishAt
		if err := activity(domain.ActFailed, nil, domain.ItemScheduled, domain.ItemFailed, *it.PublishError, last); err != nil {
			return nil, err
		}
	}
	if err := tx.UpdateItem(ctx, it); err != nil {
		return nil, err
	}
	if err := tx.BackdateItem(ctx, it.ID, it.CreatedAt, c.before(last, time.Hour)); err != nil {
		return nil, err
	}
	if rank[target] >= rank[domain.ItemDesign] {
		if err := c.design(ctx, tx, it, spec); err != nil {
			return nil, err
		}
	}
	return &it, nil
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// design adds the item's design task with files and feedback matching its
// stage, dated between copy done and design done.
func (c *clientSeed) design(ctx context.Context, tx domain.Store, it domain.Item, spec itemSpec) error {
	preset := domain.DefaultSizePreset(it.Format)
	start := it.CreatedAt
	if it.CopyDoneAt != nil {
		start = *it.CopyDoneAt
	}
	task := domain.DesignTask{
		ID: ids.New(), ItemID: it.ID, ClientID: c.client.ID, Title: "Desain: " + it.Title, Brief: it.VisualBrief,
		References: []string{"https://www.pinterest.com/search/pins/?q=" + hashtag(it.Pillar)}, SizePreset: preset.Key,
		Width: preset.Width, Height: preset.Height, DesignerID: c.designer(), DueDate: it.DesignDue, Status: domain.DesignInProgress, CreatedBy: c.lead(),
		CreatedAt: start,
	}
	if err := tx.InsertDesignTask(ctx, task); err != nil {
		return err
	}
	versions := 1
	switch spec.status {
	case domain.ItemDesign:
		// A first draft is up; nobody has reviewed it yet.
	case domain.ItemReview:
		task.Status = domain.DesignInReview
	case domain.ItemRevision:
		task.Status, versions = domain.DesignChangesRequested, 1
	default:
		task.Status = domain.DesignApproved
		versions = 1 + int(spec.rounds)
	}
	end := start.Add(20 * time.Hour)
	if it.DesignDoneAt != nil && it.DesignDoneAt.After(start) {
		end = *it.DesignDoneAt
	}
	step := end.Sub(start) / time.Duration(versions+1)
	slides := 1
	if it.Format == domain.FormatCarousel {
		slides = 3
	}
	w, h := int(preset.Width/4), int(preset.Height/4)
	for v := 1; v <= versions; v++ {
		uploaded := c.before(start.Add(step*time.Duration(v)), 2*time.Hour)
		for s := 0; s < slides; s++ {
			obj, err := c.put("png", c.image(v+s, w, h))
			if err != nil {
				return err
			}
			name := fmt.Sprintf("%s-v%d-%d.png", hashtag(it.Pillar), v, s+1)
			if err := tx.InsertDesignFile(ctx, domain.DesignFile{ID: ids.New(), TaskID: task.ID, Version: int32(v), SortOrder: int32(s), FileKey: obj.Key, FileName: name, MimeType: "image/png", SizeBytes: obj.SizeBytes, UploadedBy: c.designer(), CreatedAt: uploaded}); err != nil {
				return err
			}
		}
		if v < versions || task.Status == domain.DesignChangesRequested {
			if err := tx.InsertDesignFeedback(ctx, domain.DesignFeedback{ID: ids.New(), TaskID: task.ID, Version: int32(v), UserID: c.lead(), Decision: domain.DecisionChangesRequested, Body: "Logo kurang terlihat, perbesar 20% dan naikkan kontras teks.", CreatedAt: c.before(uploaded.Add(step/2), time.Hour)}); err != nil {
				return err
			}
		}
	}
	if task.Status == domain.DesignApproved {
		if err := tx.InsertDesignFeedback(ctx, domain.DesignFeedback{ID: ids.New(), TaskID: task.ID, Version: int32(versions), UserID: c.lead(), Decision: domain.DecisionApproved, Body: "Mantap, siap tayang.", CreatedAt: c.before(end, time.Hour)}); err != nil {
			return err
		}
	}
	task.CurrentVersion = int32(versions)
	return tx.UpdateDesignTask(ctx, task)
}

// assets builds the brand library once per client.
func (c *clientSeed) assets(ctx context.Context) (int, error) {
	folders, err := c.Store.ListFolders(ctx, c.client.ID)
	if err != nil || len(folders) > 0 {
		return 0, err
	}
	folderIDs := map[string]string{}
	for _, name := range []string{"Logo", "Brand Guideline", "Font", "Foto Produk", "Template"} {
		f := domain.AssetFolder{ID: ids.New(), ClientID: c.client.ID, Name: name, CreatedBy: c.lead()}
		if err := c.Store.InsertFolder(ctx, f); err != nil {
			return 0, err
		}
		folderIDs[name] = f.ID
	}
	guide, err := guidelinePDF(c.client.Name)
	if err != nil {
		return 0, err
	}
	font, err := fontZip()
	if err != nil {
		return 0, err
	}
	type file struct {
		name, ext, note string
		data            []byte
	}
	specs := []struct {
		folder, name, desc string
		kind               domain.AssetKind
		tags               []string
		files              []file
	}{
		{"Logo", "Logo utama", "Logo full colour untuk latar terang.", domain.AssetLogo, []string{"logo", "primary"}, []file{{"logo-2025.png", "png", "Versi lama", c.image(1, 256, 256)}, {"logo-2026.png", "png", "Refresh warna 2026", c.image(2, 256, 256)}}},
		{"Logo", "Logo putih", "Untuk latar gelap dan foto.", domain.AssetLogo, []string{"logo", "monokrom"}, []file{{"logo-putih.png", "png", "", c.image(3, 256, 256)}}},
		{"Brand Guideline", "Brand guideline 2026", "Warna, tipografi, tone of voice dan do/don't.", domain.AssetBrandGuideline, []string{"guideline", "warna", "tipografi"}, []file{{"brand-guideline-2026.pdf", "pdf", "", guide}}},
		{"Font", "Font brand", "Font utama untuk headline dan body.", domain.AssetFont, []string{"font", "tipografi"}, []file{{"font-brand.zip", "zip", "", font}}},
		{"Foto Produk", "Foto produk unggulan", "Foto studio latar polos, siap dipakai feed.", domain.AssetPhoto, []string{"foto", "produk"}, []file{{"produk-unggulan.png", "png", "", c.image(4, 540, 675)}}},
		{"Foto Produk", "Foto suasana outlet", "Foto lifestyle untuk story dan reel cover.", domain.AssetPhoto, []string{"foto", "lifestyle"}, []file{{"suasana.png", "png", "", c.image(5, 540, 960)}}},
		{"Template", "Template carousel edukasi", "Template 4:5 lima slide dengan grid brand.", domain.AssetTemplate, []string{"template", "carousel"}, []file{{"template-carousel.png", "png", "", c.image(6, 540, 675)}}},
	}
	count := 0
	for _, s := range specs {
		a := domain.Asset{ID: ids.New(), ClientID: c.client.ID, FolderID: strPtr(folderIDs[s.folder]), Kind: s.kind, Name: s.name, Description: s.desc, Tags: s.tags, CurrentVersion: int32(len(s.files)), CreatedBy: c.lead()}
		err := c.Store.WithTx(ctx, func(tx domain.Store) error {
			if err := tx.InsertAsset(ctx, a); err != nil {
				return err
			}
			for v, f := range s.files {
				obj, err := c.put(f.ext, f.data)
				if err != nil {
					return err
				}
				mime, _ := domain.SniffFile(f.data)
				if err := tx.InsertAssetVersion(ctx, domain.AssetVersion{ID: ids.New(), AssetID: a.ID, Version: int32(v + 1), FileKey: obj.Key, FileName: f.name, MimeType: mime, SizeBytes: obj.SizeBytes, Note: f.note, UploadedBy: c.designer()}); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// report writes the month's report from the demo engagement snapshots
// (the seed has no real platform posts to read).
func (c *clientSeed) report(ctx context.Context, month domain.Month, items []domain.Item) error {
	metrics, err := c.metrics(ctx, items)
	if err != nil {
		return err
	}
	end := month.Last().AddDate(0, 0, 3)
	data := domain.ComputeReport(month, c.client.Name, items, metrics, end)
	summary, model, err := c.AI.SummarizeReport(ctx, data)
	if err != nil {
		return err
	}
	_, err = c.Store.UpsertReport(ctx, domain.Report{ID: ids.New(), ClientID: c.client.ID, Month: month, Data: data, Summary: summary, AIModel: model, CreatedBy: c.lead()})
	return err
}

func hashtag(s string) string {
	var b []rune
	upper := true
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			if upper {
				r -= 'a' - 'A'
			}
			b = append(b, r)
			upper = false
		case r >= 'A' && r <= 'Z' || r >= '0' && r <= '9':
			b = append(b, r)
			upper = false
		default:
			upper = true
		}
	}
	return string(b)
}

func guidelinePDF(client string) ([]byte, error) {
	f := fpdf.New("P", "mm", "A4", "")
	f.AddPage()
	f.SetFont("Helvetica", "B", 22)
	f.Cell(0, 14, "Brand Guideline "+client)
	f.Ln(18)
	f.SetFont("Helvetica", "", 11)
	for _, line := range []string{"1. Logo: ruang kosong minimal setinggi huruf utama.", "2. Warna utama dan sekunder sesuai palet.", "3. Tipografi: headline tebal, body regular.", "4. Tone of voice: hangat, jelas, tanpa jargon."} {
		f.Cell(0, 8, line)
		f.Ln(9)
	}
	var buf bytes.Buffer
	err := f.Output(&buf)
	return buf.Bytes(), err
}

// fontZip is a font package placeholder: the license note a real package
// would carry (the demo ships no font binaries).
func fontZip() ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("LISENSI.txt")
	if err != nil {
		return nil, err
	}
	if _, err := f.Write([]byte("Paket font brand (demo). Ganti dengan file .otf/.ttf resmi dari klien.\n")); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
