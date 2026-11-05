import type { Metadata } from "next";
import { KpiTile } from "@/components/dashboard/kpi-tile";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { FollowupQueue } from "@/features/erp/finance/components/followups";
import { fs } from "@/features/erp/finance/strings";
import { InvoicesTable } from "@/features/erp/sales/components/billing";
import { salesCaps } from "@/features/erp/sales/lib";
import { listInvoiceFollowups, listInvoices } from "@/features/erp/sales/queries";
import { load, sp1 } from "@/features/erp/shared/load";
import { formatRupiah, sumMoney } from "@/features/erp/shared/money";
import { ErpLoadError } from "@/features/erp/shared/states";
import { matchesText } from "@/features/erp/shared/list-filter";
import { UrlTabs } from "@/features/erp/shared/url-tabs";

const s = fs.salesInvoices;
export const metadata: Metadata = { title: s.title };

export default async function FinanceInvoicesPage(props: PageProps<"/erp/finance/invoices">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("finance.read");
  const header = <PageHeader eyebrow={fs.eyebrow} title={s.title} description={s.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;

  const tabs = (
    <UrlTabs
      label={s.title}
      tabs={[
        { value: "list", label: fs.followups.tabInvoices },
        { value: "followups", label: fs.followups.tab },
      ]}
      reset={["q", "status", "from", "to"]}
    />
  );
  const caps = salesCaps(access);
  if (sp1(sp.tab) === "followups") {
    const queue = await load(() => listInvoiceFollowups());
    return (
      <>
        {header}
        {tabs}
        {queue.error ? <ErpLoadError error={queue.error} /> : <FollowupQueue rows={(queue.data.followups ?? []).filter((f) => matchesText(sp1(sp.q), f.invoiceNumber, f.clientName) && (!sp1(sp.status) || f.status === sp1(sp.status)) && (!sp1(sp.stage) || String(f.stage) === sp1(sp.stage)))} canWrite={caps.finance} whatsappEnabled={queue.data.whatsappEnabled} />}
      </>
    );
  }

  const res = await load(() => listInvoices({ q: sp1(sp.q), status: sp1(sp.status), dateFrom: sp1(sp.from), dateTo: sp1(sp.to) }));
  if (res.error) return <>{header}<ErpLoadError error={res.error} /></>;

  const live = res.data.filter((i) => i.status !== "batal");
  const billed = sumMoney(live.map((i) => i.amount));
  const paid = sumMoney(live.map((i) => i.paidTotal));
  return (
    <>
      {header}
      {tabs}
      <div className="grid grid-cols-2 gap-(--gap) md:grid-cols-4">
        <KpiTile label={s.toIssue} value={String(live.filter((i) => i.status === "diajukan").length)} size="sm" />
        <KpiTile label={s.billed} value={formatRupiah(billed)} size="sm" />
        <KpiTile label={s.paid} value={formatRupiah(paid)} size="sm" />
        <KpiTile label={s.outstanding} value={formatRupiah(billed - paid)} size="sm" />
      </div>
      <InvoicesTable invoices={res.data} caps={caps} finance />
    </>
  );
}
