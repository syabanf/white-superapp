import "server-only";
import { cache } from "react";
import { db } from "@/lib/db";
import type { DesignDraft } from "@/generated/prisma/client";
import { isDesignFormat, parseLayout, type DraftRow } from "@/features/design-studio/lib";

export function toDraftRow(d: DesignDraft): DraftRow {
  return {
    id: d.id,
    title: d.title,
    brief: d.brief,
    format: isDesignFormat(d.format) ? d.format : "FEED_SQUARE",
    layout: parseLayout(d.layout),
    imagePrompt: d.imagePrompt,
    imageUrl: d.imageUrl,
    mediaAssetId: d.mediaAssetId,
    model: d.model,
    createdAt: d.createdAt.toISOString(),
  };
}

/** Latest design drafts for the client (newest first). */
export const listDesignDrafts = cache(async (clientId: string, take = 24): Promise<DraftRow[]> => {
  const rows = await db.designDraft.findMany({ where: { clientId }, orderBy: { createdAt: "desc" }, take });
  return rows.map(toDraftRow);
});

export const hasBrandKit = cache(async (clientId: string): Promise<boolean> => {
  const k = await db.brandKit.findUnique({ where: { clientId }, select: { id: true } });
  return Boolean(k);
});
