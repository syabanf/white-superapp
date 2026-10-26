package domain

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// MonthRange is monthRange: [from, to) as YYYY-MM-DD.
func MonthRange(month string) (from, to string) {
	parts := strings.Split(month, "-")
	y, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	from = fmt.Sprintf("%d-%02d-01", y, m)
	if m == 12 {
		return from, fmt.Sprintf("%d-01-01", y+1)
	}
	return from, fmt.Sprintf("%d-%02d-01", y, m+1)
}

// ForecastDeal is ForecastDealRow.
type ForecastDeal struct {
	OwnerUserID *string
	Value       Money
	Probability int
	Category    string
}

// Target is TargetSource (UserID nil = the company target).
type Target struct {
	ID          string
	UserID      *string
	Month       string
	TargetValue Money
	TargetDeals *int32
}

// ForecastRow is one salesperson row (or the total row, which also carries
// AllocatedTargetValue and CompanyTargetSet).
type ForecastRow struct {
	UserID               *string
	UserName             string
	TargetValue          Money
	TargetDeals          *int32
	WonValue             Money
	WonDeals             int
	CommitValue          Money
	BestCaseValue        Money
	PipelineValue        Money
	WeightedValue        Money
	OpenDeals            int
	AttainmentPercent    float64
	Gap                  Money
	AllocatedTargetValue *Money
	CompanyTargetSet     bool
}

// CompanyTarget is the summed company-wide target.
type CompanyTarget struct {
	TargetValue Money
	TargetDeals *int32
}

func addDeals(cur, add *int32) *int32 {
	if add == nil {
		return cur
	}
	n := *add
	if cur != nil {
		n += *cur
	}
	return &n
}

// SplitTargets is splitTargets.
func SplitTargets(targets []Target) (company *CompanyTarget, users []Target) {
	for _, t := range targets {
		if t.UserID == nil {
			if company == nil {
				company = &CompanyTarget{}
			}
			company.TargetValue += t.TargetValue
			company.TargetDeals = addDeals(company.TargetDeals, t.TargetDeals)
			continue
		}
		users = append(users, t)
	}
	return company, users
}

// ForecastUser is a candidate owner row.
type ForecastUser struct{ ID, Name string }

func attainment(projected, target Money) (percent float64, gap Money) {
	if target > 0 {
		percent = math.Round(float64(projected)/float64(target)*1000) / 10
	}
	return percent, max(0, target-projected)
}

// AggregateForecast is aggregateForecast: one row per owner (users first,
// then target owners, then deal owners), sorted by projected value with the
// unowned row last.
func AggregateForecast(deals []ForecastDeal, targets []Target, users []ForecastUser) []*ForecastRow {
	_, userTargets := SplitTargets(targets)
	names := map[string]string{}
	for _, u := range users {
		names[u.ID] = u.Name
	}
	var order []*ForecastRow
	byID := map[string]*ForecastRow{}
	var unowned *ForecastRow
	ensure := func(id *string, name string) *ForecastRow {
		if id == nil {
			if unowned == nil {
				unowned = &ForecastRow{UserName: name}
				order = append(order, unowned)
			}
			return unowned
		}
		if r, ok := byID[*id]; ok {
			return r
		}
		v := *id
		r := &ForecastRow{UserID: &v, UserName: name}
		byID[v] = r
		order = append(order, r)
		return r
	}
	for _, u := range users {
		id := u.ID
		ensure(&id, u.Name)
	}
	for _, t := range userTargets {
		name := names[*t.UserID]
		if name == "" {
			name = "-"
		}
		r := ensure(t.UserID, name)
		r.TargetValue += t.TargetValue
		r.TargetDeals = addDeals(r.TargetDeals, t.TargetDeals)
	}
	for _, d := range deals {
		name := "Tanpa PJ"
		if d.OwnerUserID != nil {
			if n := names[*d.OwnerUserID]; n != "" {
				name = n
			} else {
				name = "-"
			}
		}
		r := ensure(d.OwnerUserID, name)
		switch d.Category {
		case "closed_won":
			r.WonValue += d.Value
			r.WonDeals++
		case "closed_lost":
		default:
			r.OpenDeals++
			r.WeightedValue += Money(roundHalfUp(float64(d.Value)*float64(d.Probability)/100/100) * 100)
			switch d.Category {
			case "commit":
				r.CommitValue += d.Value
			case "best_case":
				r.BestCaseValue += d.Value
			default:
				r.PipelineValue += d.Value
			}
		}
	}
	for _, r := range order {
		r.AttainmentPercent, r.Gap = attainment(r.WonValue+r.WeightedValue, r.TargetValue)
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, b := order[i], order[j]
		if a.UserID == nil {
			return false
		}
		if b.UserID == nil {
			return true
		}
		return b.WonValue+b.WeightedValue < a.WonValue+a.WeightedValue
	})
	return order
}

// SumForecast is sumForecast: the total row; a company target replaces the
// summed salesperson targets, which stay in AllocatedTargetValue.
func SumForecast(rows []*ForecastRow, company *CompanyTarget) *ForecastRow {
	total := "total"
	t := &ForecastRow{UserID: &total, UserName: "Total"}
	for _, r := range rows {
		t.TargetValue += r.TargetValue
		t.TargetDeals = addDeals(t.TargetDeals, r.TargetDeals)
		t.WonValue += r.WonValue
		t.WonDeals += r.WonDeals
		t.CommitValue += r.CommitValue
		t.BestCaseValue += r.BestCaseValue
		t.PipelineValue += r.PipelineValue
		t.WeightedValue += r.WeightedValue
		t.OpenDeals += r.OpenDeals
	}
	allocated := t.TargetValue
	t.AllocatedTargetValue = &allocated
	t.CompanyTargetSet = company != nil && company.TargetValue > 0
	if t.CompanyTargetSet {
		t.TargetValue = company.TargetValue
		t.TargetDeals = company.TargetDeals
	}
	t.AttainmentPercent, t.Gap = attainment(t.WonValue+t.WeightedValue, t.TargetValue)
	return t
}
