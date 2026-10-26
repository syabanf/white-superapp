"use client";

import * as React from "react";
import { CalendarDays, Check, Loader2, SlidersHorizontal, X } from "lucide-react";
import type { DateRange as DayPickerRange } from "react-day-picker";
import { id as idLocale } from "date-fns/locale";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Sheet, SheetContent, SheetFooter, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { cn } from "@/lib/utils";
import { useUrlFilters } from "@/features/erp/shared/url-state";
import { parseIsoDate, toIsoDate } from "@/features/erp/shared/period-picker";
import { HR_PRESETS, type HrRange } from "@/features/erp/hr/dashboard/range";
import { rangeLabel } from "@/features/erp/hr/dashboard/format";
import { thd } from "@/features/erp/hr/dashboard/strings";

const KEYS = ["preset", "from", "to", "compare", "dept", "status", "position"] as const;
const ALL = "__all__";
const f = thd.filters;

export type FilterOption = { value: string; label: string };

/**
 * HR dashboard filter bar. Period presets or a custom range, comparison
 * basis, and department / employment status / position; every value lives
 * in the URL. On phones the dimension filters move into a sheet.
 */
export function HrFilterBar({
  range,
  compareFrom,
  compareTo,
  departments,
  statuses,
  positions,
}: {
  range: HrRange;
  compareFrom: string;
  compareTo: string;
  departments: FilterOption[];
  statuses: FilterOption[];
  positions: FilterOption[];
}) {
  const { values, set, isPending } = useUrlFilters(KEYS);
  React.useEffect(() => {
    document.body.dataset.navPending = isPending ? "true" : "false";
    return () => {
      delete document.body.dataset.navPending;
    };
  }, [isPending]);
  const dims = [
    { param: "dept" as const, label: f.department, all: f.allDepartments, options: departments },
    { param: "status" as const, label: f.status, all: f.allStatuses, options: statuses },
    { param: "position" as const, label: f.position, all: f.allPositions, options: positions },
  ];
  const activeDims = dims.filter((d) => values[d.param]).length;
  const dirty = KEYS.some((k) => values[k]);

  const selects = (className?: string) =>
    dims.map((d) => (
      <Select key={d.param} value={values[d.param] ?? ALL} onValueChange={(v) => set({ [d.param]: v === ALL ? null : v })}>
        <SelectTrigger size="sm" aria-label={d.label} className={cn("min-w-0 bg-background", className)}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL}>{d.all}</SelectItem>
          {d.options.map((o) => (
            <SelectItem key={o.value} value={o.value}>
              {o.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    ));

  return (
    <div className="flex flex-col gap-3 rounded-[1.25rem] bg-card p-3 shadow-(--card-shadow) sm:p-4">
      <div className="flex flex-wrap items-center gap-3">
        <PeriodPicker range={range} pending={isPending} onChange={set} />
        <ToggleGroup
          type="single"
          value={range.compare}
          onValueChange={(v) => v && set({ compare: v === "year" ? "year" : null })}
          variant="outline"
          size="sm"
          aria-label={f.compare}
          className="rounded-full"
        >
          <ToggleGroupItem value="previous" className="rounded-l-full px-3 text-xs">
            {f.comparePrevious}
          </ToggleGroupItem>
          <ToggleGroupItem value="year" className="rounded-r-full px-3 text-xs">
            {f.compareYear}
          </ToggleGroupItem>
        </ToggleGroup>
        <div className="hidden min-w-0 flex-1 flex-wrap items-center gap-3 lg:flex">{selects("w-40")}</div>
        <Sheet>
          <SheetTrigger asChild>
            <Button variant="outline" size="sm" className="bg-background lg:hidden">
              <SlidersHorizontal className="size-3.5" />
              {f.open}
              {activeDims ? <span className="rounded-full bg-brand px-1.5 text-[10px] text-white tabular">{activeDims}</span> : null}
            </Button>
          </SheetTrigger>
          <SheetContent side="bottom" className="rounded-t-3xl">
            <SheetHeader>
              <SheetTitle>{f.title}</SheetTitle>
            </SheetHeader>
            <div className="grid gap-3 px-4">
              {dims.map((d, i) => (
                <label key={d.param} className="grid gap-1.5 text-xs text-muted-foreground">
                  {d.label}
                  {selects("w-full")[i]}
                </label>
              ))}
            </div>
            <SheetFooter>
              <Button variant="ghost" onClick={() => set(Object.fromEntries(dims.map((d) => [d.param, null])))}>
                {f.reset}
              </Button>
            </SheetFooter>
          </SheetContent>
        </Sheet>
      </div>
      <div className="flex min-h-7 flex-wrap items-center justify-between gap-2 px-1">
        <p className="text-xs text-muted-foreground">
          <span className="font-medium text-foreground tabular">{rangeLabel(range.from, range.to)}</span>{" "}
          <span className="tabular">{f.comparing(rangeLabel(compareFrom, compareTo))}</span>
          {activeDims ? <span> · {f.active(activeDims)}</span> : null}
        </p>
        {dirty ? (
          <Button variant="ghost" size="xs" onClick={() => set(Object.fromEntries(KEYS.map((k) => [k, null])))}>
            <X className="size-3.5" /> {f.reset}
          </Button>
        ) : null}
      </div>
    </div>
  );
}

function PeriodPicker({
  range,
  pending,
  onChange,
}: {
  range: HrRange;
  pending: boolean;
  onChange: (patch: Partial<Record<(typeof KEYS)[number], string | null>>) => void;
}) {
  const [open, setOpenState] = React.useState(false);
  const [draft, setDraft] = React.useState<DayPickerRange | undefined>();
  const setOpen = (next: boolean) => {
    if (next) setDraft({ from: parseIsoDate(range.from) ?? undefined, to: parseIsoDate(range.to) ?? undefined });
    setOpenState(next);
  };
  const label = range.preset ? f.presets[range.preset] : f.custom;
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="bg-background font-normal shadow-none">
          {pending ? <Loader2 className="size-3.5 animate-spin" /> : <CalendarDays className="size-3.5 text-muted-foreground" />}
          <span className="font-medium">{label}</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-[calc(100vw-1rem)] p-0 sm:w-auto">
        <div className="flex flex-col sm:flex-row">
          <ul className="flex flex-wrap gap-1 border-b p-2 sm:block sm:w-44 sm:border-r sm:border-b-0" role="listbox" aria-label={f.period}>
            {HR_PRESETS.map((p) => {
              const selected = range.preset === p;
              return (
                <li key={p} className="shrink-0">
                  <button
                    type="button"
                    role="option"
                    aria-selected={selected}
                    onClick={() => {
                      onChange({ preset: p === "thisMonth" ? null : p, from: null, to: null });
                      setOpenState(false);
                    }}
                    className={cn(
                      "flex w-full items-center justify-between gap-2 rounded-full px-3 py-1.5 text-left text-sm whitespace-nowrap hover:bg-accent sm:rounded-lg",
                      selected && "bg-accent font-medium",
                    )}
                  >
                    {f.presets[p]}
                    {selected ? <Check className="size-4" /> : null}
                  </button>
                </li>
              );
            })}
          </ul>
          <div className="p-2">
            <p className="px-2 pb-1 text-xs font-medium text-muted-foreground">{f.customRange}</p>
            <Calendar
              mode="range"
              selected={draft}
              onSelect={setDraft}
              defaultMonth={draft?.from}
              disabled={{ after: new Date() }}
              weekStartsOn={1}
              locale={idLocale}
              captionLayout="dropdown"
            />
            <div className="flex justify-end gap-2 border-t px-1 pt-2">
              <Button variant="ghost" size="sm" onClick={() => setOpenState(false)}>
                {f.cancel}
              </Button>
              <Button
                size="sm"
                disabled={!draft?.from}
                onClick={() => {
                  if (!draft?.from) return;
                  onChange({ preset: null, from: toIsoDate(draft.from), to: toIsoDate(draft.to ?? draft.from) });
                  setOpenState(false);
                }}
              >
                {f.apply}
              </Button>
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
