"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { ChevronLeft, ChevronRight, Pencil, Plus, Trash2 } from "lucide-react";
import type { ActionResult } from "@/lib/action-result";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Toolbar } from "@/components/dashboard/layout";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { SelectFilter, type FilterOption } from "@/features/erp/shared/url-filters";
import { ss } from "@/features/erp/shared/strings";
import { useUrlFilters } from "@/features/erp/shared/url-state";
import { formatDate } from "@/lib/format";
import { cn } from "@/lib/utils";
import { deleteHolidayAction, deleteShiftAction, saveHolidayAction, saveShiftAction } from "../actions";
import { weekdayDate } from "../format";
import { th } from "../strings";
import type { Holiday, Shift, TeamSchedule } from "../types";
import { ConfirmAction, Fld, FormDialog, FormFooter, SimpleSelect, useFieldErrors } from "./ui";

function Toggle({ id, label, value, onChange }: { id: string; label: string; value: boolean; onChange: (v: boolean) => void }) {
  return (
    <div className="flex items-center gap-2">
      <Switch id={id} checked={value} onCheckedChange={onChange} />
      <Label htmlFor={id}>{label}</Label>
    </div>
  );
}

function RowButtons({
  canWrite,
  edit,
  onDelete,
}: {
  canWrite: boolean;
  edit: React.ReactNode;
  onDelete: () => Promise<ActionResult<unknown>>;
}) {
  if (!canWrite) return null;
  return (
    <div className="flex justify-end gap-1">
      {edit}
      <ConfirmAction
        title={th.confirmDelete}
        confirmLabel={th.delete}
        success={th.deleted}
        action={onDelete}
        trigger={
          <Button size="icon-xs" variant="ghost" aria-label={th.delete}>
            <Trash2 />
          </Button>
        }
      />
    </div>
  );
}

const editButton = (
  <Button size="icon-xs" variant="ghost" aria-label={th.edit}>
    <Pencil />
  </Button>
);

// Shifts

export function ShiftsPanel({ shifts, canWrite }: { shifts: Shift[]; canWrite: boolean }) {
  const s = th.shifts;
  const dialog = (shift?: Shift) => (
    <FormDialog
      title={shift ? s.editShift : s.newShift}
      trigger={
        shift ? (
          editButton
        ) : (
          <Button>
            <Plus className="size-4" /> {s.newShift}
          </Button>
        )
      }
    >
      {(close) => <ShiftForm shift={shift} close={close} nextOrder={shifts.length + 1} />}
    </FormDialog>
  );
  const columns: ColumnDef<Shift, unknown>[] = [
    { accessorKey: "name", header: s.name },
    {
      id: "time",
      header: `${s.start} – ${s.end}`,
      cell: ({ row }) => (
        <span className="tabular">
          {row.original.startTime}–{row.original.endTime}
          {row.original.isOvernight ? ` (${s.overnight.toLowerCase()})` : ""}
        </span>
      ),
    },
    { accessorKey: "breakMinutes", header: s.breakMinutes, meta: { align: "right" } },
    { accessorKey: "lateToleranceMinutes", header: s.tolerance, meta: { align: "right" } },
    {
      accessorKey: "isActive",
      header: th.status,
      cell: ({ getValue }) => <ErpStatusBadge status={getValue<boolean>() ? "ACTIVE" : "INACTIVE"} />,
    },
    {
      id: "actions",
      header: "",
      cell: ({ row }) => (
        <RowButtons canWrite={canWrite} edit={dialog(row.original)} onDelete={() => deleteShiftAction(row.original.id)} />
      ),
    },
  ];
  return <ErpDataTable columns={columns} data={shifts} toolbar={canWrite ? <div className="ml-auto">{dialog()}</div> : null} />;
}

