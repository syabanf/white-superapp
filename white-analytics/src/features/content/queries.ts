import "server-only";
import { cache } from "react";
import { notFound } from "next/navigation";
import { db } from "@/lib/db";
import { gatewayFetch, GatewayError } from "@/lib/erp/gateway";
import { requireClientAccess, requireUser } from "@/lib/rbac";
import type {
  Asset,
  AssetDetail,
  AssetFolder,
  ContentReport,
  DesignTask,
  DesignTaskDetail,
  Item,
  ItemDetail,
  Person,
  Plan,
  PlanDetail,
  SizePreset,
} from "./types";
import type { ContentDashboard } from "./dashboard";

export type ContentRole = "ADMIN" | "MANAGER" | "VIEWER";

/** Client, caller and the people who can be assigned, resolved once per request. */
export const contentContext = cache(async (slug: string) => {
  const [{ client, role }, user] = await Promise.all([requireClientAccess(slug), requireUser()]);
  const people: Person[] = await db.user.findMany({
    where: { isActive: true, OR: [{ role: "ADMIN" }, { memberships: { some: { clientId: client.id } } }] },
    select: { id: true, name: true },
    orderBy: { name: "asc" },
  });
  const contentRole: ContentRole = role === "ADMIN" ? "ADMIN" : role === "MANAGER" ? "MANAGER" : "VIEWER";
  return {
    client,
    role: contentRole,
    canWrite: contentRole !== "VIEWER",
    userId: user.id,
    people,
    names: Object.fromEntries(people.map((p) => [p.id, p.name])) as Record<string, string>,
  };
});

export type GatewayResult<T> = { ok: true; data: T } | { ok: false; message: string };

/** Runs gateway reads for a page: 404 becomes notFound(), other failures an offline notice. */
export async function loadContent<T>(fn: () => Promise<T>): Promise<GatewayResult<T>> {
  try {
    return { ok: true, data: await fn() };
  } catch (e) {
    if (e instanceof GatewayError) {
      if (e.status === 404) notFound();
      return { ok: false, message: e.message };
    }
    throw e;
  }
}

const base = (clientId: string) => `/v1/content/clients/${encodeURIComponent(clientId)}`;

function qs(params: Record<string, string | undefined>): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) if (v) q.set(k, v);
  const s = q.toString();
  return s ? `?${s}` : "";
}

export async function listPlans(clientId: string): Promise<Plan[]> {
  const res = await gatewayFetch<{ plans?: Plan[] }>(`${base(clientId)}/plans`);
  return res.plans ?? [];
}

export function getPlan(clientId: string, planId: string): Promise<PlanDetail> {
  return gatewayFetch<PlanDetail>(`${base(clientId)}/plans/${encodeURIComponent(planId)}`);
}

export async function listItems(
  clientId: string,
  filter: { planId?: string; status?: string; assigneeId?: string; month?: string } = {},
): Promise<Item[]> {
  const res = await gatewayFetch<{ items?: Item[] }>(`${base(clientId)}/items${qs(filter)}`);
  return res.items ?? [];
}

export function getItem(clientId: string, itemId: string): Promise<ItemDetail> {
  return gatewayFetch<ItemDetail>(`${base(clientId)}/items/${encodeURIComponent(itemId)}`);
}

export async function listDesignTasks(
  clientId: string,
  filter: { itemId?: string; designerId?: string; status?: string } = {},
): Promise<DesignTask[]> {
  const res = await gatewayFetch<{ tasks?: DesignTask[] }>(`${base(clientId)}/design-tasks${qs(filter)}`);
  return res.tasks ?? [];
}

export function getDesignTask(clientId: string, taskId: string): Promise<DesignTaskDetail> {
  return gatewayFetch<DesignTaskDetail>(`${base(clientId)}/design-tasks/${encodeURIComponent(taskId)}`);
}

export async function listSizePresets(): Promise<SizePreset[]> {
  const res = await gatewayFetch<{ presets?: SizePreset[] }>(`/v1/content/design-size-presets`);
  return res.presets ?? [];
}

export async function listFolders(clientId: string): Promise<AssetFolder[]> {
  const res = await gatewayFetch<{ folders?: AssetFolder[] }>(`${base(clientId)}/asset-folders`);
  return res.folders ?? [];
}

export async function searchAssets(
  clientId: string,
  filter: { query?: string; kind?: string; folderId?: string; tag?: string } = {},
): Promise<Asset[]> {
  const res = await gatewayFetch<{ assets?: Asset[] }>(`${base(clientId)}/assets${qs(filter)}`);
  return res.assets ?? [];
}

export function getAsset(clientId: string, assetId: string): Promise<AssetDetail> {
  return gatewayFetch<AssetDetail>(`${base(clientId)}/assets/${encodeURIComponent(assetId)}`);
}

export async function listReports(clientId: string): Promise<ContentReport[]> {
  const res = await gatewayFetch<{ reports?: ContentReport[] }>(`${base(clientId)}/reports`);
  return res.reports ?? [];
}

export function getReport(clientId: string, reportId: string): Promise<ContentReport> {
  return gatewayFetch<ContentReport>(`${base(clientId)}/reports/${encodeURIComponent(reportId)}`);
}

export function getDashboard(clientId: string, query: string): Promise<ContentDashboard> {
  return gatewayFetch<ContentDashboard>(`/v1/content/dashboard/clients/${encodeURIComponent(clientId)}?${query}`);
}
