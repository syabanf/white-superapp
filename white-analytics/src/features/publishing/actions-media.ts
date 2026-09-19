"use server";

import { z } from "zod";
import { db } from "@/lib/db";
import { fail, ok, type ActionResult } from "@/lib/action-result";
import { deleteObject, keyFromUrl, MAX_UPLOAD_BYTES, putObject, sniffContentType } from "@/lib/storage";
import { resolvePublishingAccess, revalidatePublishing } from "@/features/publishing/access";
import type { MediaRow } from "@/features/publishing/queries";
import { p } from "@/features/publishing/strings";
import { t } from "@/i18n/id";

function toRow(a: {
  id: string;
  kind: "IMAGE" | "VIDEO";
  url: string;
  thumbnailUrl: string | null;
  filename: string;
  mimeType: string;
  sizeBytes: number;
  width: number | null;
  height: number | null;
  altText: string | null;
  tags: string[];
  createdAt: Date;
}): MediaRow {
  return { ...a, createdAt: a.createdAt.toISOString(), usageCount: 0 };
}

/** Multipart upload: `clientId` + one or more `files`. Stored through `putObject`. */
export async function uploadMedia(formData: FormData): Promise<ActionResult<{ assets: MediaRow[] }>> {
  const clientId = formData.get("clientId");
  if (typeof clientId !== "string" || !clientId) return fail(t.errors.VALIDATION, "VALIDATION");
  const access = await resolvePublishingAccess(clientId);
  if (!access || access.role === "VIEWER") return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");

  const files = formData.getAll("files").filter((f): f is File => typeof f === "object" && f != null && "arrayBuffer" in f);
  if (files.length === 0) return fail(t.errors.VALIDATION, "VALIDATION");
  // Validate every file before storing any: size first, then the real type from its bytes.
  const checked: { file: File; buf: Buffer; type: string }[] = [];
  for (const f of files) {
    if (f.size > MAX_UPLOAD_BYTES) return fail(`${p.fileTooLarge} (${f.name})`, "VALIDATION");
    const buf = Buffer.from(await f.arrayBuffer());
    const type = sniffContentType(buf);
    if (!type) return fail(`${p.fileTypeNotAllowed} (${f.name})`, "VALIDATION");
    checked.push({ file: f, buf, type });
  }

  const assets: MediaRow[] = [];
  for (const { file: f, buf, type } of checked) {
    const stored = await putObject(buf, { clientId, contentType: type });
    const asset = await db.mediaAsset.create({
      data: {
        clientId,
        kind: type.startsWith("video/") ? "VIDEO" : "IMAGE",
        url: stored.url,
        filename: f.name.slice(0, 200),
        mimeType: type,
        sizeBytes: stored.sizeBytes,
        uploadedById: access.userId,
      },
    });
    assets.push(toRow(asset));
  }
  revalidatePublishing(access.slug);
  return ok({ assets });
}

const externalSchema = z.object({
  clientId: z.string().min(1),
  url: z.url({ protocol: /^https$/ }).max(2000),
  kind: z.enum(["IMAGE", "VIDEO"]),
  altText: z.string().max(500).optional(),
  tags: z.array(z.string().trim().min(1).max(40)).max(20).optional(),
});

export async function addExternalMedia(input: z.input<typeof externalSchema>): Promise<ActionResult<{ asset: MediaRow }>> {
  const parsed = externalSchema.safeParse(input);
  if (!parsed.success) return fail(p.invalidHttpsUrl, "VALIDATION");
  const d = parsed.data;
  const access = await resolvePublishingAccess(d.clientId);
  if (!access || access.role === "VIEWER") return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const path = new URL(d.url).pathname;
  const filename = decodeURIComponent(path.split("/").filter(Boolean).pop() ?? "media").slice(0, 200) || "media";
  const ext = filename.split(".").pop()?.toLowerCase() ?? "";
  const mimeType = d.kind === "VIDEO" ? (ext === "mov" ? "video/quicktime" : "video/mp4") : ext === "png" ? "image/png" : ext === "webp" ? "image/webp" : ext === "gif" ? "image/gif" : "image/jpeg";
  const asset = await db.mediaAsset.create({
    data: {
      clientId: d.clientId,
      kind: d.kind,
      url: d.url,
      filename,
      mimeType,
      altText: d.altText?.trim() || null,
      tags: d.tags ?? [],
      uploadedById: access.userId,
    },
  });
  revalidatePublishing(access.slug);
  return ok({ asset: toRow(asset) });
}

const updateSchema = z.object({
  id: z.string().min(1),
  altText: z.string().max(500).nullable().optional(),
  tags: z.array(z.string().trim().min(1).max(40)).max(20).optional(),
});

export async function updateMedia(input: z.input<typeof updateSchema>): Promise<ActionResult<null>> {
  const parsed = updateSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const asset = await db.mediaAsset.findUnique({ where: { id: parsed.data.id }, select: { clientId: true } });
  if (!asset) return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  const access = await resolvePublishingAccess(asset.clientId);
  if (!access || access.role === "VIEWER") return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  await db.mediaAsset.update({
    where: { id: parsed.data.id },
    data: {
      ...(parsed.data.altText !== undefined ? { altText: parsed.data.altText?.trim() || null } : {}),
      ...(parsed.data.tags ? { tags: Array.from(new Set(parsed.data.tags)) } : {}),
    },
  });
  revalidatePublishing(access.slug);
  return ok(null);
}

export async function deleteMedia(id: string): Promise<ActionResult<null>> {
  if (!id || typeof id !== "string") return fail(t.errors.VALIDATION, "VALIDATION");
  const asset = await db.mediaAsset.findUnique({ where: { id }, include: { _count: { select: { usages: true } } } });
  if (!asset) return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  const access = await resolvePublishingAccess(asset.clientId);
  if (!access || access.role === "VIEWER") return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  if (asset._count.usages > 0) return fail(p.mediaInUse, "VALIDATION");
  await db.mediaAsset.delete({ where: { id } });
  const key = keyFromUrl(asset.url);
  if (key) await deleteObject(key);
  revalidatePublishing(access.slug);
  return ok(null);
}
