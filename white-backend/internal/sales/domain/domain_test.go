package domain

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func eq(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func rp(s string) Money { return MustMoney(s) }

func ptr[T any](v T) *T { return &v }

func TestMoney(t *testing.T) {
	for in, want := range map[string]string{"1250000": "1250000.00", "1250000.5": "1250000.50", "0.005": "0.01", "-12.3": "-12.30", "": "0.00"} {
		m, err := ParseMoney(in)
		if err != nil {
			t.Fatal(in, err)
		}
		eq(t, m.String(), want)
	}
	for _, bad := range []string{"1.250.000", "abc", "1,5", "."} {
		if _, err := ParseMoney(bad); err == nil {
			t.Fatalf("%q should fail", bad)
		}
	}
	eq(t, rp("1250000.49").Rupiah(), "Rp1.250.000")
	eq(t, rp("1250000.50").Rupiah(), "Rp1.250.001")
	eq(t, FormatNumber(1234.5678, 3), "1.234,568")
}

// Ported from salesfunnel/domain TestComputeTotals (amounts in sen).
func TestComputeTotals(t *testing.T) {
	p := &QuotationPayload{UsePpn: true, PpnPercent: 11, DiscountPercent: 10,
		Items: []QuotationLine{{Qty: 3, UnitPrice: rp("33333.33")}, {Qty: 1.5, UnitPrice: rp("1000")}}}
	tot := ComputeTotals(p)
	eq(t, p.Items[0].LineTotal.String(), "99999.99")
	eq(t, tot.Subtotal.String(), "101499.99")
	eq(t, tot.DiscountNominal.String(), "10150.00")
	eq(t, tot.PpnNominal.String(), "10048.50")
	eq(t, tot.Total.String(), "101398.49")
}

func TestTermsAndAllocation(t *testing.T) {
	eq(t, TermsSumTo100(nil), true)
	eq(t, TermsSumTo100([]QuotationTermInput{{Percent: 33.33}, {Percent: 33.33}, {Percent: 33.34}}), true)
	eq(t, TermsSumTo100([]QuotationTermInput{{Percent: 50}, {Percent: 40}}), false)
	eq(t, AllocateTermAmounts(rp("10000000"), []float64{33.33, 33.33, 33.34}), []Money{rp("3333000"), rp("3333000"), rp("3334000")})
	b := AllocateTermAmounts(rp("1000.01"), []float64{50, 50})
	eq(t, (b[0] + b[1]).String(), "1000.01")
	eq(t, AllocateTermAmounts(rp("5000"), nil), []Money{})
}

func TestTermProgress(t *testing.T) {
	terms := []TermInput{{Label: "DP", Percent: 50}, {Label: "Pelunasan", Percent: 50}}
	status := func(p []TermProgress) []string {
		out := []string{}
		for _, x := range p {
			out = append(out, x.Status)
		}
		return out
	}
	total := rp("10000000")
	eq(t, status(TermProgressOf(terms, total, 0)), []string{"belum", "belum"})
	eq(t, status(TermProgressOf(terms, total, rp("5000000"))), []string{"lunas", "belum"})
	p := TermProgressOf(terms, total, rp("7000000"))
	eq(t, []any{p[0].Status, p[0].Paid, p[1].Status, p[1].Paid}, []any{"lunas", rp("5000000"), "sebagian", rp("2000000")})
	p = TermProgressOf(terms, total, rp("12000000"))
	eq(t, status(p), []string{"lunas", "lunas"})
	eq(t, p[1].Paid, rp("5000000"))
}

func TestValidateQuotation(t *testing.T) {
	errs := ValidateQuotation(QuotationPayload{DiscountPercent: 120, Terms: []QuotationTermInput{{Label: "DP", Percent: 60}}})
	eq(t, errs["items"], "Minimal 1 item")
	eq(t, errs["discountPercent"], "Diskon harus 0-100%")
	eq(t, errs["terms"], "Total persentase termin harus 100%")
	eq(t, len(ValidateQuotation(QuotationPayload{Items: []QuotationLine{{ItemType: "bebas", Description: "Jasa", Qty: 1, UnitPrice: 100}}})), 0)
}

func TestQuoteRules(t *testing.T) {
	eq(t, QuoteTransitionAllowed(QuoteDraft, QuoteSent), true)
	eq(t, QuoteTransitionAllowed(QuoteSuperseded, QuoteSent), false)
	eq(t, QuoteTransitionAllowed(QuoteAccepted, QuoteRejected), false)
	eq(t, CanRelease(ApprovalPending), false)
	eq(t, CanRelease(ApprovalApproved), true)
	now := time.Date(2026, 10, 26, 3, 0, 0, 0, time.UTC)
	vu := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)
	q := Quotation{Status: QuoteSent, ValidUntil: &vu}
	eq(t, ExpiresSoon(q, now), true)
	far := vu.AddDate(0, 0, 1)
	q.ValidUntil = &far
	eq(t, ExpiresSoon(q, now), false)
}

