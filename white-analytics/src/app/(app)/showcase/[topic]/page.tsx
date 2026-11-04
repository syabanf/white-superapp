import type { ComponentType } from "react";
import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { PageHeader } from "@/components/dashboard/page-header";
import { AdsDemo } from "@/features/showcase/components/ads-demo";
import { DesignDemo } from "@/features/showcase/components/design-demo";
import { IntegrationsDemo } from "@/features/showcase/components/integrations-demo";
import { LeadsDemo } from "@/features/showcase/components/leads-demo";
import { MarketplaceDemo } from "@/features/showcase/components/marketplace-demo";
import { ReportDemo } from "@/features/showcase/components/report-demo";
import { SimulationNotice } from "@/features/showcase/components/shared";
import { TeamDemo } from "@/features/showcase/components/team-demo";
import { WhatsappDemo } from "@/features/showcase/components/whatsapp-demo";
import { SHOWCASE_TOPICS } from "@/features/showcase/topics";
import { requireUser } from "@/lib/rbac";

const DEMOS: Record<string, ComponentType> = {
  whatsapp: WhatsappDemo,
  leads: LeadsDemo,
  iklan: AdsDemo,
  marketplace: MarketplaceDemo,
  desain: DesignDemo,
  tim: TeamDemo,
  report: ReportDemo,
  integrasi: IntegrationsDemo,
};

export async function generateMetadata(props: PageProps<"/showcase/[topic]">): Promise<Metadata> {
  const { topic } = await props.params;
  return { title: SHOWCASE_TOPICS.find((t) => t.id === topic)?.title ?? "Showcase AI" };
}

export default async function ShowcaseTopicPage(props: PageProps<"/showcase/[topic]">) {
  await requireUser();
  const { topic: id } = await props.params;
  const topic = SHOWCASE_TOPICS.find((t) => t.id === id);
  const Demo = DEMOS[id];
  if (!topic || !Demo) notFound();
  return (
    <>
      <PageHeader back={{ href: "/showcase", label: "Showcase AI" }} title={topic.title} description={topic.summary} />
      {/* The integration list states real status, so it is not labelled a simulation. */}
      {id === "integrasi" ? null : <SimulationNotice live={topic.live} />}
      <Demo />
    </>
  );
}
