import { formatCompact, formatCurrency, formatNumber, formatPercent } from "@/lib/format";
import type { GenerateInsightInput, InsightAdsData, InsightPayload, InsightProvider, InsightSeoData, InsightSocialData } from "./types";

/**
 * Deterministic template adapter. Fills the exact numbers from the payload so the
 * output reads like a real strategist's analysis — same input, same output.
 * Section headings match the real adapter's contract (always Indonesian).
 */

const trend = (pct: number | null, lang: "id" | "en") => {
  if (pct == null) return lang === "id" ? "stabil" : "flat";
  if (pct > 1) return lang === "id" ? `naik ${formatPercent(pct, 1)}` : `up ${formatPercent(pct, 1)}`;
  if (pct < -1) return lang === "id" ? `turun ${formatPercent(Math.abs(pct), 1)}` : `down ${formatPercent(Math.abs(pct), 1)}`;
  return lang === "id" ? "relatif stabil" : "roughly flat";
};

function socialLines(s: InsightSocialData, lang: "id" | "en"): { highlight: string[]; fix: string[]; rec: string[] } {
  const best = s.topPosts[0];
  const id = {
    highlight: [
      `**Followers** kini ${formatCompact(s.followers)} (${trend(s.followerGrowthPct, lang)} dengan tambahan ${formatNumber(s.followerGrowth)} followers).`,
      `**Engagement rate** rata-rata ${formatPercent(s.engagementRate)} dari ${formatNumber(s.posts)} postingan; jangkauan ${trend(s.reachDeltaPct, lang)} di ${formatCompact(s.reach)}.`,
      ...(best ? [`Konten terbaik: "${best.caption}" (${best.platform}, ${best.type}) dengan ${formatCompact(best.engagements)} interaksi.`] : []),
    ],
    fix: [
      (s.engagementRateDeltaPct ?? 0) < 0
        ? `Engagement rate melemah ${formatPercent(Math.abs(s.engagementRateDeltaPct ?? 0), 1)} — kualitas interaksi perlu jadi fokus, bukan sekadar frekuensi posting.`
        : `Pertahankan momentum engagement; ruang tumbuh masih ada di format video pendek.`,
      (s.reachDeltaPct ?? 0) < 0 ? `Jangkauan menurun — distribusi konten terlalu bergantung pada followers existing.` : `Perluas jangkauan non-follower lewat kolaborasi dan konten shareable.`,
    ],
    rec: [
      `Replikasi pola konten terbaik${best ? ` ("${best.caption.slice(0, 40)}…")` : ""} minimal 2x per minggu.`,
      `Naikkan rasio konten video/Reels untuk mendongkrak jangkauan organik.`,
    ],
  };
  const en = {
    highlight: [
      `**Followers** now at ${formatCompact(s.followers)} (${trend(s.followerGrowthPct, lang)}, +${formatNumber(s.followerGrowth)} net).`,
      `Average **engagement rate** ${formatPercent(s.engagementRate)} across ${formatNumber(s.posts)} posts; reach ${trend(s.reachDeltaPct, lang)} at ${formatCompact(s.reach)}.`,
      ...(best ? [`Best content: "${best.caption}" (${best.platform}, ${best.type}) with ${formatCompact(best.engagements)} engagements.`] : []),
    ],
    fix: [
      (s.engagementRateDeltaPct ?? 0) < 0
        ? `Engagement rate softened ${formatPercent(Math.abs(s.engagementRateDeltaPct ?? 0), 1)} — prioritise interaction quality over posting volume.`
        : `Keep the engagement momentum; short-form video still has headroom.`,
      (s.reachDeltaPct ?? 0) < 0 ? `Reach declined — distribution relies too heavily on the existing follower base.` : `Expand non-follower reach through collaborations and shareable formats.`,
    ],
    rec: [
      `Replicate the top-performing content pattern${best ? ` ("${best.caption.slice(0, 40)}…")` : ""} at least twice a week.`,
      `Increase the share of video/Reels to lift organic reach.`,
    ],
  };
  return lang === "id" ? id : en;
}