func TestPlanStageMove(t *testing.T) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	if _, err := PlanStageMove(StageFlags{IsWon: true}, nil, nil, nil, "", now); err != ErrWonNeedsValue {
		t.Fatal("won needs final value")
	}
	v := rp("5000000")
	mv, err := PlanStageMove(StageFlags{IsWon: true}, nil, &v, ptr("lr-x"), "", now)
	if err != nil || mv.ForecastCategory != "closed_won" || mv.ClosedAt == nil || mv.LostReasonID != nil {
		t.Fatalf("won: %+v %v", mv, err)
	}
	if _, err := PlanStageMove(StageFlags{IsLost: true}, nil, nil, nil, "", now); err != ErrLostNeedsReason {
		t.Fatal("lost needs reason")
	}
	mv, _ = PlanStageMove(StageFlags{Probability: 50}, &v, nil, ptr("lr-x"), "commit", now)
	eq(t, mv.ForecastCategory, "commit")
	eq(t, mv.ClosedAt == nil && mv.LostReasonID == nil, true)
	eq(t, CategoryFromStage(StageFlags{Probability: 75}), "commit")
	eq(t, CategoryFromStage(StageFlags{Probability: 10}), "pipeline")
}

func TestAggregateForecast(t *testing.T) {
	u1 := "u1"
	rows := AggregateForecast([]ForecastDeal{
		{OwnerUserID: &u1, Value: rp("10000000"), Probability: 100, Category: "closed_won"},
		{OwnerUserID: &u1, Value: rp("20000000"), Probability: 75, Category: "commit"},
		{OwnerUserID: &u1, Value: rp("8000000"), Probability: 50, Category: "best_case"},
		{OwnerUserID: &u1, Value: rp("5000000"), Category: "closed_lost"},
		{Value: rp("4000000"), Probability: 10, Category: "pipeline"},
	}, []Target{{UserID: &u1, TargetValue: rp("50000000"), TargetDeals: ptr[int32](3)}}, []ForecastUser{{"u1", "Ani"}, {"u2", "Budi"}})
	var a *ForecastRow
	for _, r := range rows {
		if r.UserID != nil && *r.UserID == "u1" {
			a = r
		}
	}
	eq(t, a.UserName, "Ani")
	eq(t, a.WonValue, rp("10000000"))
	eq(t, a.WeightedValue, rp("19000000"))
	eq(t, a.OpenDeals, 2)
	eq(t, a.AttainmentPercent, 58.0)
	eq(t, a.Gap, rp("21000000"))
	none := rows[len(rows)-1]
	eq(t, none.UserID == nil, true)
	eq(t, none.PipelineValue, rp("4000000"))
	total := SumForecast(rows, nil)
	eq(t, total.TargetValue, rp("50000000"))
	eq(t, total.WeightedValue, rp("19400000"))

	company, users := SplitTargets([]Target{{UserID: &u1, TargetValue: rp("30000000")}, {TargetValue: rp("80000000"), TargetDeals: ptr[int32](5)}})
	eq(t, len(users), 1)
	total = SumForecast(rows, company)
	eq(t, total.CompanyTargetSet, true)
	eq(t, total.TargetValue, rp("80000000"))
	eq(t, *total.AllocatedTargetValue, rp("50000000"))
}

