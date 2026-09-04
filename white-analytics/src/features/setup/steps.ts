/**
 * Workspace setup wizard — step model (pure). Runs once per workspace, before
 * the first project; every step is skippable (demo mode keeps working).
 */
import { z } from "zod";

export const SETUP_STEPS = ["meta", "google", "seo-data", "apify", "ai", "selesai"] as const;
export type SetupStep = (typeof SETUP_STEPS)[number];

export type SetupStepMeta = { key: SetupStep; index: string; title: string; description: string };

export const SETUP_META: Record<SetupStep, SetupStepMeta> = {
  meta: {
    key: "meta",
    index: "01",
    title: "Meta (Instagram, Facebook, Ads)",
    description: "Kredensial aplikasi Meta dan checklist App Review — penentu apakah publikasi bisa dikirim sungguhan.",
  },
  google: {
    key: "google",
    index: "02",
    title: "Google (Search Console & GA4)",
    description: "OAuth client untuk menarik data pencarian dan analitik situs klien.",
  },
  "seo-data": {
    key: "seo-data",
    index: "03",
    title: "Data SEO (DataForSEO)",
    description: "Volume kata kunci, SERP harian, backlink, dan domain kompetitor. Bayar per panggilan — ada estimasi biaya di bawah.",
  },
  apify: {
    key: "apify",
    index: "04",
    title: "Data publik (Apify)",
    description: "Posisi SERP untuk rank tracker serta profil & post publik Instagram, TikTok, Facebook — tanpa App Review dan tanpa langganan. Bayar per hasil.",
  },
  ai: {
    key: "ai",
    index: "05",
    title: "AI insight (OpenRouter)",
    description: "Ringkasan naratif di laporan. Tanpa kunci, ringkasan memakai templat.",
  },
  selesai: {
    key: "selesai",
    index: "06",
    title: "Selesai",
    description: "Ringkasan mode tiap integrasi dan langkah berikutnya.",
  },
};

export const setupStepSchema = z.enum(SETUP_STEPS).catch("meta");

export function resolveSetupStep(requested: string | undefined): SetupStep {
  return setupStepSchema.parse(requested ?? "meta");
}

export function setupStepNumber(step: SetupStep): number {
  return SETUP_STEPS.indexOf(step) + 1;
}

export function nextSetupStep(step: SetupStep): SetupStep | null {
  const i = SETUP_STEPS.indexOf(step);
  return i >= 0 && i < SETUP_STEPS.length - 1 ? SETUP_STEPS[i + 1]! : null;
}

export function prevSetupStep(step: SetupStep): SetupStep | null {
  const i = SETUP_STEPS.indexOf(step);
  return i > 0 ? SETUP_STEPS[i - 1]! : null;
}

export function setupHref(step: SetupStep): string {
  return step === "meta" ? "/setup" : `/setup?step=${step}`;
}
