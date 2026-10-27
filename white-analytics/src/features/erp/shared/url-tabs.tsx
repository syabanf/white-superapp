"use client";

import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { useUrlFilters } from "./url-state";

/**
 * Tab bar bound to a URL param (default `tab`) for server-rendered pages:
 * picking a tab refetches the page with the new param. The first tab is the
 * default and clears the param.
 */
export function UrlTabs({
  tabs,
  param = "tab",
  label,
  className,
  reset = [],
}: {
  tabs: { value: string; label: string }[];
  param?: string;
  label: string;
  className?: string;
  /** other params to clear when the tab changes */
  reset?: string[];
}) {
  const { values, set } = useUrlFilters([param, ...reset]);
  const current = values[param] ?? tabs[0]?.value;
  return (
    <Tabs
      value={current}
      onValueChange={(v) =>
        set({ [param]: v === tabs[0]?.value ? null : v, ...Object.fromEntries(reset.map((r) => [r, null])) })
      }
      className={cn("min-w-0", className)}
    >
      <TabsList aria-label={label} className="max-w-full overflow-x-auto">
        {tabs.map((t) => (
          <TabsTrigger key={t.value} value={t.value}>
            {t.label}
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  );
}
