/**
 * Formatting helpers — Indonesian locale by default (id-ID), IDR currency.
 * Pure functions, safe on server and client.
 */

const LOCALE = "id-ID";

const intCache = new Map<string, Intl.NumberFormat>();
function nf(key: string, opts: Intl.NumberFormatOptions): Intl.NumberFormat {
  let f = intCache.get(key);
  if (!f) {
    f = new Intl.NumberFormat(LOCALE, opts);
    intCache.set(key, f);
  }
  return f;
}

/** 1234567 → "1.234.567" */
export function formatNumber(value: number | null | undefined, digits = 0): string {
  if (value == null || Number.isNaN(value)) return "–";
  return nf(`n${digits}`, { maximumFractionDigits: digits, minimumFractionDigits: digits }).format(value);
}

/** 1284 → "1,3 rb", 1_284_000 → "1,3 jt", 2_100_000_000 → "2,1 M" (Indonesian short scale) */
export function formatCompact(value: number | null | undefined, digits = 1): string {
  if (value == null || Number.isNaN(value)) return "–";
  const abs = Math.abs(value);
  const sign = value < 0 ? "-" : "";
  // NBSP keeps the unit glued to the number so chart axes never break it across lines
  const NB = "\u00a0";
  if (abs >= 1_000_000_000) return `${sign}${trimZero((abs / 1_000_000_000).toFixed(digits))}${NB}M`;
  if (abs >= 1_000_000) return `${sign}${trimZero((abs / 1_000_000).toFixed(digits))}${NB}jt`;
  if (abs >= 1_000) return `${sign}${trimZero((abs / 1_000).toFixed(digits))}${NB}rb`;
  return `${sign}${formatNumber(abs, abs % 1 === 0 ? 0 : digits)}`;
}

function trimZero(s: string): string {
  // "1.0" → "1", "1.50" → "1.5"; then swap decimal point for id-ID comma
  return s.replace(/\.0+$/, "").replace(/(\.\d*?)0+$/, "$1").replace(".", ",");
}

/** 1500000 → "Rp 1.500.000" (no decimals for IDR) */
export function formatCurrency(
  value: number | null | undefined,
  currency = "IDR",
  opts: { compact?: boolean } = {},
): string {
  if (value == null || Number.isNaN(value)) return "–";
  if (opts.compact) {
    const symbol = currency === "IDR" ? "Rp" : currency;
    return `${symbol} ${formatCompact(value)}`;
  }
  const digits = currency === "IDR" ? 0 : 2;
  return nf(`c${currency}${digits}`, {
    style: "currency",
    currency,
    maximumFractionDigits: digits,
    minimumFractionDigits: digits,
  })
    .format(value)
    .replace(/ /g, " ");
}

/** 4.234 → "4,23%" — value is already a percentage (0–100) */
export function formatPercent(value: number | null | undefined, digits = 2): string {
  if (value == null || Number.isNaN(value)) return "–";
  return `${formatNumber(value, digits)}%`;
}

/** Signed delta percent: 12.3 → "+12,3%", -4 → "−4,0%" */
export function formatDeltaPercent(value: number | null | undefined, digits = 1): string {
  if (value == null || Number.isNaN(value) || !Number.isFinite(value)) return "–";
  const sign = value > 0 ? "+" : value < 0 ? "−" : "";
  return `${sign}${formatNumber(Math.abs(value), digits)}%`;
}

/** Signed absolute delta: 120 → "+120", -35 → "−35" */
export function formatDeltaNumber(value: number | null | undefined, digits = 0): string {
  if (value == null || Number.isNaN(value)) return "–";
  const sign = value > 0 ? "+" : value < 0 ? "−" : "";
  return `${sign}${formatNumber(Math.abs(value), digits)}`;
}

/** 2.5 → "2,5" — average position etc. */
export function formatDecimal(value: number | null | undefined, digits = 1): string {
  return formatNumber(value, digits);
}

