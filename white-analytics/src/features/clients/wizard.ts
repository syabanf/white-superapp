/**
 * "Tambah klien" wizard — step model.
 *
 * The project *type* is chosen first and drives the rest of the flow: which
 * fields matter, which data sources are asked for, and what the client link
 * defaults to. Pure — no DB, no React — so the page, the rail and the tests all
 * agree on one source of truth.
 */
import { z } from "zod";

export const WIZARD_STEPS = ["jenis", "profil", "sumber", "akses", "selesai"] as const;
export type WizardStep = (typeof WIZARD_STEPS)[number];

export const PROJECT_TYPES = ["SEO", "SOCIAL", "ADS"] as const;
export type ProjectType = (typeof PROJECT_TYPES)[number];

export type WizardStepMeta = {
  key: WizardStep;
  index: string;
  title: string;
  description: string;
  /** step cannot render before the client row exists */
  needsClient: boolean;
  /** step cannot render before a project type is chosen */
  needsTypes: boolean;
};

export const WIZARD_META: Record<WizardStep, WizardStepMeta> = {
  jenis: {
    key: "jenis",
    index: "01",
    title: "Pilih kanal",
    description: "Pilih kanal yang akan dikelola. Langkah berikutnya menyesuaikan pilihan ini.",
    needsClient: false,
    needsTypes: false,
  },
  profil: {
    key: "profil",
    index: "02",
    title: "Profil klien",
    description: "Nama klien, alamat dashboard, dan preferensi angka.",
    needsClient: false,
    needsTypes: true,
  },
  sumber: {
    key: "sumber",
    index: "03",
    title: "Sumber data",
    description: "Hubungkan akun dan daftarkan properti yang akan ditarik datanya.",
    needsClient: true,
    needsTypes: false,
  },
  akses: {
    key: "akses",
    index: "04",
    title: "Akses & laporan",
    description: "Siapa yang boleh melihat, dan tautan read-only untuk klien.",
    needsClient: true,
    needsTypes: false,
  },
  selesai: {
    key: "selesai",
    index: "05",
    title: "Selesai",
    description: "Ringkasan konfigurasi dan langkah berikutnya.",
    needsClient: true,
    needsTypes: false,
  },
};

export const wizardStepSchema = z.enum(WIZARD_STEPS).catch("jenis");
export const projectTypesSchema = z.array(z.enum(PROJECT_TYPES)).min(1);

export function stepNumber(step: WizardStep): number {
  return WIZARD_STEPS.indexOf(step) + 1;
}

export function nextStep(step: WizardStep): WizardStep | null {
  const i = WIZARD_STEPS.indexOf(step);
  return i >= 0 && i < WIZARD_STEPS.length - 1 ? WIZARD_STEPS[i + 1]! : null;
}

export function prevStep(step: WizardStep): WizardStep | null {
  const i = WIZARD_STEPS.indexOf(step);
  return i > 0 ? WIZARD_STEPS[i - 1]! : null;
}

/** `?types=SEO,ADS` → ordered, de-duplicated, invalid entries dropped. */
export function parseTypes(raw: string | undefined): ProjectType[] {
  if (!raw) return [];
  const wanted = new Set(raw.split(",").map((s) => s.trim().toUpperCase()));
  return PROJECT_TYPES.filter((p) => wanted.has(p));
}

export function serializeTypes(types: readonly ProjectType[]): string {
  return PROJECT_TYPES.filter((p) => types.includes(p)).join(",");
}

/**
 * Guard for direct URL access. A step can only render once its prerequisites
 * exist, and finished stages are skipped rather than re-asked.
 */
export function resolveStep(
  requested: string | undefined,
  state: { hasClient: boolean; hasTypes: boolean },
): WizardStep {
  const step = wizardStepSchema.parse(requested ?? "jenis");
  const meta = WIZARD_META[step];
  if (meta.needsClient && !state.hasClient) return state.hasTypes ? "profil" : "jenis";
  if (meta.needsTypes && !state.hasTypes) return "jenis";
  // Type and profile are settled once the client exists.
  if (!meta.needsClient && state.hasClient) return "sumber";
  return step;
}

export function wizardHref(
  step: WizardStep,
  opts: { slug?: string; types?: readonly ProjectType[] } = {},
): string {
  const q = new URLSearchParams({ step });
  if (opts.types && opts.types.length > 0) q.set("types", serializeTypes(opts.types));
  if (opts.slug) q.set("client", opts.slug);
  return `/clients/new?${q.toString()}`;
}

/** Which source sections step 3 should render, given the project types. */
export function sourceSectionsFor(types: readonly string[]): {
  social: boolean;
  seo: boolean;
  ads: boolean;
} {
  return {
    social: types.includes("SOCIAL"),
    seo: types.includes("SEO"),
    ads: types.includes("ADS"),
  };
}

/** SEO projects are anchored on a site, so the website field stops being optional. */
export function websiteRequiredFor(types: readonly string[]): boolean {
  return types.includes("SEO");
}
