"use client";

import * as React from "react";
import { parseAsString, useQueryStates } from "nuqs";

/**
 * URL-backed filter state for server-rendered lists. Any change refetches
 * the page (shallow: false) and resets `page` unless `keepPage` is set.
 */
export function useUrlFilters<K extends string>(keys: readonly K[]) {
  const [isPending, startTransition] = React.useTransition();
  const signature = keys.join("|");
  const parsers = React.useMemo(
    () =>
      Object.fromEntries([...signature.split("|"), "page"].map((k) => [k, parseAsString])) as Record<
        K | "page",
        typeof parseAsString
      >,
    [signature],
  );
  const [values, setValues] = useQueryStates(parsers, { shallow: false, history: "replace", startTransition });
  const set = React.useCallback(
    (patch: Partial<Record<K | "page", string | null>>, opts: { keepPage?: boolean } = {}) => {
      const next = Object.fromEntries(Object.entries(patch).map(([k, v]) => [k, v === "" ? null : v]));
      // nuqs typing cannot follow the dynamic key set; values are string | null either way
      void setValues((opts.keepPage || "page" in patch ? next : { ...next, page: null }) as never);
    },
    [setValues],
  );
  return { values: values as Record<K | "page", string | null>, set, isPending };
}
