"use client";

import * as React from "react";
import { ChevronDown, Pin } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { decideLeaveAction, decideOvertimeAction } from "@/features/erp/hr/actions";
import { DecisionButtons } from "@/features/erp/hr/components/ui";
import { weekdayDate } from "@/features/erp/hr/format";
import { th } from "@/features/erp/hr/strings";
import type { Announcement, MyApprovals, TeamScheduleDay } from "@/features/erp/hr/types";
import { formatDate, formatRelative } from "@/lib/format";
import { cn } from "@/lib/utils";
import { readMyAnnouncementAction } from "../actions";
import { tm } from "../strings";

export function WeekStrip({ week, today }: { week: TeamScheduleDay[]; today: string }) {
  return (
    <ul className="-mx-1 flex snap-x gap-2 overflow-x-auto px-1 pb-1 scrollbar-thin">
      {week.map((d) => (
        <li
          key={d.date}
          className={cn(
            "flex min-w-[88px] snap-start flex-col gap-0.5 rounded-xl border px-3 py-2 text-xs",
            d.date === today && "border-brand bg-brand/10",
            d.status !== "shift" && "text-muted-foreground",
          )}
        >
          <span className="font-medium whitespace-nowrap">{weekdayDate(d.date)}</span>
          <span className="truncate">{d.status === "shift" ? d.shiftName : th.schedule.teamStatus[d.status] ?? d.status}</span>
          {d.status === "shift" ? (
            <span className="tabular opacity-80">
              {d.startTime}–{d.endTime}
            </span>
          ) : null}
        </li>
      ))}
    </ul>
  );
}

/** Announcements; expanding an unread one marks it read. */
export function MyAnnouncements({ items }: { items: Announcement[] }) {
  const [read, setRead] = React.useState<Set<string>>(new Set());
  if (!items.length) return <p className="text-sm text-muted-foreground">{tm.home.noAnnouncements}</p>;
  const sorted = [...items].sort((a, b) => Number(b.isPinned) - Number(a.isPinned));
  return (
    <ul className="flex flex-col gap-2">
      {sorted.map((a) => {
        const unread = !a.isRead && !read.has(a.id);
        return (
          <li key={a.id}>
            <Collapsible
              onOpenChange={(open) => {
                if (open && unread) {
                  setRead((s) => new Set(s).add(a.id));
                  void readMyAnnouncementAction(a.id);
                }
              }}
            >
              <Card className="gap-0 py-0">
                <CollapsibleTrigger className="group flex w-full items-start gap-2 px-4 py-3 text-left">
                  {a.isPinned ? <Pin className="mt-0.5 size-3.5 shrink-0 text-brand" /> : null}
                  <span className="min-w-0 flex-1">
                    <span className="flex items-center gap-2">
                      <span className={cn("truncate text-sm", unread && "font-semibold")}>{a.title}</span>
                      {unread ? <Badge className="px-1.5 py-0 text-[10px]">{th.announcements.unread}</Badge> : null}
                    </span>
                    <span className="text-xs text-muted-foreground">{formatRelative(a.publishAt ?? a.createdAt)}</span>
                  </span>
                  <ChevronDown className="size-4 shrink-0 text-muted-foreground transition-transform group-data-[state=open]:rotate-180" />
                </CollapsibleTrigger>
                <CollapsibleContent className="px-4 pb-3">
                  <p className="text-sm whitespace-pre-line">{a.body}</p>
                  {a.tags.length ? (
                    <div className="mt-2 flex flex-wrap gap-1">
                      {a.tags.map((t) => (
                        <Badge key={t} variant="secondary" className="text-[11px]">
                          #{t}
                        </Badge>
                      ))}
                    </div>
                  ) : null}
                </CollapsibleContent>
              </Card>
            </Collapsible>
          </li>
        );
      })}
    </ul>
  );
}

/** Pending leave and overtime of the manager's direct reports. */
export function MyApprovalsList({ data }: { data: MyApprovals }) {
  if (!data.leaves.length && !data.overtime.length)
    return <p className="text-sm text-muted-foreground">{tm.home.approvalsEmpty}</p>;
  return (
    <ul className="flex flex-col gap-2">
      {data.leaves.map((l) => (
        <li key={l.id}>
          <Card className="gap-2 p-4">
            <div className="flex items-start justify-between gap-2">
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{l.employee.fullName}</p>
                <p className="text-xs text-muted-foreground">
                  {tm.home.leaveRequests} · {formatDate(l.startDate)}
                  {l.endDate !== l.startDate ? ` – ${formatDate(l.endDate)}` : ""} · {th.days(l.totalDays)}
                </p>
              </div>
              <Badge variant="outline">{th.leave.statuses[l.status] ?? l.status}</Badge>
            </div>
            <p className="line-clamp-2 text-sm">{l.reason}</p>
            {l.canDecide ? (
              <DecisionButtons
                size="sm"
                onApprove={() => decideLeaveAction(l.id, true)}
                onReject={(r) => decideLeaveAction(l.id, false, r)}
              />
            ) : null}
          </Card>
        </li>
      ))}
      {data.overtime.map((o) => (
        <li key={o.id}>
          <Card className="gap-2 p-4">
            <div className="min-w-0">
              <p className="truncate text-sm font-medium">{o.employee.fullName}</p>
              <p className="text-xs text-muted-foreground tabular">
                {tm.home.overtimeRequests} · {formatDate(o.date)} · {o.startTime}–{o.endTime} · {th.hours(o.hours)}
              </p>
            </div>
            <p className="line-clamp-2 text-sm">{o.reason}</p>
            <DecisionButtons
              size="sm"
              onApprove={() => decideOvertimeAction(o.id, "approve")}
              onReject={(r) => decideOvertimeAction(o.id, "reject", r)}
            />
          </Card>
        </li>
      ))}
    </ul>
  );
}
