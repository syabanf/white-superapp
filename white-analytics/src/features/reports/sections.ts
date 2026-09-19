import "server-only";
import type { DateRange } from "@/lib/dates";
import { getBacklinkData, getRankData, getSuiteProperty } from "@/features/seo-suite/queries";
import { buildBacklinkSection, buildPublishingSection, buildRankSection, type ReportSection } from "@/features/reports/extras";
import { getPublishingSummary } from "@/features/reports/queries";
import { pdfLabels, type PdfLang } from "@/features/reports/strings";

export type ReportSections = { SOCIAL: ReportSection[]; SEO: ReportSection[] };

/**
 * Extra blocks per parent module, shared by the PDF and the builder preview.
 * A block is present only when its module is selected and the client has data for it.
 */
export async function getReportSections(args: {
  clientId: string;
  range: DateRange;
  previous: DateRange;
  compare: boolean;
  modules: string[];
  lang: PdfLang;
}): Promise<ReportSections> {
  const L = pdfLabels[args.lang];
  const [publishing, seo] = await Promise.all([
    args.modules.includes("SOCIAL") ? getPublishingSummary(args.clientId, args.range) : null,
    args.modules.includes("SEO") ? getSeoSuiteData(args.clientId, args.range, args.previous) : null,
  ]);
  const present = (list: (ReportSection | null)[]) => list.filter((x): x is ReportSection => x != null);
  return {
    SOCIAL: present([publishing ? buildPublishingSection(publishing, L) : null]),
    SEO: present(seo ? [buildRankSection(seo.rank, args.compare, L), buildBacklinkSection(seo.backlinks, L)] : []),
  };
}

async function getSeoSuiteData(clientId: string, range: DateRange, previous: DateRange) {
  const property = await getSuiteProperty(clientId);
  if (!property) return null;
  const [rank, backlinks] = await Promise.all([getRankData(property.id, range, previous), getBacklinkData(property.id, range)]);
  return { rank, backlinks };
}
