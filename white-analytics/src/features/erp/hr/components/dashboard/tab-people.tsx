"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { CalendarClock, FileSignature } from "lucide-react";
import { ChartCard } from "@/components/dashboard/chart-card";
import { StatStrip } from "@/components/dashboard/stat-strip";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { Button } from "@/components/ui/button";
import { th } from "@/features/erp/hr/strings";
import { hrLinks } from "@/features/erp/hr/dashboard/links";
import { dayLabelLong, delta, monthLabel, num0, pct1 } from "@/features/erp/hr/dashboard/format";
import { thd } from "@/features/erp/hr/dashboard/strings";
import type { HrDashSlice } from "@/features/erp/hr/dashboard/types";
import { ChartEmpty, ComboChart, Donut } from "./charts";
import { useHrDrill, useHrScope } from "./drill";
import { deptPalette, Grid, type TabProps } from "./shared";

const c = thd.charts;
const k = thd.kpi;

function bands(rows: HrDashSlice[], labels: Record<string, string>) {
  return rows.map((r) => ({ key: r.key, label: labels[r.key] ?? r.key, tip: labels[r.key] ?? r.key, count: r.count }));
}

export function PeopleTab({ data }: TabProps) {
  const router = useRouter();
  const drill = useHrDrill();
  const { dept } = useHrScope();
  const cur = data.current;
  const prev = data.previous;
  const depts = deptPalette(data);
  const levels = th.masters.levels;

  return (
    <div className="space-y-5">
      <StatStrip
        items={[
          {
            label: k.headcount,
            value: num0(cur.headcount),
            delta: delta(cur.headcount, prev.headcount),
            hint: k.headcountHint,
            href: hrLinks.employees({ dept }),
          },
          { label: k.joiners, value: num0(cur.joiners), delta: delta(cur.joiners, prev.joiners), href: hrLinks.employees({ dept }) },
          {
            label: k.leavers,
            value: num0(cur.leavers),
            delta: delta(cur.leavers, prev.leavers),
            lowerIsBetter: true,
            href: hrLinks.employees({ dept, inactive: true }),
          },
          {
            label: k.turnover,
            value: pct1(cur.turnoverRate),
            delta: delta(cur.turnoverRate, prev.turnoverRate),
            lowerIsBetter: true,
            hint: k.turnoverHint,
            href: hrLinks.employees({ dept, inactive: true }),
          },
        ]}
      />

      <Grid>
        <ChartCard title={c.byDept} description={c.compositionQ}>
          <HBarChart
            items={[...data.departments]
              .filter((d) => d.headcount > 0)
              .sort((a, b) => b.headcount - a.headcount)
              .map((d) => ({ label: depts.name(d.departmentId), value: d.headcount, key: d.departmentId }))}
            onSelect={(item) => drill("emp", { dept: item.key })}
            format={num0}
            showShare
            emptyLabel={thd.empty}
          />
        </ChartCard>
        <ChartCard title={c.byStatus} description={c.compositionQ}>
          <Donut
            slices={data.byStatus.map((s) => ({ key: s.key, label: s.label || s.key, value: s.count }))}
            format={num0}
            centerLabel={thd.units.people}
            onSelect={(sl) => drill("emp", { status: sl.key, label: sl.label })}
          />
        </ChartCard>
        <ChartCard title={c.byContract} description={c.compositionQ}>
          <Donut
            slices={data.byContractType.map((s) => ({ key: s.key, label: th.contracts.typesShort[s.key] ?? (s.key || "–"), value: s.count }))}
            format={num0}
            centerLabel={thd.units.people}
            onSelect={(s) => router.push(hrLinks.contracts({ type: s.key, status: "active" }))}
          />
        </ChartCard>
      </Grid>

      <Grid>
        <ChartCard title={c.byLevel} description={c.compositionQ}>
          <HBarChart
            items={data.byLevel.map((s) => ({ label: levels[s.key] ?? (s.key || "–"), value: s.count, key: s.key }))}
            onSelect={(item) => item.key && drill("emp", { level: item.key })}
            format={num0}
            showShare
            emptyLabel={thd.empty}
          />
        </ChartCard>
        <ChartCard title={c.byPosition} description={c.compositionQ}>
          <HBarChart
            items={data.byPosition.slice(0, 8).map((s) => ({ label: s.label || "–", value: s.count, key: s.key }))}
            onSelect={(item) => item.key && drill("emp", { position: item.key })}
            format={num0}
            emptyLabel={thd.empty}
          />
        </ChartCard>
        <ChartCard title={c.byGender} description={c.compositionQ}>
          <Donut
            slices={data.byGender.map((s) => ({ key: s.key, label: th.genders[s.key] ?? (s.key || "–"), value: s.count }))}
            format={num0}
            centerLabel={thd.units.people}
            onSelect={(sl) => sl.key && drill("emp", { gender: sl.key })}
          />
        </ChartCard>
      </Grid>

      <Grid>
        <ChartCard title={c.tenure} description={c.tenureQ}>
          <ComboChart
            data={bands(data.tenureBands, thd.tenureBands)}
            series={[{ key: "count", label: thd.series.employees, color: slotColor(0), format: num0 }]}
            yFormat={num0}
            legend={false}
            summary={c.tenure}
            onSelect={(r) => drill("emp", { tenure: String(r.key) })}
            height={220}
          />
        </ChartCard>
        <ChartCard title={c.age} description={c.ageQ}>
          <ComboChart
            data={bands(data.ageBands, thd.ageBands)}
            series={[{ key: "count", label: thd.series.employees, color: slotColor(6), format: num0 }]}
            yFormat={num0}
            legend={false}
            summary={c.age}
            height={220}
          />
        </ChartCard>
        <ChartCard title={c.turnoverTrend} description={`${c.turnoverTrendQ} ${thd.monthsWindow}.`}>
          <ComboChart
            data={data.months.map((b) => ({ label: monthLabel(b.date), tip: monthLabel(b.date), date: b.date, rate: b.turnoverRate }))}
            onSelect={(r) => drill("emp", { left: String(r.date).slice(0, 7) })}
            series={[{ key: "rate", label: thd.series.turnover, color: slotColor(7), format: pct1 }]}
            yFormat={(v) => `${v}%`}
            legend={false}
            summary={c.turnoverTrend}
            height={220}
          />
        </ChartCard>
      </Grid>

      <ChartCard
        title={c.expiring}
        description={c.expiringQ}
        actions={
          <Button asChild variant="ghost" size="sm">
            <Link href={hrLinks.expiring()}>{thd.openList}</Link>
          </Button>
        }
      >
        <ExpiryTimeline data={data} />
      </ChartCard>
    </div>
  );
}

