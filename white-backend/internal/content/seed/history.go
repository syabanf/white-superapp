package seed

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/providers/mockrand"
)

// historyMonths is how far back the history pass fills approved plans, so
// the dashboard can compare a full year with the year before.
const historyMonths = 24

// history adds an approved, mostly published plan for every missing month
// from 24 months ago to two months ago plus the current month in
// progress, backdated brand asset uploads and the demo engagement snapshot
// of every seeded published item. Months that already have a plan are left
// alone, so the pass is idempotent.
func (c *clientSeed) history(ctx context.Context, res *Result) error {
	cur := domain.MonthOf(c.Now)
	for back := historyMonths; back >= 2; back-- {
		if err := c.historyMonth(ctx, domain.Month{Time: cur.AddDate(0, -back, 0)}, back, res); err != nil {
			return err
		}
	}
	if err := c.historyMonth(ctx, cur, 0, res); err != nil {
		return err
	}
	if err := c.reviewPlan(ctx, domain.Month{Time: cur.AddDate(0, 2, 0)}, res); err != nil {
		return err
	}
	if err := c.assetHistory(ctx, res); err != nil {
		return err
	}
	if err := c.repair(ctx); err != nil {
		return err
	}
	if err := c.repairPlans(ctx); err != nil {
		return err
	}
	if err := c.repairDrafts(ctx); err != nil {
		return err
	}
	return c.backfillMetrics(ctx, res)
}

func (c *clientSeed) historyMonth(ctx context.Context, m domain.Month, back int, res *Result) error {
	existing, err := c.Store.GetPlanByMonth(ctx, c.client.ID, m)
	if err != nil || existing != nil {
		return err
	}
	specs := c.monthSpecs(m, back)
	plan := domain.Plan{ID: ids.New(), ClientID: c.client.ID, Month: m, Title: "EP " + m.Label(), Status: domain.PlanApproved, Brief: c.brief(),
		CreatedBy: c.copywriter(), CreatedAt: c.before(*at(m.AddDate(0, 0, -12), 10), 72*time.Hour)}
	var items []domain.Item
	err = c.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertPlan(ctx, plan); err != nil {
			return err
		}
		model := "mock"
		plan.AIModel, plan.GeneratedAt = &model, &plan.CreatedAt
		plan.SubmittedBy, plan.SubmittedAt = c.copywriter(), c.beforePtr(at(m.AddDate(0, 0, -9), 15), 48*time.Hour)
		plan.ApprovedBy, plan.ApprovedAt = c.lead(), c.beforePtr(at(m.AddDate(0, 0, -7), 11), 24*time.Hour)
		plan.ReviewNote = strPtr("Sudah sesuai brief, lanjut produksi.")
		if err := tx.UpdatePlan(ctx, plan); err != nil {
			return err
		}
		for i, spec := range specs {
			it, err := c.item(ctx, tx, plan, spec, 1000+(historyMonths-back)*50+i)
			if err != nil {
				return err
			}
			items = append(items, *it)
		}
		return nil
	})
	if err != nil {
		return err
	}
	res.Plans++
	res.Items += len(items)
	res.HistoryMonths++
	if back == 0 {
		_, err = c.metrics(ctx, items)
		return err
	}
	if err := c.report(ctx, m, items); err != nil {
		return err
	}
	res.Reports++
	return nil
}

var reviewSpecs = []itemSpec{
	{3, domain.PlatformInstagram, domain.FormatCarousel, domain.ItemCopywriting, 0, false, 0, 19},
	{6, domain.PlatformInstagram, domain.FormatReel, domain.ItemCopywriting, 0, false, 0, 12},
	{10, domain.PlatformFacebook, domain.FormatFeed, domain.ItemIdea, 0, false, 0, 9},
	{14, domain.PlatformTikTok, domain.FormatReel, domain.ItemIdea, 0, false, 0, 20},
	{18, domain.PlatformInstagram, domain.FormatFeed, domain.ItemIdea, 0, false, 0, 17},
	{23, domain.PlatformInstagram, domain.FormatStory, domain.ItemIdea, 0, false, 0, 19},
	{27, domain.PlatformInstagram, domain.FormatCarousel, domain.ItemIdea, 0, false, 0, 18},
}

