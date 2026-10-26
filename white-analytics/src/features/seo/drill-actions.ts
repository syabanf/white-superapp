"use server";

import { z } from "zod";
import { db } from "@/lib/db";
import { ctr } from "@/lib/metrics";
import { requireClientAccess } from "@/lib/rbac";
import { getSeoProperty } from "./queries";

const input = z.object({ slug: z.string().min(1), date: z.string().regex(/^\d{4}-\d{2}-\d{2}$/) });

export type SeoDayRow = { key: string; clicks: number; impressions: number; ctr: number; position: number };
export type SeoDayDrill = { totals: { clicks: number; impressions: number; position: number | null }; queries: SeoDayRow[]; total: number };

/** Search Console queries of one day, by clicks: the records behind a daily SEO point. */
export async function getSeoDayDrill(raw: z.input<typeof input>): Promise<SeoDayDrill> {
  const { slug, date } = input.parse(raw);
  const { client } = await requireClientAccess(slug);
  const property = await getSeoProperty(client.id);
  if (!property) return { totals: { clicks: 0, impressions: 0, position: null }, queries: [], total: 0 };
  const day = new Date(`${date}T00:00:00.000Z`);
  const [daily, queries] = await Promise.all([
    db.seoDailyMetric.findFirst({ where: { propertyId: property.id, date: day }, select: { clicks: true, impressions: true, position: true } }),
    db.seoDimensionMetric.findMany({
      where: { propertyId: property.id, dimension: "QUERY", date: day },
      orderBy: [{ clicks: "desc" }, { impressions: "desc" }],
      select: { key: true, clicks: true, impressions: true, position: true },
    }),
  ]);
  return {
    totals: { clicks: daily?.clicks ?? 0, impressions: daily?.impressions ?? 0, position: daily?.position ?? null },
    queries: queries.slice(0, 25).map((q) => ({ ...q, ctr: ctr(q.clicks, q.impressions) })),
    total: queries.length,
  };
}
