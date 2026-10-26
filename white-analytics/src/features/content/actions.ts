"use server";

import { revalidatePath } from "next/cache";
import { z } from "zod";
import type { ActionResult } from "@/lib/action-result";
import { gatewayFetch, GatewayError } from "@/lib/erp/gateway";
import { requireClientAccess } from "@/lib/rbac";
import { isMonthKey } from "./lib";
import type {
  AssetKind,
  ContentFormat,
  ContentReport,
  DesignStatus,
  DesignTaskInput,
  ItemDetail,
  ItemInput,
  ItemStatus,
  PlanBrief,
  PlanDetail,
  ReviewDecision,
} from "./types";

/**
 * Content mutations. Each action resolves the client from the slug (RBAC on
 * the marketing side), calls the content service with the user's token (the
 * service enforces membership roles and approval rules) and refreshes every
 * /content route of the client.
 */
async function run<T>(slug: string, fn: (clientPath: string) => Promise<T>): Promise<ActionResult<T>> {
  const { client } = await requireClientAccess(slug);
  try {
    const data = await fn(`/v1/content/clients/${encodeURIComponent(client.id)}`);
    revalidatePath(`/clients/${slug}/content`, "layout");
    return { ok: true, data };
  } catch (e) {
    if (e instanceof GatewayError) return { ok: false, error: e.message, fieldErrors: e.fieldErrors };
    throw e;
  }
}

const post = <T>(path: string, body?: unknown) =>
  gatewayFetch<T>(path, { method: "POST", body: body === undefined ? undefined : JSON.stringify(body) });
const patch = <T>(path: string, body: unknown) => gatewayFetch<T>(path, { method: "PATCH", body: JSON.stringify(body) });
const del = (path: string) => gatewayFetch<unknown>(path, { method: "DELETE" });
const id = (v: string) => encodeURIComponent(v);

const invalid = (error: string): ActionResult<never> => ({ ok: false, error, code: "VALIDATION" });

// ── plans ────────────────────────────────────────────────────

export async function createPlan(input: { slug: string; month: string; title: string; brief?: PlanBrief }) {
  if (!isMonthKey(input.month)) return invalid("Pilih bulan yang valid.");
  return run(input.slug, (c) =>
    post<PlanDetail>(`${c}/plans`, { month: input.month, title: input.title.trim(), brief: input.brief }).then((d) => ({
      id: d.plan.id,
    })),
  );
}

export async function updatePlan(input: { slug: string; planId: string; title?: string; brief: PlanBrief }) {
  return run(input.slug, (c) =>
    patch<PlanDetail>(`${c}/plans/${id(input.planId)}`, { title: input.title, brief: input.brief }).then(() => null),
  );
}

export async function deletePlan(input: { slug: string; planId: string }) {
  return run(input.slug, (c) => del(`${c}/plans/${id(input.planId)}`).then(() => null));
}

export async function generatePlanDraft(input: { slug: string; planId: string; replace: boolean }) {
  return run(input.slug, (c) =>
    post<PlanDetail>(`${c}/plans/${id(input.planId)}:generate`, { replace: input.replace }).then((d) => ({
      count: d.items?.length ?? 0,
    })),
  );
}

export async function submitPlan(input: { slug: string; planId: string }) {
  return run(input.slug, (c) => post(`${c}/plans/${id(input.planId)}:submit`).then(() => null));
}

export async function approvePlan(input: { slug: string; planId: string; note?: string }) {
  return run(input.slug, (c) => post(`${c}/plans/${id(input.planId)}:approve`, { note: input.note ?? "" }).then(() => null));
}

export async function requestPlanChanges(input: { slug: string; planId: string; note: string }) {
  if (!input.note.trim()) return invalid("Catatan revisi wajib diisi.");
  return run(input.slug, (c) =>
    post(`${c}/plans/${id(input.planId)}:requestChanges`, { note: input.note.trim() }).then(() => null),
  );
}

export async function addPlanComment(input: { slug: string; planId: string; body: string }) {
  if (!input.body.trim()) return invalid("Komentar kosong.");
  return run(input.slug, (c) => post(`${c}/plans/${id(input.planId)}/comments`, { body: input.body.trim() }).then(() => null));
}

// ── items ────────────────────────────────────────────────────

const itemSchema = z.object({
  plannedDate: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, "Pilih tanggal rencana."),
  publishAt: z.string().optional(),
  platform: z.enum(["INSTAGRAM", "FACEBOOK", "TIKTOK"]),
  format: z.enum(["FEED", "CAROUSEL", "REEL", "STORY"]),
  pillar: z.string(),
  title: z.string().trim().min(1, "Judul wajib diisi."),
  visualBrief: z.string(),
  copywriterId: z.string(),
  designerId: z.string(),
  copyDue: z.string(),
  designDue: z.string(),
  reviewDue: z.string(),
});

function checkItem(item: ItemInput): ActionResult<never> | null {
  const parsed = itemSchema.safeParse(item);
  if (parsed.success) return null;
  const fieldErrors: Record<string, string[]> = {};
  for (const issue of parsed.error.issues) (fieldErrors[String(issue.path[0])] ??= []).push(issue.message);
  return { ok: false, error: parsed.error.issues[0]!.message, code: "VALIDATION", fieldErrors };
}

export async function createItem(input: {
  slug: string;
  planId: string;
  item: ItemInput;
  hook: string;
  caption: string;
  hashtags: string[];
}) {
  const bad = checkItem(input.item);
  if (bad) return bad;
  return run(input.slug, (c) =>
    post<ItemDetail>(`${c}/plans/${id(input.planId)}/items`, {
      item: input.item,
      hook: input.hook,
      caption: input.caption,
      hashtags: input.hashtags,
    }).then((d) => ({ id: d.item.id })),
  );
}

