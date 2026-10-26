import type { Metadata } from "next";
import { KpiTile } from "@/components/dashboard/kpi-tile";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { formatRupiah, sumMoney } from "@/features/erp/shared/money";
import { ErpError } from "@/features/erp/shared/states";
import { InvoicesTable } from "@/features/erp/sales/components/billing";
import { salesCaps } from "@/features/erp/sales/lib";
import { listInvoices } from "@/features/erp/sales/queries";
import { INVOICE_STATUS, ss } from "@/features/erp/sales/strings";

const s = ss.invoices;
export const metadata: Metadata = { title: s.title };

export default async function InvoicesPage(props: PageProps<"/erp/sales/invoices">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const invoices = await load(() => listInvoices({ status: sp1(sp.status), dateFrom: sp1(sp.from), dateTo: sp1(sp.to), limit: 500 }));
  const live = (invoices.data ?? []).filter((i) => i.status !== "batal");
  const billed = sumMoney(live.map((i) => i.amount));
  const paid = sumMoney(live.map((i) => i.paidTotal));

  return (
    <>
      <PageHeader eyebrow={ss.eyebrow} title={s.title} description={s.subtitle} />
      {invoices.data ? (
        <>
          <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
            <KpiTile label={s.amount} value={formatRupiah(billed)} size="sm" />
            <KpiTile label={s.paid} value={formatRupiah(paid)} size="sm" />
            <KpiTile label={ss.reports.gap} value={formatRupiah(billed - paid)} size="sm" />
            <KpiTile
              label={INVOICE_STATUS.diajukan}
              value={String(live.filter((i) => i.status === "diajukan").length)}
              size="sm"
              caption={caps.finance ? undefined : s.financeOnly}
            />
          </div>
          <InvoicesTable invoices={invoices.data} caps={caps} />
        </>
      ) : (
        <ErpError message={invoices.error?.message} />
      )}
    </>
  );
}
