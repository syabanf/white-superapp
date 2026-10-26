package domain

import (
	"slices"
	"time"
)

// SharePresets are the ranges the public dashboard offers (default 28d).
var SharePresets = []string{"7d", "28d", "90d"}

// SharePreset falls back to 28d for anything not offered.
func SharePreset(p string) string {
	if slices.Contains(SharePresets, p) {
		return p
	}
	return "28d"
}

// ShareTitle is the PDF title of the public share export.
func ShareTitle(clientName string) string { return "Laporan Performa: " + clientName }

// PdfFilename is Laporan_{slug}_{from}_{to}.pdf.
func PdfFilename(slug string, from, to time.Time) string {
	return "Laporan_" + slug + "_" + from.UTC().Format(time.DateOnly) + "_" + to.UTC().Format(time.DateOnly) + ".pdf"
}

// ShareView is getShareSocial/Seo/Ads over fetched data: a block without
// data is nil, social daily totals cover the range only and captions are
// truncated to 90 characters.
func ShareView(data ReportData, from time.Time) ReportData {
	var out ReportData
	if s := data.Social; s != nil && s.Overview.HasData {
		daily := []FollowersPoint{}
		for _, p := range s.DailyFollowers {
			if !p.Date.Before(from) {
				daily = append(daily, p)
			}
		}
		posts := make([]TopPost, 0, len(s.TopPosts))
		for _, p := range firstN(s.TopPosts, 10) {
			p.Caption = Truncate(p.Caption, 90)
			posts = append(posts, p)
		}
		out.Social = &SocialData{Overview: s.Overview, DailyFollowers: daily, TopPosts: posts}
	}
	if s := data.Seo; s != nil && s.HasData {
		cp := *s
		cp.TopQueries = firstN(s.TopQueries, 10)
		out.Seo = &cp
	}
	if a := data.Ads; a != nil && a.HasData {
		out.Ads = a
	}
	return out
}
