package domain

import "testing"

func TestParseStatementCSV(t *testing.T) {
	t.Run("debit and credit columns with a bank preamble", func(t *testing.T) {
		rows, err := ParseStatementCSV([]byte("Rekening,123\nPeriode,Nov\nTanggal;Keterangan;Debit;Kredit\n" +
			"03/11/2026;TRF DARI BUDI;;1.250.000,50\n04/11/2026;BIAYA ADM;15.000;\nSaldo Akhir;;;\n"))
		if err == nil && len(rows) == 0 {
			t.Fatal("no rows")
		}
		if err != nil {
			t.Fatal(err)
		}
		if rows[0].Date != "2026-11-03" || rows[0].Amount != 125000050 || rows[1].Amount != -1500000 {
			t.Fatalf("%+v", rows)
		}
	})
	t.Run("one amount column with CR and DB marks", func(t *testing.T) {
		rows, err := ParseStatementCSV([]byte("Tanggal,Keterangan,Mutasi\n2026-11-03,SETORAN,\"1,000,000.00 CR\"\n2026-11-04,TARIK,\"200,000.00 DB\"\n"))
		if err != nil {
			t.Fatal(err)
		}
		if rows[0].Amount != 100000000 || rows[1].Amount != -20000000 {
			t.Fatalf("%+v", rows)
		}
	})
	t.Run("identical rows stay apart, the same file keeps its keys", func(t *testing.T) {
		data := []byte("Tanggal,Keterangan,Debit,Kredit\n2026-11-03,QRIS,,50.000\n2026-11-03,QRIS,,50.000\n")
		a, err := ParseStatementCSV(data)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := ParseStatementCSV(data)
		if a[0].DedupeKey == a[1].DedupeKey || a[0].DedupeKey != b[0].DedupeKey {
			t.Fatalf("keys %+v", a)
		}
	})
	t.Run("unknown columns are refused", func(t *testing.T) {
		if _, err := ParseStatementCSV([]byte("a,b\n1,2\n")); err == nil {
			t.Fatal("want error")
		}
	})
}

func TestParseBankMoney(t *testing.T) {
	for in, want := range map[string]Money{"1.250.000,50": 125000050, "1,250,000.50": 125000050, "1.000.000": 100000000,
		"1,000": 100000, "1000": 100000, "1000.50": 100050, "Rp 15.000": 1500000, "(5.000)": -500000, "": 0} {
		got, err := parseBankMoney(in)
		if err != nil || got != want {
			t.Errorf("%q = %d, %v; want %d", in, got, err, want)
		}
	}
}

func TestDecide(t *testing.T) {
	line := StatementLine{LineDate: "2026-11-03", Description: "TRF DARI KOPI SENJA", Amount: 100}
	journal := ScoreCandidate(line, MatchCandidate{Kind: SuggestJournal, Date: "2026-11-03", Text: "Penerimaan Kopi Senja"})
	if status, top := Decide([]MatchCandidate{journal}); status != LineMatched || top.Score != 100 {
		t.Fatalf("exact journal: %s %+v", status, top)
	}
	nameless := ScoreCandidate(StatementLine{LineDate: "2026-11-03", Amount: 100}, MatchCandidate{Kind: SuggestJournal, Date: "2026-11-01", Text: "x"})
	if status, _ := Decide([]MatchCandidate{nameless}); status != LineSuggested {
		t.Fatalf("no name stays a suggestion: %s", status)
	}
	invoice := ScoreCandidate(line, MatchCandidate{Kind: SuggestInvoice, Text: "Kopi Senja"})
	if status, _ := Decide([]MatchCandidate{invoice}); status != LineSuggested {
		t.Fatalf("invoices never match alone: %s", status)
	}
	if status, top := Decide([]MatchCandidate{nameless, nameless}); status != LineUnmatched || top != nil {
		t.Fatalf("a tie is unmatched: %s", status)
	}
	if status, _ := Decide(nil); status != LineUnmatched {
		t.Fatal(status)
	}
}

func TestFindDuplicates(t *testing.T) {
	a, near, far := int64(0b1111), int64(0b1110), int64(0x7fffffff)
	own := []ReceiptPrint{{ReceiptID: "r1", ContentHash: "h1", ImageHash: &a}}
	if got := FindDuplicates(own, []ReceiptPrint{{ClaimNo: "RB-1", ContentHash: "h1"}}); len(got) != 1 || !got[0].Exact {
		t.Fatalf("exact %+v", got)
	}
	if got := FindDuplicates(own, []ReceiptPrint{{ClaimNo: "RB-2", ContentHash: "h2", ImageHash: &near}}); len(got) != 1 || got[0].Exact {
		t.Fatalf("similar %+v", got)
	}
	if got := FindDuplicates(own, []ReceiptPrint{{ClaimNo: "RB-3", ContentHash: "h3", ImageHash: &far}}); len(got) != 0 {
		t.Fatalf("different %+v", got)
	}
}

func TestNextRunAfter(t *testing.T) {
	for _, c := range []struct {
		day, freq string
		anchor    int
		want      string
	}{{"2026-01-31", FreqMonthly, 31, "2026-02-28"}, {"2026-02-28", FreqMonthly, 31, "2026-03-31"},
		{"2026-11-15", FreqQuarterly, 15, "2027-02-15"}, {"2026-03-01", FreqYearly, 1, "2027-03-01"}} {
		if got, err := NextRunAfter(c.day, c.freq, c.anchor); err != nil || got != c.want {
			t.Errorf("%+v = %s, %v", c, got, err)
		}
	}
}

func TestDepreciation(t *testing.T) {
	a := FixedAsset{AcquiredOn: "2026-01-15", Cost: Rupiah(50_000_000), Salvage: Rupiah(1), LifeMonths: 7}
	var total Money
	for n := 1; n <= 7; n++ {
		total += a.DepreciationFor(n)
	}
	if total != a.Cost-a.Salvage {
		t.Fatalf("months add up to %d", total)
	}
	if end, _ := a.MonthEnd(2); end != "2026-02-28" {
		t.Fatal(end)
	}
	if end, _ := a.MonthEnd(1); end != "2026-01-31" {
		t.Fatal(end)
	}
}
