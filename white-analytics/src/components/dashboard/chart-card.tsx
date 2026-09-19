"use client";

import * as React from "react";
import { BarChart3, Table2 } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { InfoHint } from "@/components/dashboard/info-hint";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { cn } from "@/lib/utils";
import { t } from "@/i18n/id";

/**
 * Card wrapper for a chart with an optional table-view twin (accessibility rule:
 * every chart has a table). Pass `table` to enable the toggle.
 */
export function ChartCard({
  title,
  description,
  actions,
  children,
  table,
  className,
  contentClassName,
  footer,
}: {
  title: React.ReactNode;
  description?: React.ReactNode;
  actions?: React.ReactNode;
  children: React.ReactNode;
  table?: React.ReactNode;
  className?: string;
  contentClassName?: string;
  footer?: React.ReactNode;
}) {
  const [view, setView] = React.useState<"chart" | "table">("chart");
  return (
    <Card className={cn("gap-4 py-0 transition-shadow duration-200 hover:shadow-[var(--lift-shadow)]", className)}>
      <CardHeader className="flex flex-row items-center justify-between gap-3 space-y-0 px-6 pt-5 pb-0">
        <div className="flex min-w-0 items-center gap-1.5">
          <CardTitle className="truncate text-[15px] font-semibold tracking-[-0.015em]">{title}</CardTitle>
          {description ? <InfoHint>{description}</InfoHint> : null}
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {actions}
          {table ? (
            <ToggleGroup
              type="single"
              value={view}
              onValueChange={(v) => v && setView(v as "chart" | "table")}
              variant="outline"
              size="sm"
              aria-label="Tampilan"
              className="rounded-full"
            >
              <ToggleGroupItem value="chart" aria-label={t.common.viewChart} className="size-8 rounded-full p-0">
                <BarChart3 className="size-4" />
              </ToggleGroupItem>
              <ToggleGroupItem value="table" aria-label={t.common.viewTable} className="size-8 rounded-full p-0">
                <Table2 className="size-4" />
              </ToggleGroupItem>
            </ToggleGroup>
          ) : null}
        </div>
      </CardHeader>
      <CardContent className={cn("px-6 pb-6", contentClassName)}>
        {view === "table" && table ? <div className="max-h-[360px] overflow-auto scrollbar-thin">{table}</div> : children}
        {footer ? <div className="mt-3 border-t pt-3 text-xs text-muted-foreground">{footer}</div> : null}
      </CardContent>
    </Card>
  );
}
