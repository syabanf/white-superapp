package domain

import (
	"context"
	"time"
)

// DashFilter narrows every dashboard aggregate. Empty fields match all.
type DashFilter struct {
	PipelineID  string
	OwnerUserID string
	ServiceLine string
	Source      string
}

// DashPeriod is the resolved dashboard window. From/To are inclusive WIB
// days; Months is the monthly trend window and CompareMonths the same
// months shifted by the comparison.
type DashPeriod struct {
	From, To               time.Time
	Compare                string
	CompareFrom, CompareTo time.Time
	Months, CompareMonths  []string
}

const (
	CompareYear     = "year"
	ComparePrevious = "previous"
	dashMinMonths   = 12
	dashMaxDays     = 3 * 366
)

// ResolveDashPeriod applies the defaults (current month up to today) and
// the comparison rule: "previous" is the same number of days ending the day
// before From, "year" the same days one year earlier. badField names the
// offending input when it is invalid.
func ResolveDashPeriod(fromS, toS, compare string, now time.Time) (p DashPeriod, badField string) {
	today := dayStart(now.In(Jakarta))
	p.From = time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, Jakarta)
	p.To = today
	if fromS != "" {
		t, err := time.ParseInLocation("2006-01-02", fromS, Jakarta)
		if err != nil {
			return p, "from"
		}
		p.From = t
	}
	if toS != "" {
		t, err := time.ParseInLocation("2006-01-02", toS, Jakarta)
		if err != nil {
			return p, "to"
		}
		p.To = t
	}
	if p.To.Before(p.From) || p.To.Sub(p.From) > dashMaxDays*24*time.Hour {
		return p, "to"
	}
	p.Compare = ComparePrevious
	if compare == CompareYear {
		p.Compare = CompareYear
		p.CompareFrom, p.CompareTo = p.From.AddDate(-1, 0, 0), p.To.AddDate(-1, 0, 0)
	} else {
		days := DaysBetween(p.From, p.To) + 1
		p.CompareTo = p.From.AddDate(0, 0, -1)
		p.CompareFrom = p.CompareTo.AddDate(0, 0, -(days - 1))
	}
	first := monthStart(p.From)
	last := monthStart(p.To)
	if earliest := last.AddDate(0, -(dashMinMonths - 1), 0); earliest.Before(first) {
		first = earliest
	}
	shift := 12
	for m := first; !m.After(last); m = m.AddDate(0, 1, 0) {
		p.Months = append(p.Months, m.Format("2006-01"))
	}
	if p.Compare == ComparePrevious {
		shift = len(p.Months)
	}
	for m := first.AddDate(0, -shift, 0); len(p.CompareMonths) < len(p.Months); m = m.AddDate(0, 1, 0) {
		p.CompareMonths = append(p.CompareMonths, m.Format("2006-01"))
	}
	return p, ""
}

// Bounds is the period as [start, end) instants.
func (p DashPeriod) Bounds() (time.Time, time.Time) { return p.From, p.To.AddDate(0, 0, 1) }

// CompareBounds is the comparison period as [start, end) instants.
func (p DashPeriod) CompareBounds() (time.Time, time.Time) {
	return p.CompareFrom, p.CompareTo.AddDate(0, 0, 1)
}

// MonthBounds spans whole months as [start, end) instants.
func MonthBounds(months []string) (time.Time, time.Time) {
	if len(months) == 0 {
		return time.Time{}, time.Time{}
	}
	first, _ := time.ParseInLocation("2006-01", months[0], Jakarta)
	last, _ := time.ParseInLocation("2006-01", months[len(months)-1], Jakarta)
	return first, last.AddDate(0, 1, 0)
}

// MonthEnds returns, per month, the instant its pipeline snapshot is taken:
// the month's end, or now for the running month.
func MonthEnds(months []string, now time.Time) []time.Time {
	out := make([]time.Time, len(months))
	for i, m := range months {
		_, end := MonthBounds([]string{m})
		out[i] = minTime(end, now)
	}
	return out
}

// DaysBetween counts whole days from a to b (WIB calendar days).
func DaysBetween(a, b time.Time) int {
	return int(dayStart(b.In(Jakarta)).Sub(dayStart(a.In(Jakarta))).Hours()/24 + 0.5)
}

func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func minTime(a, b time.Time) time.Time {
	if b.Before(a) {
		return b
	}
	return a
}

// DashTarget is one monthly target row (UserID "" = company).
type DashTarget struct {
	UserID      string
	PeriodMonth string
	TargetValue float64
}

