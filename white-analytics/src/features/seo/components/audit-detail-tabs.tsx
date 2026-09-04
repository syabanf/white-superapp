"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { StatusBadge, type StatusKind } from "@/components/dashboard/status-badge";
import { formatMs, formatNumber, truncate } from "@/lib/format";
import { t } from "@/i18n/id";
import { s } from "@/features/seo/strings";
import type { CrawlPageRow, IssueRow } from "@/features/seo/queries";

const right: ColMeta = { align: "right" };
const mono: ColMeta = { mono: true };

const SEVERITY_KIND: Record<IssueRow["severity"], StatusKind> = { ERROR: "critical", WARNING: "warning", NOTICE: "neutral" };
const SEVERITY_LABEL: Record<IssueRow["severity"], string> = { ERROR: "Error", WARNING: t.common.warning, NOTICE: t.common.notice };

/** Isu on-page + halaman hasil crawl — two tabs with searchable, exportable tables. */
export function AuditDetailTabs({ issues, pages }: { issues: IssueRow[]; pages: CrawlPageRow[] }) {
  const issueColumns: ColumnDef<IssueRow, unknown>[] = [
    {
      accessorKey: "severity",
      header: t.seo.severity,
      cell: ({ row }) => <StatusBadge kind={SEVERITY_KIND[row.original.severity]}>{SEVERITY_LABEL[row.original.severity]}</StatusBadge>,
    },
    { accessorKey: "code", header: s.codeCol, meta: mono, cell: ({ row }) => row.original.code },
    { accessorKey: "message", header: s.messageCol, cell: ({ row }) => <span title={row.original.message}>{truncate(row.original.message, 72)}</span> },
    { accessorKey: "url", header: t.seo.affectedUrl, meta: mono, cell: ({ row }) => <span title={row.original.url}>{truncate(row.original.url, 56)}</span> },
  ];

  const pageColumns: ColumnDef<CrawlPageRow, unknown>[] = [
    { accessorKey: "path", header: t.seo.affectedUrl, meta: mono, cell: ({ row }) => <span title={row.original.url}>{truncate(row.original.path, 48)}</span> },
    {
      accessorKey: "statusCode",
      header: t.common.status,
      meta: right,
      cell: ({ row }) => (
        <Badge variant="outline" className={row.original.statusCode >= 400 ? "border-negative/30 text-negative" : "text-muted-foreground"}>
          {row.original.statusCode}
        </Badge>
      ),
    },
    { accessorKey: "titleLength", header: s.titleLenCol, meta: right, cell: ({ row }) => formatNumber(row.original.titleLength) },
    { accessorKey: "metaLength", header: s.metaLenCol, meta: right, cell: ({ row }) => formatNumber(row.original.metaLength) },
    { accessorKey: "h1Count", header: "H1", meta: right, cell: ({ row }) => formatNumber(row.original.h1Count) },
    { accessorKey: "wordCount", header: s.wordsCol, meta: right, cell: ({ row }) => formatNumber(row.original.wordCount) },
    { accessorKey: "imagesMissingAlt", header: s.imgsNoAltCol, meta: right, cell: ({ row }) => formatNumber(row.original.imagesMissingAlt) },
    {
      accessorKey: "indexable",
      header: s.indexableCol,
      cell: ({ row }) =>
        row.original.indexable ? (
          <span className="text-xs text-muted-foreground">{t.common.yes}</span>
        ) : (
          <StatusBadge kind="warning">{t.common.no}</StatusBadge>
        ),
    },
    { accessorKey: "loadMs", header: s.loadMsCol, meta: right, cell: ({ row }) => (row.original.loadMs == null ? "–" : formatMs(row.original.loadMs)) },
  ];

  return (
    <Card className="gap-0 py-0">
      <CardContent className="p-5">
        <Tabs defaultValue="issues">
          <TabsList>
            <TabsTrigger value="issues">
              {s.issuesTab} <span className="tabular text-muted-foreground">{formatNumber(issues.length)}</span>
            </TabsTrigger>
            <TabsTrigger value="pages">
              {s.pagesTab} <span className="tabular text-muted-foreground">{formatNumber(pages.length)}</span>
            </TabsTrigger>
          </TabsList>
          <TabsContent value="issues" className="mt-3">
            <DataTable
              data={issues}
              columns={issueColumns}
              searchKey="url"
              searchPlaceholder={s.searchUrl}
              exportName="isu-seo"
              pageSize={10}
              emptyMessage={t.common.noData}
            />
          </TabsContent>
          <TabsContent value="pages" className="mt-3">
            <DataTable
              data={pages}
              columns={pageColumns}
              searchKey="path"
              searchPlaceholder={s.searchPage}
              exportName="halaman-crawl"
              pageSize={10}
              emptyMessage={t.common.noData}
            />
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  );
}
