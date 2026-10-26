import "server-only";
import type { z } from "zod";
import { fail, ok, type ActionResult } from "@/lib/action-result";
import { GatewayError } from "@/lib/erp/gateway";
import { ss } from "./strings";

/** Zod issues keyed by the first path segment, for form field errors. */
export function zodFieldErrors(err: z.ZodError): Record<string, string[]> {
  const out: Record<string, string[]> = {};
  for (const issue of err.issues) {
    const key = issue.path.length ? issue.path.map(String).join(".") : "form";
    (out[key] ??= []).push(issue.message);
  }
  return out;
}

/** Maps a thrown gateway error onto an ActionResult the client can toast. */
export function gatewayFail<T = never>(e: unknown): ActionResult<T> {
  if (e instanceof GatewayError) {
    if (e.code === "UNAUTHORIZED" || e.code === "UNAUTHENTICATED")
      return fail(e.message || ss.permissionDenied, "UNAUTHORIZED");
    if (e.code === "VALIDATION") return fail(e.message, "VALIDATION", e.fieldErrors && camelKeys(e.fieldErrors));
    if (e.code === "NOT_FOUND") return fail(e.message, "NOT_FOUND");
    if (e.code === "NETWORK") return fail(e.message, "NETWORK");
    return fail(e.message);
  }
  return fail(ss.unknownError);
}

/** Some services name fields in snake_case ("meeting_link"); forms use the camelCase JSON names. */
function camelKeys(errors: Record<string, string[]>): Record<string, string[]> {
  return Object.fromEntries(
    Object.entries(errors).map(([k, v]) => [k.replace(/_([a-z])/g, (_, c: string) => c.toUpperCase()), v]),
  );
}

/** Runs a gateway mutation and wraps the result. */
export async function runGateway<T>(fn: () => Promise<T>): Promise<ActionResult<T>> {
  try {
    return ok(await fn());
  } catch (e) {
    return gatewayFail<T>(e);
  }
}
