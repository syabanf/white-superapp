import type { Metadata } from "next";
import { FileText } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { EmptyState } from "@/components/dashboard/empty-state";
import { GenerateReportForm, ReportsTable } from "@/features/content/components/reports/report-list";
import { ContentOffline } from "@/features/content/components/offline";
import { shiftMonth } from "@/features/content/lib";
import { contentContext, listReports, loadContent } from "@/features/content/queries";
import { c } from "@/features/content/strings";
import { todayIn } from "@/features/publishing/time";
import { t } from "@/i18n/id";
import { withClientMode } from "@/lib/client-mode";

export async function generateMetadata(props: PageProps<"/clients/[slug]/content/reports">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await contentContext(slug);
  return { title: `${client.name} · ${t.nav.contentReports}` };
}

async function ContentReportsPageInner(props: PageProps<"/clients/[slug]/content/reports">) {
  const { slug } = await props.params;
  const { client, canWrite } = await contentContext(slug);
  const res = await loadContent(() => listReports(client.id));
  const lastMonth = shiftMonth(todayIn(client.timezone).slice(0, 7), -1);
  const form = canWrite ? <GenerateReportForm slug={slug} defaultMonth={lastMonth} /> : null;

  return (
    <>
      <PageHeader eyebrow={t.nav.content} title={c.reportsTitle} description={c.reportsDesc} actions={res.ok ? form : null} />
      {!res.ok ? (
        <ContentOffline message={res.message} />
      ) : res.data.length === 0 ? (
        <EmptyState icon={<FileText />} title={c.noReports} description={c.noReportsDesc} />
      ) : (
        <ReportsTable slug={slug} reports={res.data} />
      )}
    </>
  );
}

/** Klien demo dirender dalam mode demo: adapter mock + DemoBanner walau kredensial asli terpasang. */
export default async function ContentReportsPage(props: Parameters<typeof ContentReportsPageInner>[0]) {
  const { slug } = await props.params;
  return withClientMode({ slug }, () => ContentReportsPageInner(props));
}
