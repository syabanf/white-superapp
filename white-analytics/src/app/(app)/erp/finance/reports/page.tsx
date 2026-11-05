import type { Metadata } from "next";
import { Toolbar } from "@/components/dashboard/layout";
import Link from "next/link";
import { BookOpen, Users } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { cn } from "@/lib/utils";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { formatRupiah } from "@/features/erp/shared/money";
import { DateFilter, DateRangeFilter } from "@/features/erp/shared/period-picker";
import { PrintButton } from "@/features/erp/shared/print";
import { ErpEmpty, ErpLoadError } from "@/features/erp/shared/states";
import { SearchSelectFilter } from "@/features/erp/shared/url-filters";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { LedgerTable } from "@/features/erp/finance/components/ledger-table";
import {
  BalanceHealthView,
  BalanceSheetView,
  BudgetVsActualView,
  CashFlowView,
  ClientEconomicsView,
  EquityChangesView,
  IncomeStatementView,
  REPORTS,
  ReportHub,
  SubsidiaryLedgerView,
  TrialBalanceView,
  type ReportKey,
} from "@/features/erp/finance/components/report-views";
import {
  getBalanceSheet,
  getBudgetVsActual,
  getClientEconomics,
  getEquityChanges,
  getCashFlow,
  getGeneralLedger,
  getIncomeStatement,
  getSubsidiaryLedger,
  getTrialBalance,
  listGeneralLedgerAccounts,
  listPostableAccounts,
  listSubsidiaryParties,
} from "@/features/erp/finance/queries";
import type { LedgerKind } from "@/features/erp/finance/types";
import { SubsidiaryColumnToggles } from "@/features/erp/finance/components/subsidiary-columns";
import { getRatioTrend } from "@/features/erp/finance/dashboard/queries";
import { fs } from "@/features/erp/finance/strings";

export const metadata: Metadata = { title: fs.reports.title };

const r = fs.reports;