// reviewPlan adds the plan two months ahead, submitted and waiting for
// review with a short discussion; its copywriting items already have a
// design task waiting to start, so every plan and design status exists.
func (c *clientSeed) reviewPlan(ctx context.Context, m domain.Month, res *Result) error {
	existing, err := c.Store.GetPlanByMonth(ctx, c.client.ID, m)
	if err != nil || existing != nil {
		return err
	}
	plan := domain.Plan{ID: ids.New(), ClientID: c.client.ID, Month: m, Title: "EP " + m.Label(), Status: domain.PlanInReview, Brief: c.brief(),
		CreatedBy: c.copywriter(), CreatedAt: c.Now.Add(-96 * time.Hour)}
	return c.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertPlan(ctx, plan); err != nil {
			return err
		}
		model := "mock"
		plan.AIModel, plan.GeneratedAt = &model, &plan.CreatedAt
		plan.SubmittedBy, plan.SubmittedAt = c.copywriter(), strPtrTime(c.Now.Add(-20*time.Hour))
		if err := tx.UpdatePlan(ctx, plan); err != nil {
			return err
		}
		for i, comment := range []struct {
			body string
			user *string
		}{
			{"Draft sudah saya ajukan. Fokus bulan ini kampanye akhir tahun.", c.copywriter()},
			{"Pilar edukasi masih kurang, tambah satu carousel tips di minggu kedua ya.", c.lead()},
			{"Siap, carousel tanggal 10 saya ganti jadi edukasi.", c.copywriter()},
		} {
			if err := tx.InsertPlanComment(ctx, domain.PlanComment{ID: ids.New(), PlanID: plan.ID, UserID: comment.user, Body: comment.body, CreatedAt: c.Now.Add(time.Duration(i*3-18) * time.Hour)}); err != nil {
				return err
			}
		}
		for i, spec := range reviewSpecs {
			it, err := c.item(ctx, tx, plan, spec, 2000+i)
			if err != nil {
				return err
			}
			res.Items++
			if spec.status != domain.ItemCopywriting {
				continue
			}
			preset := domain.DefaultSizePreset(it.Format)
			task := domain.DesignTask{ID: ids.New(), ItemID: it.ID, ClientID: c.client.ID, Title: "Desain: " + it.Title, Brief: it.VisualBrief,
				SizePreset: preset.Key, Width: preset.Width, Height: preset.Height, DesignerID: c.designer(), DueDate: it.DesignDue,
				Status: domain.DesignTodo, CreatedBy: c.lead(), CreatedAt: c.Now.Add(-10 * time.Hour)}
			if err := tx.InsertDesignTask(ctx, task); err != nil {
				return err
			}
		}
		res.Plans++
		return nil
	})
}

func strPtrTime(t time.Time) *time.Time { return &t }

// Weighted pools: evenings dominate posting times, Instagram dominates
// platforms.
var (
	historyHours   = []int{8, 9, 11, 12, 12, 13, 16, 17, 18, 19, 19, 19, 20, 20, 21}
	instagramMixes = []domain.Format{domain.FormatFeed, domain.FormatFeed, domain.FormatCarousel, domain.FormatCarousel, domain.FormatCarousel, domain.FormatReel, domain.FormatReel, domain.FormatStory}
)

