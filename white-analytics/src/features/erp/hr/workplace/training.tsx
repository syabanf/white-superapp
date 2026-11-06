"use client";

import * as React from "react";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { Pencil, Plus, Trash2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { Money } from "@/features/erp/shared/money-display";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { formatDate } from "@/lib/format";
import { ConfirmAction, EmployeePicker, Fld, FormDialog, FormFooter, SimpleSelect, type EmployeeOption, type FieldErrors } from "../components/ui";
import { deleteTrainingAction, recordTrainingResultAction, saveTrainingAction, setTrainingParticipantsAction } from "./actions";
import { tw } from "./strings";
import type { Training, TrainingParticipant } from "./types";

const s = tw.training;
const statusKinds = { planned: "info", done: "good", cancelled: "neutral" } as const;
const attendanceKinds = { registered: "neutral", attended: "good", absent: "critical" } as const;
const opts = (r: Record<string, string>) => Object.entries(r).map(([value, label]) => ({ value, label }));

function TrainingForm({ training, today, close }: { training?: Training; today: string; close: () => void }) {
  const [v, setV] = React.useState({
    id: training?.id ?? "",
    title: training?.title ?? "",
    provider: training?.provider ?? "",
    startDate: training?.startDate ?? today,
    endDate: training?.endDate ?? today,
    location: training?.location ?? "",
    cost: training?.cost ?? "",
    status: training?.status ?? "planned",
    notes: training?.notes ?? "",
  });
  const [errors, setErrors] = React.useState<FieldErrors>();
  const { pending, run } = useErpAction();
  const set = (p: Partial<typeof v>) => setV((x) => ({ ...x, ...p }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveTrainingAction(v), { success: s.saved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={s.titleField} htmlFor="tr-title" error={errors?.title} className="sm:col-span-2">
        <Input id="tr-title" value={v.title} onChange={(e) => set({ title: e.target.value })} required />
      </Fld>
      <Fld label={s.provider} htmlFor="tr-provider">
        <Input id="tr-provider" value={v.provider} onChange={(e) => set({ provider: e.target.value })} />
      </Fld>
      <Fld label={s.location} htmlFor="tr-location">
        <Input id="tr-location" value={v.location} onChange={(e) => set({ location: e.target.value })} />
      </Fld>
      <Fld label={s.start} htmlFor="tr-start" error={errors?.startDate}>
        <DatePicker id="tr-start" value={v.startDate} onChange={(x) => set({ startDate: x })} />
      </Fld>
      <Fld label={s.end} htmlFor="tr-end" error={errors?.endDate}>
        <DatePicker id="tr-end" value={v.endDate} onChange={(x) => set({ endDate: x })} />
      </Fld>
      <Fld label={s.cost} htmlFor="tr-cost" error={errors?.cost}>
        <MoneyInput id="tr-cost" value={v.cost} onChange={(x) => set({ cost: x })} />
      </Fld>
      <Fld label={s.status} htmlFor="tr-status">
        <SimpleSelect id="tr-status" value={v.status} onChange={(x) => set({ status: x as Training["status"] })} options={opts(s.statuses)} />
      </Fld>
      <Fld label={s.notes} htmlFor="tr-notes" className="sm:col-span-2">
        <Input id="tr-notes" value={v.notes} onChange={(e) => set({ notes: e.target.value })} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

export function TrainingsAdmin({ trainings, canWrite, today }: { trainings: Training[]; canWrite: boolean; today: string }) {
  const columns: ColumnDef<Training>[] = [
    {
      accessorKey: "title",
      header: s.titleField,
      cell: ({ row: { original: t } }) => (
        <Link href={`/erp/hr/training/${t.id}`} className="hover:underline">
          {t.title}
          <span className="block text-xs text-muted-foreground">{t.provider}</span>
        </Link>
      ),
    },
    { accessorKey: "startDate", header: s.start, cell: ({ row: { original: t } }) => `${formatDate(t.startDate)} – ${formatDate(t.endDate)}` },
    { accessorKey: "participantCount", header: s.participants, meta: { align: "right" } },
    { accessorKey: "cost", header: s.cost, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.cost} muteZero /> },
    { accessorKey: "status", header: s.status, cell: ({ row }) => <ErpStatusBadge status={row.original.status} labels={s.statuses} kinds={statusKinds} /> },
    {
      id: "actions",
      header: "",
      cell: ({ row: { original: t } }) =>
        canWrite ? (
          <span className="flex justify-end gap-0.5">
            <FormDialog title={s.edit} wide trigger={<Button size="icon-xs" variant="ghost" aria-label={s.edit}><Pencil className="size-3.5" /></Button>}>
              {(close) => <TrainingForm training={t} today={today} close={close} />}
            </FormDialog>
            <ConfirmAction
              title={s.deleteConfirm}
              success={s.deleted}
              action={() => deleteTrainingAction(t.id)}
              trigger={<Button size="icon-xs" variant="ghost" aria-label={s.delete}><Trash2 className="size-3.5" /></Button>}
            />
          </span>
        ) : null,
    },
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={trainings}
      getRowId={(t) => t.id}
      emptyMessage={s.empty}
      exportName="pelatihan"
      toolbar={
        canWrite ? (
          <FormDialog title={s.add} wide trigger={<Button size="sm"><Plus className="size-4" /> {s.add}</Button>}>
            {(close) => <TrainingForm today={today} close={close} />}
          </FormDialog>
        ) : undefined
      }
    />
  );
}

function ResultForm({ trainingId, p, close }: { trainingId: string; p: TrainingParticipant; close: () => void }) {
  const [v, setV] = React.useState({ status: p.status, score: p.score, certificate: p.certificate });
  const [errors, setErrors] = React.useState<FieldErrors>();
  const { pending, run } = useErpAction();
  return (
    <form
      className="grid gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => recordTrainingResultAction(trainingId, p.employee.id, v), { success: s.resultSaved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={s.status} htmlFor="rs-status" error={errors?.status}>
        <SimpleSelect id="rs-status" value={v.status} onChange={(x) => setV({ ...v, status: x as TrainingParticipant["status"] })} options={opts(s.attendance)} />
      </Fld>
      <Fld label={s.score} htmlFor="rs-score" error={errors?.score}>
        <Input id="rs-score" inputMode="decimal" value={v.score} onChange={(e) => setV({ ...v, score: e.target.value })} />
      </Fld>
      <Fld label={s.certificate} htmlFor="rs-cert">
        <Input id="rs-cert" value={v.certificate} onChange={(e) => setV({ ...v, certificate: e.target.value })} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} submitLabel={s.saveResult} />
    </form>
  );
}

export function TrainingParticipants({
  training,
  participants,
  employees,
  canWrite,
}: {
  training: Training;
  participants: TrainingParticipant[];
  employees: EmployeeOption[];
  canWrite: boolean;
}) {
  const [adding, setAdding] = React.useState("");
  const { pending, run } = useErpAction();
  const ids = participants.map((p) => p.employee.id);
  const save = (next: string[]) => void run(() => setTrainingParticipantsAction(training.id, next), { success: s.participantsSaved });
  return (
    <Card className="gap-3 p-(--card-pad)">
      <div className="flex flex-wrap items-end justify-between gap-2">
        <h2 className="text-sm font-semibold">
          {s.participants} ({participants.length})
        </h2>
        {canWrite ? (
          <div className="flex w-full max-w-md items-center gap-2">
            <EmployeePicker value={adding} onChange={setAdding} options={employees.filter((e) => !ids.includes(e.id))} />
            <Button
              size="sm"
              disabled={!adding || pending}
              onClick={() => {
                save([...ids, adding]);
                setAdding("");
              }}
            >
              <Plus className="size-4" /> {s.addParticipant}
            </Button>
          </div>
        ) : null}
      </div>
      {participants.length === 0 ? (
        <p className="text-sm text-muted-foreground">{s.noParticipants}</p>
      ) : (
        <ul className="divide-y">
          {participants.map((p) => (
            <li key={p.employee.id} className="flex flex-wrap items-center justify-between gap-2 py-2">
              <div className="min-w-0">
                <p className="truncate font-medium">{p.employee.fullName}</p>
                <p className="text-xs text-muted-foreground">
                  {p.employee.departmentName}
                  {p.score ? ` · ${s.score} ${Number(p.score)}` : ""}
                  {p.certificate ? ` · ${p.certificate}` : ""}
                </p>
              </div>
              <div className="flex items-center gap-1">
                <ErpStatusBadge status={p.status} labels={s.attendance} kinds={attendanceKinds} />
                {canWrite ? (
                  <>
                    <FormDialog title={`${s.saveResult} · ${p.employee.fullName}`} trigger={<Button size="xs" variant="ghost">{s.saveResult}</Button>}>
                      {(close) => <ResultForm trainingId={training.id} p={p} close={close} />}
                    </FormDialog>
                    <Button size="icon-xs" variant="ghost" aria-label={s.remove} disabled={pending} onClick={() => save(ids.filter((x) => x !== p.employee.id))}>
                      <X className="size-3.5" />
                    </Button>
                  </>
                ) : null}
              </div>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
