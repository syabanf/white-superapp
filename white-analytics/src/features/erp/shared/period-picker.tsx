"use client";

import * as React from "react";
import { CalendarDays, ChevronLeft, ChevronRight, X } from "lucide-react";
import type { DateRange as DayPickerRange } from "react-day-picker";
import { id as idLocale } from "date-fns/locale";
import { format, parse, isValid } from "date-fns";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { currentMonth, formatMonth, shiftMonth } from "./month";
import { useUrlFilters } from "./url-state";
import { ss } from "./strings";

const ISO = "yyyy-MM-dd";

/** "2026-10-01" → local Date (no timezone shift); null when empty or invalid. */
export function parseIsoDate(value: string | null | undefined): Date | null {
  if (!value) return null;
  const d = parse(value.slice(0, 10), ISO, new Date());
  return isValid(d) ? d : null;
}

export function toIsoDate(d: Date): string {
  return format(d, ISO);
}

function dayLabel(d: Date | null): string {
  return d ? format(d, "d MMM yyyy", { locale: idLocale }) : "";
}

/** Controlled single-date picker for forms; value is "YYYY-MM-DD" or "". */
export function DatePicker({
  value,
  onChange,
  id,
  placeholder = ss.pickDate,
  disabled,
  invalid,
  clearable,
  className,
}: {
  value: string;
  onChange: (value: string) => void;
  id?: string;
  placeholder?: string;
  disabled?: boolean;
  invalid?: boolean;
  clearable?: boolean;
  className?: string;
}) {
  const [open, setOpen] = React.useState(false);
  const date = parseIsoDate(value);
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          id={id}
          type="button"
          variant="outline"
          disabled={disabled}
          aria-invalid={invalid || undefined}
          className={cn("w-full justify-start font-normal shadow-none", !date && "text-muted-foreground", className)}
        >
          <CalendarDays className="size-4 text-muted-foreground" />
          <span className="flex-1 truncate text-left tabular">{date ? dayLabel(date) : placeholder}</span>
          {clearable && date ? (
            <X
              className="size-3.5 text-muted-foreground"
              role="button"
              aria-label={ss.reset}
              onClick={(e) => {
                e.stopPropagation();
                onChange("");
              }}
            />
          ) : null}
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-auto p-0">
        <Calendar
          mode="single"
          selected={date ?? undefined}
          defaultMonth={date ?? undefined}
          onSelect={(d) => {
            onChange(d ? toIsoDate(d) : "");
            setOpen(false);
          }}
          weekStartsOn={1}
          locale={idLocale}
          captionLayout="dropdown"
        />
      </PopoverContent>
    </Popover>
  );
}

/** Date range bound to two URL params (default `from`/`to`, "YYYY-MM-DD"). */
export function DateRangeFilter({
  fromParam = "from",
  toParam = "to",
  placeholder = ss.pickRange,
  className,
}: {
  fromParam?: string;
  toParam?: string;
  placeholder?: string;
  className?: string;
}) {
  const { values, set } = useUrlFilters([fromParam, toParam]);
  const from = parseIsoDate(values[fromParam]);
  const to = parseIsoDate(values[toParam]);
  const [open, setOpenState] = React.useState(false);
  const [draft, setDraft] = React.useState<DayPickerRange | undefined>();
  const setOpen = (next: boolean) => {
    if (next) setDraft(from ? { from, to: to ?? undefined } : undefined);
    setOpenState(next);
  };
  const apply = () => {
    set({
      [fromParam]: draft?.from ? toIsoDate(draft.from) : null,
      [toParam]: draft?.from ? toIsoDate(draft.to ?? draft.from) : null,
    });
    setOpenState(false);
  };
  const text = from ? `${dayLabel(from)} – ${dayLabel(to ?? from)}` : placeholder;
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          className={cn("font-normal shadow-none", !from && "text-muted-foreground", className)}
        >
          <CalendarDays />
          <span className="tabular">{text}</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-auto p-2">
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
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              set({ [fromParam]: null, [toParam]: null });
              setOpenState(false);
            }}
          >
            {ss.reset}
          </Button>
          <Button size="sm" onClick={apply} disabled={!draft?.from}>
            {ss.apply}
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];




/** Month stepper bound to a URL param ("YYYY-MM"); empty param means `fallback`. */
export function MonthFilter({ param = "month", fallback }: { param?: string; fallback?: string }) {
  const { values, set } = useUrlFilters([param]);
  const value = values[param] ?? fallback ?? currentMonth();
  const [open, setOpen] = React.useState(false);
  const [year, setYear] = React.useState(Number(value.slice(0, 4)));
  return (
    <div className="flex items-center gap-1">
      <Button variant="outline" size="icon" aria-label={ss.prev} onClick={() => set({ [param]: shiftMonth(value, -1) })}>
        <ChevronLeft className="size-4" />
      </Button>
      <Popover
        open={open}
        onOpenChange={(o) => {
          if (o) setYear(Number(value.slice(0, 4)));
          setOpen(o);
        }}
      >
        <PopoverTrigger asChild>
          <Button variant="outline" className="min-w-36 font-normal shadow-none" aria-label={ss.pickMonth}>
            <CalendarDays />
            {formatMonth(value)}
          </Button>
        </PopoverTrigger>
        <PopoverContent align="center" className="w-64 p-3">
          <div className="mb-2 flex items-center justify-between">
            <Button variant="ghost" size="icon-xs" aria-label={ss.prev} onClick={() => setYear((y) => y - 1)}>
              <ChevronLeft className="size-4" />
            </Button>
            <span className="text-sm font-medium tabular">{year}</span>
            <Button variant="ghost" size="icon-xs" aria-label={ss.next} onClick={() => setYear((y) => y + 1)}>
              <ChevronRight className="size-4" />
            </Button>
          </div>
          <div className="grid grid-cols-3 gap-1">
            {MONTHS.map((m, i) => {
              const ym = `${year}-${String(i + 1).padStart(2, "0")}`;
              return (
                <Button
                  key={m}
                  size="sm"
                  variant={ym === value ? "default" : "ghost"}
                  onClick={() => {
                    set({ [param]: ym });
                    setOpen(false);
                  }}
                >
                  {m}
                </Button>
              );
            })}
          </div>
        </PopoverContent>
      </Popover>
      <Button variant="outline" size="icon" aria-label={ss.next} onClick={() => set({ [param]: shiftMonth(value, 1) })}>
        <ChevronRight className="size-4" />
      </Button>
    </div>
  );
}

/** Single date bound to a URL param ("YYYY-MM-DD"), e.g. a report's "as of". */
export function DateFilter({
  param,
  placeholder = ss.pickDate,
  label,
  className,
}: {
  param: string;
  placeholder?: string;
  /** shown before the date, e.g. "Per" */
  label?: string;
  className?: string;
}) {
  const { values, set } = useUrlFilters([param]);
  const date = parseIsoDate(values[param]);
  const [open, setOpen] = React.useState(false);
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="outline" className={cn("font-normal shadow-none", !date && "text-muted-foreground", className)}>
          <CalendarDays />
          {label ? <span className="text-muted-foreground">{label}</span> : null}
          <span className="tabular">{date ? dayLabel(date) : placeholder}</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-auto p-0">
        <Calendar
          mode="single"
          selected={date ?? undefined}
          defaultMonth={date ?? undefined}
          onSelect={(d) => {
            set({ [param]: d ? toIsoDate(d) : null });
            setOpen(false);
          }}
          weekStartsOn={1}
          locale={idLocale}
          captionLayout="dropdown"
        />
      </PopoverContent>
    </Popover>
  );
}

export { currentMonth, formatMonth };
