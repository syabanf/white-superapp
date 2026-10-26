import type { StatusKind } from "@/components/dashboard/status-badge";
import { canAccess, type ErpAccess } from "@/components/shell/nav-config";

/** What the signed-in user may do in Sales; the backend enforces it again. */
export type SalesCaps = { write: boolean; approve: boolean; admin: boolean; finance: boolean; userId?: string };

export function salesCaps(access: ErpAccess): SalesCaps {
  return {
    write: canAccess(access, "sales.write"),
    approve: canAccess(access, "sales.approve"),
    admin: access.role === "ADMIN",
    finance: canAccess(access, "finance.write"),
  };
}

/** Badge colors for the Indonesian value sets. */
export const KIND: Record<string, StatusKind> = {
  // leads
  baru: "info",
  dihubungi: "warning",
  qualified: "good",
  "tidak-cocok": "neutral",
  panas: "critical",
  hangat: "warning",
  dingin: "info",
  // quotations, approvals
  draft: "neutral",
  terkirim: "info",
  diterima: "good",
  ditolak: "critical",
  superseded: "neutral",
  none: "neutral",
  pending: "warning",
  approved: "good",
  rejected: "critical",
  // invoices, payments
  diajukan: "warning",
  batal: "neutral",
  belum: "warning",
  sebagian: "warning",
  lunas: "good",
  // contracts
  sent: "info",
  signed: "good",
  active: "good",
  ended: "neutral",
  cancelled: "neutral",
  // onboarding
  aktif: "info",
  selesai: "good",
  todo: "neutral",
  done: "good",
  skipped: "neutral",
  // chat
  open: "info",
  in_progress: "warning",
  waiting_customer: "neutral",
  resolved: "good",
  positif: "good",
  netral: "neutral",
  negatif: "critical",
  // forms, workflow runs
  accepted: "good",
  spam: "critical",
  invalid: "warning",
  success: "good",
  partial: "warning",
  failed: "critical",
  scheduled: "info",
};

/** Whole-percent split check with a small float tolerance. */
export function sumsTo100(values: number[]): boolean {
  return Math.abs(values.reduce((a, b) => a + (Number(b) || 0), 0) - 100) < 0.05;
}

/** Name lookup for user ids (owners list). */
export function nameOf(names: Record<string, string>, id: string | null | undefined): string {
  if (!id) return "-";
  return names[id] ?? id;
}

/** Days since an RFC 3339 timestamp, floored. */
export function daysSince(ts: string | null | undefined, now = Date.now()): number {
  if (!ts) return 0;
  return Math.max(0, Math.floor((now - new Date(ts).getTime()) / 86_400_000));
}

/** "/v1/public/sales/contracts/{token}" → "{token}". */
export function tokenFromPath(path: string): string {
  return path.split("/").filter(Boolean).pop() ?? "";
}

/** Public signing page for a sent contract. */
export const signPagePath = (c: { signPath: string }) => (c.signPath ? `/kontrak/${tokenFromPath(c.signPath)}` : "");

/** "YYYY-MM" of today in Jakarta. */
export function currentMonth(now = new Date()): string {
  return new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Jakarta", year: "numeric", month: "2-digit" }).format(now);
}

/** Human file size. */
export function fileSize(bytes: string | number): string {
  const n = Number(bytes) || 0;
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(0)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}
