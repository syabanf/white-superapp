"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { AlertCircle, Camera, CheckCircle2, Pencil, Plus, ShieldCheck, Trash2 } from "lucide-react";
import { FormGrid } from "@/components/dashboard/layout";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable, type ServerPage } from "@/features/erp/shared/data-table";
import { DownloadButton } from "@/features/erp/shared/download-button";
import { DatePicker, DateRangeFilter } from "@/features/erp/shared/period-picker";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { ResetFilters, SelectFilter, type FilterOption } from "@/features/erp/shared/url-filters";
import { formatDate } from "@/lib/format";
import { deleteAttendanceRecordAction, saveAttendanceRecordAction, validateAttendanceAction } from "../actions";
import { hhmmWib, timeWib, toQuery } from "../format";
import { kinds, th } from "../strings";
import type { AttendanceRecord, DailyRoster, RosterEntry } from "../types";
import { PEOPLE_PARAMS, PeopleFilters } from "./list-filters";
import {
  ConfirmAction,
  EmployeePicker,
  Fld,
  FormDialog,
  FormFooter,
  SimpleSelect,
  UrlDate,
  UrlSwitch,
  useFieldErrors,
  type EmployeeOption,
} from "./ui";

const a = th.attendance;

/** Selfie in a modal so HR keeps its place in the list (NüHabit owner request). */
export function PhotoButton({
  attendanceId,
  kind,
  name,
  time,
  base = "/api/erp/v1/hr/attendance",
}: {
  attendanceId: string;
  kind: "in" | "out";
  name: string;
  time: string;
  base?: string;
}) {
  const label = kind === "in" ? a.photoIn : a.photoOut;
  return (
    <Dialog>
      <Tooltip>
        <TooltipTrigger asChild>
          <DialogTrigger asChild>
            <Button size="icon-xs" variant="ghost" aria-label={label}>
              <Camera />
            </Button>
          </DialogTrigger>
        </TooltipTrigger>
        <TooltipContent>{label}</TooltipContent>
      </Tooltip>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>
            {label} · {name}
          </DialogTitle>
        </DialogHeader>
        {/* eslint-disable-next-line @next/next/no-img-element -- authenticated byte route through the proxy */}
        <img
          src={`${base}/${attendanceId}/photo?kind=${kind}`}
          alt={`${label} ${name}`}
          className="max-h-[60dvh] w-full rounded-lg bg-muted object-contain"
        />
        <p className="text-center text-sm text-muted-foreground tabular">{time}</p>
      </DialogContent>
    </Dialog>
  );
}

function Photos({ r }: { r: AttendanceRecord }) {
  if (!r.hasClockInPhoto && !r.hasClockOutPhoto) return <span className="text-muted-foreground">–</span>;
  return (
    <span className="flex gap-0.5">
      {r.hasClockInPhoto ? (
        <PhotoButton attendanceId={r.id} kind="in" name={r.employee.fullName} time={timeWib(r.clockIn)} />
      ) : null}
      {r.hasClockOutPhoto ? (
        <PhotoButton attendanceId={r.id} kind="out" name={r.employee.fullName} time={timeWib(r.clockOut)} />
      ) : null}
    </span>
  );
}

// Daily roster

