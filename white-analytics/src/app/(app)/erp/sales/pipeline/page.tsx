import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { ResetFilters, SearchFilter, SelectFilter } from "@/features/erp/shared/url-filters";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { PipelineBoard } from "@/features/erp/sales/components/pipeline-board";
import { salesCaps } from "@/features/erp/sales/lib";
import { listDeals, listLostReasons } from "@/features/erp/sales/queries";
import { dealFormRefs } from "@/features/erp/sales/refs";
import { ss } from "@/features/erp/sales/strings";

const s = ss.pipeline;
export const metadata: Metadata = { title: s.title };

export default async function PipelinePage(props: PageProps<"/erp/sales/pipeline">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const refs = await dealFormRefs();
  const pipelines = [...refs.pipelines].sort((a, b) => a.sortOrder - b.sortOrder);
  const fallback = pipelines.find((p) => p.isDefault) ?? pipelines[0];
  const active = pipelines.find((p) => p.id === sp1(sp.pipeline)) ?? fallback;

  const [deals, reasons] = await Promise.all([
    active
      ? load(() => listDeals({ pipelineId: active.id, q: sp1(sp.q), ownerUserId: sp1(sp.owner), pageSize: 500 }))
      : Promise.resolve(null),
    load(listLostReasons),
  ]);
  const stages = (active?.stages ?? []).filter((st) => st.isActive).sort((a, b) => a.sortOrder - b.sortOrder);

  return (
    <>
      <PageHeader eyebrow={ss.eyebrow} title={s.title} description={s.subtitle} />
      {pipelines.length > 1 && fallback ? (
        <UrlTabs
          param="pipeline"
          label="Pipeline"
          tabs={[fallback, ...pipelines.filter((p) => p.id !== fallback.id)].map((p) => ({ value: p.id, label: p.name }))}
        />
      ) : null}
      <div className="flex flex-wrap items-center gap-2">
        <SearchFilter placeholder={s.searchPlaceholder} />
        <SelectFilter param="owner" label={ss.owner} allLabel={s.allOwner} options={refs.owners} />
        <ResetFilters params={["q", "owner"]} />
      </div>
      {!active || !deals ? (
        <ErpError message={ss.none} />
      ) : deals.data ? (
        <PipelineBoard stages={stages} deals={deals.data.deals} lostReasons={reasons.data ?? []} refs={refs} caps={caps} />
      ) : (
        <ErpError message={deals.error?.message} />
      )}
    </>
  );
}