export default async function ReportsPage(props: PageProps<"/erp/finance/reports">) {
  const sp = await props.searchParams;
  const { offline } = await requireErp("finance.read");
  const report = REPORTS.find((x) => x.key === sp1(sp.r))?.key as ReportKey | undefined;
  const asOf = sp1(sp.asOf);
  const from = sp1(sp.from);
  const to = sp1(sp.to);

  const filters =
    report === "trial-balance" || report === "balance-sheet" || report === "balance-health" ? (
      <DateFilter param="asOf" label={fs.asOf} />
    ) : report ? (
      <DateRangeFilter />
    ) : null;

  const header = (
    <PageHeader
      eyebrow={fs.eyebrow}
      title={report ? REPORTS.find((x) => x.key === report)!.label : r.title}
      description={report ? r.descriptions[report] : r.subtitle}
      actions={
        report ? (
          <div className="flex flex-wrap items-center gap-2" data-print-hide>
            {filters}
            {report === "budget-vs-actual" ? (
              <Button asChild size="sm" variant="outline">
                <Link href="/erp/finance/budgets">{r.editBudget}</Link>
              </Button>
            ) : null}
            <PrintButton label={fs.print} />
          </div>
        ) : null
      }
    />
  );
  const nav = report ? (
    <nav className="flex flex-wrap gap-1.5" data-print-hide aria-label={r.title}>
      {REPORTS.map((x) => (
        <Button key={x.key} asChild size="xs" variant={x.key === report ? "default" : "outline"} className={cn("rounded-full")}>
          <Link href={`/erp/finance/reports?r=${x.key}`}>{x.label}</Link>
        </Button>
      ))}
    </nav>
  ) : null;

  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  if (!report) return <>{header}<ReportHub /></>;

  let body: React.ReactNode = null;
  switch (report) {
    case "trial-balance": {
      const res = await load(() => getTrialBalance(asOf));
      body = res.error ? <ErpLoadError error={res.error} /> : <TrialBalanceView data={res.data} />;
      break;
    }
    case "balance-sheet": {
      const res = await load(() => getBalanceSheet(asOf));
      body = res.error ? <ErpLoadError error={res.error} /> : <BalanceSheetView data={res.data} />;
      break;
    }
    case "income-statement": {
      const res = await load(() => getIncomeStatement(from, to));
      body = res.error ? <ErpLoadError error={res.error} /> : <IncomeStatementView data={res.data} />;
      break;
    }
    case "budget-vs-actual": {
      const res = await load(() => getBudgetVsActual(from, to));
      body = res.error ? <ErpLoadError error={res.error} /> : <BudgetVsActualView data={res.data} />;
      break;
    }
    case "equity-changes": {
      const res = await load(() => getEquityChanges(from, to));
      body = res.error ? <ErpLoadError error={res.error} /> : <EquityChangesView data={res.data} />;
      break;
    }
    case "client-economics": {
      const res = await load(() => getClientEconomics(from, to));
      body = res.error ? <ErpLoadError error={res.error} /> : <ClientEconomicsView data={res.data} />;
      break;
    }
    case "balance-health": {
      const day = asOf ?? new Date().toISOString().slice(0, 10);
      const res = await load(() => getRatioTrend(day, 3));
      body = res.error ? <ErpLoadError error={res.error} /> : <BalanceHealthView series={res.data.series ?? []} asOf={day} />;
      break;
    }
    case "cash-flow": {
      const res = await load(() => getCashFlow(from, to));
      body = res.error ? <ErpLoadError error={res.error} /> : <CashFlowView data={res.data} />;
      break;
    }
    case "general-ledger": {
      const account = sp1(sp.account);
      const [accounts, ledger] = await Promise.all([
        load(() => listGeneralLedgerAccounts(to)),
        account ? load(() => getGeneralLedger(account, from, to)) : null,
      ]);
      body = (
        <>
          <Toolbar label={fs.pickAccount} className="print:hidden">
            <SearchSelectFilter
              param="account"
              placeholder={fs.pickAccount}
              options={(accounts.data ?? []).map((a) => ({
                value: a.id,
                label: `${a.name} · ${formatRupiah(a.balance)}`,
                hint: a.codeDisplay || a.code,
              }))}
            />
          </Toolbar>
          {accounts.error ? <ErpLoadError error={accounts.error} /> : null}
          {!ledger ? (
            <ErpEmpty icon={<BookOpen className="size-5" />} title={r.pickAccount} />
          ) : ledger.error ? (
            <ErpLoadError error={ledger.error} />
          ) : (
            <Section title={`${ledger.data.account?.codeDisplay ?? ""} ${ledger.data.account?.name ?? ""}`}>
              <LedgerTable ledger={ledger.data} />
            </Section>
          )}
        </>
      );
      break;
    }
    case "subsidiary-ledger": {
      const kind: LedgerKind = sp1(sp.kind) === "AR" ? "AR" : "AP";
      const party = sp1(sp.party);
      const account = sp1(sp.account);
      const hidden = (sp1(sp.hide) ?? "").split(",").filter(Boolean);
      const [parties, coa, ledger] = await Promise.all([
        load(() => listSubsidiaryParties(kind)),
        load(listPostableAccounts),
        party || account ? load(() => getSubsidiaryLedger(kind, party, account, from, to)) : null,
      ]);
      body = (
        <>
          <Toolbar label={r.party} className="print:hidden">
            <UrlTabs
              param="kind"
              label={r.party}
              tabs={[
                { value: "AP", label: r.kindAP },
                { value: "AR", label: r.kindAR },
              ]}
              reset={["party"]}
            />
            <SearchSelectFilter
              param="party"
              placeholder={r.party}
              options={(parties.data ?? []).map((p) => ({
                value: p.partyName,
                label: `${p.partyName} · ${formatRupiah(p.outstanding)}`,
              }))}
            />
            <SearchSelectFilter
              param="account"
              placeholder={r.allCoa}
              options={(coa.data ?? []).map((a) => ({ value: a.id, label: `${a.codeDisplay} ${a.name}` }))}
            />
            <SubsidiaryColumnToggles hidden={hidden} />
          </Toolbar>
          {parties.error ? <ErpLoadError error={parties.error} /> : null}
          {!ledger ? (
            <ErpEmpty icon={<Users className="size-5" />} title={r.pickParty} />
          ) : ledger.error ? (
            <ErpLoadError error={ledger.error} />
          ) : (
            <SubsidiaryLedgerView data={ledger.data} hidden={hidden} />
          )}
        </>
      );
      break;
    }
  }
  return (
    <>
      {header}
      {nav}
      {body}
    </>
  );
}
