package domain

import (
	"sort"
	"time"
)

// DashboardFilter narrows the content dashboard. From and To are inclusive
// agency days (UTC midnight); empty strings mean "any".
type DashboardFilter struct {
	From, To   time.Time
	Platform   Platform
	Format     Format
	Pillar     string
	PlanID     string
	AssigneeID string
}

// Compare modes for the comparison window.
const (
	ComparePrevious = "previous"
	CompareYear     = "year"
)

// CompareWindow is the window the deltas compare against: the same length
// ending the day before, or the same days one year earlier.
func CompareWindow(from, to time.Time, mode string) (time.Time, time.Time) {
	if mode == CompareYear {
		return from.AddDate(-1, 0, 0), to.AddDate(-1, 0, 0)
	}
	days := int(to.Sub(from).Hours()/24) + 1
	prevTo := from.AddDate(0, 0, -1)
	return prevTo.AddDate(0, 0, -(days - 1)), prevTo
}

// DashItem is the slice of an item the dashboard reads.
type DashItem struct {
	ID              string
	PlanID          string
	PlannedDate     time.Time
	Platform        Platform
	Format          Format
	Pillar          string
	Title           string
	Status          ItemStatus
	RevisionRound   int32
	CopywriterID    *string
	DesignerID      *string
	CopyDue         *time.Time
	DesignDue       *time.Time
	ReviewDue       *time.Time
	CreatedAt       time.Time
	CopyDoneAt      *time.Time
	DesignDoneAt    *time.Time
	ApprovedAt      *time.Time
	PublishedAt     *time.Time
	Permalink       *string
	ExternalIDs     []string
	ThumbnailFileID string
}

type DashDesignTask struct {
	ID               string
	ItemID           string
	DesignerID       *string
	Status           DesignStatus
	Versions         int32
	CreatedAt        time.Time
	ApprovedAt       *time.Time
	ChangesRequested int32
}

type DashAssetVersion struct {
	AssetID        string
	Name           string
	Kind           AssetKind
	CurrentVersion int32
	Version        int32
	CreatedAt      time.Time
}

// DashboardInput is one window's rows plus engagement keyed by item id.
type DashboardInput struct {
	From, To time.Time
	Items    []DashItem
	Tasks    []DashDesignTask
	Assets   []DashAssetVersion
	Metrics  map[string]PostMetrics
}

type DashKpis struct {
	Planned, Published, Failed, InReview, Overdue int32
	DeadlinesMet, DeadlinesTotal                  int32
	OnTimeRate                                    float64
	RevisionsTotal                                int32
	RevisionAvg                                   float64
	Engagements, Reach                            int64
	EngagementRate                                float64
	MeasuredPosts                                 int32
	CycleDaysAvg                                  float64
	DesignTasks, DesignApproved                   int32
	DesignTurnaroundDays                          float64
	FirstPassRate                                 float64
	AssetUploads                                  int32
}

// DashBucket is one week or month of the window. Stage* count work in
// progress at the end of the bucket.
type DashBucket struct {
	Start                        time.Time
	Planned, Published           int32
	Engagements, Reach           int64
	DeadlinesMet, DeadlinesTotal int32
	OnTimeRate                   float64
	RevisionAvg                  float64
	StageCopy, StageDesign       int32
	StageReview, StageReady      int32
	DesignTurnaroundDays         float64
	AssetUploads                 int32
}

type DashCount struct {
	Key   string
	Count int32
}

type DashStageCycle struct {
	Stage      string
	AvgDays    float64
	MedianDays float64
	Count      int32
}

// DashPerson is one assignee in one role (COPYWRITER or DESIGNER).
type DashPerson struct {
	UserID                                    string
	Role                                      string
	Assigned, Published                       int32
	Overdue                                   int32
	OverdueCopy, OverdueDesign, OverdueReview int32
	RevisionAvg                               float64
	OnTimeRate                                float64
	DesignTasks, DesignApproved               int32
	TurnaroundDays                            float64
	FirstPassRate                             float64
}

