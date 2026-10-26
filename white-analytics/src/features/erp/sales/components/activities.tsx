"use client";

import * as React from "react";
import Link from "next/link";
import { CalendarClock, Check, MessageSquare, Pencil, Phone, Plus, StickyNote, Trash2, Users, Mail, ListTodo, Repeat } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { formatDateTime } from "@/lib/format";
import { cn } from "@/lib/utils";
import { deleteActivityAction, saveActivityAction, setActivityStatusAction, type ActivityValues } from "../actions";
import { nameOf } from "../lib";
import { ACTIVITY_STATUS, ACTIVITY_TYPE, PRIORITY, lbl, opts, ss } from "../strings";
import type { Activity } from "../types";
import { ConfirmAction, type FieldErrors, Fld, FormDialog, FormFooter, type Option, Picker, SalesBadge, SimpleSelect } from "./ui";

const s = ss.activity;

const ICON: Record<string, React.ElementType> = {
  telepon: Phone,
  wa: MessageSquare,
  meeting: Users,
  catatan: StickyNote,
  tugas: ListTodo,
  email: Mail,
};

/** RFC 3339 → value for <input type="datetime-local"> in local time. */
function toLocalInput(ts: string | null): string {
  if (!ts) return "";
  const d = new Date(ts);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}
const fromLocalInput = (v: string) => (v ? new Date(v).toISOString() : null);

