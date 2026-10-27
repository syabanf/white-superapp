import { Suspense, type ReactNode } from "react";
import type { Metadata } from "next";
import Link from "next/link";
import { ArrowUpRight, ShieldAlert } from "lucide-react";
import { Alert, AlertTitle } from "@/components/ui/alert";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { CardGrid } from "@/components/dashboard/layout";
import { EmptyState } from "@/components/dashboard/empty-state";
import { canAccess, visibleErpNav } from "@/components/shell/nav-config";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { resolvePeriod } from "@/features/erp/shared/dashboard-period";
import { DashboardFilterBar } from "@/features/erp/shared/dashboard-filter-bar";
import { sp1 } from "@/features/erp/shared/load";
import { AttentionSkeleton, ModuleSkeleton } from "@/features/erp/home/skeletons";
import {
  AttentionStrip,
  FinanceSection,
  HrSection,
  PerformanceSection,
  RecruitmentSection,
  SalesSection,
  type AttentionAccess,
} from "@/features/erp/home/sections";
import { hs } from "@/features/erp/home/strings";
import { t } from "@/i18n/id";

export const metadata: Metadata = { title: t.erp.title };

const DESCRIPTIONS: Record<string, string> = {
  "/erp/finance": t.erp.modules.finance,
  "/erp/hr": t.erp.modules.hr,
  "/erp/hr/recruitment": hs.recruitmentModule,
  "/erp/performance": t.erp.modules.performance,
  "/erp/sales": t.erp.modules.sales,
  "/erp/me": t.erp.modules.me,
};

export default async function ErpHomePage(props: PageProps<"/erp">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp();
  const period = resolvePeriod(
    { preset: sp1(sp.preset), from: sp1(sp.from), to: sp1(sp.to), compare: sp1(sp.compare) },
    "thisMonth",
  );
  // Without hr.read, performance shows the user's own scope as a personal summary.
  const perfMe = !canAccess(access, "hr.read") && canAccess(access, "self.service");
  const can: AttentionAccess = {
    finance: canAccess(access, "finance.read"),
    hr: canAccess(access, "hr.read"),
    performance: canAccess(access, ["hr.read", "self.service"]),
    perfMe,
    sales: canAccess(access, "sales.read"),
    recruitment: canAccess(access, "recruitment.read"),
  };
  const anyDashboard = can.finance || can.hr || can.performance || can.sales || can.recruitment;
  // One compact card per module the user can open.
  const modules = visibleErpNav(access)
    .flatMap((g) => g.items)
    .filter((it) => DESCRIPTIONS[it.href]);
  const key = `${period.from}|${period.to}|${period.compare}`;
  // Keyed by period so a filter change re-suspends each section on its own.
  const stream = (name: string, node: ReactNode, fallback = <ModuleSkeleton />) => (
    <Suspense key={`${name}|${key}`} fallback={fallback}>
      {node}
    </Suspense>
  );

  return (
    <>
      <PageHeader banner eyebrow={t.erp.eyebrow} title={t.erp.title} description={t.erp.subtitle} />
      {offline ? <OfflineNotice message={offline.message} /> : null}
      {sp.unauthorized === "1" ? (
        <Alert className="rounded-2xl">
          <ShieldAlert className="size-4" />
          <AlertTitle>{t.erp.unauthorized}</AlertTitle>
        </Alert>
      ) : null}
      {!offline && anyDashboard ? (
        <>
          <DashboardFilterBar period={period} />
          {stream("attention", <AttentionStrip period={period} can={can} />, <AttentionSkeleton />)}
          {perfMe ? stream("me", <PerformanceSection period={period} me />) : null}
          {can.finance ? stream("finance", <FinanceSection period={period} />) : null}
          {can.sales ? stream("sales", <SalesSection period={period} />) : null}
          {can.hr ? stream("hr", <HrSection period={period} />) : null}
          {can.performance && !perfMe ? stream("me", <PerformanceSection period={period} me={false} />) : null}
          {can.recruitment ? stream("recruitment", <RecruitmentSection period={period} />) : null}
        </>
      ) : null}
      {modules.length === 0 ? (
        <EmptyState icon={<ShieldAlert className="size-5" />} title={t.erp.noModules} />
      ) : (
        <Section title={hs.modulesTitle}>
          <CardGrid cols={3}>
            {modules.map((m) => {
              const Icon = m.icon;
              return (
                <Link
                  key={m.href}
                  href={m.href}
                  className="group lift flex min-w-0 items-center gap-3 rounded-2xl bg-card p-(--card-pad) shadow-(--card-shadow) outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  <span className="flex size-10 shrink-0 items-center justify-center rounded-2xl bg-[color-mix(in_oklab,var(--brand)_14%,transparent)] text-brand">
                    <Icon className="size-4" />
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="block text-sm font-semibold">{m.label}</span>
                    <span className="block truncate text-xs text-muted-foreground">{DESCRIPTIONS[m.href]}</span>
                  </span>
                  <ArrowUpRight className="size-4 shrink-0 text-muted-foreground transition-transform group-hover:-translate-y-0.5 group-hover:translate-x-0.5" />
                </Link>
              );
            })}
          </CardGrid>
        </Section>
      )}
    </>
  );
}