/** Contracts and probations ending in 90 days, placed on a 0-90 day track. */
function ExpiryTimeline({ data }: { data: TabProps["data"] }) {
  if (data.expiring.length === 0) return <ChartEmpty height={120}>{thd.emptyExpiring}</ChartEmpty>;
  return (
    <ul className="divide-y">
      {data.expiring.map((e) => {
        const Icon = e.kind === "contract" ? FileSignature : CalendarClock;
        const urgent = e.daysLeft <= 30;
        return (
          <li key={`${e.kind}-${e.employeeId}-${e.contractId}`}>
            <Link href={hrLinks.employee(e.employeeId)} className="grid grid-cols-[1fr_auto] items-center gap-x-4 gap-y-1.5 py-2.5 hover:bg-muted/30 sm:grid-cols-[minmax(0,14rem)_1fr_auto]">
              <span className="flex min-w-0 items-center gap-2 text-sm">
                <Icon className="size-4 shrink-0 text-muted-foreground" />
                <span className="min-w-0">
                  <span className="block truncate font-medium">{e.fullName}</span>
                  <span className="block truncate text-xs text-muted-foreground">
                    {thd.kinds[e.kind]}
                    {e.contractType ? ` ${th.contracts.typesShort[e.contractType] ?? e.contractType}` : ""} · {e.departmentName}
                  </span>
                </span>
              </span>
              <span className="relative order-last col-span-2 h-2 rounded-full bg-muted/80 sm:order-none sm:col-span-1" aria-hidden>
                <span
                  className="absolute top-1/2 size-3 -translate-x-1/2 -translate-y-1/2 rounded-full ring-2 ring-card"
                  style={{ left: `${Math.min(100, (e.daysLeft / 90) * 100)}%`, background: urgent ? "var(--status-serious)" : slotColor(0) }}
                />
              </span>
              <span className="text-right text-xs tabular">
                <span className="block font-medium">{thd.daysLeft(e.daysLeft)}</span>
                <span className="block text-muted-foreground">{dayLabelLong(e.endDate)}</span>
              </span>
            </Link>
          </li>
        );
      })}
    </ul>
  );
}
