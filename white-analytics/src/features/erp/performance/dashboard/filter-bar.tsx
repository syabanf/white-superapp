"use client";

import * as React from "react";
import { Loader2, SlidersHorizontal, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import {
  COMPARE_MODES,
  CUSTOM_LABEL,
  PERIOD_PRESETS,
  formatPeriod,
} from "@/features/erp/shared/dashboard-period";
import { DateRangeFilter } from "@/features/erp/shared/period-picker";
import { SearchableSelect } from "@/features/erp/shared/searchable-select";
import { useUrlFilters } from "@/features/erp/shared/url-state";
import { ds } from "./strings";
import type { DashContext } from "./types";

const PARAMS = ["preset", "from", "to", "compare", "dept", "emp", "scope"] as const;
const ALL = "__all__";
const f = ds.filters;

/**
 * Filter bar of the performance dashboard. Every value lives in the URL
 * (preset, from/to, compare, dept, emp, scope) and applies to every tab.
 * Below md the dimension filters move into a sheet.
 */
export function DashboardFilterBar({ context, preset }: { context: DashContext; preset: string }) {
  const { values, set, isPending } = useUrlFilters(PARAMS);
  const { window: w, scopes, departments, people } = context;
  const custom = preset === "custom";
  const active = PARAMS.some((p) => values[p]);

  const period = (
    <Select
      value={preset}
      onValueChange={(v) =>
        set(
          v === "custom" ? { preset: "custom", from: w.from, to: w.to } : { preset: v, from: null, to: null },
        )
      }
    >
      <SelectTrigger size="sm" className="min-w-40" aria-label={f.period}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {PERIOD_PRESETS.map((p) => (
          <SelectItem key={p.value} value={p.value}>
            {p.label}
          </SelectItem>
        ))}
        <SelectItem value="custom">{CUSTOM_LABEL}</SelectItem>
      </SelectContent>
    </Select>
  );

  const dims = (
    <>
      <Select
        value={values.compare ?? "previous"}
        onValueChange={(v) => set({ compare: v === "previous" ? null : v })}
      >
        <SelectTrigger size="sm" className="min-w-44" aria-label={f.compare}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {COMPARE_MODES.map((c) => (
            <SelectItem key={c.value} value={c.value}>
              {c.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {scopes.length > 1 ? (
        <Select value={w.scope} onValueChange={(v) => set({ scope: v, dept: null, emp: null })}>
          <SelectTrigger size="sm" className="min-w-32" aria-label={f.scope}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {scopes.map((s) => (
              <SelectItem key={s} value={s}>
                {f.scopes[s]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ) : null}
      {w.scope !== "me" && departments.length > 1 ? (
        <Select
          value={values.dept ?? ALL}
          onValueChange={(v) => set({ dept: v === ALL ? null : v, emp: null })}
        >
          <SelectTrigger size="sm" className="min-w-44" aria-label={f.department}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>{f.allDepartments}</SelectItem>
            {departments.map((d) => (
              <SelectItem key={d.id} value={d.id}>
                {d.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ) : null}
      {w.scope !== "me" && people.length > 1 ? (
        <SearchableSelect
          options={people.map((p) => ({ value: p.id, label: p.name, hint: p.departmentName }))}
          value={values.emp ?? ""}
          onChange={(v) => set({ emp: v || null })}
          placeholder={f.allEmployees}
          className="h-8 w-full text-[13px] md:w-56"
          clearable
        />
      ) : null}
    </>
  );

  return (
    <div role="group" aria-label={f.label} className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2">
        {period}
        {custom ? <DateRangeFilter /> : null}
        <div className="hidden flex-wrap items-center gap-2 md:flex">{dims}</div>
        <Sheet>
          <SheetTrigger asChild>
            <Button variant="outline" size="sm" className="md:hidden">
              <SlidersHorizontal className="size-3.5" /> {f.open}
            </Button>
          </SheetTrigger>
          <SheetContent side="bottom" className="max-h-[85svh] overflow-y-auto">
            <SheetHeader>
              <SheetTitle>{f.label}</SheetTitle>
            </SheetHeader>
            <div className="flex flex-col gap-3 px-4 pb-6 [&_button[role=combobox]]:w-full">{dims}</div>
          </SheetContent>
        </Sheet>
        {active ? (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => set(Object.fromEntries(PARAMS.map((p) => [p, null])))}
          >
            <X className="size-3.5" /> {f.reset}
          </Button>
        ) : null}
        {isPending ? (
          <Loader2 className="size-4 animate-spin text-muted-foreground" aria-label="Memuat" />
        ) : null}
      </div>
      <p className="text-xs text-muted-foreground tabular">
        {formatPeriod(w.from, w.to)}
        {w.compareFrom ? ` · ${f.vs(formatPeriod(w.compareFrom, w.compareTo))}` : ""}
      </p>
    </div>
  );
}
