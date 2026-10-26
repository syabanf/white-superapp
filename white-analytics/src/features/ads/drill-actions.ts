"use server";

import { z } from "zod";
import { db } from "@/lib/db";
import { requireClientAccess } from "@/lib/rbac";
import { kpisWithConversion, type InsightRow } from "./queries-insights";

const input = z.object({
  slug: z.string().min(1),
  date: z.string().regex(/^\d{4}-\d{2}-\d{2}$/),
  campaignId: z.string().min(1).nullish(),
});

export type AdsDayCampaign = { id: string; name: string; spend: number; results: number; cpr: number; ctr: number };
export type AdsDayDrill = { spend: number; results: number; cpr: number; campaigns: AdsDayCampaign[] };

/** Spend and results per campaign on one day: the records behind a daily ads bar. */
export async function getAdsDayDrill(raw: z.input<typeof input>): Promise<AdsDayDrill> {
  const { slug, date, campaignId } = input.parse(raw);
  const { client } = await requireClientAccess(slug);
  const day = new Date(`${date}T00:00:00.000Z`);
  const rows = (await db.adDailyInsight.findMany({
    where: {
      adAccount: { clientId: client.id },
      adId: { not: null },
      date: day,
      ...(campaignId ? { campaignId } : {}),
    },
  })) as InsightRow[];

  const byCampaign = new Map<string, InsightRow[]>();
  for (const r of rows) {
    const key = r.campaignId ?? "";
    byCampaign.set(key, [...(byCampaign.get(key) ?? []), r]);
  }
  const names = await db.adCampaign.findMany({
    where: { id: { in: [...byCampaign.keys()].filter(Boolean) } },
    select: { id: true, name: true },
  });
  const nameOf = new Map(names.map((c) => [c.id, c.name]));
  const total = kpisWithConversion(rows);

  return {
    spend: total.spend,
    results: total.results,
    cpr: total.cpr,
    campaigns: [...byCampaign.entries()]
      .map(([id, rs]) => {
        const k = kpisWithConversion(rs);
        return { id, name: nameOf.get(id) ?? "–", spend: k.spend, results: k.results, cpr: k.cpr, ctr: k.ctr };
      })
      .sort((a, b) => b.spend - a.spend),
  };
}