// MonthTarget is the target for one month: the owner's when owner is set,
// otherwise the company target, falling back to the sum of personal ones.
func MonthTarget(rows []DashTarget, month, owner string) float64 {
	var company, personal float64
	hasCompany := false
	for _, r := range rows {
		if r.PeriodMonth != month {
			continue
		}
		switch {
		case owner != "" && r.UserID == owner:
			return r.TargetValue
		case r.UserID == "":
			company, hasCompany = r.TargetValue, true
		default:
			personal += r.TargetValue
		}
	}
	if owner != "" {
		return 0
	}
	if hasCompany {
		return company
	}
	return personal
}

// ProratedTarget sums the monthly targets over [from, to] (inclusive days),
// counting each month by the share of its days inside the range.
func ProratedTarget(rows []DashTarget, from, to time.Time, owner string) float64 {
	total := 0.0
	for m := monthStart(from); !m.After(to); m = m.AddDate(0, 1, 0) {
		end := m.AddDate(0, 1, -1)
		lo, hi := from, to
		if m.After(lo) {
			lo = m
		}
		if end.Before(hi) {
			hi = end
		}
		share := float64(DaysBetween(lo, hi)+1) / float64(end.Day())
		total += MonthTarget(rows, m.Format("2006-01"), owner) * share
	}
	return total
}

// StageAgeBucket buckets days in the current stage.
func StageAgeBucket(days int) string {
	switch {
	case days <= 7:
		return "0-7"
	case days <= 14:
		return "8-14"
	case days <= 30:
		return "15-30"
	default:
		return "31+"
	}
}

// StageAgeBuckets is the display order of StageAgeBucket.
var StageAgeBuckets = []string{"0-7", "8-14", "15-30", "31+"}

// ReceivableBucket buckets an unpaid invoice by days past its due date.
func ReceivableBucket(daysOverdue int) string {
	switch {
	case daysOverdue <= 0:
		return "current"
	case daysOverdue <= 30:
		return "1-30"
	case daysOverdue <= 60:
		return "31-60"
	case daysOverdue <= 90:
		return "61-90"
	default:
		return "90+"
	}
}

// ReceivableBuckets is the display order of ReceivableBucket.
var ReceivableBuckets = []string{"current", "1-30", "31-60", "61-90", "90+"}

// Pct is a/b as a percentage, 0 when b is 0.
func Pct(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b * 100
}

