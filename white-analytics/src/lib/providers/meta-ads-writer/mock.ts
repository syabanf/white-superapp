/**
 * Simulation writer: used whenever the real gate fails (no Meta app, demo
 * client, no real connection, or no `ads_management` scope). Creates nothing
 * in Meta and spends nothing. Ids are deterministic per draft so a repeat
 * submission of the same draft yields the same `sim_…` id.
 *
 * A draft name containing `[fail]` throws, so the error path can be exercised
 * from the UI without a real account (same convention as the publisher mock).
 */
import { ProviderError } from "@/lib/action-result";
import type { CreateCampaignResult, MetaAdsWriter, WriterContext, WriterDraft } from "./types";

export const MOCK_WRITER_FAIL_MARKER = "[fail]";

function hash36(input: string): string {
  let h = 2166136261;
  for (let i = 0; i < input.length; i++) {
    h ^= input.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return (h >>> 0).toString(36).padStart(7, "0");
}

export function mockCreateCampaign(draft: WriterDraft, ctx: Pick<WriterContext, "adAccountExternalId">): CreateCampaignResult {
  if (draft.name.includes(MOCK_WRITER_FAIL_MARKER)) {
    throw new ProviderError("meta-ads-writer", "UNKNOWN", "Simulasi gagal (penanda [fail] di nama kampanye).", { requiresReconnect: false });
  }
  const base = hash36(`${ctx.adAccountExternalId}|${draft.id}`);
  return {
    externalId: `sim_${base}`,
    simulated: true,
    adSetId: `sim_${base}_as`,
    adIds: draft.creatives.map((_, i) => `sim_${base}_ad${i + 1}`),
  };
}

export const mockAdsWriter: MetaAdsWriter = {
  id: "mock",
  async createCampaign(draft, ctx) {
    return mockCreateCampaign(draft, ctx);
  },
};
