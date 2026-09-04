"use server";

import { compare } from "bcryptjs";
import { cookies, headers } from "next/headers";
import { redirect } from "next/navigation";
import { z } from "zod";
import { fail, type ActionResult } from "@/lib/action-result";
import { db } from "@/lib/db";
import { shareCookieName, shareCookieValue } from "@/features/share/access";
import { t } from "@/i18n/id";

/** In-memory rate limit: 5 attempts / 10 minutes per slug+IP (per server instance). */
const WINDOW_MS = 10 * 60 * 1000;
const MAX_ATTEMPTS = 5;
const COOKIE_MAX_AGE = 12 * 60 * 60; // 12h

type Bucket = { count: number; resetAt: number };
const globalStore = globalThis as unknown as { __sharePinAttempts?: Map<string, Bucket> };
const attempts: Map<string, Bucket> = (globalStore.__sharePinAttempts ??= new Map());

function tooManyAttempts(key: string): boolean {
  const now = Date.now();
  const bucket = attempts.get(key);
  if (!bucket || bucket.resetAt <= now) {
    attempts.set(key, { count: 1, resetAt: now + WINDOW_MS });
    return false;
  }
  bucket.count += 1;
  return bucket.count > MAX_ATTEMPTS;
}

const pinSchema = z.string().regex(/^\d{6}$/);

/**
 * Verify the 6-digit share PIN. On success sets the httpOnly unlock cookie (12h)
 * and redirects back to the share page; on failure returns an ActionResult error.
 */
export async function verifySharePin(slug: string, pin: string): Promise<ActionResult<never>> {
  const hdrs = await headers();
  const ip = hdrs.get("x-forwarded-for")?.split(",")[0]?.trim() || hdrs.get("x-real-ip") || "local";
  if (tooManyAttempts(`${slug}:${ip}`)) return fail(t.share.pinTooMany, "RATE_LIMIT");

  const parsed = pinSchema.safeParse(pin);
  if (!parsed.success) return fail(t.share.pinWrong, "VALIDATION");

  const client = await db.client.findUnique({
    where: { slug },
    select: { id: true, shareEnabled: true, sharePinHash: true },
  });
  if (!client || !client.shareEnabled) return fail(t.share.notFound, "NOT_FOUND");
  if (!client.sharePinHash) redirect(`/share/${slug}`);

  const valid = await compare(parsed.data, client.sharePinHash);
  if (!valid) return fail(t.share.pinWrong);

  const store = await cookies();
  store.set(shareCookieName(client.id), shareCookieValue(client.sharePinHash), {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    maxAge: COOKIE_MAX_AGE,
    path: "/",
  });
  redirect(`/share/${slug}`);
}

/** Form-friendly wrapper for useActionState. */
export async function verifySharePinForm(slug: string, _prev: ActionResult<never> | null, formData: FormData): Promise<ActionResult<never> | null> {
  const pin = String(formData.get("pin") ?? "");
  return verifySharePin(slug, pin);
}
