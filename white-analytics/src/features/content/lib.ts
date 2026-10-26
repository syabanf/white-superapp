/** Pure helpers for the content module (no server or browser APIs). Tested in tests/content.test.ts. */
import type { AssetKind, ContentFormat, DesignStatus, Item, ItemInput, ItemStatus, Platform, PlanStatus } from "./types";

export const PLATFORMS: Platform[] = ["INSTAGRAM", "FACEBOOK", "TIKTOK"];
export const FORMATS: ContentFormat[] = ["FEED", "CAROUSEL", "REEL", "STORY"];
export const ITEM_STATUSES: ItemStatus[] = [
  "IDEA",
  "COPYWRITING",
  "DESIGN",
  "REVIEW",
  "REVISION",
  "APPROVED",
  "SCHEDULED",
  "PUBLISHED",
  "FAILED",
];
export const DESIGN_STATUSES: DesignStatus[] = [
  "DESIGN_TODO",
  "DESIGN_IN_PROGRESS",
  "DESIGN_IN_REVIEW",
  "DESIGN_CHANGES_REQUESTED",
  "DESIGN_APPROVED",
];
export const ASSET_KINDS: AssetKind[] = ["LOGO", "BRAND_GUIDELINE", "FONT", "PHOTO", "TEMPLATE", "OTHER"];

export type Tone = "neutral" | "warning" | "info" | "brand" | "good" | "critical";

export const TONE_VAR: Record<Tone, string> = {
  neutral: "var(--muted-foreground)",
  warning: "var(--status-warning)",
  info: "var(--chart-3)",
  brand: "var(--chart-1)",
  good: "var(--status-good)",
  critical: "var(--status-critical)",
};

export const ITEM_TONE: Record<ItemStatus, Tone> = {
  IDEA: "neutral",
  COPYWRITING: "info",
  DESIGN: "info",
  REVIEW: "warning",
  REVISION: "critical",
  APPROVED: "good",
  SCHEDULED: "brand",
  PUBLISHED: "good",
  FAILED: "critical",
};

export const PLAN_TONE: Record<PlanStatus, Tone> = {
  PLAN_DRAFT: "neutral",
  PLAN_IN_REVIEW: "warning",
  PLAN_APPROVED: "good",
};

export const DESIGN_TONE: Record<DesignStatus, Tone> = {
  DESIGN_TODO: "neutral",
  DESIGN_IN_PROGRESS: "info",
  DESIGN_IN_REVIEW: "warning",
  DESIGN_CHANGES_REQUESTED: "critical",
  DESIGN_APPROVED: "good",
};

export function isItemStatus(v: string | undefined | null): v is ItemStatus {
  return !!v && (ITEM_STATUSES as string[]).includes(v);
}
export function isDesignStatus(v: string | undefined | null): v is DesignStatus {
  return !!v && (DESIGN_STATUSES as string[]).includes(v);
}
export function isAssetKind(v: string | undefined | null): v is AssetKind {
  return !!v && (ASSET_KINDS as string[]).includes(v);
}
export function isMonthKey(v: string | undefined | null): v is string {
  return !!v && /^\d{4}-(0[1-9]|1[0-2])$/.test(v);
}

export type Stage = "COPY" | "DESIGN" | "REVIEW";

/** The deadline that applies to an item in its current status, if any. */
export function currentDue(item: Pick<Item, "status" | "copyDue" | "designDue" | "reviewDue">): { stage: Stage; date: string } | null {
  switch (item.status) {
    case "IDEA":
    case "COPYWRITING":
      return item.copyDue ? { stage: "COPY", date: item.copyDue } : null;
    case "DESIGN":
      return item.designDue ? { stage: "DESIGN", date: item.designDue } : null;
    case "REVIEW":
    case "REVISION":
      return item.reviewDue ? { stage: "REVIEW", date: item.reviewDue } : null;
    default:
      return null;
  }
}

