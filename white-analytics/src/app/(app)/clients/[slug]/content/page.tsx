import type { Metadata } from "next";
import { Suspense } from "react";
import Link from "next/link";
import { ClipboardList, FileText, KanbanSquare } from "lucide-react";
import { Button } from "@/components/ui/button";
import { HeroStage } from "@/components/dashboard/hero-stage";
import { PageHeader } from "@/components/dashboard/page-header";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { DashFilterBar } from "@/features/content/components/dashboard/filters";
import { ActionLists, ActionListsSkeleton } from "@/features/content/components/dashboard/action-lists";
import { SummaryTab, ProductionTab, CalendarTab } from "@/features/content/components/dashboard/views-ops";
import { PerformanceTab, AssetsTab } from "@/features/content/components/dashboard/views-results";
import type { DashCtx } from "@/features/content/components/dashboard/shared";
import { ContentOffline } from "@/features/content/components/offline";
import { dashboardQuery, kpiDelta, parseDashFilters, publishRate, type DashTab } from "@/features/content/dashboard";
import { monthLabel } from "@/features/publishing/time";
import { num } from "@/features/content/lib";
import { contentContext, getDashboard, listPlans, loadContent } from "@/features/content/queries";
import { c, cd } from "@/features/content/strings";
import { getRange } from "@/lib/range-params";
import { formatCompact, formatDateRange, formatNumber, formatPercent } from "@/lib/format";
import { parseISODate, toISODate } from "@/lib/dates";
import { t } from "@/i18n/id";

export async function generateMetadata(props: PageProps<"/clients/[slug]/content">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await contentContext(slug);
  return { title: `${client.name} · ${t.nav.content}` };
}

const TAB_VIEWS = { ringkasan: SummaryTab, produksi: ProductionTab, kalender: CalendarTab, performa: PerformanceTab, aset: AssetsTab } satisfies Record<DashTab, unknown>;

export default async function ContentOverviewPage(props: PageProps<"/clients/[slug]/content">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client, canWrite, people, names } = await contentContext(slug);
  const base = `/clients/${slug}/content`;
  const { range, compare } = getRange(sp);
  const filters = parseDashFilters(sp);
  const from = toISODate(range.from);
  const to = toISODate(range.to);

  const res = await loadContent(() => Promise.all([getDashboard(client.id, dashboardQuery(from, to, filters)), listPlans(client.id)]));
  if (!res.ok) {
    return (
      <>
        <PageHeader eyebrow={t.nav.content} title={c.overviewTitle} description={c.overviewDesc} />
        <ContentOffline message={res.message} />
      </>
    );
  }
  const [data, plans] = res.data;
  const k = data.current.kpis;
  const p = data.previous.kpis;
  const d = (cur: number | string, prev: number | string) => (compare ? kpiDelta(cur, prev) : null);

  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(sp)) if (typeof value === "string" && value && key !== "tab") query.set(key, value);
  const ctx: DashCtx = {
    base,
    clientId: client.id,
    query: query.toString(),
    names,
    compare,
    plans: plans.map((pl) => ({ id: pl.id, month: pl.month })),
  };
  const prevFrom = parseISODate(data.prevFrom);
  const prevTo = parseISODate(data.prevTo);
  const compareLabel = compare && prevFrom && prevTo ? cd.comparedWith(formatDateRange(prevFrom, prevTo)) : null;
  const View = TAB_VIEWS[filters.tab];

  return (
    <>
      <HeroStage
        eyebrow={t.nav.content}
        title={
          <>
            {client.name} <span className="text-muted-foreground">{cd.heroTail}</span>
          </>
        }
        description={`${cd.heroDesc} · ${formatDateRange(range.from, range.to)}`}
        actions={
          <>
            <Button asChild variant="outline" size="sm">
              <Link href={`${base}/plans`}>
                <ClipboardList className="size-4" /> {t.nav.contentPlans}
              </Link>
            </Button>
            <Button asChild variant="outline" size="sm">
              <Link href={`${base}/reports`}>
                <FileText className="size-4" /> {t.nav.contentReports}
              </Link>
            </Button>
            <Button asChild size="sm">
              <Link href={`${base}/board`}>
                <KanbanSquare className="size-4" /> {t.nav.contentBoard}
              </Link>
            </Button>
          </>
        }
        stats={[
          {
            label: cd.kPublished,
            value: formatNumber(k.published),
            delta: d(k.published, p.published),
            caption: cd.ofPlannedCount(k.planned, formatPercent(publishRate(k), 0)),
            href: `${base}/board`,
            hint: cd.hPublished,
          },
          { label: cd.kOnTime, value: formatPercent(k.onTimeRate, 1), delta: d(k.onTimeRate, p.onTimeRate), hint: cd.hOnTime(k.deadlinesMet, k.deadlinesTotal) },
          { label: cd.engagement, value: formatCompact(num(k.engagements)), delta: d(k.engagements, p.engagements), hint: cd.hEngagement },
          { label: cd.er, value: formatPercent(k.engagementRate, 1), delta: d(k.engagementRate, p.engagementRate), hint: cd.hEr },
        ]}
      />

      <DashFilterBar
        periodLabel={formatDateRange(range.from, range.to)}
        compareLabel={compareLabel}
        pillars={data.pillars ?? []}
        people={people.map((x) => ({ value: x.id, label: x.name }))}
        plans={plans.map((pl) => ({ value: pl.id, label: `${monthLabel(pl.month)} · ${c.planStatus[pl.status]}` }))}
      />

      <UrlTabs
        label={cd.tabsLabel}
        tabs={[
          { value: "ringkasan", label: cd.tabSummary },
          { value: "produksi", label: cd.tabProduction },
          { value: "kalender", label: cd.tabCalendar },
          { value: "performa", label: cd.tabPerformance },
          { value: "aset", label: cd.tabAssets },
        ]}
      />

      <View data={data} ctx={ctx} />

      {filters.tab === "ringkasan" ? (
        <Suspense fallback={<ActionListsSkeleton />}>
          <ActionLists slug={slug} clientId={client.id} timezone={client.timezone} canWrite={canWrite} plans={plans} />
        </Suspense>
      ) : null}
    </>
  );
}
