"use client";

import * as React from "react";
import { useTransition } from "react";
import { parseAsString, useQueryStates } from "nuqs";
import { CalendarDays, Check, GitCompareArrows, Loader2 } from "lucide-react";
import type { DateRange as DayPickerRange } from "react-day-picker";
import { id as idLocale } from "date-fns/locale";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { PRESETS, resolveRange, toISODate, utcDate, type PresetKey } from "@/lib/dates";
import { formatDateRange } from "@/lib/format";
import { t } from "@/i18n/id";

const parsers = {
  from: parseAsString,
  to: parseAsString,
  preset: parseAsString,
  compare: parseAsString,
};
const DATE_PREF_KEY = "white:date-range";

function persistDateCookie(value: string) {
  document.cookie = `white_date_range=${encodeURIComponent(value)}; Path=/; Max-Age=31536000; SameSite=Lax`;
}

function utcToLocal(d: Date): Date {
  return new Date(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate());
}
function localToUtc(d: Date): Date {
  return utcDate(d.getFullYear(), d.getMonth(), d.getDate());
}

export function DateRangePicker({
  className,
  showCompare = true,
}: {
  className?: string;
  showCompare?: boolean;
}) {
  const [isPending, startTransition] = useTransition();
  const [calendarMonths, setCalendarMonths] = React.useState(1);
  React.useEffect(() => {
    const query = window.matchMedia("(min-width: 768px)");
    const sync = () => setCalendarMonths(query.matches ? 2 : 1);
    sync();
    query.addEventListener("change", sync);
    return () => query.removeEventListener("change", sync);
  }, []);
  // Hold the frame while the range refetches: dim the content instead of
  // flashing a skeleton (no layout jump, numbers never disappear).
  React.useEffect(() => {
    document.body.dataset.navPending = isPending ? "true" : "false";
    return () => {
      delete document.body.dataset.navPending;
    };
  }, [isPending]);
  const [params, setParams] = useQueryStates(parsers, {
    shallow: false,
    history: "replace",
    startTransition,
  });
  const persist = (value: Partial<Record<keyof typeof parsers, string | null>>) => {
    const next = { ...params, ...value };
    const serialized = JSON.stringify(next);
    window.localStorage.setItem(DATE_PREF_KEY, serialized);
    persistDateCookie(serialized);
  };
  const resolved = React.useMemo(
    () =>
      resolveRange({
        from: params.from ?? undefined,
        to: params.to ?? undefined,
        preset: params.preset ?? undefined,
        compare: params.compare ?? undefined,
      }),
    [params],
  );
  const [open, setOpenState] = React.useState(false);
  const [draft, setDraft] = React.useState<DayPickerRange | undefined>();
  const setOpen = (next: boolean) => {
    if (next) setDraft({ from: utcToLocal(resolved.range.from), to: utcToLocal(resolved.range.to) });
    setOpenState(next);
  };

  const presetLabel = resolved.preset
    ? PRESETS.find((p) => p.key === resolved.preset)?.label
    : t.common.customRange;

  const choosePreset = (key: PresetKey) => {
    persist({ preset: key, from: null, to: null });
    void setParams({ preset: key, from: null, to: null });
    setOpen(false);
  };
  const applyCustom = () => {
    if (!draft?.from) return;
    const from = localToUtc(draft.from);
    const to = localToUtc(draft.to ?? draft.from);
    persist({ from: toISODate(from), to: toISODate(to), preset: null });
    void setParams({ from: toISODate(from), to: toISODate(to), preset: null });
    setOpen(false);
  };
  const toggleCompare = (v: boolean) => {
    persist({ compare: v ? null : "0" });
    void setParams({ compare: v ? null : "0" });
  };

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button variant="outline" className="bg-background font-normal shadow-none">
            {isPending ? (
              <Loader2 className="size-4 animate-spin text-muted-foreground" />
            ) : (
              <CalendarDays className="size-4 text-muted-foreground" />
            )}
            <span className="hidden xl:inline">{presetLabel}</span>
            <span className="hidden text-muted-foreground tabular sm:inline">
              {formatDateRange(resolved.range.from, resolved.range.to)}
            </span>
            <span className="text-muted-foreground sm:hidden">{presetLabel}</span>
          </Button>
        </PopoverTrigger>
        <PopoverContent align="end" className="w-[min(94vw,46rem)] p-0 sm:w-auto">
          <div className="flex flex-col sm:flex-row">
            <ul
              className="flex w-full gap-1 overflow-x-auto border-b p-2 scrollbar-thin sm:block sm:w-44 sm:border-r sm:border-b-0 sm:py-2"
              role="listbox"
              aria-label={t.common.period}
            >
              {PRESETS.map((p) => {
                const selected = resolved.preset === p.key;
                return (
                  <li key={p.key} className="shrink-0">
                    <button
                      type="button"
                      role="option"
                      aria-selected={selected}
                      onClick={() => choosePreset(p.key)}
                      className={cn(
                        "flex w-full items-center justify-between gap-2 rounded-full px-3 py-1.5 text-left text-sm whitespace-nowrap hover:bg-accent sm:rounded-none",
                        selected && "bg-accent font-medium",
                      )}
                    >
                      {p.label}
                      {selected ? <Check className="size-4" strokeWidth={2.5} /> : null}
                    </button>
                  </li>
                );
              })}
            </ul>
            <div className="p-2">
              <Calendar
                mode="range"
                numberOfMonths={calendarMonths}
                selected={draft}
                onSelect={setDraft}
                defaultMonth={draft?.from}
                disabled={{ after: new Date() }}
                weekStartsOn={1}
                locale={idLocale}
              />
              <Separator className="my-2" />
              {showCompare ? (
                <div className="flex items-center justify-between gap-3 px-2 pb-3 md:hidden">
                  <Label htmlFor="compare-mobile" className="cursor-pointer gap-2 text-sm font-normal">
                    <GitCompareArrows className="size-4 text-muted-foreground" />
                    {t.common.comparePrevious}
                  </Label>
                  <Switch id="compare-mobile" checked={resolved.compare} onCheckedChange={toggleCompare} />
                </div>
              ) : null}
              <div className="flex items-center justify-between gap-3 px-2 pb-1">
                <span className="text-xs text-muted-foreground tabular">
                  {draft?.from
                    ? formatDateRange(localToUtc(draft.from), localToUtc(draft.to ?? draft.from))
                    : t.common.customRange}
                </span>
                <div className="flex gap-2">
                  <Button variant="ghost" size="sm" onClick={() => setOpen(false)}>
                    {t.common.cancel}
                  </Button>
                  <Button size="sm" onClick={applyCustom} disabled={!draft?.from}>
                    {t.common.apply}
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </PopoverContent>
      </Popover>
      {showCompare ? (
        <div className="hidden items-center gap-2 md:flex">
          <Switch
            id="compare"
            checked={resolved.compare}
            onCheckedChange={toggleCompare}
            aria-label={t.common.comparePrevious}
          />
          <Label htmlFor="compare" className="cursor-pointer gap-1 text-xs font-normal text-muted-foreground">
            <GitCompareArrows className="size-3.5" /> {t.common.vsPrevious}
          </Label>
        </div>
      ) : null}
    </div>
  );
}
