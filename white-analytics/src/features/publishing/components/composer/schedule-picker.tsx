"use client";

import { Sparkles, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { BestTimeSlot } from "@/features/publishing/queries";
import { dayLabelLong, toZoned, tzLabel, weekdayLabel } from "@/features/publishing/time";
import { p } from "@/features/publishing/strings";
import { cn } from "@/lib/utils";

const pad = (n: number) => String(n).padStart(2, "0");

/**
 * Date + time inputs in the CLIENT's timezone (native pickers, no deps) with
 * three "best time" chips derived from the account's posting history.
 */
export function SchedulePicker({
  date,
  time,
  onChange,
  timeZone,
  bestTimes,
  disabled,
  min,
}: {
  date: string;
  time: string;
  onChange: (date: string, time: string) => void;
  timeZone: string;
  bestTimes: BestTimeSlot[];
  disabled?: boolean;
  /** earliest selectable calendar day (client tz) */
  min?: string;
}) {
  const label = tzLabel(timeZone);
  const fromHistory = bestTimes.some((b) => b.fromHistory);
  return (
    <div className="space-y-3">
      <div className="grid grid-cols-[1fr_auto] gap-2 sm:max-w-sm">
        <div className="space-y-1">
          <Label htmlFor="sched-date">{p.scheduleDate}</Label>
          <Input id="sched-date" type="date" value={date} min={min} onChange={(e) => onChange(e.target.value, time || "10:00")} disabled={disabled} className="h-9" />
        </div>
        <div className="space-y-1">
          <Label htmlFor="sched-time">
            {p.scheduleTime} <span className="label-mono text-muted-foreground">{label}</span>
          </Label>
          <Input id="sched-time" type="time" value={time} step={300} onChange={(e) => onChange(date, e.target.value)} disabled={disabled} className="h-9 w-28" />
        </div>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <span className="label-mono inline-flex items-center gap-1 text-brand">
          <Sparkles className="size-3" /> {p.bestTimes}
        </span>
        {bestTimes.map((b) => {
          const z = toZoned(new Date(b.at), timeZone);
          const active = z.date === date && z.time === time;
          return (
            <Button
              key={`${b.day}-${b.hour}`}
              type="button"
              variant={active ? "default" : "outline"}
              size="xs"
              onClick={() => onChange(z.date, z.time)}
              disabled={disabled}
              className={cn("tabular")}
              title={dayLabelLong(z.date)}
            >
              {weekdayLabel(z.weekday)} {pad(z.hour)}.{pad(z.minute)}
            </Button>
          );
        })}
        {date ? (
          <Button type="button" variant="ghost" size="xs" onClick={() => onChange("", "")} disabled={disabled}>
            <X className="size-3" /> {p.clearSchedule}
          </Button>
        ) : null}
      </div>
      <p className="text-xs text-muted-foreground">{fromHistory ? p.bestTimesHint : p.bestTimesDefault}</p>
    </div>
  );
}
