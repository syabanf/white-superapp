"use client";

import * as React from "react";
import type { ActionResult } from "@/lib/action-result";
import { toast } from "@/lib/toast";
import { ss } from "./strings";

/** Error thrown by erpClientFetch; message is user-safe Indonesian from the gateway. */
export class ErpClientError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
  }
}

/** JSON call from client components through the same-origin proxy (`/api/erp` + path). */
export async function erpClientFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`/api/erp${path}`, {
      ...init,
      headers: { "Content-Type": "application/json", ...init.headers },
    });
  } catch {
    throw new ErpClientError(503, "NETWORK", ss.unknownError);
  }
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new ErpClientError(res.status, body.code ?? "UNKNOWN", body.error ?? ss.unknownError);
  return body as T;
}

/**
 * Runs a server action in a transition and toasts the outcome.
 * `run(fn, {success, onOk})` returns the result so callers can read field errors.
 */
export function useErpAction() {
  const [pending, startTransition] = React.useTransition();
  const run = React.useCallback(
    <T,>(
      fn: () => Promise<ActionResult<T>>,
      opts: { success?: string; onOk?: (data: T) => void; onFail?: (res: Extract<ActionResult<T>, { ok: false }>) => void } = {},
    ) =>
      new Promise<ActionResult<T>>((resolve) => {
        startTransition(async () => {
          const res = await fn();
          if (res.ok) {
            if (opts.success) toast.success(opts.success);
            opts.onOk?.(res.data);
          } else {
            toast.error(res.error);
            opts.onFail?.(res);
          }
          resolve(res);
        });
      }),
    [],
  );
  return { pending, run };
}
