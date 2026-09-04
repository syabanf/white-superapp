import type { Metadata } from "next";
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

  const [posts, stats] = await Promise.all([getCalendarPosts(client.id, range.from, range.to), getPublishingStats(client.id)]);

  return (
    <>
      <PageHeader eyebrow={t.nav.publish} title={p.calendarTitle} description={`${p.calendarSubtitle} ${p.timezoneNote(tz)}`} />
      {!isPublisherConfigured() ? <DemoBanner message={p.demoBanner} settingsHref={`/clients/${slug}/settings`} /> : null}
      <StatStrip
        items={[
          { label: p.statScheduled7d, value: formatNumber(stats.scheduled7d) },
          { label: p.statInReview, value: formatNumber(stats.inReview) },
          { label: p.statDrafts, value: formatNumber(stats.drafts) },
          { label: p.statPublished30d, value: formatNumber(stats.published30d) },
          { label: p.statFailed, value: formatNumber(stats.failed) },
        ]}
      />
      <CalendarView slug={slug} timezone={tz} view={view} date={date} today={today} days={days} posts={posts} role={role} platform={platform} status={status} />
    </>
  );
}
