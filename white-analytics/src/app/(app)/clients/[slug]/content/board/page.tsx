import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { ContentBoard } from "@/features/content/components/board/board";
import { ContentOffline } from "@/features/content/components/offline";
import { contentContext, listItems, listPlans, loadContent } from "@/features/content/queries";
import { c } from "@/features/content/strings";
import { todayIn } from "@/features/publishing/time";
import { t } from "@/i18n/id";

export async function generateMetadata(props: PageProps<"/clients/[slug]/content/board">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await contentContext(slug);
  return { title: `${client.name} · ${t.nav.contentBoard}` };
}

const one = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v) ?? "";

export default async function ContentBoardPage(props: PageProps<"/clients/[slug]/content/board">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client, canWrite, people } = await contentContext(slug);
  const assigneeId = one(sp.assignee);
  const today = todayIn(client.timezone);
  // Two years of plans make an unfiltered board unreadable, so it opens on
  // this month's plan (or the latest one); "plan=all" shows everything.
  const res = await loadContent(async () => {
    const plans = await listPlans(client.id);
    const requested = one(sp.plan);
    const planId =
      requested === "all"
        ? ""
        : requested ||
          (plans.find((p) => p.month === today.slice(0, 7)) ?? [...plans].sort((a, b) => b.month.localeCompare(a.month))[0])?.id ||
          "";
    return [plans, await listItems(client.id, { planId, assigneeId }), planId] as const;
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
          assigneeId={assigneeId}
          today={today}
          canWrite={canWrite}
        />
      )}
    </>
  );
}
