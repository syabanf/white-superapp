package domain

import (
	"slices"
	"time"
)

// DashboardFilter narrows every dashboard widget. Empty fields match all;
// Stage is the candidate's current status.
type DashboardFilter struct {
	JobOpeningID string
	Department   string
	Source       string
	Stage        string
}

// Period is an inclusive range of WIB days with its instant bounds
// (ToTS exclusive).
type Period struct {
	From, To     time.Time // UTC midnight of the WIB day
	FromTS, ToTS time.Time
}

// Compare modes of the dashboard.
const (
	ComparePrevious = "previous"
	CompareYear     = "year"
)

// MaxDashboardDays caps the dashboard range.
const MaxDashboardDays = 3 * 366

// NewPeriod builds a period from two WIB days.
func NewPeriod(from, to time.Time) Period {
	start := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, Jakarta)
	end := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, Jakarta).AddDate(0, 0, 1)
	return Period{From: from, To: to, FromTS: start, ToTS: end}
}

// Days is the inclusive day count.
func (p Period) Days() int { return int(p.To.Sub(p.From).Hours()/24) + 1 }

// ComparePeriod is the comparison range: the same length ending the day
// before (previous) or the same days one year earlier (year).
func ComparePeriod(p Period, mode string) Period {
	if mode == CompareYear {
		return NewPeriod(p.From.AddDate(-1, 0, 0), p.To.AddDate(-1, 0, 0))
	}
	to := p.From.AddDate(0, 0, -1)
	return NewPeriod(to.AddDate(0, 0, -(p.Days()-1)), to)
}

// TrendBucket is one week of the dashboard: flows inside the week and
// pipeline snapshots at its end.
type TrendBucket struct {
	Start             string
	Applications      int
	Hires             int
	HireDays          float64
	OffersSent        int
	OffersAccepted    int
	OffersDeclined    int
	OffersExpired     int
	OffersNegotiating int
	InProcess         int
	OpenPositions     int
	OpenHeadcount     int
}

// Kpis are the headline numbers of a period.
type Kpis struct {
	OpenPositions  int
	OpenHeadcount  int
	Applicants     int
	InProcess      int
	OffersSent     int
	OffersAccepted int
	OffersDeclined int
	OffersExpired  int
	Hires          int
	AvgDaysToHire  *float64
	// Accepted over decided (accepted, declined, expired), 0..100.
	OfferAcceptanceRate *float64
}

// KpisFromTrend sums the flows and takes the snapshots of the last week.
func KpisFromTrend(trend []TrendBucket) Kpis {
	var k Kpis
	var hireDays float64
	for _, b := range trend {
		k.Applicants += b.Applications
		k.Hires += b.Hires
		hireDays += b.HireDays
		k.OffersSent += b.OffersSent
		k.OffersAccepted += b.OffersAccepted
		k.OffersDeclined += b.OffersDeclined
		k.OffersExpired += b.OffersExpired
	}
	if n := len(trend); n > 0 {
		last := trend[n-1]
		k.InProcess, k.OpenPositions, k.OpenHeadcount = last.InProcess, last.OpenPositions, last.OpenHeadcount
	}
	if k.Hires > 0 {
		k.AvgDaysToHire = round1(hireDays / float64(k.Hires))
	}
	if decided := k.OffersAccepted + k.OffersDeclined + k.OffersExpired; decided > 0 {
		k.OfferAcceptanceRate = round1(float64(k.OffersAccepted) * 100 / float64(decided))
	}
	return k
}

func round1(v float64) *float64 {
	r := float64(int64(v*10+0.5)) / 10
	return &r
}

// FunnelStatuses are the pipeline steps in order; their index is the rank
// of DashboardCohort.MaxRank.
var FunnelStatuses = []string{"applied", "screening", "psikotes", "interview", "offer", "hired"}

// ExitStatuses leave the funnel.
var ExitStatuses = []string{"rejected", "talent_pool", "archived"}

// CohortRow counts applications of a period by source, current status and
// furthest funnel rank reached.
type CohortRow struct {
	Source   string
	Status   string
	MaxRank  int
	N        int
	HireDays float64
}

// FunnelStage is one funnel step: how many reached it, sit in it now, and
// left the pipeline from it.
type FunnelStage struct {
	Stage   string
	Reached int
	Current int
	Exited  int
}

// Funnel folds cohort rows into the pipeline funnel.
func Funnel(rows []CohortRow) []FunnelStage {
	out := make([]FunnelStage, len(FunnelStatuses))
	for i, s := range FunnelStatuses {
		out[i].Stage = s
	}
	for _, r := range rows {
		rank := min(max(r.MaxRank, 0), len(out)-1)
		for k := 0; k <= rank; k++ {
			out[k].Reached += r.N
		}
		if r.Status == out[rank].Stage {
			out[rank].Current += r.N
		}
		if slices.Contains(ExitStatuses, r.Status) {
			out[rank].Exited += r.N
		}
	}
	return out
}

