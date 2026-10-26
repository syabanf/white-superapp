"use client";

import * as React from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { parseAsString, useQueryState } from "nuqs";
import { DRILL_PARAM, formatDrill, parseDrill, withParams, type DrillTarget, type ParamPatch } from "./href";

// Drill entries this tab pushed and has not popped yet. close() walks back
// through them so the browser Back button and the X button agree.
let pushedDepth = 0;

/**
 * The open drill sheet, stored in the URL as `?drill=<kind>:<key>`.
 * `open` pushes a history entry (Back closes the sheet); `close` pops it,
 * or strips the param when the page was loaded with it.
 * Shallow: opening a sheet does not re-render the server page.
 */
export function useDrill() {
  const [raw, setRaw] = useQueryState(
    DRILL_PARAM,
    parseAsString.withOptions({ history: "push", shallow: true, scroll: false }),
  );
  const drill = React.useMemo(() => parseDrill(raw), [raw]);
  const pathname = usePathname();
  const searchParams = useSearchParams();

  React.useEffect(() => {
    if (!raw) pushedDepth = 0;
  }, [raw]);

  const open = React.useCallback(
    (kind: string, key: string | number) => {
      pushedDepth += 1;
      void setRaw(formatDrill(kind, key));
    },
    [setRaw],
  );

  const close = React.useCallback(() => {
    if (pushedDepth > 0) {
      pushedDepth -= 1;
      window.history.back();
    } else {
      void setRaw(null, { history: "replace" });
    }
  }, [setRaw]);

  /** Link that opens a drill (for <Link href>, keeps every other param). */
  const hrefFor = React.useCallback(
    (kind: string, key: string | number) =>
      withParams(pathname, searchParams.toString(), { [DRILL_PARAM]: formatDrill(kind, key) }, []),
    [pathname, searchParams],
  );

  const isOpen = React.useCallback((kind: string) => drill?.kind === kind, [drill]);

  return { drill, open, close, hrefFor, isOpen };
}

/**
 * In-dashboard drill: apply or clear dashboard filters through the URL
 * (server re-render, new history entry so Back goes up a level).
 * `hrefWith` builds the same URL for links and breadcrumb items.
 */
export function useDrillParams() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [isPending, startTransition] = React.useTransition();

  const hrefWith = React.useCallback(
    (patch: ParamPatch, drop: readonly string[] = ["page", DRILL_PARAM]) =>
      withParams(pathname, searchParams.toString(), patch, drop),
    [pathname, searchParams],
  );

  const apply = React.useCallback(
    (patch: ParamPatch, drop?: readonly string[]) => {
      const href = hrefWith(patch, drop);
      startTransition(() => router.push(href, { scroll: false }));
    },
    [hrefWith, router],
  );

  const get = React.useCallback((key: string) => searchParams.get(key), [searchParams]);

  return { apply, hrefWith, get, isPending };
}

export type { DrillTarget };