func TestRecurrence(t *testing.T) {
	from := time.Date(2026, 1, 31, 2, 0, 0, 0, time.UTC)
	eq(t, NextOccurrence(from, Recurrence{Freq: "daily", Interval: 3}).Format(time.RFC3339), "2026-02-03T02:00:00Z")
	eq(t, NextOccurrence(from, Recurrence{Freq: "weekly", Interval: 2}).Format(time.RFC3339), "2026-02-14T02:00:00Z")
	eq(t, NextOccurrence(from, Recurrence{Freq: "monthly", Interval: 1}).Format(time.RFC3339), "2026-02-28T02:00:00Z")
	if NextOccurrence(from, Recurrence{Freq: "daily", Interval: 1, Until: ptr("2026-01-31")}) != nil {
		t.Fatal("past until")
	}
	if NextOccurrence(from, Recurrence{Freq: "daily", Interval: 1, Until: ptr("2026-02-01")}) == nil {
		t.Fatal("until is inclusive")
	}
	due := time.Date(2026, 9, 15, 9, 0, 0, 0, time.UTC)
	rem := due.Add(-time.Hour)
	next, nextRem, ok := SpawnNextTask(due, &rem, Recurrence{Freq: "weekly", Interval: 1})
	eq(t, ok, true)
	eq(t, next.Format(time.RFC3339), "2026-09-22T09:00:00Z")
	eq(t, nextRem.Format(time.RFC3339), "2026-09-22T08:00:00Z")
	r, ok := ParseRecurrence([]byte(`{"freq":"weekly","interval":2,"until":null}`))
	eq(t, ok, true)
	eq(t, r.Interval, 2)
	_, ok = ParseRecurrence([]byte(`{"freq":"weekly","interval":99}`))
	eq(t, ok, false)
	_, ok = ParseRecurrence([]byte(`{"freq":"daily","extra":1}`))
	eq(t, ok, false)
}

func TestLeadImport(t *testing.T) {
	rows := ParseCSVMatrix("\ufeffNama Instansi,PIC,No WA,Email,Suhu\n\"PT Maju, Tbk\",Andi,0812-3456-7890,andi@maju,panas\n,,,,\nCV Kecil,Budi,123,,\n")
	headers := make([]string, len(rows[0]))
	for i, h := range rows[0] {
		headers[i] = NormalizeLeadHeader(h)
	}
	eq(t, headers, []string{"nama_instansi", "nama_pic", "wa_pic", "email_pic", "suhu"})
	ok := MapLeadImportRow(headers, rows[1])
	eq(t, ok.Kind, LeadRowOK)
	eq(t, ok.Lead.OrgName, "PT Maju, Tbk")
	eq(t, ok.Lead.PicPhone, "6281234567890")
	eq(t, ok.Lead.Temperature, "panas")
	eq(t, ok.Lead.Source, "lainnya")
	eq(t, strings.HasPrefix(ok.Warning, "Email diabaikan"), true)
	eq(t, MapLeadImportRow(headers, rows[2]).Kind, LeadRowEmpty)
	bad := MapLeadImportRow(headers, rows[3])
	eq(t, bad.Message, "No. WA tidak valid: 123")
	eq(t, LeadDedupKey("62812", " PT A "), "62812|pt a")
}

