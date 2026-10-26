package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/content/app"
	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/providers/mockrand"
)

// Mock fills Indonesian agency templates from the brief: same input, same
// output.
type Mock struct{}

func (Mock) model() string { return MockModel }

var defaultPillars = []string{"Edukasi", "Promo", "Engagement", "Behind the scene"}

var formatCycle = map[domain.Platform][]domain.Format{
	domain.PlatformInstagram: {domain.FormatFeed, domain.FormatCarousel, domain.FormatReel, domain.FormatStory},
	domain.PlatformFacebook:  {domain.FormatFeed, domain.FormatCarousel, domain.FormatReel},
	domain.PlatformTikTok:    {domain.FormatReel},
}

var titleTemplates = []string{
	"5 hal soal %s yang jarang dibahas",
	"Kenapa %s penting buat kamu minggu ini",
	"Cerita di balik %s",
	"Checklist %s sebelum akhir bulan",
	"Mitos vs fakta: %s",
	"Tanya jawab seputar %s",
}

var hookTemplates = []string{
	"Kamu pasti pernah ngalamin ini.",
	"Stop scroll sebentar, ini penting.",
	"Banyak yang belum tahu soal ini.",
	"Simpan dulu, nanti pasti kepake.",
	"Satu kebiasaan kecil, dampaknya besar.",
}

var ctaTemplates = []string{
	"Tulis pendapatmu di kolom komentar ya!",
	"Simpan postingan ini dan bagikan ke temanmu.",
	"Cek link di bio untuk info lengkapnya.",
	"Tag teman yang perlu lihat ini.",
}

// hashtagWord turns "Kopi Senja" into "KopiSenja".
func hashtagWord(s string) string {
	var b strings.Builder
	for _, w := range strings.Fields(s) {
		w = strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, w)
		if w != "" {
			b.WriteString(strings.ToUpper(w[:1]) + w[1:])
		}
	}
	return b.String()
}

func (Mock) draftPlan(_ context.Context, in app.PlanPrompt) ([]app.ItemDraft, error) {
	pillars := in.Brief.Pillars
	if len(pillars) == 0 {
		pillars = defaultPillars
	}
	days := in.Month.Last().Day()
	brand := hashtagWord(in.ClientName)
	var out []app.ItemDraft
	n := 0
	for _, slot := range slots(in.Month, in.Brief.Frequency) {
		formats := formatCycle[slot.Platform]
		for k := 0; k < int(slot.PerWeek); k++ {
			r := mockrand.New(fmt.Sprintf("%s|%s|%s|%d", in.ClientName, in.Month, slot.Platform, k))
			day := 1 + (2*k+1)*days/(2*int(slot.PerWeek))
			pillar := pillars[n%len(pillars)]
			format := formats[k%len(formats)]
			topic := strings.ToLower(pillar)
			title := fmt.Sprintf(mockrand.Pick(r, titleTemplates), topic)
			hook := mockrand.Pick(r, hookTemplates)
			caption := fmt.Sprintf("%s\n\n%s dari %s: %s. %s", hook, pillar, in.ClientName, strings.ToLower(title), mockrand.Pick(r, ctaTemplates))
			if in.Brief.Tone != "" {
				caption += "\n\n(Tone: " + in.Brief.Tone + ")"
			}
			if len(in.TopPosts) > 0 {
				caption += "\n\nTerinspirasi dari konten dengan engagement tertinggi bulan lalu."
			}
			visual := fmt.Sprintf("%s %s, warna brand %s, teks utama \"%s\".", domain.FormatLabel(format), domain.PlatformLabel(slot.Platform), in.ClientName, title)
			tags := []string{"#" + brand, "#" + hashtagWord(pillar), "#InfoHariIni"}
			out = append(out, app.ItemDraft{
				PlannedDate: time.Date(in.Month.Year(), in.Month.Month(), min(day, days), 0, 0, 0, 0, time.UTC),
				Platform:    slot.Platform, Format: format, Pillar: pillar, Title: title, Hook: hook, Caption: caption, Hashtags: tags, VisualBrief: visual,
			})
			n++
		}
	}
	return out, nil
}

func pct(v float64) string { return strings.Replace(fmt.Sprintf("%.1f%%", v), ".", ",", 1) }

func (Mock) summarize(_ context.Context, d domain.ReportData) (string, error) {
	month, _ := domain.ParseMonth(d.Month)
	var b strings.Builder
	fmt.Fprintf(&b, "## Ringkasan\n%s menayangkan %d dari %d konten yang direncanakan pada %s", d.ClientName, d.Published, d.Planned, month.Label())
	if d.Planned > 0 {
		fmt.Fprintf(&b, " (%s)", pct(float64(d.Published)*100/float64(d.Planned)))
	}
	fmt.Fprintf(&b, ". Ketepatan waktu terhadap deadline %s (%d dari %d deadline), dengan rata-rata %s putaran revisi per konten.\n\n", pct(d.OnTimeRate), d.DeadlinesMet, d.DeadlinesTotal, strings.Replace(fmt.Sprintf("%.1f", d.RevisionAvg), ".", ",", 1))
	b.WriteString("## Sorotan\n")
	if d.EngagementAvailable && len(d.Top) > 0 {
		fmt.Fprintf(&b, "- Total engagement %d dengan jangkauan %d.\n", d.TotalEngagements, d.TotalReach)
		fmt.Fprintf(&b, "- Konten terbaik: **%s** (%s %s) dengan %d engagement.\n", d.Top[0].Title, domain.PlatformLabel(d.Top[0].Platform), domain.FormatLabel(d.Top[0].Format), d.Top[0].Engagements)
	} else {
		b.WriteString("- Data engagement belum tersedia dari analytics untuk bulan ini.\n")
	}
	if d.Failed > 0 {
		fmt.Fprintf(&b, "- %d konten gagal terbit dan perlu dijadwalkan ulang.\n", d.Failed)
	}
	b.WriteString("\n## Rekomendasi\n")
	recs := []string{}
	if d.OnTimeRate < 80 {
		recs = append(recs, "Majukan deadline copywriting dua hari agar review tidak menumpuk di akhir minggu.")
	}
	if d.RevisionAvg >= 1.5 {
		recs = append(recs, "Perjelas brief dan contoh referensi sebelum copywriting untuk menekan jumlah revisi.")
	}
	if d.EngagementAvailable && len(d.Top) > 0 {
		recs = append(recs, fmt.Sprintf("Perbanyak format %s yang terbukti paling banyak engagement.", domain.FormatLabel(d.Top[0].Format)))
	}
	if d.Published < d.Planned {
		recs = append(recs, "Siapkan cadangan konten evergreen untuk slot yang tertunda.")
	}
	recs = append(recs, "Evaluasi pilar konten dengan engagement terendah dan ganti dengan topik yang lebih relevan.")
	for i, r := range recs {
		fmt.Fprintf(&b, "%d. %s\n", i+1, r)
	}
	return strings.TrimSpace(b.String()), nil
}
