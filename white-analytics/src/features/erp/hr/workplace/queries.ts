import "server-only";
import { gatewayFetch } from "@/lib/erp/gateway";
import type { Asset, AssetAssignment, ProjectOption, TimesheetEntry, Training, TrainingParticipant } from "./types";

const H = "/v1/hr";

function qs(params: Record<string, string | number | undefined>): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) if (v !== undefined && v !== "") q.set(k, String(v));
  const s = q.toString();
  return s ? `?${s}` : "";
}

export async function listAssets(f: { status?: string; search?: string; employeeId?: string } = {}) {
  return (await gatewayFetch<{ assets?: Asset[] }>(`${H}/assets${qs(f)}`)).assets ?? [];
}

export async function getAsset(id: string) {
  const res = await gatewayFetch<{ asset: Asset; history?: AssetAssignment[] }>(`${H}/assets/${encodeURIComponent(id)}`);
  return { asset: res.asset, history: res.history ?? [] };
}

export async function listTrainings(f: { status?: string; year?: number } = {}) {
  return (await gatewayFetch<{ trainings?: Training[] }>(`${H}/trainings${qs(f)}`)).trainings ?? [];
}

export async function getTraining(id: string) {
  const res = await gatewayFetch<{ training: Training; participants?: TrainingParticipant[] }>(`${H}/trainings/${encodeURIComponent(id)}`);
  return { training: res.training, participants: res.participants ?? [] };
}

export async function listTimesheet(f: { from: string; to: string; status?: string; employeeId?: string }) {
  return (await gatewayFetch<{ entries?: TimesheetEntry[] }>(`${H}/timesheet${qs(f)}`)).entries ?? [];
}

/** Active Finance projects; employees with self.service see names and clients only. */
export async function listProjectOptions(): Promise<ProjectOption[]> {
  const res = await gatewayFetch<{ tags?: ProjectOption[] }>(`/v1/finance/project-tags?activeOnly=true`);
  return (res.tags ?? []).map((t) => ({ id: t.id, name: t.name, clientName: t.clientName ?? "" }));
}
