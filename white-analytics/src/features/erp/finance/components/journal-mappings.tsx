"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Loader2, Pencil, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { ConfirmAction } from "@/features/erp/shared/confirm-action";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { FormField } from "@/features/erp/shared/form-field";
import { SearchableSelect } from "@/features/erp/shared/searchable-select";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { formatDateTime } from "@/lib/format";
import { deleteJournalMappingAction, saveJournalMappingAction } from "../actions";
import type { JournalEventsResponse, JournalMapping, PostableAccount } from "../types";
import { fs } from "../strings";
import { AccountPicker } from "./account-picker";

const m = fs.accounts.mappings;

type LineState = {
  key: string;
  entrySide: "DEBIT" | "CREDIT";
  lineRole: string;
  accountId: string;
  amountSource: string;
  isRequired: boolean;
};

let seq = 0;
const blankLine = (entrySide: "DEBIT" | "CREDIT", amountSource: string): LineState => ({
  key: `m${++seq}`,
  entrySide,
  lineRole: "",
  accountId: "",
  amountSource,
  isRequired: true,
});

function MappingDialog({
  mapping,
  events,
  accounts,
  onClose,
}: {
  mapping?: JournalMapping;
  events: JournalEventsResponse;
  accounts: PostableAccount[];
  onClose: () => void;
}) {
  const defaultSource = events.amountSources[0] ?? "TOTAL";
  const [form, setForm] = React.useState({
    eventCode: mapping?.eventCode ?? "",
    name: mapping?.name ?? "",
    description: mapping?.description ?? "",
    module: mapping?.module ?? "",
    isActive: mapping?.isActive ?? true,
  });
  const [lines, setLines] = React.useState<LineState[]>(() =>
    mapping
      ? [...mapping.lines]
          .sort((x, y) => x.sortOrder - y.sortOrder)
          .map((l) => ({
            key: l.id,
            entrySide: l.entrySide === "CREDIT" ? "CREDIT" : "DEBIT",
            lineRole: l.lineRole,
            accountId: l.accountId ?? "",
            amountSource: l.amountSource || defaultSource,
            isRequired: l.isRequired,
          }))
      : [blankLine("DEBIT", defaultSource), blankLine("CREDIT", defaultSource)],
  );
  const [errors, setErrors] = React.useState<Record<string, string[]>>({});
  const { pending, run } = useErpAction();
  const patch = (key: string, p: Partial<LineState>) => setLines((ls) => ls.map((l) => (l.key === key ? { ...l, ...p } : l)));

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    if (lines.some((l) => l.isRequired && !l.accountId)) {
      setErrors({ lines: [m.lineAccountMissing] });
      return;
    }
    void run(
      () =>
        saveJournalMappingAction(mapping?.id ?? null, {
          ...form,
          lines: lines.map((l, i) => ({
            entrySide: l.entrySide,
            lineRole: l.lineRole,
            accountId: l.accountId || null,
            amountSource: l.amountSource,
            sortOrder: (i + 1) * 10,
            isRequired: l.isRequired,
          })),
        }),
      { success: fs.saved, onOk: onClose, onFail: (r) => setErrors(r.fieldErrors ?? {}) },
    );
  };

  const cells = {
    side: (l: LineState) => (
      <Select value={l.entrySide} onValueChange={(v) => patch(l.key, { entrySide: v as "DEBIT" | "CREDIT" })}>
        <SelectTrigger size="sm" className="w-full" aria-label={fs.type}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="DEBIT">{fs.debit}</SelectItem>
          <SelectItem value="CREDIT">{fs.credit}</SelectItem>
        </SelectContent>
      </Select>
    ),
    role: (l: LineState) => (
      <Input
        className="h-8"
        value={l.lineRole}
        onChange={(e) => patch(l.key, { lineRole: e.target.value })}
        aria-label={m.lineRole}
        placeholder={m.lineRole}
      />
    ),
    account: (l: LineState) => (
      <AccountPicker accounts={accounts} value={l.accountId} onChange={(v) => patch(l.key, { accountId: v })} clearable />
    ),
    source: (l: LineState) => (
      <Select value={l.amountSource} onValueChange={(v) => patch(l.key, { amountSource: v })}>
        <SelectTrigger size="sm" className="w-full" aria-label={m.amountSource}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {events.amountSources.map((s) => (
            <SelectItem key={s} value={s}>
              {m.amountSources[s] ?? s}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    ),
    required: (l: LineState) => (
      <Checkbox checked={l.isRequired} onCheckedChange={(v) => patch(l.key, { isRequired: v === true })} aria-label={m.requiredLine} />
    ),
    remove: (l: LineState) => (
      <Button type="button" variant="ghost" size="icon-xs" aria-label={fs.delete} onClick={() => setLines((ls) => ls.filter((x) => x.key !== l.key))}>
        <Trash2 />
      </Button>
    ),
  };

  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-4xl">
        <form onSubmit={submit} className="grid gap-4">
          <DialogHeader>
            <DialogTitle>{mapping ? m.editTitle : m.newTitle}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField label={m.event} error={errors.eventCode} required>
              <SearchableSelect
                options={events.events.map((ev) => ({ value: ev.eventCode, label: ev.name, hint: ev.eventCode }))}
                value={form.eventCode}
                onChange={(v) => {
                  const ev = events.events.find((x) => x.eventCode === v);
                  setForm((f) => ({
                    ...f,
                    eventCode: v,
                    name: f.name || ev?.name || "",
                    module: ev?.module ?? f.module,
                    description: f.description || ev?.description || "",
                  }));
                }}
                disabled={!!mapping}
              />
            </FormField>
            <FormField label={fs.name} htmlFor="map-name" error={errors.name} required>
              <Input id="map-name" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </FormField>
          </div>
          <FormField label={fs.description} htmlFor="map-desc">
            <Textarea id="map-desc" rows={2} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
          </FormField>
          <div className="flex items-center gap-2">
            <Switch id="map-active" checked={form.isActive} onCheckedChange={(v) => setForm({ ...form, isActive: v })} />
            <Label htmlFor="map-active">{fs.active}</Label>
            {form.module ? <span className="ml-auto text-xs text-muted-foreground">{`${m.module}: ${form.module}`}</span> : null}
          </div>

          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold">{m.lines}</h3>
            <Button type="button" size="sm" variant="outline" onClick={() => setLines((ls) => [...ls, blankLine("DEBIT", defaultSource)])}>
              <Plus className="size-3.5" /> {m.addLine}
            </Button>
          </div>
          {/* Phones: one card per line; the table needs ~760px. */}
          <ul className="divide-y rounded-lg border md:hidden">
            {lines.map((l) => (
              <li key={l.key} className="grid gap-2 p-3">
                <div className="flex items-center gap-2">
                  <div className="w-28 shrink-0">{cells.side(l)}</div>
                  <div className="min-w-0 flex-1">{cells.role(l)}</div>
                  {cells.remove(l)}
                </div>
                {cells.account(l)}
                <div className="flex items-center gap-3">
                  <div className="min-w-0 flex-1">{cells.source(l)}</div>
                  <label className="flex shrink-0 items-center gap-1.5 text-xs text-muted-foreground">
                    {cells.required(l)}
                    {m.requiredLine}
                  </label>
                </div>
              </li>
            ))}
          </ul>
          <div className="hidden max-h-[45vh] overflow-auto rounded-lg border scrollbar-thin md:block">
            <table className="w-full min-w-[760px] text-sm">
              <thead className="sticky top-0 z-10 bg-card text-xs text-muted-foreground">
                <tr>
                  <th className="w-28 px-2 py-2 text-left font-medium">{fs.type}</th>
                  <th className="w-40 px-2 py-2 text-left font-medium">{m.lineRole}</th>
                  <th className="px-2 py-2 text-left font-medium">{fs.account}</th>
                  <th className="w-36 px-2 py-2 text-left font-medium">{m.amountSource}</th>
                  <th className="w-16 px-2 py-2 text-center font-medium">{m.requiredLine}</th>
                  <th className="w-10" />
                </tr>
              </thead>
              <tbody>
                {lines.map((l) => (
                  <tr key={l.key} className="border-t">
                    <td className="px-2 py-1.5">{cells.side(l)}</td>
                    <td className="px-2 py-1.5">{cells.role(l)}</td>
                    <td className="px-2 py-1.5">{cells.account(l)}</td>
                    <td className="px-2 py-1.5">{cells.source(l)}</td>
                    <td className="px-2 py-1.5 text-center">{cells.required(l)}</td>
                    <td className="px-1 py-1.5">{cells.remove(l)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {errors.lines ? <p className="text-xs text-destructive">{errors.lines[0]}</p> : null}
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={onClose}>
              {fs.cancel}
            </Button>
            <Button type="submit" disabled={pending}>
              {pending ? <Loader2 className="size-4 animate-spin" /> : null}
              {fs.save}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export function JournalMappingsTable({
  mappings,
  events,
  accounts,
  canWrite,
}: {
  mappings: JournalMapping[];
  events: JournalEventsResponse;
  accounts: PostableAccount[];
  canWrite: boolean;
}) {
  const [dialog, setDialog] = React.useState<{ mapping?: JournalMapping } | null>(null);
  const [module, setModule] = React.useState("all");
  const { run } = useErpAction();
  const rows = module === "all" ? mappings : mappings.filter((x) => x.module === module);
  const columns: ColumnDef<JournalMapping, unknown>[] = [
    {
      id: "name",
      accessorFn: (r) => `${r.name} ${r.eventCode}`,
      header: m.event,
      cell: ({ row }) => (
        <span>
          <span className="block font-medium">{row.original.name}</span>
          <span className="font-mono text-xs text-muted-foreground">{row.original.eventCode}</span>
        </span>
      ),
    },
    { accessorKey: "module", header: m.module },
    {
      id: "lines",
      header: m.lines,
      accessorFn: (r) => r.lines.length,
      cell: ({ row }) => (
        <ul className="space-y-0.5 text-xs">
          {row.original.lines.map((l) => (
            <li key={l.id} className="flex gap-1.5">
              <span className="w-10 shrink-0 text-muted-foreground">{l.entrySide === "DEBIT" ? "Dr" : "Cr"}</span>
              <span className={l.accountId ? "" : "text-negative"}>{l.accountId ? `${l.accountCode} ${l.accountName}` : "–"}</span>
              <span className="text-muted-foreground">· {m.amountSources[l.amountSource] ?? l.amountSource}</span>
            </li>
          ))}
        </ul>
      ),
    },
    {
      accessorKey: "isReady",
      header: fs.status,
      cell: ({ row }) =>
        !row.original.isActive ? (
          <StatusBadge kind="neutral">{fs.inactive}</StatusBadge>
        ) : row.original.isReady ? (
          <StatusBadge kind="good">{m.ready}</StatusBadge>
        ) : (
          <StatusBadge kind="warning">{m.notReady}</StatusBadge>
        ),
    },
    { accessorKey: "updatedAt", header: fs.journals.logs.updated, cell: ({ row }) => <span className="text-xs text-muted-foreground">{formatDateTime(row.original.updatedAt)}</span> },
    {
      id: "actions",
      header: fs.actions,
      enableSorting: false,
      meta: { align: "right" },
      cell: ({ row }) =>
        canWrite ? (
          <span className="flex justify-end gap-0.5">
            <Button variant="ghost" size="icon-xs" aria-label={fs.edit} onClick={() => setDialog({ mapping: row.original })}>
              <Pencil />
            </Button>
            <ConfirmAction
              trigger={
                <Button variant="ghost" size="icon-xs" aria-label={fs.delete} className="text-destructive">
                  <Trash2 />
                </Button>
              }
              title={fs.confirmDeleteTitle}
              description={m.deleteDesc(row.original.name)}
              confirmLabel={fs.delete}
              destructive
              onConfirm={async () => (await run(() => deleteJournalMappingAction(row.original.id), { success: fs.deleted })).ok}
            />
          </span>
        ) : null,
    },
  ];
  return (
    <>
      <ErpDataTable
        columns={columns}
        data={rows}
        emptyMessage={m.empty}
        toolbar={
          <>
            <Select value={module} onValueChange={setModule}>
              <SelectTrigger className="min-w-36" aria-label={m.module}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{m.allModules}</SelectItem>
                {events.modules.map((mod) => (
                  <SelectItem key={mod} value={mod}>
                    {mod}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {canWrite ? (
              <Button className="ml-auto" onClick={() => setDialog({})}>
                <Plus className="size-4" /> {m.new}
              </Button>
            ) : null}
          </>
        }
      />
      {dialog ? <MappingDialog mapping={dialog.mapping} events={events} accounts={accounts} onClose={() => setDialog(null)} /> : null}
    </>
  );
}
