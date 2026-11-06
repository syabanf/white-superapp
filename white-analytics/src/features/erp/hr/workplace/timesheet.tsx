"use client";

import * as React from "react";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { ChevronLeft, ChevronRight, Pencil, Plus, Send, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { formatDate, formatDecimal } from "@/lib/format";
import { toast } from "@/lib/toast";
import { ConfirmAction, DecisionButtons, Fld, FormDialog, FormFooter, SimpleSelect, type FieldErrors } from "../components/ui";
import { decideTimesheetAction, deleteTimesheetEntryAction, saveTimesheetEntryAction, submitTimesheetAction } from "./actions";
import { tw } from "./strings";
import type { ProjectOption, TimesheetEntry } from "./types";

const s = tw.timesheet;
const statusKinds = { draft: "neutral", submitted: "warning", approved: "good", rejected: "critical" } as const;
const editable = (e: TimesheetEntry) => e.status === "draft" || e.status === "rejected";

function EntryForm({ entry, projects, today, close }: { entry?: TimesheetEntry; projects: ProjectOption[]; today: string; close: () => void }) {
  const [v, setV] = React.useState({
    id: entry?.id ?? "",
    workDate: entry?.workDate ?? today,
    projectId: entry?.projectId ?? "",
    hours: entry ? String(Number(entry.hours)) : "",
    description: entry?.description ?? "",
  });
  const [errors, setErrors] = React.useState<FieldErrors>();
  const { pending, run } = useErpAction();
  const set = (p: Partial<typeof v>) => setV((x) => ({ ...x, ...p }));
  const options = projects.map((p) => ({ value: p.id, label: p.clientName ? `${p.name} · ${p.clientName}` : p.name }));
  return (
    <form
      className="grid gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        const projectName = projects.find((p) => p.id === v.projectId)?.name ?? "";
        void run(() => saveTimesheetEntryAction({ ...v, hours: v.hours.replace(",", "."), projectName }), {
          success: s.saved,
          onOk: close,
          onFail: (r) => setErrors(r.fieldErrors),
        });
      }}
    >
      <Fld label={s.date} htmlFor="ts-date" error={errors?.workDate}>
        <DatePicker id="ts-date" value={v.workDate} onChange={(x) => set({ workDate: x > today ? today : x })} />
      </Fld>
      <Fld label={s.project} htmlFor="ts-project" error={errors?.projectId}>
        <SimpleSelect id="ts-project" value={v.projectId} onChange={(x) => set({ projectId: x })} options={options} emptyLabel={s.internal} />
      </Fld>
      <Fld label={s.hours} htmlFor="ts-hours" error={errors?.hours} hint={s.hoursHint}>
        <Input id="ts-hours" inputMode="decimal" value={v.hours} onChange={(e) => set({ hours: e.target.value })} required />
      </Fld>
      <Fld label={s.description} htmlFor="ts-desc">
        <Textarea id="ts-desc" rows={2} value={v.description} onChange={(e) => set({ description: e.target.value })} maxLength={500} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

/** The week switcher: links keep the week in the URL. */
function WeekNav({ from, to, prev, next }: { from: string; to: string; prev: string; next: string | null }) {
  return (
    <div className="flex items-center gap-1">
      <Button size="icon-sm" variant="outline" asChild aria-label={s.prevWeek}>
        <Link href={`?week=${prev}`}>
          <ChevronLeft className="size-4" />
        </Link>
      </Button>
      <span className="px-2 text-sm font-medium tabular-nums">{s.week(formatDate(from), formatDate(to))}</span>
      {next ? (
        <Button size="icon-sm" variant="outline" asChild aria-label={s.nextWeek}>
          <Link href={`?week=${next}`}>
            <ChevronRight className="size-4" />
          </Link>
        </Button>
      ) : null}
    </div>
  );
}

export function MyTimesheet({
  entries,
  projects,
  from,
  to,
  prev,
  next,
  today,
}: {
  entries: TimesheetEntry[];
  projects: ProjectOption[];
  from: string;
  to: string;
  prev: string;
  next: string | null;
  today: string;
}) {
  const { pending, run } = useErpAction();
  const total = entries.reduce((sum, e) => sum + Number(e.hours), 0);
  const drafts = entries.filter(editable).length;
  const columns: ColumnDef<TimesheetEntry>[] = [
    { accessorKey: "workDate", header: s.date, cell: ({ row }) => formatDate(row.original.workDate) },
    { accessorKey: "projectName", header: s.project, cell: ({ row }) => row.original.projectName || <span className="text-muted-foreground">{s.internal}</span> },
    { accessorKey: "hours", header: s.hours, meta: { align: "right" }, cell: ({ row }) => formatDecimal(Number(row.original.hours), 2) },
    { accessorKey: "description", header: s.description },
    {
      accessorKey: "status",
      header: s.status,
      cell: ({ row: { original: e } }) => (
        <span className="flex flex-col gap-0.5">
          <ErpStatusBadge status={e.status} labels={s.statuses} kinds={statusKinds} />
          {e.rejectReason ? <span className="text-xs text-muted-foreground">{e.rejectReason}</span> : null}
        </span>
      ),
    },
    {
      id: "actions",
      header: "",
      cell: ({ row: { original: e } }) =>
        editable(e) ? (
          <span className="flex justify-end gap-0.5">
            <FormDialog title={s.edit} trigger={<Button size="icon-xs" variant="ghost" aria-label={s.edit}><Pencil className="size-3.5" /></Button>}>
              {(close) => <EntryForm entry={e} projects={projects} today={today} close={close} />}
            </FormDialog>
            <ConfirmAction
              title={s.deleteConfirm}
              success={s.deleted}
              action={() => deleteTimesheetEntryAction(e.id)}
              trigger={<Button size="icon-xs" variant="ghost" aria-label={s.delete}><Trash2 className="size-3.5" /></Button>}
            />
          </span>
        ) : null,
    },
  ];
  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <WeekNav from={from} to={to} prev={prev} next={next} />
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm text-muted-foreground">
            {s.total} <span className="font-medium text-foreground tabular-nums">{formatDecimal(total, 2)}</span> {s.hours.toLowerCase()}
          </span>
          <FormDialog title={s.add} trigger={<Button size="sm" variant="outline"><Plus className="size-4" /> {s.add}</Button>}>
            {(close) => <EntryForm projects={projects} today={today} close={close} />}
          </FormDialog>
          <Button
            size="sm"
            disabled={pending || drafts === 0}
            onClick={() =>
              void run(() => submitTimesheetAction(from, to), {
                onOk: (d) => {
                  const n = d?.count ?? 0;
                  if (n > 0) toast.success(s.submitted(n));
                  else toast.info(s.nothingToSubmit);
                },
              })
            }
          >
            <Send className="size-4" /> {s.submit}
          </Button>
        </div>
      </div>
      <ErpDataTable columns={columns} data={entries} getRowId={(e) => e.id} emptyMessage={s.empty} dense />
    </div>
  );
}

