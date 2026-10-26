// Package seed loads finance reference data (account types, a standard
// Indonesian agency chart of accounts, journal mappings, reimbursement
// policies) and demo transactions for WHITE. Every step is idempotent.
package seed

import (
	"context"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

type coaRow struct {
	code, name, typeCode, cashFlow string
	cashBank, contra               bool
}

// chart is the standard chart of accounts of an Indonesian digital agency
// (7-digit codes, display "1 1 01 001").
var chart = []coaRow{
	{"1000000", "ASET", domain.TypeAsset, "", false, false},
	{"1100000", "Aset Lancar", domain.TypeAsset, "", false, false},
	{"1101000", "Kas", domain.TypeAsset, "", false, false},
	{"1101001", "Kas Kecil", domain.TypeAsset, "OPERATING", true, false},
	{"1101002", "Kas Besar", domain.TypeAsset, "OPERATING", true, false},
	{"1102000", "Bank", domain.TypeAsset, "", false, false},
	{"1102001", "Bank BCA", domain.TypeAsset, "OPERATING", true, false},
	{"1102002", "Bank Mandiri", domain.TypeAsset, "OPERATING", true, false},
	{"1103000", "Piutang", domain.TypeAsset, "", false, false},
	{"1103001", "Piutang Usaha", domain.TypeAsset, "OPERATING", false, false},
	{"1103002", "Piutang Karyawan", domain.TypeAsset, "OPERATING", false, false},
	{"1104000", "Pajak Dibayar di Muka", domain.TypeAsset, "", false, false},
	{"1104001", "PPN Masukan", domain.TypeAsset, "OPERATING", false, false},
	{"1104002", "PPh 23 Dibayar di Muka", domain.TypeAsset, "OPERATING", false, false},
	{"1105000", "Biaya Dibayar di Muka", domain.TypeAsset, "", false, false},
	{"1105001", "Sewa Dibayar di Muka", domain.TypeAsset, "OPERATING", false, false},
	{"1200000", "Aset Tetap", domain.TypeAsset, "", false, false},
	{"1201000", "Peralatan", domain.TypeAsset, "", false, false},
	{"1201001", "Peralatan Kantor dan Komputer", domain.TypeAsset, "INVESTING", false, false},
	{"1201002", "Akumulasi Penyusutan Peralatan", domain.TypeAsset, "NON_CASH", false, true},
	{"2000000", "LIABILITAS", domain.TypeLiability, "", false, false},
	{"2100000", "Liabilitas Jangka Pendek", domain.TypeLiability, "", false, false},
	{"2101000", "Hutang Usaha", domain.TypeLiability, "", false, false},
	{"2101001", "Hutang Usaha", domain.TypeLiability, "OPERATING", false, false},
	{"2102000", "Hutang Gaji", domain.TypeLiability, "", false, false},
	{"2102001", "Hutang Gaji", domain.TypeLiability, "OPERATING", false, false},
	{"2103000", "Hutang Pajak", domain.TypeLiability, "", false, false},
	{"2103001", "Hutang PPh 21", domain.TypeLiability, "OPERATING", false, false},
	{"2103002", "Hutang PPh 23", domain.TypeLiability, "OPERATING", false, false},
	{"2103003", "Hutang PPN Keluaran", domain.TypeLiability, "OPERATING", false, false},
	{"2104000", "Hutang Iuran", domain.TypeLiability, "", false, false},
	{"2104001", "Hutang BPJS", domain.TypeLiability, "OPERATING", false, false},
	{"2104002", "Hutang Tapera", domain.TypeLiability, "OPERATING", false, false},
	{"2105000", "Pendapatan Diterima di Muka", domain.TypeLiability, "", false, false},
	{"2105001", "Uang Muka Klien", domain.TypeLiability, "OPERATING", false, false},
	{"3000000", "EKUITAS", domain.TypeEquity, "", false, false},
	{"3100000", "Modal", domain.TypeEquity, "", false, false},
	{"3101000", "Modal Disetor", domain.TypeEquity, "", false, false},
	{"3101001", "Modal Disetor", domain.TypeEquity, "FINANCING", false, false},
	{"3200000", "Saldo Laba", domain.TypeEquity, "", false, false},
	{"3201000", "Laba Ditahan", domain.TypeEquity, "", false, false},
	{"3201001", "Laba Ditahan", domain.TypeEquity, "FINANCING", false, false},
	{"4000000", "PENDAPATAN", domain.TypeRevenue, "", false, false},
	{"4100000", "Pendapatan Usaha", domain.TypeRevenue, "", false, false},
	{"4101000", "Pendapatan Jasa", domain.TypeRevenue, "", false, false},
	{"4101001", "Pendapatan Jasa Digital Marketing", domain.TypeRevenue, "OPERATING", false, false},
	{"4101002", "Pendapatan Jasa Social Media Management", domain.TypeRevenue, "OPERATING", false, false},
	{"4101003", "Pendapatan Jasa SEO", domain.TypeRevenue, "OPERATING", false, false},
	{"4101004", "Pendapatan Jasa Ads Management", domain.TypeRevenue, "OPERATING", false, false},
	{"4101005", "Pendapatan Jasa Kreatif dan Desain", domain.TypeRevenue, "OPERATING", false, false},
	{"5000000", "HARGA POKOK PENJUALAN", domain.TypeCOGS, "", false, false},
	{"5100000", "Biaya Langsung Proyek", domain.TypeCOGS, "", false, false},
	{"5101000", "Biaya Media dan Produksi", domain.TypeCOGS, "", false, false},
	{"5101001", "Biaya Media Iklan Klien", domain.TypeCOGS, "OPERATING", false, false},
	{"5101002", "Biaya Freelancer dan Produksi", domain.TypeCOGS, "OPERATING", false, false},
	{"6000000", "BEBAN OPERASIONAL", domain.TypeExpense, "", false, false},
	{"6100000", "Beban Karyawan", domain.TypeExpense, "", false, false},
	{"6101000", "Beban Gaji dan Tunjangan", domain.TypeExpense, "", false, false},
	{"6101001", "Beban Gaji", domain.TypeExpense, "OPERATING", false, false},
	{"6101002", "Beban BPJS", domain.TypeExpense, "OPERATING", false, false},
	{"6101003", "Beban Tapera", domain.TypeExpense, "OPERATING", false, false},
	{"6200000", "Beban Pemasaran", domain.TypeExpense, "", false, false},
	{"6201000", "Beban Iklan", domain.TypeExpense, "", false, false},
	{"6201001", "Beban Iklan", domain.TypeExpense, "OPERATING", false, false},
	{"6300000", "Beban Umum dan Administrasi", domain.TypeExpense, "", false, false},
	{"6301000", "Beban Kantor", domain.TypeExpense, "", false, false},
	{"6301001", "Beban Sewa Kantor", domain.TypeExpense, "OPERATING", false, false},
	{"6301002", "Beban Listrik, Internet dan Telepon", domain.TypeExpense, "OPERATING", false, false},
	{"6301003", "Beban Software dan Langganan", domain.TypeExpense, "OPERATING", false, false},
	{"6301004", "Beban Perlengkapan Kantor", domain.TypeExpense, "OPERATING", false, false},
	{"6302000", "Beban Reimbursement", domain.TypeExpense, "", false, false},
	{"6302001", "Beban Transport", domain.TypeExpense, "OPERATING", false, false},
	{"6302002", "Beban Makan dan Konsumsi", domain.TypeExpense, "OPERATING", false, false},
	{"6302003", "Beban Akomodasi dan Perjalanan", domain.TypeExpense, "OPERATING", false, false},
	{"6302004", "Beban Operasional Proyek", domain.TypeExpense, "OPERATING", false, false},
	{"6302005", "Beban Reimbursement Lainnya", domain.TypeExpense, "OPERATING", false, false},
	{"6303000", "Beban Penyusutan", domain.TypeExpense, "", false, false},
	{"6303001", "Beban Penyusutan Peralatan", domain.TypeExpense, "NON_CASH", false, false},
	{"8000000", "PENDAPATAN DAN BEBAN LAIN-LAIN", domain.TypeOtherExpense, "", false, false},
	{"8100000", "Beban Lain-lain", domain.TypeOtherExpense, "", false, false},
	{"8101000", "Beban Bank", domain.TypeOtherExpense, "", false, false},
	{"8101001", "Biaya Administrasi Bank", domain.TypeOtherExpense, "OPERATING", false, false},
	{"8200000", "Pendapatan Lain-lain", domain.TypeOtherIncome, "", false, false},
	{"8201000", "Pendapatan Bunga", domain.TypeOtherIncome, "", false, false},
	{"8201001", "Pendapatan Bunga Bank", domain.TypeOtherIncome, "OPERATING", false, false},
}

type mappingLine struct {
	side, role, code, source string
	required                 bool
}

type mappingDef struct {
	event string
	lines []mappingLine
}

// mappings are the journal templates WHITE posts automatically.
var mappings = []mappingDef{
	{domain.EventPayrollAccrual, []mappingLine{{domain.Debit, "EXPENSE", "6101001", "TOTAL", true}, {domain.Credit, "PAYABLE", "2102001", "TOTAL", true}}},
	{domain.EventPayrollPPh21, []mappingLine{{domain.Debit, "PAYABLE", "2102001", "TAX", true}, {domain.Credit, "TAX_PAYABLE", "2103001", "TAX", true}}},
	{domain.EventPayrollLoanDeduction, []mappingLine{{domain.Debit, "PAYABLE", "2102001", "PAID", true}, {domain.Credit, "RECEIVABLE", "1103002", "PAID", true}}},
	{domain.EventPayrollBpjsTkEmployee, []mappingLine{{domain.Debit, "PAYABLE", "2102001", "TOTAL", true}, {domain.Credit, "BPJS_PAYABLE", "2104001", "TOTAL", true}}},
	{domain.EventPayrollBpjsKesEmployee, []mappingLine{{domain.Debit, "PAYABLE", "2102001", "TOTAL", true}, {domain.Credit, "BPJS_PAYABLE", "2104001", "TOTAL", true}}},
	{domain.EventPayrollTaperaEmployee, []mappingLine{{domain.Debit, "PAYABLE", "2102001", "TOTAL", true}, {domain.Credit, "TAPERA_PAYABLE", "2104002", "TOTAL", true}}},
	{domain.EventPayrollPayment, []mappingLine{{domain.Debit, "PAYABLE", "2102001", "PAID", true}, {domain.Credit, "CASH", "1102001", "PAID", true}}},
	{domain.EventPayrollBpjsTkEmployer, []mappingLine{{domain.Debit, "EXPENSE", "6101002", "TOTAL", true}, {domain.Credit, "BPJS_PAYABLE", "2104001", "TOTAL", true}}},
	{domain.EventPayrollBpjsKesEmployer, []mappingLine{{domain.Debit, "EXPENSE", "6101002", "TOTAL", true}, {domain.Credit, "BPJS_PAYABLE", "2104001", "TOTAL", true}}},
	{domain.EventPayrollTaperaEmployer, []mappingLine{{domain.Debit, "EXPENSE", "6101003", "TOTAL", true}, {domain.Credit, "TAPERA_PAYABLE", "2104002", "TOTAL", true}}},
	{domain.EventSaleARInvoice, []mappingLine{{domain.Debit, "RECEIVABLE", "1103001", "TOTAL", true}, {domain.Credit, "REVENUE", "4101001", "SUBTOTAL", true}, {domain.Credit, "TAX", "2103003", "TAX", false}}},
	{domain.EventSaleARReceipt, []mappingLine{{domain.Debit, "CASH", "1102001", "PAID", true}, {domain.Credit, "RECEIVABLE", "1103001", "PAID", true}}},
	{domain.EventPurchaseAPInvoice, []mappingLine{{domain.Debit, "EXPENSE", "6301004", "SUBTOTAL", true}, {domain.Debit, "TAX", "1104001", "TAX", false}, {domain.Credit, "PAYABLE", "2101001", "TOTAL", true}}},
	{domain.EventPurchasePayment, []mappingLine{{domain.Debit, "PAYABLE", "2101001", "PAID", true}, {domain.Credit, "CASH", "1102001", "PAID", true}}},
	{domain.EventReimbursementPayment, []mappingLine{{domain.Debit, "EXPENSE", "6302005", "TOTAL", true}, {domain.Credit, "CASH", "1102001", "PAID", true}}},
}

type policyDef struct {
	category          string
	monthly, perClaim int64
	account           string
	autoApprove       bool
}

var policies = []policyDef{
	{domain.CategoryTransport, 1_500_000, 500_000, "6302001", true},
	{domain.CategoryMakan, 1_000_000, 300_000, "6302002", true},
	{domain.CategoryAkomodasi, 5_000_000, 2_500_000, "6302003", false},
	{domain.CategoryOperasional, 5_000_000, 2_000_000, "6302004", false},
	{domain.CategoryLainnya, 1_000_000, 500_000, "6302005", false},
}

// Reference loads account types, the chart, mappings and policies. Existing
// rows are kept as they are, so edits made in the UI survive a re-run.
func Reference(ctx context.Context, d app.Deps) error {
	s := d.Store
	types, err := s.ListAccountTypes(ctx, true)
	if err != nil {
		return err
	}
	typeID := map[string]string{}
	for _, t := range types {
		typeID[t.Code] = t.ID
	}
	for _, t := range domain.StandardAccountTypes {
		if typeID[t.Code] != "" {
			continue
		}
		if err := s.UpsertAccountType(ctx, t, ""); err != nil {
			return err
		}
		typeID[t.Code] = t.ID
	}
	accounts, err := s.AccountsByCode(ctx)
	if err != nil {
		return err
	}
	codes := map[string]int{}
	for i, r := range chart {
		codes[r.code] = i
	}
	err = s.WithTx(ctx, func(tx domain.Store) error {
		for _, r := range chart {
			if _, ok := accounts[r.code]; ok {
				continue
			}
			parent := accounts[domain.NearestParentCode(r.code, codes)]
			a := domain.Account{ID: ids.New(), Code: r.code, Name: r.name, ParentID: parent.ID, AccountTypeID: typeID[r.typeCode],
				Level: domain.InferAccountLevel(r.code), IsContra: r.contra, IsCashBank: r.cashBank, CashFlowCategory: r.cashFlow, IsActive: true}
			if err := tx.InsertAccount(ctx, a, ""); err != nil {
				return err
			}
			accounts[r.code] = a
		}
		return tx.RecomputeAllPostable(ctx)
	})
	if err != nil {
		return err
	}
	events := map[string]domain.JournalEvent{}
	for _, e := range domain.JournalEvents {
		events[e.Code] = e
	}
	for _, m := range mappings {
		if existing, err := s.ListMappings(ctx, domain.MappingFilter{Search: m.event}); err != nil {
			return err
		} else if hasEvent(existing, m.event) {
			continue
		}
		ev := events[m.event]
		var lines []domain.MappingLine
		for i, l := range m.lines {
			lines = append(lines, domain.MappingLine{Side: l.side, LineRole: l.role, AccountID: accounts[l.code].ID, AmountSource: l.source, SortOrder: (i + 1) * 10, IsRequired: l.required})
		}
		mapping := domain.JournalMapping{ID: ids.New(), EventCode: m.event, Name: ev.Name, Description: ev.Description, Module: ev.Module, IsActive: true}
		err := s.WithTx(ctx, func(tx domain.Store) error {
			if err := tx.InsertMapping(ctx, mapping, ""); err != nil {
				return err
			}
			return tx.ReplaceMappingLines(ctx, mapping.ID, lines)
		})
		if err != nil {
			return err
		}
	}
	for _, p := range policies {
		if existing, err := s.GetPolicy(ctx, p.category); err != nil {
			return err
		} else if existing != nil {
			continue
		}
		pol := domain.Policy{Category: p.category, Label: domain.CategoryLabel(p.category), MonthlyLimit: domain.Rupiah(p.monthly),
			PerClaimLimit: domain.Rupiah(p.perClaim), ExpenseAccountID: accounts[p.account].ID, AutoApprove: p.autoApprove, IsActive: true}
		if err := s.UpsertPolicy(ctx, pol, ""); err != nil {
			return err
		}
	}
	return nil
}

func hasEvent(ms []domain.JournalMapping, event string) bool {
	for _, m := range ms {
		if m.EventCode == event {
			return true
		}
	}
	return false
}
