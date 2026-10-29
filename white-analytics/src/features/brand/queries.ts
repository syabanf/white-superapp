import "server-only";
import { cache } from "react";
import { db } from "@/lib/db";

/** Serializable Brand Kit for the form and preview. */
export type BrandKitData = {
  primaryColor: string;
  secondaryColor: string;
  accentColors: string[];
  headingFont: string;
  bodyFont: string;
  logoUrl: string | null;
  toneOfVoice: string;
  audience: string;
  doList: string[];
  dontList: string[];
  keywords: string[];
  hashtags: string[];
  sampleCaptions: string[];
  visualStyle: string;
  notes: string;
};

export type BrandKitRow = BrandKitData & { updatedAt: string; updatedByName: string | null };

export const getBrandKit = cache(async (clientId: string): Promise<BrandKitRow | null> => {
  const k = await db.brandKit.findUnique({ where: { clientId } });
  if (!k) return null;
  const updater = k.updatedById
    ? await db.user.findUnique({ where: { id: k.updatedById }, select: { name: true } })
    : null;
  return {
    primaryColor: k.primaryColor,
    secondaryColor: k.secondaryColor,
    accentColors: k.accentColors,
    headingFont: k.headingFont,
    bodyFont: k.bodyFont,
    logoUrl: k.logoUrl,
    toneOfVoice: k.toneOfVoice,
    audience: k.audience,
    doList: k.doList,
    dontList: k.dontList,
    keywords: k.keywords,
    hashtags: k.hashtags,
    sampleCaptions: k.sampleCaptions,
    visualStyle: k.visualStyle,
    notes: k.notes,
    updatedAt: k.updatedAt.toISOString(),
    updatedByName: updater?.name ?? null,
  };
});
