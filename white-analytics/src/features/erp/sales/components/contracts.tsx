"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { useRouter } from "next/navigation";
import { Ban, CheckCircle2, Eye, FilePlus2, Flag, Loader2, Pencil, Plus, Send, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { DownloadButton } from "@/features/erp/shared/download-button";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { SelectFilter } from "@/features/erp/shared/url-filters";
import { formatDate } from "@/lib/format";
import {
  deleteContractTemplateAction,
  generateContractAction,
  saveContractTemplateAction,
  sendContractAction,
  setContractStatusAction,
  updateContractAction,
} from "../actions";
import { signPagePath, type SalesCaps } from "../lib";
import { CONTRACT_STATUS, CONTRACT_TYPE, lbl, opts, ss } from "../strings";
import type { Contract, ContractTemplate, Quotation } from "../types";
import { Markdown } from "./markdown";
import {
  ActionButton,
  ConfirmAction,
  CopyLinkButton,
  Fld,
  FormDialog,
  FormFooter,
  type Option,
  Picker,
  ReasonDialog,
  SalesBadge,
  SimpleSelect,
} from "./ui";

const s = ss.contracts;

export function ContractsTable({
  contracts,
  caps,
  quotations,
  templates,
  generateFor,
}: {
  contracts: Contract[];
  caps: SalesCaps;
  quotations: Quotation[];
  templates: ContractTemplate[];
  /** opens the generate dialog for this quotation (from ?generate=) */
  generateFor?: string;
}) {
  const router = useRouter();
  const columns: ColumnDef<Contract, unknown>[] = [
    {
      accessorKey: "contractNumber",
      header: s.number,
      cell: ({ row }) => (
        <div className="min-w-0">
          <div className="font-medium tabular">{row.original.contractNumber}</div>
          <div className="truncate text-xs text-muted-foreground">{row.original.title}</div>
        </div>
      ),
    },
    { accessorKey: "clientName", header: s.client },
    { accessorKey: "type", header: s.type, cell: ({ row }) => lbl(CONTRACT_TYPE, row.original.type) },
    {
      id: "period",
      header: s.period,
      accessorFn: (c) => c.startDate,
      cell: ({ row }) =>
        row.original.startDate ? `${formatDate(row.original.startDate)} – ${formatDate(row.original.endDate || null)}` : ss.dash,
    },
    { accessorKey: "status", header: s.status, cell: ({ row }) => <SalesBadge value={row.original.status} map={CONTRACT_STATUS} /> },
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={contracts}
      getRowId={(c) => c.id}
      onRowClick={(row) => router.push(`/erp/sales/contracts/${row.original.id}`)}
      exportName="kontrak"
      toolbar={
        <>
          <SelectFilter param="status" label={s.status} allLabel={s.allStatus} options={opts(CONTRACT_STATUS)} />
          {caps.write ? <GenerateContractDialog quotations={quotations} templates={templates} initialQuotation={generateFor} /> : null}
        </>
      }
    />
  );
}

export function GenerateContractDialog({
  quotations,
  templates,
  initialQuotation,
  size,
}: {
  quotations: Quotation[];
  templates: ContractTemplate[];
  initialQuotation?: string;
  /** "sm" inside a card header */
  size?: "sm";
}) {
  const [open, setOpen] = React.useState(!!initialQuotation);
  return (
    <FormDialog
      open={open}
      onOpenChange={setOpen}
      title={s.generateTitle}
      trigger={
        <Button size={size}>
          <FilePlus2 /> {s.generate}
        </Button>
      }
    >
      {(close) => <GenerateForm quotations={quotations} templates={templates} initial={initialQuotation} close={close} />}
    </FormDialog>
  );
}

function GenerateForm({
  quotations,
  templates,
  initial,
  close,
}: {
  quotations: Quotation[];
  templates: ContractTemplate[];
  initial?: string;
  close: () => void;
}) {
  const router = useRouter();
  const active = templates.filter((t) => t.isActive);
  const [quotationId, setQuotationId] = React.useState(initial ?? "");
  const [templateId, setTemplateId] = React.useState(active[0]?.id ?? "");
  const [title, setTitle] = React.useState("");
  const [startDate, setStartDate] = React.useState("");
  const [endDate, setEndDate] = React.useState("");
  const { pending, run } = useErpAction();
  const qOptions: Option[] = quotations.map((q) => ({ value: q.id, label: `${q.quoteNumber} · ${q.clientName}` }));
  if (!qOptions.length) return <p className="text-sm text-muted-foreground">{s.noAccepted}</p>;
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        if (!quotationId) return;
        void run(() => generateContractAction(quotationId, { templateId, title, startDate, endDate }), {
          success: s.generated,
          onOk: (c) => {
            close();
            router.push(`/erp/sales/contracts/${c.id}`);
          },
        });
      }}
    >
      <Fld label={`${s.quotation} *`} htmlFor="gc-q" className="sm:col-span-2">
        <Picker id="gc-q" value={quotationId} onChange={setQuotationId} options={qOptions} />
      </Fld>
      <Fld label={`${s.template} *`} htmlFor="gc-t" className="sm:col-span-2">
        <SimpleSelect
          id="gc-t"
          value={templateId}
          onChange={setTemplateId}
          options={active.map((t) => ({ value: t.id, label: `${t.name} · ${lbl(CONTRACT_TYPE, t.type)}` }))}
        />
      </Fld>
      <Fld label={s.contractTitle} htmlFor="gc-title" className="sm:col-span-2">
        <Input id="gc-title" value={title} onChange={(e) => setTitle(e.target.value)} />
      </Fld>
      <Fld label={s.start} htmlFor="gc-start">
        <DatePicker id="gc-start" value={startDate} onChange={setStartDate} clearable />
      </Fld>
      <Fld label={s.end} htmlFor="gc-end">
        <DatePicker id="gc-end" value={endDate} onChange={setEndDate} clearable />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} submitLabel={s.generate} disabled={!quotationId || !templateId} />
      </div>
    </form>
  );
}