type DashDay struct {
	Date               time.Time
	Planned, Published int32
}

type DashHeatCell struct {
	// Weekday 0 = Monday.
	Weekday, Hour int32
	Posts         int32
	Engagements   int64
}

type DashBreakdown struct {
	Key                string
	Planned, Published int32
	Engagements, Reach int64
	Measured           int32
}

type DashPost struct {
	ItemID          string
	Title           string
	Platform        Platform
	Format          Format
	Pillar          string
	PlannedDate     time.Time
	PublishedAt     *time.Time
	Permalink       *string
	ThumbnailFileID string
	HasMetrics      bool
	Metrics         PostMetrics
}

type DashAsset struct {
	AssetID        string
	Name           string
	Kind           AssetKind
	Uploads        int32
	CurrentVersion int32
	LastUploadedAt time.Time
}

// Dashboard is the computed body of one window.
type Dashboard struct {
	Granularity      string
	Kpis             DashKpis
	Buckets          []DashBucket
	Funnel           []DashCount
	Statuses         []DashCount
	StageCycles      []DashStageCycle
	OverdueByStage   []DashCount
	RevisionRounds   []DashCount
	DesignVersions   []DashCount
	People           []DashPerson
	Days             []DashDay
	PostingHeat      []DashHeatCell
	ByPlatform       []DashBreakdown
	ByFormat         []DashBreakdown
	ByPillar         []DashBreakdown
	Posts            []DashPost
	TopAssets        []DashAsset
	AssetUploadsKind []DashCount
}

// Workflow stages for cycle times and overdue counts.
const (
	StageCopy    = "COPY"
	StageDesign  = "DESIGN"
	StageReview  = "REVIEW"
	StagePublish = "PUBLISH"
)

// MaxDashPosts caps the post list (top and bottom tables, scatter).
const MaxDashPosts = 300

// funnelOrder is the happy path; REVISION counts as REVIEW reached and
// FAILED as SCHEDULED reached.
var funnelOrder = []ItemStatus{ItemIdea, ItemCopywriting, ItemDesign, ItemReview, ItemApproved, ItemScheduled, ItemPublished}

func funnelRank(s ItemStatus) int {
	switch s {
	case ItemRevision:
		s = ItemReview
	case ItemFailed:
		s = ItemScheduled
	}
	for i, f := range funnelOrder {
		if f == s {
			return i
		}
	}
	return 0
}

// Granularity is "week" up to 120 days, "month" beyond.
func Granularity(from, to time.Time) string {
	if to.Sub(from).Hours()/24 > 120 {
		return "month"
	}
	return "week"
}

func bucketStart(day time.Time, gran string) time.Time {
	if gran == "month" {
		return MonthOf(day).Time
	}
	// ISO week, Monday first.
	offset := (int(day.Weekday()) + 6) % 7
	return day.AddDate(0, 0, -offset)
}

func nextBucket(start time.Time, gran string) time.Time {
	if gran == "month" {
		return start.AddDate(0, 1, 0)
	}
	return start.AddDate(0, 0, 7)
}

// endOfDay is the instant the agency day ends.
func endOfDay(day time.Time) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day()+1, 0, 0, 0, 0, AgencyLocation())
}

func days(a, b time.Time) float64 { return b.Sub(a).Hours() / 24 }

func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	m := len(s) / 2
	if len(s)%2 == 1 {
		return s[m]
	}
	return (s[m-1] + s[m]) / 2
}

func avg(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	sum := 0.0
	for _, x := range v {
		sum += x
	}
	return sum / float64(len(v))
}

func pct(n, d int32) float64 {
	if d == 0 {
		return 0
	}
	return round1(float64(n) * 100 / float64(d))
}