/** 1830 ms → "1,8 dtk" ; 250 → "250 ms" */
export function formatMs(value: number | null | undefined): string {
  if (value == null || Number.isNaN(value)) return "–";
  if (value >= 1000) return `${formatNumber(value / 1000, 1)} dtk`;
  return `${formatNumber(value, 0)} ms`;
}

/** 95 → "1 mnt 35 dtk" */
export function formatDuration(seconds: number | null | undefined): string {
  if (seconds == null || Number.isNaN(seconds)) return "–";
  const s = Math.round(seconds);
  const m = Math.floor(s / 60);
  const rest = s % 60;
  if (m === 0) return `${rest} dtk`;
  return `${m} mnt ${rest} dtk`;
}

// ── Dates ────────────────────────────────────────────────────

const dtCache = new Map<string, Intl.DateTimeFormat>();
function df(key: string, opts: Intl.DateTimeFormatOptions): Intl.DateTimeFormat {
  let f = dtCache.get(key);
  if (!f) {
    f = new Intl.DateTimeFormat(LOCALE, { timeZone: "UTC", ...opts });
    dtCache.set(key, f);
  }
  return f;
}

/** "19 Agu 2026" */
export function formatDate(date: Date | string | null | undefined): string {
  if (!date) return "–";
  const d = typeof date === "string" ? new Date(date) : date;
  return df("d", { day: "numeric", month: "short", year: "numeric" }).format(d);
}

/** "19 Agu" */
export function formatDateShort(date: Date | string | null | undefined): string {
  if (!date) return "–";
  const d = typeof date === "string" ? new Date(date) : date;
  return df("ds", { day: "numeric", month: "short" }).format(d);
}

/** "19 Agustus 2026" */
export function formatDateLong(date: Date | string | null | undefined): string {
  if (!date) return "–";
  const d = typeof date === "string" ? new Date(date) : date;
  return df("dl", { day: "numeric", month: "long", year: "numeric" }).format(d);
}

/** "19 Agu 2026, 14.05" (Asia/Jakarta) */
export function formatDateTime(date: Date | string | null | undefined, timeZone = "Asia/Jakarta"): string {
  if (!date) return "–";
  const d = typeof date === "string" ? new Date(date) : date;
  return df(`dt${timeZone}`, {
    timeZone,
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(d);
}

/** "19 Agu – 15 Sep 2026" */
export function formatDateRange(from: Date, to: Date): string {
  const sameYear = from.getUTCFullYear() === to.getUTCFullYear();
  const left = sameYear ? formatDateShort(from) : formatDate(from);
  return `${left} – ${formatDate(to)}`;
}

/** Relative time in Indonesian: "3 jam lalu", "2 hari lalu" */
export function formatRelative(date: Date | string | null | undefined, now = new Date()): string {
  if (!date) return "–";
  const d = typeof date === "string" ? new Date(date) : date;
  const diff = (now.getTime() - d.getTime()) / 1000;
  if (diff < 60) return "baru saja";
  if (diff < 3600) return `${Math.floor(diff / 60)} mnt lalu`;
  if (diff < 86400) return `${Math.floor(diff / 3600)} jam lalu`;
  if (diff < 86400 * 30) return `${Math.floor(diff / 86400)} hari lalu`;
  return formatDate(d);
}

/** Truncate with ellipsis */
export function truncate(text: string, max = 60): string {
  if (text.length <= max) return text;
  return `${text.slice(0, max - 1).trimEnd()}…`;
}

/** Initials for avatars: "Kopi Nusantara" → "KN" */
export function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w[0]!.toUpperCase())
    .join("");
}

/** URL-safe slug: "Adiharjo Project" → "adiharjo-project" */
export function slugify(name: string): string {
  return name
    .toLowerCase()
    .normalize("NFKD")
    .replace(/[̀-ͯ]/g, "")
    .replace(/[^a-z0-9\s-]/g, "")
    .trim()
    .replace(/\s+/g, "-")
    .replace(/-+/g, "-");
}
