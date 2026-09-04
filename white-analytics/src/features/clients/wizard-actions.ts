"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { z } from "zod";
import { Prisma } from "@/generated/prisma/client";
import { fail, type ActionResult } from "@/lib/action-result";
import { db } from "@/lib/db";
import { getSessionUser } from "@/lib/rbac";
import { t } from "@/i18n/id";
import { tc } from "@/features/clients/strings";
import { CLIENT_CURRENCIES, CLIENT_TIMEZONES } from "@/features/clients/constants";
import { projectTypesSchema, wizardHref, type ProjectType } from "@/features/clients/wizard";

/**
 * Wizard-specific actions. They reuse the same validation shape as the regular
 * client form but advance through the wizard instead of jumping to the dashboard.
 */

function zodErrors(err: z.ZodError): Record<string, string[]> {
  const out: Record<string, string[]> = {};
  for (const issue of err.issues) {
    const key = String(issue.path[0] ?? "form");
    (out[key] ??= []).push(issue.message);
  }
  return out;
}

const profileSchema = z.object({
  name: z.string().trim().min(1, tc.form.nameRequired).max(80),
  slug: z
    .string()
    .trim()
    .toLowerCase()
    .min(2, tc.form.slugRequired)
    .max(60)
    .regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/, tc.form.slugInvalid),
  description: z.string().trim().max(500).default(""),
  industry: z.string().trim().max(60).default(""),
  websiteUrl: z.union([z.literal(""), z.string().trim().url(tc.form.websiteInvalid)]).default(""),
  currency: z.enum(CLIENT_CURRENCIES),
  timezone: z.enum(CLIENT_TIMEZONES),
});

export type WizardProfileInput = z.input<typeof profileSchema>;

function emptyToNull(v: string | undefined): string | null {
  const s = (v ?? "").trim();
  return s.length === 0 ? null : s;
}

/**
 * Step 2 → creates the client for the chosen project types and moves to sources.
 * When it is an SEO project and a website was given, the Search Console property
 * is registered straight away so the operator does not type the URL twice.
 */
export async function wizardCreateClientAction(
  input: WizardProfileInput,
  types: ProjectType[],
): Promise<ActionResult<never>> {
  const user = await getSessionUser();
  if (!user || user.role !== "ADMIN") return fail(t.errors.UNAUTHORIZED, "UNAUTHORIZED");
  const parsedTypes = projectTypesSchema.safeParse(types);
  if (!parsedTypes.success) return fail(tc.wizard.pickOneChannel, "VALIDATION");
  const parsed = profileSchema.safeParse(input);
  if (!parsed.success) return fail(t.errors.VALIDATION, "VALIDATION", zodErrors(parsed.error));
  const d = parsed.data;

  const exists = await db.client.findUnique({ where: { slug: d.slug }, select: { id: true } });
  if (exists) return fail(t.clients.slugTaken, "VALIDATION", { slug: [t.clients.slugTaken] });

  try {
    const created = await db.client.create({
      data: {
        name: d.name,
        slug: d.slug,
        description: d.description,
        industry: emptyToNull(d.industry),
        websiteUrl: emptyToNull(d.websiteUrl),
        currency: d.currency,
        timezone: d.timezone,
        // Nothing is shared until the operator says so in step 4.
        shareEnabled: false,
        shareModules: parsedTypes.data,
        createdById: user.id,
      },
      select: { id: true },
    });

    const website = emptyToNull(d.websiteUrl);
    if (parsedTypes.data.includes("SEO") && website) {
      await db.seoProperty.create({ data: { clientId: created.id, siteUrl: website } });
    }
  } catch (e) {
    if (e instanceof Prisma.PrismaClientKnownRequestError && e.code === "P2002") {
      return fail(t.clients.slugTaken, "VALIDATION", { slug: [t.clients.slugTaken] });
    }
    return fail(t.errors.UNKNOWN);
  }

  revalidatePath("/");
  revalidatePath("/clients");
  redirect(wizardHref("sumber", { slug: d.slug }));
}

