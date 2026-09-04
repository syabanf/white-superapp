/**
 * Timezone + calendar helpers for publishing (pure, Intl only — no deps).
 * All "calendar days" are `YYYY-MM-DD` strings in the CLIENT's timezone;
 * instants are UTC `Date`s. Tested in tests/publishing.test.ts.
 */

export type ZonedParts = { date: string; time: string; year: number; month: number; day: number; hour: number; minute: number; weekday: number };

const partsCache = new Map<string, Intl.DateTimeFormat>();
function fmt(timeZone: string): Intl.DateTimeFormat {
  let f = partsCache.get(timeZone);
  if (!f) {
    f = new Intl.DateTimeFormat("en-US", {
      timeZone,
      hourCycle: "h23",
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      weekday: "short",
    });
    partsCache.set(timeZone, f);
  }
  return f;
}

const WD: Record<string, number> = { Mon: 0, Tue: 1, Wed: 2, Thu: 3, Fri: 4, Sat: 5, Sun: 6 };
const pad = (n: number) => String(n).padStart(2, "0");

/** Break an instant into wall-clock parts in `timeZone` (weekday Mon=0). */
export function toZoned(date: Date, timeZone: string): ZonedParts {
  const p: Record<string, string> = {};
  for (const part of fmt(timeZone).formatToParts(date)) p[part.type] = part.value;
  const hour = Number(p.hour) === 24 ? 0 : Number(p.hour);
  const year = Number(p.year);
  const month = Number(p.month);
  const day = Number(p.day);
  return {
    year,
    month,
    day,
    hour,
    minute: Number(p.minute),
    weekday: WD[p.weekday ?? "Mon"] ?? 0,
    date: `${year}-${pad(month)}-${pad(day)}`,
    time: `${pad(hour)}:${pad(Number(p.minute))}`,
  };
}

/** Offset of `timeZone` from UTC in minutes at the given instant (Asia/Jakarta → 420). */
export function tzOffsetMinutes(date: Date, timeZone: string): number {
  const z = toZoned(date, timeZone);
  const asUtc = Date.UTC(z.year, z.month - 1, z.day, z.hour, z.minute, 0, 0);
  const truncated = Math.floor(date.getTime() / 60000) * 60000;
  return Math.round((asUtc - truncated) / 60000);
}

/** Wall-clock `YYYY-MM-DD` + `HH:mm` in `timeZone` → UTC instant. */
export function fromZoned(date: string, time: string, timeZone: string): Date {
  const [y, m, d] = date.split("-").map(Number) as [number, number, number];
  const [hh, mm] = time.split(":").map(Number) as [number, number];
  const naive = Date.UTC(y, m - 1, d, hh, mm || 0, 0, 0);
  // Two passes handle offsets that change around the target instant.
  let guess = naive - tzOffsetMinutes(new Date(naive), timeZone) * 60000;
  guess = naive - tzOffsetMinutes(new Date(guess), timeZone) * 60000;
  return new Date(guess);
}

export function todayIn(timeZone: string, now: Date = new Date()): string {
  return toZoned(now, timeZone).date;
}

export function isValidDateKey(v: string | undefined | null): v is string {
  if (!v || !/^\d{4}-\d{2}-\d{2}$/.test(v)) return false;
  const [y, m, d] = v.split("-").map(Number) as [number, number, number];
  const dt = new Date(Date.UTC(y, m - 1, d));
  return dt.getUTCFullYear() === y && dt.getUTCMonth() === m - 1 && dt.getUTCDate() === d;
}

export function parseDateKey(raw: string | string[] | undefined, fallback: string): string {
  const v = Array.isArray(raw) ? raw[0] : raw;
  return isValidDateKey(v) ? v : fallback;
}

// ── day-key arithmetic (calendar dates, no tz involved) ───────

