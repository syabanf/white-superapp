import "server-only";
import { db } from "@/lib/db";
import { getConnectionToken } from "@/lib/connections";
import { isDemoConnectionId } from "@/lib/providers/meta-oauth";
import { adsWriterGate, pickAdsWriter, type CreateCampaignResult, type WriterDraft, type WriterGate } from "@/lib/providers/meta-ads-writer";
import type { CampaignDraft } from "@/generated/prisma/client";
import { filledCreatives, isObjective, parseAudience, parseCreatives, parsePlacements } from "./lib";

/**
 * Gather the facts the writer gate needs for one ad account (or none).
 * Call inside `withClientMode` so the demo flag is right.
 */
export async function writerTargetFor(clientId: string, adAccountId: string | null) {
  const account = adAccountId
    ? await db.adAccount.findFirst({
        where: { id: adAccountId, clientId },
        select: { id: true, externalId: true, currency: true, connection: { select: { id: true, provider: true, accountId: true, scopes: true } } },
      })
    : null;
  const conn = account?.connection ?? null;
  const realConnection = Boolean(conn && conn.provider === "META" && !isDemoConnectionId(conn.accountId));
  const gate: WriterGate = adsWriterGate({ realConnection, scopes: conn?.scopes ?? [] });
  return { gate, account, connectionId: conn?.id ?? null };
}

/** Relative media paths (`/api/media/…`) → absolute URL Meta can fetch. */
function absoluteUrl(url: string | null): string | null {
  if (!url) return null;
  if (/^https?:\/\//.test(url)) return url;
  const base = process.env.NEXT_PUBLIC_APP_URL?.replace(/\/$/, "");
  return base ? `${base}${url.startsWith("/") ? "" : "/"}${url}` : null;
}

export function toWriterDraft(row: CampaignDraft, currency: string): WriterDraft {
  return {
    id: row.id,
    name: row.name,
    objective: isObjective(row.objective) ? row.objective : "OUTCOME_TRAFFIC",
    dailyBudget: row.dailyBudget,
    currency,
    startDate: row.startDate,
    endDate: row.endDate,
    audience: parseAudience(row.audience),
    placements: parsePlacements(row.placements),
    creatives: filledCreatives(parseCreatives(row.creatives)).map((cr) => ({ ...cr, mediaUrl: absoluteUrl(cr.mediaUrl) })),
  };
}

/**
 * Submit one draft through the writer the gate allows. Real path: decrypted
 * token, the client's own Facebook Page and website. Simulation otherwise.
 * Throws ProviderError (caught by the action).
 */
export async function submitDraftToMeta(row: CampaignDraft, client: { id: string; currency: string; websiteUrl: string | null }): Promise<CreateCampaignResult & { gate: WriterGate }> {
  const target = await writerTargetFor(client.id, row.adAccountId);
  const currency = target.account?.currency ?? client.currency;
  const draft = toWriterDraft(row, currency);
  const writer = pickAdsWriter(target.gate);

  if (target.gate.mode === "real" && target.account) {
    const token = await getConnectionToken(target.connectionId);
    if (token) {
      const page = await db.socialAccount.findFirst({
        where: { clientId: client.id, platform: "FACEBOOK", isCompetitor: false },
        select: { externalId: true },
      });
      const result = await writer.createCampaign(draft, {
        accessToken: token.token,
        adAccountExternalId: target.account.externalId,
        pageId: page?.externalId ?? null,
        linkUrl: client.websiteUrl,
      });
      return { ...result, gate: target.gate };
    }
  }
  // Simulation (also when the gate passed but no usable token was found).
  const simGate: WriterGate = target.gate.mode === "real" ? { mode: "simulation", reasons: ["NO_CONNECTION"] } : target.gate;
  const result = await pickAdsWriter(simGate).createCampaign(draft, {
    accessToken: "",
    adAccountExternalId: target.account?.externalId ?? `client_${client.id}`,
  });
  return { ...result, gate: simGate };
}
