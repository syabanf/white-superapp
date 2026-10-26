package domain

import (
	"math"
	"sort"
	"time"
)

// PostMetrics is the engagement analytics synced for one published post.
type PostMetrics struct {
	Likes, Comments, Shares, Saves, Reach, Views int64
}

// Engagements is likes + comments + shares + saves, as analytics and
// publishing count it.
func (m PostMetrics) Engagements() int64 { return m.Likes + m.Comments + m.Shares + m.Saves }

func (m *PostMetrics) Add(o PostMetrics) {
	m.Likes += o.Likes
	m.Comments += o.Comments
	m.Shares += o.Shares
	m.Saves += o.Saves
	m.Reach += o.Reach
	m.Views += o.Views
}

type Breakdown struct {
	Key       string `json:"key"`
	Planned   int32  `json:"planned"`
	Published int32  `json:"published"`
}

type ReportPost struct {
	ItemID      string     `json:"itemId"`
	Title       string     `json:"title"`
	Platform    Platform   `json:"platform"`
	Format      Format     `json:"format"`
	PlannedDate string     `json:"plannedDate"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	Permalink   *string    `json:"permalink,omitempty"`
	HasMetrics  bool       `json:"hasMetrics"`
	Likes       int64      `json:"likes"`
	Comments    int64      `json:"comments"`
	Shares      int64      `json:"shares"`
	Saves       int64      `json:"saves"`
	Reach       int64      `json:"reach"`
	Views       int64      `json:"views"`
	Engagements int64      `json:"engagements"`
}

// ReportData is the computed body of a monthly content report, stored as
// JSON so a report keeps the numbers it was generated with.
type ReportData struct {
	Month               string       `json:"month"`
	ClientName          string       `json:"clientName"`
	Planned             int32        `json:"planned"`
	Published           int32        `json:"published"`
	Failed              int32        `json:"failed"`
	Scheduled           int32        `json:"scheduled"`
	InProgress          int32        `json:"inProgress"`
	DeadlinesMet        int32        `json:"deadlinesMet"`
	DeadlinesTotal      int32        `json:"deadlinesTotal"`
	OnTimeRate          float64      `json:"onTimeRate"`
	RevisionsTotal      int32        `json:"revisionsTotal"`
	RevisionAvg         float64      `json:"revisionAvg"`
	CaptionVersions     int32        `json:"captionVersions"`
	PerPlatform         []Breakdown  `json:"perPlatform"`
	PerFormat           []Breakdown  `json:"perFormat"`
	EngagementAvailable bool         `json:"engagementAvailable"`
	TotalEngagements    int64        `json:"totalEngagements"`
	TotalReach          int64        `json:"totalReach"`
	Posts               []ReportPost `json:"posts"`
	Top                 []ReportPost `json:"top"`
	Bottom              []ReportPost `json:"bottom"`
}

// TopN is how many posts the best and weakest lists hold.
const TopN = 3

func round1(v float64) float64 { return math.Round(v*10) / 10 }

// deadlineHit scores one deadline: counted once it is done or overdue,
// met when the work finished on or before the due day.
func deadlineHit(due *time.Time, done *time.Time, today time.Time) (counted, met bool) {
	if due == nil {
		return false, false
	}
	if done != nil {
		return true, !Day(*done).After(*due)
	}
	return today.After(*due), false
}

// ComputeReport builds the month's numbers from the plan items and the
// engagement found for published ones (keyed by item id).
func ComputeReport(month Month, clientName string, items []Item, metrics map[string]PostMetrics, now time.Time) ReportData {
	today := Day(now)
	d := ReportData{Month: month.String(), ClientName: clientName, PerPlatform: []Breakdown{}, PerFormat: []Breakdown{}, Posts: []ReportPost{}, Top: []ReportPost{}, Bottom: []ReportPost{}}
	platforms := map[string]*Breakdown{}
	formats := map[string]*Breakdown{}
	// Breakdowns keep first-seen order.
	bump := func(m map[string]*Breakdown, order *[]string, key string, published bool) {
		b := m[key]
		if b == nil {
			*order = append(*order, key)
			b = &Breakdown{Key: key}
			m[key] = b
		}
		b.Planned++
		if published {
			b.Published++
		}
	}
	var platformOrder, formatOrder []string
	for _, it := range items {
		d.Planned++
		published := it.Status == ItemPublished
		switch it.Status {
		case ItemPublished:
			d.Published++
		case ItemFailed:
			d.Failed++
		case ItemScheduled:
			d.Scheduled++
		default:
			d.InProgress++
		}
		bump(platforms, &platformOrder, string(it.Platform), published)
		bump(formats, &formatOrder, string(it.Format), published)
		d.RevisionsTotal += it.RevisionRound
		d.CaptionVersions += it.CaptionVersion

		var publishedAt *time.Time
		if published {
			publishedAt = it.PublishedAt
		}
		planned := it.PlannedDate
		for _, dl := range []struct{ due, done *time.Time }{{it.CopyDue, it.CopyDoneAt}, {it.DesignDue, it.DesignDoneAt}, {it.ReviewDue, it.ApprovedAt}, {&planned, publishedAt}} {
			if counted, met := deadlineHit(dl.due, dl.done, today); counted {
				d.DeadlinesTotal++
				if met {
					d.DeadlinesMet++
				}
			}
		}

		if !published {
			continue
		}
		p := ReportPost{ItemID: it.ID, Title: it.Title, Platform: it.Platform, Format: it.Format, PlannedDate: it.PlannedDate.Format(time.DateOnly), PublishedAt: it.PublishedAt, Permalink: it.Permalink}
		if m, ok := metrics[it.ID]; ok {
			p.HasMetrics = true
			p.Likes, p.Comments, p.Shares, p.Saves, p.Reach, p.Views = m.Likes, m.Comments, m.Shares, m.Saves, m.Reach, m.Views
			p.Engagements = m.Engagements()
			d.EngagementAvailable = true
			d.TotalEngagements += p.Engagements
			d.TotalReach += p.Reach
		}
		d.Posts = append(d.Posts, p)
	}
	for _, k := range platformOrder {
		d.PerPlatform = append(d.PerPlatform, *platforms[k])
	}
	for _, k := range formatOrder {
		d.PerFormat = append(d.PerFormat, *formats[k])
	}
	if d.DeadlinesTotal > 0 {
		d.OnTimeRate = round1(float64(d.DeadlinesMet) * 100 / float64(d.DeadlinesTotal))
	}
	if d.Planned > 0 {
		d.RevisionAvg = round1(float64(d.RevisionsTotal) / float64(d.Planned))
	}

	var measured []ReportPost
	for _, p := range d.Posts {
		if p.HasMetrics {
			measured = append(measured, p)
		}
	}
	sort.SliceStable(measured, func(i, j int) bool { return measured[i].Engagements > measured[j].Engagements })
	n := min(TopN, len(measured))
	d.Top = append(d.Top, measured[:n]...)
	// Bottom posts never repeat a top post.
	for i := len(measured) - 1; i >= n && len(d.Bottom) < TopN; i-- {
		d.Bottom = append(d.Bottom, measured[i])
	}
	return d
}
