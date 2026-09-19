import "server-only";
import { addDays, type DateRange } from "@/lib/dates";
import { formatCompact, formatCurrency, formatDateRange, formatDateShort, formatDateTime, formatDeltaNumber, formatDeltaPercent, formatNumber, formatPercent, truncate } from "@/lib/format";
import { contentType, isGoodChange, type Delta } from "@/lib/metrics";
import { getOverviewAds, getOverviewSeo, getOverviewSocial } from "@/features/overview/queries";
import { getCampaignRows, getSocialDailyFollowers, getTopPostRows, getTopQueryRows } from "@/features/reports/queries";
import type { ReportSection } from "@/features/reports/extras";
import { getReportSections } from "@/features/reports/sections";
import { pdfLabels, type PdfLang } from "@/features/reports/strings";
import type { PdfKpi, PdfModuleBlock, PdfSection, ReportPdfData } from "./types";

const OBJECTIVE_LABEL: Record<string, string> = {
  OUTCOME_TRAFFIC: "Traffic",
  OUTCOME_SALES: "Sales",
  OUTCOME_LEADS: "Leads",
  OUTCOME_ENGAGEMENT: "Engagement",
  OUTCOME_AWARENESS: "Awareness",
  OUTCOME_APP_PROMOTION: "App",
};

function kpi(label: string, value: string, delta: Delta | null, opts: { lowerIsBetter?: boolean; suffix?: string } = {}): PdfKpi {
  if (!delta || delta.pct == null) return { label, value, delta: null, deltaGood: null };
  return {
    label,
    value,
    delta: `${formatDeltaPercent(delta.pct)}${opts.suffix ?? ""}`,
    deltaGood: isGoodChange(delta, opts.lowerIsBetter ?? false),
  };
}

const toPdfSections = (sections: ReportSection[]): PdfSection[] =>
  sections.map((s) => ({ title: s.title, kpis: s.kpis.map((k) => kpi(k.label, k.value, k.delta, { lowerIsBetter: k.lowerIsBetter })), table: s.table }));

export type BuildReportPdfArgs = {
  clientId: string;
  clientName: string;
  currency: string;
  title: string;
  range: DateRange;
  previous: DateRange;
  compare: boolean;
  modules: string[];
  language: string;
  aiSummary: string | null;
};