/** Send, activate, end, cancel and PDF for one contract. */
export function ContractActions({ contract: c, caps }: { contract: Contract; caps: SalesCaps }) {
  const pdf = (
    <DownloadButton href={`/api/erp/api/sales/contracts/${c.id}/pdf`} fileName={`${c.contractNumber}.pdf`} size="default">
      {s.pdf}
    </DownloadButton>
  );
  if (!caps.write) return pdf;
  const open = ["draft", "sent", "signed", "active"].includes(c.status);
  return (
    <>
      {open ? (
        <ReasonDialog
          title={s.cancel}
          label={s.cancelReason}
          submitLabel={s.cancel}
          success={s.statusChanged}
          onSubmit={(r) => setContractStatusAction(c.id, "cancelled", r)}
          trigger={
            <Button variant="ghost">
              <Ban /> {s.cancel}
            </Button>
          }
        />
      ) : null}
      {pdf}
      {c.status === "sent" ? <CopyLinkButton path={signPagePath(c)} label={s.signLink} size="default" /> : null}
      {c.status === "draft" ? (
        <ActionButton action={() => sendContractAction(c.id)} success={s.sentOk}>
          <Send /> {s.send}
        </ActionButton>
      ) : null}
      {c.status === "signed" ? (
        <ActionButton action={() => setContractStatusAction(c.id, "active")} success={s.statusChanged}>
          <CheckCircle2 /> {s.activate}
        </ActionButton>
      ) : null}
      {c.status === "active" ? (
        <ActionButton variant="outline" action={() => setContractStatusAction(c.id, "ended")} success={s.statusChanged}>
          <Flag /> {s.end_}
        </ActionButton>
      ) : null}
    </>
  );
}

/** Draft editor: title, period and Markdown body with a live preview. */
export function ContractEditor({ contract }: { contract: Contract }) {
  const [title, setTitle] = React.useState(contract.title);
  const [body, setBody] = React.useState(contract.body);
  const [startDate, setStartDate] = React.useState(contract.startDate);
  const [endDate, setEndDate] = React.useState(contract.endDate);
  const [view, setView] = React.useState<"edit" | "preview">("edit");
  const { pending, run } = useErpAction();
  return (
    <Card className="gap-4 p-(--card-pad)">
      <div className="grid gap-4 sm:grid-cols-[1fr_180px_180px]">
        <Fld label={s.contractTitle} htmlFor="ce-title">
          <Input id="ce-title" value={title} onChange={(e) => setTitle(e.target.value)} />
        </Fld>
        <Fld label={s.start} htmlFor="ce-start">
          <DatePicker id="ce-start" value={startDate} onChange={setStartDate} clearable />
        </Fld>
        <Fld label={s.end} htmlFor="ce-end">
          <DatePicker id="ce-end" value={endDate} onChange={setEndDate} clearable />
        </Fld>
      </div>
      <div className="flex items-center justify-between gap-2">
        <span className="text-sm font-medium">{s.body}</span>
        <ToggleGroup type="single" value={view} onValueChange={(v) => v && setView(v as "edit" | "preview")} variant="outline" size="sm">
          <ToggleGroupItem value="edit" aria-label={ss.edit}>
            <Pencil className="size-3.5" />
          </ToggleGroupItem>
          <ToggleGroupItem value="preview" aria-label={s.preview}>
            <Eye className="size-3.5" />
          </ToggleGroupItem>
        </ToggleGroup>
      </div>
      {view === "edit" ? (
        <Textarea value={body} onChange={(e) => setBody(e.target.value)} rows={24} className="font-mono text-xs leading-relaxed" aria-label={s.body} />
      ) : (
        <div className="rounded-lg border bg-background p-(--card-pad)">
          <Markdown source={body} />
        </div>
      )}
      <div className="flex justify-end">
        <Button
          disabled={pending}
          onClick={() => void run(() => updateContractAction(contract.id, { title, body, startDate, endDate }), { success: ss.saved })}
        >
          {pending ? <Loader2 className="size-4 animate-spin" /> : null}
          {ss.save}
        </Button>
      </div>
    </Card>
  );
}

