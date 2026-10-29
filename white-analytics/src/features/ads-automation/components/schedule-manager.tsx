"use client";

import { useTransition } from "react";
import { useRouter } from "next/navigation";
import { CalendarClock, Pencil, Play, Plus, Sparkles, Trash2 } from "lucide-react";
import { toast } from "@/lib/toast";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import { InfoHint } from "@/components/dashboard/info-hint";
import { formatDate, formatRelative } from "@/lib/format";
import { deleteReportSchedule, runReportScheduleNow, toggleReportSchedule } from "@/features/ads-automation/actions";
import type { ScheduleItem } from "@/features/ads-automation/queries";
import { aa } from "@/features/ads-automation/strings";
import { t } from "@/i18n/id";
import { ScheduleFormDialog } from "./schedule-form-dialog";

const MODULE_LABEL: Record<string, string> = { SOCIAL: t.overview.socialCard, SEO: t.overview.seoCard, ADS: t.overview.adsCard };

function useAction() {
  const router = useRouter();
  const [pending, start] = useTransition();
  const run = (fn: () => Promise<{ ok: true } | { ok: false; error: string }>, success: string) =>
    start(async () => {
      const res = await fn();
      if (res.ok) {
        toast.success(success);
        router.refresh();
      } else toast.error(res.error);
    });
  return { pending, run };
}

function ScheduleRow({ slug, s, canManage }: { slug: string; s: ScheduleItem; canManage: boolean }) {
  const { pending, run } = useAction();
  return (
    <li className="flex flex-col gap-3 py-3 first:pt-0 last:pb-0 sm:flex-row sm:items-center">
      <div className="min-w-0 flex-1 space-y-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="truncate text-sm font-medium">{s.name}</span>
          <Badge variant="secondary">{aa.schedules.cadenceLabel[s.cadence]}</Badge>
          {s.withInsight ? <Sparkles className="size-3.5 text-muted-foreground" aria-label={aa.schedules.withInsight} /> : null}
        </div>
        <p className="text-xs text-muted-foreground">
          {s.modules.map((m) => MODULE_LABEL[m] ?? m).join(", ")} ·{" "}
          {s.enabled ? `${aa.schedules.nextRun} ${formatDate(s.nextRunAt)}` : aa.schedules.paused} · {aa.schedules.lastRun}{" "}
          {s.lastRunAt ? formatRelative(s.lastRunAt) : aa.schedules.never}
        </p>
      </div>
      {canManage ? (
        <div className="flex shrink-0 items-center gap-1">
          <Switch
            checked={s.enabled}
            disabled={pending}
            aria-label={aa.schedules.enabled}
            onCheckedChange={(v) =>
              run(() => toggleReportSchedule(slug, s.id, v), v ? aa.schedules.toggledOn : aa.schedules.toggledOff)
            }
          />
          <Button
            variant="ghost"
            size="sm"
            disabled={pending}
            onClick={() => run(() => runReportScheduleNow(slug, s.id), aa.schedules.ran)}
          >
            <Play className="size-4" />
            {pending ? aa.schedules.running : aa.schedules.runNow}
          </Button>
          <ScheduleFormDialog
            slug={slug}
            schedule={s}
            trigger={
              <Button variant="ghost" size="icon-sm" aria-label={aa.schedules.edit} title={aa.schedules.edit}>
                <Pencil className="size-4" />
              </Button>
            }
          />
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button variant="ghost" size="icon-sm" aria-label={aa.schedules.delete} title={aa.schedules.delete} disabled={pending}>
                <Trash2 className="size-4" />
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>{aa.schedules.deleteTitle}</AlertDialogTitle>
                <AlertDialogDescription>{aa.schedules.deleteDesc}</AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>{aa.schedules.cancel}</AlertDialogCancel>
                <AlertDialogAction onClick={() => run(() => deleteReportSchedule(slug, s.id), aa.schedules.deleted)}>
                  {aa.schedules.delete}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
      ) : null}
    </li>
  );
}

/** "Laporan otomatis": recurring report schedules (Reports page). */
export function ScheduleManager({ slug, canManage, schedules }: { slug: string; canManage: boolean; schedules: ScheduleItem[] }) {
  const addButton = (
    <ScheduleFormDialog
      slug={slug}
      trigger={
        <Button size="sm">
          <Plus className="size-4" /> {aa.schedules.add}
        </Button>
      }
    />
  );
  return (
    <Card>
      <CardHeader className="flex min-h-8 flex-row flex-wrap items-center justify-between gap-3 space-y-0">
        <div className="flex min-w-0 items-center gap-1.5">
          <CardTitle className="truncate text-[15px] font-semibold tracking-[-0.015em]">{aa.schedules.title}</CardTitle>
          <InfoHint>{aa.schedules.hint}</InfoHint>
        </div>
        {canManage && schedules.length > 0 ? addButton : null}
      </CardHeader>
      <CardContent className="min-w-0">
        {schedules.length === 0 ? (
          <div className="flex flex-col items-center gap-2 py-4 text-center">
            <span className="flex size-9 items-center justify-center rounded-full bg-muted text-muted-foreground">
              <CalendarClock className="size-4" />
            </span>
            <p className="text-sm font-medium">{aa.schedules.emptyTitle}</p>
            <p className="text-xs text-muted-foreground">{aa.schedules.emptyDesc}</p>
            {canManage ? <div className="pt-1">{addButton}</div> : null}
          </div>
        ) : (
          <ul className="divide-y">
            {schedules.map((s) => (
              <ScheduleRow key={s.id} slug={slug} s={s} canManage={canManage} />
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