export async function updateItem(input: { slug: string; itemId: string; item: ItemInput }) {
  const bad = checkItem(input.item);
  if (bad) return bad;
  return run(input.slug, (c) => patch(`${c}/items/${id(input.itemId)}`, { item: input.item }).then(() => null));
}

export async function deleteItem(input: { slug: string; itemId: string }) {
  return run(input.slug, (c) => del(`${c}/items/${id(input.itemId)}`).then(() => null));
}

export async function moveItem(input: { slug: string; itemId: string; to: ItemStatus; note?: string }) {
  if (input.to === "REVISION" && !input.note?.trim()) return invalid("Catatan revisi wajib diisi.");
  return run(input.slug, (c) =>
    post(`${c}/items/${id(input.itemId)}:move`, { to: input.to, note: input.note?.trim() ?? "" }).then(() => null),
  );
}

export async function reviseCaption(input: {
  slug: string;
  itemId: string;
  hook: string;
  caption: string;
  hashtags: string[];
  note: string;
}) {
  const { slug, itemId, ...body } = input;
  return run(slug, (c) => post(`${c}/items/${id(itemId)}/revisions`, body).then(() => null));
}

export async function addItemComment(input: { slug: string; itemId: string; body: string }) {
  if (!input.body.trim()) return invalid("Komentar kosong.");
  return run(input.slug, (c) => post(`${c}/items/${id(input.itemId)}/comments`, { body: input.body.trim() }).then(() => null));
}

/** Sets the format first when it changed (UpdateItem takes the full planning input), then schedules. */
export async function scheduleItem(input: { slug: string; itemId: string; publishAt: string; format: ContentFormat; item: ItemInput }) {
  return run(input.slug, async (c) => {
    if (input.item.format !== input.format) {
      await patch(`${c}/items/${id(input.itemId)}`, { item: { ...input.item, format: input.format } });
    }
    const d = await post<ItemDetail>(`${c}/items/${id(input.itemId)}:schedule`, { publishAt: input.publishAt });
    return { status: d.item.status, publishError: d.item.publishError ?? null };
  });
}

export async function syncItem(input: { slug: string; itemId: string }) {
  return run(input.slug, (c) =>
    post<ItemDetail>(`${c}/items/${id(input.itemId)}:sync`).then((d) => ({ status: d.item.status })),
  );
}

// ── design ───────────────────────────────────────────────────

export async function createDesignTask(input: { slug: string; itemId: string; task: DesignTaskInput }) {
  if (!input.task.title.trim()) return invalid("Judul tugas wajib diisi.");
  return run(input.slug, (c) =>
    post<{ task: { id: string } }>(`${c}/items/${id(input.itemId)}/design-tasks`, { task: input.task }).then((d) => ({
      id: d.task.id,
    })),
  );
}

export async function updateDesignTask(input: {
  slug: string;
  taskId: string;
  task: DesignTaskInput;
  status?: DesignStatus;
}) {
  if (!input.task.title.trim()) return invalid("Judul tugas wajib diisi.");
  return run(input.slug, (c) =>
    patch(`${c}/design-tasks/${id(input.taskId)}`, { task: input.task, status: input.status }).then(() => null),
  );
}

export async function reviewDesign(input: {
  slug: string;
  taskId: string;
  decision: ReviewDecision;
  body: string;
  fileId?: string;
}) {
  if (input.decision !== "APPROVE" && !input.body.trim()) return invalid("Tulis masukan terlebih dahulu.");
  return run(input.slug, (c) =>
    post(`${c}/design-tasks/${id(input.taskId)}:review`, {
      decision: input.decision,
      body: input.body.trim(),
      fileId: input.fileId ?? "",
    }).then(() => null),
  );
}

// ── assets ───────────────────────────────────────────────────

export async function createFolder(input: { slug: string; name: string }) {
  if (!input.name.trim()) return invalid("Nama folder wajib diisi.");
  return run(input.slug, (c) => post(`${c}/asset-folders`, { name: input.name.trim() }).then(() => null));
}

export async function renameFolder(input: { slug: string; folderId: string; name: string }) {
  if (!input.name.trim()) return invalid("Nama folder wajib diisi.");
  return run(input.slug, (c) => patch(`${c}/asset-folders/${id(input.folderId)}`, { name: input.name.trim() }).then(() => null));
}

export async function deleteFolder(input: { slug: string; folderId: string }) {
  return run(input.slug, (c) => del(`${c}/asset-folders/${id(input.folderId)}`).then(() => null));
}

export async function updateAsset(input: {
  slug: string;
  assetId: string;
  name: string;
  kind: AssetKind;
  description: string;
  tags: string[];
  folderId: string;
}) {
  if (!input.name.trim()) return invalid("Nama aset wajib diisi.");
  const { slug, assetId, ...body } = input;
  return run(slug, (c) => patch(`${c}/assets/${id(assetId)}`, body).then(() => null));
}

export async function deleteAsset(input: { slug: string; assetId: string }) {
  return run(input.slug, (c) => del(`${c}/assets/${id(input.assetId)}`).then(() => null));
}

// ── reports ──────────────────────────────────────────────────

export async function generateReport(input: { slug: string; month: string }) {
  if (!isMonthKey(input.month)) return invalid("Pilih bulan yang valid.");
  return run(input.slug, (c) => post<ContentReport>(`${c}/reports`, { month: input.month }).then((r) => ({ id: r.id })));
}