function seoLines(s: InsightSeoData, lang: "id" | "en"): { highlight: string[]; fix: string[]; rec: string[]; keywords: string[] } {
  const top = s.topQueries[0];
  const opp = s.topQueries.filter((q) => q.position > 3);
  const id = {
    highlight: [
      `**Klik organik** ${formatCompact(s.clicks)} (${trend(s.clicksDeltaPct, lang)}) dari ${formatCompact(s.impressions)} impresi; CTR ${formatPercent(s.ctr)}.`,
      `Posisi rata-rata ${formatNumber(s.position, 1)}${s.health != null ? `; skor kesehatan situs ${formatNumber(s.health)}/100` : ""}.`,
      ...(top ? [`Kata kunci terkuat: "${top.query}" — ${formatNumber(top.clicks)} klik di posisi ${formatNumber(top.position, 1)}.`] : []),
    ],
    fix: [
      (s.clicksDeltaPct ?? 0) < 0 ? `Klik organik menurun — audit konten yang kehilangan peringkat dan segarkan title/meta.` : `Jaga tren klik dengan memperbarui konten pilar secara berkala.`,
      s.positionDeltaAbs > 0 ? `Posisi rata-rata melemah ${formatNumber(Math.abs(s.positionDeltaAbs), 1)} poin — periksa halaman yang tergeser kompetitor.` : `Perkuat internal linking ke halaman komersial utama.`,
    ],
    rec: [
      ...(top ? [`Optimalkan halaman untuk "${top.query}" agar bertahan di posisi teratas.`] : []),
      `Perbaiki title & meta description pada halaman ber-CTR rendah di halaman 1.`,
    ],
    keywords: opp.slice(0, 5).map((q) => `**"${q.query}"** — posisi ${formatNumber(q.position, 1)}, ${formatCompact(q.impressions)} impresi: satu dorongan konten lagi menuju top 3.`),
  };
  const en = {
    highlight: [
      `**Organic clicks** ${formatCompact(s.clicks)} (${trend(s.clicksDeltaPct, lang)}) from ${formatCompact(s.impressions)} impressions; CTR ${formatPercent(s.ctr)}.`,
      `Average position ${formatNumber(s.position, 1)}${s.health != null ? `; site health score ${formatNumber(s.health)}/100` : ""}.`,
      ...(top ? [`Strongest query: "${top.query}" — ${formatNumber(top.clicks)} clicks at position ${formatNumber(top.position, 1)}.`] : []),
    ],
    fix: [
      (s.clicksDeltaPct ?? 0) < 0 ? `Organic clicks declined — audit pages losing rank and refresh titles/meta.` : `Protect the click trend by refreshing pillar content regularly.`,
      s.positionDeltaAbs > 0 ? `Average position slipped ${formatNumber(Math.abs(s.positionDeltaAbs), 1)} points — review pages displaced by competitors.` : `Strengthen internal links into key commercial pages.`,
    ],
    rec: [
      ...(top ? [`Defend the top position for "${top.query}" with content upkeep.`] : []),
      `Rewrite titles & meta descriptions on page-1 URLs with below-expected CTR.`,
    ],
    keywords: opp.slice(0, 5).map((q) => `**"${q.query}"** — position ${formatNumber(q.position, 1)}, ${formatCompact(q.impressions)} impressions: one content push away from top 3.`),
  };
  return lang === "id" ? id : en;
}

