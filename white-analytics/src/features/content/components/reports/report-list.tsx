"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import type { ColumnDef } from "@tanstack/react-table";
import { Loader2, Sparkles } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { DataTable } from "@/components/dashboard/data-table";
import { monthLabel } from "@/features/publishing/time";
import { generateReport } from "@/features/content/actions";
import { c } from "@/features/content/strings";
import type { ContentReport } from "@/features/content/types";
import { formatDecimal, formatRelative } from "@/lib/format";

export function GenerateReportForm({ slug, defaultMonth }: { slug: string; defaultMonth: string }) {
  const router = useRouter();
  const [month, setMonth] = React.useState(defaultMonth);
  const [pending, start] = React.useTransition();
  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    start(async () => {
      const res = await generateReport({ slug, month });
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      toast.success(c.reportGenerated);
      router.push(`/clients/${slug}/content/reports/${res.data.id}`);
    });
  };
  return (
    <form onSubmit={submit} className="flex items-center gap-2">
      <Input type="month" value={month} onChange={(e) => setMonth(e.target.value)} className="w-44" aria-label={c.month} required />
      <Button type="submit" disabled={pending || !month}>
        {pending ? <Loader2 className="animate-spin" /> : <Sparkles />} {c.generateReport}
      </Button>
    </form>
  );
}

export function ReportsTable({ slug, reports }: { slug: string; reports: ContentReport[] }) {
  const router = useRouter();
  const columns = React.useMemo<ColumnDef<ContentReport>[]>(
    () => [
      { id: "month", accessorKey: "month", header: c.colMonth, cell: ({ row }) => <span className="text-[13px] font-medium">{monthLabel(row.original.month)}</span> },
      {
        id: "published",
        accessorFn: (r) => (r.planned ? r.published / r.planned : 0),
        header: `${c.published} / ${c.planned}`,
        size: 150,
        cell: ({ row }) => (
          <span className="tabular text-[13px]">
            {row.original.published} / {row.original.planned}
          </span>
        ),
      },
      {
        id: "ontime",
        accessorKey: "onTimeRate",
        header: c.onTime,
        size: 110,
        cell: ({ row }) => <span className="tabular text-[13px]">{formatDecimal(row.original.onTimeRate)}%</span>,
      },
      {
        id: "revisions",
        accessorKey: "revisionsTotal",
        header: c.revisionsTotal,
        size: 100,
        cell: ({ row }) => <span className="tabular text-[13px]">{row.original.revisionsTotal}</span>,
      },
      {
        id: "updated",
        accessorKey: "updatedAt",
        header: c.colUpdated,
        size: 130,
        cell: ({ row }) => <span className="text-xs text-muted-foreground">{formatRelative(row.original.updatedAt)}</span>,
      },
    ],
    [],
  );
  return (
    <DataTable
      columns={columns}
      data={reports}
      initialSorting={[{ id: "month", desc: true }]}
      pageSize={12}
      emptyMessage={c.noReports}
      onRowClick={(row) => router.push(`/clients/${slug}/content/reports/${row.original.id}`)}
      rowClassName={() => "cursor-pointer"}
      getRowId={(r) => r.id}
    />
  );
}

export function RegenerateButton({ slug, month }: { slug: string; month: string }) {
  const router = useRouter();
  const [pending, start] = React.useTransition();
  return (
    <Button
      variant="outline"
      disabled={pending}
      onClick={() =>
        start(async () => {
          const res = await generateReport({ slug, month });
          if (!res.ok) {
            toast.error(res.error);
            return;
          }
          toast.success(c.reportGenerated);
          router.push(`/clients/${slug}/content/reports/${res.data.id}`);
          router.refresh();
        })
      }
    >
      {pending ? <Loader2 className="animate-spin" /> : <Sparkles />} {c.regenerate}
    </Button>
  );
}
