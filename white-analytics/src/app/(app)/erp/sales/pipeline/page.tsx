import type { Metadata } from "next";
import { Toolbar } from "@/components/dashboard/layout";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { DateRangeFilter } from "@/features/erp/shared/period-picker";
import { ResetFilters, SearchFilter, SelectFilter } from "@/features/erp/shared/url-filters";
import { UrlFilterChips, type UrlChip } from "@/features/erp/sales/components/filter-chips";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { PipelineBoard } from "@/features/erp/sales/components/pipeline-board";
import { salesCaps } from "@/features/erp/sales/lib";
import { listDeals, listLostReasons } from "@/features/erp/sales/queries";
import { dealFormRefs } from "@/features/erp/sales/refs";
import { LEAD_SOURCE, SERVICE_LINE, ss } from "@/features/erp/sales/strings";

const s = ss.pipeline;
const FILTERS = ["q", "owner", "line", "source", "status", "from", "to", "date", "industry", "stage", "reason"];
const opts = (m: Record<string, string>) => Object.entries(m).map(([value, label]) => ({ value, label }));
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
      ? load(() =>
          listDeals({
            pipelineId: active.id,
            q: sp1(sp.q),
            ownerUserId: sp1(sp.owner),
            serviceLine: sp1(sp.line),
            industry: sp1(sp.industry),
            stageId: sp1(sp.stage),
            source: sp1(sp.source),
            status: sp1(sp.status),
            lostReasonId: sp1(sp.reason),
            dateFrom: sp1(sp.from),
            dateTo: sp1(sp.to),
            dateField: sp1(sp.date) === "closed" ? "closed" : "created",
            pageSize: 500,
          }),
        )
      : Promise.resolve(null),
    load(listLostReasons),
  ]);
  const stages = (active?.stages ?? []).filter((st) => st.isActive).sort((a, b) => a.sortOrder - b.sortOrder);
  const industry = sp1(sp.industry);
  const stage = stages.find((st) => st.id === sp1(sp.stage));
  const reason = sp1(sp.reason);
  const chips: UrlChip[] = [
    industry ? { label: s.filters.industry, value: industry, params: ["industry"] } : null,
    stage ? { label: s.stage, value: stage.name, params: ["stage"] } : null,
    reason
      ? { label: s.lostReason, value: reason === "none" ? s.filters.noReason : (reasons.data?.find((r) => r.id === reason)?.name ?? reason), params: ["reason"] }
      : null,
  ].filter((c): c is UrlChip => c != null);

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
      <Toolbar label={s.title}>
        <SearchFilter placeholder={s.searchPlaceholder} />
        <SelectFilter param="owner" label={ss.owner} allLabel={s.allOwner} options={refs.owners} />
        <SelectFilter param="status" label={s.filters.status} allLabel={s.filters.allStatus} options={opts(s.status)} />
        <SelectFilter param="line" label={s.serviceLines} allLabel={s.filters.allLines} options={opts(SERVICE_LINE)} />
        <SelectFilter param="source" label={s.filters.source} allLabel={s.filters.allSources} options={opts(LEAD_SOURCE)} />
        <DateRangeFilter />
        {sp1(sp.from) ? (
          <SelectFilter param="date" label={s.filters.dateBasis} allLabel={s.filters.created} options={[{ value: "closed", label: s.filters.closed }]} />
        ) : null}
        <ResetFilters params={FILTERS} />
      </Toolbar>
      <UrlFilterChips chips={chips} />
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
