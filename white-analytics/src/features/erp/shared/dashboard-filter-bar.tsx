"use client";

import * as React from "react";
import { CalendarDays, GitCompareArrows, Loader2, X } from "lucide-react";
import type { DateRange as DayPickerRange } from "react-day-picker";
import { id as idLocale } from "date-fns/locale";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { parseIsoDate, toIsoDate } from "./period-picker";
import { useUrlFilters } from "./url-state";
import {
  COMPARE_MODES,
  CUSTOM_LABEL,
  PERIOD_PARAMS,
  PERIOD_PRESETS,
  formatPeriod,
  type ResolvedPeriod,
} from "./dashboard-period";

/**
 * Filter row for ERP analytics dashboards: period preset or custom range,
 * comparison mode, the module's dimension filters (children) and a reset
 * that clears every param in `params` plus the period. All state lives in
 * the URL, so the page is linkable and survives refresh. Wraps on phones.
 */
export function DashboardFilterBar({
  period,
  params = [],
  children,
  className,
}: {
  /** the period the server resolved, shown in the controls */
  period: ResolvedPeriod;
  /** dimension params cleared by Reset */
  params?: string[];
  children?: React.ReactNode;
  className?: string;
}) {
  const keys = React.useMemo(() => [...PERIOD_PARAMS, ...params], [params]);
  const { values, set, isPending } = useUrlFilters(keys);
  const dirty = keys.some((k) => values[k]);
  // dim the page while a filter change refetches instead of flashing skeletons
  React.useEffect(() => {
    document.body.dataset.navPending = isPending ? "true" : "false";
    return () => {
      delete document.body.dataset.navPending;
    };
  }, [isPending]);

  return (
    <div
      role="toolbar"
      aria-label="Filter dashboard"
      className={cn(
        // Phones: the period picker gets its own row so its label stays readable; the rest pair up.
        "flex flex-wrap items-center gap-(--gap-inner) [&>*]:max-sm:min-w-0 [&>*]:max-sm:flex-1 [&>*]:max-sm:basis-[calc(50%-0.375rem)] [&>*:first-child]:max-sm:basis-full",
        className,
      )}
    >
      <PeriodControl period={period} onChange={(patch) => set(patch)} pending={isPending} />
      <Select value={period.compare} onValueChange={(v) => set({ compare: v === "previous" ? null : v })}>
        <SelectTrigger aria-label="Pembanding" className="min-w-44">
          <GitCompareArrows className="size-3.5 text-muted-foreground" />
          <SelectValue>{COMPARE_MODES.find((m) => m.value === period.compare)?.label}</SelectValue>
        </SelectTrigger>
        <SelectContent>
          {COMPARE_MODES.map((m) => (
            <SelectItem key={m.value} value={m.value}>
              {m.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {children}
      {dirty ? (
        <Button variant="ghost" onClick={() => set(Object.fromEntries(keys.map((k) => [k, null])))}>
          <X /> Atur ulang
        </Button>
      ) : null}
    </div>
  );
}

function PeriodControl({
  period,
  onChange,
  pending,
}: {
  period: ResolvedPeriod;
  onChange: (patch: Record<string, string | null>) => void;
  pending: boolean;
}) {
  const [open, setOpen] = React.useState(false);
  const [draft, setDraft] = React.useState<DayPickerRange | undefined>();
  const openPicker = (next: boolean) => {
    if (next) setDraft({ from: parseIsoDate(period.from) ?? undefined, to: parseIsoDate(period.to) ?? undefined });
    setOpen(next);
  };
  const choose = (preset: string) => {
    onChange({ preset, from: null, to: null });
    setOpen(false);
  };
  const apply = () => {
    if (!draft?.from) return;
    onChange({ preset: null, from: toIsoDate(draft.from), to: toIsoDate(draft.to ?? draft.from) });
    setOpen(false);
  };
  const presetLabel = PERIOD_PRESETS.find((p) => p.value === period.preset)?.label ?? CUSTOM_LABEL;

  return (
    <Popover open={open} onOpenChange={openPicker}>
      <PopoverTrigger asChild>
        <Button variant="outline" className="justify-start font-normal shadow-none" aria-label={`Periode: ${period.label}`}>
          {pending ? (
            <Loader2 className="size-3.5 animate-spin text-muted-foreground" />
          ) : (
            <CalendarDays className="size-3.5 text-muted-foreground" />
          )}
          <span className="truncate">{presetLabel}</span>
          <span className="hidden truncate text-muted-foreground tabular lg:inline">{formatPeriod(period.from, period.to)}</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-[min(94vw,40rem)] p-0">
        <div className="flex flex-col sm:flex-row">
          <ul className="flex gap-1 overflow-x-auto border-b p-2 scrollbar-thin sm:block sm:w-48 sm:border-r sm:border-b-0" aria-label="Preset periode">
            {PERIOD_PRESETS.map((p) => (
              <li key={p.value} className="shrink-0">
                <button
                  type="button"
                  aria-pressed={period.preset === p.value}
                  onClick={() => choose(p.value)}
                  className={cn(
                    "w-full rounded-full px-3 py-1.5 text-left text-sm whitespace-nowrap hover:bg-accent",
                    period.preset === p.value && "bg-accent font-medium",
                  )}
                >
                  {p.label}
                </button>
              </li>
            ))}
          </ul>
          <div className="p-2">
            <p className="px-1 pb-1 text-xs font-medium text-muted-foreground">{CUSTOM_LABEL}</p>
            <Calendar
              mode="range"
              selected={draft}
              onSelect={setDraft}
              defaultMonth={draft?.from}
              numberOfMonths={1}
              weekStartsOn={1}
              locale={idLocale}
              captionLayout="dropdown"
            />
            <div className="flex justify-end gap-2 border-t px-1 pt-2">
              <Button variant="ghost" size="sm" onClick={() => setOpen(false)}>
                Batal
              </Button>
              <Button size="sm" onClick={apply} disabled={!draft?.from}>
                Terapkan
              </Button>
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