function keyToUtc(key: string): Date {
  const [y, m, d] = key.split("-").map(Number) as [number, number, number];
  return new Date(Date.UTC(y, m - 1, d));
}
function utcToKey(d: Date): string {
  return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())}`;
}

export function shiftDays(key: string, n: number): string {
  const d = keyToUtc(key);
  d.setUTCDate(d.getUTCDate() + n);
  return utcToKey(d);
}

export function shiftMonths(key: string, n: number): string {
  const d = keyToUtc(key);
  const day = d.getUTCDate();
  d.setUTCDate(1);
  d.setUTCMonth(d.getUTCMonth() + n);
  const last = new Date(Date.UTC(d.getUTCFullYear(), d.getUTCMonth() + 1, 0)).getUTCDate();
  d.setUTCDate(Math.min(day, last));
  return utcToKey(d);
}

/** Monday of the week containing `key`. */
export function startOfWeek(key: string): string {
  const d = keyToUtc(key);
  const wd = (d.getUTCDay() + 6) % 7;
  return shiftDays(key, -wd);
}

export function weekDays(key: string): string[] {
  const start = startOfWeek(key);
  return Array.from({ length: 7 }, (_, i) => shiftDays(start, i));
}

export function monthOf(key: string): string {
  return key.slice(0, 7);
}

/** 6 rows × 7 days (Mon-first) covering the month of `key`, padded with neighbours. */
export function monthGrid(key: string): string[][] {
  const first = `${monthOf(key)}-01`;
  let cursor = startOfWeek(first);
  const rows: string[][] = [];
  for (let r = 0; r < 6; r++) {
    const row: string[] = [];
    for (let c = 0; c < 7; c++) {
      row.push(cursor);
      cursor = shiftDays(cursor, 1);
    }
    rows.push(row);
  }
  return rows;
}

/** UTC instants that bound a set of client-timezone calendar days, `[from, to)`. */
export function rangeForDays(days: string[], timeZone: string): { from: Date; to: Date } {
  const first = days[0]!;
  const last = days[days.length - 1]!;
  return { from: fromZoned(first, "00:00", timeZone), to: fromZoned(shiftDays(last, 1), "00:00", timeZone) };
}

/** Next `count` instants after `from` (+lead) matching each (weekday, hour) slot, in `timeZone`. */
export function nextOccurrences(
  slots: { day: number; hour: number }[],
  timeZone: string,
  from: Date = new Date(),
  leadMs = 5 * 60 * 1000,
): Date[] {
  const min = new Date(from.getTime() + leadMs);
  const today = toZoned(min, timeZone);
  return slots.map((s) => {
    for (let i = 0; i < 8; i++) {
      const key = shiftDays(today.date, i);
      const wd = (keyToUtc(key).getUTCDay() + 6) % 7;
      if (wd !== s.day) continue;
      const at = fromZoned(key, `${pad(s.hour)}:00`, timeZone);
      if (at.getTime() >= min.getTime()) return at;
    }
    return fromZoned(shiftDays(today.date, 7), `${pad(s.hour)}:00`, timeZone);
  });
}

/** Move an instant to another calendar day, keeping its wall-clock time in `timeZone`. */
export function moveToDay(instant: Date, dayKey: string, timeZone: string): Date {
  const z = toZoned(instant, timeZone);
  return fromZoned(dayKey, z.time, timeZone);
}

const WEEKDAY_ID = ["Sen", "Sel", "Rab", "Kam", "Jum", "Sab", "Min"];
const WEEKDAY_LONG_ID = ["Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu", "Minggu"];
const MONTH_ID = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];
const MONTH_LONG_ID = ["Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"];

export function weekdayLabel(idx: number, long = false): string {
  return (long ? WEEKDAY_LONG_ID : WEEKDAY_ID)[((idx % 7) + 7) % 7]!;
}

/** "September 2026" */
export function monthLabel(key: string): string {
  const [y, m] = key.split("-").map(Number) as [number, number];
  return `${MONTH_LONG_ID[m - 1]} ${y}`;
}

/** "2 Sep" / "2 Sep 2026" */
export function dayLabel(key: string, withYear = false): string {
  const [y, m, d] = key.split("-").map(Number) as [number, number, number];
  return `${d} ${MONTH_ID[m - 1]}${withYear ? ` ${y}` : ""}`;
}

/** "Sen, 2 Sep 2026" */
export function dayLabelLong(key: string): string {
  const wd = (keyToUtc(key).getUTCDay() + 6) % 7;
  return `${weekdayLabel(wd)}, ${dayLabel(key, true)}`;
}

/** "14.05" — clock time in `timeZone` (Indonesian style separator). */
export function clock(date: Date | string, timeZone: string): string {
  const z = toZoned(typeof date === "string" ? new Date(date) : date, timeZone);
  return `${pad(z.hour)}.${pad(z.minute)}`;
}

/** Short tz label like "WIB" / "GMT+7" for the given zone. */
export function tzLabel(timeZone: string, at: Date = new Date()): string {
  if (timeZone === "Asia/Jakarta") return "WIB";
  if (timeZone === "Asia/Makassar") return "WITA";
  if (timeZone === "Asia/Jayapura") return "WIT";
  try {
    const part = new Intl.DateTimeFormat("en-US", { timeZone, timeZoneName: "short" }).formatToParts(at).find((p) => p.type === "timeZoneName");
    return part?.value ?? timeZone;
  } catch {
    return timeZone;
  }
}