// monthSpecs plans 12 to 16 items spread over the month. Older months run
// later and need more revision rounds, so the year shows the team improving.
// back 0 is the current month: past days are mostly published, upcoming
// days sit in the stage their deadline implies, a few are overdue.
func (c *clientSeed) monthSpecs(m domain.Month, back int) []itemSpec {
	r := mockrand.New(fmt.Sprintf("%s|history|%s", c.client.ID, m))
	progress := float64(historyMonths-back) / historyMonths
	count := 12 + r.IntN(5)
	last := m.Last().Day()
	today := domain.Day(c.Now)
	specs := make([]itemSpec, 0, count)
	prevDay := 0
	for k := 0; k < count; k++ {
		day := max(prevDay+1, 1+k*last/count+r.IntN(2))
		if day > last {
			break
		}
		prevDay = day
		s := itemSpec{dayOfMonth: day, hour: mockrand.Pick(r, historyHours)}
		switch p := r.Float64(); {
		case p < 0.6:
			s.platform, s.format = domain.PlatformInstagram, mockrand.Pick(r, instagramMixes)
		case p < 0.8:
			s.platform, s.format = domain.PlatformFacebook, mockrand.Pick(r, []domain.Format{domain.FormatFeed, domain.FormatFeed, domain.FormatReel})
		default:
			s.platform, s.format = domain.PlatformTikTok, domain.FormatReel
		}
		switch x := r.Float64(); {
		case x < 0.40+0.2*progress:
		case x < 0.72+0.1*progress:
			s.rounds = 1
		case x < 0.93:
			s.rounds = 2
		default:
			s.rounds = 3
		}
		if r.Float64() < 0.32-0.2*progress {
			s.slip = 1 + r.IntN(3)
		}
		s.status = domain.ItemPublished
		if back > 0 {
			switch x := r.Float64(); {
			case x < 0.05:
				s.status = domain.ItemFailed
			case x < 0.09:
				s.status = domain.ItemApproved
			}
		} else {
			s.status, s.rounds = currentStatus(r, m.AddDate(0, 0, day-1), today, s.rounds)
		}
		specs = append(specs, s)
	}
	return specs
}

// currentStatus picks the stage of a current-month item from how far its
// planned day is from today.
func currentStatus(r *mockrand.Source, planned, today time.Time, rounds int32) (domain.ItemStatus, int32) {
	days := int(planned.Sub(today).Hours() / 24)
	pick := func(xs ...domain.ItemStatus) domain.ItemStatus { return mockrand.Pick(r, xs) }
	switch {
	case days < -1:
		if r.Float64() < 0.15 {
			return pick(domain.ItemReview, domain.ItemCopywriting, domain.ItemDesign), rounds
		}
		return domain.ItemPublished, rounds
	case days <= 2:
		return pick(domain.ItemScheduled, domain.ItemScheduled, domain.ItemApproved, domain.ItemReview), rounds
	case days <= 6:
		s := pick(domain.ItemDesign, domain.ItemReview, domain.ItemRevision)
		if s == domain.ItemRevision {
			rounds = max(rounds, 1)
		}
		return s, rounds
	}
	return pick(domain.ItemCopywriting, domain.ItemIdea, domain.ItemIdea), 0
}

// demoMetrics is deterministic engagement for a seeded external id: reels
// reach furthest, carousels get saved, evening posts do better and the
// audience grows about 3% a month (1.0 is the level a year ago).
func (c *clientSeed) demoMetrics(it domain.Item) domain.PostMetrics {
	r := mockrand.New(it.ExternalIDs[0])
	lo, hi := 1800.0, 5500.0
	switch it.Format {
	case domain.FormatReel:
		lo, hi = 5000, 13000
	case domain.FormatCarousel:
		lo, hi = 3000, 8500
	case domain.FormatStory:
		lo, hi = 800, 2600
	}
	reach := r.Between(lo, hi)
	switch it.Platform {
	case domain.PlatformTikTok:
		reach *= 1.5
	case domain.PlatformFacebook:
		reach *= 0.6
	}
	age := int(c.Now.Sub(it.PlannedDate).Hours() / 24 / 30)
	reach *= 1 + 0.03*float64(12-age)
	if it.PublishedAt != nil {
		switch h := it.PublishedAt.In(domain.AgencyLocation()).Hour(); {
		case h >= 17 && h <= 21:
			reach *= 1.15
		case h < 10:
			reach *= 0.85
		}
	}
	shares, saves, views := r.Between(0.001, 0.005), r.Between(0.002, 0.008), 1.2
	switch it.Format {
	case domain.FormatReel:
		shares, views = r.Between(0.006, 0.015), r.Between(1.8, 2.8)
	case domain.FormatCarousel:
		saves = r.Between(0.01, 0.03)
	case domain.FormatStory:
		views = r.Between(1.4, 2)
	}
	n := func(f float64) int64 { return int64(reach * f) }
	return domain.PostMetrics{Likes: n(r.Between(0.03, 0.08)), Comments: n(r.Between(0.002, 0.012)), Shares: n(shares), Saves: n(saves), Reach: int64(reach), Views: n(views)}
}

