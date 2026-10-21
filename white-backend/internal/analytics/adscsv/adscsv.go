// Package adscsv parses Meta Ads Manager CSV exports (English and Indonesian
// headers, local number and date formats). Pure: no database.
package adscsv

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Field string

const (
	FCampaignName  Field = "campaignName"
	FAdSetName     Field = "adSetName"
	FAdName        Field = "adName"
	FCampaignID    Field = "campaignId"
	FAdSetID       Field = "adSetId"
	FAdID          Field = "adId"
	FDay           Field = "day"
	FReportingEnds Field = "reportingEnds"
	FSpend         Field = "spend"
	FImpressions   Field = "impressions"
	FReach         Field = "reach"
	FLinkClicks    Field = "linkClicks"
	FClicks        Field = "clicks"
	FResults       Field = "results"
	FResultType    Field = "resultType"
	FAge           Field = "age"
	FGender        Field = "gender"
	FPurchaseValue Field = "purchaseValue"
)

// fieldOrder is the detection order; the first alias that matches wins.
var fieldOrder = []Field{FCampaignName, FAdSetName, FAdName, FCampaignID, FAdSetID, FAdID, FDay, FReportingEnds, FSpend, FImpressions, FReach, FLinkClicks, FClicks, FResults, FResultType, FAge, FGender, FPurchaseValue}

var aliases = map[Field][]string{
	FCampaignName:  {"campaign name", "nama kampanye"},
	FAdSetName:     {"ad set name", "nama set iklan"},
	FAdName:        {"ad name", "nama iklan"},
	FCampaignID:    {"campaign id", "id kampanye"},
	FAdSetID:       {"ad set id", "id set iklan"},
	FAdID:          {"ad id", "id iklan"},
	FDay:           {"day", "hari", "reporting starts", "tanggal mulai pelaporan"},
	FReportingEnds: {"reporting ends", "tanggal selesai pelaporan"},
	FSpend:         {"amount spent", "jumlah yang dibelanjakan"},
	FImpressions:   {"impressions", "tayangan"},
	FReach:         {"reach", "jangkauan"},
	FLinkClicks:    {"link clicks", "klik tautan"},
	FClicks:        {"clicks (all)", "klik (semua)"},
	FResults:       {"results", "hasil"},
	FResultType:    {"result indicator", "result type", "indikator hasil", "tipe hasil"},
	FAge:           {"age", "usia"},
	FGender:        {"gender", "jenis kelamin"},
	FPurchaseValue: {"purchases conversion value", "nilai konversi pembelian", "purchase conversion value"},
}

// prefixFields may match by prefix (the currency suffix varies).
var prefixFields = map[Field]bool{FSpend: true, FPurchaseValue: true}

type Row struct {
	// daily | demographic
	Kind               string
	CampaignName       string
	CampaignExternalID *string
	AdSetName          *string
	AdSetExternalID    *string
	AdName             *string
	AdExternalID       *string
	// YYYY-MM-DD; nil only on demographic rows.
	Date          *string
	Age           *string
	Gender        *string
	Spend         float64
	Impressions   int
	Reach         int
	Clicks        int
	LinkClicks    int
	Results       int
	ResultType    *string
	PurchaseValue *float64
}

type Result struct {
	Rows []Row
	// Detected maps a field to the original header name.
	Detected map[Field]string
	Warnings []string
}

const MaxWarnings = 8

var spaceRe = regexp.MustCompile(`\s+`)

func normalizeHeader(h string) string {
	return spaceRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(strings.TrimPrefix(h, "﻿"))), " ")
}

// DetectColumns maps fields to header names using the EN/ID aliases.
func DetectColumns(headers []string) map[Field]string {
	type h struct{ original, norm string }
	normalized := make([]h, 0, len(headers))
	for _, x := range headers {
		normalized = append(normalized, h{x, normalizeHeader(x)})
	}
	used := map[string]bool{}
	out := map[Field]string{}
	for _, field := range fieldOrder {
	aliasLoop:
		for _, alias := range aliases[field] {
			for _, cand := range normalized {
				if used[cand.original] {
					continue
				}
				if cand.norm == alias || (prefixFields[field] && strings.HasPrefix(cand.norm, alias)) {
					out[field] = cand.original
					used[cand.original] = true
					break aliasLoop
				}
			}
		}
	}
	return out
}

