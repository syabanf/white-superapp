"use client";

import { CalendarDays, CheckSquare, MessageSquare, Paperclip, Repeat, Target } from "lucide-react";
import { Checkbox } from "@/components/ui/checkbox";
import { useErpAction } from "@/features/erp/shared/client";
import { formatDate } from "@/lib/format";
import { cn } from "@/lib/utils";
import { occurrenceAction } from "../actions";
import { sp } from "../strings";
import type { TaskItem } from "../types";
import { PriorityBadge, TaskStatusBadge } from "./common";

const s = sp.tasks;

/** Date, progress, cadence, KPI link and counters of one occurrence. */
export function TaskMetaLine({ item, showAssignee = false }: { item: TaskItem; showAssignee?: boolean }) {
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
      <span className={cn("inline-flex items-center gap-1", item.overdue && "font-medium text-destructive")}>
        <CalendarDays className="size-3.5" />
        {formatDate(item.occurrenceDate)}
      </span>
      {item.status === "in_progress" ? <span className="tabular">{item.progress}%</span> : null}
      {item.recurrence !== "once" ? (
        <span className="inline-flex items-center gap-1">
          <Repeat className="size-3.5" />
          {sp.recurrence[item.recurrence]}
        </span>
      ) : null}
      {item.kpiIndicatorName ? (
        <span className="inline-flex items-center gap-1">
          <Target className="size-3.5" />
          {item.kpiIndicatorName}
          {item.kpiContribution != null ? ` (+${item.kpiContribution})` : ""}
        </span>
      ) : null}
      {item.subtaskCount > 0 ? (
        <span className="inline-flex items-center gap-1">
          <CheckSquare className="size-3.5" />
          {s.subtaskCount(item.subtaskCount)}
        </span>
      ) : null}
      {item.commentCount > 0 ? (
        <span className="inline-flex items-center gap-1">
          <MessageSquare className="size-3.5" />
          {item.commentCount}
        </span>
      ) : null}
      {item.attachmentCount > 0 ? (
        <span className="inline-flex items-center gap-1">
          <Paperclip className="size-3.5" />
          {item.attachmentCount}
        </span>
      ) : null}
      {showAssignee ? <span>{s.pic(item.assigneeName)}</span> : null}
      {item.lastNote ? (
        <span className="max-w-full truncate" title={item.lastNote}>
          “{item.lastNote}”
        </span>
      ) : null}
    </div>
  );
}

/**
 * One list row: tick to finish (personal tasks close at once, department
 * tasks go to review), tap the title for the detail sheet.
 */
export function TaskRow({
  item,
  onOpen,
  showAssignee = false,
}: {
  item: TaskItem;
  onOpen: (id: string) => void;
  showAssignee?: boolean;
}) {
  const { pending, run } = useErpAction();
  const closed = item.status === "approved" || item.status === "done";
  const canTick = item.canWork && item.subtaskCount === 0 && !(item.status === "approved" && item.reviewRequired);
  return (
    <li className={cn("flex items-start gap-3 px-4 py-3", item.status === "approved" && "opacity-60")}>
      <Checkbox
        aria-label={closed ? sp.myTasks.untick(item.title) : sp.myTasks.tick(item.title)}
        checked={closed}
        disabled={pending || !canTick}
        onCheckedChange={(checked) =>
          run(() => occurrenceAction(item.occurrenceId, { action: checked ? "done" : "reopen" }))
        }
        className="mt-1 size-5"
      />
      <button
        type="button"
        onClick={() => onOpen(item.occurrenceId)}
        className="min-w-0 flex-1 rounded-md text-left outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <span
          className={cn(
            "flex flex-wrap items-center gap-2 text-sm font-medium",
            item.status === "approved" && "line-through",
          )}
        >
          {item.title}
          <PriorityBadge priority={item.priority} />
          {item.status !== "pending" || item.overdue ? <TaskStatusBadge status={item.status} overdue={item.overdue} /> : null}
        </span>
        <TaskMetaLine item={item} showAssignee={showAssignee} />
      </button>
    </li>
  );
}
