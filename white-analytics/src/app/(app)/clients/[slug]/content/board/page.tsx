import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { ContentBoard } from "@/features/content/components/board/board";
import { ContentOffline } from "@/features/content/components/offline";
import { contentContext, listItems, listPlans, loadContent } from "@/features/content/queries";
import { hasBoardFilters, matchItem, parseBoardFilters } from "@/features/content/board-filters";
import { c } from "@/features/content/strings";
import { todayIn } from "@/features/publishing/time";
import { t } from "@/i18n/id";
import { withClientMode } from "@/lib/client-mode";

export async function generateMetadata(props: PageProps<"/clients/[slug]/content/board">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await contentContext(slug);
  return { title: `${client.name} · ${t.nav.contentBoard}` };
}

const one = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v) ?? "";

async function ContentBoardPageInner(props: PageProps<"/clients/[slug]/content/board">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client, canWrite, people } = await contentContext(slug);
  const assigneeId = one(sp.assignee);
  const today = todayIn(client.timezone);
  const filters = parseBoardFilters(sp);
  const filtered = hasBoardFilters(filters);
  // Two years of plans make an unfiltered board unreadable, so it opens on
  // this month's plan (or the latest one); "plan=all" shows everything, and so
  // does any drill filter (status, stage, date...) without an explicit plan.
  const res = await loadContent(async () => {
    const plans = await listPlans(client.id);
    const requested = one(sp.plan);
    const planId =
      requested === "all" || (!requested && filtered)
        ? ""
        : requested ||
          (plans.find((p) => p.month === today.slice(0, 7)) ?? [...plans].sort((a, b) => b.month.localeCompare(a.month))[0])?.id ||
          "";
    const items = await listItems(client.id, { planId, assigneeId });
    return [plans, filtered ? items.filter((it) => matchItem(it, filters, today)) : items, planId, items.length] as const;
  });

  return (
    <>
      <PageHeader eyebrow={t.nav.content} title={c.boardTitle} description={c.boardDesc} />
      {!res.ok ? (
        <ContentOffline message={res.message} />
      ) : (
        <ContentBoard
          slug={slug}
          clientId={client.id}
          plans={res.data[0]}
          items={res.data[1]}
          people={people}
          planId={res.data[2]}
          filters={filters}
          totalItems={res.data[3]}
          assigneeId={assigneeId}
          today={today}
          canWrite={canWrite}
        />
      )}
    </>
  );
}

/** Klien demo dirender dalam mode demo: adapter mock + DemoBanner walau kredensial asli terpasang. */
export default async function ContentBoardPage(props: Parameters<typeof ContentBoardPageInner>[0]) {
  const { slug } = await props.params;
  return withClientMode({ slug }, () => ContentBoardPageInner(props));
}