func TestWorkflowConditions(t *testing.T) {
	record := map[string]any{"source": "wa", "temperature": "hangat", "score": 55.0, "pic_email": "", "owner_user_id": nil}
	eq(t, EvaluateConditions(nil, record, nil), true)
	eq(t, EvaluateConditions([]Condition{{Field: "source", Op: "eq", Value: "WA"}, {Field: "score", Op: "gte", Value: 50.0}, {Field: "pic_email", Op: "is_empty"}, {Field: "owner_user_id", Op: "is_empty"}}, record, nil), true)
	eq(t, EvaluateConditions([]Condition{{Field: "temperature", Op: "in", Value: []any{"panas"}}}, record, nil), false)
	changes := map[string]Change{"temperature": {From: "hangat", To: "panas"}}
	for _, tc := range []struct {
		c    Condition
		want bool
	}{
		{Condition{Field: "temperature", Op: "changed"}, true},
		{Condition{Field: "temperature", Op: "changed_to", Value: "panas"}, true},
		{Condition{Field: "source", Op: "changed"}, false},
		{Condition{Field: "source", Op: "not_in", Value: "wa"}, true},
		{Condition{Field: "source", Op: "contains", Value: ""}, false},
		{Condition{Field: "score", Op: "lt", Value: "1e2"}, true},
		{Condition{Field: "temperature", Op: "neq", Value: " HANGAT "}, false},
	} {
		if got := EvaluateCondition(tc.c, record, changes); got != tc.want {
			t.Errorf("%+v = %v, want %v", tc.c, got, tc.want)
		}
	}
	eq(t, RenderTemplate("Hubungi  {{ lead.pic_name }} skor {{score}} {{nope.x}}", map[string]any{"lead": map[string]any{"pic_name": "Andi"}, "score": 1250.5}), "Hubungi Andi skor 1.250,5")
	obj, trig, ok := TriggerForEvent("task.done")
	eq(t, []any{obj, trig, ok}, []any{"task", "status_changed", true})
	_, _, ok = TriggerForEvent("task_done")
	eq(t, ok, false)
	now, later, hours := SplitAtWait([]WorkflowAction{{Type: "create_task"}, {Type: "wait", Days: 1, Hours: 2}, {Type: "notify_in_app"}})
	eq(t, []int{len(now), len(later)}, []int{1, 1})
	eq(t, hours, 26.0)
	u, _ := PickRoundRobin([]string{"a", "b"}, 3)
	eq(t, u, "b")
	eq(t, RunStatus(2, 0, true), "scheduled")
	eq(t, RunStatus(2, 2, false), "failed")
	eq(t, TriggerConfigMatches("score_reached", map[string]any{"score": 50.0}, map[string]any{"score": 60.0}, map[string]any{"previous_score": 40.0}), true)
	eq(t, TriggerConfigMatches("score_reached", map[string]any{"score": 50.0}, map[string]any{"score": 60.0}, map[string]any{"previous_score": 55.0}), false)
	eq(t, TriggerConfigMatches("stage_changed", map[string]any{"to_stage": "proposal"}, map[string]any{"stage_code": "negosiasi"}, nil), false)
	errs := ValidateWorkflowRule("x", "lead", "created", nil, []WorkflowAction{{Type: "send_wa"}})
	eq(t, errs["actions"], "Aksi tidak dikenal: send_wa")
}

func TestComputeLeadScore(t *testing.T) {
	rules := []ScoringRule{
		{ID: "a", Kind: "field", Field: "source", Operator: "eq", Value: "wa", Points: 10},
		{ID: "b", Kind: "event", EventType: "task_done", Value: "meeting", MaxCount: 2, Points: 20},
		{ID: "c", Kind: "event", EventType: "chat_inbound", MaxCount: 3, Points: 10},
		{ID: "d", Kind: "event", EventType: "deal_created", MaxCount: 1, Points: 20},
	}
	score, breakdown := ComputeLeadScore(rules, map[string]string{"source": "wa"}, map[string]int{"task_done:meeting": 5, "chat_inbound": 2, "task_done:telepon": 9})
	eq(t, score, 70)
	eq(t, len(breakdown), 3)
	eq(t, *breakdown[1].Count, 2)
	eq(t, EventCountKey("task_done", "Meeting "), "task_done:meeting")
	w7, w30 := 7, 30
	eq(t, *MaxWindow([]ScoringRule{{Kind: "event", EventType: "x", WindowDays: &w7}, {Kind: "event", EventType: "x", WindowDays: &w30}}, "x"), 30)
	eq(t, MaxWindow([]ScoringRule{{Kind: "event", EventType: "x", WindowDays: &w7}, {Kind: "event", EventType: "x"}}, "x") == nil, true)
}

