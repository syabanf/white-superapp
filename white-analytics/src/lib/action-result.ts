/**
 * Typed result for server actions. Never throw raw errors to the client —
 * return { ok: false, error } with a user-friendly Indonesian message.
 */
export type ActionResult<T = void> =
  | { ok: true; data: T }
  | { ok: false; error: string; code?: ProviderErrorCode | "VALIDATION" | "UNAUTHORIZED"; fieldErrors?: Record<string, string[]> };

export type ProviderErrorCode = "RATE_LIMIT" | "TOKEN_EXPIRED" | "PERMISSION" | "NOT_FOUND" | "NETWORK" | "UNKNOWN";

export function ok<T>(data: T): ActionResult<T> {
  return { ok: true, data };
}

export function fail<T = never>(error: string, code?: ActionResult<T> extends { ok: false } ? never : NonNullable<Extract<ActionResult<T>, { ok: false }>["code"]>, fieldErrors?: Record<string, string[]>): ActionResult<T> {
  return { ok: false, error, code, fieldErrors };
}

/** Normalized provider error — thrown by adapters, caught at the action boundary */
export class ProviderError extends Error {
  code: ProviderErrorCode;
  provider: string;
  requiresReconnect: boolean;
  constructor(provider: string, code: ProviderErrorCode, message: string, opts: { requiresReconnect?: boolean; cause?: unknown } = {}) {
    super(message, { cause: opts.cause });
    this.name = "ProviderError";
    this.provider = provider;
    this.code = code;
    this.requiresReconnect = opts.requiresReconnect ?? (code === "TOKEN_EXPIRED" || code === "PERMISSION");
  }
}

export function isProviderError(e: unknown): e is ProviderError {
  return e instanceof ProviderError;
}
