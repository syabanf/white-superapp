import type { Metadata } from "next";
import { Palette } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { EmptyState } from "@/components/dashboard/empty-state";
import { DesignTaskList } from "@/features/content/components/design/task-list";
import { ContentOffline } from "@/features/content/components/offline";
import { isDesignStatus } from "@/features/content/lib";
import { contentContext, listDesignTasks, loadContent } from "@/features/content/queries";
import { c } from "@/features/content/strings";
import { todayIn } from "@/features/publishing/time";
import { t } from "@/i18n/id";

export async function generateMetadata(props: PageProps<"/clients/[slug]/content/design">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await contentContext(slug);
  return { title: `${client.name} · ${t.nav.contentDesign}` };
}

const one = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v) ?? "";

export default async function ContentDesignPage(props: PageProps<"/clients/[slug]/content/design">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client, people } = await contentContext(slug);
  const rawStatus = one(sp.status);
  const status = isDesignStatus(rawStatus) ? rawStatus : "";
  const designerId = one(sp.designer);
  const filtered = Boolean(status || designerId);
  const res = await loadContent(() => listDesignTasks(client.id, { status, designerId }));

  return (
    <>
      <PageHeader eyebrow={t.nav.content} title={c.designTitle} description={c.designDesc} />
      {!res.ok ? (
        <ContentOffline message={res.message} />
      ) : res.data.length === 0 && !filtered ? (
        <EmptyState icon={<Palette />} title={c.noTasks} description={c.noTasksDesc} />
      ) : (
        <DesignTaskList slug={slug} tasks={res.data} people={people} status={status} designerId={designerId} today={todayIn(client.timezone)} />
      )}
    </>
  );
}