function ShiftForm({ shift, close, nextOrder }: { shift?: Shift; close: () => void; nextOrder: number }) {
  const s = th.shifts;
  const [v, setV] = React.useState({
    name: shift?.name ?? "",
    startTime: shift?.startTime ?? "09:00",
    endTime: shift?.endTime ?? "18:00",
    breakMinutes: shift?.breakMinutes ?? 60,
    lateToleranceMinutes: shift?.lateToleranceMinutes ?? 15,
    isOvernight: shift?.isOvernight ?? false,
    isActive: shift?.isActive ?? true,
    sortOrder: shift?.sortOrder ?? nextOrder,
  });
  const { errors, onFail } = useFieldErrors();
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveShiftAction(shift?.id ?? null, v), { success: th.saved, onOk: close, onFail });
      }}
    >
      <Fld label={s.name} htmlFor="name" error={errors?.name}>
        <Input id="name" value={v.name} onChange={(e) => setV({ ...v, name: e.target.value })} />
      </Fld>
      <div className="grid grid-cols-2 gap-4">
        <Fld label={s.start} htmlFor="start" error={errors?.startTime}>
          <Input id="start" type="time" value={v.startTime} onChange={(e) => setV({ ...v, startTime: e.target.value })} />
        </Fld>
        <Fld label={s.end} htmlFor="end" error={errors?.endTime}>
          <Input id="end" type="time" value={v.endTime} onChange={(e) => setV({ ...v, endTime: e.target.value })} />
        </Fld>
        <Fld label={s.breakMinutes} htmlFor="break">
          <Input
            id="break"
            type="number"
            min={0}
            value={v.breakMinutes}
            onChange={(e) => setV({ ...v, breakMinutes: Number(e.target.value) })}
          />
        </Fld>
        <Fld label={s.tolerance} htmlFor="tol">
          <Input
            id="tol"
            type="number"
            min={0}
            value={v.lateToleranceMinutes}
            onChange={(e) => setV({ ...v, lateToleranceMinutes: Number(e.target.value) })}
          />
        </Fld>
      </div>
      <div className="flex flex-wrap gap-6">
        <Toggle id="overnight" label={s.overnight} value={v.isOvernight} onChange={(isOvernight) => setV({ ...v, isOvernight })} />
        <Toggle id="active" label={th.active} value={v.isActive} onChange={(isActive) => setV({ ...v, isActive })} />
      </div>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

// Holidays

/** `year` is the default (current) year; other years go through the `year` URL param. */
export function HolidaysPanel({ holidays, year, canWrite }: { holidays: Holiday[]; year: number; canWrite: boolean }) {
  const h = th.holidays;
  const dialog = (holiday?: Holiday) => (
    <FormDialog
      title={holiday ? h.editHoliday : h.newHoliday}
      trigger={
        holiday ? (
          editButton
        ) : (
          <Button>
            <Plus className="size-4" /> {h.newHoliday}
          </Button>
        )
      }
    >
      {(close) => <HolidayForm holiday={holiday} close={close} />}
    </FormDialog>
  );
  const columns: ColumnDef<Holiday, unknown>[] = [
    {
      accessorKey: "holidayDate",
      header: th.date,
      cell: ({ getValue }) => <span className="whitespace-nowrap tabular">{weekdayDate(getValue<string>())}</span>,
    },
    { accessorKey: "name", header: h.name },
    { accessorKey: "type", header: th.type, cell: ({ getValue }) => h.types[getValue<string>()] ?? getValue<string>() },
    { accessorKey: "deductsLeave", header: h.deductsLeave, cell: ({ getValue }) => (getValue<boolean>() ? th.yes : th.no) },
    {
      accessorKey: "status",
      header: th.status,
      cell: ({ getValue }) => (
        <ErpStatusBadge status={getValue<string>()} labels={h.statuses} kinds={{ aktif: "good", draft: "neutral" }} />
      ),
    },
    {
      id: "actions",
      header: "",
      cell: ({ row }) => (
        <RowButtons canWrite={canWrite} edit={dialog(row.original)} onDelete={() => deleteHolidayAction(row.original.id)} />
      ),
    },
  ];
  const years = [year - 1, year + 1].map((y) => ({ value: String(y), label: String(y) }));
  return (
    <ErpDataTable
      columns={columns}
      data={holidays}
      toolbar={
        <>
          <SelectFilter param="year" options={years} allLabel={String(year)} label={h.year} />
          {canWrite ? <div className="ml-auto">{dialog()}</div> : null}
        </>
      }
    />
  );
}

