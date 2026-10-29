"use client";

import { useState, useTransition, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { createReportSchedule, updateReportSchedule } from "@/features/ads-automation/actions";
import type { ScheduleItem } from "@/features/ads-automation/queries";
import { aa } from "@/features/ads-automation/strings";
import { t } from "@/i18n/id";

type Module = "SOCIAL" | "SEO" | "ADS";
const MODULES: { key: Module; label: string }[] = [
  { key: "SOCIAL", label: t.overview.socialCard },
  { key: "SEO", label: t.overview.seoCard },
  { key: "ADS", label: t.overview.adsCard },
];
const DEFAULT_NAME = { WEEKLY: aa.schedules.defaultNameWeekly, MONTHLY: aa.schedules.defaultNameMonthly };

/** Create (no `schedule`) or edit a report schedule. */
export function ScheduleFormDialog({ slug, schedule, trigger }: { slug: string; schedule?: ScheduleItem; trigger: ReactNode }) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [pending, start] = useTransition();
  const [name, setName] = useState(schedule?.name ?? DEFAULT_NAME.WEEKLY);
  const [cadence, setCadence] = useState<"WEEKLY" | "MONTHLY">(schedule?.cadence ?? "WEEKLY");
  const [modules, setModules] = useState<Module[]>(schedule?.modules ?? ["SOCIAL", "SEO", "ADS"]);
  const [withInsight, setWithInsight] = useState(schedule?.withInsight ?? true);

  const changeCadence = (v: string) => {
    const next = v === "MONTHLY" ? "MONTHLY" : "WEEKLY";
    // keep a default name in step with the cadence; a typed name stays
    if (name === DEFAULT_NAME[cadence]) setName(DEFAULT_NAME[next]);
    setCadence(next);
  };
  const toggleModule = (m: Module, on: boolean) =>
    setModules((cur) => (on ? MODULES.map((x) => x.key).filter((k) => k === m || cur.includes(k)) : cur.filter((k) => k !== m)));

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    start(async () => {
      const input = { name, cadence, modules, withInsight };
      const res = schedule ? await updateReportSchedule(slug, schedule.id, input) : await createReportSchedule(slug, input);
      if (res.ok) {
        toast.success(schedule ? aa.schedules.saved : aa.schedules.created);
        setOpen(false);
        router.refresh();
      } else toast.error(res.error);
    });
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={submit} className="space-y-5">
          <DialogHeader>
            <DialogTitle>{schedule ? aa.schedules.edit : aa.schedules.add}</DialogTitle>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="schedule-name">{aa.schedules.name}</Label>
            <Input
              id="schedule-name"
              value={name}
              maxLength={120}
              placeholder={aa.schedules.namePlaceholder}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label>{aa.schedules.cadence}</Label>
            <Select value={cadence} onValueChange={changeCadence}>
              <SelectTrigger size="sm" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {(["WEEKLY", "MONTHLY"] as const).map((c) => (
                  <SelectItem key={c} value={c}>
                    {aa.schedules.cadenceLabel[c]} · {aa.schedules.cadenceDesc[c]}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <fieldset className="space-y-2">
            <legend className="text-sm font-medium">{aa.schedules.modules}</legend>
            <div className="flex flex-wrap gap-x-5 gap-y-2">
              {MODULES.map((m) => (
                <label key={m.key} className="flex items-center gap-2 text-sm">
                  <Checkbox checked={modules.includes(m.key)} onCheckedChange={(v) => toggleModule(m.key, v === true)} />
                  {m.label}
                </label>
              ))}
            </div>
          </fieldset>
          <label className="flex items-center justify-between gap-3 text-sm">
            {aa.schedules.withInsight}
            <Switch checked={withInsight} onCheckedChange={setWithInsight} />
          </label>
          <DialogFooter>
            <Button type="button" variant="ghost" size="sm" onClick={() => setOpen(false)}>
              {aa.schedules.cancel}
            </Button>
            <Button type="submit" size="sm" disabled={pending || modules.length === 0 || !name.trim()}>
              {aa.schedules.save}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
