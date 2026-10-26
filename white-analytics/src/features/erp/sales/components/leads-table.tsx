"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Plus } from "lucide-react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { ErpDataTable, type ServerPage } from "@/features/erp/shared/data-table";
import { ResetFilters, SearchFilter, SelectFilter } from "@/features/erp/shared/url-filters";
import { formatDate } from "@/lib/format";
import { nameOf, type SalesCaps } from "../lib";
import { LEAD_SOURCE, LEAD_STATUS, ORG_TYPE, TEMPERATURE, lbl, opts, ss } from "../strings";
import type { CustomField, Lead } from "../types";
import { LeadFormDialog, LeadImportDialog } from "./lead-form";
import { SalesBadge, type Option } from "./ui";

const s = ss.leads;

export function ScoreBadge({ score }: { score: number }) {
  const tone = score >= 50 ? "bg-positive/15 text-positive" : score >= 20 ? "bg-amber-500/15 text-amber-700 dark:text-amber-400" : "bg-muted text-muted-foreground";
  return <span className={`inline-flex min-w-8 justify-center rounded-full px-2 py-0.5 text-xs font-semibold tabular ${tone}`}>{score}</span>;
}

export function LeadsTable({
  leads,
  pagination,
  owners,
  accounts,
  customFields,
  caps,
}: {
  leads: Lead[];
  pagination: ServerPage;
  owners: Option[];
  accounts: Option[];
  customFields: CustomField[];
  caps: SalesCaps;
}) {
  const router = useRouter();
  const names = React.useMemo(() => Object.fromEntries(owners.map((o) => [o.value, o.label])), [owners]);
  const listFields = customFields.filter((f) => f.isActive && f.showInList);
  const columns: ColumnDef<Lead, unknown>[] = [
    {
      accessorKey: "orgName",
      header: s.orgName,
      cell: ({ row }) => (
        <div className="min-w-0">
          <div className="truncate font-medium">{row.original.orgName}</div>
          <div className="text-xs text-muted-foreground">
            {lbl(ORG_TYPE, row.original.orgType)}
            {row.original.city ? ` · ${row.original.city}` : ""}
          </div>
        </div>
      ),
    },
    {
      accessorKey: "picName",
      header: s.pic,
      cell: ({ row }) => (
        <div className="min-w-0">
          <div className="truncate">{row.original.picName}</div>
          <div className="text-xs text-muted-foreground tabular">{row.original.picPhone}</div>
        </div>
      ),
    },
    { accessorKey: "source", header: s.source, cell: ({ row }) => lbl(LEAD_SOURCE, row.original.source) },
    {
      accessorKey: "temperature",
      header: s.temperature,
      cell: ({ row }) => <SalesBadge value={row.original.temperature} map={TEMPERATURE} />,
    },
    { accessorKey: "status", header: s.status, cell: ({ row }) => <SalesBadge value={row.original.status} map={LEAD_STATUS} /> },
    { accessorKey: "score", header: s.score, cell: ({ row }) => <ScoreBadge score={row.original.score} />, meta: { align: "right" } },
    ...listFields.map(
      (f): ColumnDef<Lead, unknown> => ({
        id: `cf-${f.key}`,
        header: f.label,
        accessorFn: (l) => {
          const v = l.custom?.[f.key];
          return Array.isArray(v) ? v.join(", ") : v == null ? "" : String(v);
        },
      }),
    ),
    { id: "owner", header: ss.owner, accessorFn: (l) => nameOf(names, l.ownerUserId) },
    { id: "created", header: ss.createdAt, accessorFn: (l) => l.createdAt ?? "", cell: ({ row }) => formatDate(row.original.createdAt) },
  ];

  return (
    <ErpDataTable
      columns={columns}
      data={leads}
      pagination={pagination}
      getRowId={(l) => l.id}
      onRowClick={(row) => router.push(`/erp/sales/leads/${row.original.id}`)}
      exportName="lead"
      toolbar={
        <>
          <SearchFilter placeholder={s.searchPlaceholder} />
          <SelectFilter param="status" label={s.status} allLabel={s.allStatus} options={opts(LEAD_STATUS)} />
          <SelectFilter param="temperature" label={s.temperature} allLabel={s.allTemp} options={opts(TEMPERATURE)} />
          <SelectFilter param="source" label={s.source} allLabel={s.allSource} options={opts(LEAD_SOURCE)} />
          <SelectFilter param="owner" label={ss.owner} allLabel={s.allOwner} options={owners} />
          <ResetFilters params={["q", "status", "temperature", "source", "owner"]} />
          {caps.write ? (
            <div className="ml-auto flex gap-2">
              <LeadImportDialog />
              <LeadFormDialog
                lead={null}
                owners={owners}
                accounts={accounts}
                customFields={customFields}
                trigger={
                  <Button size="sm">
                    <Plus className="size-3.5" /> {s.add}
                  </Button>
                }
              />
            </div>
          ) : null}
        </>
      }
    />
  );
}
