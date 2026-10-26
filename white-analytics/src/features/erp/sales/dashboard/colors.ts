import { MUTED_SERIES, slotColor } from "@/components/dashboard/charts/chart-primitives";
import { LEAD_SOURCE, SERVICE_LINE } from "../strings";

/** Service lines keep one color each, in the catalog order. */
export const LINE_ORDER = Object.keys(SERVICE_LINE);
export const lineColor = (key: string) => (LINE_ORDER.includes(key) ? slotColor(LINE_ORDER.indexOf(key)) : MUTED_SERIES);
export const lineLabel = (key: string) => SERVICE_LINE[key] ?? "Lainnya";

/** The seven main lead sources get a slot; the rest fold into "Lainnya". */
export const SOURCE_ORDER = ["instagram", "google", "referral", "website", "wa", "linkedin", "chatbot"];
export const sourceGroup = (key: string) => (SOURCE_ORDER.includes(key) ? key : "lainnya");
export const sourceColor = (key: string) => (SOURCE_ORDER.includes(key) ? slotColor(SOURCE_ORDER.indexOf(key)) : MUTED_SERIES);
export const sourceLabel = (key: string) => LEAD_SOURCE[key] ?? key;

/** Ordinal ramp for ordered buckets (light to dark). */
export const SEQ = ["var(--seq-200)", "var(--seq-300)", "var(--seq-450)", "var(--seq-600)", "var(--seq-700)"];
