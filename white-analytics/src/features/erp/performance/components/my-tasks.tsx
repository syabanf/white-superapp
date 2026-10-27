"use client";

import * as React from "react";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { ErpEmpty } from "@/features/erp/shared/states";
import { useErpAction } from "@/features/erp/shared/client";
import { formatDate } from "@/lib/format";
import { cn } from "@/lib/utils";
import { createTaskAction } from "../actions";
import { groupBySection, parseQuickCapture, scheduleFor, SECTION_ORDER } from "../lib";
import { sp } from "../strings";
import type { TaskItem, TaskMeta } from "../types";
import { TaskDetailSheet } from "./task-detail-sheet";
import { TaskRow } from "./task-row";

const m = sp.myTasks;

/** One box to add a personal task: type, Enter. Focus stays for adding several in a row. */
function QuickCapture({ meta }: { meta: TaskMeta }) {
  const [text, setText] = React.useState("");
  const { pending, run } = useErpAction();
  const parsed = parseQuickCapture(text, meta.today);
  const employeeId = meta.me.employeeId;
  const departmentId = meta.me.departmentId;
  const linked = !!employeeId && !!departmentId;

  const submit = () => {
    if (!parsed || pending || !linked) return;
    run(
      () =>
        createTaskAction({
          title: parsed.title,
          recurrence: parsed.recurrence,
          priority: parsed.priority ?? "normal",
          assigneeEmployeeId: employeeId,
          departmentId,
          ...scheduleFor(parsed.recurrence, parsed.dueDate),
          subtasks: [],
        }),
      { success: m.added, onOk: () => setText("") },
    );
  };

  const preview =
    parsed && (parsed.dueDate !== meta.today || parsed.priority || parsed.recurrence !== "once")
      ? [
          parsed.recurrence !== "once"
            ? m.quickPreviewRecurrence(sp.recurrence[parsed.recurrence])
            : m.quickPreviewDue(formatDate(parsed.dueDate)),
          parsed.priority ? m.quickPreviewPriority(sp.priority[parsed.priority]) : null,
        ]
          .filter(Boolean)
          .join(" · ")
      : m.quickHint;

  return (
    <form
      className="flex flex-col gap-1.5"
      onSubmit={(e) => {
        e.preventDefault();
        submit();
      }}
    >
      <div className="flex gap-2">
        <Input
          aria-label={m.quickAdd}
          placeholder={m.quickPlaceholder}
          value={text}
          onChange={(e) => setText(e.target.value)}
          className="h-11 flex-1 text-base sm:text-sm"
          autoComplete="off"
          enterKeyHint="done"
          disabled={!linked}
        />
        <Button type="submit" className="h-11" disabled={!parsed || pending || !linked} aria-label={m.quickAdd}>
          <Plus /> <span className="hidden sm:inline">{sp.common.add}</span>
        </Button>
      </div>
      <p className="text-xs text-muted-foreground" aria-live="polite">
        {linked ? preview : sp.common.unlinked}
      </p>
    </form>
  );
}

/** Own occurrences grouped into overdue, today, upcoming, waiting for review and done. */
export function MyTasks({ items, meta }: { items: TaskItem[]; meta: TaskMeta }) {
  const [openId, setOpenId] = React.useState<string | null>(null);
  const sections = React.useMemo(() => groupBySection(items, meta.today), [items, meta.today]);
  return (
    <>
      <QuickCapture meta={meta} />
      {items.length === 0 ? (
        <ErpEmpty title={sp.tasks.emptyMine} />
      ) : (
        <Card className="gap-0 overflow-hidden py-0">
          {SECTION_ORDER.filter((k) => sections[k].length > 0).map((k) => (
            <section key={k} aria-labelledby={`sec-${k}`}>
              <h2
                id={`sec-${k}`}
                className={cn(
                  "label-mono border-b bg-muted/40 px-(--card-pad) py-2",
                  k === "overdue" ? "text-destructive" : "text-muted-foreground",
                )}
              >
                {m.sections[k]} · {sections[k].length}
              </h2>
              <ul className="divide-y">
                {sections[k].map((t) => (
                  <TaskRow key={t.occurrenceId} item={t} onOpen={setOpenId} />
                ))}
              </ul>
            </section>
          ))}
        </Card>
      )}
      <TaskDetailSheet occurrenceId={openId} meta={meta} onClose={() => setOpenId(null)} />
    </>
  );
}
