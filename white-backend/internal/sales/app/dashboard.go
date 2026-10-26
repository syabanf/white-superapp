package app

import (
	"context"
	"sort"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// Dashboard builds the sales dashboard tabs. Every tab shares the period,
// comparison and filters in DashQuery.
type Dashboard struct{ *Deps }

func NewDashboard(d *Deps) *Dashboard { return &Dashboard{d} }

type DashQuery struct {
	From, To, Compare string
	Filter            domain.DashFilter
}

// dashCtx carries the resolved windows of one request.
type dashCtx struct {
	p                        domain.DashPeriod
	f                        domain.DashFilter
	now                      time.Time
	from, to, cfrom, cto     time.Time // period and comparison, [from, to)
	mfrom, mto, cmfrom, cmto time.Time // trend months and their comparison
}

func (d *Dashboard) begin(q DashQuery) (*dashCtx, error) {
	now := d.now()
	p, bad := domain.ResolveDashPeriod(q.From, q.To, q.Compare, now)
	if bad != "" {
		return nil, apperr.Field(bad, "Periode tidak valid")
	}
	c := &dashCtx{p: p, f: q.Filter, now: now}
	c.from, c.to = p.Bounds()
	c.cfrom, c.cto = p.CompareBounds()
	c.mfrom, c.mto = domain.MonthBounds(p.Months)
	c.cmfrom, c.cmto = domain.MonthBounds(p.CompareMonths)
	return c, nil
}

// periodEnd is the instant a period's pipeline snapshot is taken.
func (c *dashCtx) periodEnd(end time.Time) time.Time {
	if end.After(c.now) {
		return c.now
	}
	return end
}

// byMonth places month-keyed values on the trend axis.
func byMonth(months []string, pick func(month string) float64) []float64 {
	out := make([]float64, len(months))
	for i, m := range months {
		out[i] = pick(m)
	}
	return out
}

type closedTotals struct {
	won, lost                  float64
	wonValue, lostValue, cycle float64
}

func sumClosed(rows []domain.DashClosedMonth) closedTotals {
	var t closedTotals
	for _, r := range rows {
		t.won += float64(r.WonCount)
		t.lost += float64(r.LostCount)
		t.wonValue += r.WonValue
		t.lostValue += r.LostValue
		t.cycle += r.WonCycleDays
	}
	return t
}

func (t closedTotals) winRate() float64   { return domain.Pct(t.won, t.won+t.lost) }
func (t closedTotals) avgDeal() float64   { return domain.Div(t.wonValue, t.won) }
func (t closedTotals) cycleDays() float64 { return domain.Div(t.cycle, t.won) }

// closedByMonth indexes monthly closed rows.
func closedByMonth(rows []domain.DashClosedMonth) map[string]closedTotals {
	out := map[string]closedTotals{}
	for _, r := range rows {
		out[r.Month] = sumClosed([]domain.DashClosedMonth{r})
	}
	return out
}

// closed loads the closed-deal totals of the period, the comparison and both
// trend windows.
func (d *Dashboard) closed(ctx context.Context, c *dashCtx) (cur, prev closedTotals, trend, ctrend map[string]closedTotals, err error) {
	var rows [4][]domain.DashClosedMonth
	for i, w := range [4][2]time.Time{{c.from, c.to}, {c.cfrom, c.cto}, {c.mfrom, c.mto}, {c.cmfrom, c.cmto}} {
		if rows[i], err = d.Store.DashClosedMonthly(ctx, w[0], w[1], c.f); err != nil {
			return
		}
	}
	return sumClosed(rows[0]), sumClosed(rows[1]), closedByMonth(rows[2]), closedByMonth(rows[3]), nil
}

type snapshots struct {
	cur, prev domain.DashOpenSnapshot
	months    []domain.DashOpenSnapshot
}

// snapshots reads the open pipeline at the period end, the comparison end
// and every trend month end.
func (d *Dashboard) snapshots(ctx context.Context, c *dashCtx) (snapshots, error) {
	var s snapshots
	var err error
	if s.cur, err = d.Store.DashOpenAsOf(ctx, c.periodEnd(c.to), c.f); err != nil {
		return s, err
	}
	if s.prev, err = d.Store.DashOpenAsOf(ctx, c.periodEnd(c.cto), c.f); err != nil {
		return s, err
	}
	for _, at := range domain.MonthEnds(c.p.Months, c.now) {
		snap, err := d.Store.DashOpenAsOf(ctx, at, c.f)
		if err != nil {
			return s, err
		}
		s.months = append(s.months, snap)
	}
	return s, nil
}

func (s snapshots) spark(pick func(domain.DashOpenSnapshot) float64) []float64 {
	out := make([]float64, len(s.months))
	for i, m := range s.months {
		out[i] = pick(m)
	}
	return out
}

func openValue(s domain.DashOpenSnapshot) float64    { return s.OpenValue }
func openWeighted(s domain.DashOpenSnapshot) float64 { return s.WeightedValue }
func openCount(s domain.DashOpenSnapshot) float64    { return float64(s.OpenCount) }

func (d *Dashboard) ownerNames(ctx context.Context) map[string]string {
	names := map[string]string{}
	if d.Directory == nil {
		return names
	}
	users, err := d.Directory.ListUsers(ctx)
	if err != nil {
		d.warn("list users failed", "err", err)
	}
	for _, u := range users {
		names[u.ID] = u.Name
	}
	return names
}

// openStages lists the open stages in pipeline order (filtered pipeline only).
func (d *Dashboard) openStages(ctx context.Context, pipelineID string) ([]domain.Stage, error) {
	pls, err := d.Store.ListPipelines(ctx)
	if err != nil {
		return nil, err
	}
	var out []domain.Stage
	for _, p := range pls {
		if pipelineID != "" && p.ID != pipelineID {
			continue
		}
		for _, s := range p.Stages {
			if s.IsActive && !s.IsWon && !s.IsLost {
				out = append(out, s)
			}
		}
	}
	return out, nil
}

// pipelineFor is the filtered pipeline, else the default one.
func (d *Dashboard) pipelineFor(ctx context.Context, id string) (string, error) {
	if id != "" {
		return id, nil
	}
	pls, err := d.Store.ListPipelines(ctx)
	if err != nil || len(pls) == 0 {
		return "", err
	}
	for _, p := range pls {
		if p.IsDefault {
			return p.ID, nil
		}
	}
	return pls[0].ID, nil
}

func sortSlices(s []domain.DashSlice) {
	sort.SliceStable(s, func(i, j int) bool {
		if s[i].Value != s[j].Value {
			return s[i].Value > s[j].Value
		}
		return s[i].Count > s[j].Count
	})
}

// ── Ringkasan ──────────────────────────────────────────────────

func (d *Dashboard) Summary(ctx context.Context, q DashQuery) (*domain.DashSummary, error) {
	c, err := d.begin(q)
	if err != nil {
		return nil, err
	}
	cur, prev, trend, ctrend, err := d.closed(ctx, c)
	if err != nil {
		return nil, err
	}
	snap, err := d.snapshots(ctx, c)
	if err != nil {
		return nil, err
	}
	targets, err := d.Store.DashTargets(ctx, c.p.Months[0], c.p.Months[len(c.p.Months)-1])
	if err != nil {
		return nil, err
	}
	months := c.p.Months
	sparkClosed := func(pick func(closedTotals) float64) []float64 {
		return byMonth(months, func(m string) float64 { return pick(trend[m]) })
	}
	out := &domain.DashSummary{Period: c.p}
	out.Kpis = []domain.DashKpi{
		{Key: "pipeline_value", Value: snap.cur.OpenValue, Previous: snap.prev.OpenValue, Spark: snap.spark(openValue)},
		{Key: "weighted_value", Value: snap.cur.WeightedValue, Previous: snap.prev.WeightedValue, Spark: snap.spark(openWeighted)},
		{Key: "won_value", Value: cur.wonValue, Previous: prev.wonValue, Spark: sparkClosed(func(t closedTotals) float64 { return t.wonValue })},
		{Key: "win_rate", Value: cur.winRate(), Previous: prev.winRate(), Spark: sparkClosed(closedTotals.winRate)},
		{Key: "avg_deal", Value: cur.avgDeal(), Previous: prev.avgDeal(), Spark: sparkClosed(closedTotals.avgDeal)},
		{Key: "cycle_days", Value: cur.cycleDays(), Previous: prev.cycleDays(), Spark: sparkClosed(closedTotals.cycleDays)},
		{Key: "open_deals", Value: float64(snap.cur.OpenCount), Previous: float64(snap.prev.OpenCount), Spark: snap.spark(openCount)},
		{Key: "won_deals", Value: cur.won, Previous: prev.won, Spark: sparkClosed(func(t closedTotals) float64 { return t.won })},
	}
	out.Trend = []domain.DashSeries{
		{Key: "won_value", Values: sparkClosed(func(t closedTotals) float64 { return t.wonValue }),
			Previous: byMonth(c.p.CompareMonths, func(m string) float64 { return ctrend[m].wonValue })},
		{Key: "target", Values: byMonth(months, func(m string) float64 { return domain.MonthTarget(targets, m, c.f.OwnerUserID) })},
	}
	open, err := d.Store.DashOpenDeals(ctx, c.f)
	if err != nil {
		return nil, err
	}
	stages, err := d.openStages(ctx, c.f.PipelineID)
	if err != nil {
		return nil, err
	}
	for _, s := range stages {
		sl := domain.DashSlice{Key: s.ID, Label: s.Name}
		for _, o := range open {
			if o.StageID == s.ID {
				sl.Count++
				sl.Value += o.Value
				sl.Extra += o.Value * float64(o.Probability) / 100
			}
		}
		out.Stages = append(out.Stages, sl)
	}
	return out, nil
}

// ── Pipeline ───────────────────────────────────────────────────

func (d *Dashboard) Pipeline(ctx context.Context, q DashQuery) (*domain.DashPipeline, error) {
	c, err := d.begin(q)
	if err != nil {
		return nil, err
	}
	pipelineID, err := d.pipelineFor(ctx, c.f.PipelineID)
	if err != nil {
		return nil, err
	}
	out := &domain.DashPipeline{Period: c.p, PipelineID: pipelineID}
	cur, prev, trend, _, err := d.closed(ctx, c)
	if err != nil {
		return nil, err
	}
	snap, err := d.snapshots(ctx, c)
	if err != nil {
		return nil, err
	}
	open, err := d.Store.DashOpenDeals(ctx, c.f)
	if err != nil {
		return nil, err
	}
	names := d.ownerNames(ctx)
	stuck, ageSum := 0, 0.0
	lines := map[[2]string]*domain.DashSlice{}
	aging := map[[2]string]*domain.DashSlice{}
	for _, o := range open {
		days := domain.DaysBetween(o.EnteredStageAt, c.now)
		ageSum += float64(domain.DaysBetween(o.CreatedAt, c.now))
		if o.ThresholdDays > 0 && days > int(o.ThresholdDays) {
			stuck++
			out.Stuck = append(out.Stuck, domain.DashDeal{ID: o.ID, Title: o.Title, AccountName: o.AccountName, StageID: o.StageID, StageName: o.StageName,
				OwnerUserID: o.OwnerUserID, OwnerName: names[o.OwnerUserID], Value: o.Value, DaysInStage: days, ThresholdDays: int(o.ThresholdDays)})
		}
		ls := o.ServiceLines
		if len(ls) == 0 {
			ls = []string{"lainnya"}
		}
		for _, l := range ls {
			k := [2]string{o.StageID, l}
			if lines[k] == nil {
				lines[k] = &domain.DashSlice{Key: o.StageID, Label: o.StageName, Group: l}
			}
			lines[k].Value += o.Value / float64(len(ls))
			lines[k].Count++
		}
		k := [2]string{o.StageID, domain.StageAgeBucket(days)}
		if aging[k] == nil {
			aging[k] = &domain.DashSlice{Key: o.StageID, Label: o.StageName, Group: k[1]}
		}
		aging[k].Count++
		aging[k].Value += o.Value
	}
	sort.Slice(out.Stuck, func(i, j int) bool {
		ri := float64(out.Stuck[i].DaysInStage) / float64(out.Stuck[i].ThresholdDays)
		rj := float64(out.Stuck[j].DaysInStage) / float64(out.Stuck[j].ThresholdDays)
		return ri > rj
	})
	if len(out.Stuck) > 20 {
		out.Stuck = out.Stuck[:20]
	}
	stages, err := d.openStages(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	for _, s := range stages {
		for _, l := range domain.ServiceLines {
			if v := lines[[2]string{s.ID, l}]; v != nil {
				out.StageLines = append(out.StageLines, *v)
			}
		}
		if v := lines[[2]string{s.ID, "lainnya"}]; v != nil {
			out.StageLines = append(out.StageLines, *v)
		}
		for _, b := range domain.StageAgeBuckets {
			if v := aging[[2]string{s.ID, b}]; v != nil {
				out.Aging = append(out.Aging, *v)
			}
		}
	}
	// Velocity: open deals x win rate x average deal / cycle days, rupiah per day.
	velocity := func(open float64, t closedTotals) float64 {
		return domain.Div(open*t.winRate()/100*t.avgDeal(), t.cycleDays())
	}
	monthVelocity := make([]float64, len(c.p.Months))
	for i, m := range c.p.Months {
		monthVelocity[i] = velocity(float64(snap.months[i].OpenCount), trend[m])
	}
	out.Kpis = []domain.DashKpi{
		{Key: "open_deals", Value: float64(len(open)), Previous: float64(snap.prev.OpenCount), Spark: snap.spark(openCount)},
		{Key: "pipeline_value", Value: snap.cur.OpenValue, Previous: snap.prev.OpenValue, Spark: snap.spark(openValue)},
		{Key: "weighted_value", Value: snap.cur.WeightedValue, Previous: snap.prev.WeightedValue, Spark: snap.spark(openWeighted)},
		{Key: "stuck_deals", Value: float64(stuck)},
		{Key: "avg_age_days", Value: domain.Div(ageSum, float64(len(open)))},
		{Key: "velocity", Value: velocity(float64(snap.cur.OpenCount), cur), Previous: velocity(float64(snap.prev.OpenCount), prev), Spark: monthVelocity},
	}
	funnel, err := d.Store.DashFunnel(ctx, pipelineID, c.from, c.to, c.f)
	if err != nil {
		return nil, err
	}
	for _, s := range funnel {
		out.Funnel = append(out.Funnel, domain.DashSlice{Key: s.ID, Label: s.Name, Count: int(s.Deals), Value: s.Value})
	}
	days, err := d.Store.DashStageDays(ctx, c.from, c.to, c.f)
	if err != nil {
		return nil, err
	}
	for _, s := range days {
		out.StageDays = append(out.StageDays, domain.DashSlice{Key: s.ID, Label: s.Name, Count: int(s.Exits), Value: s.AvgDays})
	}
	return out, nil
}

// ── Penjualan ──────────────────────────────────────────────────

type quoteTotals struct{ sent, accepted, rejected, discount, acceptDays, acceptDecided float64 }

func sumQuotes(rows []domain.DashQuoteStatus) quoteTotals {
	var t quoteTotals
	for _, r := range rows {
		t.sent += float64(r.Quotes)
		t.discount += r.DiscountSum
		switch r.Status {
		case domain.QuoteAccepted:
			t.accepted += float64(r.Quotes)
			t.acceptDays += r.DecideDays
			t.acceptDecided += float64(r.Decided)
		case domain.QuoteRejected:
			t.rejected += float64(r.Quotes)
		}
	}
	return t
}

func (t quoteTotals) acceptance() float64   { return domain.Pct(t.accepted, t.accepted+t.rejected) }
func (t quoteTotals) avgDiscount() float64  { return domain.Div(t.discount, t.sent) }
func (t quoteTotals) timeToAccept() float64 { return domain.Div(t.acceptDays, t.acceptDecided) }

func (d *Dashboard) Sales(ctx context.Context, q DashQuery) (*domain.DashSales, error) {
	c, err := d.begin(q)
	if err != nil {
		return nil, err
	}
	cur, prev, trend, ctrend, err := d.closed(ctx, c)
	if err != nil {
		return nil, err
	}
	qc, err := d.Store.DashQuotes(ctx, c.from, c.to, c.f)
	if err != nil {
		return nil, err
	}
	qp, err := d.Store.DashQuotes(ctx, c.cfrom, c.cto, c.f)
	if err != nil {
		return nil, err
	}
	qcur, qprev := sumQuotes(qc), sumQuotes(qp)
	series := func(key string, pick func(closedTotals) float64) domain.DashSeries {
		return domain.DashSeries{Key: key,
			Values:   byMonth(c.p.Months, func(m string) float64 { return pick(trend[m]) }),
			Previous: byMonth(c.p.CompareMonths, func(m string) float64 { return pick(ctrend[m]) })}
	}
	wonValue := func(t closedTotals) float64 { return t.wonValue }
	out := &domain.DashSales{Period: c.p}
	out.Monthly = []domain.DashSeries{
		series("won_count", func(t closedTotals) float64 { return t.won }),
		series("lost_count", func(t closedTotals) float64 { return t.lost }),
		series("won_value", wonValue),
		series("lost_value", func(t closedTotals) float64 { return t.lostValue }),
		series("win_rate", closedTotals.winRate),
	}
	out.Kpis = []domain.DashKpi{
		{Key: "won_value", Value: cur.wonValue, Previous: prev.wonValue, Spark: out.Monthly[2].Values},
		{Key: "won_deals", Value: cur.won, Previous: prev.won, Spark: out.Monthly[0].Values},
		{Key: "lost_deals", Value: cur.lost, Previous: prev.lost, Spark: out.Monthly[1].Values},
		{Key: "win_rate", Value: cur.winRate(), Previous: prev.winRate(), Spark: out.Monthly[4].Values},
		{Key: "avg_discount", Value: qcur.avgDiscount(), Previous: qprev.avgDiscount()},
		{Key: "quote_acceptance", Value: qcur.acceptance(), Previous: qprev.acceptance()},
		{Key: "time_to_accept_days", Value: qcur.timeToAccept(), Previous: qprev.timeToAccept()},
		{Key: "quotes_sent", Value: qcur.sent, Previous: qprev.sent},
	}
	for _, r := range qc {
		out.QuoteStatuses = append(out.QuoteStatuses, domain.DashSlice{Key: r.Status, Count: int(r.Quotes), Value: r.Total})
	}
	rows, err := d.Store.DashClosedBreakdown(ctx, c.from, c.to, c.f)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		label := r.Label
		if label == "" {
			label = r.Key
		}
		sl := domain.DashSlice{Key: r.Key, Label: label, Value: r.Value, Count: int(r.Deals)}
		switch r.Dim {
		case "line":
			sl.Label = domain.ServiceLineLabel(r.Key)
			out.ServiceLines = append(out.ServiceLines, sl)
		case "industry":
			out.Industries = append(out.Industries, sl)
		case "source":
			out.Sources = append(out.Sources, sl)
		case "lost_reason":
			out.LostReasons = append(out.LostReasons, sl)
		}
	}
	for _, s := range [][]domain.DashSlice{out.ServiceLines, out.Industries, out.Sources} {
		sortSlices(s)
	}
	sort.SliceStable(out.LostReasons, func(i, j int) bool { return out.LostReasons[i].Count > out.LostReasons[j].Count })
	return out, nil
}

// ── Tim ────────────────────────────────────────────────────────

func (d *Dashboard) Team(ctx context.Context, q DashQuery) (*domain.DashTeam, error) {
	c, err := d.begin(q)
	if err != nil {
		return nil, err
	}
	rows, err := d.Store.DashOwners(ctx, c.from, c.to, c.cfrom, c.cto, c.f)
	if err != nil {
		return nil, err
	}
	firstMonth := minString(c.p.Months[0], c.p.CompareMonths[0], c.p.CompareFrom.Format("2006-01"))
	targets, err := d.Store.DashTargets(ctx, firstMonth, c.p.Months[len(c.p.Months)-1])
	if err != nil {
		return nil, err
	}
	names := d.ownerNames(ctx)
	seen := map[string]bool{}
	out := &domain.DashTeam{Period: c.p}
	var won, acts, prevWon float64
	for _, r := range rows {
		seen[r.OwnerUserID] = true
		target := domain.ProratedTarget(targets, c.p.From, c.p.To, r.OwnerUserID)
		out.Members = append(out.Members, domain.DashMember{UserID: r.OwnerUserID, Name: nameOr(names, r.OwnerUserID),
			OpenDeals: int(r.OpenDeals), PipelineValue: r.PipelineValue, WonDeals: int(r.WonDeals), WonValue: r.WonValue, LostDeals: int(r.LostDeals),
			WinRate: domain.Pct(float64(r.WonDeals), float64(r.WonDeals+r.LostDeals)), Activities: int(r.Activities), ActivitiesDone: int(r.ActivitiesDone),
			TargetValue: target, AttainmentPercent: domain.Pct(r.WonValue, target), PreviousWonValue: r.PreviousWonValue})
		won += r.WonValue
		acts += float64(r.Activities)
		prevWon += r.PreviousWonValue
	}
	for _, t := range targets {
		if t.UserID == "" || seen[t.UserID] || (c.f.OwnerUserID != "" && t.UserID != c.f.OwnerUserID) {
			continue
		}
		if target := domain.ProratedTarget(targets, c.p.From, c.p.To, t.UserID); target > 0 {
			seen[t.UserID] = true
			out.Members = append(out.Members, domain.DashMember{UserID: t.UserID, Name: nameOr(names, t.UserID), TargetValue: target})
		}
	}
	monthly, err := d.Store.DashWonByOwnerMonthly(ctx, c.mfrom, c.mto, c.f)
	if err != nil {
		return nil, err
	}
	idx := map[string]int{}
	for i, m := range c.p.Months {
		idx[m] = i
	}
	series := map[string]*domain.DashSeries{}
	total := make([]float64, len(c.p.Months))
	for _, r := range monthly {
		i, ok := idx[r.Month]
		if !ok || r.OwnerUserID == "" {
			continue
		}
		if series[r.OwnerUserID] == nil {
			series[r.OwnerUserID] = &domain.DashSeries{Key: r.OwnerUserID, Values: make([]float64, len(c.p.Months))}
		}
		series[r.OwnerUserID].Values[i] += r.WonValue
		total[i] += r.WonValue
	}
	for _, m := range out.Members {
		if s := series[m.UserID]; s != nil {
			out.WonByOwner = append(out.WonByOwner, *s)
		}
	}
	prevRows, err := d.Store.DashOwners(ctx, c.cfrom, c.cto, c.cfrom, c.cto, c.f)
	if err != nil {
		return nil, err
	}
	prevActs := 0.0
	for _, r := range prevRows {
		prevActs += float64(r.Activities)
	}
	target := domain.ProratedTarget(targets, c.p.From, c.p.To, c.f.OwnerUserID)
	prevTarget := domain.ProratedTarget(targets, c.p.CompareFrom, c.p.CompareTo, c.f.OwnerUserID)
	out.Kpis = []domain.DashKpi{
		{Key: "won_value", Value: won, Previous: prevWon, Spark: total},
		{Key: "target_value", Value: target, Previous: prevTarget},
		{Key: "attainment", Value: domain.Pct(won, target), Previous: domain.Pct(prevWon, prevTarget)},
		{Key: "activities", Value: acts, Previous: prevActs},
	}
	return out, nil
}

func nameOr(names map[string]string, id string) string {
	if n := names[id]; n != "" {
		return n
	}
	return id
}

func minString(vals ...string) string {
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// ── Lead & sumber ──────────────────────────────────────────────

type leadTotals struct{ leads, qualified, converted float64 }

func sumLeads(rows []domain.DashLeadSource) leadTotals {
	var t leadTotals
	for _, r := range rows {
		t.leads += float64(r.Leads)
		t.qualified += float64(r.Qualified)
		t.converted += float64(r.Converted)
	}
	return t
}

type convTotals struct{ conversations, responded, seconds, breaches float64 }

func sumConversations(rows []domain.DashConversationMonth) convTotals {
	var t convTotals
	for _, r := range rows {
		t.conversations += float64(r.Conversations)
		t.responded += float64(r.Responded)
		t.seconds += r.ResponseSeconds
		t.breaches += float64(r.Breaches)
	}
	return t
}

func (t convTotals) responseMinutes() float64 { return domain.Div(t.seconds, t.responded) / 60 }

func sumAccepted(rows []domain.DashFormCount) float64 {
	n := 0.0
	for _, r := range rows {
		n += float64(r.Accepted)
	}
	return n
}

// scoreBuckets labels DashLeadScores buckets 0..4.
var scoreBuckets = []string{"0-19", "20-39", "40-59", "60-79", "80+"}

func (d *Dashboard) Leads(ctx context.Context, q DashQuery) (*domain.DashLeads, error) {
	c, err := d.begin(q)
	if err != nil {
		return nil, err
	}
	out := &domain.DashLeads{Period: c.p}
	monthly, err := d.Store.DashLeadsMonthly(ctx, c.mfrom, c.mto, c.f)
	if err != nil {
		return nil, err
	}
	perMonth := map[string]float64{}
	for _, r := range monthly {
		perMonth[r.Month] += float64(r.Leads)
		out.MonthlySources = append(out.MonthlySources, domain.DashSlice{Key: r.Month, Group: r.Source, Count: int(r.Leads)})
	}
	sc, err := d.Store.DashLeadSources(ctx, c.from, c.to, c.f)
	if err != nil {
		return nil, err
	}
	sp, err := d.Store.DashLeadSources(ctx, c.cfrom, c.cto, c.f)
	if err != nil {
		return nil, err
	}
	for _, r := range sc {
		out.Sources = append(out.Sources, domain.DashSlice{Key: r.Source, Label: r.Source, Count: int(r.Leads), Value: float64(r.Converted), Extra: float64(r.Won)})
	}
	scores, err := d.Store.DashLeadScores(ctx, c.from, c.to, c.f)
	if err != nil {
		return nil, err
	}
	for i, b := range scoreBuckets {
		sl := domain.DashSlice{Key: b, Label: b}
		for _, r := range scores {
			if int(r.Bucket) == i {
				sl.Count = int(r.Leads)
			}
		}
		out.Scores = append(out.Scores, sl)
	}
	var conv [3][]domain.DashConversationMonth
	for i, w := range [3][2]time.Time{{c.from, c.to}, {c.cfrom, c.cto}, {c.mfrom, c.mto}} {
		if conv[i], err = d.Store.DashConversations(ctx, w[0], w[1], c.f); err != nil {
			return nil, err
		}
	}
	ccur, cprev := sumConversations(conv[0]), sumConversations(conv[1])
	convMonth := map[string][]domain.DashConversationMonth{}
	for _, r := range conv[2] {
		convMonth[r.Month] = append(convMonth[r.Month], r)
	}
	pickConv := func(f func(convTotals) float64) []float64 {
		return byMonth(c.p.Months, func(m string) float64 { return f(sumConversations(convMonth[m])) })
	}
	out.Response = []domain.DashSeries{
		{Key: "conversations", Values: pickConv(func(t convTotals) float64 { return t.conversations })},
		{Key: "breaches", Values: pickConv(func(t convTotals) float64 { return t.breaches })},
		{Key: "response_minutes", Values: pickConv(convTotals.responseMinutes)},
	}
	channels := map[string][]domain.DashConversationMonth{}
	for _, r := range conv[0] {
		channels[r.Channel] = append(channels[r.Channel], r)
	}
	for _, ch := range []string{"web", "whatsapp"} {
		t := sumConversations(channels[ch])
		out.Channels = append(out.Channels, domain.DashSlice{Key: ch, Label: ch, Count: int(t.conversations), Value: t.responseMinutes(), Extra: t.breaches})
	}
	fc, err := d.Store.DashFormSubmissions(ctx, c.from, c.to)
	if err != nil {
		return nil, err
	}
	fp, err := d.Store.DashFormSubmissions(ctx, c.cfrom, c.cto)
	if err != nil {
		return nil, err
	}
	for _, r := range fc {
		out.Forms = append(out.Forms, domain.DashSlice{Key: r.ID, Label: r.Name, Count: int(r.Accepted), Extra: float64(r.Rejected)})
	}
	lc, lp := sumLeads(sc), sumLeads(sp)
	out.Kpis = []domain.DashKpi{
		{Key: "leads", Value: lc.leads, Previous: lp.leads, Spark: byMonth(c.p.Months, func(m string) float64 { return perMonth[m] })},
		{Key: "qualified", Value: lc.qualified, Previous: lp.qualified},
		{Key: "converted", Value: lc.converted, Previous: lp.converted},
		{Key: "conversion_rate", Value: domain.Pct(lc.converted, lc.leads), Previous: domain.Pct(lp.converted, lp.leads)},
		{Key: "conversations", Value: ccur.conversations, Previous: cprev.conversations, Spark: out.Response[0].Values},
		{Key: "first_response_minutes", Value: ccur.responseMinutes(), Previous: cprev.responseMinutes(), Spark: out.Response[2].Values},
		{Key: "sla_breaches", Value: ccur.breaches, Previous: cprev.breaches, Spark: out.Response[1].Values},
		{Key: "submissions", Value: sumAccepted(fc), Previous: sumAccepted(fp)},
	}
	return out, nil
}

// ── Tagihan ────────────────────────────────────────────────────

type billTotals struct{ invoiced, collected, payments, payDays float64 }

func sumBilling(rows []domain.DashBillingMonth) billTotals {
	var t billTotals
	for _, r := range rows {
		if r.Kind == "invoiced" {
			t.invoiced += r.Amount
			continue
		}
		t.collected += r.Amount
		t.payments += float64(r.N)
		t.payDays += r.PayDays
	}
	return t
}

func billingByMonth(rows []domain.DashBillingMonth) map[string]billTotals {
	out := map[string]billTotals{}
	for _, r := range rows {
		t := out[r.Month]
		one := sumBilling([]domain.DashBillingMonth{r})
		t.invoiced += one.invoiced
		t.collected += one.collected
		out[r.Month] = t
	}
	return out
}

type receivableTotals struct{ outstanding, overdue float64 }

func (d *Dashboard) receivables(ctx context.Context, asOf time.Time, f domain.DashFilter) ([]domain.DashReceivable, receivableTotals, error) {
	rows, err := d.Store.DashReceivables(ctx, asOf, f)
	if err != nil {
		return nil, receivableTotals{}, err
	}
	var t receivableTotals
	open := rows[:0]
	for _, r := range rows {
		left := r.Amount - r.Paid
		if left <= 0.5 {
			continue
		}
		if r.DueDate != nil {
			due := time.Date(r.DueDate.Year(), r.DueDate.Month(), r.DueDate.Day(), 0, 0, 0, 0, domain.Jakarta)
			r.DaysOverdue = domain.DaysBetween(due, asOf)
		}
		t.outstanding += left
		if r.DaysOverdue > 0 {
			t.overdue += left
		}
		open = append(open, r)
	}
	return open, t, nil
}

func (d *Dashboard) Billing(ctx context.Context, q DashQuery) (*domain.DashBilling, error) {
	c, err := d.begin(q)
	if err != nil {
		return nil, err
	}
	var rows [4][]domain.DashBillingMonth
	for i, w := range [4][2]time.Time{{c.from, c.to}, {c.cfrom, c.cto}, {c.mfrom, c.mto}, {c.cmfrom, c.cmto}} {
		if rows[i], err = d.Store.DashBillingMonthly(ctx, w[0], w[1], c.f); err != nil {
			return nil, err
		}
	}
	cur, prev := sumBilling(rows[0]), sumBilling(rows[1])
	trend, ctrend := billingByMonth(rows[2]), billingByMonth(rows[3])
	series := func(key string, pick func(billTotals) float64) domain.DashSeries {
		return domain.DashSeries{Key: key,
			Values:   byMonth(c.p.Months, func(m string) float64 { return pick(trend[m]) }),
			Previous: byMonth(c.p.CompareMonths, func(m string) float64 { return pick(ctrend[m]) })}
	}
	out := &domain.DashBilling{Period: c.p}
	out.Monthly = []domain.DashSeries{
		series("invoiced", func(t billTotals) float64 { return t.invoiced }),
		series("collected", func(t billTotals) float64 { return t.collected }),
	}
	// Receivables are read as of the last day of each period (today at most).
	lastDay := func(end time.Time) time.Time {
		t := c.periodEnd(end).Add(-time.Nanosecond).In(domain.Jakarta)
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, domain.Jakarta)
	}
	open, rc, err := d.receivables(ctx, lastDay(c.to), c.f)
	if err != nil {
		return nil, err
	}
	_, rp, err := d.receivables(ctx, lastDay(c.cto), c.f)
	if err != nil {
		return nil, err
	}
	buckets := map[string]*domain.DashSlice{}
	for _, b := range domain.ReceivableBuckets {
		buckets[b] = &domain.DashSlice{Key: b, Label: b}
	}
	for _, r := range open {
		b := buckets[domain.ReceivableBucket(r.DaysOverdue)]
		b.Count++
		b.Value += r.Amount - r.Paid
		if r.DaysOverdue > 0 {
			out.Overdue = append(out.Overdue, r)
		}
	}
	for _, b := range domain.ReceivableBuckets {
		out.Aging = append(out.Aging, *buckets[b])
	}
	sort.SliceStable(out.Overdue, func(i, j int) bool { return out.Overdue[i].DaysOverdue > out.Overdue[j].DaysOverdue })
	if len(out.Overdue) > 15 {
		out.Overdue = out.Overdue[:15]
	}
	if out.Terms, err = d.Store.DashTermProgress(ctx, c.f); err != nil {
		return nil, err
	}
	out.Kpis = []domain.DashKpi{
		{Key: "invoiced", Value: cur.invoiced, Previous: prev.invoiced, Spark: out.Monthly[0].Values},
		{Key: "collected", Value: cur.collected, Previous: prev.collected, Spark: out.Monthly[1].Values},
		{Key: "outstanding", Value: rc.outstanding, Previous: rp.outstanding},
		{Key: "overdue", Value: rc.overdue, Previous: rp.overdue},
		{Key: "collection_rate", Value: domain.Pct(cur.collected, cur.invoiced), Previous: domain.Pct(prev.collected, prev.invoiced)},
		{Key: "days_to_pay", Value: domain.Div(cur.payDays, cur.payments), Previous: domain.Div(prev.payDays, prev.payments)},
	}
	return out, nil
}