func isSeeded(it domain.Item) bool {
	return it.Status == domain.ItemPublished && len(it.ExternalIDs) > 0 && strings.HasPrefix(it.ExternalIDs[0], "seed_")
}

// metrics stores the demo snapshot of the seeded published items and
// returns it keyed by item id.
func (c *clientSeed) metrics(ctx context.Context, items []domain.Item) (map[string]domain.PostMetrics, error) {
	out := map[string]domain.PostMetrics{}
	for _, it := range items {
		if !isSeeded(it) {
			continue
		}
		m := c.demoMetrics(it)
		if err := c.Store.UpsertItemMetrics(ctx, it.ID, m); err != nil {
			return nil, err
		}
		out[it.ID] = m
	}
	return out, nil
}

// backfillMetrics gives seeded items from earlier seed runs their snapshot
// and regenerates the reports of the months that changed.
func (c *clientSeed) backfillMetrics(ctx context.Context, res *Result) error {
	items, err := c.Store.ListItems(ctx, c.client.ID, domain.ItemFilter{})
	if err != nil {
		return err
	}
	var ids []string
	for _, it := range items {
		if isSeeded(it) {
			ids = append(ids, it.ID)
		}
	}
	have, err := c.Store.ItemMetrics(ctx, ids)
	if err != nil {
		return err
	}
	var missing []domain.Item
	touched := map[string]bool{}
	for _, it := range items {
		if _, ok := have[it.ID]; isSeeded(it) && !ok {
			missing = append(missing, it)
			touched[domain.MonthOf(it.PlannedDate).String()] = true
		}
	}
	if len(missing) == 0 {
		return nil
	}
	if _, err := c.metrics(ctx, missing); err != nil {
		return err
	}
	reports, err := c.Store.ListReports(ctx, c.client.ID)
	if err != nil {
		return err
	}
	for _, rep := range reports {
		if !touched[rep.Month.String()] {
			continue
		}
		monthItems, err := c.Store.ListItems(ctx, c.client.ID, domain.ItemFilter{Month: rep.Month})
		if err != nil {
			return err
		}
		if err := c.report(ctx, rep.Month, monthItems); err != nil {
			return err
		}
		res.Reports++
	}
	return nil
}

