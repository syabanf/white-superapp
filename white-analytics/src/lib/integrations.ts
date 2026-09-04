import "server-only";
import { cache } from "react";
import { db } from "@/lib/db";
import { decrypt, encrypt } from "@/lib/crypto";
import {
  INTEGRATIONS,
  INTEGRATION_KEYS,
  isConfiguredPair,
  maskSecret,
  type IntegrationKey,
} from "@/features/setup/integrations";

/**
 * App-level integration credentials.
 *
 * Source of truth for adapters stays `process.env` (adapters never touch the
 * DB). Values saved through the setup wizard are stored encrypted in
 * `IntegrationSetting` and copied into process.env on boot (instrumentation.ts)
 * and immediately on save — DB wins over a deployed env var, because the admin
 * changing it in the UI is the more recent intent.
 */

export type IntegrationStatus = {
  key: IntegrationKey;
  configured: boolean;
  /** where the effective value comes from */
  source: "db" | "env" | "none";
  publicId: string | null;
  secretMasked: string | null;
  meta: unknown;
  /** ISO */
  verifiedAt: string | null;
};

type Row = { provider: string; publicId: string | null; secretEnc: string | null; meta: unknown; verifiedAt: Date | null };

function applyRow(row: Row): void {
  const def = INTEGRATIONS[row.provider as IntegrationKey];
  if (!def) return;
  if (row.publicId) process.env[def.publicEnv] = row.publicId;
  if (row.secretEnc) {
    try {
      process.env[def.secretEnv] = decrypt(row.secretEnc);
    } catch {
      /* wrong ENCRYPTION_KEY — leave env untouched */
    }
  }
}

/** Copy every stored credential into process.env. Safe to call repeatedly. */
export async function hydrateIntegrationEnv(): Promise<number> {
  const rows = await db.integrationSetting.findMany();
  for (const r of rows) applyRow(r);
  return rows.length;
}

export const getIntegrationStatuses = cache(async (): Promise<Record<IntegrationKey, IntegrationStatus>> => {
  const rows = await db.integrationSetting.findMany();
  const byKey = new Map(rows.map((r) => [r.provider, r]));
  const out = {} as Record<IntegrationKey, IntegrationStatus>;
  for (const key of INTEGRATION_KEYS) {
    const def = INTEGRATIONS[key];
    const row = byKey.get(key);
    let secret: string | null = null;
    if (row?.secretEnc) {
      try {
        secret = decrypt(row.secretEnc);
      } catch {
        secret = null;
      }
    }
    const fromDb = Boolean(row && (row.publicId || secret));
    const publicId = (fromDb ? row?.publicId : null) ?? process.env[def.publicEnv] ?? null;
    const effSecret = secret ?? process.env[def.secretEnv] ?? null;
    const configured = isConfiguredPair(def, publicId, effSecret);
    out[key] = {
      key,
      configured,
      source: fromDb ? "db" : configured ? "env" : "none",
      publicId,
      secretMasked: effSecret ? maskSecret(effSecret) : null,
      meta: row?.meta ?? null,
      verifiedAt: row?.verifiedAt ? row.verifiedAt.toISOString() : null,
    };
  }
  return out;
});

export async function saveIntegration(
  key: IntegrationKey,
  input: { publicId?: string | null; secret?: string | null; meta?: unknown; verifiedAt?: Date | null; updatedById?: string },
): Promise<void> {
  const data: { publicId?: string | null; secretEnc?: string | null; meta?: object; verifiedAt?: Date | null; updatedById?: string } = {};
  if (input.publicId !== undefined) data.publicId = input.publicId || null;
  if (input.secret !== undefined && input.secret !== null && input.secret !== "") data.secretEnc = encrypt(input.secret);
  if (input.secret === null) data.secretEnc = null;
  if (input.meta !== undefined) data.meta = input.meta as object;
  if (input.verifiedAt !== undefined) data.verifiedAt = input.verifiedAt;
  if (input.updatedById) data.updatedById = input.updatedById;
  const row = await db.integrationSetting.upsert({ where: { provider: key }, create: { provider: key, ...data }, update: data });
  applyRow(row);
  // Clearing a secret must also clear the live env, otherwise the adapter stays "real".
  if (input.secret === null) delete process.env[INTEGRATIONS[key].secretEnv];
  if (input.publicId === null || input.publicId === "") delete process.env[INTEGRATIONS[key].publicEnv];
}

export async function clearIntegration(key: IntegrationKey): Promise<void> {
  await db.integrationSetting.deleteMany({ where: { provider: key } });
  delete process.env[INTEGRATIONS[key].publicEnv];
  delete process.env[INTEGRATIONS[key].secretEnv];
}

// ── workspace flags ───────────────────────────────────────────

export type SetupState = { completedAt: string | null; skippedAt: string | null };

export const getSetupState = cache(async (): Promise<SetupState> => {
  const row = await db.appSetting.findUnique({ where: { key: "setup" } });
  const v = (row?.value ?? {}) as Partial<SetupState>;
  return { completedAt: v.completedAt ?? null, skippedAt: v.skippedAt ?? null };
});

export async function markSetup(kind: "completedAt" | "skippedAt"): Promise<void> {
  const current = await getSetupState();
  const value = { ...current, [kind]: new Date().toISOString() };
  await db.appSetting.upsert({ where: { key: "setup" }, create: { key: "setup", value }, update: { value } });
}
