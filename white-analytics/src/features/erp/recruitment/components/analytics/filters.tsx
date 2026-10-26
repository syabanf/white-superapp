"use client";

import { DashboardFilterBar } from "@/features/erp/shared/dashboard-filter-bar";
import type { ResolvedPeriod } from "@/features/erp/shared/dashboard-period";
import { SelectFilter, type FilterOption } from "@/features/erp/shared/url-filters";
import { CANDIDATE_STATUS, SOURCE, STAGE_ORDER, ra } from "../../strings";

const FILTER_PARAMS = ["job", "department", "source", "stage"];

export function RecruitmentFilters({
  period,
  jobs,
  departments,
}: {
  period: ResolvedPeriod;
  jobs: FilterOption[];
  departments: string[];
}) {
  const f = ra.filters;
  return (
    <DashboardFilterBar period={period} params={FILTER_PARAMS}>
      <SelectFilter param="job" label={f.job} allLabel={f.allJobs} options={jobs} className="sm:max-w-56" />
      <SelectFilter
        param="department"
        label={f.department}
        allLabel={f.allDepartments}
        options={departments.map((d) => ({ value: d, label: d }))}
      />
      <SelectFilter
        param="source"
        label={f.source}
        allLabel={f.allSources}
        options={Object.entries(SOURCE).map(([value, label]) => ({ value, label }))}
      />
      <SelectFilter
        param="stage"
        label={f.stage}
        allLabel={f.allStages}
        options={STAGE_ORDER.map((s) => ({ value: s, label: CANDIDATE_STATUS[s] ?? s }))}
      />
    </DashboardFilterBar>
  );
}
