/**
 * Pemilih adapter Meta Marketing: real bila kredensial app tersedia DAN
 * ada access token koneksi; selain itu mock (mode demo + DemoBanner).
 */
import type { MetaMarketingAdapter } from "./types";
import { mockMetaMarketing } from "./mock";
import { realMetaMarketing } from "./real";

export type { InsightsQuery, MetaAd, MetaAdSet, MetaCampaign, MetaInsightRow, MetaMarketingAdapter } from "./types";
export { mockMetaMarketing } from "./mock";
export { realMetaMarketing } from "./real";

/** Kredensial app Meta tersedia di env? */
export function isMetaConfigured(): boolean {
  return Boolean(process.env.META_APP_ID && process.env.META_APP_SECRET);
}

/** Adapter aktif untuk satu koneksi (token sudah didekripsi oleh pemanggil). */
export function getMetaMarketing(accessToken?: string | null): MetaMarketingAdapter {
  if (isMetaConfigured() && accessToken) return realMetaMarketing(accessToken);
  return mockMetaMarketing;
}