// itemDeadlines scores the item's four deadlines like the monthly report.
func itemDeadlines(it DashItem, today time.Time) (met, total int32) {
	var publishedAt *time.Time
	if it.Status == ItemPublished {
		publishedAt = it.PublishedAt
	}
	planned := it.PlannedDate
	for _, dl := range []struct{ due, done *time.Time }{{it.CopyDue, it.CopyDoneAt}, {it.DesignDue, it.DesignDoneAt}, {it.ReviewDue, it.ApprovedAt}, {&planned, publishedAt}} {
		if counted, ok := deadlineHit(dl.due, dl.done, today); counted {
			total++
			if ok {
				met++
			}
		}
	}
	return met, total
}

// dashDue is the deadline of the item's current stage, like currentDue in the web app.
func dashDue(it DashItem) (string, *time.Time) {
	switch it.Status {
	case ItemIdea, ItemCopywriting:
		return StageCopy, it.CopyDue
	case ItemDesign:
		return StageDesign, it.DesignDue
	case ItemReview, ItemRevision:
		return StageReview, it.ReviewDue
	}
	return "", nil
}

func isOverdue(it DashItem, today time.Time) (string, bool) {
	stage, due := dashDue(it)
	if due == nil {
		return "", false
	}
	return stage, due.Before(today)
}

// wipGraceDays is how long past its planned day an unpublished item still
// counts as work in progress; after that it is dropped (missed or failed).
const wipGraceDays = 7

// stageAt is the work-in-progress stage at instant t ("" when not started
// yet, already published or abandoned).
func stageAt(it DashItem, t time.Time) string {
	before := func(p *time.Time) bool { return p != nil && !p.After(t) }
	switch {
	case it.CreatedAt.After(t), before(it.PublishedAt), t.After(endOfDay(it.PlannedDate.AddDate(0, 0, wipGraceDays))):
		return ""
	case before(it.ApprovedAt):
		return "READY"
	case before(it.DesignDoneAt):
		return StageReview
	case before(it.CopyDoneAt):
		return StageDesign
	}
	return StageCopy
}