/** Collects and pre-formats everything the PDF document needs (serializable only). */
export async function buildReportPdfData(args: BuildReportPdfArgs): Promise<ReportPdfData> {
  const lang: PdfLang = args.language === "en" ? "en" : "id";
  const L = pdfLabels[lang];
  const { clientId, range, previous, compare, currency } = args;
  const blocks: PdfModuleBlock[] = [];
  const d = (x: Delta) => (compare ? x : null);
  const sections = await getReportSections({ clientId, range, previous, compare, modules: args.modules, lang });
  // A module without channel data still gets its page when an extra block has data.
  const pushSectionsOnly = (key: "SOCIAL" | "SEO", title: string, extra: ReportSection[]) => {
    if (extra.length > 0) blocks.push({ key, title, kpis: [], chart: null, tables: [], sections: toPdfSections(extra) });
  };

  if (args.modules.includes("SOCIAL")) {
    // One extra leading day so the first in-range growth delta has a base.
    const [social, dailyExt, topPosts] = await Promise.all([
      getOverviewSocial(clientId, range, previous),
      getSocialDailyFollowers(clientId, { from: addDays(range.from, -1), to: range.to }),
      getTopPostRows(clientId, range, 7),
    ]);
    // Running totals are near-constant (bars would all be full height) — plot the
    // daily NET GROWTH instead; the zero baseline then carries the story.
    const growth = dailyExt.slice(1).map((p, i) => ({ date: formatDateShort(p.date), value: p.followers - dailyExt[i]!.followers }));
    if (social.hasData) {
      blocks.push({
        key: "SOCIAL",
        title: L.social,
        kpis: [
          kpi("Followers", formatCompact(social.followers), d(social.followersDelta)),
          kpi("Engagement rate", formatPercent(social.engagementRate), d(social.engagementRateDelta)),
          kpi("Jangkauan", formatCompact(social.reach), d(social.reachDelta)),
          kpi("Postingan", formatNumber(social.posts), d(social.postsDelta)),
        ],
        chart:
          growth.length > 1
            ? {
                title: L.dailyFollowers,
                maxLabel: `${formatDeltaNumber(Math.max(...growth.map((p) => p.value)))}${L.perDay}`,
                points: growth,
              }
            : null,
        tables: [
          {
            title: L.topPosts,
            columns: [
              { label: "Platform", width: 14 },
              { label: "Caption", width: 44 },
              { label: "Tipe", width: 12 },
              { label: "Suka", width: 10, align: "right" },
              { label: "Komentar", width: 10, align: "right" },
              { label: "Interaksi", width: 10, align: "right" },
            ],
            rows: topPosts.map((p) => [
              p.platform === "INSTAGRAM" ? "Instagram" : p.platform === "FACEBOOK" ? "Facebook" : "TikTok",
              truncate(p.caption, 64),
              contentType(p),
              formatNumber(p.likes),
              formatNumber(p.comments),
              formatNumber(p.engagements),
            ]),
          },
        ],
        sections: toPdfSections(sections.SOCIAL),
      });
    } else {
      pushSectionsOnly("SOCIAL", L.social, sections.SOCIAL);
    }
  }

  if (args.modules.includes("SEO")) {
    const [seo, topQueries] = await Promise.all([getOverviewSeo(clientId, range, previous), getTopQueryRows(clientId, range, 8)]);
    if (seo.hasData) {
      blocks.push({
        key: "SEO",
        title: L.seo,
        kpis: [
          kpi("Klik", formatCompact(seo.clicks), d(seo.clicksDelta)),
          kpi("Impresi", formatCompact(seo.impressions), d(seo.impressionsDelta)),
          kpi("CTR", formatPercent(seo.ctr), d(seo.ctrDelta)),
          kpi("Posisi rata-rata", formatNumber(seo.position, 1), d(seo.positionDelta), { lowerIsBetter: true }),
          ...(seo.health != null ? [{ label: "Skor kesehatan", value: `${formatNumber(seo.health)}/100`, delta: null, deltaGood: null } satisfies PdfKpi] : []),
        ],
        chart:
          seo.daily.length > 1
            ? {
                title: L.dailyClicks,
                maxLabel: formatCompact(Math.max(...seo.daily.map((p) => p.clicks))),
                points: seo.daily.map((p) => ({ date: formatDateShort(p.date), value: p.clicks })),
              }
            : null,
        tables: [
          {
            title: L.topQueries,
            columns: [
              { label: "Kata kunci", width: 44 },
              { label: "Klik", width: 14, align: "right" },
              { label: "Impresi", width: 14, align: "right" },
              { label: "CTR", width: 14, align: "right" },
              { label: "Posisi", width: 14, align: "right" },
            ],
            rows: topQueries.map((q) => [q.query, formatNumber(q.clicks), formatCompact(q.impressions), formatPercent(q.ctr), formatNumber(q.position, 1)]),
          },
        ],
        sections: toPdfSections(sections.SEO),
      });
    } else {
      pushSectionsOnly("SEO", L.seo, sections.SEO);
    }
  }

  if (args.modules.includes("ADS")) {
    const [ads, campaigns] = await Promise.all([getOverviewAds(clientId, range, previous), getCampaignRows(clientId, range)]);
    if (ads.hasData) {
      blocks.push({
        key: "ADS",
        title: L.ads,
        kpis: [
          kpi("Belanja iklan", formatCurrency(ads.kpis.spend, currency, { compact: true }), d(ads.spendDelta)),
          kpi("Hasil", formatNumber(ads.kpis.results), d(ads.resultsDelta)),
          kpi("Biaya / hasil", formatCurrency(ads.kpis.cpr, currency), d(ads.cprDelta), { lowerIsBetter: true }),
          { label: "CTR", value: formatPercent(ads.kpis.ctr), delta: null, deltaGood: null },
          { label: "CPM", value: formatCurrency(ads.kpis.cpm, currency), delta: null, deltaGood: null },
          ...(ads.kpis.roas != null ? [{ label: "ROAS", value: `${formatNumber(ads.kpis.roas, 2)}x`, delta: null, deltaGood: null } satisfies PdfKpi] : []),
        ],
        chart:
          ads.daily.length > 1
            ? {
                title: L.dailySpend,
                maxLabel: formatCurrency(Math.max(...ads.daily.map((p) => p.spend)), currency, { compact: true }),
                points: ads.daily.map((p) => ({ date: formatDateShort(p.date), value: p.spend })),
              }
            : null,
        tables: [
          {
            title: L.campaigns,
            columns: [
              { label: "Kampanye", width: 38 },
              { label: "Tujuan", width: 14 },
              { label: "Belanja", width: 16, align: "right" },
              { label: "Hasil", width: 12, align: "right" },
              { label: "Biaya/hasil", width: 20, align: "right" },
            ],
            rows: campaigns.map((c) => [
              truncate(c.name, 42),
              OBJECTIVE_LABEL[c.objective] ?? c.objective,
              formatCurrency(c.spend, currency, { compact: true }),
              formatNumber(c.results),
              c.isConversion ? formatCurrency(c.cpr, currency) : "—",
            ]),
          },
        ],
        sections: [],
      });
    }
  }

  return {
    lang,
    title: args.title,
    clientName: args.clientName,
    periodLabel: formatDateRange(range.from, range.to),
    comparedLabel: compare ? `${L.vsPrevious}: ${formatDateRange(previous.from, previous.to)}` : null,
    generatedLabel: formatDateTime(new Date()),
    modules: blocks,
    aiSummary: args.aiSummary,
  };
}