function adsLines(a: InsightAdsData, lang: "id" | "en"): { highlight: string[]; fix: string[]; rec: string[]; budget: string[] } {
  const cur = a.currency || "IDR";
  const sorted = [...a.campaigns].sort((x, y) => (x.cpr || Infinity) - (y.cpr || Infinity));
  const best = sorted.find((c) => c.results > 0 && c.cpr > 0);
  const worst = [...sorted].reverse().find((c) => c.results > 0 && c.cpr > 0 && c !== best);
  const totalSpend = a.campaigns.reduce((s, c) => s + c.spend, 0) || a.spend;
  const id = {
    highlight: [
      `**Belanja iklan** ${formatCurrency(a.spend, cur, { compact: true })} (${trend(a.spendDeltaPct, lang)}) menghasilkan ${formatNumber(a.results)} hasil (${trend(a.resultsDeltaPct, lang)}).`,
      `Biaya per hasil ${formatCurrency(a.cpr, cur)} (${trend(a.cprDeltaPct, lang)}); CTR ${formatPercent(a.ctr)}, CPM ${formatCurrency(a.cpm, cur)}${a.roas != null ? `, ROAS ${formatNumber(a.roas, 2)}x` : ""}.`,
      ...(best ? [`Kampanye paling efisien: **${best.name}** — CPR ${formatCurrency(best.cpr, cur)} untuk ${formatNumber(best.results)} hasil.`] : []),
    ],
    fix: [
      (a.cprDeltaPct ?? 0) > 0 ? `Biaya per hasil naik ${formatPercent(a.cprDeltaPct ?? 0, 1)} — segarkan materi iklan sebelum fatigue menggerus efisiensi.` : `Efisiensi biaya terjaga; uji kreatif baru agar tidak stagnan.`,
      ...(worst ? [`Kampanye **${worst.name}** paling boros (CPR ${formatCurrency(worst.cpr, cur)}) — evaluasi targeting dan penawarannya.`] : []),
    ],
    rec: [
      ...(best ? [`Geser anggaran bertahap ke kampanye "${best.name}" yang terbukti paling efisien.`] : []),
      `Siapkan 2–3 variasi kreatif baru untuk rotasi bulan depan.`,
    ],
    budget: a.campaigns.slice(0, 5).map((c) => {
      const share = totalSpend > 0 ? (c.spend / totalSpend) * 100 : 0;
      return `**${c.name}** — ${formatCurrency(c.spend, cur, { compact: true })} (${formatPercent(share, 1)} dari total), ${formatNumber(c.results)} hasil${c.results > 0 && c.cpr > 0 ? `, CPR ${formatCurrency(c.cpr, cur)}` : ""}.`;
    }),
  };
  const en = {
    highlight: [
      `**Ad spend** ${formatCurrency(a.spend, cur, { compact: true })} (${trend(a.spendDeltaPct, lang)}) produced ${formatNumber(a.results)} results (${trend(a.resultsDeltaPct, lang)}).`,
      `Cost per result ${formatCurrency(a.cpr, cur)} (${trend(a.cprDeltaPct, lang)}); CTR ${formatPercent(a.ctr)}, CPM ${formatCurrency(a.cpm, cur)}${a.roas != null ? `, ROAS ${formatNumber(a.roas, 2)}x` : ""}.`,
      ...(best ? [`Most efficient campaign: **${best.name}** — CPR ${formatCurrency(best.cpr, cur)} for ${formatNumber(best.results)} results.`] : []),
    ],
    fix: [
      (a.cprDeltaPct ?? 0) > 0 ? `Cost per result rose ${formatPercent(a.cprDeltaPct ?? 0, 1)} — refresh creatives before fatigue erodes efficiency.` : `Cost efficiency holding; keep testing new creatives to avoid stagnation.`,
      ...(worst ? [`Campaign **${worst.name}** is the least efficient (CPR ${formatCurrency(worst.cpr, cur)}) — review targeting and offer.`] : []),
    ],
    rec: [
      ...(best ? [`Gradually shift budget toward "${best.name}", the proven efficiency leader.`] : []),
      `Prepare 2–3 new creative variations for next month's rotation.`,
    ],
    budget: a.campaigns.slice(0, 5).map((c) => {
      const share = totalSpend > 0 ? (c.spend / totalSpend) * 100 : 0;
      return `**${c.name}** — ${formatCurrency(c.spend, cur, { compact: true })} (${formatPercent(share, 1)} of total), ${formatNumber(c.results)} results${c.results > 0 && c.cpr > 0 ? `, CPR ${formatCurrency(c.cpr, cur)}` : ""}.`;
    }),
  };
  return lang === "id" ? id : en;
}

