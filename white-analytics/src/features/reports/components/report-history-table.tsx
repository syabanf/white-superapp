"use client";

import * as React from "react";
import { useTransition } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import type { ColumnDef } from "@tanstack/react-table";
import { Download, Loader2, Sparkles, Trash2 } from "lucide-react";
import { toast } from "@/lib/toast";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/dashboard/data-table";
import { EmptyState } from "@/components/dashboard/empty-state";
import { deleteReport } from "@/features/reports/actions";
import { formatDate, formatDateRange } from "@/lib/format";
import { t } from "@/i18n/id";
import { rs } from "@/features/reports/strings";

export type ReportRow = {
  id: string;
  title: string;
  from: string; // ISO
  to: string; // ISO
  modules: string[];
  language: string;
  hasAiSummary: boolean;
  createdByName: string | null;
  createdAt: string; // ISO
};

const MODULE_LABEL: Record<string, string> = {
  SOCIAL: t.overview.socialCard,
  SEO: t.overview.seoCard,
  ADS: t.overview.adsCard,
};

function DeleteButton({ slug, id }: { slug: string; id: string }) {
  const router = useRouter();
  const [pending, startTransition] = useTransition();
  const run = () => {
    startTransition(async () => {
      const res = await deleteReport(slug, id);
      if (res.ok) {
        toast.success(rs.history.deleted);
        router.refresh();
      } else {
        toast.error(res.error);
      }
    });
  };
  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="size-7 text-muted-foreground hover:text-destructive"
          aria-label={t.common.delete}
          disabled={pending}
        >
          {pending ? <Loader2 className="size-3.5 animate-spin" /> : <Trash2 className="size-3.5" />}
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{rs.history.deleteTitle}</AlertDialogTitle>
          <AlertDialogDescription>{t.common.confirmDeleteDesc}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{t.common.cancel}</AlertDialogCancel>
          <AlertDialogAction onClick={run} className="bg-destructive text-white hover:bg-destructive/90">
            {t.common.delete}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

/** Fires the "report created" toast once after the builder redirect. */
function CreatedToast() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const created = searchParams.get("created");
  React.useEffect(() => {
    if (!created) return;
    toast.success(t.reports.generated);
    const params = new URLSearchParams(searchParams.toString());
    params.delete("created");
    router.replace(`?${params.toString()}`, { scroll: false });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [created]);
  return null;
}

export function ReportHistoryTable({
  slug,
  rows,
  canManage,
}: {
  slug: string;
  rows: ReportRow[];
  canManage: boolean;
}) {
  const columns: ColumnDef<ReportRow, unknown>[] = [
    {
      accessorKey: "title",
      header: t.reports.reportTitle,
      cell: ({ row }) => (
        <span className="flex items-center gap-2 font-medium">
          {row.original.title}
          {row.original.hasAiSummary ? (
            <Sparkles className="size-3.5 shrink-0 text-muted-foreground" aria-label={t.reports.aiInsight} />
          ) : null}
        </span>
      ),
    },
    {
      id: "period",
      header: rs.history.period,
      accessorFn: (r) => `${r.from}..${r.to}`,
      cell: ({ row }) => (
        <span className="tabular text-muted-foreground">
          {formatDateRange(new Date(row.original.from), new Date(row.original.to))}
        </span>
      ),
    },
    {
      id: "modules",
      header: rs.history.modules,
      accessorFn: (r) => r.modules.join(","),
      enableSorting: false,
      cell: ({ row }) => (
        <span className="flex flex-wrap gap-1">
          {row.original.modules.map((m) => (
            <Badge key={m} variant="secondary" className="text-xs">
              {MODULE_LABEL[m] ?? m}
            </Badge>
          ))}
        </span>
      ),
    },
    {
      accessorKey: "createdByName",
      header: t.reports.createdBy,
      cell: ({ getValue }) => <span className="text-muted-foreground">{String(getValue() ?? "–")}</span>,
    },
    {
      accessorKey: "createdAt",
      header: rs.history.createdAt,
      cell: ({ getValue }) => <span className="text-muted-foreground">{formatDate(String(getValue()))}</span>,
    },
    {
      id: "actions",
      header: t.common.actions,
      enableSorting: false,
      cell: ({ row }) => (
        <span className="flex items-center justify-end gap-1">
          <Button asChild variant="outline" size="xs">
            <a href={`/api/reports/${row.original.id}/pdf`}>
              <Download className="size-3.5" /> {t.reports.downloadPdf}
            </a>
          </Button>
          {canManage ? <DeleteButton slug={slug} id={row.original.id} /> : null}
        </span>
      ),
      meta: { align: "right" },
    },
  ];

  return (
    <>
      <CreatedToast />
      {rows.length === 0 ? (
        <EmptyState title={t.reports.historyEmpty} description={t.reports.historyEmptyDesc} />
      ) : (
        <DataTable
          columns={columns}
          data={rows}
          searchKey="title"
          exportName={`laporan-${slug}`}
          pageSize={10}
          getRowId={(r) => r.id}
          initialSorting={[{ id: "createdAt", desc: true }]}
        />
      )}
    </>
  );
}
