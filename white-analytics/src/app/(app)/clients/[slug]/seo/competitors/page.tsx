import type { Metadata } from "next";
import Link from "next/link";
import { Swords } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { DemoBanner } from "@/components/dashboard/banners";
import { EmptyState } from "@/components/dashboard/empty-state";
import { requireClientAccess } from "@/lib/rbac";
import { getRange } from "@/lib/range-params";
import { formatDateRange } from "@/lib/format";
import { t } from "@/i18n/id";
import { s } from "@/features/seo-suite/strings";
import { getCompetitorData, getSuiteProperty, isSuiteDemo, lastJobAt } from "@/features/seo-suite/queries";
import { CompetitorManager } from "@/features/seo-suite/components/competitor-manager";
import { BenchmarkTable, TrafficChart } from "@/features/seo-suite/components/competitor-charts";
import { RefreshButton } from "@/features/seo-suite/components/refresh-button";

export async function generateMetadata(props: PageProps<"/clients/[slug]/seo/competitors">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.nav.seoCompetitors}` };
}

export default async function SeoCompetitorsPage(props: PageProps<"/clients/[slug]/seo/competitors">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client, canManage } = await requireClientAccess(slug);
  const { range } = getRange(sp);

  const property = await getSuiteProperty(client.id);
  if (!property) {
    return (
      <>
        <PageHeader eyebrow={t.nav.seo} title={t.nav.seoCompetitors} description={s.competitorsSubtitle} />
        <EmptyState
          icon={<Swords />}
          title={t.seo.noProperty}
          description={t.seo.noPropertyDesc}
          action={
            <Button asChild size="sm">
              <Link href={`/clients/${slug}/settings`}>{t.seo.addProperty}</Link>
            </Button>
          }
        />
      </>
    );
  }

  const [demo, data, lastRun] = await Promise.all([isSuiteDemo(), getCompetitorData(property, range), lastJobAt(client.id, ["SEO_SUITE_DOMAINS", "SEO_SUITE_DAILY"])]);
  const gapHref = `/clients/${slug}/seo/research`;

  return (
    <>
      <PageHeader
        eyebrow={t.nav.seo}
        title={t.nav.seoCompetitors}
        description={`${property.host} · ${formatDateRange(range.from, range.to)}`}
        actions={<RefreshButton kind="domains" clientId={client.id} propertyId={property.id} lastRunAt={lastRun} />}
      />
      {demo ? <DemoBanner message={s.demoBanner} /> : null}

      <CompetitorManager clientId={client.id} propertyId={property.id} ownDomain={data.ownDomain} competitors={data.competitors} suggestions={data.suggestions} canManage={canManage} />

      {data.competitors.length === 0 ? (
        <EmptyState icon={<Swords />} title={s.noCompetitorsTitle} description={s.noCompetitorsDesc} />
      ) : (
        <>
          <Section index="01" title={s.benchmarkTitle} description={s.benchmarkDesc}>
            <BenchmarkTable rows={data.benchmark} gapHref={gapHref} />
          </Section>
          <Section index="02" title={s.trafficChart}>
            <TrafficChart traffic={data.traffic} />
          </Section>
        </>
      )}
    </>
  );
}