// assetHistory adds brand assets that were refreshed over the seeded history
// (versions dated monthly or quarterly) once per client.
func (c *clientSeed) assetHistory(ctx context.Context, res *Result) error {
	folders, err := c.Store.ListFolders(ctx, c.client.ID)
	if err != nil {
		return err
	}
	folderID := map[string]*string{}
	for _, f := range folders {
		folderID[f.Name] = strPtr(f.ID)
	}
	specs := []struct {
		folder, name, desc string
		kind               domain.AssetKind
		tags               []string
		everyMonths        int
		w, h               int
	}{
		{"Template", "Template promo bulanan", "Template feed promo, diperbarui tiap bulan mengikuti kampanye.", domain.AssetTemplate, []string{"template", "promo"}, 1, 540, 675},
		{"Foto Produk", "Foto katalog musiman", "Foto katalog per musim untuk feed dan iklan.", domain.AssetPhoto, []string{"foto", "katalog"}, 3, 540, 675},
		{"Template", "Cover highlight story", "Ikon cover highlight Instagram.", domain.AssetTemplate, []string{"template", "story"}, 4, 270, 480},
		{"", "Press kit media", "Foto dan fakta singkat brand untuk media dan partner.", domain.AssetOther, []string{"press", "media"}, 6, 540, 540},
	}
	start := domain.MonthOf(c.Now).AddDate(0, -historyMonths+1, 0)
	for i, s := range specs {
		found, err := c.Store.SearchAssets(ctx, c.client.ID, domain.AssetFilter{Query: s.name})
		if err != nil {
			return err
		}
		if len(found) > 0 {
			continue
		}
		var dates []time.Time
		for d := start.AddDate(0, 0, 3+i*4); d.Before(c.Now); d = d.AddDate(0, s.everyMonths, 0) {
			dates = append(dates, *at(d, 14))
		}
		a := domain.Asset{ID: ids.New(), ClientID: c.client.ID, FolderID: folderID[s.folder], Kind: s.kind, Name: s.name, Description: s.desc, Tags: s.tags, CurrentVersion: int32(len(dates)), CreatedBy: c.lead()}
		err = c.Store.WithTx(ctx, func(tx domain.Store) error {
			if err := tx.InsertAsset(ctx, a); err != nil {
				return err
			}
			for v, d := range dates {
				obj, err := c.put("png", c.image(v, s.w, s.h))
				if err != nil {
					return err
				}
				name := fmt.Sprintf("%s-%s.png", hashtag(s.name), d.Format("2006-01"))
				if err := tx.InsertAssetVersion(ctx, domain.AssetVersion{ID: ids.New(), AssetID: a.ID, Version: int32(v + 1), FileKey: obj.Key, FileName: name, MimeType: "image/png", SizeBytes: obj.SizeBytes, Note: "Pembaruan " + domain.MonthOf(d).Label(), UploadedBy: c.designer(), CreatedAt: d}); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		res.Assets++
	}
	return nil
}

// repair brings seeded items from earlier seed runs in line with the
// history: items created on the seed day for a past plan get a creation
// date before their deadlines, past items rotate between the team so the
// workload charts compare people, and items without comments get a short
// discussion. Applying it twice changes nothing.
func (c *clientSeed) repair(ctx context.Context) error {
	items, err := c.Store.ListItems(ctx, c.client.ID, domain.ItemFilter{})
	if err != nil {
		return err
	}
	cur := domain.MonthOf(c.Now).Time
	for _, it := range items {
		if !strings.HasSuffix(it.VisualBrief, seedBriefTail) {
			continue
		}
		past := it.PlannedDate.Before(cur)
		created := it.CreatedAt
		if past && created.After(it.PlannedDate.AddDate(0, 0, -3)) {
			created = c.before(*at(it.PlannedDate.AddDate(0, 0, -12), 9), 72*time.Hour)
		}
		copyw, designer := it.CopywriterID, it.DesignerID
		if n := len(c.users); n > 1 && past {
			h := int(mockrand.Seed(it.ID))
			copyw, designer = &c.users[h%n], &c.users[(h+1)%n]
		}
		reassigned := !sameID(copyw, it.CopywriterID) || !sameID(designer, it.DesignerID)
		comments, err := c.Store.ListItemComments(ctx, it.ID)
		if err != nil {
			return err
		}
		discuss := len(comments) == 0
		if !reassigned && !discuss && created.Equal(it.CreatedAt) {
			continue
		}
		updated := it.UpdatedAt
		err = c.Store.WithTx(ctx, func(tx domain.Store) error {
			if reassigned {
				it.CopywriterID, it.DesignerID = copyw, designer
				if err := tx.UpdateItem(ctx, it); err != nil {
					return err
				}
				tasks, err := tx.ListDesignTasks(ctx, c.client.ID, it.ID, "", "")
				if err != nil {
					return err
				}
				for _, t := range tasks {
					t.DesignerID = designer
					if err := tx.UpdateDesignTask(ctx, t); err != nil {
						return err
					}
				}
			}
			if discuss {
				if err := c.discussion(ctx, tx, it, created); err != nil {
					return err
				}
			}
			return tx.BackdateItem(ctx, it.ID, created, updated)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

var discussionLines = []struct{ lead, reply string }{
	{"Referensi visualnya sudah cocok, tolong pakai foto produk terbaru.", "Siap, saya ambil dari folder Foto Produk."},
	{"Jam tayang bisa digeser ke malam? Audiens kita paling aktif jam 19.00.", "Oke, saya sesuaikan jadwalnya."},
	{"Caption sudah bagus, tambahkan ajakan simpan postingan di akhir.", "Sudah saya tambahkan di versi terbaru."},
	{"Pastikan logo terlihat jelas di slide pertama.", "Noted, logo saya taruh di pojok kanan bawah."},
}

// discussion adds a short lead and copywriter exchange to an item that has
// no comments yet, dated a day after it was created.
func (c *clientSeed) discussion(ctx context.Context, tx domain.Store, it domain.Item, created time.Time) error {
	line := discussionLines[int(mockrand.Seed(it.ID+"|talk"))%len(discussionLines)]
	at := c.before(created.Add(26*time.Hour), 2*time.Hour)
	if err := tx.InsertItemComment(ctx, domain.ItemComment{ID: ids.New(), ItemID: it.ID, UserID: c.lead(), Body: line.lead, Kind: domain.DecisionComment, CreatedAt: at}); err != nil {
		return err
	}
	return tx.InsertItemComment(ctx, domain.ItemComment{ID: ids.New(), ItemID: it.ID, UserID: it.CopywriterID, Body: line.reply, Kind: domain.DecisionComment, CreatedAt: c.before(at.Add(90*time.Minute), time.Hour)})
}

func sameID(a, b *string) bool {
	return (a == nil) == (b == nil) && (a == nil || *a == *b)
}

var planDiscussion = []string{
	"Brief bulan ini sudah saya rangkum, mohon dicek pilar dan frekuensinya.",
	"Sudah oke. Tolong pastikan ada satu konten promo di minggu gajian.",
}

// repairPlans gives seeded plans without comments a short review thread.
func (c *clientSeed) repairPlans(ctx context.Context) error {
	plans, err := c.Store.ListPlans(ctx, c.client.ID)
	if err != nil {
		return err
	}
	for _, p := range plans {
		if !strings.HasPrefix(p.Title, "EP ") {
			continue
		}
		comments, err := c.Store.ListPlanComments(ctx, p.ID)
		if err != nil || len(comments) > 0 {
			if err != nil {
				return err
			}
			continue
		}
		for i, body := range planDiscussion {
			user := c.copywriter()
			if i == 1 {
				user = c.lead()
			}
			at := c.before(p.CreatedAt.Add(time.Duration(24+i*3)*time.Hour), time.Hour)
			if err := c.Store.InsertPlanComment(ctx, domain.PlanComment{ID: ids.New(), PlanID: p.ID, UserID: user, Body: body, CreatedAt: at}); err != nil {
				return err
			}
		}
	}
	return nil
}

// repairDrafts uploads a first draft to in-progress design tasks that have
// no file yet (seed runs before drafts existed).
func (c *clientSeed) repairDrafts(ctx context.Context) error {
	tasks, err := c.Store.ListDesignTasks(ctx, c.client.ID, "", "", domain.DesignInProgress)
	if err != nil {
		return err
	}
	for _, t := range tasks {
		if t.CurrentVersion > 0 || !strings.HasPrefix(t.Title, "Desain: ") {
			continue
		}
		obj, err := c.put("png", c.image(1, int(t.Width/4), int(t.Height/4)))
		if err != nil {
			return err
		}
		err = c.Store.WithTx(ctx, func(tx domain.Store) error {
			at := c.before(t.CreatedAt.Add(20*time.Hour), 2*time.Hour)
			if err := tx.InsertDesignFile(ctx, domain.DesignFile{ID: ids.New(), TaskID: t.ID, Version: 1, FileKey: obj.Key, FileName: "draft-v1.png", MimeType: "image/png", SizeBytes: obj.SizeBytes, UploadedBy: t.DesignerID, CreatedAt: at}); err != nil {
				return err
			}
			t.CurrentVersion = 1
			return tx.UpdateDesignTask(ctx, t)
		})
		if err != nil {
			return err
		}
	}
	return nil
}
