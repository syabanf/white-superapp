"use client";

import * as React from "react";
import Link from "next/link";
import { ChevronRight, Network } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { ErpEmpty } from "@/features/erp/shared/states";
import { cn } from "@/lib/utils";
import { th } from "../strings";
import type { OrgChart, OrgNode } from "../types";

function collectIds(nodes: OrgNode[], out: string[] = []): string[] {
  for (const n of nodes) {
    if (n.reports.length) out.push(n.employeeId);
    collectIds(n.reports, out);
  }
  return out;
}

/** Collapsible reporting tree built from employees' direct supervisors. */
export function OrgChartTree({ chart }: { chart: OrgChart }) {
  const all = React.useMemo(() => collectIds(chart.roots), [chart.roots]);
  const [open, setOpen] = React.useState<Set<string>>(() => new Set(chart.roots.map((r) => r.employeeId)));
  if (!chart.roots.length) return <ErpEmpty icon={<Network className="size-5" />} title={th.org.empty} />;
  const toggle = (id: string) =>
    setOpen((s) => {
      const next = new Set(s);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  return (
    <Card className="gap-3 px-(--card-pad)">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="text-sm text-muted-foreground">{th.org.total(chart.total)}</span>
        <div className="flex gap-2">
          <Button size="xs" variant="outline" onClick={() => setOpen(new Set(all))}>
            {th.org.expandAll}
          </Button>
          <Button size="xs" variant="outline" onClick={() => setOpen(new Set())}>
            {th.org.collapseAll}
          </Button>
        </div>
      </div>
      <ul className="flex flex-col gap-1">
        {chart.roots.map((n) => (
          <Node key={n.employeeId} node={n} depth={0} open={open} toggle={toggle} />
        ))}
      </ul>
    </Card>
  );
}

function Node({
  node,
  depth,
  open,
  toggle,
}: {
  node: OrgNode;
  depth: number;
  open: Set<string>;
  toggle: (id: string) => void;
}) {
  const expanded = open.has(node.employeeId);
  const hasReports = node.reports.length > 0;
  return (
    <li>
      <div className="flex items-center gap-1.5 rounded-lg py-1 pr-2 hover:bg-accent/50" style={{ paddingLeft: depth * 20 }}>
        <button
          type="button"
          onClick={() => toggle(node.employeeId)}
          disabled={!hasReports}
          aria-label={node.fullName}
          aria-expanded={hasReports ? expanded : undefined}
          className="flex size-6 items-center justify-center rounded text-muted-foreground disabled:opacity-0"
        >
          <ChevronRight className={cn("size-4 transition-transform", expanded && "rotate-90")} />
        </button>
        <Link href={`/erp/hr/employees/${node.employeeId}`} className="min-w-0 flex-1 truncate text-sm hover:underline">
          <span className="font-medium">{node.fullName}</span>
          <span className="text-muted-foreground">
            {" "}
            · {node.positionName || "–"}
            {node.departmentName ? ` · ${node.departmentName}` : ""}
          </span>
        </Link>
        {hasReports ? (
          <span className="shrink-0 text-xs text-muted-foreground tabular">{th.org.reports(node.reports.length)}</span>
        ) : null}
      </div>
      {hasReports && expanded ? (
        <ul className="flex flex-col gap-1">
          {node.reports.map((r) => (
            <Node key={r.employeeId} node={r} depth={depth + 1} open={open} toggle={toggle} />
          ))}
        </ul>
      ) : null}
    </li>
  );
}
