"use server";

import { revalidatePath } from "next/cache";
import { ok, type ActionResult } from "@/lib/action-result";
import { gatewayFetch } from "@/lib/erp/gateway";
import { requireAdmin } from "@/lib/rbac";
import { runGateway } from "@/features/erp/shared/action";
import type { CompanyProfile } from "./data";

const path = "/admin/company";

/** bankLine is derived by Finance; the service ignores it on save. */
export async function saveCompanyProfileAction(input: CompanyProfile): Promise<ActionResult<CompanyProfile>> {
  await requireAdmin();
  const res = await runGateway(() => gatewayFetch<CompanyProfile>("/v1/finance/company-profile", { method: "PUT", body: JSON.stringify(input) }));
  if (res.ok) revalidatePath(path);
  return res;
}

/** Saves the From address and, when typed, a new key; then asks Resend whether both work. */
export async function saveMailAction(input: { from: string; apiKey: string }): Promise<ActionResult<{ ok: boolean; message: string }>> {
  await requireAdmin();
  const body: Record<string, unknown> = { publicId: input.from.trim(), verify: true };
  if (input.apiKey.trim()) body.secret = input.apiKey.trim();
  const res = await runGateway(() =>
    gatewayFetch<{ verified?: { ok?: boolean; message?: string } }>("/v1/integrations/RESEND", { method: "PUT", body: JSON.stringify(body) }),
  );
  if (!res.ok) return res;
  revalidatePath(path);
  return ok({ ok: res.data.verified?.ok ?? false, message: res.data.verified?.message ?? "" });
}

export async function clearMailAction() {
  await requireAdmin();
  const res = await runGateway(() => gatewayFetch("/v1/integrations/RESEND", { method: "DELETE" }));
  if (res.ok) revalidatePath(path);
  return res;
}
