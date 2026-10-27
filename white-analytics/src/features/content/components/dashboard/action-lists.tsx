import Link from "next/link";
import { AlertTriangle, CalendarClock, ClipboardList } from "lucide-react";
import { Progress } from "@/components/ui/progress";
import { Section } from "@/components/dashboard/page-header";
import { EmptyState } from "@/components/dashboard/empty-state";
import { Skeleton } from "@/components/ui/skeleton";
import { PlatformIcon } from "@/features/social/components/platform-icon";
import { dayLabel, monthLabel, todayIn, weekDays } from "@/features/publishing/time";
import { CreatePlanButton } from "@/features/content/components/plans/plan-list";
import { ItemStatusBadge, PlanStatusBadge } from "@/features/content/components/status";
import { currentDue, isOverdue, type Stage } from "@/features/content/lib";
import { listItems, loadContent } from "@/features/content/queries";
import { c, cd } from "@/features/content/strings";
import type { Item, Plan } from "@/features/content/types";
import { formatRelative } from "@/lib/format";

const STAGES: Stage[] = ["COPY", "DESIGN", "REVIEW"];
const card = "rounded-[1.5rem] bg-card shadow-(--card-shadow)";
// list cards keep an 8px inset for the row hover; rows add the rest so text sits on --card-pad
const rowX = "px-[calc(var(--card-pad)-0.5rem)]";

function ItemRow({ item, href, meta, late }: { item: Item; href: string; meta: string; late?: boolean }) {
  return (
    <li>
      <Link href={href} className={`flex items-center gap-3 rounded-xl py-2.5 hover:bg-muted/50 ${rowX}`}>
        <PlatformIcon platform={item.platform} className="size-4" />
        <span className="min-w-0 flex-1">
          <span className="block truncate text-[13px] font-medium">{item.title}</span>
          <span className={late ? "block text-xs font-medium text-destructive" : "block text-xs text-muted-foreground"}>{meta}</span>
        </span>
        <ItemStatusBadge status={item.status} />
      </Link>
    </li>
  );
}

/**
 * What needs doing now, independent of the period filter: this month's
 * plan, deadlines this week, overdue items and recent changes. Streams in
 * after the charts.
 */
export async function ActionLists({ slug, clientId, timezone, canWrite, plans }: { slug: string; clientId: string; timezone: string; canWrite: boolean; plans: Plan[] }) {
  const base = `/clients/${slug}/content`;
  const today = todayIn(timezone);
  const month = today.slice(0, 7);
  const res = await loadContent(() => listItems(clientId));
  if (!res.ok) return null;
  const items = res.data;
  const plan = plans.find((p) => p.month === month);
  const week = weekDays(today);
  const [weekStart, weekEnd] = [week[0]!, week[6]!];
  const overdue = items.filter((i) => isOverdue(i, today)).sort((a, b) => (currentDue(a)!.date < currentDue(b)!.date ? -1 : 1));
  const dueThisWeek = items.filter((i) => {
    const d = currentDue(i);
    return d && d.date >= today && d.date >= weekStart && d.date <= weekEnd;
  });
  const recent = items.toSorted((a, b) => b.updatedAt.localeCompare(a.updatedAt)).slice(0, 8);
  const itemHref = (id: string) => `${base}/board/${id}`;

  return (
    <div className="page-stack">
      <h2 className="text-lg font-bold tracking-[-0.015em]">{cd.actionTitle}</h2>
      <Section index="01" title={`${c.thisMonthPlan} · ${monthLabel(month)}`}>
        {plan ? (
          <Link href={`${base}/plans/${plan.id}`} className={`${card} block space-y-3 p-(--card-pad) transition-colors hover:bg-muted/30`}>
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-base font-semibold">{plan.title}</span>
              <PlanStatusBadge status={plan.status} />
            </div>
            <Progress value={plan.itemCount ? (plan.publishedCount / plan.itemCount) * 100 : 0} className="h-2" />
            <p className="text-xs text-muted-foreground">{c.progress(plan.publishedCount, plan.itemCount)}</p>
          </Link>
        ) : (
          <EmptyState compact icon={<ClipboardList />} title={c.noPlanThisMonth} action={canWrite ? <CreatePlanButton slug={slug} defaultMonth={month} /> : undefined} />
        )}
      </Section>

      <div className="grid gap-(--gap) [&>*]:min-w-0 lg:grid-cols-2">
        <Section index="02" title={c.dueThisWeek} description={c.dueThisWeekDesc}>
          <div className={`${card} p-2`}>
            {dueThisWeek.length === 0 ? (
              <p className={`flex items-center gap-2 py-4 text-sm text-muted-foreground ${rowX}`}>
                <CalendarClock className="size-4" /> {c.nothingDue}
              </p>
            ) : (
              STAGES.map((stage) => {
                const list = dueThisWeek.filter((i) => currentDue(i)!.stage === stage);
                if (list.length === 0) return null;
                return (
                  <div key={stage} className="py-1">
                    <p className={`label-mono pt-2 pb-1 text-muted-foreground ${rowX}`}>
                      {c.stageDue[stage]} · {list.length}
                    </p>
                    <ul>
                      {list.map((i) => (
                        <ItemRow key={i.id} item={i} href={itemHref(i.id)} meta={c.dueOn(dayLabel(currentDue(i)!.date))} />
                      ))}
                    </ul>
                  </div>
                );
              })
            )}
          </div>
        </Section>

        <Section index="03" title={c.overdue} description={c.overdueDesc}>
          <div className={`${card} p-2`}>
            {overdue.length === 0 ? (
              <p className={`flex items-center gap-2 py-4 text-sm text-muted-foreground ${rowX}`}>
                <AlertTriangle className="size-4" /> {c.noOverdue}
              </p>
            ) : (
              <ul>
                {overdue.slice(0, 10).map((i) => {
                  const due = currentDue(i)!;
                  return <ItemRow key={i.id} item={i} href={itemHref(i.id)} meta={`${c.stageDue[due.stage]} · ${c.dueOn(dayLabel(due.date))}`} late />;
                })}
              </ul>
            )}
          </div>
        </Section>
      </div>

      <Section index="04" title={c.recentActivity}>
        <div className={`${card} p-2`}>
          {recent.length === 0 ? (
            <p className={`py-4 text-sm text-muted-foreground ${rowX}`}>{c.noActivity}</p>
          ) : (
            <ul>
              {recent.map((i) => (
                <ItemRow key={i.id} item={i} href={itemHref(i.id)} meta={`${c.colUpdated} ${formatRelative(i.updatedAt)}`} />
              ))}
            </ul>
          )}
        </div>
      </Section>
    </div>
  );
}

export function ActionListsSkeleton() {
  return (
    <div className="page-stack">
      <Skeleton className="h-6 w-40" />
      <Skeleton className="h-28 rounded-[1.5rem]" />
      <div className="grid gap-(--gap) [&>*]:min-w-0 lg:grid-cols-2">
        <Skeleton className="h-48 rounded-[1.5rem]" />
        <Skeleton className="h-48 rounded-[1.5rem]" />
      </div>
    </div>
  );
}