function summarize(data: InsightPayload, clientName: string, days: number, lang: "id" | "en"): string {
  const bits: string[] = [];
  if (data.social) bits.push(lang === "id" ? `${formatCompact(data.social.followers)} followers (${trend(data.social.followerGrowthPct, lang)})` : `${formatCompact(data.social.followers)} followers (${trend(data.social.followerGrowthPct, lang)})`);
  if (data.seo) bits.push(lang === "id" ? `${formatCompact(data.seo.clicks)} klik organik (${trend(data.seo.clicksDeltaPct, lang)})` : `${formatCompact(data.seo.clicks)} organic clicks (${trend(data.seo.clicksDeltaPct, lang)})`);
  if (data.ads) bits.push(lang === "id" ? `belanja iklan ${formatCurrency(data.ads.spend, data.ads.currency, { compact: true })} dengan CPR ${trend(data.ads.cprDeltaPct == null ? null : -data.ads.cprDeltaPct, lang)}` : `${formatCurrency(data.ads.spend, data.ads.currency, { compact: true })} ad spend with CPR ${trend(data.ads.cprDeltaPct == null ? null : -data.ads.cprDeltaPct, lang)}`);
  const joined = bits.join(", ");
  return lang === "id"
    ? `Dalam ${days} hari terakhir, ${clientName} mencatat ${joined || "aktivitas pemasaran yang terbatas"}. Analisa di bawah merangkum pendorong utama performa periode ini beserta prioritas eksekusi 30 hari ke depan.`
    : `Over the last ${days} days, ${clientName} recorded ${joined || "limited marketing activity"}. The analysis below summarises the main performance drivers this period and the execution priorities for the next 30 days.`;
}

/** Deterministic markdown insight from the payload numbers. */
export function generateMockInsight(input: GenerateInsightInput): string {
  const { data, language: lang, clientName, range } = input;
  const social = data.social ? socialLines(data.social, lang) : null;
  const seo = data.seo ? seoLines(data.seo, lang) : null;
  const ads = data.ads ? adsLines(data.ads, lang) : null;

  const highlights = [...(social?.highlight ?? []), ...(seo?.highlight ?? []), ...(ads?.highlight ?? [])];
  const fixes = [...(social?.fix ?? []), ...(seo?.fix ?? []), ...(ads?.fix ?? [])];
  const recPool = [
    ...(ads?.rec ?? []),
    ...(seo?.rec ?? []),
    ...(social?.rec ?? []),
    lang === "id" ? `Tinjau KPI mingguan bersama tim dan sepakati satu eksperimen baru per kanal.` : `Review KPIs weekly with the team and agree one new experiment per channel.`,
    lang === "id" ? `Dokumentasikan pembelajaran periode ini sebagai basis kalender konten & media plan bulan depan.` : `Document this period's learnings as the basis for next month's content calendar & media plan.`,
    lang === "id" ? `Selaraskan pesan organik dan iklan agar funnel dari jangkauan ke konversi konsisten.` : `Align organic and paid messaging so the funnel from reach to conversion stays consistent.`,
  ];
  const recs = recPool.slice(0, 5);
  while (recs.length < 5) recs.push(lang === "id" ? "Pertahankan ritme evaluasi dua mingguan." : "Keep a fortnightly evaluation cadence.");

  const out: string[] = [];
  out.push(`## Ringkasan eksekutif`);
  out.push(summarize(data, clientName, range.days, lang));
  out.push("");
  out.push(`## Sorotan performa`);
  out.push(...highlights.map((h) => `- ${h}`));
  out.push("");
  out.push(`## Area perbaikan`);
  out.push(...fixes.map((f) => `- ${f}`));
  out.push("");
  if (seo && input.module === "SEO") {
    out.push(`## Peluang kata kunci`);
    out.push(...(seo.keywords.length ? seo.keywords.map((k) => `- ${k}`) : [lang === "id" ? "- Belum ada peluang menonjol pada periode ini." : "- No standout opportunities this period."]));
    out.push("");
  }
  if (ads && input.module === "ADS") {
    out.push(`## Alokasi anggaran`);
    out.push(...(ads.budget.length ? ads.budget.map((b) => `- ${b}`) : [lang === "id" ? "- Belum ada kampanye berjalan pada periode ini." : "- No campaigns ran this period."]));
    out.push("");
  }
  out.push(`## Rekomendasi 30 hari`);
  out.push(...recs.map((r, i) => `${i + 1}. ${r}`));
  return out.join("\n");
}

export const mockInsightProvider: InsightProvider = {
  id: "openrouter-mock",
  model: "mock",
  async generateInsight(input) {
    return generateMockInsight(input);
  },
};
