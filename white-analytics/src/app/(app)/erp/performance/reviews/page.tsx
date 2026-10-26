import type { Metadata } from "next";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/dashboard/page-header";
import { canAccess } from "@/components/shell/nav-config";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1, spInt } from "@/features/erp/shared/load";
import { ErpEmpty } from "@/features/erp/shared/states";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { UrlSelect } from "@/features/erp/performance/components/common";
import { CycleBar, RealtimeGrid, ReviewsGrid } from "@/features/erp/performance/components/reviews";
import { SectionError } from "@/features/erp/performance/components/section-error";
import { jakartaToday, quarterOf, splitMonth } from "@/features/erp/performance/lib";
import { getRealtimeKpi, listReviewCycles, listReviews } from "@/features/erp/performance/queries";
import { sp } from "@/features/erp/performance/strings";

export const metadata: Metadata = { title: sp.reviews.title };

const r = sp.reviews;

export default async function ReviewsPage(props: PageProps<"/erp/performance/reviews">) {
  const q = await props.searchParams;
  const { access, offline } = await requireErp("hr.read");
  const tab = sp1(q.tab) ?? "realtime";
  const today = splitMonth(jakartaToday().slice(0, 7));
  const year = spInt(q.year, today.year);
  const quarter = Math.min(4, spInt(q.quarter, quarterOf(today.month)));

  return (
    <>
      <PageHeader
        eyebrow={sp.common.eyebrow}
        title={r.title}
        description={r.subtitle}
        actions={
          <Button asChild size="sm" variant="secondary">
            <Link href="/erp/performance/feedback">{r.feedbackLink}</Link>
          </Button>
        }
      />
      {offline ? (
        <OfflineNotice message={offline.message} />
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-2">
            <UrlTabs
              label={r.title}
              reset={["cycle"]}
              tabs={[
                { value: "realtime", label: r.tabRealtime },
                { value: "cycles", label: r.tabCycles },
              ]}
            />
            {tab === "realtime" ? (
              <>
                <UrlSelect
                  param="year"
                  label={sp.common.year}
                  fallback={String(today.year)}
                  options={[today.year - 1, today.year, today.year + 1].map((y) => ({ value: String(y), label: String(y) }))}
                />
                <UrlSelect
                  param="quarter"
                  label={r.quarter}
                  fallback={String(quarterOf(today.month))}
                  options={[1, 2, 3, 4].map((n) => ({ value: String(n), label: `Q${n}` }))}
                />
              </>
            ) : null}
          </div>
          {tab === "cycles" ? (
            <CyclesSection cycleParam={sp1(q.cycle)} canWrite={canAccess(access, "hr.write")} />
          ) : (
            <RealtimeSection year={year} quarter={quarter} />
          )}
        </>
      )}
    </>
  );
}

async function RealtimeSection({ year, quarter }: { year: number; quarter: number }) {
  const res = await load(() => getRealtimeKpi(year, quarter));
  if (res.error) return <SectionError error={res.error} />;
  return <RealtimeGrid data={res.data} />;
}

async function CyclesSection({ cycleParam, canWrite }: { cycleParam?: string; canWrite: boolean }) {
  const res = await load(listReviewCycles);
  if (res.error) return <SectionError error={res.error} />;
  const { cycles } = res.data;
  // The service leaves can_manage unset; opening and closing need hr.write.
  const canManage = res.data.canManage || canWrite;
  const active = cycles.find((c) => c.id === cycleParam)?.id ?? cycles[0]?.id ?? "";
  const reviews = active ? await load(() => listReviews(active)) : null;
  return (
    <>
      <CycleBar cycles={cycles} active={active} canManage={canManage} />
      {cycles.length === 0 ? (
        <ErpEmpty title={r.noCycles} />
      ) : reviews?.error ? (
        <SectionError error={reviews.error} />
      ) : (
        <ReviewsGrid reviews={reviews?.data?.reviews ?? []} />
      )}
    </>
  );
}