/** Activity and task list with done toggles. */
export function ActivityList({
  activities,
  owners,
  canWrite,
  leadId,
  dealId,
  showSubject,
  emptyLabel = s.empty,
}: {
  activities: Activity[];
  owners: Option[];
  canWrite: boolean;
  leadId?: string;
  dealId?: string;
  /** links each row to its lead or deal (dashboard) */
  showSubject?: boolean;
  emptyLabel?: string;
}) {
  const names = Object.fromEntries(owners.map((o) => [o.value, o.label]));
  const { pending, run } = useErpAction();
  const [now] = React.useState(() => Date.now());
  return (
    <div className="flex flex-col gap-3">
      {canWrite && (leadId || dealId) ? (
        <div>
          <ActivityFormDialog
            activity={null}
            owners={owners}
            leadId={leadId}
            dealId={dealId}
            trigger={
              <Button size="sm" variant="outline">
                <Plus className="size-3.5" /> {s.add}
              </Button>
            }
          />
        </div>
      ) : null}
      {activities.length === 0 ? (
        <p className="py-4 text-sm text-muted-foreground">{emptyLabel}</p>
      ) : (
        <ul className="divide-y rounded-lg border">
          {activities.map((act) => {
            const Icon = ICON[act.activityType] ?? CalendarClock;
            const done = act.status === "done";
            const overdue = !done && act.status !== "cancelled" && act.dueAt && new Date(act.dueAt).getTime() < now;
            const subject = act.dealId ? `/erp/sales/pipeline/${act.dealId}` : act.leadId ? `/erp/sales/leads/${act.leadId}` : null;
            return (
              <li key={act.id} className="flex items-start gap-3 px-3 py-2.5">
                {canWrite ? (
                  <Checkbox
                    className="mt-0.5"
                    checked={done}
                    disabled={pending}
                    aria-label={done ? s.reopen : s.markDone}
                    onCheckedChange={(c) =>
                      void run(() => setActivityStatusAction(act, c === true ? "done" : "open"), {
                        success: c === true ? s.done : undefined,
                      })
                    }
                  />
                ) : (
                  <Check className={cn("mt-0.5 size-4", done ? "text-positive" : "text-muted-foreground/30")} />
                )}
                <Icon className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
                <div className="min-w-0 flex-1">
                  <div className={cn("text-sm font-medium", done && "text-muted-foreground line-through")}>
                    {showSubject && subject ? (
                      <Link href={subject} className="hover:underline">
                        {act.title}
                      </Link>
                    ) : (
                      act.title
                    )}
                  </div>
                  <div className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
                    <span>{lbl(ACTIVITY_TYPE, act.activityType)}</span>
                    {act.dueAt ? (
                      <span className={cn("tabular", overdue && "font-medium text-destructive")}>
                        {overdue ? `${s.overdue} · ` : ""}
                        {formatDateTime(act.dueAt)}
                      </span>
                    ) : null}
                    {act.recurrence?.freq ? (
                      <span className="inline-flex items-center gap-0.5">
                        <Repeat className="size-3" /> {s.freq[act.recurrence.freq] ?? act.recurrence.freq}
                      </span>
                    ) : null}
                    <span>{nameOf(names, act.ownerUserId)}</span>
                    {act.priority && act.priority !== "normal" ? <span>{lbl(PRIORITY, act.priority)}</span> : null}
                    {act.status !== "open" && act.status !== "done" ? (
                      <SalesBadge value={act.status} map={ACTIVITY_STATUS} />
                    ) : null}
                  </div>
                  {act.notes ? <p className="mt-1 text-xs whitespace-pre-line text-muted-foreground">{act.notes}</p> : null}
                </div>
                {canWrite ? (
                  <div className="flex shrink-0 gap-0.5">
                    <ActivityFormDialog
                      activity={act}
                      owners={owners}
                      trigger={
                        <Button size="icon-sm" variant="ghost" aria-label={ss.edit}>
                          <Pencil className="size-3.5" />
                        </Button>
                      }
                    />
                    <ConfirmAction
                      title={s.deleteTitle}
                      action={() => deleteActivityAction(act.id)}
                      trigger={
                        <Button size="icon-sm" variant="ghost" aria-label={ss.delete}>
                          <Trash2 className="size-3.5" />
                        </Button>
                      }
                    />
                  </div>
                ) : null}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

function ActivityFormDialog({
  activity,
  owners,
  leadId,
  dealId,
  trigger,
}: {
  activity: Activity | null;
  owners: Option[];
  leadId?: string;
  dealId?: string;
  trigger: React.ReactNode;
}) {
  return (
    <FormDialog trigger={trigger} title={activity ? s.editTitle : s.addTitle}>
      {(close) => <ActivityForm activity={activity} owners={owners} leadId={leadId} dealId={dealId} close={close} />}
    </FormDialog>
  );
}

function ActivityForm({
  activity,
  owners,
  leadId,
  dealId,
  close,
}: {
  activity: Activity | null;
  owners: Option[];
  leadId?: string;
  dealId?: string;
  close: () => void;
}) {
  const [v, setV] = React.useState<ActivityValues>({
    leadId: activity?.leadId ?? leadId ?? "",
    dealId: activity?.dealId ?? dealId ?? "",
    activityType: activity?.activityType ?? "telepon",
    title: activity?.title ?? "",
    notes: activity?.notes ?? "",
    priority: activity?.priority || "normal",
    status: activity?.status || "open",
    dueAt: activity?.dueAt ?? null,
    reminderAt: activity?.reminderAt ?? null,
    recurrence: activity?.recurrence?.freq ? activity.recurrence : null,
    ownerUserId: activity?.ownerUserId ?? "",
  });
  const [errors, setErrors] = React.useState<FieldErrors>();
  const { pending, run } = useErpAction();
  const set = <K extends keyof ActivityValues>(k: K, val: ActivityValues[K]) => setV((p) => ({ ...p, [k]: val }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveActivityAction(activity?.id ?? null, v), {
          success: ss.saved,
          onOk: close,
          onFail: (r) => setErrors(r.fieldErrors),
        });
      }}
    >
      <Fld label={`${s.title} *`} htmlFor="act-title" error={errors?.title} className="sm:col-span-2">
        <Input id="act-title" value={v.title} onChange={(e) => set("title", e.target.value)} />
      </Fld>
      <Fld label={s.type} htmlFor="act-type">
        <SimpleSelect id="act-type" value={v.activityType} onChange={(x) => set("activityType", x)} options={opts(ACTIVITY_TYPE)} />
      </Fld>
      <Fld label={s.priority} htmlFor="act-priority">
        <SimpleSelect id="act-priority" value={v.priority ?? "normal"} onChange={(x) => set("priority", x)} options={opts(PRIORITY)} />
      </Fld>
      <Fld label={s.due} htmlFor="act-due">
        <Input
          id="act-due"
          type="datetime-local"
          value={toLocalInput(v.dueAt ?? null)}
          onChange={(e) => set("dueAt", fromLocalInput(e.target.value))}
        />
      </Fld>
      <Fld label={s.reminder} htmlFor="act-reminder">
        <Input
          id="act-reminder"
          type="datetime-local"
          value={toLocalInput(v.reminderAt ?? null)}
          onChange={(e) => set("reminderAt", fromLocalInput(e.target.value))}
        />
      </Fld>
      <Fld label={s.status} htmlFor="act-status">
        <SimpleSelect id="act-status" value={v.status ?? "open"} onChange={(x) => set("status", x)} options={opts(ACTIVITY_STATUS)} />
      </Fld>
      <Fld label={ss.owner} htmlFor="act-owner">
        <Picker id="act-owner" value={v.ownerUserId ?? ""} onChange={(x) => set("ownerUserId", x)} options={owners} allowEmpty={ss.noOwner} />
      </Fld>
      <Fld label={s.recurrence} htmlFor="act-freq">
        <SimpleSelect
          id="act-freq"
          value={v.recurrence?.freq ?? ""}
          onChange={(x) => set("recurrence", x ? { freq: x, interval: v.recurrence?.interval ?? 1, until: v.recurrence?.until ?? "" } : null)}
          options={opts(s.freq)}
          emptyLabel={s.recurrenceNone}
        />
      </Fld>
      {v.recurrence ? (
        <Fld label={ss.contracts.end} htmlFor="act-until">
          <Input
            id="act-until"
            type="date"
            value={v.recurrence.until}
            onChange={(e) => set("recurrence", { ...v.recurrence!, until: e.target.value })}
          />
        </Fld>
      ) : null}
      <Fld label={ss.notes} htmlFor="act-notes" className="sm:col-span-2">
        <Textarea id="act-notes" rows={3} value={v.notes ?? ""} onChange={(e) => set("notes", e.target.value)} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}