var (
	nonNumRe       = regexp.MustCompile(`[^\d.,-]`)
	commaGroupsRe  = regexp.MustCompile(`^\d{1,3}(,\d{3})+$`)
	dotGroupsRe    = regexp.MustCompile(`^\d{1,3}(\.\d{3})+$`)
	isoDateRe      = regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2})`)
	slashDateRe    = regexp.MustCompile(`^(\d{1,2})[/.](\d{1,2})[/.](\d{4})$`)
	whitespaceToUS = regexp.MustCompile(`\s+`)
)

// ParseNumber reads "1.234.567,89", "1,234,567.89", "1.234", "12,5", "Rp 1.500".
// nil for empty or unreadable cells.
func ParseNumber(raw *string) *float64 {
	if raw == nil {
		return nil
	}
	s := strings.TrimSpace(*raw)
	if s == "" || s == "--" || s == "-" || s == "–" {
		return nil
	}
	s = nonNumRe.ReplaceAllString(s, "")
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	if s == "" {
		return nil
	}
	lastDot := strings.LastIndex(s, ".")
	lastComma := strings.LastIndex(s, ",")
	switch {
	case lastDot != -1 && lastComma != -1:
		if lastDot > lastComma {
			s = strings.ReplaceAll(s, ",", "")
		} else {
			s = strings.Replace(strings.ReplaceAll(s, ".", ""), ",", ".", 1)
		}
	case lastComma != -1:
		switch {
		case commaGroupsRe.MatchString(s):
			s = strings.ReplaceAll(s, ",", "")
		case strings.Count(s, ",") == 1:
			s = strings.Replace(s, ",", ".", 1)
		default:
			return nil
		}
	case lastDot != -1:
		if dotGroupsRe.MatchString(s) {
			s = strings.ReplaceAll(s, ".", "")
		} else if strings.Count(s, ".") > 1 {
			return nil
		}
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsInf(n, 0) || math.IsNaN(n) {
		return nil
	}
	if neg {
		n = -n
	}
	return &n
}

func isoDate(y, m, d int) *string {
	if m < 1 || m > 12 || d < 1 || d > 31 {
		return nil
	}
	t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	if t.Year() != y || int(t.Month()) != m || t.Day() != d {
		return nil
	}
	s := fmt.Sprintf("%04d-%02d-%02d", y, m, d)
	return &s
}

// ParseDate reads YYYY-M-D first, then D/M/YYYY (or dots); the slash form
// is DD/MM unless that is impossible and MM/DD is valid.
func ParseDate(raw *string) *string {
	if raw == nil {
		return nil
	}
	s := strings.TrimSpace(*raw)
	if s == "" {
		return nil
	}
	if m := isoDateRe.FindStringSubmatch(s); m != nil {
		return isoDate(atoi(m[1]), atoi(m[2]), atoi(m[3]))
	}
	if m := slashDateRe.FindStringSubmatch(s); m != nil {
		a, b, y := atoi(m[1]), atoi(m[2]), atoi(m[3])
		if b > 12 && a <= 12 {
			return isoDate(y, a, b)
		}
		return isoDate(y, b, a)
	}
	return nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// NormalizeGender maps EN/ID values to male, female or unknown; all/semua
// and empty mean "no breakdown" (nil).
func NormalizeGender(raw *string) *string {
	if raw == nil {
		return nil
	}
	g := strings.ToLower(strings.TrimSpace(*raw))
	if g == "" || g == "all" || g == "semua" {
		return nil
	}
	out := "unknown"
	switch g {
	case "male", "pria", "laki-laki", "m":
		out = "male"
	case "female", "wanita", "perempuan", "f":
		out = "female"
	}
	return &out
}

func normalizeAge(raw *string) *string {
	if raw == nil {
		return nil
	}
	a := strings.TrimSpace(*raw)
	if a == "" {
		return nil
	}
	switch strings.ToLower(a) {
	case "all", "semua":
		return nil
	case "unknown", "uncategorized", "tidak diketahui":
		u := "unknown"
		return &u
	}
	return &a
}

// NormalizeResultType maps Meta's result indicator to a canonical type.
func NormalizeResultType(raw *string) *string {
	if raw == nil {
		return nil
	}
	v := strings.ToLower(strings.TrimSpace(*raw))
	if v == "" {
		return nil
	}
	has := func(subs ...string) bool {
		for _, s := range subs {
			if strings.Contains(v, s) {
				return true
			}
		}
		return false
	}
	var out string
	switch {
	case has("purchase", "pembelian"):
		out = "purchase"
	case has("messaging_conversation", "messaging conversation", "percakapan"):
		out = "messaging_conversation_started"
	case has("lead", "prospek"):
		out = "lead"
	case has("link_click", "link click", "klik tautan"):
		out = "link_click"
	case has("thruplay"):
		out = "thruplay"
	case has("video_view", "video view"):
		out = "video_view"
	case has("reach", "jangkauan"):
		out = "reach"
	case has("impression", "impresi", "tayangan"):
		out = "impressions"
	default:
		parts := strings.Split(v, ":")
		tail := parts[len(parts)-1]
		dots := strings.Split(tail, ".")
		tail = whitespaceToUS.ReplaceAllString(strings.TrimSpace(dots[len(dots)-1]), "_")
		if tail == "" {
			return nil
		}
		out = tail
	}
	return &out
}

func jsRound(x float64) int { return int(math.Floor(x + 0.5)) }

// Parse reads a whole CSV export. Rows without a campaign name are
// subtotal/footer lines and are skipped; daily rows need a date.
func Parse(content string) Result {
	warnings := []string{}
	content = strings.TrimPrefix(content, "﻿")
	reader := csv.NewReader(strings.NewReader(content))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = false

	var headers []string
	var records [][]string
	line := 0
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		line++
		if err != nil {
			if len(warnings) < 3 {
				warnings = append(warnings, fmt.Sprintf("Baris %d: %s", line, err.Error()))
			}
			continue
		}
		empty := true
		for _, c := range rec {
			if strings.TrimSpace(c) != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}
		if headers == nil {
			headers = make([]string, 0, len(rec))
			for _, h := range rec {
				headers = append(headers, strings.TrimSpace(strings.TrimPrefix(h, "﻿")))
			}
			continue
		}
		records = append(records, rec)
	}
	detected := DetectColumns(headers)
	index := map[string]int{}
	for i, h := range headers {
		index[h] = i
	}

	var missing []string
	if _, ok := detected[FCampaignName]; !ok {
		missing = append(missing, "Campaign name / Nama kampanye")
	}
	if _, ok := detected[FSpend]; !ok {
		missing = append(missing, "Amount spent / Jumlah yang dibelanjakan")
	}
	if len(missing) > 0 {
		warnings = append(warnings, fmt.Sprintf("Kolom wajib tidak ditemukan: %s.", strings.Join(missing, ", ")))
		return Result{Rows: []Row{}, Detected: detected, Warnings: warnings}
	}
	if _, ok := detected[FDay]; !ok {
		warnings = append(warnings, "Kolom tanggal (Day / Hari / Reporting starts) tidak ditemukan — baris harian tanpa tanggal dilewati.")
	}

	get := func(rec []string, f Field) *string {
		h, ok := detected[f]
		if !ok {
			return nil
		}
		i, ok := index[h]
		if !ok || i >= len(rec) {
			return nil
		}
		v := rec[i]
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return &v
	}
	num := func(rec []string, f Field) float64 {
		if n := ParseNumber(get(rec, f)); n != nil {
			return *n
		}
		return 0
	}
	trimmed := func(s *string) *string {
		if s == nil {
			return nil
		}
		t := strings.TrimSpace(*s)
		return &t
	}

	rows := []Row{}
	skippedNoName, skippedNoDate, badDates := 0, 0, 0
	for _, rec := range records {
		name := trimmed(get(rec, FCampaignName))
		if name == nil || *name == "" {
			skippedNoName++
			continue
		}
		age := normalizeAge(get(rec, FAge))
		gender := NormalizeGender(get(rec, FGender))
		kind := "daily"
		if age != nil || gender != nil {
			kind = "demographic"
		}
		dayRaw := get(rec, FDay)
		date := ParseDate(dayRaw)
		if dayRaw != nil && date == nil {
			badDates++
		}
		if kind == "daily" && date == nil {
			skippedNoDate++
			continue
		}
		if age == nil && gender != nil {
			u := "unknown"
			age = &u
		}
		if gender == nil && age != nil {
			u := "unknown"
			gender = &u
		}
		rows = append(rows, Row{
			Kind: kind, CampaignName: *name,
			CampaignExternalID: trimmed(get(rec, FCampaignID)), AdSetName: trimmed(get(rec, FAdSetName)), AdSetExternalID: trimmed(get(rec, FAdSetID)),
			AdName: trimmed(get(rec, FAdName)), AdExternalID: trimmed(get(rec, FAdID)), Date: date, Age: age, Gender: gender,
			Spend: num(rec, FSpend), Impressions: jsRound(num(rec, FImpressions)), Reach: jsRound(num(rec, FReach)), Clicks: jsRound(num(rec, FClicks)),
			LinkClicks: jsRound(num(rec, FLinkClicks)), Results: jsRound(num(rec, FResults)),
			ResultType: NormalizeResultType(get(rec, FResultType)), PurchaseValue: ParseNumber(get(rec, FPurchaseValue)),
		})
	}
	if skippedNoDate > 0 {
		warnings = append(warnings, fmt.Sprintf("%d baris harian dilewati karena tanggal kosong/tak terbaca.", skippedNoDate))
	}
	if badDates > 0 && badDates != skippedNoDate {
		warnings = append(warnings, fmt.Sprintf("%d nilai tanggal tidak terbaca.", badDates))
	}
	if skippedNoName > 0 && len(rows) == 0 {
		warnings = append(warnings, fmt.Sprintf("%d baris tanpa nama kampanye dilewati.", skippedNoName))
	}
	if len(rows) == 0 && len(records) > 0 && len(warnings) == 0 {
		warnings = append(warnings, "Tidak ada baris data yang bisa dibaca dari file.")
	}
	if len(warnings) > MaxWarnings {
		warnings = warnings[:MaxWarnings]
	}
	return Result{Rows: rows, Detected: detected, Warnings: warnings}
}
