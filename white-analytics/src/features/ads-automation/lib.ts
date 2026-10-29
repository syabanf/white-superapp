/**
 * Pure helpers for recurring reports (tested in tests/report-schedule.test.ts).
 * All dates are UTC midnight, like every other @db.Date in the app.
 *
 *  - WEEKLY runs on Monday and covers the last full week (Monday to Sunday).
 *  - MONTHLY runs on the 1st and covers the previous calendar month.
 * The daily cron (02:00 UTC) picks up every schedule whose `nextRunAt` has passed.
 */
import { addDays, endOfMonthUtc, todayUtc, utcDate, type DateRange } from "@/lib/dates";

export type Cadence = "WEEKLY" | "MONTHLY";
export const CADENCES: Cadence[] = ["WEEKLY", "MONTHLY"];

/** Next run strictly after the day of `from`: next Monday (weekly) or the 1st of next month. */
export function computeNextRun(cadence: Cadence, from: Date): Date {
  const day = todayUtc(from);
  if (cadence === "MONTHLY") return utcDate(day.getUTCFullYear(), day.getUTCMonth() + 1, 1);
  const dow = day.getUTCDay(); // 0 = Sunday, 1 = Monday
  return addDays(day, (8 - dow) % 7 || 7);
}

/** The period a run at `runAt` reports on: last full Mon–Sun week, or the previous month. */
export function reportPeriod(cadence: Cadence, runAt: Date): DateRange {
  const day = todayUtc(runAt);
  if (cadence === "MONTHLY") {
    const from = utcDate(day.getUTCFullYear(), day.getUTCMonth() - 1, 1);
    return { from, to: endOfMonthUtc(from) };
  }
  const thisMonday = addDays(day, -((day.getUTCDay() + 6) % 7));
  return { from: addDays(thisMonday, -7), to: addDays(thisMonday, -1) };
}

/** A schedule is due once its nextRunAt has passed (enabled only). */
export function isDue(s: { enabled: boolean; nextRunAt: Date }, now: Date): boolean {
  return s.enabled && s.nextRunAt.getTime() <= now.getTime();
}

const MONTH = new Intl.DateTimeFormat("id-ID", { month: "long", year: "numeric", timeZone: "UTC" });
const DAY = new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short", timeZone: "UTC" });
const DAY_Y = new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" });

/** "Oktober 2026" for a month, "20 Okt – 26 Okt 2026" for a week. */
export function periodLabel(cadence: Cadence, period: DateRange): string {
  if (cadence === "MONTHLY") return MONTH.format(period.from);
  return `${DAY.format(period.from)} – ${DAY_Y.format(period.to)}`;
}

/** Report title for a scheduled run: "<schedule name> · <period>". */
export function scheduledReportTitle(name: string, cadence: Cadence, period: DateRange): string {
  return `${name.trim()} · ${periodLabel(cadence, period)}`.slice(0, 160);
}

/** Auto weekly ADS insight: regenerate only when the last one is ≥ 7 days old. */
export const AUTO_INSIGHT_INTERVAL_DAYS = 7;

export function isInsightStale(lastCreatedAt: Date | null, now: Date): boolean {
  if (!lastCreatedAt) return true;
  return now.getTime() - lastCreatedAt.getTime() >= AUTO_INSIGHT_INTERVAL_DAYS * 86_400_000 - 60 * 60 * 1000;
}