export function RosterView({
  roster,
  today,
  departments,
  employees,
  canWrite,
}: {
  roster: DailyRoster;
  today: string;
  departments: FilterOption[];
  employees: EmployeeOption[];
  canWrite: boolean;
}) {
  const [target, setTarget] = React.useState<RecordTarget | null>(null);
  const order = Object.keys(a.rosterStatuses);
  const columns: ColumnDef<RosterEntry, unknown>[] = [
    {
      id: "employee",
      accessorFn: (r) => r.employee.fullName,
      header: th.employee,
      cell: ({ row }) => (
        <span>
          <span className="block font-medium">{row.original.employee.fullName}</span>
          <span className="text-xs text-muted-foreground">{row.original.employee.departmentName}</span>
        </span>
      ),
    },
    {
      id: "shift",
      header: a.shift,
      cell: ({ row }) =>
        row.original.shiftName ? (
          <span className="whitespace-nowrap">
            {row.original.shiftName}{" "}
            <span className="text-xs text-muted-foreground tabular">
              {row.original.shiftStart}–{row.original.shiftEnd}
            </span>
          </span>
        ) : (
          "–"
        ),
    },
    {
      accessorKey: "status",
      header: th.status,
      sortingFn: (x, y) => order.indexOf(x.original.status) - order.indexOf(y.original.status),
      cell: ({ row }) => (
        <span className="flex items-center gap-1.5">
          <ErpStatusBadge status={row.original.status} labels={a.rosterStatuses} kinds={kinds.roster} />
          {row.original.isOverdue ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <AlertCircle className="size-3.5 text-destructive" aria-label={a.overdue} />
              </TooltipTrigger>
              <TooltipContent>{a.overdue}</TooltipContent>
            </Tooltip>
          ) : null}
          {row.original.leaveType ? (
            <span className="text-xs text-muted-foreground">{row.original.leaveType}</span>
          ) : null}
        </span>
      ),
    },
    { id: "in", header: a.clockIn, cell: ({ row }) => <span className="tabular">{timeWib(row.original.clockIn)}</span> },
    { id: "out", header: a.clockOut, cell: ({ row }) => <span className="tabular">{timeWib(row.original.clockOut)}</span> },
    {
      accessorKey: "lateMinutes",
      header: a.late,
      meta: { align: "right" },
      cell: ({ getValue }) => (getValue<number>() ? th.minutes(getValue<number>()) : "–"),
    },
  ];
  if (canWrite)
    columns.push({
      id: "actions",
      header: "",
      cell: ({ row }) => {
        const r = row.original;
        const name = r.employee.fullName;
        if (!r.attendanceId)
          return roster.date > today ? null : (
            <div className="flex justify-end">
              <Button
                size="icon-xs"
                variant="ghost"
                aria-label={a.record}
                onClick={() => setTarget({ id: null, name, values: { ...emptyRecord, employeeId: r.employee.id, date: roster.date } })}
              >
                <Plus />
              </Button>
            </div>
          );
        return (
          <div className="flex justify-end gap-1">
            <Button
              size="icon-xs"
              variant="ghost"
              aria-label={a.editRecord}
              onClick={() =>
                setTarget({
                  id: r.attendanceId,
                  name,
                  values: {
                    employeeId: r.employee.id,
                    date: roster.date,
                    clockIn: hhmmWib(r.clockIn),
                    clockOut: hhmmWib(r.clockOut),
                    status: DERIVED_STATUSES.includes(r.attendanceStatus) ? "" : r.attendanceStatus,
                    notes: r.attendanceNotes,
                  },
                })
              }
            >
              <Pencil />
            </Button>
            <ConfirmAction
              title={a.deleteTitle}
              description={a.deleteBody(name, formatDate(roster.date))}
              confirmLabel={th.delete}
              success={a.deletedOk}
              action={() => deleteAttendanceRecordAction(r.attendanceId)}
              trigger={
                <Button size="icon-xs" variant="ghost" aria-label={th.delete}>
                  <Trash2 />
                </Button>
              }
            />
          </div>
        );
      },
    });
  const summary = order.filter((k) => roster.summary[k]);
  return (
    <div className="flex flex-col gap-3">
      {roster.holidayName ? (
        <p className="rounded-xl bg-brand/10 px-4 py-2 text-sm">{a.holiday(roster.holidayName)}</p>
      ) : null}
      {summary.length ? (
        <div className="flex flex-wrap gap-2">
          {summary.map((k) => (
            <span key={k} className="flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs">
              {a.rosterStatuses[k]} <span className="font-semibold tabular">{roster.summary[k]}</span>
            </span>
          ))}
        </div>
      ) : null}
      <ErpDataTable
        columns={columns}
        data={roster.entries}
        getRowId={(r) => r.employee.id}
        toolbar={
          <>
            <UrlDate fallback={today} />
            <SelectFilter param="dept" options={departments} allLabel={th.allDepartments} label={th.department} />
            {canWrite ? (
              <div className="ml-auto">
                <Button
                  onClick={() => setTarget({ id: null, values: { ...emptyRecord, date: roster.date > today ? today : roster.date } })}
                >
                  <Plus className="size-4" /> {a.record}
                </Button>
              </div>
            ) : null}
          </>
        }
      />
      <FormDialog
        title={target?.id ? a.editRecord : a.record}
        description={target?.name ? `${target.name} · ${formatDate(target.values.date)}` : undefined}
        open={target !== null}
        onOpenChange={(o) => !o && setTarget(null)}
      >
        {(close) => target && <RecordForm target={target} employees={employees} close={close} />}
      </FormDialog>
    </div>
  );
}

