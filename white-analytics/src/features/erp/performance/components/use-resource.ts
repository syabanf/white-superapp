"use client";

import * as React from "react";
import { erpClientFetch } from "@/features/erp/shared/client";
import { sp } from "../strings";

async function settle<T>(path: string): Promise<{ path: string; data?: T; error?: string }> {
  try {
    return { path, data: await erpClientFetch<T>(path) };
  } catch (e) {
    return { path, error: e instanceof Error ? e.message : sp.common.loadFailed };
  }
}

/**
 * Loads a gateway path through the proxy for dialogs and sheets. Results are
 * keyed by path, so switching to another record never shows the previous one.
 */
export function useResource<T>(path: string | null) {
  const [state, setState] = React.useState<{ path: string; data?: T; error?: string } | null>(null);
  const reload = React.useCallback(async () => {
    if (path) setState(await settle<T>(path));
  }, [path]);
  React.useEffect(() => {
    let live = true;
    if (path) void settle<T>(path).then((next) => live && setState(next));
    return () => {
      live = false;
    };
  }, [path]);
  const current = state?.path === path ? state : null;
  return { data: current?.data ?? null, error: current?.error ?? null, reload };
}