function HolidayForm({ holiday, close }: { holiday?: Holiday; close: () => void }) {
  const h = th.holidays;
  const [v, setV] = React.useState({
    holidayDate: holiday?.holidayDate ?? "",
    name: holiday?.name ?? "",
    type: (holiday?.type ?? "nasional") as "nasional" | "cuti_bersama" | "perusahaan",
    deductsLeave: holiday?.deductsLeave ?? false,
    status: (holiday?.status ?? "aktif") as "draft" | "aktif",
    note: holiday?.note ?? "",
  });
  const { errors, onFail } = useFieldErrors();
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveHolidayAction(holiday?.id ?? null, v), { success: th.saved, onOk: close, onFail });
      }}
    >
      <div className="grid gap-4 sm:grid-cols-2">
        <Fld label={th.date} error={errors?.holidayDate}>
          <DatePicker value={v.holidayDate} onChange={(holidayDate) => setV({ ...v, holidayDate })} />
        </Fld>
        <Fld label={th.type}>
          <SimpleSelect
            value={v.type}
            onChange={(t) => setV({ ...v, type: t as typeof v.type, deductsLeave: t === "cuti_bersama" })}
            options={Object.entries(h.types).map(([value, label]) => ({ value, label }))}
          />
        </Fld>
      </div>
      <Fld label={h.name} htmlFor="name" error={errors?.name}>
        <Input id="name" value={v.name} onChange={(e) => setV({ ...v, name: e.target.value })} />
      </Fld>
      <Fld label={h.note} htmlFor="note">
        <Input id="note" value={v.note} onChange={(e) => setV({ ...v, note: e.target.value })} />
      </Fld>
      <div className="flex flex-wrap gap-6">
        <Toggle id="deducts" label={h.deductsLeave} value={v.deductsLeave} onChange={(deductsLeave) => setV({ ...v, deductsLeave })} />
        <Toggle
          id="aktif"
          label={h.statuses.aktif}
          value={v.status === "aktif"}
          onChange={(on) => setV({ ...v, status: on ? "aktif" : "draft" })}
        />
      </div>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

// Team schedule

function shiftWeek(day: string, delta: number): string {
  const d = new Date(`${day}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + delta * 7);
  return d.toISOString().slice(0, 10);
}

const DAY_TONE: Record<string, string> = {
  shift: "bg-brand/10",
  libur: "bg-muted text-muted-foreground",
  libur_nasional: "bg-amber-500/10 text-amber-700 dark:text-amber-300",
  cuti: "bg-sky-500/10 text-sky-700 dark:text-sky-300",
  none: "text-muted-foreground",
};

/** Week grid of everyone's shifts (NüHabit shift-tim). */
export function TeamScheduleGrid({ data, departments }: { data: TeamSchedule; departments?: FilterOption[] }) {
  const { set } = useUrlFilters(["week"]);
  const days = data.rows[0]?.days.map((d) => d.date) ?? [];
  const s = th.schedule;
  return (
    <div className="flex flex-col gap-3">
      <Toolbar label={s.week}>
        <Button size="icon" variant="outline" aria-label={ss.prev} onClick={() => set({ week: shiftWeek(data.weekStart, -1) })}>
          <ChevronLeft className="size-4" />
        </Button>
        <span className="text-sm font-medium tabular">
          {s.week} {formatDate(data.weekStart)}
        </span>
        <Button size="icon" variant="outline" aria-label={ss.next} onClick={() => set({ week: shiftWeek(data.weekStart, 1) })}>
          <ChevronRight className="size-4" />
        </Button>
        {departments ? (
          <SelectFilter param="dept" options={departments} allLabel={th.allDepartments} label={th.department} />
        ) : null}
      </Toolbar>
      <Card className="overflow-x-auto py-0 scrollbar-thin">
        <table className="w-full min-w-[720px] text-sm">
          <thead>
            <tr className="border-b text-xs text-muted-foreground">
              <th className="sticky left-0 bg-card px-3 py-2 pl-(--card-pad) text-left font-medium">{th.employee}</th>
              {days.map((d) => (
                <th key={d} className="px-2 py-2 text-left font-medium whitespace-nowrap">
                  {weekdayDate(d)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {data.rows.map((r) => (
              <tr key={r.employee.id} className="border-b last:border-0">
                <td className="sticky left-0 bg-card px-3 py-2 pl-(--card-pad)">
                  <span className="block font-medium whitespace-nowrap">{r.employee.fullName}</span>
                  <span className="text-xs text-muted-foreground">{r.employee.departmentName}</span>
                </td>
                {r.days.map((d) => (
                  <td key={d.date} className="px-1 py-1 last:pr-(--card-pad)">
                    <div className={cn("rounded-lg px-2 py-1.5 text-xs", DAY_TONE[d.status])}>
                      <span className="block truncate font-medium">{d.status === "shift" ? d.shiftName : s.teamStatus[d.status] ?? d.status}</span>
                      {d.status === "shift" ? (
                        <span className="tabular opacity-80">
                          {d.startTime}–{d.endTime}
                        </span>
                      ) : null}
                    </div>
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </Card>
    </div>
  );
}
