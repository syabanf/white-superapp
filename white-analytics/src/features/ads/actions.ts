"use server";

import { revalidatePath } from "next/cache";
import { z } from "zod";
import { db } from "@/lib/db";
import { requireUser } from "@/lib/rbac";
import { fail, ok, type ActionResult } from "@/lib/action-result";
import { t } from "@/i18n/id";
import { parseAdsCsv } from "./csv-import";
import { persistCsvRows, type ImportSummary } from "./csv-persist";
import { syncAdsForClient } from "./sync";
import { s } from "./strings";

const MAX_FILE_BYTES = 10 * 1024 * 1024; // 10MB

/** ADMIN atau MANAGER klien; null bila tidak berhak / klien tidak ada. */
async function requireClientManage(clientId: string): Promise<{ id: string; slug: string } | null> {
  const user = await requireUser();
  const client = await db.client.findUnique({ where: { id: clientId }, select: { id: true, slug: true } });
  if (!client) return null;
  if (user.role === "ADMIN") return client;
  const membership = await db.clientMember.findUnique({ where: { userId_clientId: { userId: user.id, clientId } } });
  return membership?.role === "MANAGER" ? client : null;
}

/**
 * Impor CSV ekspor Meta Ads Manager (field form "file", maks 10MB).
 * Parse (EN/ID) → upsert hierarki + AdDailyInsight + AdDemographic (idempoten:
 * impor ulang file yang sama tidak menggandakan data) → SyncJob ADS_CSV_IMPORT.
 */
export async function importAdsCsv(clientId: string, formData: FormData): Promise<ActionResult<ImportSummary>> {
  const parsedId = z.string().min(1).safeParse(clientId);
  if (!parsedId.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const client = await requireClientManage(parsedId.data);
  if (!client) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");

  const file = formData.get("file");
  if (!(file instanceof File) || file.size === 0) return fail(s.noFile, "VALIDATION");
  if (file.size > MAX_FILE_BYTES) return fail(s.fileTooLarge, "VALIDATION");
  if (!/\.csv$/i.test(file.name) && !file.type.includes("csv") && file.type !== "text/plain") {
    return fail(s.notCsv, "VALIDATION");
  }

  let text: string;
  try {
    text = await file.text();
  } catch {
    return fail(t.errors.UNKNOWN);
  }

  const { rows, warnings } = parseAdsCsv(text);
  if (rows.length === 0) {
    return fail(warnings[0] ?? s.csvNoRows, "VALIDATION");
  }

  try {
    const summary = await persistCsvRows(client.id, rows, warnings);
    await db.syncJob.create({
      data: {
        clientId: client.id,
        kind: "ADS_CSV_IMPORT",
        status: "SUCCESS",
        finishedAt: new Date(),
        message: s.importSummary(summary),
      },
    });
    revalidatePath(`/clients/${client.slug}/ads`);
    revalidatePath(`/clients/${client.slug}`);
    return ok(summary);
  } catch (e) {
    await db.syncJob
      .create({
        data: {
          clientId: client.id,
          kind: "ADS_CSV_IMPORT",
          status: "FAILED",
          finishedAt: new Date(),
          error: e instanceof Error ? e.message : String(e),
        },
      })
      .catch(() => undefined);
    return fail(t.errors.UNKNOWN);
  }
}

/**
 * Sinkronisasi mock: menyentuh lastSyncedAt + mencatat SyncJob ADS_INSIGHTS.
 * (Sinkronisasi nyata membutuhkan Connection Meta — lihat lib/providers/meta-marketing.)
 */
export async function syncAds(clientId: string): Promise<ActionResult<{ lastSyncedAt: string }>> {
  const parsedId = z.string().min(1).safeParse(clientId);
  if (!parsedId.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const client = await requireClientManage(parsedId.data);
  if (!client) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");

  const accounts = await db.adAccount.findMany({ where: { clientId: client.id }, select: { id: true, connectionId: true } });
  if (accounts.length === 0) return fail(s.syncNoAccount, "VALIDATION");

  const now = new Date();
  // Real Marketing API pull when at least one account has a Meta connection.
  if (accounts.some((a) => a.connectionId)) {
    const r = await syncAdsForClient(client.id, { days: 90 });
    revalidatePath(`/clients/${client.slug}/ads`);
    revalidatePath(`/clients/${client.slug}`);
    const failed = r.results.find((x) => x.error);
    if (r.real === 0 && failed) return fail(failed.error!);
    if (r.real > 0) return ok({ lastSyncedAt: now.toISOString() });
    // connection rows exist but credentials are not configured → fall through to demo touch
  }
  try {
    const campaignCount = await db.adCampaign.count({ where: { adAccountId: { in: accounts.map((a) => a.id) } } });
    await db.adAccount.updateMany({ where: { id: { in: accounts.map((a) => a.id) } }, data: { lastSyncedAt: now } });
    await db.syncJob.create({
      data: {
        clientId: client.id,
        kind: "ADS_INSIGHTS",
        status: "SUCCESS",
        finishedAt: now,
        message: s.syncDemoMessage(campaignCount),
      },
    });
    revalidatePath(`/clients/${client.slug}/ads`);
    revalidatePath(`/clients/${client.slug}`);
    return ok({ lastSyncedAt: now.toISOString() });
  } catch (e) {
    await db.syncJob
      .create({
        data: {
          clientId: client.id,
          kind: "ADS_INSIGHTS",
          status: "FAILED",
          finishedAt: new Date(),
          error: e instanceof Error ? e.message : String(e),
        },
      })
      .catch(() => undefined);
    return fail(t.errors.UNKNOWN);
  }
}