func TestApprovals(t *testing.T) {
	perm, adm, boss := ptr(ApproverPerm), ptr(ApproverAdmin), ptr("user-1")
	rules := []ApprovalRule{{ID: "a", Level: 1, MinDiscount: 10, ApproverRole: perm}, {ID: "b", Level: 2, MinDiscount: 20, ApproverRole: adm}, {ID: "c", Level: 1, MinDiscount: 15, ApproverUserID: boss}}
	ids := func(ls []ApprovalRule) []string {
		out := []string{}
		for _, l := range ls {
			out = append(out, l.ID)
		}
		return out
	}
	eq(t, ids(RequiredApprovalLevels(10, rules)), []string{})
	eq(t, ids(RequiredApprovalLevels(12, rules)), []string{"a"})
	eq(t, ids(RequiredApprovalLevels(16, rules)), []string{"c"})
	eq(t, ids(RequiredApprovalLevels(25, rules)), []string{"c", "b"})
	eq(t, CanDecideStep(nil, boss, "user-1", false, false), true)
	eq(t, CanDecideStep(nil, boss, "user-2", false, true), false)
	eq(t, CanDecideStep(perm, nil, "u", false, true), true)
	eq(t, CanDecideStep(adm, nil, "u", false, true), false)
	eq(t, CanDecideStep(adm, nil, "u", true, false), true)
}

func TestPublicForms(t *testing.T) {
	fields := []FormField{{Key: "pic_name", Label: "Nama", Type: "text", Required: true}, {Key: "pic_email", Label: "Email", Type: "email"}, {Key: "pic_phone", Label: "WA", Type: "phone"}, {Key: "budget", Label: "Budget", Type: "number"}}
	s := ValidateSubmission(fields, map[string]any{"pic_name": "Andi", "budget": "abc"})
	eq(t, s.OK(), false)
	eq(t, s.Errors[0].Key, "budget")
	s = ValidateSubmission(fields, map[string]any{"pic_name": "Andi", "budget": "5000"})
	eq(t, s.Errors[0].Message, "Isi nomor WhatsApp atau email agar kami bisa menghubungi Anda")
	s = ValidateSubmission(fields, map[string]any{"pic_name": " Andi ", "pic_email": "andi@x.id", "budget": 7.0})
	eq(t, s.OK(), true)
	eq(t, s.Lead["pic_name"], "Andi")
	eq(t, s.Custom["budget"], 7.0)
	now := time.Now()
	bot, reason := IsLikelyBot(map[string]any{"website_url": "x"}, now)
	eq(t, []any{bot, reason}, []any{true, "honeypot terisi"})
	bot, _ = IsLikelyBot(map[string]any{"form_started_at": float64(now.UnixMilli() - 1000)}, now)
	eq(t, bot, true)
	bot, _ = IsLikelyBot(map[string]any{"form_started_at": float64(now.UnixMilli() - 10000)}, now)
	eq(t, bot, false)
	a := ParseAttribution(map[string]any{"utm_source": " IG ", "utm_campaign": "promo"})
	eq(t, SourceFromAttribution(a, "website"), "instagram")
	eq(t, SourceFromAttribution(Attribution{}, "aneh"), "lainnya")
	eq(t, strings.Contains(BuildLeadAlert("Kontak", map[string]string{"pic_name": "Andi", "pic_phone": "62812"}, a), "Asal: IG / promo"), true)
}

