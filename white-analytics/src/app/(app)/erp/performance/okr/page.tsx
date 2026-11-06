import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { PageHeader } from "@/components/dashboard/page-header";
import { canAccess } from "@/components/shell/nav-config";
import { requireErp } from "@/features/erp/access";
import { OkrBoardView } from "@/features/erp/performance/okr/board";
import { getOkrBoard } from "@/features/erp/performance/okr/queries";
import { quarterOf, to } from "@/features/erp/performance/okr/strings";
import { sp } from "@/features/erp/performance/strings";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";

export const metadata: Metadata = { title: to.title };

export default async function OkrPage(props: PageProps<"/erp/performance/okr">) {
  const q = await props.searchParams;
  const { access, offline } = await requireErp();
  if (!canAccess(access, ["hr.read", "self.service"])) redirect("/erp?unauthorized=1");
  const header = <PageHeader eyebrow={sp.common.eyebrow} title={to.title} description={to.subtitle} />;
  if (offline) return <>{header}<OfflineNotice message={offline.message} /></>;
  const raw = sp1(q.period);
  const period = raw && /^\d{4}-Q[1-4]$/.test(raw) ? raw : quarterOf(new Date());
  const res = await load(() => getOkrBoard(period));
  return (
    <>
      {header}
      {res.data ? <OkrBoardView board={res.data} period={period} /> : <ErpError message={res.error.message} />}
    </>
  );
}
