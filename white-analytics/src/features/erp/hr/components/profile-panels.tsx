"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { CalendarDays, ClipboardList, Eye, FileText, Link2, Pencil, Plus, Trash2, UserMinus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Progress } from "@/components/ui/progress";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { FileUpload } from "@/features/erp/shared/file-upload";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { ErpEmpty } from "@/features/erp/shared/states";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { formatDate, formatDateTime } from "@/lib/format";
import { cn } from "@/lib/utils";
import {
  addOnboardingTaskAction,
  applyOnboardingTemplateAction,
  deleteDocumentAction,
  deleteEmployeeAction,
  deleteOnboardingTaskAction,
  linkEmployeeUserAction,
  setEmployeeScheduleAction,
  startOffboardingAction,
  updateLeaveBalanceAction,
  updateOffboardingAction,
  updateOnboardingTaskAction,
} from "../actions";
import { kinds, th } from "../strings";
import type {
  EmployeeDocument,
  EmployeeProfile,
  EmployeeSchedule,
  LeaveBalance,
  Offboarding,
  OnboardingChecklist,
  Shift,
  UserOption,
} from "../types";
import { ConfirmAction, Fld, FormDialog, FormFooter, SimpleSelect, useFieldErrors } from "./ui";

// Header actions

export function ProfileActions({ profile, users }: { profile: EmployeeProfile; users: UserOption[] }) {
  const router = useRouter();
  const e = profile.employee;
  return (
    <>
      <FormDialog
        title={th.employees.linkTitle}
        description={th.employees.linkBody}
        trigger={
          <Button size="sm" variant="outline">
            <Link2 className="size-4" /> {th.employees.linkedUser}
          </Button>
        }
      >
        {(close) => <LinkUserForm employeeId={e.id} current={e.userId ?? ""} users={users} close={close} />}
      </FormDialog>
      <Button size="sm" variant="outline" asChild>
        <Link href={`/erp/hr/employees/${e.id}/edit`}>
          <Pencil className="size-4" /> {th.edit}
        </Link>
      </Button>
      {profile.isActive ? (
        <ConfirmAction
          title={th.employees.deleteTitle}
          description={th.employees.deleteBody}
          confirmLabel={th.employees.deactivate}
          success={th.employees.deactivated}
          action={async () => {
            const res = await deleteEmployeeAction(e.id);
            if (res.ok) router.push("/erp/hr/employees");
            return res;
          }}
          trigger={
            <Button size="sm" variant="ghost" className="text-destructive" aria-label={th.employees.deactivate}>
              <UserMinus className="size-4" />
            </Button>
          }
        />
      ) : null}
    </>
  );
}

function LinkUserForm({
  employeeId,
  current,
  users,
  close,
}: {
  employeeId: string;
  current: string;
  users: UserOption[];
  close: () => void;
}) {
  const [userId, setUserId] = React.useState(current);
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => linkEmployeeUserAction(employeeId, userId), {
          success: userId ? th.employees.linked : th.employees.unlinked,
          onOk: close,
        });
      }}
    >
      <Fld label={th.employees.linkedUser}>
        <SimpleSelect
          value={userId}
          onChange={setUserId}
          emptyLabel={th.employees.notLinked}
          options={users.map((u) => ({ value: u.id, label: `${u.name} · ${u.email}` }))}
        />
      </Fld>
      <FormFooter onCancel={close} pending={pending} submitLabel={userId ? th.save : th.employees.unlink} />
    </form>
  );
}

// Documents