// Manual records (HR)

type RecordValues = { employeeId: string; date: string; clockIn: string; clockOut: string; status: string; notes: string };
/** `name` is set when the employee and date are fixed by the row. */
type RecordTarget = { id: string | null; name?: string; values: RecordValues };

const emptyRecord: RecordValues = { employeeId: "", date: "", clockIn: "", clockOut: "", status: "", notes: "" };
/** Statuses the backend derives from the shift; editing leaves them on "auto". */
const DERIVED_STATUSES = ["present", "late"];

function RecordForm({
  target,
  employees,
  close,
}: {
  target: RecordTarget;
  employees: EmployeeOption[];
  close: () => void;
}) {
  const [v, setV] = React.useState(target.values);
  const { errors, onFail } = useFieldErrors();
  const { pending, run } = useErpAction();
  const statusOptions = Object.entries(a.statuses).map(([value, label]) => ({ value, label }));
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveAttendanceRecordAction(target.id, v), {
          success: target.id ? a.updatedOk : a.recordedOk,
          onOk: close,
          onFail,
        });
      }}
    >
      <p className="text-sm text-muted-foreground">{a.recordBody}</p>
      {target.name ? null : (
        <FormGrid cols={2}>
          <Fld label={th.employee} error={errors?.employeeId}>
            <EmployeePicker
              value={v.employeeId}
              onChange={(employeeId) => setV({ ...v, employeeId })}
              options={employees}
              invalid={Boolean(errors?.employeeId)}
            />
          </Fld>
          <Fld label={th.date} error={errors?.date}>
            <DatePicker value={v.date} onChange={(date) => setV({ ...v, date })} invalid={Boolean(errors?.date)} />
          </Fld>
        </FormGrid>
      )}
      <FormGrid cols={2}>
        <Fld label={a.clockIn} htmlFor="clockIn" error={errors?.clockIn}>
          <Input id="clockIn" type="time" value={v.clockIn} onChange={(e) => setV({ ...v, clockIn: e.target.value })} />
        </Fld>
        <Fld label={a.clockOut} htmlFor="clockOut" error={errors?.clockOut}>
          <Input id="clockOut" type="time" value={v.clockOut} onChange={(e) => setV({ ...v, clockOut: e.target.value })} />
        </Fld>
      </FormGrid>
      <Fld label={th.status} error={errors?.status}>
        <SimpleSelect
          value={v.status}
          onChange={(status) => setV({ ...v, status })}
          emptyLabel={a.statusAuto}
          options={statusOptions}
        />
      </Fld>
      <Fld label={th.notes} htmlFor="notes" error={errors?.notes}>
        <Textarea id="notes" rows={2} value={v.notes} onChange={(e) => setV({ ...v, notes: e.target.value })} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

// Attendance list (rekap)

