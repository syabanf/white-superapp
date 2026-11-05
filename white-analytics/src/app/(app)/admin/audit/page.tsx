import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { AuditTable, type AuditLog } from "@/features/admin/components/audit-table";
import { load, qs, sp1, spInt } from "@/features/erp/shared/load";
import { ErpLoadError } from "@/features/erp/shared/states";
import { gatewayFetch } from "@/lib/erp/gateway";
import { requireAdmin } from "@/lib/rbac";
import { t } from "@/i18n/id";

export const metadata: Metadata = { title: t.nav.auditTrail };

/** Admin-only change log: who changed what through the ERP API, and the result. */
export default async function AuditPage(props: PageProps<"/admin/audit">) {
  await requireAdmin();
  const sp = await props.searchParams;
  const page = spInt(sp.page, 1);
  const pageSize = spInt(sp.pageSize, 50);
  const res = await load(() =>
    gatewayFetch<{ logs: AuditLog[]; total: number }>(
      `/v1/audit-logs${qs({ q: sp1(sp.q), failedOnly: sp1(sp.failed) === "1" || undefined, dateFrom: sp1(sp.from), dateTo: sp1(sp.to), page, pageSize })}`,
    ),
  );
  return (
    <>
      <PageHeader eyebrow={t.nav.admin} title={t.nav.auditTrail} description="Setiap perubahan data lewat API ERP: siapa, kapan, apa, dan hasilnya. Isi data tidak disimpan." />
      {res.error ? <ErpLoadError error={res.error} /> : <AuditTable logs={res.data.logs ?? []} page={{ page, pageSize, total: res.data.total }} />}
    </>
  );
}
