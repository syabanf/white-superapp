"use client";

import * as React from "react";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { createTaskAction } from "../actions";
import { lines } from "../lib";
import { sp } from "../strings";
import type { Priority, Recurrence, TaskMeta } from "../types";
import { FormRow } from "./common";

const s = sp.tasks;
const NONE = "__none__";

/** New department task: recurring or one-off, assignee, priority, KPI link, weighted subtasks. */
export function TaskFormDialog({ meta, departmentId }: { meta: TaskMeta; departmentId: string }) {
  const [open, setOpen] = React.useState(false);
  const { pending, run } = useErpAction();
  const [errors, setErrors] = React.useState<Record<string, string[]>>({});
  const initial = {
    title: "",
    description: "",
    assignee: NONE,
    recurrence: "daily" as Recurrence,
    weeklyDay: "1",
    monthlyDay: "1",
    dueDate: meta.today,
    priority: "normal" as Priority,
    indicator: NONE,
    contribution: "",
    subtasks: "",
  };
  const [form, setForm] = React.useState(initial);
  const set = <K extends keyof typeof form>(k: K, v: (typeof form)[K]) => setForm((f) => ({ ...f, [k]: v }));
  const members = meta.members.filter((m) => m.departmentId === departmentId);
  const indicators = meta.indicators.filter((i) => i.departmentIds.length === 0 || i.departmentIds.includes(departmentId));

  const submit = () =>
    run(
      () =>
        createTaskAction({
          departmentId,
          title: form.title,
          description: form.description || undefined,
          assigneeEmployeeId: form.assignee === NONE ? undefined : form.assignee,
          recurrence: form.recurrence,
          weeklyDay: form.recurrence === "weekly" ? Number(form.weeklyDay) : undefined,
          monthlyDay: form.recurrence === "monthly" ? Number(form.monthlyDay) : undefined,
          dueDate: form.recurrence === "once" ? form.dueDate : undefined,
          priority: form.priority,
          kpiIndicatorId: form.indicator === NONE ? undefined : form.indicator,
          kpiContribution: form.indicator !== NONE && form.contribution !== "" ? Number(form.contribution) : undefined,
          subtasks: lines(form.subtasks),
        }),
      {
        success: s.created,
        onOk: () => {
          setOpen(false);
          setForm(initial);
          setErrors({});
        },
        onFail: (r) => setErrors(r.fieldErrors ?? {}),
      },
    );

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm" disabled={!departmentId}>
          <Plus className="size-4" /> {s.newTask}
        </Button>
      </DialogTrigger>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-2xl">
        <form
          onSubmit={(e) => {
            e.preventDefault();
            submit();
          }}
          className="flex flex-col gap-4"
        >
          <DialogHeader>
            <DialogTitle>{s.newTaskTitle}</DialogTitle>
            <DialogDescription>{s.newTaskDesc}</DialogDescription>
          </DialogHeader>
          <div className="grid gap-3 sm:grid-cols-2">
            <FormRow label={s.titleLabel} htmlFor="task-title" className="sm:col-span-2" error={errors.title?.[0]}>
              <Input id="task-title" value={form.title} onChange={(e) => set("title", e.target.value)} required minLength={3} maxLength={200} autoFocus />
            </FormRow>
            <FormRow label={s.assignee}>
              <Select value={form.assignee} onValueChange={(v) => set("assignee", v)}>
                <SelectTrigger className="w-full" aria-label={s.assignee}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={NONE}>{s.wholeDepartment}</SelectItem>
                  {members.map((m) => (
                    <SelectItem key={m.id} value={m.id}>
                      {m.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormRow>
            <FormRow label={s.recurrenceLabel}>
              <Select value={form.recurrence} onValueChange={(v) => set("recurrence", v as Recurrence)}>
                <SelectTrigger className="w-full" aria-label={s.recurrenceLabel}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {Object.entries(sp.recurrence).map(([v, l]) => (
                    <SelectItem key={v} value={v}>
                      {l}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormRow>
            {form.recurrence === "weekly" ? (
              <FormRow label={s.weekday}>
                <Select value={form.weeklyDay} onValueChange={(v) => set("weeklyDay", v)}>
                  <SelectTrigger className="w-full" aria-label={s.weekday}>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {[1, 2, 3, 4, 5, 6, 7].map((d) => (
                      <SelectItem key={d} value={String(d)}>
                        {sp.days[d]}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormRow>
            ) : null}
            {form.recurrence === "monthly" ? (
              <FormRow label={s.monthDay} htmlFor="task-mday">
                <Input id="task-mday" type="number" min={1} max={28} value={form.monthlyDay} onChange={(e) => set("monthlyDay", e.target.value)} />
              </FormRow>
            ) : null}
            {form.recurrence === "once" ? (
              <FormRow label={s.dueDate} error={errors.dueDate?.[0]}>
                <DatePicker value={form.dueDate} onChange={(v) => set("dueDate", v || meta.today)} />
              </FormRow>
            ) : null}
            <FormRow label={s.priorityLabel}>
              <Select value={form.priority} onValueChange={(v) => set("priority", v as Priority)}>
                <SelectTrigger className="w-full" aria-label={s.priorityLabel}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {Object.entries(sp.priority).map(([v, l]) => (
                    <SelectItem key={v} value={v}>
                      {l}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormRow>
            <FormRow label={s.indicator}>
              <Select value={form.indicator} onValueChange={(v) => set("indicator", v)}>
                <SelectTrigger className="w-full" aria-label={s.indicator}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={NONE}>{s.noIndicator}</SelectItem>
                  {indicators.map((i) => (
                    <SelectItem key={i.id} value={i.id}>
                      {i.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormRow>
            <FormRow label={s.contribution} htmlFor="task-contrib" hint={s.contributionHint}>
              <Input
                id="task-contrib"
                type="number"
                step="any"
                inputMode="decimal"
                value={form.contribution}
                disabled={form.indicator === NONE}
                onChange={(e) => set("contribution", e.target.value)}
              />
            </FormRow>
            <FormRow label={s.description} htmlFor="task-desc" className="sm:col-span-2">
              <Textarea id="task-desc" value={form.description} onChange={(e) => set("description", e.target.value)} />
            </FormRow>
            <FormRow label={s.subtasks} htmlFor="task-subs" hint={s.subtasksHint} className="sm:col-span-2">
              <Textarea id="task-subs" value={form.subtasks} onChange={(e) => set("subtasks", e.target.value)} />
            </FormRow>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              {sp.common.cancel}
            </Button>
            <Button type="submit" disabled={pending || form.title.trim().length < 3}>
              {pending ? sp.common.saving : sp.common.save}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
