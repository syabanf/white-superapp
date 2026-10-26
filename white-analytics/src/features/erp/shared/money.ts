/**
 * Money helpers for ERP payloads. The Go API sends money as decimal strings
 * ("1250000.00"); the UI shows rupiah without decimals and sends strings back.
 */

/** "1250000.00" | 1250000 | null → 1250000 (0 for empty or invalid). */
export function parseMoney(value: string | number | null | undefined): number {
  if (value == null || value === "") return 0;
  const n = typeof value === "number" ? value : Number(value);
  return Number.isFinite(n) ? n : 0;
}

/** 1250000 → "1250000.00", the decimal string the API expects. */
export function toMoneyString(value: number): string {
  return (Math.round(value * 100) / 100).toFixed(2);
}

const rupiah = new Intl.NumberFormat("id-ID", { maximumFractionDigits: 0, minimumFractionDigits: 0 });

/** 1250000 → "Rp 1.250.000", negatives as "-Rp 1.250.000". */
export function formatRupiah(value: string | number | null | undefined, opts: { symbol?: boolean } = {}): string {
  const n = parseMoney(value);
  const body = rupiah.format(Math.abs(Math.round(n)));
  const sign = Math.round(n) < 0 ? "-" : "";
  return opts.symbol === false ? `${sign}${body}` : `${sign}Rp ${body}`;
}

/** "Rp 1.250.000" or "1.250.000" typed by a user → 1250000. Commas are decimals (id-ID). */
export function parseRupiahInput(text: string): number {
  const cleaned = text.replace(/[^\d,-]/g, "").replace(",", ".");
  const n = Number(cleaned);
  return Number.isFinite(n) ? n : 0;
}

/** Sum of decimal strings, rounded to cents to avoid float drift. */
export function sumMoney(values: (string | number | null | undefined)[]): number {
  return Math.round(values.reduce<number>((acc, v) => acc + parseMoney(v) * 100, 0)) / 100;
}
