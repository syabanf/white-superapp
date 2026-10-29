import "server-only";
import { db } from "@/lib/db";
import type { BrandContext } from "@/lib/providers/ai-studio/types";

/**
 * Brand memory for AI prompts. Falls back to neutral defaults when the client
 * has no Brand Kit yet, so every AI feature works before the kit is filled in.
 */
export async function getBrandContext(clientId: string): Promise<BrandContext> {
  const client = await db.client.findUniqueOrThrow({
    where: { id: clientId },
    select: { name: true, industry: true, brandKit: true },
  });
  const k = client.brandKit;
  return {
    clientName: client.name,
    industry: client.industry,
    primaryColor: k?.primaryColor ?? "#2b2265",
    secondaryColor: k?.secondaryColor ?? "#fbeaf2",
    accentColors: k?.accentColors ?? [],
    headingFont: k?.headingFont ?? "",
    bodyFont: k?.bodyFont ?? "",
    toneOfVoice: k?.toneOfVoice ?? "",
    audience: k?.audience ?? "",
    doList: k?.doList ?? [],
    dontList: k?.dontList ?? [],
    keywords: k?.keywords ?? [],
    hashtags: k?.hashtags ?? [],
    sampleCaptions: k?.sampleCaptions ?? [],
    visualStyle: k?.visualStyle ?? "",
  };
}
