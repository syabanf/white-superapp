import type { Metadata } from "next";
import { ClipboardList } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { EmptyState } from "@/components/dashboard/empty-state";
import { CreatePlanButton, PlansTable } from "@/features/content/components/plans/plan-list";
import { ContentOffline } from "@/features/content/components/offline";
import { shiftMonth } from "@/features/content/lib";
import { contentContext, listPlans, loadContent } from "@/features/content/queries";
import { c } from "@/features/content/strings";
import { todayIn } from "@/features/publishing/time";
import { t } from "@/i18n/id";
import { withClientMode } from "@/lib/client-mode";

export async function generateMetadata(props: PageProps<"/clients/[slug]/content/plans">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await contentContext(slug);
  return { title: `${client.name} · ${t.nav.contentPlans}` };
}

async function ContentPlansPageInner(props: PageProps<"/clients/[slug]/content/plans">) {
  const { slug } = await props.params;
  const { client, canWrite } = await contentContext(slug);
  const res = await loadContent(() => listPlans(client.id));
  const thisMonth = todayIn(client.timezone).slice(0, 7);
  const months = res.ok ? new Set(res.data.map((p) => p.month)) : new Set<string>();
  const suggested = months.has(thisMonth) ? shiftMonth(thisMonth, 1) : thisMonth;
  const create = canWrite ? <CreatePlanButton slug={slug} defaultMonth={suggested} /> : null;

  return (
    <>
      <PageHeader eyebrow={t.nav.content} title={c.plansTitle} description={c.plansDesc} actions={res.ok ? create : null} />
      {!res.ok ? (
        <ContentOffline message={res.message} />
      ) : res.data.length === 0 ? (
        <EmptyState icon={<ClipboardList />} title={c.noPlans} description={c.noPlansDesc} action={create ?? undefined} />
      ) : (
        <PlansTable plans={res.data} slug={slug} />
      )}
    </>
  );
}

/** Klien demo dirender dalam mode demo: adapter mock + DemoBanner walau kredensial asli terpasang. */
export default async function ContentPlansPage(props: Parameters<typeof ContentPlansPageInner>[0]) {
  const { slug } = await props.params;
  return withClientMode({ slug }, () => ContentPlansPageInner(props));
}
