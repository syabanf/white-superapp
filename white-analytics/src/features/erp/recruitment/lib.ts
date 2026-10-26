import { canAccess, type ErpAccess } from "@/components/shell/nav-config";

export type RecPerms = { read: boolean; write: boolean };

export function recPerms(access: ErpAccess): RecPerms {
  return { read: canAccess(access, "recruitment.read"), write: canAccess(access, "recruitment.write") };
}

/** Candidate stages from which the pipeline report PDF is available. */
export const REPORT_STAGES = ["offer", "hired"];

/** Open offer statuses (the candidate can still respond). */
export const OPEN_OFFER = ["sent", "negotiating"];

/** 125 → "02:05". */
export function formatCountdown(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds));
  return `${String(Math.floor(s / 60)).padStart(2, "0")}:${String(s % 60).padStart(2, "0")}`;
}

/** "9500000" | "9500000.00" | undefined → whole rupiah string for an int64 field, or undefined. */
export function toRupiahInt(value: string | undefined | null): string | undefined {
  if (value == null || value.trim() === "") return undefined;
  const n = Math.round(Number(value));
  return Number.isFinite(n) && n >= 0 ? String(n) : undefined;
}

/** Interview start in WIB from a "YYYY-MM-DD" day and "HH:mm" time. */
export function wibInstant(day: string, time: string): string | null {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(day) || !/^\d{2}:\d{2}$/.test(time)) return null;
  const d = new Date(`${day}T${time}:00+07:00`);
  return Number.isNaN(d.getTime()) ? null : d.toISOString();
}

/** ISO instant → { day, time } in WIB, for editing a scheduled interview. */
export function wibParts(iso: string | null): { day: string; time: string } {
  if (!iso) return { day: "", time: "" };
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Jakarta",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
  }).formatToParts(new Date(iso));
  const get = (t: string) => parts.find((p) => p.type === t)?.value ?? "";
  return { day: `${get("year")}-${get("month")}-${get("day")}`, time: `${get("hour")}:${get("minute")}` };
}

/** Average of 1..5 scores, one decimal; null when there are none. */
export function averageScore(scores: number[]): number | null {
  const valid = scores.filter((s) => s >= 1 && s <= 5);
  if (!valid.length) return null;
  return Math.round((valid.reduce((a, b) => a + b, 0) / valid.length) * 10) / 10;
}

/** Benefits textarea → trimmed non-empty lines. */
export function splitLines(text: string): string[] {
  return text
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean);
}
