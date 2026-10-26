"use client";

import { DashboardFilterBar } from "@/features/erp/shared/dashboard-filter-bar";
import type { ResolvedPeriod } from "@/features/erp/shared/dashboard-period";
import { SelectFilter, type FilterOption } from "@/features/erp/shared/url-filters";
import { LEAD_SOURCE, SERVICE_LINE } from "../../strings";
import { FILTER_PARAMS } from "../params";
import { dd } from "../strings";

const f = dd.filters;
const entries = (m: Record<string, string>) => Object.entries(m).map(([value, label]) => ({ value, label }));

/** Period, comparison and the sales dimensions, all in the URL. */
export function SalesDashboardFilters({ period, pipelines, owners }: { period: ResolvedPeriod; pipelines: FilterOption[]; owners: FilterOption[] }) {
  return (
    <DashboardFilterBar period={period} params={[...FILTER_PARAMS]}>
      {pipelines.length ? <SelectFilter param="pipeline" label={f.pipeline} allLabel={f.allPipelines} options={pipelines} /> : null}
      <SelectFilter param="owner" label={f.owner} allLabel={f.allOwners} options={owners} />
      <SelectFilter param="line" label={f.line} allLabel={f.allLines} options={entries(SERVICE_LINE)} />
      <SelectFilter param="source" label={f.source} allLabel={f.allSources} options={entries(LEAD_SOURCE)} />
    </DashboardFilterBar>
  );
}
