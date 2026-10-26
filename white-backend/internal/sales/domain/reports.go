package domain

import (
	"math"
	"sort"
	"time"
)

// Funnel report (salesfunnel insights.go "reports", computed in Go over
// the deals and leads of a period).

type StageBucket struct {
	StageID   string
	StageName string
	SortOrder int
	Deals     int
	Value     Money
}

type SourceBucket struct {
	Source    string
	Leads     int
	Converted int
}

type ReasonBucket struct {
	ReasonID string
	Name     string
	Deals    int
}

type FunnelReport struct {
	Leads           int
	QualifiedLeads  int
	DealsCreated    int
	DealsWon        int
	DealsLost       int
	WonValue        Money
	OpenValue       Money
	WinRatePercent  float64
	AvgCycleDays    float64
	AvgDealValue    Money
	Stages          []StageBucket
	Sources         []SourceBucket
	LostReasons     []ReasonBucket
	ServiceLineWins map[string]int
}

func dealValue(d Deal) Money {
	switch {
	case d.ValueFinal != nil:
		return *d.ValueFinal
	case d.ValueEstimate != nil:
		return *d.ValueEstimate
	}
	return 0
}

func inRange(t time.Time, from, to time.Time) bool { return !t.Before(from) && t.Before(to) }

// BuildFunnelReport aggregates leads created and deals created or closed
// in [from, to). Stage buckets show the current position of open deals.
func BuildFunnelReport(leads []Lead, deals []Deal, stages []Stage, reasons []LostReason, from, to time.Time) FunnelReport {
	r := FunnelReport{ServiceLineWins: map[string]int{}}
	bucket := map[string]*StageBucket{}
	for _, s := range stages {
		if s.IsWon || s.IsLost {
			continue
		}
		b := &StageBucket{StageID: s.ID, StageName: s.Name, SortOrder: s.SortOrder}
		bucket[s.ID] = b
	}
	leadsWithDeal := map[string]bool{}
	for _, d := range deals {
		if d.LeadID != nil {
			leadsWithDeal[*d.LeadID] = true
		}
	}
	src := map[string]*SourceBucket{}
	for _, l := range leads {
		if !inRange(l.CreatedAt, from, to) {
			continue
		}
		r.Leads++
		if l.Status == "qualified" {
			r.QualifiedLeads++
		}
		b := src[l.Source]
		if b == nil {
			b = &SourceBucket{Source: l.Source}
			src[l.Source] = b
		}
		b.Leads++
		if leadsWithDeal[l.ID] {
			b.Converted++
		}
	}
	reasonNames := map[string]string{}
	for _, rs := range reasons {
		reasonNames[rs.ID] = rs.Name
	}
	lost := map[string]*ReasonBucket{}
	cycleDays := 0.0
	for _, d := range deals {
		v := dealValue(d)
		if inRange(d.CreatedAt, from, to) {
			r.DealsCreated++
		}
		switch {
		case d.IsWon && d.ClosedAt != nil && inRange(*d.ClosedAt, from, to):
			r.DealsWon++
			r.WonValue += v
			cycleDays += d.ClosedAt.Sub(d.CreatedAt).Hours() / 24
			for _, line := range d.ServiceLines {
				r.ServiceLineWins[line]++
			}
		case d.IsLost && d.ClosedAt != nil && inRange(*d.ClosedAt, from, to):
			r.DealsLost++
			id := ""
			if d.LostReasonID != nil {
				id = *d.LostReasonID
			}
			b := lost[id]
			if b == nil {
				name := reasonNames[id]
				if name == "" {
					name = "Tanpa alasan"
				}
				b = &ReasonBucket{ReasonID: id, Name: name}
				lost[id] = b
			}
			b.Deals++
		case !d.IsWon && !d.IsLost:
			r.OpenValue += v
			if b := bucket[d.StageID]; b != nil {
				b.Deals++
				b.Value += v
			}
		}
	}
	if closed := r.DealsWon + r.DealsLost; closed > 0 {
		r.WinRatePercent = math.Round(float64(r.DealsWon)/float64(closed)*1000) / 10
	}
	if r.DealsWon > 0 {
		r.AvgCycleDays = math.Round(cycleDays/float64(r.DealsWon)*10) / 10
		r.AvgDealValue = Money(int64(r.WonValue) / int64(r.DealsWon))
	}
	for _, b := range bucket {
		r.Stages = append(r.Stages, *b)
	}
	sort.Slice(r.Stages, func(i, j int) bool { return r.Stages[i].SortOrder < r.Stages[j].SortOrder })
	for _, b := range src {
		r.Sources = append(r.Sources, *b)
	}
	sort.Slice(r.Sources, func(i, j int) bool {
		if r.Sources[i].Leads != r.Sources[j].Leads {
			return r.Sources[i].Leads > r.Sources[j].Leads
		}
		return r.Sources[i].Source < r.Sources[j].Source
	})
	for _, b := range lost {
		r.LostReasons = append(r.LostReasons, *b)
	}
	sort.Slice(r.LostReasons, func(i, j int) bool {
		if r.LostReasons[i].Deals != r.LostReasons[j].Deals {
			return r.LostReasons[i].Deals > r.LostReasons[j].Deals
		}
		return r.LostReasons[i].Name < r.LostReasons[j].Name
	})
	return r
}
