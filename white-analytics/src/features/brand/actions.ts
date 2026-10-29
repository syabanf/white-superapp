"use server";

import { revalidatePath } from "next/cache";
import { z } from "zod";
import { db } from "@/lib/db";
import { fail, isProviderError, ok, type ActionResult } from "@/lib/action-result";
import { getSessionUser, requireClientAccess } from "@/lib/rbac";
import { withClientMode } from "@/lib/client-mode";
import { aiStudio, isAiStudioConfigured, type CaptionVariant } from "@/lib/providers/ai-studio";
import { getBrandContext } from "@/features/brand/context";
import { BRAND_LIMITS, normalizeHex } from "@/features/brand/lib";
import { b } from "@/features/brand/strings";
import { t } from "@/i18n/id";

const hex = z
  .string()
  .transform((v, ctx) => {
    const h = normalizeHex(v);
    if (!h) {
      ctx.addIssue({ code: "custom", message: b.invalidHex });
      return z.NEVER;
    }
    return h;
  });

const text = (max: number) => z.string().trim().max(max).default("");
const list = (items: number, chars: number) =>
  z.array(z.string().trim().min(1).max(chars)).max(items).default([]);
const tags = list(BRAND_LIMITS.tags, BRAND_LIMITS.tagChars).transform((l) =>
  Array.from(new Set(l.map((x) => x.replace(/^#+/, "")).filter(Boolean))),
);

const brandKitSchema = z.object({
  primaryColor: hex,
  secondaryColor: hex,
  accentColors: z.array(hex).max(BRAND_LIMITS.accentColors).default([]),
  headingFont: text(BRAND_LIMITS.font),
  bodyFont: text(BRAND_LIMITS.font),
  logoUrl: z
    .string()
    .trim()
    .max(2000)
    .refine((v) => v === "" || /^https:\/\/\S+$/.test(v) || /^\/api\/media\/\S+$/.test(v), {
      message: "URL logo harus diawali https://",
    })
    .transform((v) => v || null)
    .default(""),
  toneOfVoice: text(BRAND_LIMITS.text),
  audience: text(BRAND_LIMITS.text),
  doList: list(BRAND_LIMITS.listItems, BRAND_LIMITS.listItemChars),
  dontList: list(BRAND_LIMITS.listItems, BRAND_LIMITS.listItemChars),
  keywords: list(BRAND_LIMITS.tags, BRAND_LIMITS.tagChars),
  hashtags: tags,
  sampleCaptions: list(BRAND_LIMITS.sampleCaptions, BRAND_LIMITS.sampleCaptionChars),
  visualStyle: text(BRAND_LIMITS.text),
  notes: text(BRAND_LIMITS.notes),
});

export type BrandKitInput = z.input<typeof brandKitSchema>;

/** Create or update the client's Brand Kit (MANAGER/ADMIN). */
export async function saveBrandKit(slug: string, input: BrandKitInput): Promise<ActionResult<{ updatedAt: string }>> {
  const { client, canManage } = await requireClientAccess(slug);
  if (!canManage) return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsed = brandKitSchema.safeParse(input);
  if (!parsed.success) {
    const flat = z.flattenError(parsed.error);
    const first = Object.values(flat.fieldErrors).flat()[0] ?? t.errors.VALIDATION;
    return fail(first, "VALIDATION", flat.fieldErrors as Record<string, string[]>);
  }
  const user = await getSessionUser();
  const data = { ...parsed.data, updatedById: user?.id ?? null };
  const kit = await db.brandKit.upsert({
    where: { clientId: client.id },
    create: { clientId: client.id, ...data },
    update: data,
    select: { updatedAt: true },
  });
  revalidatePath(`/clients/${slug}/content/brand`);
  revalidatePath(`/clients/${slug}/content/studio`);
  return ok({ updatedAt: kit.updatedAt.toISOString() });
}

const rewriteSchema = z.object({
  slug: z.string().min(1),
  hook: z.string().max(500).default(""),
  caption: z.string().max(5000).default(""),
  hashtags: z.array(z.string().max(100)).max(60).default([]),
  instruction: z.string().trim().max(500).default(""),
  platform: z.enum(["INSTAGRAM", "FACEBOOK", "TIKTOK"]).default("INSTAGRAM"),
});

export type RewriteCaptionInput = z.input<typeof rewriteSchema>;

/**
 * Three on-brand caption variants. Writes nothing: the editor fills its own
 * fields and the user saves through the normal flow.
 */
export async function rewriteCaptionWithAi(
  input: RewriteCaptionInput,
): Promise<ActionResult<{ variants: CaptionVariant[]; demo: boolean }>> {
  const parsed = rewriteSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION");
  const d = parsed.data;
  if (!d.caption.trim() && !d.hook.trim()) return fail(b.emptyInput, "VALIDATION");
  const { client } = await requireClientAccess(d.slug);

  return withClientMode({ id: client.id }, async () => {
    try {
      const brand = await getBrandContext(client.id);
      const variants = await aiStudio.rewriteCaption({
        brand,
        hook: d.hook,
        caption: d.caption,
        hashtags: d.hashtags.map((h) => h.replace(/^#+/, "")).filter(Boolean),
        instruction: d.instruction,
        platform: d.platform,
      });
      const clean = variants
        .map((v) => ({ ...v, hashtags: Array.from(new Set(v.hashtags.map((h) => h.replace(/^#+/, "").trim()).filter(Boolean))) }))
        .slice(0, 3);
      if (clean.length === 0) return fail<{ variants: CaptionVariant[]; demo: boolean }>(b.noVariants);
      return ok({ variants: clean, demo: !isAiStudioConfigured() });
    } catch (e) {
      if (isProviderError(e)) return fail<{ variants: CaptionVariant[]; demo: boolean }>(e.message || t.errors[e.code], e.code);
      console.error("rewriteCaptionWithAi failed", e);
      return fail<{ variants: CaptionVariant[]; demo: boolean }>(t.errors.UNKNOWN);
    }
  });
}
