package domain

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Quotation automation: a draft built from the deal's service lines, the
// service catalog, package tiers and retainer discounts. The salesperson
// edits the result like any other draft.

var PackageTiers = []string{"basic", "standard", "premium"}

var tierLabels = map[string]string{"basic": "Basic", "standard": "Standard", "premium": "Premium"}

// CatalogItem is one sellable service.
type CatalogItem struct {
	ID               string
	Code             string
	Name             string
	ServiceLine      string
	Unit             string
	DefaultPrice     Money
	Description      string
	IsRetainer       bool
	IsDefaultForLine bool
	IsActive         bool
	SortOrder        int
	Tiers            []CatalogTier
}

// CatalogTier is a package tier price.
type CatalogTier struct {
	Tier        string
	Price       Money
	Description string
}

// PriceFor returns the tier price (and its description) or the default.
func (c CatalogItem) PriceFor(tier string) (Money, string) {
	for _, t := range c.Tiers {
		if t.Tier == tier {
			return t.Price, t.Description
		}
	}
	return c.DefaultPrice, ""
}

// RetainerDiscount is the discount for committing to min_months or more.
type RetainerDiscount struct {
	MinMonths       int
	DiscountPercent float64
}

// RetainerDiscountFor picks the discount of the highest bar reached.
func RetainerDiscountFor(months int, rules []RetainerDiscount) float64 {
	best, bar := 0.0, 0
	for _, r := range rules {
		if months >= r.MinMonths && r.MinMonths > bar {
			best, bar = r.DiscountPercent, r.MinMonths
		}
	}
	return best
}

// DraftOptions steer GenerateQuotationDraft.
type DraftOptions struct {
	Tier           string
	RetainerMonths int
	UsePpn         bool
	PpnPercent     float64
	ValidDays      int
	StartDate      time.Time
}

// Normalize fills defaults: standard tier, 3 months, PPN 11%, valid 14 days.
func (o DraftOptions) Normalize(now time.Time) DraftOptions {
	if !Contains(PackageTiers, o.Tier) {
		o.Tier = "standard"
	}
	if o.RetainerMonths <= 0 {
		o.RetainerMonths = 3
	}
	o.RetainerMonths = min(o.RetainerMonths, 24)
	if o.PpnPercent <= 0 {
		o.PpnPercent = 11
	}
	if o.ValidDays <= 0 {
		o.ValidDays = 14
	}
	if o.StartDate.IsZero() {
		n := now.In(Jakarta)
		o.StartDate = time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
	}
	return o
}

// BuildQuotationDraft turns service lines into quotation lines. Lines the
// catalog cannot price become free-text items at zero for the salesperson
// to fill in; they are also returned so the UI can flag them.
func BuildQuotationDraft(lines []string, catalog []CatalogItem, discounts []RetainerDiscount, opt DraftOptions, now time.Time) (QuotationPayload, []string) {
	p := QuotationPayload{UsePpn: opt.UsePpn, PpnPercent: opt.PpnPercent}
	var missing []string
	hasRetainer, hasAds := false, false
	sorted := append([]CatalogItem(nil), catalog...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].SortOrder < sorted[j].SortOrder })
	for _, line := range lines {
		var picks []CatalogItem
		for _, c := range sorted {
			if c.IsActive && c.ServiceLine == line && c.IsDefaultForLine {
				picks = append(picks, c)
			}
		}
		if len(picks) == 0 {
			for _, c := range sorted {
				if c.IsActive && c.ServiceLine == line {
					picks = append(picks, c)
					break
				}
			}
		}
		if line == LineAds {
			hasAds = true
		}
		if len(picks) == 0 {
			missing = append(missing, line)
			p.Items = append(p.Items, QuotationLine{ItemType: "bebas", Description: ServiceLineLabel(line) + " (harga menyusul)", Unit: "paket", Qty: 1})
			continue
		}
		for _, c := range picks {
			price, tierDesc := c.PriceFor(opt.Tier)
			qty := 1.0
			if c.IsRetainer {
				qty = float64(opt.RetainerMonths)
				hasRetainer = true
			}
			desc := c.Name
			if len(c.Tiers) > 0 {
				desc += " (" + tierLabels[opt.Tier] + ")"
			}
			scope := c.Description
			if tierDesc != "" {
				scope = strings.TrimSpace(scope + "\n" + tierDesc)
			}
			id := c.ID
			p.Items = append(p.Items, QuotationLine{ItemType: "katalog", CatalogID: &id, Description: desc, Scope: scope, Unit: c.Unit, Qty: qty, UnitPrice: price})
		}
	}
	if hasRetainer {
		p.DiscountPercent = RetainerDiscountFor(opt.RetainerMonths, discounts)
	}
	p.Terms = DraftTerms(hasRetainer, opt.RetainerMonths, opt.StartDate)
	valid := now.In(Jakarta).AddDate(0, 0, opt.ValidDays)
	vu := time.Date(valid.Year(), valid.Month(), valid.Day(), 0, 0, 0, 0, time.UTC)
	p.ValidUntil = &vu
	notes := []string{"Harga dalam Rupiah dan berlaku sampai tanggal masa berlaku penawaran."}
	if hasRetainer {
		notes = append(notes, "Kontrak retainer "+strconv.Itoa(opt.RetainerMonths)+" bulan, ditagih per bulan di awal periode.")
	}
	if hasAds {
		notes = append(notes, "Biaya iklan (ad spend) dibayar terpisah langsung ke platform dan tidak termasuk dalam penawaran ini.")
	}
	n := strings.Join(notes, "\n")
	p.Notes = &n
	return p, missing
}

// DraftTerms are the default payment terms: one per month for a retainer
// up to 12 months, otherwise 50% down payment and 50% on completion.
// Percents keep two decimals and the last term absorbs the rounding.
func DraftTerms(retainer bool, months int, start time.Time) []QuotationTermInput {
	if !retainer || months > 12 || months < 1 {
		due := start
		end := start.AddDate(0, 1, 0)
		return []QuotationTermInput{{Label: "Uang muka (DP)", Percent: 50, DueDate: &due}, {Label: "Pelunasan", Percent: 50, DueDate: &end}}
	}
	each := math.Floor(10000/float64(months)) / 100
	terms := make([]QuotationTermInput, months)
	used := 0.0
	for i := range terms {
		pct := each
		if i == months-1 {
			pct = math.Round((100-used)*100) / 100
		}
		used += pct
		due := start.AddDate(0, i, 0)
		terms[i] = QuotationTermInput{Label: "Retainer bulan " + strconv.Itoa(i+1), Percent: pct, DueDate: &due}
	}
	return terms
}