// PaidSources are channels that cost money (listings, booths, fees).
var PaidSources = []string{"jobstreet", "headhunter", "jobfair"}

// SourceStat is the yield of one channel.
type SourceStat struct {
	Source        string
	Paid          bool
	Applications  int
	Screened      int
	Interviewed   int
	Offered       int
	Hires         int
	AvgDaysToHire *float64
}

// Sources folds cohort rows per channel, biggest first.
func Sources(rows []CohortRow) []SourceStat {
	idx := map[string]int{}
	var out []SourceStat
	days := map[string]float64{}
	for _, r := range rows {
		i, ok := idx[r.Source]
		if !ok {
			i = len(out)
			idx[r.Source] = i
			out = append(out, SourceStat{Source: r.Source, Paid: slices.Contains(PaidSources, r.Source)})
		}
		s := &out[i]
		s.Applications += r.N
		if r.MaxRank >= 1 {
			s.Screened += r.N
		}
		if r.MaxRank >= 3 {
			s.Interviewed += r.N
		}
		if r.MaxRank >= 4 {
			s.Offered += r.N
		}
		if r.Status == "hired" {
			s.Hires += r.N
			days[r.Source] += r.HireDays
		}
	}
	for i := range out {
		if out[i].Hires > 0 {
			out[i].AvgDaysToHire = round1(days[out[i].Source] / float64(out[i].Hires))
		}
	}
	slices.SortStableFunc(out, func(a, b SourceStat) int {
		if a.Applications != b.Applications {
			return b.Applications - a.Applications
		}
		if a.Source < b.Source {
			return -1
		}
		return 1
	})
	return out
}

// StageDuration is the spread of days spent in a stage.
type StageDuration struct {
	Stage                           string
	N                               int
	Avg, Min, P25, Median, P75, Max float64
}

// OpeningStageCount is applications of an opening in one current status.
type OpeningStageCount struct {
	JobOpeningID, Title, Status string
	N                           int
}

// StuckCandidate sits in an active stage longer than the threshold.
type StuckCandidate struct {
	ID, FullName, JobOpeningID, Title, Status string
	EnteredAt                                 time.Time
}

// SourceWeek is applications of one channel in a week.
type SourceWeek struct {
	Start, Source string
	N             int
}

// OpeningStat is one opening on the dashboard.
type OpeningStat struct {
	ID, Title, Department, Status string
	Headcount                     int
	OpenedAt                      time.Time
	ClosedAt                      *time.Time
	ClosingDate                   *string
	ApplicantsTotal               int
	ApplicantsPeriod              int
	Stages                        []Count
	Hired                         int
	// Days from publication to the hire that filled the headcount.
	FillDays *float64
}

// ScoreBucket counts scored psikotes tests in a 10-point band.
type ScoreBucket struct {
	Instrument string
	Bucket, N  int
	AvgScore   float64
}

// CriterionScore is the average scorecard rating of a criterion.
type CriterionScore struct {
	Type, Criterion string
	N               int
	Avg             float64
}

// InterviewerLoad is the interviews of one interviewer.
type InterviewerLoad struct {
	Name, EmployeeID                                       string
	Scheduled, Completed, Cancelled, Proceed, Pool, Reject int
	AvgScore                                               *float64
}

// OfferRow is the latest offer version of a candidate.
type OfferRow struct {
	CandidateID, FullName, JobOpeningID, Title, Status string
	BaseSalary                                         Money
	SalaryMin, SalaryMax                               *Money
	ExpectedSalary                                     *int64
	SentAt                                             time.Time
	RespondedAt                                        *time.Time
	Versions                                           int
}

// DashboardDetail is everything of a period except the weekly trend.
type DashboardDetail struct {
	Cohort         []CohortRow
	StageDurations []StageDuration
	OpeningStages  []OpeningStageCount
	Stuck          []StuckCandidate
	SourceTrend    []SourceWeek
	Openings       []OpeningStat
	PsikotesStatus []Count
	PsikotesScores []ScoreBucket
	AiStatus       []Count
	Scorecards     []CriterionScore
	Interviewers   []InterviewerLoad
	Offers         []OfferRow
}

// StuckAfterDays is how long a candidate may sit in an active stage before
// the dashboard flags it.
const StuckAfterDays = 14

// Dashboard is the recruitment analytics page of a period and its comparison.
type Dashboard struct {
	Current, Previous Period
	Compare           string
	// Now is when the dashboard was computed (days open, days in stage).
	Now              time.Time
	Kpis, PrevKpis   Kpis
	Trend, PrevTrend []TrendBucket
	Funnel           []FunnelStage
	Sources          []SourceStat
	Detail           *DashboardDetail
}
