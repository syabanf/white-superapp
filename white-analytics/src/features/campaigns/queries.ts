import "server-only";
import { cache } from "react";
import { db } from "@/lib/db";
import type { CampaignDraft } from "@/generated/prisma/client";
import { dateKeyOf, parseAudience, parseCreatives, parsePlacements, type Audience, type Creative, type DraftStatus, type Placement } from "./lib";
import { writerTargetFor } from "./submit";
import type { WriterGate } from "@/lib/providers/meta-ads-writer";

export type DraftDTO = {
  id: string;
  name: string;
  objective: string;
  dailyBudget: number;
  startDate: string;
  endDate: string | null;
  adAccountId: string | null;
  audience: Audience;
  placements: Placement[];
  creatives: Creative[];
  aiNotes: string;
  status: DraftStatus;
  externalId: string | null;
  simulated: boolean;
  submittedAt: string | null;
  updatedAt: string;
};

export function toDraftDTO(row: CampaignDraft): DraftDTO {
  return {
    id: row.id,
    name: row.name,
    objective: row.objective,
    dailyBudget: row.dailyBudget,
    startDate: dateKeyOf(row.startDate),
    endDate: row.endDate ? dateKeyOf(row.endDate) : null,
    adAccountId: row.adAccountId,
    audience: parseAudience(row.audience),
    placements: parsePlacements(row.placements),
    creatives: parseCreatives(row.creatives),
    aiNotes: row.aiNotes,
    status: row.status,
    externalId: row.externalId,
    simulated: row.simulated,
    submittedAt: row.submittedAt?.toISOString() ?? null,
    updatedAt: row.updatedAt.toISOString(),
  };
}

export type DraftListRow = {
  id: string;
  name: string;
  objective: string;
  dailyBudget: number;
  currency: string;
  startDate: string;
  endDate: string | null;
  status: DraftStatus;
  simulated: boolean;
  creatives: number;
  updatedAt: string;
  submittedAt: string | null;
};

export const listDrafts = cache(async (clientId: string, fallbackCurrency: string): Promise<DraftListRow[]> => {
  const [rows, accounts] = await Promise.all([
    db.campaignDraft.findMany({ where: { clientId }, orderBy: { updatedAt: "desc" }, take: 200 }),
    db.adAccount.findMany({ where: { clientId }, select: { id: true, currency: true } }),
  ]);
  const currencyOf = new Map(accounts.map((a) => [a.id, a.currency]));
  return rows.map((r) => {
    const d = toDraftDTO(r);
    return {
      id: d.id,
      name: d.name,
      objective: d.objective,
      dailyBudget: d.dailyBudget,
      currency: (d.adAccountId && currencyOf.get(d.adAccountId)) || fallbackCurrency,
      startDate: d.startDate,
      endDate: d.endDate,
      status: d.status,
      simulated: d.simulated,
      creatives: d.creatives.filter((cr) => cr.headline.trim() || cr.primaryText.trim()).length,
      updatedAt: d.updatedAt,
      submittedAt: d.submittedAt,
    };
  });
});

export const getDraft = cache(async (clientId: string, id: string): Promise<DraftDTO | null> => {
  const row = await db.campaignDraft.findFirst({ where: { id, clientId } });
  return row ? toDraftDTO(row) : null;
});

export type BuilderAdAccount = { id: string; name: string; externalId: string; currency: string; gate: WriterGate };
export type BuilderImage = { id: string; url: string; thumbnailUrl: string | null; filename: string; altText: string | null };
export type BuilderBrand = { clientName: string; primaryColor: string; secondaryColor: string; logoUrl: string | null };

/**
 * Everything the builder needs besides the draft. Must run inside
 * `withClientMode` so the per-account gate sees the right demo flag.
 */
export const getBuilderContext = cache(async (clientId: string) => {
  const [accounts, images, client] = await Promise.all([
    db.adAccount.findMany({ where: { clientId }, orderBy: { name: "asc" }, select: { id: true, name: true, externalId: true, currency: true } }),
    db.mediaAsset.findMany({
      where: { clientId, kind: "IMAGE" },
      orderBy: { createdAt: "desc" },
      take: 60,
      select: { id: true, url: true, thumbnailUrl: true, filename: true, altText: true },
    }),
    db.client.findUniqueOrThrow({
      where: { id: clientId },
      select: { name: true, brandKit: { select: { primaryColor: true, secondaryColor: true, logoUrl: true } } },
    }),
  ]);
  const adAccounts: BuilderAdAccount[] = await Promise.all(
    accounts.map(async (a) => ({ ...a, gate: (await writerTargetFor(clientId, a.id)).gate })),
  );
  const noAccountGate = (await writerTargetFor(clientId, null)).gate;
  const brand: BuilderBrand = {
    clientName: client.name,
    primaryColor: client.brandKit?.primaryColor ?? "#2b2265",
    secondaryColor: client.brandKit?.secondaryColor ?? "#fbeaf2",
    logoUrl: client.brandKit?.logoUrl ?? null,
  };
  return { adAccounts, images: images as BuilderImage[], brand, noAccountGate };
});
