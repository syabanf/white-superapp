"use server";

import { z } from "zod";
import { db } from "@/lib/db";
import { addDays, toISODate } from "@/lib/dates";
import { requireClientAccess } from "@/lib/rbac";
import { getSuiteProperty } from "./queries";

const input = z.object({ slug: z.string().min(1), date: z.string().regex(/^\d{4}-\d{2}-\d{2}$/) });

export type RankDayRow = { id: string; keyword: string; device: string; position: number | null; previous: number | null };

/**
 * Tracked keywords with their position on one day (last snapshot on or before it)
 * and the snapshot before that: the records behind a rank trend point.
 */
export async function getRankDayDrill(raw: z.input<typeof input>): Promise<RankDayRow[]> {
  const { slug, date } = input.parse(raw);
  const { client } = await requireClientAccess(slug);
  const property = await getSuiteProperty(client.id);
  if (!property) return [];
  const day = new Date(`${date}T00:00:00.000Z`);
  const keywords = await db.trackedKeyword.findMany({
    where: { propertyId: property.id },
    select: { id: true, keyword: true, device: true },
  });
  const snaps = await db.rankSnapshot.findMany({
    where: { keywordId: { in: keywords.map((k) => k.id) }, date: { gte: addDays(day, -14), lt: addDays(day, 1) } },
    orderBy: { date: "desc" },
    select: { keywordId: true, date: true, position: true },
  });
  const byKeyword = new Map<string, { date: string; position: number | null }[]>();
  for (const s of snaps) {
    byKeyword.set(s.keywordId, [...(byKeyword.get(s.keywordId) ?? []), { date: toISODate(s.date), position: s.position }]);
  }
  return keywords
    .map((k) => {
      const [cur, prev] = byKeyword.get(k.id) ?? [];
      return { id: k.id, keyword: k.keyword, device: k.device, position: cur?.position ?? null, previous: prev?.position ?? null };
    })
    .sort((a, b) => (a.position ?? 999) - (b.position ?? 999));
}
