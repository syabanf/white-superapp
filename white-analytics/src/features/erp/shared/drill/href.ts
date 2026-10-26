/** URL helpers for drill links. Pure; safe on server and client. */

export type ParamPatch = Record<string, string | number | null | undefined>;

/**
 * `pathname` + current params with `patch` applied (null/undefined/"" removes a key).
 * `drop` removes more keys, e.g. `page` or `drill` when the filter changes.
 */
export function withParams(
  pathname: string,
  current: URLSearchParams | string | Record<string, string | string[] | undefined> | null | undefined,
  patch: ParamPatch,
  drop: readonly string[] = ["page"],
): string {
  const params = toSearchParams(current);
  for (const key of drop) params.delete(key);
  for (const [key, value] of Object.entries(patch)) {
    if (value == null || value === "") params.delete(key);
    else params.set(key, String(value));
  }
  const qs = params.toString();
  return qs ? `${pathname}?${qs}` : pathname;
}

function toSearchParams(
  current: URLSearchParams | string | Record<string, string | string[] | undefined> | null | undefined,
): URLSearchParams {
  if (!current) return new URLSearchParams();
  if (typeof current === "string" || current instanceof URLSearchParams) return new URLSearchParams(current);
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(current)) {
    if (Array.isArray(value)) value.forEach((v) => params.append(key, v));
    else if (value != null) params.set(key, value);
  }
  return params;
}

export const DRILL_PARAM = "drill";

export type DrillTarget = { kind: string; key: string };

/** "month:2026-03" → {kind: "month", key: "2026-03"}; the key may contain colons. */
export function parseDrill(raw: string | null | undefined): DrillTarget | null {
  if (!raw) return null;
  const i = raw.indexOf(":");
  if (i <= 0) return null;
  return { kind: raw.slice(0, i), key: raw.slice(i + 1) };
}

export function formatDrill(kind: string, key: string | number): string {
  return `${kind}:${key}`;
}
