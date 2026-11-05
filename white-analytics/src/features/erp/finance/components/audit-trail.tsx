"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { Eye } from "lucide-react";
import { Button } from "@/components/ui/button";
import { FormDialog } from "@/features/erp/sales/components/ui";
import { ErpDataTable, type ServerPage } from "@/features/erp/shared/data-table";
import { DownloadButton } from "@/features/erp/shared/download-button";
import { DateFilter } from "@/features/erp/shared/period-picker";
import { ResetFilters, SearchFilter, SelectFilter } from "@/features/erp/shared/url-filters";
import { fs } from "../strings";
import type { AuditEntry } from "../types";

const s = fs.auditTrail;
const when = new Intl.DateTimeFormat("id-ID", { dateStyle: "medium", timeStyle: "medium", timeZone: "Asia/Jakarta" });

function pretty(body: string) {
  try {
    return JSON.stringify(JSON.parse(body), null, 2);
  } catch {
    return body;
  }
}

function Body({ label, body }: { label: string; body: string }) {
  if (!body) return null;
  return (
    <div>
      <h3 className="text-xs font-semibold text-muted-foreground">{label}</h3>
      <pre className="mt-1 max-h-72 overflow-auto rounded-lg bg-muted p-3 text-xs whitespace-pre-wrap break-words">{pretty(body)}</pre>
    </div>
  );
}

export function AuditTable({ rows, page, query }: { rows: AuditEntry[]; page: ServerPage; query: string }) {
  const columns: ColumnDef<AuditEntry>[] = [
    { accessorKey: "at", header: s.at, cell: ({ row }) => <span className="tabular whitespace-nowrap text-xs">{when.format(new Date(row.original.at))}</span> },
    { accessorKey: "actorName", header: s.actor, cell: ({ row }) => row.original.actorName || s.system },
    { accessorKey: "action", header: s.action, cell: ({ row }) => s.actions[row.original.action] ?? row.original.action },
    { accessorKey: "docNo", header: s.document, cell: ({ row: { original: e } }) => e.docNo || <span className="text-xs text-muted-foreground">{e.docId.slice(0, 8)}</span> },
    {
      accessorKey: "result",
      header: s.result,
      cell: ({ row: { original: e } }) => (
        <span className={e.result === "REFUSED" ? "text-destructive" : undefined}>
          {s.results[e.result]}
          {e.message ? <span className="block max-w-72 text-xs text-muted-foreground">{e.message}</span> : null}
        </span>
      ),
    },
    {
      id: "detail",
      header: "",
      cell: ({ row: { original: e } }) => (
        <FormDialog
          title={s.actions[e.action] ?? e.action}
          description={`${e.actorName || s.system} · ${when.format(new Date(e.at))}`}
          size="lg"
          trigger={
            <Button size="xs" variant="ghost">
              <Eye className="size-3.5" /> {s.detail}
            </Button>
          }
        >
          {() => (
            <div className="grid gap-3">
              <Body label={s.request} body={e.request} />
              <Body label={s.response} body={e.response} />
            </div>
          )}
        </FormDialog>
      ),
    },
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={rows}
      getRowId={(e) => e.id}
      emptyMessage={s.empty}
      pagination={page}
      toolbar={
        <>
          <SearchFilter placeholder={s.search} />
          <SelectFilter param="result" label={s.result} allLabel={s.allResults} options={Object.entries(s.results).map(([value, label]) => ({ value, label }))} />
          <DateFilter param="from" label={s.from} />
          <DateFilter param="to" label={s.to} />
          <ResetFilters params={["q", "result", "from", "to"]} />
          <DownloadButton href={`/api/erp/v1/finance/reports/audit-trail-export?${query}`} fileName="audit-trail.csv">
            {s.export}
          </DownloadButton>
        </>
      }
    />
  );
}