export function TimesheetQueue({ entries, canApprove, queue }: { entries: TimesheetEntry[]; canApprove: boolean; queue: boolean }) {
  const { pending, run } = useErpAction();
  const columns: ColumnDef<TimesheetEntry>[] = [
    { id: "employee", accessorFn: (e) => e.employee.fullName, header: s.employee },
    { accessorKey: "workDate", header: s.date, cell: ({ row }) => formatDate(row.original.workDate) },
    { accessorKey: "projectName", header: s.project, cell: ({ row }) => row.original.projectName || <span className="text-muted-foreground">{s.internal}</span> },
    { accessorKey: "hours", header: s.hours, meta: { align: "right" }, cell: ({ row }) => formatDecimal(Number(row.original.hours), 2) },
    { accessorKey: "description", header: s.description },
    { accessorKey: "status", header: s.status, cell: ({ row }) => <ErpStatusBadge status={row.original.status} labels={s.statuses} kinds={statusKinds} /> },
    {
      id: "actions",
      header: "",
      cell: ({ row: { original: e } }) =>
        canApprove && e.status === "submitted" ? (
          <DecisionButtons onApprove={() => decideTimesheetAction([e.id], true)} onReject={(reason) => decideTimesheetAction([e.id], false, reason)} />
        ) : null,
    },
  ];
  const pendingIds = entries.filter((e) => e.status === "submitted").map((e) => e.id);
  return (
    <div className="flex flex-col gap-3">
      {queue && canApprove && pendingIds.length > 1 ? (
        <div className="flex justify-end">
          <Button size="sm" disabled={pending} onClick={() => void run(() => decideTimesheetAction(pendingIds, true), { success: s.decided(pendingIds.length) })}>
            {s.approveAll}
          </Button>
        </div>
      ) : null}
      <ErpDataTable columns={columns} data={entries} getRowId={(e) => e.id} emptyMessage={queue ? s.queueEmpty : s.empty} exportName="timesheet" />
    </div>
  );
}
