import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { ControlsCard, DeferralDialog, DeferralsTable, PositionsTable, RemittancesTable, TaxActions } from "@/features/erp/finance/components/period-end";
import { financePerms } from "@/features/erp/finance/perms";
import { getSeparateApprover, listDeferrals, listPostableAccounts, listTaxPositions, listTaxRemittances } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";

const s = fs.periodEnd;
export const metadata: Metadata = { title: s.title };

export default async function PeriodEndPage(props: PageProps<"/erp/finance/period-end">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("finance.read");
  const { canApprove } = financePerms(access);
  const header = (actions?: React.ReactNode) => <PageHeader eyebrow={fs.eyebrow} title={s.title} description={s.subtitle} actions={actions} />;
  if (offline) return <>{header()}<OfflineNotice message={offline.message} /></>;

  const tab = sp1(sp.tab);
  const tabs = <UrlTabs label={s.title} tabs={[{ value: "deferrals", label: s.tabDeferrals }, { value: "tax", label: s.tabTax }, { value: "controls", label: s.tabControls }]} />;

  if (tab === "controls") {
    const res = await load(getSeparateApprover);
    return <>{header()}{tabs}{res.error ? <ErpLoadError error={res.error} /> : <ControlsCard separateApprover={res.data} canApprove={canApprove} />}</>;
  }

  if (tab === "tax") {
    const res = await load(() => Promise.all([listTaxPositions(), listTaxRemittances(), listPostableAccounts()]));
    if (res.error) return <>{header()}{tabs}<ErpLoadError error={res.error} /></>;
    const [positions, remittances, accounts] = res.data;
    return (
      <>
        {header(canApprove ? <TaxActions positions={positions} accounts={accounts} /> : null)}
        {tabs}
        <section className="flex flex-col gap-2">
          <h2 className="text-sm font-semibold">{s.positions}</h2>
          <p className="text-xs text-muted-foreground">{s.positionsHint}</p>
          <PositionsTable rows={positions} />
        </section>
        <section className="flex flex-col gap-2">
          <h2 className="text-sm font-semibold">{s.remittances}</h2>
          <RemittancesTable rows={remittances} canApprove={canApprove} />
        </section>
      </>
    );
  }

  const res = await load(() => Promise.all([listDeferrals(), listPostableAccounts()]));
  if (res.error) return <>{header()}{tabs}<ErpLoadError error={res.error} /></>;
  return (
    <>
      {header(canApprove ? <DeferralDialog accounts={res.data[1]} /> : null)}
      {tabs}
      <DeferralsTable rows={res.data[0]} canApprove={canApprove} />
    </>
  );
}
