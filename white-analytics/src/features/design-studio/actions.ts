"use server";

import { revalidatePath } from "next/cache";
import { z } from "zod";
import { db } from "@/lib/db";
import { fail, isProviderError, ok, type ActionResult } from "@/lib/action-result";
import { getSessionUser, requireClientAccess } from "@/lib/rbac";
import { withClientMode } from "@/lib/client-mode";
import { aiStudio, isAiStudioConfigured, type DesignLayout } from "@/lib/providers/ai-studio";
import { getBrandContext } from "@/features/brand/context";
import { uploadMedia } from "@/features/publishing/actions-media";
import { DESIGN_FORMATS, designFilename, type DraftRow } from "@/features/design-studio/lib";
import { toDraftRow } from "@/features/design-studio/queries";
import { ds } from "@/features/design-studio/strings";
import { t } from "@/i18n/id";

const EXT: Record<string, string> = { "image/png": "png", "image/jpeg": "jpg", "image/webp": "webp", "image/gif": "gif" };

function revalidateStudio(slug: string) {
  revalidatePath(`/clients/${slug}/content/studio`);
}

/** `data:<mime>;base64,<payload>` → bytes; null for anything else. */
function decodeDataUrl(url: string): { mimeType: string; bytes: Buffer } | null {
  const m = url.match(/^data:([a-z0-9.+/-]+);base64,(.+)$/i);
  if (!m) return null;
  return { mimeType: m[1]!.toLowerCase(), bytes: Buffer.from(m[2]!, "base64") };
}

/**
 * Store image bytes through the publishing library's own upload action, so
 * validation (size, sniffed type), RBAC and revalidation stay in one place.
 */
async function storeInLibrary(
  clientId: string,
  file: File,
): Promise<ActionResult<{ id: string; url: string }>> {
  const fd = new FormData();
  fd.set("clientId", clientId);
  fd.append("files", file);
  const res = await uploadMedia(fd);
  if (!res.ok) return res;
  const asset = res.data.assets[0];
  if (!asset) return fail(t.errors.UNKNOWN);
  return ok({ id: asset.id, url: asset.url });
}

const generateSchema = z.object({
  title: z.string().trim().min(1, ds.titleRequired).max(120),
  brief: z.string().trim().max(2000).default(""),
  format: z.enum(DESIGN_FORMATS),
});

export type GenerateDesignInput = z.input<typeof generateSchema>;

/**
 * Layout (designLayout) → image (generateImage) → DesignDraft. Mock output is an
 * SVG data URL kept on the row (a few KB); real images go to the media library
 * right away so the row holds a URL instead of megabytes of base64.
 */
export async function generateDesign(
  slug: string,
  input: GenerateDesignInput,
): Promise<ActionResult<{ draft: DraftRow; demo: boolean }>> {
  const { client, canManage } = await requireClientAccess(slug);
  if (!canManage) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = generateSchema.safeParse(input);
  if (!parsed.success) {
    const flat = z.flattenError(parsed.error);
    return fail(Object.values(flat.fieldErrors).flat()[0] ?? t.errors.VALIDATION, "VALIDATION", flat.fieldErrors as Record<string, string[]>);
  }
  const { title, brief, format } = parsed.data;
  const user = await getSessionUser();

  return withClientMode({ id: client.id }, async () => {
    let layout: DesignLayout;
    let image: { dataUrl: string; mimeType: string; model: string };
    try {
      const brand = await getBrandContext(client.id);
      layout = await aiStudio.designLayout({ brand, title, brief, format });
      image = await aiStudio.generateImage({ brand, layout, format });
    } catch (e) {
      if (isProviderError(e)) return fail<{ draft: DraftRow; demo: boolean }>(e.message || t.errors[e.code], e.code);
      console.error("generateDesign failed", e);
      return fail<{ draft: DraftRow; demo: boolean }>(t.errors.UNKNOWN);
    }

    let imageUrl: string | null = image.dataUrl;
    let mediaAssetId: string | null = null;
    const decoded = image.dataUrl.startsWith("data:image/svg") ? null : decodeDataUrl(image.dataUrl);
    if (decoded && EXT[decoded.mimeType]) {
      const file = new File([new Uint8Array(decoded.bytes)], designFilename(title, format, EXT[decoded.mimeType]!), { type: decoded.mimeType });
      const stored = await storeInLibrary(client.id, file);
      if (stored.ok) {
        imageUrl = stored.data.url;
        mediaAssetId = stored.data.id;
      } else {
        // Keep the generated image rather than lose a paid generation; the row is just bigger.
        console.error("generateDesign: storing image failed", stored.error);
      }
    } else if (/^https:\/\//.test(image.dataUrl)) {
      imageUrl = image.dataUrl;
    } else if (!image.dataUrl.startsWith("data:image/svg")) {
      imageUrl = null;
    }

    const row = await db.designDraft.create({
      data: {
        clientId: client.id,
        title,
        brief,
        format,
        layout,
        imagePrompt: layout.imagePrompt,
        imageUrl,
        mediaAssetId,
        model: image.model || aiStudio.model,
        createdById: user?.id ?? null,
      },
    });
    revalidateStudio(slug);
    return ok({ draft: toDraftRow(row), demo: !isAiStudioConfigured() });
  });
}

/**
 * "Simpan ke Pustaka Media" for a draft whose image is still inline (the mock
 * SVG). The browser rasterizes it to PNG and posts `draftId` + `file`.
 */
export async function saveDraftToLibrary(formData: FormData): Promise<ActionResult<{ draft: DraftRow }>> {
  const draftId = formData.get("draftId");
  const file = formData.get("file");
  if (typeof draftId !== "string" || !draftId) return fail(t.errors.VALIDATION, "VALIDATION");
  const draft = await db.designDraft.findUnique({ where: { id: draftId }, include: { client: { select: { slug: true } } } });
  if (!draft) return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  const { canManage } = await requireClientAccess(draft.client.slug);
  if (!canManage) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  if (draft.mediaAssetId) return ok({ draft: toDraftRow(draft) });
  if (!file || typeof file !== "object" || !("arrayBuffer" in file)) return fail(t.errors.VALIDATION, "VALIDATION");

  const stored = await storeInLibrary(draft.clientId, file as File);
  if (!stored.ok) return stored;
  const updated = await db.designDraft.update({
    where: { id: draft.id },
    data: { imageUrl: stored.data.url, mediaAssetId: stored.data.id },
  });
  revalidateStudio(draft.client.slug);
  return ok({ draft: toDraftRow(updated) });
}

/** Remove a draft from the history. A copy already in the media library stays there. */
export async function deleteDesignDraft(draftId: string): Promise<ActionResult<{ id: string }>> {
  if (typeof draftId !== "string" || !draftId) return fail(t.errors.VALIDATION, "VALIDATION");
  const draft = await db.designDraft.findUnique({ where: { id: draftId }, include: { client: { select: { slug: true } } } });
  if (!draft) return fail(t.errors.NOT_FOUND, "NOT_FOUND");
  const { canManage } = await requireClientAccess(draft.client.slug);
  if (!canManage) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  await db.designDraft.delete({ where: { id: draft.id } });
  revalidateStudio(draft.client.slug);
  return ok({ id: draft.id });
}
