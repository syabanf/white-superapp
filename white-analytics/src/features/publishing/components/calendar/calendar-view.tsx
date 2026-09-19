"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { toast } from "@/lib/toast";
import { reschedulePost } from "@/features/publishing/actions";
import { canReschedule, type Platform, type PostRole, type PostStatus } from "@/features/publishing/lib";
import type { CalendarPost } from "@/features/publishing/queries";
import {
  dayLabelLong,
  monthOf,
  moveToDay,
  toZoned,
  weekDays,
  weekdayLabel,
} from "@/features/publishing/time";
import { StatusDot, StatusLegend } from "@/features/publishing/components/post-status";
import { DND_MIME, PostChip } from "./post-chip";
import { CalendarToolbar, type CalendarView as ViewKey } from "./calendar-toolbar";
import { p } from "@/features/publishing/strings";
import { cn } from "@/lib/utils";

type Props = {
  slug: string;
  timezone: string;
  view: ViewKey;
  date: string;
  today: string;
  /** month grid rows (month view) or the 7 days (week view) */
  days: string[][];
  posts: CalendarPost[];
  role: PostRole;
  platform: Platform | "ALL";
  status: PostStatus | "ALL";
};

const MAX_CHIPS = 3;

export function CalendarView({
  slug,
  timezone,
  view,
  date,
  today,
  days,
  posts,
  role,
  platform,
  status,
}: Props) {
  const router = useRouter();
  const base = `/clients/${slug}/publish`;
  const [selected, setSelected] = React.useState<string>(() =>
    days.flat().includes(today) ? today : days[0]![0]!,
  );
  const [overDay, setOverDay] = React.useState<string | null>(null);
  const [pending, start] = React.useTransition();

  const filtered = posts.filter(
    (po) =>
      (platform === "ALL" || po.platforms.includes(platform)) && (status === "ALL" || po.status === status),
  );
  const byDay = React.useMemo(() => {
    const m = new Map<string, CalendarPost[]>();
    for (const po of filtered) {
      const key = toZoned(new Date(po.at), timezone).date;
      m.set(key, [...(m.get(key) ?? []), po]);
    }
    return m;
  }, [filtered, timezone]);
  const postById = React.useMemo(() => new Map(posts.map((po) => [po.id, po])), [posts]);
  const draggable = (po: CalendarPost) => canReschedule(po.status, role);
  const href = (po: CalendarPost) => `${base}/posts/${po.id}`;

  const onDrop = (dayKey: string, e: React.DragEvent) => {
    e.preventDefault();
    setOverDay(null);
    const id = e.dataTransfer.getData(DND_MIME) || e.dataTransfer.getData("text/plain");
    const po = postById.get(id);
    if (!po || !draggable(po)) return;
    const current = toZoned(new Date(po.at), timezone).date;
    if (current === dayKey) return;
    const next = moveToDay(new Date(po.at), dayKey, timezone);
    start(async () => {
      const res = await reschedulePost({ postId: po.id, scheduledAt: next.toISOString() });
      if (res.ok) {
        toast.success(p.rescheduled);
        router.refresh();
      } else toast.error(res.error);
    });
  };
  const dropProps = (dayKey: string) =>
    role === "VIEWER"
      ? {}
      : {
          onDragOver: (e: React.DragEvent) => {
            if (e.dataTransfer.types.includes(DND_MIME) || e.dataTransfer.types.includes("text/plain")) {
              e.preventDefault();
              e.dataTransfer.dropEffect = "move";
              if (overDay !== dayKey) setOverDay(dayKey);
            }
          },
          onDragLeave: () => setOverDay((d) => (d === dayKey ? null : d)),
          onDrop: (e: React.DragEvent) => onDrop(dayKey, e),
        };

  const month = monthOf(date);
  const weekHeader = (
    <div className="grid grid-cols-7 border-b">
      {Array.from({ length: 7 }, (_, i) => (
        <div key={i} className="label-mono px-2 py-1.5 text-muted-foreground">
          {weekdayLabel(i)}
        </div>
      ))}
    </div>
  );

  const dayList = (key: string, emptyMsg: string) => {
    const items = byDay.get(key) ?? [];
    return items.length === 0 ? (
      <p className="px-1 py-3 text-sm text-muted-foreground">{emptyMsg}</p>
    ) : (
      <ul className="space-y-1.5">
        {items.map((po) => (
          <li key={po.id}>
            <PostChip post={po} href={href(po)} timeZone={timezone} />
          </li>
        ))}
      </ul>
    );
  };

  return (
    <div className={cn("space-y-4", pending && "opacity-70")}>
      <CalendarToolbar
        view={view}
        date={date}
        today={today}
        platform={platform}
        status={status}
        composeHref={`${base}/new`}
      />

      {view === "month" ? (
        <>
          <div className="overflow-hidden rounded-[1.5rem] bg-card shadow-(--card-shadow)">
            {weekHeader}
            {days.map((row, ri) => (
              <div key={ri} className="grid grid-cols-7 border-b last:border-b-0">
                {row.map((key) => {
                  const items = byDay.get(key) ?? [];
                  const inMonth = key.startsWith(month);
                  const isToday = key === today;
                  const isSelected = key === selected;
                  const dayNum = Number(key.slice(8));
                  return (
                    <div
                      key={key}
                      {...dropProps(key)}
                      onClick={() => setSelected(key)}
                      className={cn(
                        "min-h-14 border-r p-1 transition-colors duration-150 last:border-r-0 md:min-h-28 md:p-1.5",
                        !inMonth && "bg-muted/30 text-muted-foreground",
                        overDay === key && "bg-brand/10 ring-1 ring-brand ring-inset",
                        isSelected && "md:bg-transparent bg-brand/5",
                      )}
                    >
                      <div className="flex items-center justify-between">
                        <span
                          className={cn(
                            "tabular inline-flex size-6 items-center justify-center rounded-full text-xs font-medium",
                            isToday && "bg-brand text-primary-foreground",
                          )}
                        >
                          {dayNum}
                        </span>
                        {items.length > 0 ? (
                          <span className="label-mono hidden text-muted-foreground md:inline">
                            {items.length}
                          </span>
                        ) : null}
                      </div>
                      {/* desktop: chips */}
                      <ul className="mt-1 hidden space-y-1 md:block">
                        {items.slice(0, MAX_CHIPS).map((po) => (
                          <li key={po.id}>
                            <PostChip
                              post={po}
                              href={href(po)}
                              timeZone={timezone}
                              draggable={draggable(po)}
                              compact
                            />
                          </li>
                        ))}
                        {items.length > MAX_CHIPS ? (
                          <li>
                            <Link
                              href={`${base}?view=week&date=${key}`}
                              className="block px-1 text-xs font-medium text-brand hover:underline"
                            >
                              {p.more(items.length - MAX_CHIPS)}
                            </Link>
                          </li>
                        ) : null}
                      </ul>
                      {/* mobile: dots */}
                      <div className="mt-1 flex flex-wrap gap-0.5 md:hidden">
                        {items.slice(0, 4).map((po) => (
                          <StatusDot key={po.id} status={po.status} className="size-1.5" />
                        ))}
                        {items.length > 4 ? (
                          <span className="text-xs leading-none text-muted-foreground">
                            +{items.length - 4}
                          </span>
                        ) : null}
                      </div>
                    </div>
                  );
                })}
              </div>
            ))}
          </div>
          <div className="md:hidden">
            <p className="label-mono mb-2 text-brand">{dayLabelLong(selected)}</p>
            {dayList(selected, p.noPostsDay)}
          </div>
        </>
      ) : (
        <>
          <div className="hidden overflow-hidden rounded-[1.5rem] bg-card shadow-(--card-shadow) md:block">
            {weekHeader}
            <div className="grid grid-cols-7">
              {weekDays(date).map((key) => {
                const items = byDay.get(key) ?? [];
                const isToday = key === today;
                return (
                  <div
                    key={key}
                    {...dropProps(key)}
                    className={cn(
                      "min-h-64 border-r p-1.5 transition-colors duration-150 last:border-r-0",
                      overDay === key && "bg-brand/10 ring-1 ring-brand ring-inset",
                    )}
                  >
                    <span
                      className={cn(
                        "tabular mb-1.5 inline-flex size-6 items-center justify-center rounded-full text-xs font-medium",
                        isToday && "bg-brand text-primary-foreground",
                      )}
                    >
                      {Number(key.slice(8))}
                    </span>
                    <ul className="space-y-1">
                      {items.map((po) => (
                        <li key={po.id}>
                          <PostChip
                            post={po}
                            href={href(po)}
                            timeZone={timezone}
                            draggable={draggable(po)}
                            compact
                          />
                        </li>
                      ))}
                    </ul>
                  </div>
                );
              })}
            </div>
          </div>
          <div className="space-y-4 md:hidden">
            {weekDays(date).some((k) => (byDay.get(k) ?? []).length > 0) ? (
              weekDays(date)
                .filter((k) => (byDay.get(k) ?? []).length > 0)
                .map((k) => (
                  <div key={k}>
                    <p
                      className={cn("label-mono mb-2", k === today ? "text-brand" : "text-muted-foreground")}
                    >
                      {dayLabelLong(k)}
                    </p>
                    {dayList(k, p.noPostsDay)}
                  </div>
                ))
            ) : (
              <p className="rounded-xl border border-dashed px-3 py-8 text-center text-sm text-muted-foreground">
                {p.noPostsWeek}
              </p>
            )}
          </div>
        </>
      )}

      <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
        <StatusLegend />
        {role !== "VIEWER" ? (
          <p className="hidden text-xs text-muted-foreground md:block">{p.dragHint}</p>
        ) : null}
      </div>
    </div>
  );
}
