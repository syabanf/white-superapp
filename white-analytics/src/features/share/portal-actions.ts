"use server";

import { revalidatePath } from "next/cache";
import { z } from "zod";
import { fail, type ActionResult } from "@/lib/action-result";
import { publicFetch } from "@/lib/erp/public";
import { runGateway, zodFieldErrors } from "@/features/erp/shared/action";
import { getShareContext } from "./access";
import { getPortalToken } from "./portal";
import { pt } from "./strings";

const MAX_PROOF_BYTES = 15 * 1024 * 1024;

/** Unlocked share page plus the services' token; null when either is missing. */
async function portalAccess(slug: string): Promise<{ token: string } | null> {
  const ctx = await getShareContext(slug);
  if (ctx.status !== "ok") return null;
  const token = await getPortalToken(ctx.client.id);
  return token ? { token } : null;
}

const proofSchema = z.object({
  invoiceId: z.string().min(1),
  amount: z.string().regex(/^\d+(\.\d{1,2})?$/, pt.amountInvalid),
  paidOn: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, pt.dateInvalid),
  payerName: z.string().trim().min(1, pt.payerRequired).max(120),
  note: z.string().trim().max(500).default(""),
});

export async function submitPaymentProof(slug: string, _prev: ActionResult | null, formData: FormData): Promise<ActionResult> {
  const access = await portalAccess(slug);
  if (!access) return fail(pt.locked, "UNAUTHORIZED");
  const parsed = proofSchema.safeParse({
    invoiceId: formData.get("invoiceId"),
    amount: String(formData.get("amount") ?? "").replace(/\./g, "").replace(",", "."),
    paidOn: formData.get("paidOn"),
    payerName: formData.get("payerName"),
    note: formData.get("note") ?? "",
  });
  if (!parsed.success) return fail(pt.checkForm, "VALIDATION", zodFieldErrors(parsed.error));
  const file = formData.get("file");
  if (!(file instanceof File) || file.size === 0) return fail(pt.fileRequired, "VALIDATION", { file: [pt.fileRequired] });
  if (file.size > MAX_PROOF_BYTES) return fail(pt.fileTooBig, "VALIDATION", { file: [pt.fileTooBig] });
  const data = Buffer.from(await file.arrayBuffer()).toString("base64");
  const res = await runGateway(() =>
    publicFetch(`/v1/public/sales/portal/${encodeURIComponent(slug)}/invoices/${encodeURIComponent(parsed.data.invoiceId)}/proofs`, {
      method: "POST",
      body: JSON.stringify({ ...parsed.data, unlockToken: access.token, fileName: file.name, data }),
    }),
  );
  if (!res.ok) return res;
  revalidatePath(`/share/${slug}`);
  return { ok: true, data: undefined };
}

const decideSchema = z.object({
  itemId: z.string().min(1),
  approve: z.boolean(),
  signerName: z.string().trim().min(1, pt.signerRequired).max(120),
  note: z.string().trim().max(1000).default(""),
});

export async function decideContent(slug: string, input: z.input<typeof decideSchema>): Promise<ActionResult> {
  const access = await portalAccess(slug);
  if (!access) return fail(pt.locked, "UNAUTHORIZED");
  const parsed = decideSchema.safeParse(input);
  if (!parsed.success) return fail(pt.checkForm, "VALIDATION", zodFieldErrors(parsed.error));
  if (!parsed.data.approve && !parsed.data.note) return fail(pt.noteRequired, "VALIDATION", { note: [pt.noteRequired] });
  const res = await runGateway(() =>
    publicFetch(`/v1/public/content/portal/${encodeURIComponent(slug)}/items/${encodeURIComponent(parsed.data.itemId)}:decide`, {
      method: "POST",
      body: JSON.stringify({ ...parsed.data, unlockToken: access.token }),
    }),
  );
  if (!res.ok) return res;
  revalidatePath(`/share/${slug}`);
  return { ok: true, data: undefined };
}