func TestContractRendering(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2027, 4, 30, 0, 0, 0, 0, time.UTC)
	q := &Quotation{QuoteNumber: "QT-2610-0001", Total: rp("11100000"), PpnNominal: rp("1100000"), Subtotal: rp("10000000"),
		Items: []QuotationItem{{Description: "SEO Bulanan", Scope: "4 artikel", Qty: 6, Unit: "bulan", UnitPrice: rp("1666666.67"), LineTotal: rp("10000000")}},
		Terms: []QuotationTerm{{Label: "DP", Percent: 50}, {Label: "Pelunasan", Percent: 50}}}
	v := ContractValues(ContractContext{AgencyName: "WHITE", ContractNumber: "KTR-2610-0001", Today: start, ClientName: "PT Maju", Quotation: q, StartDate: &start, EndDate: &end})
	eq(t, v["fees"], "Rp11.100.000")
	eq(t, v["duration_months"], "6")
	eq(t, strings.Contains(v["payment_terms"], "- DP (50%): Rp5.550.000"), true)
	body := RenderContract("Halo {{client_name}}, {{ fees }}. {{unknown}}", v)
	eq(t, body, "Halo PT Maju, Rp11.100.000. {{unknown}}")
	eq(t, UnknownPlaceholders("{{client_name}} {{x_y}} {{x_y}}"), []string{"x_y"})
	eq(t, ContractTransitionAllowed(ContractSent, ContractSigned), true)
	eq(t, ContractTransitionAllowed(ContractDraft, ContractSigned), false)
	eq(t, ContractTransitionAllowed(ContractEnded, ContractCancelled), false)
}

func TestParseMarkdown(t *testing.T) {
	blocks := ParseMarkdown("# Judul\n\nParagraf **tebal**\nlanjut.\n\n- satu\n- dua\n\n1. a\n2. b\n\n| A | B |\n|---|---|\n| 1 | 2 |\n\n---")
	kinds := []string{}
	for _, b := range blocks {
		kinds = append(kinds, b.Kind)
	}
	eq(t, kinds, []string{BlockHeading, BlockParagraph, BlockBullets, BlockNumbered, BlockTable, BlockRule})
	eq(t, blocks[1].Text, "Paragraf **tebal** lanjut.")
	eq(t, blocks[4].Rows, [][]string{{"A", "B"}, {"1", "2"}})
	eq(t, InlineSpans("a **b** c"), []Span{{"a ", false}, {"b", true}, {" c", false}})
}

func TestQuotationDraft(t *testing.T) {
	catalog := []CatalogItem{
		{ID: "smm", Name: "Social Media Management", ServiceLine: LineSocialMedia, Unit: "bulan", DefaultPrice: rp("8500000"), IsRetainer: true, IsDefaultForLine: true, IsActive: true, Tiers: []CatalogTier{{Tier: "premium", Price: rp("14000000"), Description: "24 konten"}}},
		{ID: "web", Name: "Website", ServiceLine: LineWebsite, Unit: "proyek", DefaultPrice: rp("25000000"), IsDefaultForLine: true, IsActive: true},
	}
	now := time.Date(2026, 10, 26, 3, 0, 0, 0, time.UTC)
	opt := DraftOptions{Tier: "premium", RetainerMonths: 6, UsePpn: true}.Normalize(now)
	p, missing := BuildQuotationDraft([]string{LineSocialMedia, LineWebsite, LineBranding}, catalog, []RetainerDiscount{{3, 0}, {6, 5}, {12, 10}}, opt, now)
	eq(t, missing, []string{LineBranding})
	eq(t, len(p.Items), 3)
	eq(t, p.Items[0].Description, "Social Media Management (Premium)")
	eq(t, []any{p.Items[0].Qty, p.Items[0].UnitPrice}, []any{6.0, rp("14000000")})
	eq(t, p.Items[1].Qty, 1.0)
	eq(t, p.Items[2].ItemType, "bebas")
	eq(t, p.DiscountPercent, 5.0)
	eq(t, len(p.Terms), 6)
	eq(t, TermsSumTo100(p.Terms), true)
	eq(t, FormatDay(p.ValidUntil), "2026-11-09")
	eq(t, len(ValidateQuotation(p)), 0)
	terms := DraftTerms(true, 3, now)
	eq(t, []float64{terms[0].Percent, terms[2].Percent}, []float64{33.33, 33.34})
}