function fileSize(bytes: string): string {
  const n = Number(bytes);
  if (n >= 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1).replace(".", ",")} MB`;
  return `${Math.max(1, Math.round(n / 1024))} KB`;
}

export function DocumentsPanel({
  employeeId,
  documents,
  canWrite,
}: {
  employeeId: string;
  documents: EmployeeDocument[];
  canWrite: boolean;
}) {
  const router = useRouter();
  const d = th.documents;
  const [docType, setDocType] = React.useState("other");
  return (
    <div className="grid gap-4 lg:grid-cols-[1fr_320px]">
      <Card className="gap-0 py-0">
        {documents.length === 0 ? (
          <div className="p-4">
            <ErpEmpty compact icon={<FileText className="size-5" />} title={d.empty} />
          </div>
        ) : (
          <ul className="divide-y">
            {documents.map((doc) => (
              <li key={doc.id} className="flex items-center gap-3 px-4 py-3">
                <FileText className="size-4 shrink-0 text-muted-foreground" />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium">{doc.name}</p>
                  <p className="text-xs text-muted-foreground">
                    {d.types[doc.docType] ?? doc.docType} · {fileSize(doc.sizeBytes)} · {formatDate(doc.createdAt)}
                  </p>
                </div>
                <Button size="icon-xs" variant="ghost" asChild aria-label={d.file}>
                  <a href={`/api/erp/v1/hr/documents/${doc.id}`} target="_blank" rel="noreferrer">
                    <Eye />
                  </a>
                </Button>
                {canWrite ? (
                  <ConfirmAction
                    title={th.confirmDelete}
                    success={th.deleted}
                    confirmLabel={th.delete}
                    action={() => deleteDocumentAction(doc.id)}
                    trigger={
                      <Button size="icon-xs" variant="ghost" aria-label={th.delete}>
                        <Trash2 />
                      </Button>
                    }
                  />
                ) : null}
              </li>
            ))}
          </ul>
        )}
      </Card>
      {canWrite ? (
        <Card className="gap-3">
          <CardHeader>
            <CardTitle className="text-[15px]">{d.upload}</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <Fld label={d.docType}>
              <SimpleSelect
                value={docType}
                onChange={setDocType}
                options={Object.entries(d.types).map(([value, label]) => ({ value, label }))}
              />
            </Fld>
            <FileUpload
              url={`/api/erp/api/hr/employees/${employeeId}/documents`}
              accept="application/pdf,image/jpeg,image/png,image/webp"
              acceptLabel="PDF, JPG, PNG, WebP"
              maxMb={10}
              multiple={false}
              extraFields={{ docType }}
              onUploaded={() => router.refresh()}
            />
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}

// Schedule

export function SchedulePanel({
  schedule,
  shifts,
  canWrite,
}: {
  schedule: EmployeeSchedule;
  shifts: Shift[];
  canWrite: boolean;
}) {
  const s = th.schedule;
  const byDay = new Map(schedule.current.map((d) => [d.dayOfWeek, d]));
  const shiftById = new Map(shifts.map((x) => [x.id, x]));
  const since = schedule.current[0]?.effectiveFrom;
  return (
    <Card className="gap-3">
      <CardHeader className="flex flex-row items-center justify-between gap-2">
        <div>
          <CardTitle className="text-[15px]">{s.current}</CardTitle>
          {since ? (
            <p className="mt-1 text-xs text-muted-foreground">
              {s.effectiveFrom} {formatDate(since)}
            </p>
          ) : null}
        </div>
        {canWrite ? (
          <FormDialog
            title={s.setTitle}
            description={s.setBody}
            trigger={
              <Button size="sm" variant="outline">
                <CalendarDays className="size-4" /> {s.setSchedule}
              </Button>
            }
          >
            {(close) => <ScheduleForm schedule={schedule} shifts={shifts} close={close} />}
          </FormDialog>
        ) : null}
      </CardHeader>
      <CardContent>
        {schedule.current.length === 0 ? (
          <p className="text-sm text-muted-foreground">{s.empty}</p>
        ) : (
          <ul className="grid gap-2 sm:grid-cols-7">
            {s.days.map((label, i) => {
              const d = byDay.get(i + 1);
              const shift = d?.shiftId ? shiftById.get(d.shiftId) : undefined;
              return (
                <li
                  key={label}
                  className={cn(
                    "flex items-center justify-between gap-2 rounded-xl border px-3 py-2 sm:flex-col sm:items-start",
                    !d?.shiftId && "bg-muted/40 text-muted-foreground",
                  )}
                >
                  <span className="text-xs font-medium">{label}</span>
                  <span className="text-sm">{d?.shiftName || s.dayOff}</span>
                  {shift ? (
                    <span className="text-xs text-muted-foreground tabular">
                      {shift.startTime}–{shift.endTime}
                    </span>
                  ) : null}
                </li>
              );
            })}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}

function ScheduleForm({ schedule, shifts, close }: { schedule: EmployeeSchedule; shifts: Shift[]; close: () => void }) {
  const [from, setFrom] = React.useState("");
  const [days, setDays] = React.useState(() =>
    [1, 2, 3, 4, 5, 6, 7].map((dow) => schedule.current.find((d) => d.dayOfWeek === dow)?.shiftId ?? ""),
  );
  const { pending, run } = useErpAction();
  const options = shifts.filter((x) => x.isActive).map((x) => ({ value: x.id, label: `${x.name} (${x.startTime}–${x.endTime})` }));
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(
          () =>
            setEmployeeScheduleAction(
              schedule.employeeId,
              from,
              days.map((shiftId, i) => ({ dayOfWeek: i + 1, shiftId, shiftName: "", effectiveFrom: from })),
            ),
          { success: th.schedule.savedOk, onOk: close },
        );
      }}
    >
      <Fld label={th.schedule.effectiveFrom}>
        <DatePicker value={from} onChange={setFrom} />
      </Fld>
      <div className="grid gap-3 sm:grid-cols-2">
        {th.schedule.days.map((label, i) => (
          <Fld key={label} label={label}>
            <SimpleSelect
              value={days[i] ?? ""}
              onChange={(v) => setDays((list) => list.map((x, j) => (j === i ? v : x)))}
              emptyLabel={th.schedule.dayOff}
              options={options}
            />
          </Fld>
        ))}
      </div>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

// Leave balance

export function BalancePanel({ balance, canWrite }: { balance: LeaveBalance; canWrite: boolean }) {
  const b = th.balance;
  const pct = balance.annualLeaveTotal ? (balance.annualLeaveUsed / balance.annualLeaveTotal) * 100 : 0;
  return (
    <Card className="gap-4">
      <CardHeader className="flex flex-row items-center justify-between gap-2">
        <CardTitle className="text-[15px]">
          {b.title} {balance.year}
        </CardTitle>
        {canWrite ? (
          <FormDialog
            title={b.adjust}
            description={b.adjustBody}
            trigger={
              <Button size="sm" variant="outline">
                <Pencil className="size-4" /> {b.adjust}
              </Button>
            }
          >
            {(close) => <BalanceForm balance={balance} close={close} />}
          </FormDialog>
        ) : null}
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div>
          <div className="flex items-baseline justify-between">
            <span className="text-3xl font-bold tabular">{balance.annualLeaveRemaining}</span>
            <span className="text-sm text-muted-foreground">
              {b.annualUsed} {balance.annualLeaveUsed} / {balance.annualLeaveTotal}
            </span>
          </div>
          <p className="text-xs text-muted-foreground">{b.annualRemaining}</p>
          <Progress value={pct} className="mt-2" />
        </div>
        <dl className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          {[
            [b.sick, balance.sickLeaveUsed],
            [b.unpaid, balance.unpaidLeaveUsed],
            [b.other, balance.otherLeaveUsed],
            [b.pending, balance.pendingDays],
          ].map(([label, value]) => (
            <div key={label} className="rounded-xl border px-3 py-2">
              <dt className="text-xs text-muted-foreground">{label}</dt>
              <dd className="text-lg font-semibold tabular">{th.days(Number(value))}</dd>
            </div>
          ))}
        </dl>
      </CardContent>
    </Card>
  );
}

function BalanceForm({ balance, close }: { balance: LeaveBalance; close: () => void }) {
  const [total, setTotal] = React.useState(balance.annualLeaveTotal);
  const [used, setUsed] = React.useState(balance.annualLeaveUsed);
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => updateLeaveBalanceAction(balance.employeeId, balance.year, total, used), {
          success: th.balance.adjusted,
          onOk: close,
        });
      }}
    >
      <div className="grid grid-cols-2 gap-4">
        <Fld label={th.balance.annualTotal} htmlFor="total">
          <Input id="total" type="number" min={0} value={total} onChange={(e) => setTotal(Number(e.target.value))} />
        </Fld>
        <Fld label={th.balance.annualUsed} htmlFor="used">
          <Input id="used" type="number" min={0} value={used} onChange={(e) => setUsed(Number(e.target.value))} />
        </Fld>
      </div>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

// Onboarding and offboarding

export function LifecyclePanel({
  employeeId,
  onboarding,
  offboarding,
  canWrite,
}: {
  employeeId: string;
  onboarding: OnboardingChecklist;
  offboarding: Offboarding | null;
  canWrite: boolean;
}) {
  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <OnboardingCard employeeId={employeeId} list={onboarding} canWrite={canWrite} />
      <OffboardingCard employeeId={employeeId} off={offboarding} canWrite={canWrite} />
    </div>
  );
}

function OnboardingCard({
  employeeId,
  list,
  canWrite,
}: {
  employeeId: string;
  list: OnboardingChecklist;
  canWrite: boolean;
}) {
  const l = th.lifecycle;
  const { pending, run } = useErpAction();
  return (
    <Card className="gap-4">
      <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2">
        <div>
          <CardTitle className="text-[15px]">{l.onboarding}</CardTitle>
          <p className="mt-1 text-xs text-muted-foreground">{l.progress(list.completed, list.total)}</p>
        </div>
        {canWrite ? (
          <div className="flex gap-2">
            <Button
              size="xs"
              variant="outline"
              disabled={pending}
              onClick={() => void run(() => applyOnboardingTemplateAction(employeeId), { success: th.saved })}
            >
              <ClipboardList className="size-3.5" /> {l.applyTemplate}
            </Button>
            <FormDialog
              title={l.addTask}
              trigger={
                <Button size="xs">
                  <Plus className="size-3.5" /> {l.addTask}
                </Button>
              }
            >
              {(close) => <TaskForm employeeId={employeeId} close={close} />}
            </FormDialog>
          </div>
        ) : null}
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {list.total ? <Progress value={(list.completed / list.total) * 100} /> : null}
        {list.tasks.length === 0 ? (
          <p className="text-sm text-muted-foreground">{l.empty}</p>
        ) : (
          <ul className="flex flex-col divide-y">
            {list.tasks.map((t) => (
              <li key={t.id} className="flex items-start gap-3 py-2.5">
                <Checkbox
                  checked={t.isCompleted}
                  disabled={!canWrite || pending}
                  aria-label={t.taskName}
                  className="mt-0.5"
                  onCheckedChange={(v) => void run(() => updateOnboardingTaskAction(t.id, { isCompleted: v === true }))}
                />
                <div className="min-w-0 flex-1">
                  <p className={cn("text-sm", t.isCompleted && "text-muted-foreground line-through")}>{t.taskName}</p>
                  <p className="text-xs text-muted-foreground">
                    {l.categories[t.category] ?? t.category} · {l.priorities[t.priority] ?? t.priority}
                    {t.dueDate ? ` · ${l.dueDate} ${formatDate(t.dueDate)}` : ""}
                    {t.isCompleted && t.completedByName ? ` · ${l.completedBy(t.completedByName)}` : ""}
                  </p>
                </div>
                {canWrite ? (
                  <Button
                    size="icon-xs"
                    variant="ghost"
                    aria-label={th.delete}
                    disabled={pending}
                    onClick={() => void run(() => deleteOnboardingTaskAction(t.id), { success: th.deleted })}
                  >
                    <Trash2 />
                  </Button>
                ) : null}
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}

function TaskForm({ employeeId, close }: { employeeId: string; close: () => void }) {
  const l = th.lifecycle;
  const [v, setV] = React.useState({ taskName: "", category: "general", priority: 2, dueDate: "" });
  const { errors, onFail } = useFieldErrors();
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => addOnboardingTaskAction(employeeId, v), { success: th.saved, onOk: close, onFail });
      }}
    >
      <Fld label={l.taskName} htmlFor="taskName" error={errors?.taskName}>
        <Input id="taskName" value={v.taskName} onChange={(e) => setV({ ...v, taskName: e.target.value })} />
      </Fld>
      <div className="grid gap-4 sm:grid-cols-3">
        <Fld label={l.category}>
          <SimpleSelect
            value={v.category}
            onChange={(category) => setV({ ...v, category })}
            options={Object.entries(l.categories).map(([value, label]) => ({ value, label }))}
          />
        </Fld>
        <Fld label={l.priority}>
          <SimpleSelect
            value={String(v.priority)}
            onChange={(p) => setV({ ...v, priority: Number(p) })}
            options={Object.entries(l.priorities).map(([value, label]) => ({ value, label }))}
          />
        </Fld>
        <Fld label={l.dueDate}>
          <DatePicker value={v.dueDate} onChange={(dueDate) => setV({ ...v, dueDate })} clearable />
        </Fld>
      </div>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

const OFF_FLOW = ["submitted", "notice_period", "exit_interview", "completed"];

function OffboardingCard({
  employeeId,
  off,
  canWrite,
}: {
  employeeId: string;
  off: Offboarding | null;
  canWrite: boolean;
}) {
  const l = th.lifecycle;
  const { pending, run } = useErpAction();
  const [notes, setNotes] = React.useState(off?.exitInterviewNotes ?? "");
  if (!off) {
    return (
      <Card className="gap-4">
        <CardHeader>
          <CardTitle className="text-[15px]">{l.offboarding}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col items-start gap-3">
          <p className="text-sm text-muted-foreground">{l.noOffboarding}</p>
          {canWrite ? (
            <FormDialog
              title={l.startOffboarding}
              trigger={
                <Button size="sm" variant="outline">
                  <UserMinus className="size-4" /> {l.startOffboarding}
                </Button>
              }
            >
              {(close) => <StartOffboardingForm employeeId={employeeId} close={close} />}
            </FormDialog>
          ) : null}
        </CardContent>
      </Card>
    );
  }
  const done = off.status === "completed";
  const next = OFF_FLOW[OFF_FLOW.indexOf(off.status) + 1];
  const clearances: [string, boolean][] = [
    ["hr", off.clearanceHr],
    ["it", off.clearanceIt],
    ["finance", off.clearanceFinance],
    ["manager", off.clearanceManager],
  ];
  return (
    <Card className="gap-4">
      <CardHeader className="flex flex-row items-center justify-between gap-2">
        <CardTitle className="text-[15px]">{l.offboarding}</CardTitle>
        <ErpStatusBadge status={off.status} labels={l.statuses} kinds={kinds.offboarding} />
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <dl className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <dt className="text-xs text-muted-foreground">{l.resignationType}</dt>
            <dd>{l.resignationTypes[off.resignationType] ?? off.resignationType}</dd>
          </div>
          <div>
            <dt className="text-xs text-muted-foreground">{l.lastWorkingDay}</dt>
            <dd className="tabular">{formatDate(off.lastWorkingDay)}</dd>
          </div>
          <div className="col-span-2">
            <dt className="text-xs text-muted-foreground">{th.reason}</dt>
            <dd>{off.reason || "–"}</dd>
          </div>
        </dl>
        <div>
          <p className="mb-2 text-xs font-medium text-muted-foreground">{l.clearance}</p>
          <div className="grid grid-cols-2 gap-2">
            {clearances.map(([key, cleared]) => (
              <label key={key} className="flex items-center gap-2 rounded-lg border px-3 py-2 text-sm">
                <Checkbox
                  checked={cleared}
                  disabled={!canWrite || done || pending}
                  onCheckedChange={(v) =>
                    void run(() => updateOffboardingAction(employeeId, { clearanceType: key, cleared: v === true }))
                  }
                />
                {l.clearances[key]}
              </label>
            ))}
          </div>
        </div>
        <Fld label={l.exitInterview} htmlFor="exit">
          <Textarea
            id="exit"
            rows={3}
            value={notes}
            disabled={!canWrite || done}
            onChange={(e) => setNotes(e.target.value)}
            onBlur={() => {
              if (notes !== off.exitInterviewNotes)
                void run(() => updateOffboardingAction(employeeId, { exitInterviewNotes: notes }), { success: th.saved });
            }}
          />
        </Fld>
        {canWrite && next ? (
          next === "completed" ? (
            <ConfirmAction
              title={l.complete}
              description={l.completeBody}
              confirmLabel={l.complete}
              success={th.saved}
              action={() => updateOffboardingAction(employeeId, { status: "completed" })}
              trigger={<Button size="sm">{l.complete}</Button>}
            />
          ) : (
            <Button
              size="sm"
              variant="outline"
              disabled={pending}
              onClick={() => void run(() => updateOffboardingAction(employeeId, { status: next }), { success: th.saved })}
            >
              {l.nextStep}: {l.statuses[next]}
            </Button>
          )
        ) : null}
        {off.completedAt ? <p className="text-xs text-muted-foreground">{formatDateTime(off.completedAt)}</p> : null}
      </CardContent>
    </Card>
  );
}

function StartOffboardingForm({ employeeId, close }: { employeeId: string; close: () => void }) {
  const l = th.lifecycle;
  const [v, setV] = React.useState({ resignationType: "voluntary", resignationDate: "", lastWorkingDay: "", reason: "" });
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => startOffboardingAction(employeeId, v), { success: th.saved, onOk: close });
      }}
    >
      <Fld label={l.resignationType}>
        <SimpleSelect
          value={v.resignationType}
          onChange={(resignationType) => setV({ ...v, resignationType })}
          options={Object.entries(l.resignationTypes).map(([value, label]) => ({ value, label }))}
        />
      </Fld>
      <div className="grid gap-4 sm:grid-cols-2">
        <Fld label={l.resignationDate}>
          <DatePicker value={v.resignationDate} onChange={(resignationDate) => setV({ ...v, resignationDate })} />
        </Fld>
        <Fld label={l.lastWorkingDay}>
          <DatePicker value={v.lastWorkingDay} onChange={(lastWorkingDay) => setV({ ...v, lastWorkingDay })} />
        </Fld>
      </div>
      <Fld label={th.reason} htmlFor="reason">
        <Textarea id="reason" rows={3} value={v.reason} onChange={(e) => setV({ ...v, reason: e.target.value })} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} submitLabel={l.startOffboarding} />
    </form>
  );
}
