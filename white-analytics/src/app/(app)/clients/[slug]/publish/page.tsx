import type { Metadata } from "next";
import Link from "next/link";
import { PageHeader } from "@/components/dashboard/page-header";
import { StatStrip } from "@/components/dashboard/stat-strip";
import { DemoBanner } from "@/components/dashboard/banners";
import { CalendarView } from "@/features/publishing/components/calendar/calendar-view";
import { isPlatform, isPostStatus } from "@/features/publishing/lib";
import { getCalendarPosts, getPublishingStats } from "@/features/publishing/queries";
import { monthGrid, parseDateKey, rangeForDays, todayIn, weekDays } from "@/features/publishing/time";
import { p } from "@/features/publishing/strings";
import { isPublisherConfigured } from "@/lib/providers/social-publisher";
import { requireClientAccess } from "@/lib/rbac";
import { formatNumber } from "@/lib/format";
import { t } from "@/i18n/id";

export async function generateMetadata(props: PageProps<"/clients/[slug]/publish">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.nav.calendar}` };
}

const one = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v);

export default async function PublishCalendarPage(props: PageProps<"/clients/[slug]/publish">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client, role } = await requireClientAccess(slug);
  const tz = client.timezone;
  const today = todayIn(tz);
  const view = one(sp.view) === "week" ? "week" : "month";
  const date = parseDateKey(sp.date, today);
  const platformRaw = one(sp.platform)?.toUpperCase();
  const platform = platformRaw && isPlatform(platformRaw) ? platformRaw : "ALL";
  const statusRaw = one(sp.status)?.toUpperCase();
  const status = isPostStatus(statusRaw) ? statusRaw : "ALL";
  const days = view === "month" ? monthGrid(date) : [weekDays(date)];
  const range = rangeForDays(days.flat(), tz);

  const [posts, stats] = await Promise.all([
    getCalendarPosts(client.id, range.from, range.to),
    getPublishingStats(client.id),
  ]);

  return (
    <>
      <PageHeader
        eyebrow={t.nav.publish}
        title={p.calendarTitle}
        description={p.timezoneNote(tz)}
      />
      {!isPublisherConfigured() ? (
        <DemoBanner message={p.demoBanner} settingsHref={`/clients/${slug}/settings`} />
      ) : null}
      {stats.inReview > 0 || stats.failed > 0 || stats.overdue > 0 ? (
        <div className="flex flex-wrap items-center gap-2 rounded-xl border border-amber-300/50 bg-amber-50/70 px-4 py-3 text-sm dark:bg-amber-950/20">
          <span className="font-medium">Perlu ditindaklanjuti:</span>
          {stats.failed > 0 ? <Link className="underline underline-offset-2" href={`/clients/${slug}/publish/posts?status=FAILED`}>{stats.failed} gagal</Link> : null}
          {stats.inReview > 0 ? <Link className="underline underline-offset-2" href={`/clients/${slug}/publish/posts?status=IN_REVIEW`}>{stats.inReview} menunggu review</Link> : null}
          {stats.overdue > 0 ? <Link className="underline underline-offset-2" href={`/clients/${slug}/publish/posts?status=SCHEDULED&window=overdue`}>{stats.overdue} jadwal terlewat</Link> : null}
        </div>
      ) : null}
      <StatStrip
        scrollOnMobile
        mobileScrollHint={p.scrollMetricsHint}
        minWidth={148}
        items={[
          { label: p.statScheduled7d, value: formatNumber(stats.scheduled7d), href: `/clients/${slug}/publish/posts?status=SCHEDULED&window=next7d` },
          { label: p.statInReview, value: formatNumber(stats.inReview), href: `/clients/${slug}/publish/posts?status=IN_REVIEW` },
          { label: p.statDrafts, value: formatNumber(stats.drafts), href: `/clients/${slug}/publish/posts?status=DRAFT` },
          { label: p.statPublished30d, value: formatNumber(stats.published30d), href: `/clients/${slug}/publish/posts?status=PUBLISHED&window=last30d` },
          { label: p.statFailed, value: formatNumber(stats.failed), href: `/clients/${slug}/publish/posts?status=FAILED` },
        ]}
      />
      <CalendarView
        slug={slug}
        timezone={tz}
        view={view}
        date={date}
        today={today}
        days={days}
        posts={posts}
        role={role}
        platform={platform}
        status={status}
      />
    </>
  );
}
