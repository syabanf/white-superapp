import "server-only";
import { resolveRange, type RangeSearchParams, type ResolvedRange } from "@/lib/dates";

export type SearchParamsLike = Record<string, string | string[] | undefined>;

/** Resolve the global date range from a page's (awaited) searchParams. */
export function getRange(searchParams: SearchParamsLike): ResolvedRange {
  return resolveRange(searchParams as RangeSearchParams);
}