// ComputeDashboard builds one window's numbers. now decides overdue items
// and which deadlines count.
func ComputeDashboard(in DashboardInput, now time.Time) Dashboard {
	today := Day(now)
	gran := Granularity(in.From, in.To)
	d := Dashboard{Granularity: gran}

	// Buckets covering the window.
	var buckets []*DashBucket
	index := map[time.Time]*DashBucket{}
	for s := bucketStart(in.From, gran); !s.After(in.To); s = nextBucket(s, gran) {
		b := &DashBucket{Start: s}
		buckets = append(buckets, b)
		index[s] = b
	}
	bucketOf := func(day time.Time) *DashBucket {
		if day.Before(in.From) || day.After(in.To) {
			return nil
		}
		return index[bucketStart(day, gran)]
	}
	bucketRevisions := map[*DashBucket][2]int32{}

	dayIndex := map[time.Time]*DashDay{}
	for day := in.From; !day.After(in.To); day = day.AddDate(0, 0, 1) {
		d.Days = append(d.Days, DashDay{Date: day})
	}
	for i := range d.Days {
		dayIndex[d.Days[i].Date] = &d.Days[i]
	}

	statusCount := map[ItemStatus]int32{}
	funnel := make([]int32, len(funnelOrder))
	overdueStage := map[string]int32{}
	rounds := map[string]int32{}
	cycles := map[string][]float64{}
	var cycleTotal []float64
	heat := map[[2]int32]*DashHeatCell{}
	platform, format, pillar := breakdowns{}, breakdowns{}, breakdowns{}
	people := map[[2]string]*personAcc{}
	person := func(id *string, role string) *personAcc {
		if id == nil || *id == "" {
			return nil
		}
		k := [2]string{*id, role}
		p := people[k]
		if p == nil {
			p = &personAcc{DashPerson: DashPerson{UserID: *id, Role: role}}
			people[k] = p
		}
		return p
	}

	k := &d.Kpis
	for _, it := range in.Items {
		k.Planned++
		statusCount[it.Status]++
		for i := 0; i <= funnelRank(it.Status); i++ {
			funnel[i]++
		}
		published := it.Status == ItemPublished
		switch it.Status {
		case ItemPublished:
			k.Published++
		case ItemFailed:
			k.Failed++
		case ItemReview, ItemRevision:
			k.InReview++
		}
		met, total := itemDeadlines(it, today)
		k.DeadlinesMet += met
		k.DeadlinesTotal += total
		k.RevisionsTotal += it.RevisionRound
		switch {
		case it.RevisionRound >= 3:
			rounds["3+"]++
		default:
			rounds[[]string{"0", "1", "2"}[it.RevisionRound]]++
		}
		stage, late := isOverdue(it, today)
		if late {
			k.Overdue++
			overdueStage[stage]++
		}

		if b := bucketOf(it.PlannedDate); b != nil {
			b.Planned++
			b.DeadlinesMet += met
			b.DeadlinesTotal += total
			r := bucketRevisions[b]
			bucketRevisions[b] = [2]int32{r[0] + it.RevisionRound, r[1] + 1}
		}
		if day := dayIndex[it.PlannedDate]; day != nil {
			day.Planned++
		}

		// Cycle times per stage.
		for _, c := range []struct {
			stage    string
			from, to *time.Time
		}{{StageCopy, &it.CreatedAt, it.CopyDoneAt}, {StageDesign, it.CopyDoneAt, it.DesignDoneAt}, {StageReview, it.DesignDoneAt, it.ApprovedAt}, {StagePublish, it.ApprovedAt, it.PublishedAt}} {
			if c.from != nil && c.to != nil && !c.to.Before(*c.from) {
				cycles[c.stage] = append(cycles[c.stage], days(*c.from, *c.to))
			}
		}

		m, measured := in.Metrics[it.ID]
		measured = measured && published
		if published && it.PublishedAt != nil {
			pubDay := Day(*it.PublishedAt)
			cycleTotal = append(cycleTotal, days(it.CreatedAt, *it.PublishedAt))
			if b := bucketOf(pubDay); b != nil {
				b.Published++
				if measured {
					b.Engagements += m.Engagements()
					b.Reach += m.Reach
				}
			}
			if day := dayIndex[pubDay]; day != nil {
				day.Published++
			}
			local := it.PublishedAt.In(AgencyLocation())
			key := [2]int32{int32((int(local.Weekday()) + 6) % 7), int32(local.Hour())}
			c := heat[key]
			if c == nil {
				c = &DashHeatCell{Weekday: key[0], Hour: key[1]}
				heat[key] = c
			}
			c.Posts++
			if measured {
				c.Engagements += m.Engagements()
			}
		}
		if measured {
			k.Engagements += m.Engagements()
			k.Reach += m.Reach
			k.MeasuredPosts++
		}
		if published {
			d.Posts = append(d.Posts, DashPost{ItemID: it.ID, Title: it.Title, Platform: it.Platform, Format: it.Format, Pillar: it.Pillar,
				PlannedDate: it.PlannedDate, PublishedAt: it.PublishedAt, Permalink: it.Permalink, ThumbnailFileID: it.ThumbnailFileID, HasMetrics: measured, Metrics: m})
		}
		platform.add(string(it.Platform), published, measured, m)
		format.add(string(it.Format), published, measured, m)
		pillar.add(it.Pillar, published, measured, m)

		for _, p := range []*personAcc{person(it.CopywriterID, "COPYWRITER"), person(it.DesignerID, "DESIGNER")} {
			if p == nil {
				continue
			}
			p.Assigned++
			if published {
				p.Published++
			}
			p.revisions += it.RevisionRound
			p.met += met
			p.total += total
		}
		if late {
			owner := person(it.CopywriterID, "COPYWRITER")
			if stage == StageDesign {
				owner = person(it.DesignerID, "DESIGNER")
			}
			if owner != nil {
				owner.Overdue++
				switch stage {
				case StageCopy:
					owner.OverdueCopy++
				case StageDesign:
					owner.OverdueDesign++
				default:
					owner.OverdueReview++
				}
			}
		}
	}

	// Work in progress at each bucket end (capped at now).
	for _, b := range buckets {
		end := endOfDay(nextBucket(b.Start, gran).AddDate(0, 0, -1))
		if limit := endOfDay(in.To); end.After(limit) {
			end = limit
		}
		if end.After(now) {
			end = now
		}
		for _, it := range in.Items {
			switch stageAt(it, end) {
			case StageCopy:
				b.StageCopy++
			case StageDesign:
				b.StageDesign++
			case StageReview:
				b.StageReview++
			case "READY":
				b.StageReady++
			}
		}
		b.OnTimeRate = pct(b.DeadlinesMet, b.DeadlinesTotal)
		if r := bucketRevisions[b]; r[1] > 0 {
			b.RevisionAvg = round1(float64(r[0]) / float64(r[1]))
		}
	}

	// Design tasks.
	versions := map[string]int32{}
	var turnaround []float64
	var firstPass int32
	bucketTurn := map[*DashBucket][]float64{}
	for _, t := range in.Tasks {
		k.DesignTasks++
		v := t.Versions
		key := "3+"
		if v <= 2 {
			key = []string{"1", "1", "2"}[max(v, 0)]
		}
		versions[key]++
		p := person(t.DesignerID, "DESIGNER")
		if p != nil {
			p.DesignTasks++
		}
		if t.ApprovedAt == nil {
			continue
		}
		k.DesignApproved++
		td := days(t.CreatedAt, *t.ApprovedAt)
		turnaround = append(turnaround, td)
		if b := bucketOf(Day(*t.ApprovedAt)); b != nil {
			bucketTurn[b] = append(bucketTurn[b], td)
		}
		if t.ChangesRequested == 0 {
			firstPass++
		}
		if p != nil {
			p.DesignApproved++
			p.turnaround = append(p.turnaround, td)
			if t.ChangesRequested == 0 {
				p.firstPass++
			}
		}
	}
	for b, v := range bucketTurn {
		b.DesignTurnaroundDays = round1(avg(v))
	}

	// Assets.
	assets := map[string]*DashAsset{}
	kinds := map[string]int32{}
	for _, v := range in.Assets {
		k.AssetUploads++
		kinds[string(v.Kind)]++
		if b := bucketOf(Day(v.CreatedAt)); b != nil {
			b.AssetUploads++
		}
		a := assets[v.AssetID]
		if a == nil {
			a = &DashAsset{AssetID: v.AssetID, Name: v.Name, Kind: v.Kind, CurrentVersion: v.CurrentVersion}
			assets[v.AssetID] = a
		}
		a.Uploads++
		if v.CreatedAt.After(a.LastUploadedAt) {
			a.LastUploadedAt = v.CreatedAt
		}
	}

	// KPI ratios.
	k.OnTimeRate = pct(k.DeadlinesMet, k.DeadlinesTotal)
	if k.Planned > 0 {
		k.RevisionAvg = round1(float64(k.RevisionsTotal) / float64(k.Planned))
	}
	if k.Reach > 0 {
		k.EngagementRate = round1(float64(k.Engagements) * 100 / float64(k.Reach))
	}
	k.CycleDaysAvg = round1(avg(cycleTotal))
	k.DesignTurnaroundDays = round1(avg(turnaround))
	k.FirstPassRate = pct(firstPass, k.DesignApproved)

	// Shape the output.
	for _, b := range buckets {
		d.Buckets = append(d.Buckets, *b)
	}
	for i, s := range funnelOrder {
		d.Funnel = append(d.Funnel, DashCount{Key: string(s), Count: funnel[i]})
	}
	for _, s := range ItemStatuses {
		if n := statusCount[s]; n > 0 {
			d.Statuses = append(d.Statuses, DashCount{Key: string(s), Count: n})
		}
	}
	for _, s := range []string{StageCopy, StageDesign, StageReview, StagePublish} {
		v := cycles[s]
		d.StageCycles = append(d.StageCycles, DashStageCycle{Stage: s, AvgDays: round1(avg(v)), MedianDays: round1(median(v)), Count: int32(len(v))})
	}
	for _, s := range []string{StageCopy, StageDesign, StageReview} {
		d.OverdueByStage = append(d.OverdueByStage, DashCount{Key: s, Count: overdueStage[s]})
	}
	for _, r := range []string{"0", "1", "2", "3+"} {
		d.RevisionRounds = append(d.RevisionRounds, DashCount{Key: r, Count: rounds[r]})
	}
	for _, v := range []string{"1", "2", "3+"} {
		d.DesignVersions = append(d.DesignVersions, DashCount{Key: v, Count: versions[v]})
	}
	for _, p := range people {
		if p.Assigned > 0 {
			p.RevisionAvg = round1(float64(p.revisions) / float64(p.Assigned))
		}
		p.OnTimeRate = pct(p.met, p.total)
		p.TurnaroundDays = round1(avg(p.turnaround))
		p.FirstPassRate = pct(p.firstPass, p.DesignApproved)
		d.People = append(d.People, p.DashPerson)
	}
	sort.Slice(d.People, func(i, j int) bool {
		a, b := d.People[i], d.People[j]
		if a.Assigned != b.Assigned {
			return a.Assigned > b.Assigned
		}
		if a.Role != b.Role {
			return a.Role < b.Role
		}
		return a.UserID < b.UserID
	})
	for _, c := range heat {
		d.PostingHeat = append(d.PostingHeat, *c)
	}
	sort.Slice(d.PostingHeat, func(i, j int) bool {
		a, b := d.PostingHeat[i], d.PostingHeat[j]
		return a.Weekday < b.Weekday || a.Weekday == b.Weekday && a.Hour < b.Hour
	})
	d.ByPlatform, d.ByFormat, d.ByPillar = platform.list(), format.list(), pillar.list()

	sort.SliceStable(d.Posts, func(i, j int) bool { return d.Posts[i].Metrics.Engagements() > d.Posts[j].Metrics.Engagements() })
	if len(d.Posts) > MaxDashPosts {
		d.Posts = d.Posts[:MaxDashPosts]
	}
	for _, a := range assets {
		d.TopAssets = append(d.TopAssets, *a)
	}
	sort.Slice(d.TopAssets, func(i, j int) bool {
		a, b := d.TopAssets[i], d.TopAssets[j]
		if a.Uploads != b.Uploads {
			return a.Uploads > b.Uploads
		}
		return a.LastUploadedAt.After(b.LastUploadedAt)
	})
	if len(d.TopAssets) > 10 {
		d.TopAssets = d.TopAssets[:10]
	}
	for _, kind := range []AssetKind{AssetLogo, AssetBrandGuideline, AssetFont, AssetPhoto, AssetTemplate, AssetOther} {
		if n := kinds[string(kind)]; n > 0 {
			d.AssetUploadsKind = append(d.AssetUploadsKind, DashCount{Key: string(kind), Count: n})
		}
	}
	return d
}

type personAcc struct {
	DashPerson
	revisions, met, total int32
	turnaround            []float64
	firstPass             int32
}

// breakdowns accumulates rows in first-seen order.
type breakdowns struct {
	order []string
	rows  map[string]*DashBreakdown
}

func (b *breakdowns) add(key string, published, measured bool, m PostMetrics) {
	if key == "" {
		key = "-"
	}
	if b.rows == nil {
		b.rows = map[string]*DashBreakdown{}
	}
	r := b.rows[key]
	if r == nil {
		r = &DashBreakdown{Key: key}
		b.rows[key] = r
		b.order = append(b.order, key)
	}
	r.Planned++
	if published {
		r.Published++
	}
	if measured {
		r.Measured++
		r.Engagements += m.Engagements()
		r.Reach += m.Reach
	}
}

// list sorts by planned count, largest first.
func (b *breakdowns) list() []DashBreakdown {
	out := make([]DashBreakdown, 0, len(b.order))
	for _, k := range b.order {
		out = append(out, *b.rows[k])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Planned > out[j].Planned })
	return out
}