func TestChatbot(t *testing.T) {
	ok, reason := DetectHandoff("Saya mau komplain, kecewa dengan hasilnya")
	eq(t, []any{ok, reason}, []any{true, HandoffComplaint})
	ok, reason = DetectHandoff("Bisa bicara dengan sales?")
	eq(t, []any{ok, reason}, []any{true, HandoffAskedHuman})
	l := ExtractLeadFields("Nama saya Melati dari brand Glowup, budget 15 juta mulai bulan depan, WA 0812-3456-7890 email melati@glowup.id untuk iklan Meta")
	eq(t, l, LeadExtract{Name: "Melati", Company: "Glowup", Need: "Ads Management", Budget: "15 juta", Timeline: "bulan depan", Phone: "6281234567890", Email: "melati@glowup.id"})
	hints := "- SMM mulai Rp5.500.000 per bulan\n- Ads mulai Rp4 juta"
	eq(t, PricesWithinHints("Mulai Rp5.500.000 per bulan, ads Rp4.000.000", hints), true)
	eq(t, PricesWithinHints("Paket hemat cuma Rp3.000.000", hints), false)
	eq(t, PricesWithinHints("Kami bantu iklan", hints), true)
	r := ParseBotReply("```json\n{\"reply\":\"Halo\",\"confidence\":1.4,\"handoff\":false,\"lead\":{\"name\":\"Ani\"}}\n```")
	eq(t, []any{r.Reply, r.Confidence, r.Lead.Name}, []any{"Halo", 1.0, "Ani"})
	eq(t, ParseBotReply("teks biasa").Confidence, 0.5)
	s := ChatbotSettings{PricingHints: hints, BusinessHoursStart: 9, BusinessHoursEnd: 18, MinConfidence: 0.55}
	at := time.Date(2026, 10, 26, 3, 0, 0, 0, time.UTC) // 10.00 WIB
	m := MockBotReply(s, LeadExtract{}, "Halo, berapa harga kelola Instagram?", at)
	eq(t, strings.Contains(m.Reply, "Rp5.500.000"), true)
	eq(t, PricesWithinHints(m.Reply, hints), true)
	eq(t, MockBotReply(s, LeadExtract{}, "Apakah kalian bisa bikin aplikasi game?", at).Confidence < s.MinConfidence, true)
	eq(t, MockBotReply(s, LeadExtract{}, "Halo, saya mau tanya", at), MockBotReply(s, LeadExtract{}, "Halo, saya mau tanya", at))
	eq(t, strings.Contains(HandoffReply(HandoffAskedHuman, s, at.Add(12*time.Hour)), "jam kerja"), true)
	eq(t, strings.Contains(BotSystemPrompt(s, at), "HANYA boleh menyebut angka"), true)
}

func TestInsights(t *testing.T) {
	msgs := []TranscriptMessage{{"in", "Halo saya mau tanya harga SEO"}, {"out", "Baik kak"}}
	eq(t, Fingerprint(msgs), Fingerprint(msgs))
	eq(t, strings.HasPrefix(Fingerprint(msgs), "2-"), true)
	eq(t, NeedsAnalysis(ptr(Fingerprint(msgs)), Fingerprint(msgs)), false)
	in := ParseInsight("```json\n{\"summary\":\"Tanya SEO\",\"topic\":\"Tanya Harga\",\"sentiment\":\"positive\",\"is_complaint\":false,\"keywords\":[\"seo\",\"harga seo\",\"ya\",\"harga seo\"]}\n```")
	eq(t, []any{in.Topic, in.Sentiment, in.Keywords}, []any{"tanya harga", "positif", []string{"seo", "harga seo"}})
	eq(t, ParseInsight("bukan json") == nil, true)
	mock := MockInsight(msgs)
	eq(t, []any{mock.Topic, mock.Keywords}, []any{"tanya harga", []string{"seo"}})
	long := TranscriptText([]TranscriptMessage{{"in", strings.Repeat("a", 7000)}}, 100)
	eq(t, strings.Contains(long, "dipotong"), true)
}

