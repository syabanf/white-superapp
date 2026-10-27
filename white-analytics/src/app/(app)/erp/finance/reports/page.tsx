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
  BalanceSheetView,
  CashFlowView,
  IncomeStatementView,
  REPORTS,
  ReportHub,
  SubsidiaryLedgerView,
  TrialBalanceView,
  type ReportKey,
} from "@/features/erp/finance/components/report-views";
import {
  getBalanceSheet,
  getCashFlow,
  getGeneralLedger,
  getIncomeStatement,
  getSubsidiaryLedger,
  getTrialBalance,
  listGeneralLedgerAccounts,
  listSubsidiaryParties,
} from "@/features/erp/finance/queries";
import type { LedgerKind } from "@/features/erp/finance/types";
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
    report === "trial-balance" || report === "balance-sheet" ? (
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
      const [parties, ledger] = await Promise.all([
        load(() => listSubsidiaryParties(kind)),
        party ? load(() => getSubsidiaryLedger(kind, party, from, to)) : null,
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
          </Toolbar>
          {parties.error ? <ErpLoadError error={parties.error} /> : null}
          {!ledger ? (
            <ErpEmpty icon={<Users className="size-5" />} title={r.pickParty} />
          ) : ledger.error ? (
            <ErpLoadError error={ledger.error} />
          ) : (
            <SubsidiaryLedgerView data={ledger.data} />
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
