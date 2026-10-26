"use client";

import { CalendarDays, Loader2, SlidersHorizontal } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { cn } from "@/lib/utils";
import { useUrlFilters } from "@/features/erp/shared/url-state";
import { FILTER_PARAMS } from "@/features/content/dashboard";
import { FORMATS, PLATFORMS } from "@/features/content/lib";
import { c, cd } from "@/features/content/strings";
import { platformLabel } from "@/features/social/components/platform-icon";

const ALL = "__all__";

type Option = { value: string; label: string };

/**
 * Dimension filters and the comparison mode, all in the URL. The period
 * itself is the top bar picker (from/to/preset), shared with every
 * marketing page.
 */
export function DashFilterBar({
  periodLabel,
  compareLabel,
  pillars,
  people,
  plans,
}: {
  periodLabel: string;
  compareLabel: string | null;
  pillars: string[];
  people: Option[];
  plans: Option[];
}) {
  const { values, set, isPending } = useUrlFilters(FILTER_PARAMS);
  const active = FILTER_PARAMS.filter((p) => p !== "cmp" && values[p]).length;
  const reset = () => set(Object.fromEntries(FILTER_PARAMS.map((p) => [p, null])), { keepPage: true });

  const fields: { param: (typeof FILTER_PARAMS)[number]; label: string; all: string; options: Option[] }[] = [
    { param: "platform", label: cd.fPlatform, all: cd.allPlatforms, options: PLATFORMS.map((p) => ({ value: p, label: platformLabel(p) })) },
    { param: "format", label: cd.fFormat, all: cd.allFormats, options: FORMATS.map((f) => ({ value: f, label: c.format[f] })) },
    { param: "pillar", label: cd.fPillar, all: cd.allPillars, options: pillars.map((p) => ({ value: p, label: p })) },
    { param: "assignee", label: cd.fAssignee, all: cd.allPeople, options: people },
    { param: "plan", label: cd.fPlan, all: cd.allPlans, options: plans },
  ];

  const controls = (stacked: boolean) => (
    <>
      <Select value={values.cmp ?? "previous"} onValueChange={(v) => set({ cmp: v === "previous" ? null : v }, { keepPage: true })}>
        <SelectTrigger size="sm" className={cn(stacked ? "w-full" : "w-[11.5rem]")} aria-label={cd.fCompare}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="previous">{cd.vsPrevious}</SelectItem>
          <SelectItem value="year">{cd.vsYear}</SelectItem>
        </SelectContent>
      </Select>
      {fields.map((f) => (
        <Select key={f.param} value={values[f.param] ?? ALL} onValueChange={(v) => set({ [f.param]: v === ALL ? null : v }, { keepPage: true })}>
          <SelectTrigger size="sm" className={cn(stacked ? "w-full" : "w-40", values[f.param] && "border-brand/50 bg-brand/5")} aria-label={f.label}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL}>{f.all}</SelectItem>
            {f.options.map((o) => (
              <SelectItem key={o.value} value={o.value}>
                {o.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ))}
      {active > 0 || values.cmp ? (
        <Button variant="ghost" size="sm" onClick={reset}>
          {cd.reset}
        </Button>
      ) : null}
    </>
  );

  return (
    <div className="flex flex-wrap items-center gap-2 rounded-[1.25rem] bg-card px-3 py-2.5 shadow-(--card-shadow)">
      <span className="flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground">
        {isPending ? <Loader2 className="size-3.5 animate-spin" /> : <CalendarDays className="size-3.5" />}
        <span className="truncate">
          <span className="font-medium text-foreground">{periodLabel}</span>
          {compareLabel ? ` · ${compareLabel}` : ""}
        </span>
      </span>
      <div className="ml-auto hidden flex-wrap items-center justify-end gap-2 lg:flex">{controls(false)}</div>
      <Sheet>
        <SheetTrigger asChild>
          <Button variant="outline" size="sm" className="ml-auto lg:hidden">
            <SlidersHorizontal className="size-3.5" /> {cd.filters}
            {active ? <span className="rounded-full bg-brand px-1.5 text-[10px] text-white">{active}</span> : null}
          </Button>
        </SheetTrigger>
        <SheetContent side="bottom" className="max-h-[85svh] overflow-y-auto rounded-t-[1.5rem]">
          <SheetHeader>
            <SheetTitle>{cd.filters}</SheetTitle>
            <SheetDescription>{cd.filtersDesc}</SheetDescription>
          </SheetHeader>
          <div className="grid gap-3 px-4 pb-6">{controls(true)}</div>
        </SheetContent>
      </Sheet>
    </div>
  );
}
