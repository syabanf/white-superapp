export type SearchParamsRecord = Record<string, string | string[] | undefined>;

/** Search params every dashboard carries into its drill targets. */
export const PERIOD_KEYS = ["from", "to", "preset", "compare"] as const;

/** Keep only `keys` from a page's search params, as a query string without "?". */
export function pickQuery(sp: SearchParamsRecord, keys: readonly string[] = PERIOD_KEYS): string {
  const q = new URLSearchParams();
  for (const k of keys) {
    const v = sp[k];
    if (typeof v === "string" && v) q.set(k, v);
  }
  return q.toString();
}

/** `path` plus `keep` plus `extra` params (null/undefined values are dropped), optional `#hash`. */
export function withQuery(
  path: string,
  keep: string,
  extra: Record<string, string | number | null | undefined> = {},
  hash?: string,
): string {
  const q = new URLSearchParams(keep);
  for (const [k, v] of Object.entries(extra)) {
    if (v == null || v === "") q.delete(k);
    else q.set(k, String(v));
  }
  const qs = q.toString();
  return `${path}${qs ? `?${qs}` : ""}${hash ? `#${hash}` : ""}`;
}

/** Set one param on an href that may already carry a query string and a hash. */
export function withParam(href: string, key: string, value: string | number | null): string {
  const [rest, hash] = href.split("#");
  const [path, qs = ""] = rest!.split("?");
  return withQuery(path!, qs, { [key]: value }, hash);
}