func TestOnboardingAndFiles(t *testing.T) {
	eq(t, OnboardingProgress([]string{"done", "skipped", "todo", "todo"}), 50)
	eq(t, OnboardingProgress(nil), 0)
	_, errs := CleanBrief(map[string]string{"brand_name": "X"})
	eq(t, errs["goals"] != "", true)
	ct, ext, err := SniffAsset([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0}, "a.png")
	eq(t, []any{ct, ext, err}, []any{"image/png", "png", nil})
	_, _, err = SniffAsset([]byte("MZ\x90\x00 evil"), "a.exe")
	eq(t, err, ErrAssetType)
	_, _, err = SniffAsset([]byte(`<svg onload="alert(1)"></svg>`), "a.svg")
	eq(t, err, ErrAssetType)
	eq(t, SafeFileName("../../etc/passwd"), "passwd")
	_, err = DecodeSignature("data:image/jpeg;base64,AAAA")
	eq(t, err, ErrSignature)
	eq(t, DueDate(time.Date(2026, 10, 26, 20, 0, 0, 0, time.UTC), 1).Format("2006-01-02"), "2026-10-28")
}

func TestFunnelReport(t *testing.T) {
	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	in := from.AddDate(0, 0, 5)
	closed := from.AddDate(0, 0, 15)
	leadID := "l1"
	v := rp("10000000")
	stages := []Stage{{ID: "s1", Name: "Prospek", SortOrder: 1}, {ID: "s2", Name: "Proposal", SortOrder: 2}, {ID: "w", IsWon: true}, {ID: "x", IsLost: true}}
	leads := []Lead{{ID: "l1", Source: "wa", Status: "qualified", Audit: Audit{CreatedAt: in}}, {ID: "l2", Source: "wa", Audit: Audit{CreatedAt: in}}, {ID: "l3", Source: "google", Audit: Audit{CreatedAt: in}}}
	deals := []Deal{
		{ID: "d1", LeadID: &leadID, StageID: "w", IsWon: true, ClosedAt: &closed, ValueFinal: &v, ServiceLines: []string{LineSEO}, Audit: Audit{CreatedAt: in}},
		{ID: "d2", StageID: "x", IsLost: true, ClosedAt: &closed, LostReasonID: ptr("lr-harga"), Audit: Audit{CreatedAt: in}},
		{ID: "d3", StageID: "s2", ValueEstimate: &v, Audit: Audit{CreatedAt: in}},
	}
	r := BuildFunnelReport(leads, deals, stages, []LostReason{{ID: "lr-harga", Name: "Harga"}}, from, to)
	eq(t, []int{r.Leads, r.QualifiedLeads, r.DealsCreated, r.DealsWon, r.DealsLost}, []int{3, 1, 3, 1, 1})
	eq(t, r.WinRatePercent, 50.0)
	eq(t, r.AvgCycleDays, 10.0)
	eq(t, r.Sources[0], SourceBucket{Source: "wa", Leads: 2, Converted: 1})
	eq(t, r.Stages[1].Deals, 1)
	eq(t, r.LostReasons[0].Name, "Harga")
	eq(t, r.ServiceLineWins[LineSEO], 1)
}

func TestDeck(t *testing.T) {
	in := DeckInput{ClientName: "PT Maju", Notes: "Penjualan turun 20%. Kompetitor agresif di TikTok.", ServiceLines: []string{LineSocialMedia, LineSEO}, Options: DeckOptions{IncludePricing: true, IncludeCaseStudies: true},
		CaseStudies: []CaseStudy{{Title: "Kopi", ClientName: "Kopi", Challenge: "a", Solution: "b", Results: "c"}}}
	c := MockDeckCopy(in)
	eq(t, c.Challenges, []string{"Penjualan turun 20%", "Kompetitor agresif di TikTok"})
	slides := BuildSlides(in, c)
	kinds := []string{}
	for _, s := range slides {
		kinds = append(kinds, s.Kind)
	}
	eq(t, kinds, []string{"cover", "about", "challenges", "strategy", "strategy", "timeline", "team", "pricing", "case_study", "next_steps"})
	_, ok := ParseDeckCopy(`{"tagline":"x","challenges":["a"],"strategy":{},"timeline":[],"next_steps":[]}`)
	eq(t, ok, true)
	_, ok = ParseDeckCopy("nope")
	eq(t, ok, false)
}
