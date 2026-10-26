import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { canAccess } from "@/components/shell/nav-config";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { MonthFilter } from "@/features/erp/shared/period-picker";
import { ResetFilters, SelectFilter } from "@/features/erp/shared/url-filters";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { UrlSelect } from "@/features/erp/performance/components/common";
import { RunSnapshotButton, ScorecardsTable } from "@/features/erp/performance/components/kpi-scorecards";
import {
  DepartmentWeights,
  IndicatorsTable,
  TargetsPanel,
} from "@/features/erp/performance/components/kpi-setup";
import { SectionError } from "@/features/erp/performance/components/section-error";
import { isMonth, jakartaToday, splitMonth } from "@/features/erp/performance/lib";
import {
  getKpiConfig,
  getRecommendations,
  getTaskMeta,
  listIndicators,
  listScorecards,
  listTargets,
} from "@/features/erp/performance/queries";
import { sp } from "@/features/erp/performance/strings";

export const metadata: Metadata = { title: sp.kpi.title };

const k = sp.kpi;

export default async function KpiPage(props: PageProps<"/erp/performance/kpi">) {
  const q = await props.searchParams;
  const { access, offline } = await requireErp("hr.read");
  const canWrite = canAccess(access, "hr.write");
  const canApprove = canAccess(access, "hr.approve");
  const tab = sp1(q.tab) ?? "scorecards";
  const thisMonth = jakartaToday().slice(0, 7);
  const month = isMonth(sp1(q.month)) ? sp1(q.month)! : thisMonth;
  const { year, month: m } = splitMonth(month);
  const departments =
    tab === "scorecards" && !offline ? ((await load(getTaskMeta)).data?.departments ?? []) : [];

  return (
    <>
      <PageHeader
        eyebrow={sp.common.eyebrow}
        title={k.title}
        description={k.subtitle}
        actions={
          tab === "scorecards" && canWrite && !offline ? <RunSnapshotButton year={year} month={m} /> : null
        }
      />
      {offline ? (
        <OfflineNotice message={offline.message} />
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-2">
            <UrlTabs
              label={k.title}
              reset={["scope", "month", "dept"]}
              tabs={[
                { value: "scorecards", label: k.tabScorecards },
                { value: "indicators", label: k.tabIndicators },
                { value: "config", label: k.tabConfig },
                { value: "targets", label: k.tabTargets },
              ]}
            />
            {tab === "scorecards" ? (
              <>
                <UrlSelect
                  param="scope"
                  label={k.scope}
                  fallback="team"
                  options={[
                    { value: "team", label: k.scopeTeam },
                    { value: "all", label: k.scopeAll },
                    { value: "me", label: k.scopeMe },
                  ]}
                />
                <MonthFilter fallback={thisMonth} />
                {departments.length > 1 ? (
                  <SelectFilter
                    param="dept"
                    label={sp.common.department}
                    allLabel={sp.common.allDepartments}
                    options={departments.map((d) => ({ value: d.id, label: d.name }))}
                  />
                ) : null}
                <ResetFilters params={["scope", "month", "dept"]} />
              </>
            ) : null}
          </div>
          {tab === "indicators" ? (
            <IndicatorsSection canWrite={canWrite} />
          ) : tab === "config" ? (
            <ConfigSection canWrite={canWrite} />
          ) : tab === "targets" ? (
            <TargetsSection canWrite={canWrite} />
          ) : (
            <ScorecardsSection
              scope={sp1(q.scope) ?? "team"}
              year={year}
              month={m}
              dept={sp1(q.dept)}
              canApprove={canApprove}
            />
          )}
        </>
      )}
    </>
  );
}

async function ScorecardsSection({
  scope,
  year,
  month,
  dept,
  canApprove,
}: {
  scope: string;
  year: number;
  month: number;
  dept?: string;
  canApprove: boolean;
}) {
  const res = await load(() =>
    listScorecards({
      periodYear: year,
      periodMonth: month,
      team: scope === "team" || undefined,
      employeeId: scope === "me" ? "me" : undefined,
    }),
  );
  if (res.error) return <SectionError error={res.error} />;
  // The dashboard drills in by department; the API filters by people only.
  if (dept) res.data.scorecards = res.data.scorecards.filter((c) => c.departmentId === dept);
  const recs = await load(() =>
    getRecommendations([...new Set(res.data.scorecards.map((s) => s.employeeId))]),
  );
  return <ScorecardsTable data={res.data} recommendations={recs.data ?? []} canApprove={canApprove} />;
}

async function IndicatorsSection({ canWrite }: { canWrite: boolean }) {
  const res = await load(() => listIndicators(true));
  if (res.error) return <SectionError error={res.error} />;
  return <IndicatorsTable indicators={res.data} canWrite={canWrite} />;
}

async function ConfigSection({ canWrite }: { canWrite: boolean }) {
  const res = await load(getKpiConfig);
  if (res.error) return <SectionError error={res.error} />;
  return <DepartmentWeights config={res.data} canWrite={canWrite} />;
}

async function TargetsSection({ canWrite }: { canWrite: boolean }) {
  const [targets, config, meta] = await Promise.all([
    load(listTargets),
    load(getKpiConfig),
    load(getTaskMeta),
  ]);
  const err = targets.error ?? config.error;
  if (err) return <SectionError error={err} />;
  return (
    <TargetsPanel
      targets={targets.data!}
      indicators={config.data!.indicators}
      departments={config.data!.departments}
      people={meta.data?.members ?? []}
      canWrite={canWrite}
    />
  );
}
