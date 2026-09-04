import type { Metadata } from "next";
import Link from "next/link";
import { Lightbulb } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { EmptyState } from "@/components/dashboard/empty-state";
import { DemoBanner } from "@/components/dashboard/banners";
import { requireClientAccess } from "@/lib/rbac";
import { normalizeDomain } from "@/lib/metrics/seo-suite";
import { t } from "@/i18n/id";
import { s } from "@/features/seo-suite/strings";
import { normalizeKeyword, normalizeMode } from "@/features/seo-suite/lib";
import { getCompetitorDomains, getKeywordGap, getResearch, getSuiteProperty, getTrackedKeywordSet, isSuiteDemo } from "@/features/seo-suite/queries";
import { ResearchForm } from "@/features/seo-suite/components/research-form";
import { ResearchResults } from "@/features/seo-suite/components/research-results";
import { KeywordGapSection } from "@/features/seo-suite/components/keyword-gap";

export async function generateMetadata(props: PageProps<"/clients/[slug]/seo/research">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.nav.research}` };
}

const one = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v) ?? "";

export default async function SeoResearchPage(props: PageProps<"/clients/[slug]/seo/research">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client, canManage } = await requireClientAccess(slug);
  const basePath = `/clients/${slug}/seo/research`;
  const seed = normalizeKeyword(one(sp.seed)).slice(0, 120);
  const mode = normalizeMode(one(sp.mode));
  const gapDomain = normalizeDomain(one(sp.gap));

  const property = await getSuiteProperty(client.id);
  if (!property) {
    return (
      <>
        <PageHeader eyebrow={t.nav.seo} title={t.nav.research} description={s.researchSubtitle} />
        <EmptyState
          icon={<Lightbulb />}
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

  const [demo, competitors, tracked] = await Promise.all([isSuiteDemo(), getCompetitorDomains(property.id), getTrackedKeywordSet(property.id)]);
  const selectedGap = gapDomain && competitors.some((c) => c.domain === gapDomain) ? gapDomain : null;
  const [research, gap] = await Promise.all([
    seed ? getResearch(client.id, seed, mode, property.locationCode, property.languageCode) : null,
    selectedGap ? getKeywordGap(property.host, selectedGap, property.locationCode, property.languageCode) : null,
  ]);

  return (
    <>
      <PageHeader eyebrow={t.nav.seo} title={t.nav.research} description={`${s.researchSubtitle} · ${property.host}`} />
      {demo ? <DemoBanner message={s.demoBanner} /> : null}

      <Section index="01" title={t.nav.research}>
        <ResearchForm seed={seed} mode={mode} basePath={basePath} gap={selectedGap ?? undefined} />
        {research ? (
          <ResearchResults
            clientId={client.id}
            propertyId={property.id}
            ideas={research.ideas}
            tracked={tracked}
            fetchedAt={research.fetchedAt}
            fromCache={research.fromCache}
            canManage={canManage}
            exportName={`riset-${mode}-${seed.replace(/\s+/g, "-")}`}
          />
        ) : (
          <EmptyState icon={<Lightbulb />} title={s.researchEmptyTitle} description={s.researchEmptyDesc} />
        )}
      </Section>

      <Section index="02" title={s.gapTitle} description={s.gapDesc}>
        <KeywordGapSection competitors={competitors} selected={selectedGap} ownDomain={property.host} gap={gap} competitorsHref={`/clients/${slug}/seo/competitors`} />
      </Section>
    </>
  );
}
