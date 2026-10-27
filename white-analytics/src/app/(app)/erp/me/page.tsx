import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { CalendarCheck, Clock4, FileText, HandCoins, Plane, UserX } from "lucide-react";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { MonthFilter } from "@/features/erp/shared/period-picker";
import { ErpEmpty, ErpError } from "@/features/erp/shared/states";
import { listLeaveTypes } from "@/features/erp/hr/queries";
import { monthParam } from "@/features/erp/hr/perms";
import { ClockPanel } from "@/features/erp/me/components/clock-panel";
import { MyAnnouncements, MyApprovalsList, WeekStrip } from "@/features/erp/me/components/home";
import {
  AttendanceCalendar,
  LeaveCard,
  MyKpis,
  OnTimeTrend,
  PayslipCard,
  PendingRequests,
  TeamToday,
  WeeklyHours,
} from "@/features/erp/me/components/dashboard";
import { getMyApprovals, getMyDashboard, getMyHome, listMyAnnouncements } from "@/features/erp/me/queries";
import { tm } from "@/features/erp/me/strings";
import { formatDateLong } from "@/lib/format";

export const metadata: Metadata = { title: tm.home.title };

const SHORTCUTS = [
  { href: "/erp/me/attendance", label: tm.attendance.title, icon: CalendarCheck },
  { href: "/erp/me/leave", label: tm.leave.title, icon: Plane },
  { href: "/erp/me/overtime", label: tm.overtime.title, icon: Clock4 },
  { href: "/erp/me/loans", label: tm.loans.title, icon: HandCoins },
  { href: "/erp/me/payslips", label: tm.payslips.title, icon: FileText },
] as const;

export default async function MyHomePage(props: PageProps<"/erp/me">) {
  const sp = await props.searchParams;
  const { offline } = await requireErp("self.service");
  if (offline) return <OfflineNotice message={offline.message} />;
  const { ym } = monthParam(sp1(sp.month));
  const [home, news, approvals, types] = await Promise.all([
    load(getMyHome),
    load(listMyAnnouncements),
    load(getMyApprovals),
    load(listLeaveTypes),
  ]);
  if (!home.data) {
    return home.error.code === "NOT_FOUND" ? (
      <ErpEmpty icon={<UserX className="size-5" />} title={tm.notLinkedTitle} description={tm.notLinkedBody} />
    ) : (
      <ErpError message={home.error.message} />
    );
  }
  const h = home.data;
  const shift = h.todayShift;
  const leaveTypes = Object.fromEntries((types.data ?? []).map((t) => [t.code, t.name]));
  const pending = (approvals.data?.leaves.length ?? 0) + (approvals.data?.overtime.length ?? 0);

  return (
    <>
      <PageHeader
        eyebrow={formatDateLong(h.today)}
        title={tm.home.greeting(h.employee.fullName)}
        description={h.employee.departmentName}
        actions={<MonthFilter fallback={ym} />}
      />
      <div className="grid grid-cols-1 gap-(--gap) lg:grid-cols-[1.2fr_minmax(0,1fr)]">
        <Card className="min-w-0 gap-4 px-(--card-pad)">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="label-mono text-muted-foreground">{tm.home.todayShift}</p>
              {shift ? (
                <>
                  <p className="mt-1 text-2xl font-bold tabular">
                    {shift.startTime}–{shift.endTime}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    {shift.name} · {tm.home.tolerance(shift.lateToleranceMinutes)}
                  </p>
                </>
              ) : (
                <p className="mt-1 text-base font-medium">{h.holidayName ? tm.home.holiday(h.holidayName) : tm.home.noShift}</p>
              )}
            </div>
            <Link href="/erp/me/leave" className="rounded-2xl bg-muted/60 px-3 py-2 text-right hover:bg-muted">
              <p className="text-[11px] text-muted-foreground">{tm.home.leaveLeft}</p>
              <p className="text-xl font-bold tabular">{h.leaveBalance.annualLeaveRemaining}</p>
            </Link>
          </div>
          <ClockPanel today={h.todayAttendance} hasShift={Boolean(shift)} />
        </Card>
        <Section title={tm.home.week} className="min-w-0">
          <WeekStrip week={h.week} today={h.today} />
          <nav aria-label={tm.home.shortcuts} className="grid grid-cols-3 gap-(--gap-inner) sm:grid-cols-5 lg:grid-cols-3">
            {SHORTCUTS.map((s) => (
              <Link
                key={s.href}
                href={s.href}
                className="flex flex-col items-center gap-1.5 rounded-2xl bg-card p-3 text-center text-xs shadow-(--card-shadow) hover:bg-accent/50"
              >
                <s.icon className="size-5 text-brand" />
                {s.label}
              </Link>
            ))}
          </nav>
        </Section>
      </div>

      <Suspense key={ym} fallback={<MyDashboardSkeleton />}>
        <MyDashboardSections month={ym} leaveTypes={leaveTypes} />
      </Suspense>

      {pending > 0 && approvals.data ? (
        <Section title={`${tm.home.approvals} (${pending})`}>
          <MyApprovalsList data={approvals.data} />
        </Section>
      ) : null}
      <Section
        title={tm.home.announcements}
        actions={h.unreadAnnouncements ? <span className="text-xs text-muted-foreground">{tm.home.unread(h.unreadAnnouncements)}</span> : null}
      >
        {news.data ? <MyAnnouncements items={news.data.announcements} /> : <ErpError compact message={news.error.message} />}
      </Section>
    </>
  );
}

async function MyDashboardSections({ month, leaveTypes }: { month: string; leaveTypes: Record<string, string> }) {
  const res = await load(() => getMyDashboard(month));
  if (!res.data) return <ErpError message={res.error.message} />;
  const data = res.data;
  return (
    <div className="page-stack">
      <MyKpis data={data} />
      <div className="grid grid-cols-1 gap-(--gap) lg:grid-cols-2">
        <AttendanceCalendar data={data} leaveTypes={leaveTypes} />
        <div className="grid gap-(--gap)">
          <OnTimeTrend data={data} />
          <WeeklyHours data={data} />
        </div>
      </div>
      {data.isManager ? (
        <div className="grid grid-cols-1 gap-(--gap) lg:grid-cols-2">
          <TeamToday data={data} leaveTypes={leaveTypes} />
          <PendingRequests data={data} leaveTypes={leaveTypes} />
        </div>
      ) : null}
      <div className="grid grid-cols-1 gap-(--gap) lg:grid-cols-2">
        <PayslipCard data={data} />
        <LeaveCard data={data} leaveTypes={leaveTypes} />
      </div>
      {data.isManager ? null : <PendingRequests data={data} leaveTypes={leaveTypes} />}
    </div>
  );
}

function MyDashboardSkeleton() {
  return (
    <div className="page-stack" aria-busy="true">
      <Skeleton className="h-20 rounded-[1.5rem]" />
      <div className="grid grid-cols-1 gap-(--gap) lg:grid-cols-2">
        <Skeleton className="h-96 rounded-2xl" />
        <Skeleton className="h-96 rounded-2xl" />
      </div>
      <div className="grid grid-cols-1 gap-(--gap) lg:grid-cols-2">
        <Skeleton className="h-80 rounded-2xl" />
        <Skeleton className="h-80 rounded-2xl" />
      </div>
    </div>
  );
}