/** Due date strings are YYYY-MM-DD, so lexical comparison is chronological. */
export function isOverdue(item: Parameters<typeof currentDue>[0], today: string): boolean {
  const due = currentDue(item);
  return !!due && due.date < today;
}

/** "a, b , ,c" → ["a", "b", "c"] */
export function splitList(raw: string): string[] {
  return raw
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
}

/** Hashtags typed with spaces, commas or newlines → "#tag" list without duplicates. */
export function parseHashtags(raw: string): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const part of raw.split(/[\s,]+/)) {
    const tag = part.replace(/^#+/, "");
    if (!tag || seen.has(tag.toLowerCase())) continue;
    seen.add(tag.toLowerCase());
    out.push(`#${tag}`);
  }
  return out;
}

export function formatBytes(raw: string | number): string {
  const n = Number(raw);
  if (!Number.isFinite(n) || n < 0) return "–";
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1).replace(".", ",")} KB`;
  return `${(n / 1024 / 1024).toFixed(1).replace(".", ",")} MB`;
}

export function mediaKind(mime: string | undefined): "image" | "video" | "pdf" | "other" {
  if (!mime) return "other";
  if (mime.startsWith("image/")) return "image";
  if (mime.startsWith("video/")) return "video";
  if (mime === "application/pdf") return "pdf";
  return "other";
}

/** UpdateItem replaces every planning field, so edits start from the full current input. */
export function itemInputFrom(item: Item): ItemInput {
  return {
    plannedDate: item.plannedDate,
    publishAt: item.publishAt,
    platform: item.platform,
    format: item.format,
    pillar: item.pillar,
    title: item.title,
    visualBrief: item.visualBrief,
    copywriterId: item.copywriterId ?? "",
    designerId: item.designerId ?? "",
    copyDue: item.copyDue ?? "",
    designDue: item.designDue ?? "",
    reviewDue: item.reviewDue ?? "",
  };
}

/** "2026-11" → "2026-12" */
export function shiftMonth(month: string, n: number): string {
  const [y, m] = month.split("-").map(Number) as [number, number];
  const d = new Date(Date.UTC(y, m - 1 + n, 1));
  return `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, "0")}`;
}

/** Engagements as a number for sorting and formatting (int64 arrives as a string). */
export function num(v: string | number | undefined): number {
  const n = Number(v ?? 0);
  return Number.isFinite(n) ? n : 0;
}

// ── same-origin proxy paths (see src/app/api/erp/[...path]) ──

const proxyClient = (clientId: string) => `/api/erp/v1/content/clients/${encodeURIComponent(clientId)}`;
const uploadClient = (clientId: string) => `/api/erp/api/content/clients/${encodeURIComponent(clientId)}`;

export const contentFiles = {
  designFile: (clientId: string, fileId: string) => `${proxyClient(clientId)}/design-files/${encodeURIComponent(fileId)}/download`,
  asset: (clientId: string, assetId: string, version = 0) =>
    `${proxyClient(clientId)}/assets/${encodeURIComponent(assetId)}/download${version ? `?version=${version}` : ""}`,
  reportPdf: (clientId: string, reportId: string) => `${proxyClient(clientId)}/reports/${encodeURIComponent(reportId)}/pdf`,
  item: (clientId: string, itemId: string) => `${proxyClient(clientId)}/items/${encodeURIComponent(itemId)}`,
  items: (clientId: string, query = "") => `${proxyClient(clientId)}/items${query ? `?${query}` : ""}`,
  uploadDesign: (clientId: string, taskId: string) => `${uploadClient(clientId)}/design-tasks/${encodeURIComponent(taskId)}/files`,
  createAsset: (clientId: string) => `${uploadClient(clientId)}/assets`,
  assetVersion: (clientId: string, assetId: string) => `${uploadClient(clientId)}/assets/${encodeURIComponent(assetId)}/versions`,
};