/** ADMIN: contract templates with {{placeholders}}. */
export function ContractTemplates({
  templates,
  placeholders,
  admin,
}: {
  templates: ContractTemplate[];
  placeholders: Record<string, string>;
  admin: boolean;
}) {
  return (
    <div className="flex flex-col gap-3">
      {admin ? (
        <div>
          <TemplateDialog
            template={null}
            placeholders={placeholders}
            trigger={
              <Button size="sm">
                <Plus className="size-3.5" /> {s.templateAdd}
              </Button>
            }
          />
        </div>
      ) : (
        <p className="text-xs text-muted-foreground">{ss.adminOnly}</p>
      )}
      <ul className="divide-y rounded-xl border bg-card">
        {templates.map((t) => (
          <li key={t.id} className="flex items-center gap-3 px-(--card-pad) py-3 text-sm">
            <span className="min-w-0 flex-1">
              <span className="block font-medium">{t.name}</span>
              <span className="text-xs text-muted-foreground">
                {lbl(CONTRACT_TYPE, t.type)}
                {!t.isActive ? ` · ${ss.inactive}` : ""}
              </span>
              {t.unknownPlaceholders.length ? (
                <span className="block text-xs text-destructive">{s.unknownPlaceholders(t.unknownPlaceholders.join(", "))}</span>
              ) : null}
            </span>
            {admin ? (
              <>
                <TemplateDialog
                  template={t}
                  placeholders={placeholders}
                  trigger={
                    <Button size="icon-sm" variant="ghost" aria-label={ss.edit}>
                      <Pencil className="size-3.5" />
                    </Button>
                  }
                />
                <ConfirmAction
                  title={s.deleteTemplate}
                  action={() => deleteContractTemplateAction(t.id)}
                  trigger={
                    <Button size="icon-sm" variant="ghost" aria-label={ss.delete}>
                      <Trash2 className="size-3.5" />
                    </Button>
                  }
                />
              </>
            ) : null}
          </li>
        ))}
      </ul>
    </div>
  );
}

function TemplateDialog({
  template,
  placeholders,
  trigger,
}: {
  template: ContractTemplate | null;
  placeholders: Record<string, string>;
  trigger: React.ReactNode;
}) {
  return (
    <FormDialog trigger={trigger} title={s.templateTitle} size="xl">
      {(close) => <TemplateForm template={template} placeholders={placeholders} close={close} />}
    </FormDialog>
  );
}

function TemplateForm({
  template,
  placeholders,
  close,
}: {
  template: ContractTemplate | null;
  placeholders: Record<string, string>;
  close: () => void;
}) {
  const [v, setV] = React.useState<ContractTemplate>(
    template ?? { id: "", name: "", type: "retainer", body: "", isActive: true, unknownPlaceholders: [] },
  );
  const { pending, run } = useErpAction();
  const ref = React.useRef<HTMLTextAreaElement>(null);
  const insert = (key: string) => {
    const el = ref.current;
    const tag = `{{${key}}}`;
    if (!el) return setV((p) => ({ ...p, body: p.body + tag }));
    const { selectionStart: a, selectionEnd: b } = el;
    setV((p) => ({ ...p, body: p.body.slice(0, a) + tag + p.body.slice(b) }));
  };
  return (
    <form
      className="grid gap-4 lg:grid-cols-[1fr_260px]"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveContractTemplateAction(v), { success: ss.saved, onOk: close });
      }}
    >
      <div className="flex flex-col gap-4">
        <div className="grid gap-4 sm:grid-cols-[1fr_160px]">
          <Fld label={`${s.templateName} *`} htmlFor="tpl-name">
            <Input id="tpl-name" value={v.name} onChange={(e) => setV((p) => ({ ...p, name: e.target.value }))} />
          </Fld>
          <Fld label={s.type} htmlFor="tpl-type">
            <SimpleSelect id="tpl-type" value={v.type} onChange={(x) => setV((p) => ({ ...p, type: x }))} options={opts(CONTRACT_TYPE)} />
          </Fld>
        </div>
        <Fld label={`${s.body} *`} htmlFor="tpl-body">
          <Textarea
            ref={ref}
            id="tpl-body"
            rows={20}
            value={v.body}
            onChange={(e) => setV((p) => ({ ...p, body: e.target.value }))}
            className="font-mono text-xs"
          />
        </Fld>
        <label className="flex items-center gap-2 text-sm">
          <Checkbox checked={v.isActive} onCheckedChange={(x) => setV((p) => ({ ...p, isActive: x === true }))} /> {ss.active}
        </label>
      </div>
      <aside className="flex max-h-[60dvh] flex-col gap-2 overflow-y-auto rounded-lg border p-3">
        <p className="text-xs font-medium">{s.placeholders}</p>
        {Object.entries(placeholders)
          .sort(([a], [b]) => a.localeCompare(b))
          .map(([key, desc]) => (
            <button
              key={key}
              type="button"
              onClick={() => insert(key)}
              className="rounded-md px-2 py-1 text-left text-xs hover:bg-accent"
            >
              <code className="font-mono text-[11px]">{`{{${key}}}`}</code>
              <span className="block text-muted-foreground">{desc}</span>
            </button>
          ))}
      </aside>
      <div className="lg:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}
