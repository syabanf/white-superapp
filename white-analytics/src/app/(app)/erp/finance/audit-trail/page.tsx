import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { AuditTable } from "@/features/erp/finance/components/audit-trail";
import { listAuditTrail } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1, spInt } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";

const s = fs.auditTrail;
export const metadata: Metadata = { title: s.title };

export default async function AuditTrailPage(props: PageProps<"/erp/finance/audit-trail">) {
  const sp = await props.searchParams;
  const { offline } = await requireErp("finance.approve");
  const header = <PageHeader eyebrow={fs.eyebrow} title={s.title} description={s.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;

  const page = spInt(sp.page, 1);
  const pageSize = spInt(sp.pageSize, 50);
  const filters = { q: sp1(sp.q), result: sp1(sp.result), dateFrom: sp1(sp.from), dateTo: sp1(sp.to) };
  const res = await load(() => listAuditTrail({ ...filters, page, pageSize }));
  const query = new URLSearchParams(Object.entries(filters).filter((e): e is [string, string] => Boolean(e[1]))).toString();
  return (
    <>
      {header}
      {res.error ? <ErpLoadError error={res.error} /> : <AuditTable rows={res.data.entries ?? []} page={{ page, pageSize, total: res.data.total ?? 0 }} query={query} />}
    </>
  );
}