// Div is a/b, 0 when b is 0.
func Div(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// ── results ────────────────────────────────────────────────────

type DashKpi struct {
	Key             string
	Value, Previous float64
	Spark           []float64
}

type DashSeries struct {
	Key              string
	Values, Previous []float64
}

// DashSlice is a breakdown row; the RPC documents what each measure holds.
type DashSlice struct {
	Key, Label, Group string
	Value             float64
	Count             int
	Extra             float64
}

type DashDeal struct {
	ID, Title, AccountName, StageID, StageName, OwnerUserID, OwnerName string
	Value                                                              float64
	DaysInStage, ThresholdDays                                         int
}

type DashMember struct {
	UserID, Name                                     string
	OpenDeals                                        int
	PipelineValue                                    float64
	WonDeals                                         int
	WonValue                                         float64
	LostDeals                                        int
	WinRate                                          float64
	Activities, ActivitiesDone                       int
	TargetValue, AttainmentPercent, PreviousWonValue float64
}

type DashSummary struct {
	Period DashPeriod
	Kpis   []DashKpi
	Trend  []DashSeries
	Stages []DashSlice
}

type DashPipeline struct {
	Period     DashPeriod
	PipelineID string
	Kpis       []DashKpi
	Funnel     []DashSlice
	StageLines []DashSlice
	Aging      []DashSlice
	StageDays  []DashSlice
	Stuck      []DashDeal
}

type DashSales struct {
	Period        DashPeriod
	Kpis          []DashKpi
	Monthly       []DashSeries
	ServiceLines  []DashSlice
	Industries    []DashSlice
	LostReasons   []DashSlice
	QuoteStatuses []DashSlice
	Sources       []DashSlice
}

type DashTeam struct {
	Period     DashPeriod
	Kpis       []DashKpi
	Members    []DashMember
	WonByOwner []DashSeries
}

type DashLeads struct {
	Period         DashPeriod
	Kpis           []DashKpi
	MonthlySources []DashSlice
	Sources        []DashSlice
	Scores         []DashSlice
	Response       []DashSeries
	Channels       []DashSlice
	Forms          []DashSlice
}

type DashBilling struct {
	Period  DashPeriod
	Kpis    []DashKpi
	Monthly []DashSeries
	Aging   []DashSlice
	Overdue []DashReceivable
	Terms   []DashTermRow
}

// ── repository rows ────────────────────────────────────────────

type DashClosedMonth struct {
	Month        string
	WonCount     int32
	WonValue     float64
	LostCount    int32
	LostValue    float64
	WonCycleDays float64
}

type DashOwnerMonth struct {
	Month       string
	OwnerUserID string
	WonValue    float64
}

type DashOpenSnapshot struct {
	OpenCount     int32
	OpenValue     float64
	WeightedValue float64
}

type DashOpenDeal struct {
	ID             string
	Title          string
	AccountName    string
	StageID        string
	StageName      string
	SortOrder      int32
	Probability    int32
	ThresholdDays  int32
	OwnerUserID    string
	Value          float64
	ServiceLines   []string
	EnteredStageAt time.Time
	CreatedAt      time.Time
}

type DashStageCount struct {
	ID        string
	Name      string
	SortOrder int32
	Deals     int32
	Value     float64
}

type DashStageDay struct {
	ID        string
	Name      string
	SortOrder int32
	Exits     int32
	AvgDays   float64
}

type DashBreakdown struct {
	Dim   string
	Key   string
	Label string
	Deals int32
	Value float64
}

type DashQuoteStatus struct {
	Status      string
	Quotes      int32
	Total       float64
	DiscountSum float64
	DecideDays  float64
	Decided     int32
}

type DashOwnerRow struct {
	OwnerUserID      string
	OpenDeals        int32
	PipelineValue    float64
	WonDeals         int32
	WonValue         float64
	LostDeals        int32
	Activities       int32
	ActivitiesDone   int32
	PreviousWonValue float64
}

type DashLeadMonth struct {
	Month  string
	Source string
	Leads  int32
}

type DashLeadSource struct {
	Source    string
	Leads     int32
	Qualified int32
	Converted int32
	Won       int32
}

type DashLeadScore struct {
	Bucket int32
	Leads  int32
}

type DashConversationMonth struct {
	Month           string
	Channel         string
	Conversations   int32
	Responded       int32
	ResponseSeconds float64
	Breaches        int32
}

type DashFormCount struct {
	ID       string
	Name     string
	Accepted int32
	Rejected int32
}

type DashBillingMonth struct {
	Kind    string
	Month   string
	N       int32
	Amount  float64
	PayDays float64
}

type DashReceivable struct {
	ID            string
	InvoiceNumber string
	DealID        string
	DealTitle     string
	AccountName   string
	Amount        float64
	DueDate       *time.Time
	Paid          float64
	DaysOverdue   int
}

type DashTermRow struct {
	DealID        string
	DealTitle     string
	AccountName   string
	ContractValue float64
	Terms         int32
	TermsInvoiced int32
	Invoiced      float64
	Paid          float64
}

// DashboardRepo reads the dashboard aggregates; times are [from, to).
type DashboardRepo interface {
	DashClosedMonthly(ctx context.Context, from, to time.Time, f DashFilter) ([]DashClosedMonth, error)
	DashWonByOwnerMonthly(ctx context.Context, from, to time.Time, f DashFilter) ([]DashOwnerMonth, error)
	DashOpenAsOf(ctx context.Context, at time.Time, f DashFilter) (DashOpenSnapshot, error)
	DashOpenDeals(ctx context.Context, f DashFilter) ([]DashOpenDeal, error)
	DashFunnel(ctx context.Context, pipelineID string, from, to time.Time, f DashFilter) ([]DashStageCount, error)
	DashStageDays(ctx context.Context, from, to time.Time, f DashFilter) ([]DashStageDay, error)
	DashClosedBreakdown(ctx context.Context, from, to time.Time, f DashFilter) ([]DashBreakdown, error)
	DashQuotes(ctx context.Context, from, to time.Time, f DashFilter) ([]DashQuoteStatus, error)
	DashOwners(ctx context.Context, from, to, prevFrom, prevTo time.Time, f DashFilter) ([]DashOwnerRow, error)
	DashTargets(ctx context.Context, fromMonth, toMonth string) ([]DashTarget, error)
	DashLeadsMonthly(ctx context.Context, from, to time.Time, f DashFilter) ([]DashLeadMonth, error)
	DashLeadSources(ctx context.Context, from, to time.Time, f DashFilter) ([]DashLeadSource, error)
	DashLeadScores(ctx context.Context, from, to time.Time, f DashFilter) ([]DashLeadScore, error)
	DashConversations(ctx context.Context, from, to time.Time, f DashFilter) ([]DashConversationMonth, error)
	DashFormSubmissions(ctx context.Context, from, to time.Time) ([]DashFormCount, error)
	DashBillingMonthly(ctx context.Context, from, to time.Time, f DashFilter) ([]DashBillingMonth, error)
	DashReceivables(ctx context.Context, asOf time.Time, f DashFilter) ([]DashReceivable, error)
	DashTermProgress(ctx context.Context, f DashFilter) ([]DashTermRow, error)
}
