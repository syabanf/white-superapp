import type { Priority, Recurrence, TaskItem, TaskStatus } from "./types";

/** "YYYY-MM-DD" of today in Asia/Jakarta. */
export function jakartaToday(now = new Date()): string {
  return new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Jakarta" }).format(now);
}

export function splitMonth(ym: string): { year: number; month: number } {
  const [y, m] = ym.split("-").map(Number);
  return { year: y, month: m };
}

export function isMonth(v: string | undefined): v is string {
  return !!v && /^\d{4}-(0[1-9]|1[0-2])$/.test(v);
}

export function isDate(v: string | undefined): v is string {
  return !!v && /^\d{4}-\d{2}-\d{2}$/.test(v);
}

export const quarterOf = (month: number) => Math.ceil(month / 3);

function addDays(day: string, n: number): string {
  const [y, m, d] = day.split("-").map(Number);
  const t = new Date(Date.UTC(y, m - 1, d + n));
  return t.toISOString().slice(0, 10);
}

/** Whole days from `date` to `today` (both YYYY-MM-DD). */
export function daysBetween(date: string, today: string): number {
  return Math.round((Date.parse(`${today}T00:00:00Z`) - Date.parse(`${date}T00:00:00Z`)) / 86_400_000);
}

/** Fraction 0-1 to "83%"; null to "–". */
export function ratio(v: number | null | undefined): string {
  return v == null ? "–" : `${Math.round(v * 100)}%`;
}

/** Text tone for a 0-100 score. */
export function scoreTone(score: number | null | undefined): string {
  if (score == null) return "text-muted-foreground";
  if (score >= 85) return "text-[var(--status-good)]";
  if (score >= 70) return "text-foreground";
  if (score >= 55) return "text-[var(--status-warning)]";
  return "text-[var(--status-critical)]";
}

const isOpenStatus = (s: TaskStatus) => s === "pending" || s === "in_progress" || s === "rejected";

export type Section = "overdue" | "today" | "upcoming" | "review" | "closed";
export const SECTION_ORDER: Section[] = ["overdue", "today", "upcoming", "review", "closed"];

/** Own list sections: overdue first, then today, upcoming, waiting for review, approved. */
export function groupBySection(items: TaskItem[], today: string): Record<Section, TaskItem[]> {
  const out: Record<Section, TaskItem[]> = { overdue: [], today: [], upcoming: [], review: [], closed: [] };
  for (const t of items) {
    if (t.status === "approved") out.closed.push(t);
    else if (t.status === "done") out.review.push(t);
    else if (t.occurrenceDate < today) out.overdue.push(t);
    else if (t.occurrenceDate === today) out.today.push(t);
    else out.upcoming.push(t);
  }
  return out;
}

export const STATUS_ORDER: TaskStatus[] = ["pending", "in_progress", "done", "approved", "rejected"];

export function groupByStatus(items: TaskItem[]): Record<TaskStatus, TaskItem[]> {
  const out: Record<TaskStatus, TaskItem[]> = { pending: [], in_progress: [], done: [], approved: [], rejected: [] };
  for (const t of items) out[t.status]?.push(t);
  return out;
}

export function completionSummary(items: TaskItem[]) {
  let open = 0;
  let overdue = 0;
  let review = 0;
  let approved = 0;
  for (const t of items) {
    if (t.status === "approved") approved++;
    else if (t.status === "done") review++;
    else if (isOpenStatus(t.status)) {
      open++;
      if (t.overdue) overdue++;
    }
  }
  return { open, overdue, review, approved };
}

/** Next step a card offers on the board, or null when none applies to this user. */
export function quickAction(item: TaskItem): "start" | "done" | "approve" | null {
  if (item.canWork && (item.status === "pending" || item.status === "rejected")) return "start";
  if (item.canWork && item.status === "in_progress" && item.subtaskCount === 0) return "done";
  if (item.canReview && item.status === "done") return "approve";
  return null;
}

// ── Quick capture ─────────────────────────────────────────────
// One line becomes a personal task. Trailing markers are stripped from the
// title: "besok", "lusa", "hari ini", a day name, "setiap hari|minggu|bulan",
// "!" (high) or "!!" (urgent). Example: "Kirim laporan besok !!".

const DAY_NAMES = ["minggu", "senin", "selasa", "rabu", "kamis", "jumat", "sabtu"];

function weekdayOf(day: string): number {
  const [y, m, d] = day.split("-").map(Number);
  return new Date(Date.UTC(y, m - 1, d)).getUTCDay();
}

function nextWeekday(today: string, weekday: number): string {
  return addDays(today, (weekday - weekdayOf(today) + 7) % 7 || 7);
}

export type QuickCapture = { title: string; dueDate: string; recurrence: Recurrence; priority?: Priority };

export function parseQuickCapture(text: string, today: string): QuickCapture | null {
  const words = text.trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) return null;
  let dueDate = today;
  let priority: Priority | undefined;
  let recurrence: Recurrence = "once";
  while (words.length > 1) {
    const last = words[words.length - 1].toLowerCase();
    const prev = words[words.length - 2].toLowerCase();
    if (last === "!!") {
      priority = "urgent";
      words.pop();
    } else if (last === "!") {
      priority = "high";
      words.pop();
    } else if (last === "besok") {
      dueDate = addDays(today, 1);
      words.pop();
    } else if (last === "lusa") {
      dueDate = addDays(today, 2);
      words.pop();
    } else if (last === "ini" && prev === "hari" && words.length > 2) {
      dueDate = today;
      words.splice(-2, 2);
    } else if (prev === "setiap" && words.length > 2 && (last === "hari" || last === "minggu" || last === "bulan")) {
      recurrence = last === "hari" ? "daily" : last === "minggu" ? "weekly" : "monthly";
      words.splice(-2, 2);
    } else if (DAY_NAMES.includes(last)) {
      dueDate = nextWeekday(today, DAY_NAMES.indexOf(last));
      words.pop();
    } else break;
  }
  const title = words.join(" ").trim();
  if (!title) return null;
  return { title, dueDate, recurrence, ...(priority ? { priority } : {}) };
}

/** Schedule fields of a create request from a recurrence and a reference date. */
export function scheduleFor(recurrence: Recurrence, dueDate: string) {
  switch (recurrence) {
    case "weekly":
      return { weeklyDay: weekdayOf(dueDate) || 7 };
    case "monthly":
      return { monthlyDay: Math.min(Number(dueDate.slice(8, 10)), 28) };
    case "daily":
      return {};
    default:
      return { dueDate };
  }
}

/** Splits a textarea into trimmed non-empty lines. */
export const lines = (text: string) =>
  text
    .split("\n")
    .map((s) => s.trim())
    .filter(Boolean);

export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${Math.round(n / 1024)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1).replace(".", ",")} MB`;
}
