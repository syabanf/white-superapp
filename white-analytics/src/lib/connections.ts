import "server-only";
import { db } from "@/lib/db";
import { decrypt, encrypt } from "@/lib/crypto";
import { ProviderError } from "@/lib/action-result";
import { googleConfigured, refreshGoogleAccessToken } from "@/lib/providers/google-oauth";
import { metaConfigured } from "@/lib/providers/meta-oauth";

/**
 * Access-token resolution for sync jobs.
 *
 * `null` means "no real credentials for this resource" — the caller either
 * falls back to demo behaviour or skips. Google tokens are refreshed when they
 * expire within a minute; Meta long-lived tokens cannot be refreshed silently,
 * so an expired one throws TOKEN_EXPIRED (→ "Hubungkan ulang" in the UI).
 */
export type ResolvedToken = { token: string; connectionId: string };

const REFRESH_SKEW_MS = 60_000;

export async function getConnectionToken(connectionId: string | null | undefined): Promise<ResolvedToken | null> {
  if (!connectionId) return null;
  const c = await db.connection.findUnique({
    where: { id: connectionId },
    select: { id: true, provider: true, accessTokenEnc: true, refreshTokenEnc: true, expiresAt: true },
  });
  if (!c) return null;
  const configured = c.provider === "GOOGLE" ? googleConfigured() : metaConfigured();
  if (!configured) return null;

  const expiring = c.expiresAt ? c.expiresAt.getTime() - Date.now() < REFRESH_SKEW_MS : false;
  if (!expiring) return { token: decrypt(c.accessTokenEnc), connectionId: c.id };

  if (c.provider === "GOOGLE" && c.refreshTokenEnc) {
    const fresh = await refreshGoogleAccessToken({
      clientId: process.env.GOOGLE_CLIENT_ID!,
      clientSecret: process.env.GOOGLE_CLIENT_SECRET!,
      refreshToken: decrypt(c.refreshTokenEnc),
    });
    await db.connection.update({
      where: { id: c.id },
      data: { accessTokenEnc: encrypt(fresh.accessToken), expiresAt: fresh.expiresIn ? new Date(Date.now() + fresh.expiresIn * 1000) : null },
    });
    return { token: fresh.accessToken, connectionId: c.id };
  }
  throw new ProviderError(c.provider.toLowerCase(), "TOKEN_EXPIRED", "Token koneksi kedaluwarsa — hubungkan ulang di Pengaturan Klien.");
}
