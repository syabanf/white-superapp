"use client";

import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { ChevronLeft, ChevronRight, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { PlatformIcon, platformLabel } from "@/features/social/components/platform-icon";
import { PLATFORMS, STATUS_ORDER, type Platform, type PostStatus } from "@/features/publishing/lib";
import { statusLabel } from "@/features/publishing/components/post-status";
import { dayLabel, monthLabel, shiftDays, shiftMonths, weekDays } from "@/features/publishing/time";
import { p } from "@/features/publishing/strings";
import { t } from "@/i18n/id";

export type CalendarView = "month" | "week";

export function CalendarToolbar({
  view,
  date,
  today,
  platform,
  status,
  composeHref,
}: {
  view: CalendarView;
  date: string;
  today: string;
  platform: Platform | "ALL";
  status: PostStatus | "ALL";
  composeHref: string;
}) {
  const pathname = usePathname();
  const sp = useSearchParams();
  const router = useRouter();

  const hrefWith = (patch: Record<string, string | null>) => {
    const q = new URLSearchParams(sp.toString());
    for (const [k, v] of Object.entries(patch)) {
      if (v == null || v === "ALL") q.delete(k);
      else q.set(k, v);
    }
    const qs = q.toString();
    return qs ? `${pathname}?${qs}` : pathname;
  };
  const step = (n: number) => (view === "month" ? shiftMonths(date, n) : shiftDays(date, 7 * n));
  const days = weekDays(date);
  const title = view === "month" ? monthLabel(date) : `${dayLabel(days[0]!)} – ${dayLabel(days[6]!, true)}`;

  return (
    <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
      <div className="flex items-center gap-2">
        <Button asChild variant="outline" size="icon-sm" aria-label={p.prev}>
          <Link href={hrefWith({ date: step(-1) })} scroll={false}>
            <ChevronLeft />
          </Link>
        </Button>
        <Button asChild variant="outline" size="sm">
          <Link href={hrefWith({ date: today })} scroll={false}>
            {p.today}
          </Link>
        </Button>
        <Button asChild variant="outline" size="icon-sm" aria-label={p.next}>
          <Link href={hrefWith({ date: step(1) })} scroll={false}>
            <ChevronRight />
          </Link>
        </Button>
        <h2 className="ml-1 truncate text-base font-semibold tracking-[-0.02em] md:text-lg">{title}</h2>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Tabs value={view}>
          <TabsList className="h-8">
            <TabsTrigger value="month" asChild>
              <Link href={hrefWith({ view: null })} scroll={false}>
                {p.month}
              </Link>
            </TabsTrigger>
            <TabsTrigger value="week" asChild>
              <Link href={hrefWith({ view: "week" })} scroll={false}>
                {p.week}
              </Link>
            </TabsTrigger>
          </TabsList>
        </Tabs>
        <Select value={platform} onValueChange={(v) => router.replace(hrefWith({ platform: v }), { scroll: false })}>
          <SelectTrigger size="sm" className="w-[150px]" aria-label={p.allPlatforms}>
            {/* Explicit children so the label is server-rendered (Radix fills it only after hydration). */}
            <SelectValue>{platform === "ALL" ? p.allPlatforms : platformLabel(platform as Platform)}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="ALL">{p.allPlatforms}</SelectItem>
            {PLATFORMS.map((pl) => (
              <SelectItem key={pl} value={pl}>
                <PlatformIcon platform={pl} className="size-3.5" /> {platformLabel(pl)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={status} onValueChange={(v) => router.replace(hrefWith({ status: v }), { scroll: false })}>
          <SelectTrigger size="sm" className="w-[160px]" aria-label={t.common.status}>
            <SelectValue>{status === "ALL" ? p.allStatuses : statusLabel(status as PostStatus)}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="ALL">{p.allStatuses}</SelectItem>
            {STATUS_ORDER.map((s) => (
              <SelectItem key={s} value={s}>
                {statusLabel(s)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button asChild size="sm">
          <Link href={composeHref}>
            <Plus className="size-3.5" /> {t.nav.compose}
          </Link>
        </Button>
      </div>
    </div>
  );
}