export function AttendanceTable({
  records,
  page,
  canApprove,
  filters,
  departments,
  employees,
}: {
  records: AttendanceRecord[];
  page: ServerPage;
  canApprove: boolean;
  filters: { from?: string; to?: string; status?: string };
  departments: FilterOption[];
  employees: EmployeeOption[];
}) {
  const [target, setTarget] = React.useState<AttendanceRecord | null>(null);
  const columns: ColumnDef<AttendanceRecord, unknown>[] = [
    {
      id: "employee",
      accessorFn: (r) => r.employee.fullName,
      header: th.employee,
      cell: ({ row }) => (
        <span>
          <span className="block font-medium">{row.original.employee.fullName}</span>
          <span className="text-xs text-muted-foreground">{row.original.employee.departmentName}</span>
        </span>
      ),
    },
    { accessorKey: "date", header: th.date, cell: ({ getValue }) => <span className="tabular">{formatDate(getValue<string>())}</span> },
    { accessorKey: "shiftName", header: a.shift, cell: ({ getValue }) => getValue<string>() || "–" },
    { id: "in", header: a.clockIn, cell: ({ row }) => <span className="tabular">{timeWib(row.original.clockIn)}</span> },
    { id: "out", header: a.clockOut, cell: ({ row }) => <span className="tabular">{timeWib(row.original.clockOut)}</span> },
    {
      accessorKey: "workHours",
      header: a.workHours,
      meta: { align: "right" },
      cell: ({ getValue }) => (getValue<number>() ? th.hours(getValue<number>()) : "–"),
    },
    {
      accessorKey: "lateMinutes",
      header: a.late,
      meta: { align: "right" },
      cell: ({ getValue }) => (getValue<number>() ? th.minutes(getValue<number>()) : "–"),
    },
    {
      accessorKey: "status",
      header: th.status,
      cell: ({ getValue }) => <ErpStatusBadge status={getValue<string>()} labels={a.statuses} kinds={kinds.attendance} />,
    },
    { id: "photo", header: a.photo, cell: ({ row }) => <Photos r={row.original} /> },
    {
      id: "validated",
      header: a.validated,
      cell: ({ row }) => {
        const r = row.original;
        if (r.isValidated)
          return (
            <Tooltip>
              <TooltipTrigger asChild>
                <CheckCircle2 className="size-4 text-positive" aria-label={a.validated} />
              </TooltipTrigger>
              <TooltipContent>
                {a.validatedBy(r.validatedByName || "–")}
                {r.validationNotes ? ` · ${r.validationNotes}` : ""}
              </TooltipContent>
            </Tooltip>
          );
        return canApprove ? (
          <Button size="xs" variant="outline" onClick={() => setTarget(r)}>
            <ShieldCheck className="size-3.5" /> {a.validate}
          </Button>
        ) : (
          <span className="text-xs text-muted-foreground">{a.unvalidated}</span>
        );
      },
    },
  ];
  const statusOptions = Object.entries(a.statuses).map(([value, label]) => ({ value, label }));
  return (
    <>
      <ErpDataTable
        columns={columns}
        data={records}
        pagination={page}
        getRowId={(r) => r.id}
        toolbar={
          <>
            <DateRangeFilter />
            <SelectFilter param="status" options={statusOptions} allLabel={th.allStatus} label={th.status} />
            <UrlSwitch param="late" label={a.onlyLate} />
            <UrlSwitch param="unvalidated" label={a.onlyUnvalidated} />
            <PeopleFilters departments={departments} employees={employees} />
            <ResetFilters params={["from", "to", "status", "late", "unvalidated", ...PEOPLE_PARAMS]} />
            <div className="ml-auto">
              <DownloadButton
                href={`/api/erp/v1/hr/attendance:export${toQuery({ startDate: filters.from, endDate: filters.to, status: filters.status })}`}
                fileName="rekap-absensi.xlsx"
              >
                {a.exportXlsx}
              </DownloadButton>
            </div>
          </>
        }
      />
      <FormDialog
        title={a.validateTitle}
        description={target ? `${target.employee.fullName} · ${formatDate(target.date)}` : undefined}
        open={target !== null}
        onOpenChange={(o) => !o && setTarget(null)}
      >
        {(close) => target && <ValidateForm record={target} close={close} statusOptions={statusOptions} />}
      </FormDialog>
    </>
  );
}

function ValidateForm({
  record,
  close,
  statusOptions,
}: {
  record: AttendanceRecord;
  close: () => void;
  statusOptions: { value: string; label: string }[];
}) {
  const [status, setStatus] = React.useState("");
  const [notes, setNotes] = React.useState("");
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => validateAttendanceAction(record.id, status, notes), { success: a.validatedOk, onOk: close });
      }}
    >
      <p className="text-sm text-muted-foreground">{a.validateBody}</p>
      <Fld label={th.status}>
        <SimpleSelect
          value={status}
          onChange={setStatus}
          emptyLabel={`${a.keepStatus} (${a.statuses[record.status] ?? record.status})`}
          options={statusOptions}
        />
      </Fld>
      <Fld label={th.notes} htmlFor="notes">
        <Textarea id="notes" rows={2} value={notes} onChange={(e) => setNotes(e.target.value)} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} submitLabel={a.validate} />
    </form>
  );
}
